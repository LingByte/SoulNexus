package workflowdef

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	runtimewf "github.com/LingByte/SoulNexus/pkg/workflow"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// WorkflowTriggerConfig 工作流触发器配置
type WorkflowTriggerConfig struct {
	// API触发配置
	API *APITriggerConfig `json:"api,omitempty"`

	// 事件触发配置
	Event *EventTriggerConfig `json:"event,omitempty"`

	// 定时触发配置
	Schedule *ScheduleTriggerConfig `json:"schedule,omitempty"`

	// Webhook触发配置
	Webhook *WebhookTriggerConfig `json:"webhook,omitempty"`

	// 智能体触发配置
	Assistant *AssistantTriggerConfig `json:"assistant,omitempty"`
}

// APITriggerConfig API触发配置
type APITriggerConfig struct {
	Enabled     bool   `json:"enabled"`          // 是否启用
	Public      bool   `json:"public"`           // 是否公开（不需要认证）
	APIKey      string `json:"apiKey,omitempty"` // API密钥（用于公开API）
	Description string `json:"description,omitempty"`
}

// EventTriggerConfig 事件触发配置
type EventTriggerConfig struct {
	Enabled   bool     `json:"enabled"`             // 是否启用
	Events    []string `json:"events"`              // 监听的事件类型列表，如 ["user.created", "order.paid"]
	Condition string   `json:"condition,omitempty"` // 可选的条件表达式
}

// ScheduleTriggerConfig 定时触发配置
type ScheduleTriggerConfig struct {
	Enabled  bool   `json:"enabled"`            // 是否启用
	CronExpr string `json:"cronExpr"`           // Cron表达式，如 "0 0 * * *" (每天0点)
	Timezone string `json:"timezone,omitempty"` // 时区，默认为系统时区
}

// WebhookTriggerConfig Webhook触发配置
type WebhookTriggerConfig struct {
	Enabled bool   `json:"enabled"`          // 是否启用
	URL     string `json:"url,omitempty"`    // Webhook URL（如果为空，使用系统生成的URL）
	Secret  string `json:"secret,omitempty"` // Webhook密钥（用于验证）
	Method  string `json:"method,omitempty"` // HTTP方法，默认POST
}

// AssistantTriggerConfig 智能体触发配置
type AssistantTriggerConfig struct {
	Enabled      bool     `json:"enabled"`                // 是否启用
	AssistantIDs []int64  `json:"assistantIds,omitempty"` // 关联的智能体ID列表（为空表示所有智能体都可以调用）
	Intents      []string `json:"intents,omitempty"`      // 触发意图列表（为空表示智能体可以自由调用）
	Description  string   `json:"description,omitempty"`  // 智能体调用时的描述
}

// ParseTriggerConfig 解析触发器配置
func ParseTriggerConfig(def *models.WorkflowDefinition) (*WorkflowTriggerConfig, error) {
	if def.Triggers == nil || len(def.Triggers) == 0 {
		return &WorkflowTriggerConfig{}, nil
	}

	// 将 JSONMap 转换为 JSON 再解析
	jsonData, err := json.Marshal(def.Triggers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal triggers: %w", err)
	}

	var config WorkflowTriggerConfig
	if err := json.Unmarshal(jsonData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse trigger config: %w", err)
	}

	return &config, nil
}

// HasAnyTrigger 检查是否有任何启用的触发器
func (c *WorkflowTriggerConfig) HasAnyTrigger() bool {
	return (c.API != nil && c.API.Enabled) ||
		(c.Event != nil && c.Event.Enabled) ||
		(c.Schedule != nil && c.Schedule.Enabled) ||
		(c.Webhook != nil && c.Webhook.Enabled) ||
		(c.Assistant != nil && c.Assistant.Enabled)
}

// GetPublicAPIKey 获取公开API密钥（如果启用）
func (c *WorkflowTriggerConfig) GetPublicAPIKey() string {
	if c.API != nil && c.API.Enabled && c.API.Public {
		return c.API.APIKey
	}
	return ""
}

// ShouldTriggerOnEvent 检查是否应该在指定事件时触发
func (c *WorkflowTriggerConfig) ShouldTriggerOnEvent(eventType string) bool {
	if c.Event == nil || !c.Event.Enabled {
		return false
	}

	// 检查事件是否在监听列表中
	for _, e := range c.Event.Events {
		if e == eventType || e == "*" {
			return true
		}
	}

	return false
}

// CanBeCalledByAssistant 检查是否可以被指定智能体调用
func (c *WorkflowTriggerConfig) CanBeCalledByAssistant(assistantID int64) bool {
	if c.Assistant == nil || !c.Assistant.Enabled {
		return false
	}

	// 如果没有指定智能体列表，所有智能体都可以调用
	if len(c.Assistant.AssistantIDs) == 0 {
		return true
	}

	// 检查智能体是否在允许列表中
	for _, id := range c.Assistant.AssistantIDs {
		if id == assistantID {
			return true
		}
	}

	return false
}

