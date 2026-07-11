package server

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"
	"strings"

	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/internal/models/auth"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// ============================================================================
// 路由注册
// ============================================================================

func (h *Handlers) registerPresetRoutes(r *gin.RouterGroup) {
	presets := r.Group("presets")
	{
		// 公开接口：浏览内置/公开模板（可选登录）
		presets.GET("", h.ListPresets)
		presets.GET("/:id", h.GetPreset)

		// 需要登录
		authGroup := presets.Group("")
		authGroup.Use(auth.AuthRequired)
		{
			authGroup.POST("", h.CreatePreset)
			authGroup.PUT("/:id", h.UpdatePreset)
			authGroup.DELETE("/:id", h.DeletePreset)

			// 应用模板到实际资源
			authGroup.POST("/:id/apply", h.ApplyPreset)
		}
	}
}

// ============================================================================
// Handler 方法
// ============================================================================

// ListPresets 列出可用的预设模板。
func (h *Handlers) ListPresets(c *gin.Context) {
	var req svcmodels.ListPresetReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Fail(c, "参数错误", nil)
		return
	}

	user := auth.CurrentUser(c)
	var groupIDs []uint
	if user != nil {
		var err error
		groupIDs, err = svcmodels.MemberGroupIDs(h.db, user.ID)
		if err != nil {
			groupIDs = nil // 降级为仅显示公开模板
		}
		// 也包含 system(group_id=0) 的内置模板
		groupIDs = append(groupIDs, 0)
	} else {
		groupIDs = []uint{0}
	}

	result, err := svcmodels.ListPresetTemplates(h.db, groupIDs, &req)
	if err != nil {
		response.Fail(c, "查询失败", err.Error())
		return
	}
	response.Success(c, "查询成功", result)
}

// GetPreset 获取单个预设模板详情。
func (h *Handlers) GetPreset(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		response.Fail(c, "参数错误", "无效的模板ID")
		return
	}

	preset, err := svcmodels.GetPresetTemplate(h.db, id)
	if err != nil {
		response.Fail(c, "模板不存在", nil)
		return
	}

	// 可见性检查
	if preset.Visibility == svcmodels.PresetVisibilityPrivate {
		user := auth.CurrentUser(c)
		if user == nil || preset.CreatedBy != user.ID {
			response.Fail(c, "无权访问", "私有模板仅创建者可见")
			return
		}
	}
	if preset.Visibility == svcmodels.PresetVisibilityGroup {
		user := auth.CurrentUser(c)
		if user != nil && !svcmodels.UserIsGroupMember(h.db, user.ID, preset.GroupID) {
			response.Fail(c, "无权访问", "组织模板仅成员可见")
			return
		}
	}

	response.Success(c, "查询成功", preset)
}

// CreatePreset 创建新的预设模板。
func (h *Handlers) CreatePreset(c *gin.Context) {
	var req svcmodels.CreatePresetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	user := auth.CurrentUser(c)

	// 校验模板类型
	if !isValidPresetType(req.Type) {
		response.Fail(c, "无效的模板类型", fmt.Sprintf("支持的类型: %s", strings.Join(svcmodels.ValidPresetTypes, ", ")))
		return
	}

	// 校验 content 是合法 JSON
	if !json.Valid([]byte(req.Content)) {
		response.Fail(c, "模板内容格式错误", "content 必须是有效的 JSON")
		return
	}

	// 解析 groupID（使用默认个人空间）
	gid, err := svcmodels.ResolveWriteGroupID(h.db, user.ID, nil)
	if err != nil {
		response.Fail(c, "解析组织失败", err.Error())
		return
	}

	visibility := strings.TrimSpace(req.Visibility)
	if visibility == "" {
		visibility = svcmodels.PresetVisibilityPrivate
	}

	preset := svcmodels.PresetTemplate{
		GroupID:     gid,
		CreatedBy:   user.ID,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		Type:        req.Type,
		Category:    strings.TrimSpace(req.Category),
		Tags:        strings.TrimSpace(req.Tags),
		Visibility:  visibility,
		Content:     req.Content,
		Status:      "active",
	}

	if err := h.db.Create(&preset).Error; err != nil {
		response.Fail(c, "创建失败", err.Error())
		return
	}

	response.Success(c, "创建成功", preset)
}

