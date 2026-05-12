// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// SpeechUsage admin/me 列表接口：可按 kind / user_id / group / channel / 时间范围筛选。

package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// handleAdminListSpeechUsage GET /api/admin/speech-usage
func (h *Handlers) handleAdminListSpeechUsage(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	q := h.db.Model(&models.SpeechUsage{})
	if k := strings.TrimSpace(c.Query("kind")); k != "" {
		q = q.Where("kind = ?", k)
	}
	if uid := strings.TrimSpace(c.Query("user_id")); uid != "" {
		if v, err := strconv.Atoi(uid); err == nil {
			q = q.Where("user_id = ?", v)
		}
	}
	if tid := strings.TrimSpace(c.Query("token_id")); tid != "" {
		if v, err := strconv.Atoi(tid); err == nil {
			q = q.Where("token_id = ?", v)
		}
	}
	if g := strings.TrimSpace(c.Query("group")); g != "" {
		q = q.Where("group_key = ?", g)
	}
	if cid := strings.TrimSpace(c.Query("channel_id")); cid != "" {
		if v, err := strconv.Atoi(cid); err == nil {
			q = q.Where("channel_id = ?", v)
		}
	}
	if p := strings.TrimSpace(c.Query("provider")); p != "" {
		q = q.Where("provider = ?", p)
	}
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			q = q.Where("status = ?", v)
		}
	}
	if s := strings.TrimSpace(c.Query("success")); s != "" {
		switch strings.ToLower(s) {
		case "true", "1":
			q = q.Where("success = ?", true)
		case "false", "0":
			q = q.Where("success = ?", false)
		}
	}
	if from := strings.TrimSpace(c.Query("from")); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if to := strings.TrimSpace(c.Query("to")); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			q = q.Where("created_at <= ?", t)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.Fail(c, "count speech usage failed", err)
		return
	}
	var rows []models.SpeechUsage
	if err := q.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		response.Fail(c, "list speech usage failed", err)
		return
	}
	response.Success(c, "speech usage fetched", gin.H{
		"items":    rows,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// handleAdminGetSpeechUsage GET /api/admin/speech-usage/:id
func (h *Handlers) handleAdminGetSpeechUsage(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.SpeechUsage
	if err := h.db.Where("id = ?", id).First(&row).Error; err != nil {
		response.Fail(c, "speech usage not found", err)
		return
	}
	response.Success(c, "speech usage fetched", gin.H{"item": row})
}

// handleAdminSpeechUsageStats GET /api/admin/speech-usage/stats
// 简单聚合：按 kind / provider / 当日成功率。
func (h *Handlers) handleAdminSpeechUsageStats(c *gin.Context) {
	var rows []struct {
		Kind     string `json:"kind"`
		Provider string `json:"provider"`
		Total    int64  `json:"total"`
		Success  int64  `json:"success"`
		Failed   int64  `json:"failed"`
	}
	if err := h.db.Raw(`
		SELECT kind, provider,
		       COUNT(1) AS total,
		       SUM(CASE WHEN success THEN 1 ELSE 0 END) AS success,
		       SUM(CASE WHEN success THEN 0 ELSE 1 END) AS failed
		  FROM speech_usage
		 GROUP BY kind, provider
		 ORDER BY total DESC
	`).Scan(&rows).Error; err != nil {
		response.Fail(c, "speech usage stats failed", err)
		return
	}
	response.Success(c, "speech usage stats", gin.H{"items": rows})
}

// handleMeListSpeechUsage GET /api/me/speech-usage
// 用户视角：仅看自己的调用。
func (h *Handlers) handleMeListSpeechUsage(c *gin.Context) {
	u := models.CurrentUser(c)
	if u == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("not authenticated"))
		return
	}
	page, pageSize := h.parsePagination(c)
	q := h.db.Model(&models.SpeechUsage{}).Where("user_id = ?", u.ID)
	if k := strings.TrimSpace(c.Query("kind")); k != "" {
		q = q.Where("kind = ?", k)
	}
	var total int64
	_ = q.Count(&total).Error
	var rows []models.SpeechUsage
	if err := q.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		response.Fail(c, "list speech usage failed", err)
		return
	}
	response.Success(c, "speech usage fetched", gin.H{
		"items":    rows,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}
