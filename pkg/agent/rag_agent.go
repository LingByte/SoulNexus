package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/knowledge"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RAGAgent RAG（检索增强生成）Agent
type RAGAgent struct {
	id          string
	name        string
	description string
	db          *gorm.DB
	kbManager   knowledge.Manager
	logger      *zap.Logger
}

// NewRAGAgent 创建新的RAG Agent
func NewRAGAgent(db *gorm.DB, kbManager knowledge.Manager, logger *zap.Logger) *RAGAgent {
	return &RAGAgent{
		id:          "rag_agent",
		name:        "RAG Agent",
		description: "检索增强生成Agent，负责从知识库中检索相关信息并生成回答",
		db:          db,
		kbManager:   kbManager,
		logger:      logger,
	}
}

// ID 返回agent ID
func (a *RAGAgent) ID() string {
	return a.id
}

// Name 返回agent名称
func (a *RAGAgent) Name() string {
	return a.name
}

// Description 返回agent描述
func (a *RAGAgent) Description() string {
	return a.description
}

// Capabilities 返回agent能力
func (a *RAGAgent) Capabilities() []Capability {
	return []Capability{
		{
			Name:        "knowledge_retrieval",
			Description: "从知识库中检索相关信息",
			Type:        "rag",
			Parameters: map[string]interface{}{
				"topK":      5,
				"threshold": 0.7,
			},
		},
		{
			Name:        "context_enhancement",
			Description: "使用检索到的信息增强上下文",
			Type:        "rag",
		},
	}
}

// CanHandle 判断是否能处理任务
func (a *RAGAgent) CanHandle(request *TaskRequest) bool {
	return request.Type == TaskTypeRAG ||
		(request.Context != nil && request.Context.KnowledgeBaseID != "")
}

// Process 处理任务
func (a *RAGAgent) Process(ctx context.Context, request *TaskRequest) (*TaskResponse, error) {
	startTime := time.Now()

	if request.Context == nil || request.Context.KnowledgeBaseID == "" {
		return &TaskResponse{
			ID:        request.ID,
			Success:   false,
			Error:     "KnowledgeBaseID is required for RAG agent",
			CreatedAt: time.Now(),
		}, nil
	}

	knowledgeKey := request.Context.KnowledgeBaseID
	query := request.Content

	// 获取TopK参数
	topK := 5
	if topKVal, ok := request.Parameters["topK"].(float64); ok {
		topK = int(topKVal)
	}

	// 检索知识库
	knowledgeResults, err := models.SearchKnowledgeBase(a.db, knowledgeKey, query, topK)
	if err != nil {
		a.logger.Error("Failed to search knowledge base",
			zap.String("knowledgeKey", knowledgeKey),
			zap.String("query", query),
			zap.Error(err),
		)
		return &TaskResponse{
			ID:        request.ID,
			Success:   false,
			Error:     fmt.Sprintf("Failed to search knowledge base: %v", err),
			CreatedAt: time.Now(),
		}, nil
	}

	// 构建增强的上下文
	var contextBuilder strings.Builder
	contextBuilder.WriteString(fmt.Sprintf("用户问题: %s\n\n", query))

	if len(knowledgeResults) > 0 {
		contextBuilder.WriteString("相关信息:\n")
		for i, result := range knowledgeResults {
			if i > 0 {
				contextBuilder.WriteString("\n\n")
			}
			contextBuilder.WriteString(fmt.Sprintf("[来源: %s, 相关性: %.2f]\n",
				result.Source, result.Score))
			contextBuilder.WriteString(result.Content)
		}
		contextBuilder.WriteString("\n\n请基于以上信息回答用户问题，回答要自然流畅，不要提及信息来源。")
	} else {
		contextBuilder.WriteString("未找到相关信息，请基于你的知识回答。")
	}

	enhancedContext := contextBuilder.String()

	a.logger.Info("RAG retrieval completed",
		zap.String("taskID", request.ID),
		zap.String("knowledgeKey", knowledgeKey),
		zap.Int("resultsCount", len(knowledgeResults)),
		zap.Duration("processingTime", time.Since(startTime)),
	)

	return &TaskResponse{
		ID:      request.ID,
		Success: true,
		Content: enhancedContext,
		Data: map[string]interface{}{
			"knowledgeResults": knowledgeResults,
			"resultsCount":     len(knowledgeResults),
		},
		AgentID:        a.id,
		ProcessingTime: time.Since(startTime),
		CreatedAt:      time.Now(),
	}, nil
}

// Health 健康检查
func (a *RAGAgent) Health(ctx context.Context) error {
	// 检查数据库连接
	if a.db != nil {
		sqlDB, err := a.db.DB()
		if err != nil {
			return fmt.Errorf("failed to get database connection: %w", err)
		}
		if err := sqlDB.PingContext(ctx); err != nil {
			return fmt.Errorf("database ping failed: %w", err)
		}
	}

	// 检查知识库管理器
	if a.kbManager == nil {
		return fmt.Errorf("knowledge base manager is nil")
	}

	return nil
}