// WorkflowTriggerManager 工作流触发器管理器
type WorkflowTriggerManager struct {
	db *gorm.DB
}

// NewWorkflowTriggerManager 创建触发器管理器
func NewWorkflowTriggerManager(db *gorm.DB) *WorkflowTriggerManager {
	return &WorkflowTriggerManager{
		db: db,
	}
}

// TriggerWorkflow 触发工作流执行
func (m *WorkflowTriggerManager) TriggerWorkflow(definitionID uint, parameters map[string]interface{}, triggerSource string) (*models.WorkflowInstance, error) {
	var def models.WorkflowDefinition
	if err := m.db.First(&def, definitionID).Error; err != nil {
		return nil, fmt.Errorf("workflow definition not found: %w", err)
	}

	// 检查工作流状态
	if def.Status != "active" {
		return nil, fmt.Errorf("workflow is not active (current status: %s)", def.Status)
	}

	// 构建并执行工作流
	runtimeWf, err := BuildRuntimeWorkflow(&def)
	if err != nil {
		return nil, fmt.Errorf("failed to build workflow: %w", err)
	}

	// 设置参数
	if runtimeWf.Context == nil {
		runtimeWf.Context = runtimewf.NewWorkflowContext(fmt.Sprintf("trigger-%d", definitionID))
	}
	if runtimeWf.Context.Parameters == nil {
		runtimeWf.Context.Parameters = make(map[string]interface{})
	}

	// 添加触发源信息
	runtimeWf.Context.Parameters["_trigger_source"] = triggerSource
	runtimeWf.Context.Parameters["_trigger_time"] = time.Now().Format(time.RFC3339)

	// 添加用户提供的参数
	for k, v := range parameters {
		runtimeWf.Context.Parameters[k] = v
	}

	// 创建实例记录
	now := time.Now()
	instance := models.WorkflowInstance{
		DefinitionID:   def.ID,
		DefinitionName: def.Name,
		Status:         "running",
		StartedAt:      &now,
		ContextData:    make(models.JSONMap),
		ResultData:     make(models.JSONMap),
	}

	if err := m.db.Create(&instance).Error; err != nil {
		return nil, fmt.Errorf("failed to create workflow instance: %w", err)
	}

	// 执行工作流
	execErr := runtimeWf.Execute()

	// 更新实例状态
	completedAt := time.Now()
	instance.CompletedAt = &completedAt

	if execErr != nil {
		instance.Status = "failed"
		instance.ResultData = models.JSONMap{
			"error": execErr.Error(),
		}
		logger.Error("Workflow execution failed",
			zap.Uint("definitionId", definitionID),
			zap.String("triggerSource", triggerSource),
			zap.Error(execErr))
	} else {
		instance.Status = "completed"
		if runtimeWf.Context != nil {
			instance.ContextData = runtimeWf.Context.NodeData
			instance.ResultData = models.JSONMap{
				"success": true,
				"context": runtimeWf.Context.NodeData,
			}
		}
		logger.Info("Workflow executed successfully",
			zap.Uint("definitionId", definitionID),
			zap.String("triggerSource", triggerSource))
	}

	if err := m.db.Save(&instance).Error; err != nil {
		return nil, fmt.Errorf("failed to update workflow instance: %w", err)
	}

	return &instance, execErr
}

// GetActiveWorkflowsByEvent 根据事件类型获取需要触发的工作流
func (m *WorkflowTriggerManager) GetActiveWorkflowsByEvent(eventType string) ([]models.WorkflowDefinition, error) {
	var workflows []models.WorkflowDefinition

	// 获取所有激活状态的工作流
	if err := m.db.Where("status = ?", "active").Find(&workflows).Error; err != nil {
		return nil, err
	}

	var result []models.WorkflowDefinition
	for _, wf := range workflows {
		config, err := ParseTriggerConfig(&wf)
		if err != nil {
			logger.Warn("Failed to parse trigger config",
				zap.Uint("workflowId", wf.ID),
				zap.Error(err))
			continue
		}

		if config.ShouldTriggerOnEvent(eventType) {
			result = append(result, wf)
		}
	}

	return result, nil
}

// GetScheduledWorkflows 获取所有需要定时执行的工作流
func (m *WorkflowTriggerManager) GetScheduledWorkflows() ([]models.WorkflowDefinition, error) {
	var workflows []models.WorkflowDefinition

	if err := m.db.Where("status = ?", "active").Find(&workflows).Error; err != nil {
		return nil, err
	}

	var result []models.WorkflowDefinition
	for _, wf := range workflows {
		config, err := ParseTriggerConfig(&wf)
		if err != nil {
			continue
		}

		if config.Schedule != nil && config.Schedule.Enabled && config.Schedule.CronExpr != "" {
			result = append(result, wf)
		}
	}

	return result, nil
}
