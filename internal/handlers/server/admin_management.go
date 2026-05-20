package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gin-gonic/gin"
)

type adminGroupUpdateReq struct {
	Name       string                  `json:"name"`
	Type       string                  `json:"type"`
	Extra      string                  `json:"extra"`
	IsArchived *bool                   `json:"isArchived"`
	Permission *models.GroupPermission `json:"permission"`
}

type adminCredentialStatusReq struct {
	Status         string  `json:"status"`
	BannedReason   string  `json:"bannedReason"`
	ExpiresAt      *string `json:"expiresAt"`
	TokenQuota     *int64  `json:"tokenQuota"`
	RequestQuota   *int64  `json:"requestQuota"`
	UseNativeQuota *bool   `json:"useNativeQuota"`
	UnlimitedQuota *bool   `json:"unlimitedQuota"`
}

type adminAgentUpdateReq struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	SystemPrompt     string   `json:"systemPrompt"`
	Temperature      *float32 `json:"temperature"`
	MaxTokens        *int     `json:"maxTokens"`
	Speaker          string   `json:"speaker"`
	TtsProvider      string   `json:"ttsProvider"`
	LLMModel         string   `json:"llmModel"`
	EnableJSONOutput *bool    `json:"enableJSONOutput"`
}
func (h *Handlers) handleAdminListGroups(c *gin.Context) {
	page, pageSize := utils.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	groupType := strings.TrimSpace(c.Query("type"))
	isArchivedRaw := strings.TrimSpace(c.Query("isArchived"))

	query := h.db.Model(&models.Group{}).Preload("Creator")
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR `type` LIKE ?", like, like)
	}
	if groupType != "" {
		query = query.Where("`type` = ?", groupType)
	}
	if isArchivedRaw != "" {
		if isArchived, err := strconv.ParseBool(isArchivedRaw); err == nil {
			query = query.Where("is_archived = ?", isArchived)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list groups failed", err)
		return
	}

	var groups []models.Group
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&groups).Error; err != nil {
		response.Fail(c, "list groups failed", err)
		return
	}

	type groupWithCount struct {
		models.Group
		MemberCount int64 `json:"memberCount"`
	}
	result := make([]groupWithCount, 0, len(groups))
	for _, g := range groups {
		var memberCount int64
		_ = h.db.Model(&models.GroupMember{}).Where("group_id = ?", g.ID).Count(&memberCount).Error
		result = append(result, groupWithCount{
			Group:       g,
			MemberCount: memberCount,
		})
	}

	response.Success(c, "groups fetched", gin.H{
		"groups":    result,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
		"page_size": pageSize,
	})
}

func (h *Handlers) handleAdminGetGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid group id"))
		return
	}
	var group models.Group
	if err = h.db.Preload("Creator").First(&group, id).Error; err != nil {
		response.Fail(c, "group not found", err)
		return
	}
	response.Success(c, "group fetched", group)
}

func (h *Handlers) handleAdminUpdateGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid group id"))
		return
	}
	var req adminGroupUpdateReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	var group models.Group
	if err = h.db.First(&group, id).Error; err != nil {
		response.Fail(c, "group not found", err)
		return
	}

	updateVals := map[string]any{}
	if req.Name != "" {
		updateVals["name"] = req.Name
	}
	if req.Type != "" {
		updateVals["type"] = req.Type
	}
	if req.Extra != "" {
		updateVals["extra"] = req.Extra
	}
	if req.IsArchived != nil {
		updateVals["is_archived"] = *req.IsArchived
	}
	if req.Permission != nil {
		updateVals["permission"] = *req.Permission
	}
	if len(updateVals) == 0 {
		response.Success(c, "nothing changed", group)
		return
	}

	if err = h.db.Model(&group).Updates(updateVals).Error; err != nil {
		response.Fail(c, "update group failed", err)
		return
	}
	_ = h.db.Preload("Creator").First(&group, group.ID).Error
	response.Success(c, "group updated", group)
}

func (h *Handlers) handleAdminDeleteGroup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid group id"))
		return
	}
	if err = h.db.Delete(&models.Group{}, id).Error; err != nil {
		response.Fail(c, "delete group failed", err)
		return
	}
	response.Success(c, "group deleted", nil)
}

