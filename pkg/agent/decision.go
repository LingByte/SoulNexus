package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// DecisionEngine 决策引擎，负责智能选择Agent
type DecisionEngine struct {
	registry   *Registry
	logger     *zap.Logger
	strategies []DecisionStrategy
	mu         sync.RWMutex
	history    *ExecutionHistory
	config     *DecisionConfig
}

// DecisionConfig 决策配置
type DecisionConfig struct {
	// Strategy 决策策略（auto, single, multi, chain）
	Strategy string

	// EnableHistory 是否启用历史记录
	EnableHistory bool

	// MaxHistorySize 最大历史记录数
	MaxHistorySize int

	// ScoreWeights 评分权重
	ScoreWeights *ScoreWeights
}

// ScoreWeights 评分权重
type ScoreWeights struct {
	CapabilityMatch float64 // 能力匹配权重
	Performance     float64 // 性能权重
	Availability    float64 // 可用性权重
	Cost            float64 // 成本权重
}

// DecisionStrategy 决策策略接口
type DecisionStrategy interface {
	// Name 策略名称
	Name() string

	// Decide 决策：选择要使用的Agents
	Decide(ctx context.Context, request *TaskRequest, candidates []Agent) ([]Agent, error)

	// Score 为Agent评分
	Score(ctx context.Context, agent Agent, request *TaskRequest) (float64, error)
}

// ExecutionHistory 执行历史
type ExecutionHistory struct {
	records map[string][]ExecutionRecord
	mu      sync.RWMutex
	maxSize int
}

// ExecutionRecord 执行记录
type ExecutionRecord struct {
	TaskID    string
	AgentID   string
	TaskType  string
	Success   bool
	Duration  time.Duration
	Timestamp time.Time
	Error     string
}

// NewDecisionEngine 创建新的决策引擎
func NewDecisionEngine(registry *Registry, logger *zap.Logger, config *DecisionConfig) *DecisionEngine {
	if config == nil {
		config = &DecisionConfig{
			Strategy:       "auto",
			EnableHistory:  true,
			MaxHistorySize: 1000,
			ScoreWeights: &ScoreWeights{
				CapabilityMatch: 0.4,
				Performance:     0.3,
				Availability:    0.2,
				Cost:            0.1,
			},
		}
	}

	engine := &DecisionEngine{
		registry:   registry,
		logger:     logger,
		strategies: make([]DecisionStrategy, 0),
		history: &ExecutionHistory{
			records: make(map[string][]ExecutionRecord),
			maxSize: config.MaxHistorySize,
		},
		config: config,
	}

	// 注册默认策略
	engine.registerDefaultStrategies()

	return engine
}

// registerDefaultStrategies 注册默认策略
func (e *DecisionEngine) registerDefaultStrategies() {
	// 1. 单Agent策略（选择最佳单个Agent）
	e.strategies = append(e.strategies, &SingleAgentStrategy{
		engine: e,
	})

	// 2. 多Agent策略（选择多个Agent并行执行）
	e.strategies = append(e.strategies, &MultiAgentStrategy{
		engine: e,
	})

	// 3. 链式策略（选择Agent链顺序执行）
	e.strategies = append(e.strategies, &ChainStrategy{
		engine: e,
	})

	// 4. 智能策略（根据任务自动选择）
	e.strategies = append(e.strategies, &IntelligentStrategy{
		engine: e,
	})
}

// Decide 决策：选择要使用的Agents
func (e *DecisionEngine) Decide(ctx context.Context, request *TaskRequest) ([]Agent, error) {
	// 1. 获取候选Agents
	candidates := e.getCandidates(request)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no agents available for task type: %s", request.Type)
	}

	e.logger.Debug("Decision candidates",
		zap.String("taskID", request.ID),
		zap.String("taskType", request.Type),
		zap.Int("candidatesCount", len(candidates)),
	)

	// 2. 选择策略
	strategy := e.selectStrategy(request)
	if strategy == nil {
		return nil, fmt.Errorf("no suitable strategy found")
	}

	e.logger.Debug("Using strategy",
		zap.String("strategy", strategy.Name()),
		zap.String("taskID", request.ID),
	)

	// 3. 执行决策
	selectedAgents, err := strategy.Decide(ctx, request, candidates)
	if err != nil {
		return nil, fmt.Errorf("decision failed: %w", err)
	}

	e.logger.Info("Agents selected",
		zap.String("taskID", request.ID),
		zap.String("strategy", strategy.Name()),
		zap.Int("agentsCount", len(selectedAgents)),
		zap.Strings("agentIDs", getAgentIDs(selectedAgents)),
	)

	return selectedAgents, nil
}

// getCandidates 获取候选Agents
func (e *DecisionEngine) getCandidates(request *TaskRequest) []Agent {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var candidates []Agent

	// 根据任务类型查找
	if request.Type != "" {
		candidates = e.registry.FindByCapability(request.Type)
	}

	// 如果没找到，尝试所有能处理的Agent
	if len(candidates) == 0 {
		allAgents := e.registry.List()
		for _, agent := range allAgents {
			if agent.CanHandle(request) {
				candidates = append(candidates, agent)
			}
		}
	}

	// 过滤掉不健康的Agent
	healthyCandidates := make([]Agent, 0)
	for _, agent := range candidates {
		status, err := e.registry.GetStatus(agent.ID())
		if err == nil && status.Health == HealthHealthy {
			healthyCandidates = append(healthyCandidates, agent)
		}
	}

	return healthyCandidates
}