// UpdatePreset 更新预设模板。
func (h *Handlers) UpdatePreset(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		response.Fail(c, "参数错误", "无效的模板ID")
		return
	}

	var preset svcmodels.PresetTemplate
	if err := h.db.First(&preset, id).Error; err != nil {
		response.Fail(c, "模板不存在", nil)
		return
	}

	// 内置模板不可编辑
	if preset.IsBuiltin {
		response.Fail(c, "内置模板不可编辑", nil)
		return
	}

	user := auth.CurrentUser(c)
	if preset.CreatedBy != user.ID {
		// 检查是否为组织管理员
		if !svcmodels.CanManageTenantResource(h.db, user.ID, preset.GroupID, preset.CreatedBy) {
			response.Fail(c, "无权编辑", "仅创建者或组织管理员可编辑")
			return
		}
	}

	var req svcmodels.UpdatePresetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = strings.TrimSpace(req.Name)
	}
	if req.Description != "" {
		updates["description"] = strings.TrimSpace(req.Description)
	}
	if req.Category != "" {
		updates["category"] = strings.TrimSpace(req.Category)
	}
	if req.Tags != "" {
		updates["tags"] = strings.TrimSpace(req.Tags)
	}
	if req.Visibility != "" {
		updates["visibility"] = strings.TrimSpace(req.Visibility)
	}
	if req.Content != "" {
		if !json.Valid([]byte(req.Content)) {
			response.Fail(c, "模板内容格式错误", "content 必须是有效的 JSON")
			return
		}
		updates["content"] = req.Content
	}

	if len(updates) == 0 {
		response.Fail(c, "无更新内容", nil)
		return
	}

	if err := h.db.Model(&preset).Updates(updates).Error; err != nil {
		response.Fail(c, "更新失败", err.Error())
		return
	}

	// 重新查询
	h.db.First(&preset, id)
	response.Success(c, "更新成功", preset)
}

// DeletePreset 删除（归档）预设模板。
func (h *Handlers) DeletePreset(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		response.Fail(c, "参数错误", "无效的模板ID")
		return
	}

	var preset svcmodels.PresetTemplate
	if err := h.db.First(&preset, id).Error; err != nil {
		response.Fail(c, "模板不存在", nil)
		return
	}

	if preset.IsBuiltin {
		response.Fail(c, "内置模板不可删除", nil)
		return
	}

	user := auth.CurrentUser(c)
	if preset.CreatedBy != user.ID {
		if !svcmodels.CanManageTenantResource(h.db, user.ID, preset.GroupID, preset.CreatedBy) {
			response.Fail(c, "无权删除", nil)
			return
		}
	}

	// 软删除（归档）
	if err := h.db.Model(&preset).Update("status", "archived").Error; err != nil {
		response.Fail(c, "删除失败", err.Error())
		return
	}

	response.Success(c, "删除成功", nil)
}

// ApplyPreset 将预设模板应用到实际资源。
func (h *Handlers) ApplyPreset(c *gin.Context) {
	id, err := parseIDParam(c, "id")
	if err != nil {
		response.Fail(c, "参数错误", "无效的模板ID")
		return
	}

	var req svcmodels.ApplyPresetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}
	req.PresetID = id

	preset, err := svcmodels.GetPresetTemplate(h.db, id)
	if err != nil {
		response.Fail(c, "模板不存在", nil)
		return
	}

	// 根据类型执行不同的应用逻辑
	switch preset.Type {
	case svcmodels.PresetTypeAgent:
		h.applyAgentPreset(c, preset)
	case svcmodels.PresetTypeSystemPrompt:
		h.applySystemPromptPreset(c, preset, &req)
	case svcmodels.PresetTypeVoice:
		h.applyVoicePreset(c, preset, &req)
	case svcmodels.PresetTypeKnowledge:
		h.applyKnowledgePreset(c, preset)
	default:
		response.Fail(c, "不支持的模板类型", preset.Type)
	}
}