func (h *Handlers) handleAdminListCredentials(c *gin.Context) {
	page, pageSize := utils.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.Query("status"))
	userIDRaw := strings.TrimSpace(c.Query("user_id"))

	query := h.db.Model(&models.UserCredential{})
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR api_key LIKE ? OR llm_provider LIKE ?", like, like, like)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if userIDRaw != "" {
		if uid, err := strconv.ParseUint(userIDRaw, 10, 64); err == nil {
			query = query.Where("user_id = ?", uid)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list credentials failed", err)
		return
	}
	var creds []models.UserCredential
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&creds).Error; err != nil {
		response.Fail(c, "list credentials failed", err)
		return
	}
	response.Success(c, "credentials fetched", gin.H{
		"credentials": creds,
		"total":       total,
		"page":        page,
		"pageSize":    pageSize,
		"page_size":   pageSize,
	})
}

func (h *Handlers) handleAdminGetCredential(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid credential id"))
		return
	}
	var cred models.UserCredential
	if err = h.db.First(&cred, id).Error; err != nil {
		response.Fail(c, "credential not found", err)
		return
	}
	response.Success(c, "credential fetched", cred)
}

func (h *Handlers) handleAdminUpdateCredentialStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid credential id"))
		return
	}
	var req adminCredentialStatusReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	var cred models.UserCredential
	if err = h.db.First(&cred, id).Error; err != nil {
		response.Fail(c, "credential not found", err)
		return
	}

	status := models.CredentialStatus(strings.TrimSpace(req.Status))
	updateVals := map[string]any{
		"status": status,
	}
	switch status {
	case models.CredentialStatusActive:
		updateVals["banned_at"] = nil
		updateVals["banned_reason"] = ""
		updateVals["banned_by"] = nil
	case models.CredentialStatusBanned:
		now := time.Now()
		updateVals["banned_at"] = &now
		updateVals["banned_reason"] = req.BannedReason
	case models.CredentialStatusSuspended:
		// no extra fields
	default:
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid credential status"))
		return
	}
	if req.ExpiresAt != nil {
		raw := strings.TrimSpace(*req.ExpiresAt)
		if raw == "" {
			updateVals["expires_at"] = nil
		} else {
			var parsed time.Time
			var parseErr error
			if strings.Contains(raw, "T") {
				parsed, parseErr = time.Parse(time.RFC3339, raw)
			} else {
				parsed, parseErr = time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local)
			}
			if parseErr != nil {
				response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid expiresAt format, expected 'YYYY-MM-DD HH:MM:SS' or RFC3339"))
				return
			}
			updateVals["expires_at"] = &parsed
		}
	}
	if req.TokenQuota != nil {
		if *req.TokenQuota < 0 {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("tokenQuota must be >= 0"))
			return
		}
		updateVals["token_quota"] = *req.TokenQuota
	}
	if req.RequestQuota != nil {
		if *req.RequestQuota < 0 {
			response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("requestQuota must be >= 0"))
			return
		}
		updateVals["request_quota"] = *req.RequestQuota
	}
	if req.UseNativeQuota != nil {
		updateVals["use_native_quota"] = *req.UseNativeQuota
	}
	if req.UnlimitedQuota != nil {
		updateVals["unlimited_quota"] = *req.UnlimitedQuota
	}
	if err = h.db.Model(&cred).Updates(updateVals).Error; err != nil {
		response.Fail(c, "update credential status failed", err)
		return
	}
	_ = h.db.First(&cred, cred.ID).Error
	response.Success(c, "credential status updated", cred)
}

func (h *Handlers) handleAdminDeleteCredential(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid credential id"))
		return
	}
	if err = h.db.Delete(&models.UserCredential{}, id).Error; err != nil {
		response.Fail(c, "delete credential failed", err)
		return
	}
	response.Success(c, "credential deleted", nil)
}

func (h *Handlers) handleAdminListJSTemplates(c *gin.Context) {
	page, pageSize := utils.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.Query("status"))
	templateType := strings.TrimSpace(c.Query("type"))
	userIDRaw := strings.TrimSpace(c.Query("user_id"))

	query := h.db.Model(&models.JSTemplate{})
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR js_source_id LIKE ?", like, like)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if templateType != "" {
		query = query.Where("type = ?", templateType)
	}
	if userIDRaw != "" {
		if uid, err := strconv.ParseUint(userIDRaw, 10, 64); err == nil {
			query = query.Where("user_id = ?", uid)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list js templates failed", err)
		return
	}
	var templates []models.JSTemplate
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&templates).Error; err != nil {
		response.Fail(c, "list js templates failed", err)
		return
	}
	response.Success(c, "js templates fetched", gin.H{
		"templates": templates,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
		"page_size": pageSize,
	})
}

