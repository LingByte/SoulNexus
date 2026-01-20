package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/pkg/graph"
	"go.uber.org/zap"
)

// GraphMemoryAgent 图记忆Agent，负责从图数据库中检索长期记忆
type GraphMemoryAgent struct {
	id          string
	name        string
	description string
	graphStore  graph.Store
	logger      *zap.Logger
}

// NewGraphMemoryAgent 创建新的图记忆Agent
func NewGraphMemoryAgent(graphStore graph.Store, logger *zap.Logger) *GraphMemoryAgent {
	return &GraphMemoryAgent{
		id:          "graph_memory_agent",
		name:        "Graph Memory Agent",
		description: "图记忆Agent，负责从图数据库中检索用户的长期记忆和上下文",
		graphStore:  graphStore,
		logger:      logger,
	}
}

// ID 返回agent ID
func (a *GraphMemoryAgent) ID() string {
	return a.id
}

// Name 返回agent名称
func (a *GraphMemoryAgent) Name() string {
	return a.name
}

// Description 返回agent描述
func (a *GraphMemoryAgent) Description() string {
	return a.description
}

// Capabilities 返回agent能力
func (a *GraphMemoryAgent) Capabilities() []Capability {
	return []Capability{
		{
			Name:        "user_context_retrieval",
			Description: "检索用户上下文（偏好、历史主题等）",
			Type:        "graph_memory",
		},
		{
			Name:        "conversation_history",
			Description: "检索对话历史和相关主题",
			Type:        "graph_memory",
		},
		{
			Name:        "knowledge_extraction",
			Description: "从对话中提取知识并存储到图数据库",
			Type:        "graph_memory",
		},
	}
}

// CanHandle 判断是否能处理任务
func (a *GraphMemoryAgent) CanHandle(request *TaskRequest) bool {
	return request.Type == TaskTypeGraphMemory ||
		(request.Context != nil && request.Context.GraphMemoryEnabled)
}

// Process 处理任务
func (a *GraphMemoryAgent) Process(ctx context.Context, request *TaskRequest) (*TaskResponse, error) {
	startTime := time.Now()

	if a.graphStore == nil {
		return &TaskResponse{
			ID:        request.ID,
			Success:   false,
			Error:     "Graph store is not available",
			CreatedAt: time.Now(),
		}, nil
	}

	if request.Context == nil {
		return &TaskResponse{
			ID:        request.ID,
			Success:   false,
			Error:     "Context is required for graph memory agent",
			CreatedAt: time.Now(),
		}, nil
	}

	// 获取用户上下文
	userContext, err := a.graphStore.GetUserContext(
		ctx,
		request.Context.UserID,
		request.Context.AssistantID,
	)
	if err != nil {
		a.logger.Warn("Failed to get user context",
			zap.Uint("userID", request.Context.UserID),
			zap.Int64("assistantID", request.Context.AssistantID),
			zap.Error(err),
		)
		// 不返回错误，只是没有上下文
		userContext = &graph.UserContext{
			UserID:      request.Context.UserID,
			AssistantID: request.Context.AssistantID,
			Topics:      []string{},
		}
	}

	// 构建上下文信息
	contextInfo := fmt.Sprintf("用户偏好主题: %v", userContext.Topics)
	if len(userContext.Topics) == 0 {
		contextInfo = "暂无用户偏好信息"
	}

	a.logger.Info("Graph memory retrieval completed",
		zap.String("taskID", request.ID),
		zap.Uint("userID", request.Context.UserID),
		zap.Int("topicsCount", len(userContext.Topics)),
		zap.Duration("processingTime", time.Since(startTime)),
	)

	return &TaskResponse{
		ID:      request.ID,
		Success: true,
		Content: contextInfo,
		Data: map[string]interface{}{
			"userContext": userContext,
			"topics":      userContext.Topics,
		},
		AgentID:        a.id,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
	}, nil
}

// Health 健康检查
func (a *GraphMemoryAgent) Health(ctx context.Context) error {
	if a.graphStore == nil {
		return fmt.Errorf("graph store is nil")
	}

	// 尝试获取一个测试上下文（如果可能）
	// 这里简化处理，只检查store是否存在
	return nil
}
