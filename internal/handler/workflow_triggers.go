package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/LingByte/SoulNexus/internal/models"
	workflowdef "github.com/LingByte/SoulNexus/internal/workflow"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/gin-gonic/gin"
)

// RegisterPublicWorkflowRoutes 注册公开的工作流API路由（不需要认证）
func (h *Handlers) RegisterPublicWorkflowRoutes(r *gin.RouterGroup) {
	public := r.Group("/public/workflows")
	{
		// 通过 slug 和 API key 触发工作流
		public.POST("/:slug/execute", h.ExecutePublicWorkflow)
		// 通过 webhook 触发工作流
		public.POST("/webhook/:slug", h.WebhookTriggerWorkflow)
	}
}

// ExecutePublicWorkflow 公开API触发工作流
func (h *Handlers) ExecutePublicWorkflow(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		response.Fail(c, "invalid slug", nil)
		return
	}

	// 获取 API Key（从 Header 或 Query）
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		apiKey = c.Query("apiKey")
	}

	// 查找工作流定义
	var def models.WorkflowDefinition
	if err := h.db.Where("slug = ? AND status = ?", slug, "active").First(&def).Error; err != nil {
		response.Fail(c, "workflow not found or not active", nil)
		return
	}

	// 解析触发器配置
	config, err := workflowdef.ParseTriggerConfig(&def)
	if err != nil {
		response.Fail(c, "invalid trigger config", err.Error())
		return
	}

	// 检查 API 触发是否启用
	if config.API == nil || !config.API.Enabled {
		response.Fail(c, "API trigger is not enabled for this workflow", nil)
		return
	}

	// 检查是否为公开API
	if !config.API.Public {
		response.Fail(c, "this workflow requires authentication", nil)
		return
	}

	// 验证 API Key
	if config.API.APIKey != "" && config.API.APIKey != apiKey {
		response.Fail(c, "invalid API key", nil)
		return
	}

	// 解析请求参数
	var input struct {
		Parameters map[string]interface{} `json:"parameters"`
	}
	if err := c.ShouldBindJSON(&input); err != nil && err.Error() != "EOF" {
		response.Fail(c, "invalid payload", err.Error())
		return
	}

	// 使用触发器管理器执行工作流
	triggerManager := workflowdef.NewWorkflowTriggerManager(h.db)
	instance, execErr := triggerManager.TriggerWorkflow(
		def.ID,
		input.Parameters,
		fmt.Sprintf("api:public:%s", slug),
	)

	if execErr != nil {
		response.Fail(c, "workflow execution failed", execErr.Error())
		return
	}

	response.Success(c, "workflow executed successfully", instance)
}

// WebhookTriggerWorkflow Webhook触发工作流
func (h *Handlers) WebhookTriggerWorkflow(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		response.Fail(c, "invalid slug", nil)
		return
	}

	// 查找工作流定义
	var def models.WorkflowDefinition
	if err := h.db.Where("slug = ? AND status = ?", slug, "active").First(&def).Error; err != nil {
		response.Fail(c, "workflow not found or not active", nil)
		return
	}

	// 解析触发器配置
	config, err := workflowdef.ParseTriggerConfig(&def)
	if err != nil {
		response.Fail(c, "invalid trigger config", err.Error())
		return
	}

	// 检查 Webhook 触发是否启用
	if config.Webhook == nil || !config.Webhook.Enabled {
		response.Fail(c, "webhook trigger is not enabled for this workflow", nil)
		return
	}

	// 验证 Webhook Secret（如果配置了）
	if config.Webhook.Secret != "" {
		signature := c.GetHeader("X-Webhook-Signature")
		if signature == "" {
			response.Fail(c, "missing webhook signature", nil)
			return
		}
		// TODO: 实现签名验证逻辑
		// 这里可以添加 HMAC 验证等
	}

	// 解析请求体作为参数
	var parameters map[string]interface{}
	if err := c.ShouldBindJSON(&parameters); err != nil && err.Error() != "EOF" {
		// 如果 JSON 解析失败，尝试作为表单数据
		parameters = make(map[string]interface{})
		c.Request.ParseForm()
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				parameters[k] = v[0]
			}
		}
	}

	// 添加请求头信息（可选）
	if config.Webhook.Method == "" || config.Webhook.Method == "POST" {
		parameters["_webhook_headers"] = c.Request.Header
		parameters["_webhook_method"] = c.Request.Method
		parameters["_webhook_path"] = c.Request.URL.Path
	}

	// 使用触发器管理器执行工作流
	triggerManager := workflowdef.NewWorkflowTriggerManager(h.db)
	instance, execErr := triggerManager.TriggerWorkflow(
		def.ID,
		parameters,
		fmt.Sprintf("webhook:%s", slug),
	)

	if execErr != nil {
		response.Fail(c, "workflow execution failed", execErr.Error())
		return
	}

	response.Success(c, "workflow executed successfully", instance)
}

// GenerateAPIKey 生成随机API密钥
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