func (h *Handlers) handleAdminGetJSTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid js template id"))
		return
	}
	var template models.JSTemplate
	if err := h.db.Where("id = ?", id).First(&template).Error; err != nil {
		response.Fail(c, "js template not found", err)
		return
	}
	response.Success(c, "js template fetched", template)
}

func (h *Handlers) handleAdminUpdateJSTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid js template id"))
		return
	}
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	delete(req, "id")
	delete(req, "user_id")
	delete(req, "created_at")
	delete(req, "createdAt")
	if len(req) == 0 {
		response.Success(c, "nothing changed", nil)
		return
	}

	if err := h.db.Model(&models.JSTemplate{}).Where("id = ?", id).Updates(req).Error; err != nil {
		response.Fail(c, "update js template failed", err)
		return
	}
	var template models.JSTemplate
	_ = h.db.Where("id = ?", id).First(&template).Error
	response.Success(c, "js template updated", template)
}

func (h *Handlers) handleAdminDeleteJSTemplate(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid js template id"))
		return
	}
	if err := h.db.Where("id = ?", id).Delete(&models.JSTemplate{}).Error; err != nil {
		response.Fail(c, "delete js template failed", err)
		return
	}
	response.Success(c, "js template deleted", nil)
}

func (h *Handlers) handleAdminListBills(c *gin.Context) {
	page, pageSize := utils.ParsePagination(c)
	status := strings.TrimSpace(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))
	userIDRaw := strings.TrimSpace(c.Query("user_id"))
	groupIDRaw := strings.TrimSpace(c.Query("group_id"))

	query := h.db.Model(&models.Bill{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("bill_no LIKE ? OR title LIKE ?", like, like)
	}
	if userIDRaw != "" {
		if uid, err := strconv.ParseUint(userIDRaw, 10, 64); err == nil {
			query = query.Where("user_id = ?", uid)
		}
	}
	if groupIDRaw != "" {
		if gid, err := strconv.ParseUint(groupIDRaw, 10, 64); err == nil {
			query = query.Where("group_id = ?", gid)
		}
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list bills failed", err)
		return
	}
	var bills []models.Bill
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&bills).Error; err != nil {
		response.Fail(c, "list bills failed", err)
		return
	}
	response.Success(c, "bills fetched", gin.H{
		"bills":     bills,
		"total":     total,
		"page":      page,
		"pageSize":  pageSize,
		"page_size": pageSize,
	})
}

func (h *Handlers) handleAdminGetBill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid bill id"))
		return
	}
	var bill models.Bill
	if err = h.db.First(&bill, id).Error; err != nil {
		response.Fail(c, "bill not found", err)
		return
	}
	response.Success(c, "bill fetched", bill)
}

func (h *Handlers) handleAdminUpdateBill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid bill id"))
		return
	}
	var req struct {
		Title  string `json:"title"`
		Notes  string `json:"notes"`
		Status string `json:"status"`
	}
	if err = c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	updateVals := map[string]any{}
	if req.Title != "" {
		updateVals["title"] = req.Title
	}
	if req.Notes != "" {
		updateVals["notes"] = req.Notes
	}
	if req.Status != "" {
		updateVals["status"] = req.Status
	}
	if len(updateVals) == 0 {
		response.Success(c, "nothing changed", nil)
		return
	}
	if err = h.db.Model(&models.Bill{}).Where("id = ?", id).Updates(updateVals).Error; err != nil {
		response.Fail(c, "update bill failed", err)
		return
	}
	var bill models.Bill
	_ = h.db.First(&bill, id).Error
	response.Success(c, "bill updated", bill)
}

func (h *Handlers) handleAdminDeleteBill(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid bill id"))
		return
	}
	if err = h.db.Delete(&models.Bill{}, id).Error; err != nil {
		response.Fail(c, "delete bill failed", err)
		return
	}
	response.Success(c, "bill deleted", nil)
}

func (h *Handlers) handleAdminListAgents(c *gin.Context) {
	page, pageSize := utils.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))

	query := h.db.Model(&models.Agent{})
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("name LIKE ? OR description LIKE ? OR persona_tag LIKE ? OR llm_model LIKE ?", like, like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list assistants failed", err)
		return
	}

	var assistants []models.Agent
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&assistants).Error; err != nil {
		response.Fail(c, "list assistants failed", err)
		return
	}

	response.Success(c, "agents fetched", gin.H{
		"agents":   assistants,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetAgent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid assistant id"))
		return
	}
	var assistant models.Agent
	if err = h.db.First(&assistant, id).Error; err != nil {
		response.Fail(c, "assistant not found", err)
		return
	}
	response.Success(c, "assistant fetched", assistant)
}