// ============================================================================
// 各类模板的应用逻辑
// ============================================================================

// applyAgentPreset 使用 agent 模板创建新 Agent。
func (h *Handlers) applyAgentPreset(c *gin.Context, preset *svcmodels.PresetTemplate) {
	var payload svcmodels.AgentPresetPayload
	if err := json.Unmarshal([]byte(preset.Content), &payload); err != nil {
		response.Fail(c, "模板内容解析失败", err.Error())
		return
	}

	user := auth.CurrentUser(c)
	gid, err := svcmodels.ResolveWriteGroupID(h.db, user.ID, nil)
	if err != nil {
		response.Fail(c, "解析组织失败", err.Error())
		return
	}

	enableVAD := true
	if payload.EnableVAD != nil {
		enableVAD = *payload.EnableVAD
	}
	vadThreshold := 500.0
	if payload.VADThreshold != nil {
		vadThreshold = *payload.VADThreshold
	}
	vadFrames := 2
	if payload.VADConsecutiveFrames != nil {
		vadFrames = *payload.VADConsecutiveFrames
	}

	agent := svcmodels.Agent{
		GroupID:              gid,
		CreatedBy:            user.ID,
		Name:                 payload.Name,
		SystemPrompt:         payload.SystemPrompt,
		OpeningStatement:     payload.OpeningStatement,
		PersonaTag:           payload.PersonaTag,
		Temperature:          payload.Temperature,
		MaxTokens:            payload.MaxTokens,
		LLMModel:             payload.LLMModel,
		Speaker:              payload.Speaker,
		VoiceCloneID:         payload.VoiceCloneID,
		TtsProvider:          payload.TtsProvider,
		EnableVAD:            enableVAD,
		VADThreshold:         vadThreshold,
		VADConsecutiveFrames: vadFrames,
	}

	if payload.EnableJSONOutput != nil {
		agent.EnableJSONOutput = *payload.EnableJSONOutput
	}

	if err := h.db.Create(&agent).Error; err != nil {
		response.Fail(c, "创建Agent失败", err.Error())
		return
	}

	// 增加使用计数
	_ = svcmodels.IncrementPresetUseCount(h.db, preset.ID)

	response.Success(c, "应用成功，已创建Agent", agent)
}

// applySystemPromptPreset 返回提示词内容供前端使用。
func (h *Handlers) applySystemPromptPreset(c *gin.Context, preset *svcmodels.PresetTemplate, req *svcmodels.ApplyPresetReq) {
	var payload svcmodels.SystemPromptPresetPayload
	if err := json.Unmarshal([]byte(preset.Content), &payload); err != nil {
		response.Fail(c, "模板内容解析失败", err.Error())
		return
	}

	// 变量替换
	systemPrompt := payload.SystemPrompt
	if req.Variables != nil {
		for k, v := range req.Variables {
			systemPrompt = strings.ReplaceAll(systemPrompt, "{{"+k+"}}", v)
		}
	}

	// 如果指定了 agentId，直接更新该 Agent 的 SystemPrompt
	if req.AgentID != nil {
		user := auth.CurrentUser(c)
		var agent svcmodels.Agent
		if err := h.db.First(&agent, *req.AgentID).Error; err != nil {
			response.Fail(c, "Agent不存在", nil)
			return
		}
		if !svcmodels.CanManageTenantResource(h.db, user.ID, agent.GroupID, agent.CreatedBy) {
			response.Fail(c, "无权修改此Agent", nil)
			return
		}
		if err := h.db.Model(&agent).Update("system_prompt", systemPrompt).Error; err != nil {
			response.Fail(c, "更新失败", err.Error())
			return
		}
		_ = svcmodels.IncrementPresetUseCount(h.db, preset.ID)
		response.Success(c, "已应用提示词到Agent", gin.H{
			"agentId":      *req.AgentID,
			"systemPrompt": systemPrompt,
		})
		return
	}

	_ = svcmodels.IncrementPresetUseCount(h.db, preset.ID)
	response.Success(c, "应用成功", gin.H{
		"systemPrompt": systemPrompt,
		"personaTag":   payload.PersonaTag,
		"variables":    payload.Variables,
	})
}

