package agent

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"go.uber.org/zap"
)

// Registry Agent注册表
type Registry struct {
	agents   map[string]Agent
	mu       sync.RWMutex
	logger   *zap.Logger
	statuses map[string]*AgentStatus
}

// NewRegistry 创建新的Agent注册表
func NewRegistry(logger *zap.Logger) *Registry {
	return &Registry{
		agents:   make(map[string]Agent),
		logger:   logger,
		statuses: make(map[string]*AgentStatus),
	}
}

// Register 注册agent
func (r *Registry) Register(agent Agent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agentID := agent.ID()
	if _, exists := r.agents[agentID]; exists {
		return fmt.Errorf("agent with ID %s already registered", agentID)
	}

	r.agents[agentID] = agent
	r.statuses[agentID] = &AgentStatus{
		ID:          agentID,
		Name:        agent.Name(),
		Status:      AgentStatusIdle,
		ActiveTasks: 0,
		TotalTasks:  0,
		Health:      HealthHealthy,
	}

	r.logger.Info("Agent registered",
		zap.String("id", agentID),
		zap.String("name", agent.Name()),
		zap.Int("capabilities", len(agent.Capabilities())),
	)

	return nil
}

// Unregister 注销agent
func (r *Registry) Unregister(agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[agentID]; !exists {
		return fmt.Errorf("agent with ID %s not found", agentID)
	}

	delete(r.agents, agentID)
	delete(r.statuses, agentID)

	r.logger.Info("Agent unregistered", zap.String("id", agentID))
	return nil
}

// Get 获取agent
func (r *Registry) Get(agentID string) (Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, exists := r.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent with ID %s not found", agentID)
	}

	return agent, nil
}

// List 列出所有agent
func (r *Registry) List() []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agents := make([]Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}

	return agents
}

// FindByCapability 根据能力查找agent
func (r *Registry) FindByCapability(capabilityType string) []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matchingAgents []Agent
	for _, agent := range r.agents {
		for _, cap := range agent.Capabilities() {
			if cap.Type == capabilityType {
				matchingAgents = append(matchingAgents, agent)
				break
			}
		}
	}

	return matchingAgents
}

// FindBestMatch 查找最适合处理任务的agent
func (r *Registry) FindBestMatch(request *TaskRequest) (Agent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var candidates []struct {
		agent    Agent
		priority int
	}

	// 首先根据任务类型匹配
	for _, agent := range r.agents {
		if agent.CanHandle(request) {
			// 计算优先级（可以根据agent状态、负载等）
			status := r.statuses[agent.ID()]
			priority := 0

			// 根据agent状态调整优先级
			if status.Status == AgentStatusIdle {
				priority += 10
			} else if status.Status == AgentStatusBusy {
				priority -= status.ActiveTasks
			}

			// 根据健康状态调整优先级
			if status.Health == HealthHealthy {
				priority += 5
			}

			candidates = append(candidates, struct {
				agent    Agent
				priority int
			}{agent: agent, priority: priority})
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no agent found that can handle task type: %s", request.Type)
	}

	// 按优先级排序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].priority > candidates[j].priority
	})

	return candidates[0].agent, nil
}

// UpdateStatus 更新agent状态
func (r *Registry) UpdateStatus(agentID string, status *AgentStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, exists := r.statuses[agentID]; exists {
		if status != nil {
			existing.Status = status.Status
			existing.ActiveTasks = status.ActiveTasks
			existing.TotalTasks = status.TotalTasks
			existing.LastActivity = status.LastActivity
			existing.Health = status.Health
		}
	}
}

// GetStatus 获取agent状态
func (r *Registry) GetStatus(agentID string) (*AgentStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status, exists := r.statuses[agentID]
	if !exists {
		return nil, fmt.Errorf("agent with ID %s not found", agentID)
	}

	return status, nil
}

// GetAllStatuses 获取所有agent状态
func (r *Registry) GetAllStatuses() map[string]*AgentStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	statuses := make(map[string]*AgentStatus)
	for id, status := range r.statuses {
		statuses[id] = status
	}

	return statuses
}

// HealthCheck 对所有agent进行健康检查
func (r *Registry) HealthCheck(ctx context.Context) map[string]error {
	r.mu.RLock()
	agents := make(map[string]Agent)
	for id, agent := range r.agents {
		agents[id] = agent
	}
	r.mu.RUnlock()

	results := make(map[string]error)
	for id, agent := range agents {
		if err := agent.Health(ctx); err != nil {
			results[id] = err
			r.UpdateStatus(id, &AgentStatus{
				Health: HealthUnhealthy,
			})
		} else {
			r.UpdateStatus(id, &AgentStatus{
				Health: HealthHealthy,
			})
		}
	}

	return results
}