func (h *Handlers) handleAdminUpdateAgent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid assistant id"))
		return
	}
	var req adminAgentUpdateReq
	if err = c.ShouldBindJSON(&req); err != nil {
		response.AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}
	var assistant models.Agent
	if err = h.db.First(&assistant, id).Error; err != nil {
		response.Fail(c, "assistant not found", err)
		return
	}
	updateVals := map[string]any{}
	if req.Name != "" {
		updateVals["name"] = strings.TrimSpace(req.Name)
	}
	if req.Description != "" {
		updateVals["description"] = strings.TrimSpace(req.Description)
	}
	if req.SystemPrompt != "" {
		updateVals["system_prompt"] = req.SystemPrompt
	}
	if req.Temperature != nil {
		updateVals["temperature"] = *req.Temperature
	}
	if req.MaxTokens != nil {
		updateVals["max_tokens"] = *req.MaxTokens
	}
	if req.Speaker != "" {
		updateVals["speaker"] = strings.TrimSpace(req.Speaker)
	}
	if req.TtsProvider != "" {
		updateVals["tts_provider"] = strings.TrimSpace(req.TtsProvider)
	}
	if req.LLMModel != "" {
		updateVals["llm_model"] = strings.TrimSpace(req.LLMModel)
	}
	if req.EnableJSONOutput != nil {
		updateVals["enable_json_output"] = *req.EnableJSONOutput
	}

	if len(updateVals) == 0 {
		response.Success(c, "nothing changed", assistant)
		return
	}
	if err = h.db.Model(&assistant).Updates(updateVals).Error; err != nil {
		response.Fail(c, "update assistant failed", err)
		return
	}
	_ = h.db.First(&assistant, id).Error
	response.Success(c, "assistant updated", assistant)
}

func (h *Handlers) handleAdminDeleteAgent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid assistant id"))
		return
	}
	if err = h.db.Delete(&models.Agent{}, id).Error; err != nil {
		response.Fail(c, "delete assistant failed", err)
		return
	}
	response.Success(c, "assistant deleted", nil)
}

func (h *Handlers) handleAdminListChatSessions(c *gin.Context) {
	page, pageSize := utils.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))

	query := h.db.Model(&models.ChatSession{})
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("id LIKE ? OR user_id LIKE ? OR title LIKE ? OR provider LIKE ? OR model LIKE ?", like, like, like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list chat sessions failed", err)
		return
	}
	var items []models.ChatSession
	if err := query.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Fail(c, "list chat sessions failed", err)
		return
	}
	response.Success(c, "chat sessions fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminListChatMessages(c *gin.Context) {
	page, pageSize := utils.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	sessionID := strings.TrimSpace(c.Query("session_id"))

	query := h.db.Model(&models.ChatMessage{})
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("id LIKE ? OR session_id LIKE ? OR role LIKE ? OR content LIKE ? OR request_id LIKE ?", like, like, like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list chat messages failed", err)
		return
	}
	var items []models.ChatMessage
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Fail(c, "list chat messages failed", err)
		return
	}
	response.Success(c, "chat messages fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminListLLMUsage(c *gin.Context) {
	page, pageSize := utils.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	sessionID := strings.TrimSpace(c.Query("session_id"))
	success := strings.TrimSpace(c.Query("success"))

	query := h.db.Model(&models.LLMUsage{})
	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if success != "" {
		if parsed, err := strconv.ParseBool(success); err == nil {
			query = query.Where("success = ?", parsed)
		}
	}
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("request_id LIKE ? OR session_id LIKE ? OR provider LIKE ? OR model LIKE ? OR ip_address LIKE ? OR user_agent LIKE ? OR response_content LIKE ?", like, like, like, like, like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list llm usage failed", err)
		return
	}
	var items []models.LLMUsage
	if err := query.Order("requested_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Fail(c, "list llm usage failed", err)
		return
	}
	response.Success(c, "llm usage fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminListGenericTable(c *gin.Context, tableName string, searchableCols ...string) {
	page, pageSize := utils.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))

	query := h.db.Table(tableName)
	if search != "" && len(searchableCols) > 0 {
		like := "%" + search + "%"
		parts := make([]string, 0, len(searchableCols))
		args := make([]any, 0, len(searchableCols))
		for _, col := range searchableCols {
			parts = append(parts, col+" LIKE ?")
			args = append(args, like)
		}
		query = query.Where(strings.Join(parts, " OR "), args...)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list records failed", err)
		return
	}

	var items []map[string]any
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Fail(c, "list records failed", err)
		return
	}

	response.Success(c, "records fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetGenericTableItem(c *gin.Context, tableName string) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	var item map[string]any
	if err = h.db.Table(tableName).Where("id = ?", id).Take(&item).Error; err != nil {
		response.Fail(c, "record not found", err)
		return
	}
	response.Success(c, "record fetched", item)
}

