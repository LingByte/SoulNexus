// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// LLMToken admin CRUD：管理员可代任意用户创建 / 启停 / 改额度 / 重置 Key / 删除。
// 用户自助接口可在后续单独开放（在 /api/me/llm-tokens 下）。
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

type adminLLMTokenWriteReq struct {
	UserID         uint    `json:"user_id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"` // llm/asr/tts；空表示保持现值（创建时默认 llm）
	Status         string  `json:"status"`
	Group          string  `json:"group"`
	ModelWhitelist string  `json:"model_whitelist"`
	UnlimitedQuota *bool   `json:"unlimited_quota"`
	TokenQuota     *int64  `json:"token_quota"`
	RequestQuota   *int64  `json:"request_quota"`
	ExpiresAt      *string `json:"expires_at"` // RFC3339 / 空字符串清除
}

func (req *adminLLMTokenWriteReq) applyTo(row *models.LLMToken) error {
	row.Name = strings.TrimSpace(req.Name)
	if req.UserID > 0 {
		row.UserID = req.UserID
	}
	if t := strings.TrimSpace(req.Type); t != "" {
		if !models.IsValidLLMTokenType(t) {
			return errors.New("invalid type; expected llm/asr/tts")
		}
		row.Type = models.LLMTokenType(t)
	} else if row.Type == "" {
		row.Type = models.LLMTokenTypeLLM
	}
	row.Group = strings.TrimSpace(req.Group)
	if row.Group == "" {
		row.Group = "default"
	}
	row.ModelWhitelist = strings.TrimSpace(req.ModelWhitelist)
	if req.UnlimitedQuota != nil {
		row.UnlimitedQuota = *req.UnlimitedQuota
	}
	if req.TokenQuota != nil {
		row.TokenQuota = *req.TokenQuota
	}
	if req.RequestQuota != nil {
		row.RequestQuota = *req.RequestQuota
	}
	if req.Status != "" {
		switch models.LLMTokenStatus(req.Status) {
		case models.LLMTokenStatusActive, models.LLMTokenStatusDisabled, models.LLMTokenStatusExpired:
			row.Status = models.LLMTokenStatus(req.Status)
		default:
			return errors.New("invalid status")
		}
	}
	if req.ExpiresAt != nil {
		s := strings.TrimSpace(*req.ExpiresAt)
		if s == "" {
			row.ExpiresAt = nil
		} else {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return errors.New("expires_at must be RFC3339")
			}
			row.ExpiresAt = &t
		}
	}
	return nil
}

func maskLLMTokenAPIKey(k string) string {
	if len(k) <= 10 {
		return strings.Repeat("*", len(k))
	}
	return k[:6] + "…" + k[len(k)-4:]
}

func (h *Handlers) handleAdminListLLMTokens(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	q := h.db.Model(&models.LLMToken{})
	if uid := strings.TrimSpace(c.Query("user_id")); uid != "" {
		if v, err := strconv.Atoi(uid); err == nil {
			q = q.Where("user_id = ?", v)
		}
	}
	if g := strings.TrimSpace(c.Query("group")); g != "" {
		q = q.Where("`group` = ?", g)
	}
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		q = q.Where("status = ?", s)
	}
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		like := "%" + s + "%"
		q = q.Where("name LIKE ? OR api_key LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.Fail(c, "list tokens failed", err)
		return
	}
	var rows []models.LLMToken
	if err := q.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		response.Fail(c, "list tokens failed", err)
		return
	}
	masked := !strings.EqualFold(c.Query("show_key"), "true")
	if masked {
		for i := range rows {
			rows[i].APIKey = maskLLMTokenAPIKey(rows[i].APIKey)
		}
	}
	response.Success(c, "tokens fetched", gin.H{
		"tokens":   rows,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetLLMToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.LLMToken
	if err := h.db.First(&row, id).Error; err != nil {
		response.Fail(c, "token not found", err)
		return
	}
	response.Success(c, "token fetched", gin.H{"token": row})
}

func (h *Handlers) handleAdminCreateLLMToken(c *gin.Context) {
	var req adminLLMTokenWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	if req.UserID == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("user_id is required"))
		return
	}
	row := &models.LLMToken{Status: models.LLMTokenStatusActive}
	if err := req.applyTo(row); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	apiKey, err := models.GenerateLLMTokenAPIKey()
	if err != nil {
		response.Fail(c, "generate api key failed", err)
		return
	}
	row.APIKey = apiKey
	if err := h.db.Create(row).Error; err != nil {
		response.Fail(c, "create token failed", err)
		return
	}
	// 返回完整 Key（仅这一次明文展示，前端需提示用户立刻复制）。
	response.Success(c, "token created", gin.H{"token": row, "raw_api_key": apiKey})
}

func (h *Handlers) handleAdminUpdateLLMToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.LLMToken
	if err := h.db.First(&row, id).Error; err != nil {
		response.Fail(c, "token not found", err)
		return
	}
	var req adminLLMTokenWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	if err := req.applyTo(&row); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.db.Save(&row).Error; err != nil {
		response.Fail(c, "update token failed", err)
		return
	}
	response.Success(c, "token updated", gin.H{"token": row})
}

func (h *Handlers) handleAdminRegenerateLLMToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.LLMToken
	if err := h.db.First(&row, id).Error; err != nil {
		response.Fail(c, "token not found", err)
		return
	}
	apiKey, err := models.GenerateLLMTokenAPIKey()
	if err != nil {
		response.Fail(c, "generate api key failed", err)
		return
	}
	row.APIKey = apiKey
	if err := h.db.Model(&row).Updates(map[string]any{"api_key": apiKey}).Error; err != nil {
		response.Fail(c, "regenerate failed", err)
		return
	}
	response.Success(c, "api key regenerated", gin.H{"token": row, "raw_api_key": apiKey})
}

func (h *Handlers) handleAdminResetLLMTokenUsage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err := h.db.Model(&models.LLMToken{}).Where("id = ?", id).Updates(map[string]any{
		"token_used":   0,
		"quota_used":   0,
		"request_used": 0,
	}).Error; err != nil {
		response.Fail(c, "reset usage failed", err)
		return
	}
	response.Success(c, "usage reset", nil)
}

func (h *Handlers) handleAdminDeleteLLMToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err := h.db.Delete(&models.LLMToken{}, id).Error; err != nil {
		response.Fail(c, "delete token failed", err)
		return
	}
	response.Success(c, "token deleted", nil)
}
