package agent

import (
	"context"
	"fmt"
	"sort"
)

// SingleAgentStrategy 单Agent策略：选择评分最高的单个Agent
type SingleAgentStrategy struct {
	engine *DecisionEngine
}

func (s *SingleAgentStrategy) Name() string {
	return "single"
}

func (s *SingleAgentStrategy) Decide(ctx context.Context, request *TaskRequest, candidates []Agent) ([]Agent, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates available")
	}

	// 为所有候选Agent评分
	scores := make([]struct {
		agent Agent
		score float64
	}, 0, len(candidates))

	for _, agent := range candidates {
		score, err := s.engine.Score(ctx, agent, request)
		if err != nil {
			continue
		}
		scores = append(scores, struct {
			agent Agent
			score float64
		}{agent: agent, score: score})
	}

	if len(scores) == 0 {
		return nil, fmt.Errorf("no agents scored successfully")
	}

	// 按评分排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 返回评分最高的Agent
	return []Agent{scores[0].agent}, nil
}

func (s *SingleAgentStrategy) Score(ctx context.Context, agent Agent, request *TaskRequest) (float64, error) {
	return s.engine.Score(ctx, agent, request)
}

// MultiAgentStrategy 多Agent策略：选择多个Agent并行执行
type MultiAgentStrategy struct {
	engine *DecisionEngine
}

func (m *MultiAgentStrategy) Name() string {
	return "multi"
}

func (m *MultiAgentStrategy) Decide(ctx context.Context, request *TaskRequest, candidates []Agent) ([]Agent, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates available")
	}

	// 为所有候选Agent评分
	scores := make([]struct {
		agent Agent
		score float64
	}, 0, len(candidates))

	for _, agent := range candidates {
		score, err := m.engine.Score(ctx, agent, request)
		if err != nil {
			continue
		}
		scores = append(scores, struct {
			agent Agent
			score float64
		}{agent: agent, score: score})
	}

	if len(scores) == 0 {
		return nil, fmt.Errorf("no agents scored successfully")
	}

	// 按评分排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 选择评分高于阈值的Agent（默认0.5）
	threshold := 0.5
	if thresholdVal, ok := request.Parameters["threshold"].(float64); ok {
		threshold = thresholdVal
	}

	selected := make([]Agent, 0)
	for _, item := range scores {
		if item.score >= threshold {
			selected = append(selected, item.agent)
		}
	}

	// 如果没选到，至少选择评分最高的一个
	if len(selected) == 0 {
		selected = []Agent{scores[0].agent}
	}

	// 限制最大数量（默认3个）
	maxAgents := 3
	if maxVal, ok := request.Parameters["maxAgents"].(float64); ok {
		maxAgents = int(maxVal)
	}
	if len(selected) > maxAgents {
		selected = selected[:maxAgents]
	}

	return selected, nil
}

func (m *MultiAgentStrategy) Score(ctx context.Context, agent Agent, request *TaskRequest) (float64, error) {
	return m.engine.Score(ctx, agent, request)
}

// ChainStrategy 链式策略：选择Agent链顺序执行
type ChainStrategy struct {
	engine *DecisionEngine
}

func (c *ChainStrategy) Name() string {
	return "chain"
}

func (c *ChainStrategy) Decide(ctx context.Context, request *TaskRequest, candidates []Agent) ([]Agent, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates available")
	}

	// 根据任务类型和上下文决定Agent链
	chain := make([]Agent, 0)

	// 1. 如果有图记忆需求，先添加Graph Memory Agent
	if request.Context != nil && request.Context.GraphMemoryEnabled {
		for _, agent := range candidates {
			if agent.ID() == "graph_memory_agent" {
				chain = append(chain, agent)
				break
			}
		}
	}

	// 2. 如果有知识库，添加RAG Agent
	if request.Context != nil && request.Context.KnowledgeBaseID != "" {
		for _, agent := range candidates {
			if agent.ID() == "rag_agent" {
				chain = append(chain, agent)
				break
			}
		}
	}

	// 3. 添加LLM Agent（通常作为最后一步）
	for _, agent := range candidates {
		if agent.ID() == "llm_agent" {
			chain = append(chain, agent)
			break
		}
	}

	// 如果链为空，使用评分选择
	if len(chain) == 0 {
		// 使用单Agent策略
		singleStrategy := &SingleAgentStrategy{engine: c.engine}
		selected, err := singleStrategy.Decide(ctx, request, candidates)
		if err != nil {
			return nil, err
		}
		chain = selected
	}

	return chain, nil
}

func (c *ChainStrategy) Score(ctx context.Context, agent Agent, request *TaskRequest) (float64, error) {
	return c.engine.Score(ctx, agent, request)
}

// IntelligentStrategy 智能策略：根据任务自动选择最佳策略
type IntelligentStrategy struct {
	engine *DecisionEngine
}

func (i *IntelligentStrategy) Name() string {
	return "intelligent"
}

func (i *IntelligentStrategy) Decide(ctx context.Context, request *TaskRequest, candidates []Agent) ([]Agent, error) {
	// 根据任务特征智能选择策略

	// 1. 如果任务需要多个步骤，使用链式策略
	if i.needsChain(request) {
		chainStrategy := &ChainStrategy{engine: i.engine}
		return chainStrategy.Decide(ctx, request, candidates)
	}

	// 2. 如果任务需要并行处理，使用多Agent策略
	if i.needsParallel(request) {
		multiStrategy := &MultiAgentStrategy{engine: i.engine}
		return multiStrategy.Decide(ctx, request, candidates)
	}

	// 3. 默认使用单Agent策略
	singleStrategy := &SingleAgentStrategy{engine: i.engine}
	return singleStrategy.Decide(ctx, request, candidates)
}

func (i *IntelligentStrategy) needsChain(request *TaskRequest) bool {
	// 判断是否需要链式执行
	// 例如：需要图记忆 + RAG + LLM
	if request.Context != nil {
		needsGraph := request.Context.GraphMemoryEnabled
		needsRAG := request.Context.KnowledgeBaseID != ""
		needsLLM := request.Type == TaskTypeLLM || request.Type == TaskTypeGeneral

		// 如果同时需要多个能力，使用链式
		count := 0
		if needsGraph {
			count++
		}
		if needsRAG {
			count++
		}
		if needsLLM {
			count++
		}

		return count >= 2
	}

	return false
}

func (i *IntelligentStrategy) needsParallel(request *TaskRequest) bool {
	// 判断是否需要并行执行
	// 例如：需要同时检索多个知识库
	if request.Parameters != nil {
		if parallel, ok := request.Parameters["parallel"].(bool); ok && parallel {
			return true
		}
	}

	return false
}

func (i *IntelligentStrategy) Score(ctx context.Context, agent Agent, request *TaskRequest) (float64, error) {
	return i.engine.Score(ctx, agent, request)
}
