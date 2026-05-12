// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

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

// llmChannelWriteReq 管理端创建/更新 LLM 渠道入参。
// 与 LingVoice 兼容；字段含义见 models.LLMChannel。
type llmChannelWriteReq struct {
	Protocol           string                 `json:"protocol"`
	Type               int                    `json:"type"`
	Key                string                 `json:"key"`
	OpenAIOrganization *string                `json:"openai_organization"`
	TestModel          *string                `json:"test_model"`
	Status             *int                   `json:"status"`
	Name               string                 `json:"name"`
	Weight             *uint                  `json:"weight"`
	BaseURL            *string                `json:"base_url"`
	Models             string                 `json:"models"`
	Group              string                 `json:"group"`
	ModelMapping       *string                `json:"model_mapping"`
	StatusCodeMapping  *string                `json:"status_code_mapping"`
	Priority           *int64                 `json:"priority"`
	AutoBan            *int                   `json:"auto_ban"`
	Tag                *string                `json:"tag"`
	ChannelInfo        *models.LLMChannelInfo `json:"channel_info"`
}

func (req *llmChannelWriteReq) applyTo(row *models.LLMChannel) error {
	p := strings.ToLower(strings.TrimSpace(req.Protocol))
	if p == "" {
		p = models.LLMChannelProtocolOpenAI
	}
	if !models.IsLLMChannelProtocolKnown(p) {
		return errors.New("unknown protocol")
	}
	row.Protocol = p
	row.Type = req.Type
	row.Key = strings.TrimSpace(req.Key)
	row.OpenAIOrganization = req.OpenAIOrganization
	row.TestModel = req.TestModel
	if req.Status != nil {
		row.Status = *req.Status
	} else if row.Status == 0 {
		row.Status = 1
	}
	row.Name = strings.TrimSpace(req.Name)
	row.Weight = req.Weight
	row.BaseURL = req.BaseURL
	row.Models = strings.TrimSpace(req.Models)
	row.Group = strings.TrimSpace(req.Group)
	if row.Group == "" {
		row.Group = "default"
	}
	row.ModelMapping = req.ModelMapping
	row.StatusCodeMapping = req.StatusCodeMapping
	row.Priority = req.Priority
	row.AutoBan = req.AutoBan
	row.Tag = req.Tag
	if req.ChannelInfo != nil {
		row.ChannelInfo = *req.ChannelInfo
	}
	return nil
}

// 隐藏 Key 内容，列表展示时只保留前后几位。
func maskLLMChannelKey(k string) string {
	k = strings.TrimSpace(k)
	if len(k) <= 8 {
		return strings.Repeat("*", len(k))
	}
	return k[:4] + "…" + k[len(k)-4:]
}

func (h *Handlers) handleAdminListLLMChannels(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	q := h.db.Model(&models.LLMChannel{})
	if g := strings.TrimSpace(c.Query("group")); g != "" {
		q = q.Where("`group` = ?", g)
	}
	if p := strings.TrimSpace(c.Query("protocol")); p != "" {
		q = q.Where("protocol = ?", p)
	}
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			q = q.Where("status = ?", v)
		}
	}
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		like := "%" + s + "%"
		q = q.Where("name LIKE ? OR `group` LIKE ? OR base_url LIKE ? OR models LIKE ?", like, like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.Fail(c, "list channels failed", err)
		return
	}
	var rows []models.LLMChannel
	if err := q.Order("priority DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		response.Fail(c, "list channels failed", err)
		return
	}
	masked := strings.EqualFold(c.Query("mask_key"), "true")
	if masked {
		for i := range rows {
			rows[i].Key = maskLLMChannelKey(rows[i].Key)
		}
	}
	response.Success(c, "channels fetched", gin.H{
		"channels": rows,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetLLMChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.LLMChannel
	if err := h.db.First(&row, id).Error; err != nil {
		response.Fail(c, "channel not found", err)
		return
	}
	response.Success(c, "channel fetched", gin.H{"channel": row})
}

func (h *Handlers) handleAdminCreateLLMChannel(c *gin.Context) {
	var req llmChannelWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	row := &models.LLMChannel{}
	if err := req.applyTo(row); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	if row.Key == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("key is required"))
		return
	}
	row.CreatedTime = time.Now().Unix()
	if err := h.db.Create(row).Error; err != nil {
		response.Fail(c, "create channel failed", err)
		return
	}
	// 同步 abilities（基于 Models + Group）。
	_ = models.SyncLLMAbilitiesFromChannel(h.db, row)
	response.Success(c, "channel created", gin.H{"channel": row})
}