// selectStrategy 选择决策策略
func (e *DecisionEngine) selectStrategy(request *TaskRequest) DecisionStrategy {
	// 如果请求中指定了策略，使用指定的
	if strategyName, ok := request.Parameters["strategy"].(string); ok {
		for _, strategy := range e.strategies {
			if strategy.Name() == strategyName {
				return strategy
			}
		}
	}

	// 根据配置选择策略
	switch e.config.Strategy {
	case "single":
		return e.strategies[0] // SingleAgentStrategy
	case "multi":
		return e.strategies[1] // MultiAgentStrategy
	case "chain":
		return e.strategies[2] // ChainStrategy
	case "auto", "intelligent":
		return e.strategies[3] // IntelligentStrategy
	default:
		return e.strategies[3] // 默认使用智能策略
	}
}

// Score 为Agent评分
func (e *DecisionEngine) Score(ctx context.Context, agent Agent, request *TaskRequest) (float64, error) {
	score := 0.0
	weights := e.config.ScoreWeights

	// 1. 能力匹配评分
	capabilityScore := e.scoreCapabilityMatch(agent, request)
	score += capabilityScore * weights.CapabilityMatch

	// 2. 性能评分（基于历史）
	performanceScore := e.scorePerformance(agent, request)
	score += performanceScore * weights.Performance

	// 3. 可用性评分
	availabilityScore := e.scoreAvailability(agent)
	score += availabilityScore * weights.Availability

	// 4. 成本评分（简化实现）
	costScore := e.scoreCost(agent)
	score += costScore * weights.Cost

	return score, nil
}

// scoreCapabilityMatch 能力匹配评分
func (e *DecisionEngine) scoreCapabilityMatch(agent Agent, request *TaskRequest) float64 {
	score := 0.0
	capabilities := agent.Capabilities()

	// 检查是否有匹配的能力类型
	for _, cap := range capabilities {
		if cap.Type == request.Type {
			score += 0.5
		}
	}

	// 检查CanHandle
	if agent.CanHandle(request) {
		score += 0.5
	}

	return score
}

// scorePerformance 性能评分（基于历史记录）
func (e *DecisionEngine) scorePerformance(agent Agent, request *TaskRequest) float64 {
	if !e.config.EnableHistory {
		return 0.5 // 默认中等性能
	}

	records := e.history.GetAgentRecords(agent.ID(), request.Type)
	if len(records) == 0 {
		return 0.5 // 无历史记录，返回中等评分
	}

	// 计算成功率
	successCount := 0
	totalDuration := time.Duration(0)
	for _, record := range records {
		if record.Success {
			successCount++
		}
		totalDuration += record.Duration
	}

	successRate := float64(successCount) / float64(len(records))
	avgDuration := totalDuration / time.Duration(len(records))

	// 成功率权重0.6，速度权重0.4
	// 速度越快（duration越小），评分越高
	// 假设理想平均时长为1秒
	idealDuration := 1 * time.Second
	speedScore := 1.0
	if avgDuration > 0 {
		if avgDuration < idealDuration {
			speedScore = 1.0
		} else {
			speedScore = idealDuration.Seconds() / avgDuration.Seconds()
			if speedScore < 0 {
				speedScore = 0
			}
		}
	}

	return successRate*0.6 + speedScore*0.4
}

// scoreAvailability 可用性评分
func (e *DecisionEngine) scoreAvailability(agent Agent) float64 {
	status, err := e.registry.GetStatus(agent.ID())
	if err != nil {
		return 0.0
	}

	// 根据状态和负载评分
	score := 1.0

	if status.Status == AgentStatusBusy {
		// 根据活跃任务数降低评分
		loadPenalty := float64(status.ActiveTasks) * 0.1
		score -= loadPenalty
		if score < 0 {
			score = 0
		}
	}

	if status.Health != HealthHealthy {
		score *= 0.5
	}

	return score
}

// scoreCost 成本评分（简化实现）
func (e *DecisionEngine) scoreCost(agent Agent) float64 {
	// 简化实现：不同类型的Agent有不同的成本
	// 实际应该从配置或Agent元数据中获取
	switch agent.ID() {
	case "llm_agent":
		return 0.3 // LLM成本较高
	case "rag_agent", "graph_memory_agent":
		return 0.7 // 中等成本
	case "tool_agent":
		return 0.9 // 工具调用成本较低
	default:
		return 0.5 // 默认中等成本
	}
}

// RecordExecution 记录执行历史
func (e *DecisionEngine) RecordExecution(record ExecutionRecord) {
	if !e.config.EnableHistory {
		return
	}

	e.history.AddRecord(record)
}

// GetAgentIDs 获取Agent ID列表
func getAgentIDs(agents []Agent) []string {
	ids := make([]string, len(agents))
	for i, agent := range agents {
		ids[i] = agent.ID()
	}
	return ids
}

// AddStrategy 添加自定义策略
func (e *DecisionEngine) AddStrategy(strategy DecisionStrategy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.strategies = append(e.strategies, strategy)
}
