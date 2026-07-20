// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Per-IP rate limiter for authentication-style endpoints (login,
// register, password reset). Backed by a sync.Map of golang.org/x/time
// rate.Limiter values, lazily created and idle-evicted by a background
// sweeper to bound memory under a sustained scan.
//
// We deliberately do NOT use a shared bucket across IPs (would punish
// legitimate users on shared NATs). The defaults here (5 req / 5min,
// burst 5) are tuned so a single misconfigured client retrying or a
// human typing their password fast won't get blocked, but a credential
// stuffing botnet hitting one IP at >1 req/min gets throttled.

package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type authLimiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// AuthRateLimiter returns a gin middleware that throttles requests by
// client IP. perWindow specifies how many requests are allowed in the
// rolling window of length window; burst is the initial token bucket
// capacity (controls "thundering" behaviour at process start).
//
// Returns 429 with a Retry-After-style hint when the limit is exceeded.
// nil-safe: zero values fall back to safe defaults (5 req / 5 min).
func AuthRateLimiter(perWindow int, window time.Duration, burst int) gin.HandlerFunc {
	if perWindow <= 0 {
		perWindow = 5
	}
	if window <= 0 {
		window = 5 * time.Minute
	}
	if burst <= 0 {
		burst = perWindow
	}
	rps := rate.Limit(float64(perWindow) / window.Seconds())

	var (
		store sync.Map // map[string]*authLimiterEntry
	)

	// Background sweeper: evict entries idle for >2 windows. Keeps
	// memory bounded under a slow scan that touches many IPs once.
	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-2 * window)
			store.Range(func(k, v any) bool {
				if e, ok := v.(*authLimiterEntry); ok && e.lastSeen.Before(cutoff) {
					store.Delete(k)
				}
				return true
			})
		}
	}()

	getLimiter := func(key string) *rate.Limiter {
		now := time.Now()
		if v, ok := store.Load(key); ok {
			e := v.(*authLimiterEntry)
			e.lastSeen = now
			return e.limiter
		}
		newEntry := &authLimiterEntry{
			limiter:  rate.NewLimiter(rps, burst),
			lastSeen: now,
		}
		actual, _ := store.LoadOrStore(key, newEntry)
		e := actual.(*authLimiterEntry)
		e.lastSeen = now
		return e.limiter
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = c.Request.RemoteAddr
		}
		if !getLimiter(ip).Allow() {
			if logger.Lg != nil {
				logger.Lg.Warn("auth rate limit triggered",
					zap.String("ip", ip),
					zap.String("path", c.Request.URL.Path),
				)
			}
			retryAfter := int(window.Seconds())
			c.Header("Retry-After", itoa(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code": http.StatusTooManyRequests,
				"msg":  "请求过于频繁，请稍后再试",
			})
			return
		}
		c.Next()
	}
}

// itoa is a tiny strconv.Itoa replacement that avoids the strconv import
// just for one Header() call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