func (h *Handlers) handleAdminUpdateLLMChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.LLMChannel
	if err := h.db.First(&row, id).Error; err != nil {
		response.Fail(c, "channel not found", err)
		return
	}
	var req llmChannelWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	if err := req.applyTo(&row); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.db.Save(&row).Error; err != nil {
		response.Fail(c, "update channel failed", err)
		return
	}
	_ = models.SyncLLMAbilitiesFromChannel(h.db, &row)
	response.Success(c, "channel updated", gin.H{"channel": row})
}

func (h *Handlers) handleAdminDeleteLLMChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err := h.db.Where("channel_id = ?", id).Delete(&models.LLMAbility{}).Error; err != nil {
		response.Fail(c, "delete abilities failed", err)
		return
	}
	if err := h.db.Delete(&models.LLMChannel{}, id).Error; err != nil {
		response.Fail(c, "delete channel failed", err)
		return
	}
	response.Success(c, "channel deleted", nil)
}

// handleAdminSyncLLMChannelAbilities POST /api/admin/llm-channels/:id/sync-abilities
// 按当前渠道 models/group 重建该 channel_id 下的 ability 行。
func (h *Handlers) handleAdminSyncLLMChannelAbilities(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.LLMChannel
	if err := h.db.First(&row, id).Error; err != nil {
		response.Fail(c, "channel not found", err)
		return
	}
	if err := models.SyncLLMAbilitiesFromChannel(h.db, &row); err != nil {
		response.Fail(c, "sync abilities failed", err)
		return
	}
	response.Success(c, "abilities synced", nil)
}

func (h *Handlers) handleAdminListLLMAbilities(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	q := h.db.Model(&models.LLMAbility{})
	if g := strings.TrimSpace(c.Query("group")); g != "" {
		q = q.Where("`group` = ?", g)
	}
	if m := strings.TrimSpace(c.Query("model")); m != "" {
		q = q.Where("model = ?", m)
	}
	if cid := strings.TrimSpace(c.Query("channel_id")); cid != "" {
		if v, err := strconv.Atoi(cid); err == nil {
			q = q.Where("channel_id = ?", v)
		}
	}
	if e := strings.TrimSpace(c.Query("enabled")); e != "" {
		if b, err := strconv.ParseBool(e); err == nil {
			q = q.Where("enabled = ?", b)
		}
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.Fail(c, "list abilities failed", err)
		return
	}
	var rows []models.LLMAbility
	if err := q.Order("priority DESC, model ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		response.Fail(c, "list abilities failed", err)
		return
	}
	response.Success(c, "abilities fetched", gin.H{
		"abilities": rows,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
	})
}

// handleAdminListLLMModelMetas GET /api/admin/llm-model-metas
func (h *Handlers) handleAdminListLLMModelMetas(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	q := h.db.Model(&models.LLMModelMeta{})
	if v := strings.TrimSpace(c.Query("vendor")); v != "" {
		q = q.Where("vendor = ?", v)
	}
	if s := strings.TrimSpace(c.Query("status")); s != "" {
		if i, err := strconv.Atoi(s); err == nil {
			q = q.Where("status = ?", i)
		}
	}
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		like := "%" + s + "%"
		q = q.Where("model_name LIKE ? OR description LIKE ? OR vendor LIKE ? OR tags LIKE ?", like, like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.Fail(c, "list model metas failed", err)
		return
	}
	var rows []models.LLMModelMeta
	if err := q.Order("sort_order DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		response.Fail(c, "list model metas failed", err)
		return
	}
	response.Success(c, "model metas fetched", gin.H{
		"items":    rows,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// handleAdminUpsertLLMModelMeta POST /api/admin/llm-model-metas (create or update by ID)
func (h *Handlers) handleAdminUpsertLLMModelMeta(c *gin.Context) {
	var row models.LLMModelMeta
	if err := c.ShouldBindJSON(&row); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	row.ModelName = strings.TrimSpace(row.ModelName)
	if row.ModelName == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("model_name is required"))
		return
	}
	now := time.Now().Unix()
	row.UpdatedTime = now
	if row.Id == 0 {
		row.CreatedTime = now
		if err := h.db.Create(&row).Error; err != nil {
			response.Fail(c, "create model meta failed", err)
			return
		}
		response.Success(c, "model meta created", gin.H{"meta": row})
		return
	}
	if err := h.db.Save(&row).Error; err != nil {
		response.Fail(c, "update model meta failed", err)
		return
	}
	response.Success(c, "model meta updated", gin.H{"meta": row})
}

func (h *Handlers) handleAdminDeleteLLMModelMeta(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err := h.db.Delete(&models.LLMModelMeta{}, id).Error; err != nil {
		response.Fail(c, "delete model meta failed", err)
		return
	}
	response.Success(c, "model meta deleted", nil)
}
