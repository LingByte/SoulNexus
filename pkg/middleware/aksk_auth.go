package middleware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils/system"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	HeaderXAPIKey = "X-API-Key"

	// CtxAuthCredentialID is set when the request is authenticated via API key.
	CtxAuthCredentialID       = "auth.credentialId"
	CtxAuthCredentialRouteIDs = "auth.credentialRouteIds"
	AuthRoleCredential        = "credential_apikey"
)

// RequireTenantJWTOrAPIKey accepts (in order): API key, tenant JWT, or platform-admin JWT.
// API key requests are checked against the platform route allowlist and credential scope.
func RequireTenantJWTOrAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		dbIface, exists := c.Get(constants.DbField)
		db, ok := dbIface.(*gorm.DB)
		if !exists || !ok || db == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "database unavailable", "data": nil})
			return
		}

		if attempted, authed := tryAttachAPIKey(c, db); attempted {
			if !authed {
				return
			}
			if !enforceAPIKeyRoutePolicy(c) {
				return
			}
			c.Next()
			return
		}

		if TryAttachTenantJWT(c) {
			if !validateBearerDeviceSession(c) {
				abortSessionRevoked(c)
				return
			}
			c.Next()
			return
		}
		if TryAttachPlatformJWT(c) {
			if !validateBearerDeviceSession(c) {
				abortSessionRevoked(c)
				return
			}
			c.Next()
			return
		}

		if raw := bearerTokenFromRequest(c); raw != "" && !models.IsAPIKeyToken(raw) {
			system.IncJWTVerifyFail()
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "missing or invalid authorization", "data": nil})
	}
}

// RequireTenantJWTOrAKSK is a compatibility alias of RequireTenantJWTOrAPIKey.
func RequireTenantJWTOrAKSK() gin.HandlerFunc {
	return RequireTenantJWTOrAPIKey()
}

func apiKeyFromRequest(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if k := strings.TrimSpace(c.GetHeader(HeaderXAPIKey)); k != "" {
		return k
	}
	if k := strings.TrimSpace(c.GetHeader("X-API-KEY")); k != "" {
		return k
	}
	if auth := bearerTokenFromRequest(c); models.IsAPIKeyToken(auth) {
		return auth
	}
	if k := strings.TrimSpace(c.Query("api_key")); k != "" {
		return k
	}
	if k := strings.TrimSpace(c.Query("apiKey")); k != "" {
		return k
	}
	return ""
}

func tryAttachAPIKey(c *gin.Context, db *gorm.DB) (attempted, ok bool) {
	raw := apiKeyFromRequest(c)
	if raw == "" {
		return false, false
	}
	attempted = true

	if !models.IsAPIKeyToken(raw) {
		abortAuthJSON(c, http.StatusUnauthorized, 401, "invalid api key")
		return true, false
	}

	cred, err := models.GetActiveCredentialByAPIKey(db, raw)
	if err != nil {
		logger.ErrorCtx(c.Request.Context(), "get active api key error", zap.Error(err))
		abortAuthJSON(c, http.StatusUnauthorized, 401, "unknown or revoked api key")
		return true, false
	}
	if models.CredentialIsExpired(cred, time.Now()) {
		abortAuthJSON(c, http.StatusUnauthorized, 401, "api key expired")
		return true, false
	}

	clientIP := strings.TrimSpace(c.ClientIP())
	if !models.CredentialClientIPAllowed(cred.AllowIP, clientIP) {
		abortAuthJSON(c, http.StatusForbidden, 403, "ip not allowed")
		return true, false
	}

	c.Set(CtxAuthUserID, uint(0))
	c.Set(CtxAuthTenantID, cred.TenantID)
	c.Set(CtxAuthTenantSlug, "")
	c.Set(CtxAuthEmail, "apikey:"+cred.AccessKey)
	c.Set(CtxAuthRole, AuthRoleCredential)
	c.Set(CtxAuthCredentialID, cred.ID)
	c.Set(CtxAuthCredentialRouteIDs, models.ParseCredentialAllowedRouteIDs(cred.AllowedRouteIDs))
	go models.RecordCredentialUse(db, cred.ID, time.Now())
	return true, true
}

func abortAuthJSON(c *gin.Context, httpStatus, code int, msg string) {
	c.AbortWithStatusJSON(httpStatus, gin.H{"code": code, "msg": msg, "data": nil})
}

// AuthCredentialRouteIDs returns route catalog ids scoped to the current API key.
func AuthCredentialRouteIDs(c *gin.Context) []string {
	if v, ok := c.Get(CtxAuthCredentialRouteIDs); ok {
		if ids, ok := v.([]string); ok {
			return ids
		}
	}
	return nil
}

func enforceAPIKeyRoutePolicy(c *gin.Context) bool {
	if AuthCredentialID(c) == 0 {
		return true
	}
	method := c.Request.Method
	path := c.Request.URL.Path
	if !models.AKSKSystemRouteAllowed(method, path) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code": 403,
			"msg":  i18n.TGin(c, i18n.KeyAKSKRouteNotAllowed),
			"data": nil,
		})
		return false
	}
	if !models.CredentialRoutesAllowed(AuthCredentialRouteIDs(c), method, path) {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"code": 403,
			"msg":  i18n.TGin(c, i18n.KeyAKSKCredentialRouteNotAllowed),
			"data": nil,
		})
		return false
	}
	return true
}

// enforceAKSKRoutePolicy is kept as an alias for older call sites/tests.
func enforceAKSKRoutePolicy(c *gin.Context) bool {
	return enforceAPIKeyRoutePolicy(c)
}
