package middleware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

// SecureResponseHeaders sets baseline browser-facing security headers on every response.
// CSP: set HTTP_CSP for a site-wide policy, or a restrictive default is applied only under /api/ (JSON APIs).
func SecureResponseHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		if csp := strings.TrimSpace(os.Getenv("HTTP_CSP")); csp != "" {
			c.Header("Content-Security-Policy", csp)
		} else {
			p := c.Request.URL.Path
			if strings.HasPrefix(p, "/api/") && !strings.Contains(p, "/docs") {
				c.Header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			}
		}
		if isHTTPSRequest(c) && !strings.EqualFold(strings.TrimSpace(os.Getenv("HTTP_HSTS_DISABLED")), "true") {
			maxAge := utils.GetIntEnvWithDefault("HTTP_HSTS_MAX_AGE", 31536000)
			c.Header("Strict-Transport-Security", "max-age="+strconv.Itoa(maxAge)+"; includeSubDomains")
		}
		c.Next()
	}
}

func isHTTPSRequest(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

// MutatingRequestTrustedOrigin blocks cross-site browser POSTs that rely on ambient credentials
// when TRUSTED_BROWSER_ORIGINS is configured. Third-party webhooks and similar paths are skipped.
//
// Env:
//   - TRUSTED_BROWSER_ORIGINS: comma-separated full origins, e.g. "https://app.example.com,https://admin.example.com"
//   - TRUSTED_ORIGIN_CHECK_SKIP_PREFIXES: extra path prefixes (comma-separated) to skip after API_PREFIX is stripped
func MutatingRequestTrustedOrigin() gin.HandlerFunc {
	trusted := parseTrustedOriginsList(strings.TrimSpace(os.Getenv("TRUSTED_BROWSER_ORIGINS")))
	extraSkips := splitCSV(os.Getenv("TRUSTED_ORIGIN_CHECK_SKIP_PREFIXES"))
	return func(c *gin.Context) {
		if len(trusted) == 0 {
			c.Next()
			return
		}
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			c.Next()
			return
		}
		path := c.Request.URL.Path
		if shouldSkipTrustedOriginCheck(path, extraSkips) {
			c.Next()
			return
		}
		authz := strings.TrimSpace(c.GetHeader("Authorization"))
		if strings.HasPrefix(strings.ToLower(authz), "bearer ") && authz != "" {
			c.Next()
			return
		}
		site := strings.ToLower(strings.TrimSpace(c.GetHeader("Sec-Fetch-Site")))
		switch site {
		case "cross-site":
			if originAllows(c, trusted) || refererAllows(c, trusted) {
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "origin_not_allowed",
				"message": "Cross-site request origin is not trusted",
			})
			return
		}
		// same-origin, same-site, none, or empty (older clients)
		if strings.TrimSpace(c.GetHeader("Origin")) == "" && strings.TrimSpace(c.GetHeader("Referer")) == "" {
			c.Next()
			return
		}
		if originAllows(c, trusted) || refererAllows(c, trusted) || sameHostReferer(c) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":   "origin_not_allowed",
			"message": "Request origin is not trusted for this operation",
		})
	}
}

func defaultSkipOriginPrefixes() []string {
	return []string{
		"/api/webhooks/",
		"/api/auth/wechat/",
		"/api/auth/github/",
		"/api/auth/oidc/",
		"/api/js-templates/webhook/",
		"/api/device/status",
		"/api/device/error",
		"/api/ota/",
	}
}

func shouldSkipTrustedOriginCheck(path string, extra []string) bool {
	for _, p := range defaultSkipOriginPrefixes() {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	for _, p := range extra {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func parseTrustedOriginsList(csv string) []string {
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, strings.TrimRight(p, "/"))
	}
	return out
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	ch := strings.Split(s, ",")
	out := make([]string, 0, len(ch))
	for _, v := range ch {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func originAllows(c *gin.Context, trusted []string) bool {
	o := strings.TrimSpace(c.GetHeader("Origin"))
	if o == "" {
		return false
	}
	u, err := url.Parse(o)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	candidate := strings.TrimRight(u.Scheme+"://"+u.Host, "/")
	for _, t := range trusted {
		if strings.EqualFold(candidate, t) {
			return true
		}
	}
	return false
}

func refererAllows(c *gin.Context, trusted []string) bool {
	ref := strings.TrimSpace(c.GetHeader("Referer"))
	if ref == "" {
		return false
	}
	u, err := url.Parse(ref)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	candidate := strings.TrimRight(u.Scheme+"://"+u.Host, "/")
	for _, t := range trusted {
		if strings.EqualFold(candidate, t) {
			return true
		}
	}
	return false
}

func sameHostReferer(c *gin.Context) bool {
	ref := strings.TrimSpace(c.GetHeader("Referer"))
	if ref == "" {
		return false
	}
	u, err := url.Parse(ref)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, c.Request.Host)
}
