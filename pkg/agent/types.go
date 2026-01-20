package agent

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// Agent 定义agent接口
type Agent interface {
	// ID 返回agent的唯一标识符
	ID() string

	// Name 返回agent的名称
	Name() string

	// Description 返回agent的描述
	Description() string

	// Capabilities 返回agent的能力列表
	Capabilities() []Capability

	// Process 处理任务请求
	Process(ctx context.Context, request *TaskRequest) (*TaskResponse, error)

	// CanHandle 判断agent是否能处理特定任务
	CanHandle(request *TaskRequest) bool

	// Health 检查agent健康状态
	Health(ctx context.Context) error
}

// Capability 定义agent的能力
type Capability struct {
	// Name 能力名称
	Name string `json:"name"`

	// Description 能力描述
	Description string `json:"description"`

	// Type 能力类型（如：rag, graph_memory, tool, llm, etc.）
	Type string `json:"type"`

	// Parameters 能力参数定义
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// TaskRequest 任务请求
type TaskRequest struct {
	// ID 任务唯一标识
	ID string `json:"id"`

	// Type 任务类型
	Type string `json:"type"`

	// Content 任务内容
	Content string `json:"content"`

	// Context 任务上下文
	Context *TaskContext `json:"context"`

	// Parameters 任务参数
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// Metadata 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"createdAt"`
}

// TaskContext 任务上下文
type TaskContext struct {
	// UserID 用户ID
	UserID uint `json:"userId"`

	// AssistantID 助手ID
	AssistantID int64 `json:"assistantId"`

	// SessionID 会话ID
	SessionID string `json:"sessionId"`

	// ConversationHistory 对话历史
	ConversationHistory []Message `json:"conversationHistory,omitempty"`

	// KnowledgeBaseID 知识库ID
	KnowledgeBaseID string `json:"knowledgeBaseId,omitempty"`

	// GraphMemoryEnabled 是否启用图记忆
	GraphMemoryEnabled bool `json:"graphMemoryEnabled"`

	// AdditionalContext 额外上下文
	AdditionalContext map[string]interface{} `json:"additionalContext,omitempty"`
}

// TaskResponse 任务响应
type TaskResponse struct {
	// ID 响应ID（对应请求ID）
	ID string `json:"id"`

	// Success 是否成功
	Success bool `json:"success"`

	// Content 响应内容
	Content string `json:"content"`

	// Data 响应数据
	Data map[string]interface{} `json:"data,omitempty"`

	// Error 错误信息
	Error string `json:"error,omitempty"`

	// Metadata 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// AgentID 处理该任务的agent ID
	AgentID string `json:"agentId"`

	// ProcessingTime 处理时间
	ProcessingTime time.Duration `json:"processingTime"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"createdAt"`
}

// Message 消息定义
type Message struct {
	// ID 消息ID
	ID string `json:"id"`

	// Role 角色（user, assistant, system）
	Role string `json:"role"`

	// Content 消息内容
	Content string `json:"content"`

	// Metadata 元数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"createdAt"`
}

// AgentConfig Agent配置
type AgentConfig struct {
	// ID Agent ID
	ID string `json:"id"`

	// Name Agent名称
	Name string `json:"name"`

	// Description Agent描述
	Description string `json:"description"`

	// Type Agent类型
	Type string `json:"type"`

	// Enabled 是否启用
	Enabled bool `json:"enabled"`

	// Config Agent特定配置
	Config map[string]interface{} `json:"config,omitempty"`

	// Priority 优先级（数字越大优先级越高）
	Priority int `json:"priority"`

	// Timeout 超时时间（秒）
	Timeout int `json:"timeout"`
}

// WorkflowStep 工作流步骤
type WorkflowStep struct {
	// ID 步骤ID
	ID string `json:"id"`

	// AgentID 负责该步骤的agent ID
	AgentID string `json:"agentId"`

	// Type 步骤类型（sequential, parallel, conditional）
	Type string `json:"type"`

	// Condition 条件（用于conditional类型）
	Condition string `json:"condition,omitempty"`

	// Dependencies 依赖的步骤ID列表
	Dependencies []string `json:"dependencies,omitempty"`

	// Parameters 步骤参数
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// Workflow 工作流定义
type Workflow struct {
	// ID 工作流ID
	ID string `json:"id"`

	// Name 工作流名称
	Name string `json:"name"`

	// Description 工作流描述
	Description string `json:"description"`

	// Steps 工作流步骤
	Steps []WorkflowStep `json:"steps"`

	// CreatedAt 创建时间
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
}

// ToolCall 工具调用定义（与MCP集成）
type ToolCall struct {
	// Name 工具名称
	Name string `json:"name"`

	// Arguments 工具参数
	Arguments map[string]interface{} `json:"arguments"`

	// Result 工具调用结果
	Result *mcp.CallToolResult `json:"result,omitempty"`

	// Error 错误信息
	Error error `json:"error,omitempty"`
}

// AgentStatus Agent状态
type AgentStatus struct {
	// ID Agent ID
	ID string `json:"id"`

	// Name Agent名称
	Name string `json:"name"`

	// Status 状态（idle, busy, error）
	Status string `json:"status"`

	// ActiveTasks 当前活跃任务数
	ActiveTasks int `json:"activeTasks"`

	// TotalTasks 总任务数
	TotalTasks int64 `json:"totalTasks"`

	// LastActivity 最后活动时间
	LastActivity time.Time `json:"lastActivity"`

	// Health 健康状态
	Health string `json:"health"`
}

// 常量定义
const (
	// Agent状态
	AgentStatusIdle  = "idle"
	AgentStatusBusy  = "busy"
	AgentStatusError = "error"

	// Agent健康状态
	HealthHealthy   = "healthy"
	HealthDegraded  = "degraded"
	HealthUnhealthy = "unhealthy"

	// 任务类型
	TaskTypeRAG         = "rag"
	TaskTypeGraphMemory = "graph_memory"
	TaskTypeToolCall    = "tool_call"
	TaskTypeLLM         = "llm"
	TaskTypeWorkflow    = "workflow"
	TaskTypeGeneral     = "general"

	// 工作流步骤类型
	StepTypeSequential  = "sequential"
	StepTypeParallel    = "parallel"
	StepTypeConditional = "conditional"

	// 消息角色
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)
