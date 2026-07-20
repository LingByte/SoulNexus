package middleware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/access"
	"github.com/gin-gonic/gin"
)

const (
	CtxAuthPlatformAdminID = "auth.platformAdminId"
	AuthRolePlatformAdmin  = "platform_admin"
)

// TryAttachPlatformJWT parses Authorization Bearer for a platform-admin token (distinct issuer).
func TryAttachPlatformJWT(c *gin.Context) bool {
	token := bearerTokenFromRequest(c)
	if token == "" || models.IsAPIKeyToken(token) {
		return false
	}
	km := access.JWTKeyManager()
	if km == nil {
		return false
	}
	p, err := access.ParsePlatformAccessTokenWithKey(token, km)
	if err != nil || p == nil || p.AdminID == 0 {
		return false
	}
	c.Set(CtxAuthPlatformAdminID, p.AdminID)
	c.Set(CtxAuthEmail, p.Email)
	c.Set(CtxAuthRole, AuthRolePlatformAdmin)
	c.Set(CtxAuthSessionID, p.SessionID)
	c.Set(CtxAuthDeviceRecordID, p.DeviceRecordID)
	c.Set(CtxAuthUserID, uint(0))
	c.Set(CtxAuthTenantID, uint(0))
	c.Set(CtxAuthTenantSlug, "")
	return true
}

func AuthPlatformAdminID(c *gin.Context) uint {
	if v, ok := c.Get(CtxAuthPlatformAdminID); ok {
		if n, ok := v.(uint); ok {
			return n
		}
	}
	return 0
}

// RequirePlatformAdmin rejects requests that are not authenticated as a platform admin.
func RequirePlatformAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthPlatformAdminID(c) == 0 {
			response.Render(c, utils.ErrOnlySuperUser)
			return
		}
		c.Next()
	}
}
