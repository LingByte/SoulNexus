// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 语音渠道（ASR / TTS）admin CRUD：列表 / 详情 / 创建 / 更新 / 删除。
// 与 LLMChannel 风格一致，但不需要 ability 同步。
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

// speechChannelWriteReq 创建/更新语音渠道入参（ASR/TTS 共用）。
type speechChannelWriteReq struct {
	Provider   string `json:"provider"`
	Name       string `json:"name"`
	Enabled    *bool  `json:"enabled"`
	Group      string `json:"group"`
	SortOrder  *int   `json:"sort_order"`
	Models     string `json:"models"`
	ConfigJSON string `json:"config_json"`
}

func (req *speechChannelWriteReq) trim() {
	req.Provider = strings.TrimSpace(req.Provider)
	req.Name = strings.TrimSpace(req.Name)
	req.Group = strings.TrimSpace(req.Group)
	req.Models = strings.TrimSpace(req.Models)
	req.ConfigJSON = strings.TrimSpace(req.ConfigJSON)
}

func (req *speechChannelWriteReq) validate() error {
	if req.Provider == "" {
		return errors.New("provider required")
	}
	if req.Name == "" {
		return errors.New("name required")
	}
	return nil
}

// applyToASR 将入参写入 ASRChannel；保留未指定字段。
func (req *speechChannelWriteReq) applyToASR(row *models.ASRChannel) {
	row.Provider = req.Provider
	row.Name = req.Name
	row.Group = req.Group
	row.Models = req.Models
	row.ConfigJSON = req.ConfigJSON
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.SortOrder != nil {
		row.SortOrder = *req.SortOrder
	}
}

// applyToTTS 将入参写入 TTSChannel；保留未指定字段。
func (req *speechChannelWriteReq) applyToTTS(row *models.TTSChannel) {
	row.Provider = req.Provider
	row.Name = req.Name
	row.Group = req.Group
	row.Models = req.Models
	row.ConfigJSON = req.ConfigJSON
	if req.Enabled != nil {
		row.Enabled = *req.Enabled
	}
	if req.SortOrder != nil {
		row.SortOrder = *req.SortOrder
	}
}

// ---------- ASR ----------

func (h *Handlers) handleAdminListASRChannels(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	q := h.db.Model(&models.ASRChannel{})
	if g := strings.TrimSpace(c.Query("group")); g != "" {
		q = q.Where("`group` = ?", g)
	}
	if p := strings.TrimSpace(c.Query("provider")); p != "" {
		q = q.Where("provider = ?", p)
	}
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		like := "%" + s + "%"
		q = q.Where("name LIKE ? OR provider LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.Fail(c, "list asr channels failed", err)
		return
	}
	var rows []models.ASRChannel
	if err := q.Order("sort_order DESC, id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		response.Fail(c, "list asr channels failed", err)
		return
	}
	response.Success(c, "asr channels fetched", gin.H{
		"channels": rows,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetASRChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.ASRChannel
	if err := h.db.First(&row, id).Error; err != nil {
		response.Fail(c, "asr channel not found", err)
		return
	}
	response.Success(c, "asr channel fetched", gin.H{"channel": row})
}

func (h *Handlers) handleAdminCreateASRChannel(c *gin.Context) {
	var req speechChannelWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	req.trim()
	if err := req.validate(); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	row := &models.ASRChannel{Enabled: true}
	req.applyToASR(row)
	if err := h.db.Create(row).Error; err != nil {
		response.Fail(c, "create asr channel failed", err)
		return
	}
	response.Success(c, "asr channel created", gin.H{"channel": row})
}

func (h *Handlers) handleAdminUpdateASRChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.ASRChannel
	if err := h.db.First(&row, id).Error; err != nil {
		response.Fail(c, "asr channel not found", err)
		return
	}
	var req speechChannelWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	req.trim()
	if err := req.validate(); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	req.applyToASR(&row)
	if err := h.db.Save(&row).Error; err != nil {
		response.Fail(c, "update asr channel failed", err)
		return
	}
	response.Success(c, "asr channel updated", gin.H{"channel": row})
}

func (h *Handlers) handleAdminDeleteASRChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err := h.db.Delete(&models.ASRChannel{}, id).Error; err != nil {
		response.Fail(c, "delete asr channel failed", err)
		return
	}
	response.Success(c, "asr channel deleted", nil)
}

// ---------- TTS ----------

func (h *Handlers) handleAdminListTTSChannels(c *gin.Context) {
	page, pageSize := h.parsePagination(c)
	q := h.db.Model(&models.TTSChannel{})
	if g := strings.TrimSpace(c.Query("group")); g != "" {
		q = q.Where("`group` = ?", g)
	}
	if p := strings.TrimSpace(c.Query("provider")); p != "" {
		q = q.Where("provider = ?", p)
	}
	if s := strings.TrimSpace(c.Query("search")); s != "" {
		like := "%" + s + "%"
		q = q.Where("name LIKE ? OR provider LIKE ?", like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.Fail(c, "list tts channels failed", err)
		return
	}
	var rows []models.TTSChannel
	if err := q.Order("sort_order DESC, id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		response.Fail(c, "list tts channels failed", err)
		return
	}
	response.Success(c, "tts channels fetched", gin.H{
		"channels": rows,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetTTSChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.TTSChannel
	if err := h.db.First(&row, id).Error; err != nil {
		response.Fail(c, "tts channel not found", err)
		return
	}
	response.Success(c, "tts channel fetched", gin.H{"channel": row})
}

func (h *Handlers) handleAdminCreateTTSChannel(c *gin.Context) {
	var req speechChannelWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	req.trim()
	if err := req.validate(); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	row := &models.TTSChannel{Enabled: true}
	req.applyToTTS(row)
	if err := h.db.Create(row).Error; err != nil {
		response.Fail(c, "create tts channel failed", err)
		return
	}
	response.Success(c, "tts channel created", gin.H{"channel": row})
}

func (h *Handlers) handleAdminUpdateTTSChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var row models.TTSChannel
	if err := h.db.First(&row, id).Error; err != nil {
		response.Fail(c, "tts channel not found", err)
		return
	}
	var req speechChannelWriteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	req.trim()
	if err := req.validate(); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	req.applyToTTS(&row)
	if err := h.db.Save(&row).Error; err != nil {
		response.Fail(c, "update tts channel failed", err)
		return
	}
	response.Success(c, "tts channel updated", gin.H{"channel": row})
}

func (h *Handlers) handleAdminDeleteTTSChannel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err := h.db.Delete(&models.TTSChannel{}, id).Error; err != nil {
		response.Fail(c, "delete tts channel failed", err)
		return
	}
	response.Success(c, "tts channel deleted", nil)
}
