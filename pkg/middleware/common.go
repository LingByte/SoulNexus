package middleware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
)

const (
	// EnvCORSAllowedOrigins is a comma-separated allowlist of Origin values
	// that may receive Access-Control-Allow-Credentials: true. When unset,
	// the middleware falls back to legacy permissive behaviour (reflect any
	// Origin) and logs a one-shot startup warning — production deployments
	// MUST set this.
	EnvCORSAllowedOrigins = "CORS_ALLOWED_ORIGINS"
)

var (
	corsAllowedOriginsOnce sync.Once
	corsAllowedOrigins     map[string]struct{}
	corsAllowAny           bool
)

func loadCORSAllowedOrigins() {
	corsAllowedOriginsOnce.Do(func() {
		raw := strings.TrimSpace(utils.GetEnv(EnvCORSAllowedOrigins))
		if raw == "" {
			corsAllowAny = true
			if logger.Lg != nil {
				logger.Lg.Warn("cors: CORS_ALLOWED_ORIGINS not set → reflecting any Origin with credentials (DEV ONLY; do not use in production)")
			}
			return
		}
		corsAllowedOrigins = make(map[string]struct{})
		for _, o := range strings.Split(raw, ",") {
			o = strings.TrimSpace(o)
			if o == "*" {
				// Wildcard with credentials is rejected by browsers anyway,
				// but mirror legacy behaviour for explicit dev opt-in.
				corsAllowAny = true
				continue
			}
			if o != "" {
				corsAllowedOrigins[o] = struct{}{}
			}
		}
	})
}

func corsOriginAllowed(origin string) bool {
	loadCORSAllowedOrigins()
	if corsAllowAny {
		return true
	}
	_, ok := corsAllowedOrigins[origin]
	return ok
}

// CorsMiddleware handles cross-origin resource sharing.
//
// Security note: when CORS_ALLOWED_ORIGINS is set we only reflect Origin
// values that appear in the allowlist; other cross-origin requests get
// no Access-Control-Allow-Origin header and the browser blocks the
// response. This protects authenticated endpoints (cookie session +
// JWT) from cross-site reads.
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowed := origin != "" && corsOriginAllowed(origin)

		if origin != "" {
			c.Writer.Header().Set("Vary", "Origin") // Avoid cache pollution
		}
		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if origin == "" {
			// Same-origin / non-browser caller: keep wildcard so health
			// probes etc. continue to work without credentials.
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Origin, Accept-Language, X-API-Key, X-API-KEY, X-Requested-With, X-Reqid, X-Req-ID, X-Request-ID, X-Device-Id")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, X-Reqid, X-Req-ID")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func WithMemSession(secret string) gin.HandlerFunc {
	store := memstore.NewStore([]byte(secret))
	store.Options(sessions.Options{Path: "/", MaxAge: 0})
	return sessions.Sessions(GetCarrotSessionField(), store)
}

func WithCookieSession(secret string, maxAge int) gin.HandlerFunc {
	store := cookie.NewStore([]byte(secret))
	store.Options(sessions.Options{Path: "/", MaxAge: maxAge})
	return sessions.Sessions(GetCarrotSessionField(), store)
}

func GetCarrotSessionField() string {
	v := utils.GetEnv(constants.ENV_SESSION_FIELD)
	if v == "" {
		return "lingecho"
	}
	return v
}

// SecurityMiddlewareChain returns security middleware chain
func SecurityMiddlewareChain() []gin.HandlerFunc {
	config := DefaultSecurityConfig()
	// CSRF is omitted: REST API uses Bearer JWT, not cookie form posts.
	// Cookie-session browser flows should mount CSRFMiddleware on a dedicated group.
	return []gin.HandlerFunc{
		SecurityMiddleware(config),
		XSSProtectionMiddleware(),
		InputValidationMiddleware(),
	}
}