// applyVoicePreset 将语音预设应用到指定 Agent。
func (h *Handlers) applyVoicePreset(c *gin.Context, preset *svcmodels.PresetTemplate, req *svcmodels.ApplyPresetReq) {
	if req.AgentID == nil {
		response.Fail(c, "缺少 agentId", "语音预设需要指定目标Agent")
		return
	}

	var payload svcmodels.VoicePresetPayload
	if err := json.Unmarshal([]byte(preset.Content), &payload); err != nil {
		response.Fail(c, "模板内容解析失败", err.Error())
		return
	}

	user := auth.CurrentUser(c)
	var agent svcmodels.Agent
	if err := h.db.First(&agent, *req.AgentID).Error; err != nil {
		response.Fail(c, "Agent不存在", nil)
		return
	}
	if !svcmodels.CanManageTenantResource(h.db, user.ID, agent.GroupID, agent.CreatedBy) {
		response.Fail(c, "无权修改此Agent", nil)
		return
	}

	updates := map[string]interface{}{
		"speaker":                payload.Speaker,
		"tts_provider":          payload.TtsProvider,
		"voice_clone_id":        payload.VoiceCloneID,
		"enable_vad":            payload.EnableVAD,
		"vad_threshold":         payload.VADThreshold,
		"vad_consecutive_frames": payload.VADConsecutiveFrames,
	}
	if err := h.db.Model(&agent).Updates(updates).Error; err != nil {
		response.Fail(c, "更新失败", err.Error())
		return
	}

	_ = svcmodels.IncrementPresetUseCount(h.db, preset.ID)
	response.Success(c, "语音配置已应用", gin.H{
		"agentId": *req.AgentID,
		"voice":   payload,
	})
}

// applyKnowledgePreset 使用知识库模板创建知识库配置。
func (h *Handlers) applyKnowledgePreset(c *gin.Context, preset *svcmodels.PresetTemplate) {
	var payload svcmodels.KnowledgePresetPayload
	if err := json.Unmarshal([]byte(preset.Content), &payload); err != nil {
		response.Fail(c, "模板内容解析失败", err.Error())
		return
	}

	user := auth.CurrentUser(c)
	gid, err := svcmodels.ResolveWriteGroupID(h.db, user.ID, nil)
	if err != nil {
		response.Fail(c, "解析组织失败", err.Error())
		return
	}

	req := &svcmodels.KnowledgeNamespaceCreateUpdate{
		Namespace:   payload.Namespace,
		Name:        payload.Name,
		Description: payload.Description,
		VectorDim:   payload.VectorDim,
	}

	ns, err := svcmodels.UpsertKnowledgeNamespace(h.db, gid, user.ID, 0, req)
	if err != nil {
		response.Fail(c, "创建知识库失败", err.Error())
		return
	}

	_ = svcmodels.IncrementPresetUseCount(h.db, preset.ID)
	response.Success(c, "知识库已创建", ns)
}

// ============================================================================
// 辅助函数
// ============================================================================

func isValidPresetType(t string) bool {
	for _, v := range svcmodels.ValidPresetTypes {
		if v == t {
			return true
		}
	}
	return false
}

// parseIDParam 从路径参数中解析 int64 ID。
func parseIDParam(c *gin.Context, name string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(c.Param(name), "%d", &id)
	return id, err
}
