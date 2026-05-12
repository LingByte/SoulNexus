// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 用户自助 LLM Token 接口（/api/me/llm-tokens）。
// 与 admin 接口共享 LLMToken 模型，但所有查询均强制限定 user_id = current_user.id。
package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// meLLMTokenWriteReq 用户自助接口创建/更新入参（受限字段）。
// user_id 强制取自 session；状态/额度只能由 admin 调；用户能改 name / type / group / model_whitelist。
type meLLMTokenWriteReq struct {
	Name           string `json:"name"`
	Type           string `json:"type"` // llm/asr/tts；空 = 创建时默认 llm，更新时保持原值
	Group          string `json:"group"`
	ModelWhitelist string `json:"model_whitelist"`
}

func (h *Handlers) handleMeListLLMTokens(c *gin.Context) {
	u := models.CurrentUser(c)
	if u == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("not authenticated"))
		return
	}
	page, pageSize := h.parsePagination(c)
	// 列表按当前激活组织过滤：org_id=0（中间件默认）只看个人 token；
	// org_id>0 时只看该组织下且 user_id 属于自己的 token。
	q := h.db.Model(&models.LLMToken{}).Where("user_id = ? AND org_id = ?", u.ID, OrgIDFromContext(c))
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		q = q.Where("status = ?", s)
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
	for i := range rows {
		rows[i].APIKey = maskLLMTokenAPIKey(rows[i].APIKey)
	}
	response.Success(c, "tokens fetched", gin.H{
		"tokens":   rows,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleMeCreateLLMToken(c *gin.Context) {
	u := models.CurrentUser(c)
	if u == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("not authenticated"))
		return
	}
	var req meLLMTokenWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	typ := models.LLMTokenType(strings.TrimSpace(req.Type))
	if typ == "" {
		typ = models.LLMTokenTypeLLM
	} else if !models.IsValidLLMTokenType(string(typ)) {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid type; expected llm/asr/tts"))
		return
	}
	// 多租户：若请求带 X-Org-ID（或 ?org_id=），中间件已校验成员身份，落到 token.OrgID；
	// 未指定时为 0，表示个人级 token。
	row := &models.LLMToken{
		UserID:         u.ID,
		OrgID:          OrgIDFromContext(c),
		Name:           strings.TrimSpace(req.Name),
		Type:           typ,
		Group:          firstNonEmpty(strings.TrimSpace(req.Group), "default"),
		ModelWhitelist: strings.TrimSpace(req.ModelWhitelist),
		Status:         models.LLMTokenStatusActive,
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
	response.Success(c, "token created", gin.H{"token": row, "raw_api_key": apiKey})
}

func (h *Handlers) handleMeUpdateLLMToken(c *gin.Context) {
	u := models.CurrentUser(c)
	if u == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("not authenticated"))
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.LLMToken
	if err := h.db.Where("id = ? AND user_id = ?", id, u.ID).First(&row).Error; err != nil {
		response.Fail(c, "token not found", err)
		return
	}
	var req meLLMTokenWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	row.Name = strings.TrimSpace(req.Name)
	if t := strings.TrimSpace(req.Type); t != "" {
		if !models.IsValidLLMTokenType(t) {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid type; expected llm/asr/tts"))
			return
		}
		row.Type = models.LLMTokenType(t)
	}
	if g := strings.TrimSpace(req.Group); g != "" {
		row.Group = g
	}
	row.ModelWhitelist = strings.TrimSpace(req.ModelWhitelist)
	if err := h.db.Save(&row).Error; err != nil {
		response.Fail(c, "update token failed", err)
		return
	}
	response.Success(c, "token updated", gin.H{"token": row})
}

func (h *Handlers) handleMeRegenerateLLMToken(c *gin.Context) {
	u := models.CurrentUser(c)
	if u == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("not authenticated"))
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.LLMToken
	if err := h.db.Where("id = ? AND user_id = ?", id, u.ID).First(&row).Error; err != nil {
		response.Fail(c, "token not found", err)
		return
	}
	apiKey, err := models.GenerateLLMTokenAPIKey()
	if err != nil {
		response.Fail(c, "generate api key failed", err)
		return
	}
	if err := h.db.Model(&row).Update("api_key", apiKey).Error; err != nil {
		response.Fail(c, "regenerate failed", err)
		return
	}
	row.APIKey = apiKey
	response.Success(c, "api key regenerated", gin.H{"token": row, "raw_api_key": apiKey})
}

func (h *Handlers) handleMeDeleteLLMToken(c *gin.Context) {
	u := models.CurrentUser(c)
	if u == nil {
		response.AbortWithJSONError(c, http.StatusUnauthorized, errors.New("not authenticated"))
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err := h.db.Where("id = ? AND user_id = ?", id, u.ID).Delete(&models.LLMToken{}).Error; err != nil {
		response.Fail(c, "delete token failed", err)
		return
	}
	response.Success(c, "token deleted", nil)
}

func firstNonEmpty(s ...string) string {
	for _, x := range s {
		if x != "" {
			return x
		}
	}
	return ""
}
