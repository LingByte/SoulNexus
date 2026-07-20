package middleware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// MetricsACL is the gate for /metrics. Prometheus metrics expose a lot
// of internal signal (active calls, error counters, queue depths,
// goroutine counts, GC stats from the Go runtime). On a public-internet
// listener that's a free reconnaissance feed: an attacker can probe
// load patterns, derive deployment topology, watch error spikes after
// they fire payloads, etc. We don't want to fail-open just because the
// operator forgot to set up Prometheus auth.
//
// Behaviour:
//   - default-deny when METRICS_ALLOWED_IPS is unset.
//   - METRICS_ALLOWED_IPS = "127.0.0.1,::1,10.0.0.0/8,172.16.0.0/12"
//     (comma list of IPs and/or CIDRs). Loopback and RFC1918 are common.
//   - "*" allows any caller (only meaningful when /metrics is fronted
//     by mTLS / nginx allow / k8s NetworkPolicy already).
//
// Implementation note: c.ClientIP() honours TrustedProxies +
// X-Forwarded-For. If the deployment puts /metrics behind a reverse
// proxy that strips X-Forwarded-For, the source IP will be the proxy
// itself — add the proxy IP to the allowlist explicitly.

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

const envMetricsAllowedIPs = "METRICS_ALLOWED_IPS"

var (
	metricsACLOnce sync.Once
	metricsACLAny  bool
	metricsACLNets []*net.IPNet
	metricsACLIPs  []net.IP
)

func loadMetricsACL() {
	raw := strings.TrimSpace(utils.GetEnv(envMetricsAllowedIPs))
	if raw == "" {
		return
	}
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		if p == "*" {
			metricsACLAny = true
			continue
		}
		if strings.Contains(p, "/") {
			if _, n, err := net.ParseCIDR(p); err == nil {
				metricsACLNets = append(metricsACLNets, n)
			}
			continue
		}
		if ip := net.ParseIP(p); ip != nil {
			metricsACLIPs = append(metricsACLIPs, ip)
		}
	}
}

func metricsACLAllows(ipStr string) bool {
	metricsACLOnce.Do(loadMetricsACL)
	if metricsACLAny {
		return true
	}
	if ipStr == "" {
		return false
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, allow := range metricsACLIPs {
		if allow.Equal(ip) {
			return true
		}
	}
	for _, n := range metricsACLNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// MetricsACL gates a Gin handler with the METRICS_ALLOWED_IPS
// allowlist. Returns 403 + a single-line message when the source IP
// isn't allowed. Use as a wrapping middleware on the /metrics route.
func MetricsACL() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First-hit warning when env is empty so default-deny is
		// loud at startup (logger import would create a cycle here;
		// surface via response header instead — operators see it the
		// first time they curl /metrics).
		metricsACLOnce.Do(loadMetricsACL)
		if !metricsACLAny && len(metricsACLIPs) == 0 && len(metricsACLNets) == 0 {
			c.Header("X-Metrics-ACL", "default-deny: set METRICS_ALLOWED_IPS")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "metrics access denied: METRICS_ALLOWED_IPS unset"})
			return
		}
		if !metricsACLAllows(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 403, "msg": "metrics access denied"})
			return
		}
		c.Next()
	}
}