func (h *Handlers) handleAdminDeleteGenericTableItem(c *gin.Context, tableName string) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid id"))
		return
	}
	if err = h.db.Table(tableName).Where("id = ?", id).Delete(nil).Error; err != nil {
		response.Fail(c, "delete record failed", err)
		return
	}
	response.Success(c, "record deleted", nil)
}

func (h *Handlers) handleAdminListVoiceTrainingTasks(c *gin.Context) {
	h.handleAdminListGenericTable(c, "voice_training_tasks", "task_name", "task_id", "asset_id")
}

func (h *Handlers) handleAdminGetVoiceTrainingTask(c *gin.Context) {
	h.handleAdminGetGenericTableItem(c, "voice_training_tasks")
}

func (h *Handlers) handleAdminDeleteVoiceTrainingTask(c *gin.Context) {
	h.handleAdminDeleteGenericTableItem(c, "voice_training_tasks")
}

func (h *Handlers) handleAdminListWorkflowDefinitions(c *gin.Context) {
	h.handleAdminListGenericTable(c, "workflow_definitions", "name", "slug", "description", "status")
}

func (h *Handlers) handleAdminGetWorkflowDefinition(c *gin.Context) {
	h.handleAdminGetGenericTableItem(c, "workflow_definitions")
}

func (h *Handlers) handleAdminDeleteWorkflowDefinition(c *gin.Context) {
	h.handleAdminDeleteGenericTableItem(c, "workflow_definitions")
}

func (h *Handlers) handleAdminListNodePlugins(c *gin.Context) {
	h.handleAdminListGenericTable(c, "node_plugins", "name", "slug", "display_name", "category", "status")
}

func (h *Handlers) handleAdminGetNodePlugin(c *gin.Context) {
	h.handleAdminGetGenericTableItem(c, "node_plugins")
}

func (h *Handlers) handleAdminDeleteNodePlugin(c *gin.Context) {
	h.handleAdminDeleteGenericTableItem(c, "node_plugins")
}

func (h *Handlers) handleAdminListWorkflowPlugins(c *gin.Context) {
	h.handleAdminListGenericTable(c, "workflow_plugins", "name", "slug", "display_name", "category", "status")
}

func (h *Handlers) handleAdminGetWorkflowPlugin(c *gin.Context) {
	h.handleAdminGetGenericTableItem(c, "workflow_plugins")
}

func (h *Handlers) handleAdminDeleteWorkflowPlugin(c *gin.Context) {
	h.handleAdminDeleteGenericTableItem(c, "workflow_plugins")
}

func (h *Handlers) handleAdminListInternalNotifications(c *gin.Context) {
	h.handleAdminListGenericTable(c, "internal_notifications", "title", "content")
}

func (h *Handlers) handleAdminGetInternalNotification(c *gin.Context) {
	h.handleAdminGetGenericTableItem(c, "internal_notifications")
}

func (h *Handlers) handleAdminDeleteInternalNotification(c *gin.Context) {
	h.handleAdminDeleteGenericTableItem(c, "internal_notifications")
}

func (h *Handlers) handleAdminListDevices(c *gin.Context) {
	page, pageSize := utils.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))

	query := h.db.Table("devices")
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("id LIKE ? OR mac_address LIKE ? OR device_name LIKE ? OR alias LIKE ?", like, like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.Fail(c, "list devices failed", err)
		return
	}

	var items []map[string]any
	if err := query.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		response.Fail(c, "list devices failed", err)
		return
	}

	response.Success(c, "devices fetched", gin.H{
		"items":    items,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handlers) handleAdminGetDevice(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid device id"))
		return
	}
	var item map[string]any
	if err := h.db.Table("devices").Where("id = ?", id).Take(&item).Error; err != nil {
		response.Fail(c, "device not found", err)
		return
	}
	response.Success(c, "device fetched", item)
}

func (h *Handlers) handleAdminDeleteDevice(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		response.AbortWithJSONError(c, http.StatusBadRequest, errors.New("invalid device id"))
		return
	}
	if err := h.db.Table("devices").Where("id = ?", id).Delete(nil).Error; err != nil {
		response.Fail(c, "delete device failed", err)
		return
	}
	response.Success(c, "device deleted", nil)
}
