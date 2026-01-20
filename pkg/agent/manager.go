package agent

import (
	"context"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/graph"
	"github.com/LingByte/SoulNexus/pkg/knowledge"
	"github.com/LingByte/SoulNexus/pkg/llm"
	lingechoMCP "github.com/LingByte/SoulNexus/pkg/mcp"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Manager Agent系统管理器
type Manager struct {
	registry     *Registry
	orchestrator *Orchestrator
	logger       *zap.Logger
	mu           sync.RWMutex
	initialized  bool
}

// Config Agent系统配置
type Config struct {
	DB          *gorm.DB
	GraphStore  graph.Store
	KBManager   knowledge.Manager
	LLMProvider llm.LLMProvider
	MCPServer   *lingechoMCP.MCPServer
	Logger      *zap.Logger
}

// NewManager 创建新的Agent管理器
func NewManager(cfg *Config) (*Manager, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	registry := NewRegistry(cfg.Logger)
	orchestrator := NewOrchestrator(registry, cfg.Logger)

	manager := &Manager{
		registry:     registry,
		orchestrator: orchestrator,
		logger:       cfg.Logger,
	}

	// 注册默认agents
	if err := manager.registerDefaultAgents(cfg); err != nil {
		return nil, err
	}

	manager.initialized = true
	manager.logger.Info("Agent manager initialized",
		zap.Int("agentsCount", len(registry.List())),
	)

	return manager, nil
}

// registerDefaultAgents 注册默认agents
func (m *Manager) registerDefaultAgents(cfg *Config) error {
	// 注册RAG Agent
	if cfg.DB != nil && cfg.KBManager != nil {
		ragAgent := NewRAGAgent(cfg.DB, cfg.KBManager, m.logger)
		if err := m.registry.Register(ragAgent); err != nil {
			return err
		}
		m.logger.Info("RAG agent registered")
	}

	// 注册Graph Memory Agent
	if cfg.GraphStore != nil {
		graphAgent := NewGraphMemoryAgent(cfg.GraphStore, m.logger)
		if err := m.registry.Register(graphAgent); err != nil {
			return err
		}
		m.logger.Info("Graph memory agent registered")
	}

	// 注册Tool Agent
	if cfg.MCPServer != nil {
		toolAgent := NewToolAgent(cfg.MCPServer, m.logger)
		if err := m.registry.Register(toolAgent); err != nil {
			return err
		}
		m.logger.Info("Tool agent registered")
	}

	// 注册LLM Agent
	if cfg.LLMProvider != nil {
		llmAgent := NewLLMAgent(cfg.LLMProvider, m.logger)
		if err := m.registry.Register(llmAgent); err != nil {
			return err
		}
		m.logger.Info("LLM agent registered")
	}

	return nil
}

// RegisterAgent 注册自定义agent
func (m *Manager) RegisterAgent(agent Agent) error {
	return m.registry.Register(agent)
}

// Process 处理任务请求
func (m *Manager) Process(ctx context.Context, request *TaskRequest) (*TaskResponse, error) {
	if !m.initialized {
		return nil, ErrNotInitialized
	}

	return m.orchestrator.Process(ctx, request)
}

// ProcessWorkflow 处理工作流
func (m *Manager) ProcessWorkflow(ctx context.Context, workflow *Workflow, request *TaskRequest) (*TaskResponse, error) {
	if !m.initialized {
		return nil, ErrNotInitialized
	}

	return m.orchestrator.ProcessWorkflow(ctx, workflow, request)
}

// GetAgent 获取agent
func (m *Manager) GetAgent(agentID string) (Agent, error) {
	return m.registry.Get(agentID)
}

// ListAgents 列出所有agents
func (m *Manager) ListAgents() []Agent {
	return m.registry.List()
}

// GetAgentStatus 获取agent状态
func (m *Manager) GetAgentStatus(agentID string) (*AgentStatus, error) {
	return m.registry.GetStatus(agentID)
}

// GetAllAgentStatuses 获取所有agent状态
func (m *Manager) GetAllAgentStatuses() map[string]*AgentStatus {
	return m.registry.GetAllStatuses()
}

// HealthCheck 健康检查
func (m *Manager) HealthCheck(ctx context.Context) map[string]error {
	return m.registry.HealthCheck(ctx)
}

// GetOrchestrator 获取协调器（用于高级操作）
func (m *Manager) GetOrchestrator() *Orchestrator {
	return m.orchestrator
}

// GetRegistry 获取注册表（用于高级操作）
func (m *Manager) GetRegistry() *Registry {
	return m.registry
}

var (
	ErrNotInitialized = &AgentError{
		Code:    "NOT_INITIALIZED",
		Message: "Agent manager is not initialized",
	}
)

// AgentError Agent错误
type AgentError struct {
	Code    string
	Message string
}

func (e *AgentError) Error() string {
	return e.Message
}
