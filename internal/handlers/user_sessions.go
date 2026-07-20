package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/LingByte/SoulNexus/pkg/i18n"
)

func (h *Handlers) revokeAllMySessions(c *gin.Context) {
	principalType, principalID, ok := authenticatedDevicePrincipal(c)
	if !ok {
		return
	}
	if err := models.RevokeAllUserDevices(h.db, principalType, principalID); err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"revokedAll": true})
}

func authenticatedDevicePrincipal(c *gin.Context) (principalType string, principalID uint, ok bool) {
	if aid := middleware.AuthPlatformAdminID(c); aid > 0 {
		return models.UserDevicePrincipalPlatformAdmin, aid, true
	}
	if uid := middleware.AuthUserID(c); uid > 0 {
		return models.UserDevicePrincipalTenantUser, uid, true
	}
	response.Render(c, response.Err(response.CodeUnauthorized))
	return "", 0, false
}
