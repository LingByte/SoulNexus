package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net/http"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/i18n"
	"github.com/LingByte/SoulNexus/pkg/middleware"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils/ginutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handlers) listMyDevices(c *gin.Context) {
	principalType, principalID, ok := authenticatedDevicePrincipal(c)
	if !ok {
		return
	}
	page, size := ginutil.QueryPage(c, 50)
	rows, total, err := models.ListUserDevicesPage(h.db, principalType, principalID, page, size)
	if err != nil {
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	currentKey := strings.TrimSpace(c.GetHeader("X-Device-Id"))
	if currentKey == "" {
		currentKey = strings.TrimSpace(c.Query("deviceId"))
	}
	out := make([]models.UserDevicePublic, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.UserDevicePublicRow(row, currentKey))
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{
		"list":  out,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

func (h *Handlers) revokeMyDevice(c *gin.Context) {
	principalType, principalID, ok := authenticatedDevicePrincipal(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if err := models.RevokeUserDevice(h.db, principalType, principalID, id, middleware.AuditOperator(c)); err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"revoked": true})
}

func (h *Handlers) deleteMyDevice(c *gin.Context) {
	principalType, principalID, ok := authenticatedDevicePrincipal(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	currentKey := strings.TrimSpace(c.GetHeader("X-Device-Id"))
	if currentKey == "" {
		currentKey = strings.TrimSpace(c.Query("deviceId"))
	}
	if err := models.DeleteUserDevice(h.db, principalType, principalID, id, currentKey, middleware.AuditOperator(c)); err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		if strings.Contains(err.Error(), "cannot delete current device") {
			response.Render(c, response.Wrap(response.CodeBadRequest, "cannot delete current device", err))
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"deleted": true})
}

func (h *Handlers) trustMyDevice(c *gin.Context) {
	principalType, principalID, ok := authenticatedDevicePrincipal(c)
	if !ok {
		return
	}
	id, ok := ginutil.ParamID(c, "id")
	if !ok {
		return
	}
	if err := models.TrustUserDevice(h.db, principalType, principalID, id, middleware.AuditOperator(c)); err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Render(c, response.Err(response.CodeNotFound))
			return
		}
		response.AbortWithStatusJSON(c, http.StatusInternalServerError, err)
		return
	}
	response.SuccessI18n(c, i18n.KeySuccess, gin.H{"trusted": true})
}
