package llm

import (
	"context"
	"fmt"
	"sync"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/hardware/errhandler"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"go.uber.org/zap"
)

// Service LLM服务实现
type Service struct {
	ctx          context.Context
	credential   *models.UserCredential
	systemPrompt string
	model        string
	temperature  float64
	maxTokens    int
	provider     llm.LLMProvider
	errorHandler *errhandler.Handler
	logger       *zap.Logger
	mu           sync.RWMutex
	closed       bool
}

// NewService 创建LLM服务
func NewService(
	ctx context.Context,
	credential *models.UserCredential,
	systemPrompt string,
	model string,
	temperature float64,
	maxTokens int,
	provider llm.LLMProvider,
	errorHandler *errhandler.Handler,
	logger *zap.Logger,
) *Service {
	return &Service{
		ctx:          ctx,
		credential:   credential,
		systemPrompt: systemPrompt,
		model:        model,
		temperature:  temperature,
		maxTokens:    maxTokens,
		provider:     provider,
		errorHandler: errorHandler,
		logger:       logger,
	}
}

// Query 查询（使用最后一条消息）
func (s *Service) Query(ctx context.Context, text string) (string, error) {
	s.mu.RLock()
	closed := s.closed
	provider := s.provider
	s.mu.RUnlock()

	if closed || provider == nil {
		return "", errhandler.NewRecoverableError("LLM", "服务已关闭", nil)
	}

	if text == "" {
		return "", errhandler.NewRecoverableError("LLM", "消息为空", nil)
	}

	// 设置系统提示（追加最大token限制提示）
	enhancedSystemPrompt := s.buildEnhancedSystemPrompt()
	if enhancedSystemPrompt != "" {
		provider.SetSystemPrompt(enhancedSystemPrompt)
	}

	// 构建查询选项
	options := llm.QueryOptions{
		Model:       s.model,
		MaxTokens:   intPtr(s.maxTokens),
		Temperature: float32Ptr(s.temperature),
		Stream:      false,
	}
	if s.maxTokens > 0 {
		options.MaxTokens = intPtr(s.maxTokens)
	}

	// 调用LLM
	response, err := provider.QueryWithOptions(text, options)
	if err != nil {
		classified := s.errorHandler.Classify(err, "LLM")
		s.logger.Error("LLM查询失败", zap.Error(classified))
		return "", classified
	}

	return response, nil
}

// buildEnhancedSystemPrompt 构建增强的系统提示词（包含最大token限制）
func (s *Service) buildEnhancedSystemPrompt() string {
	basePrompt := s.systemPrompt

	// 如果设置了最大token，追加限制提示
	if s.maxTokens > 0 {
		tokenLimitPrompt := fmt.Sprintf("\n\n重要提示：你的回复必须控制在%d个token以内。请确保回复简洁、完整，避免因超出限制而被截断。如果内容较长，请优先表达核心要点，保持回复的完整性和可理解性。", s.maxTokens)
		if basePrompt != "" {
			return basePrompt + tokenLimitPrompt
		}
		return tokenLimitPrompt
	}

	return basePrompt
}

// float32Ptr 返回 float32 指针
func float32Ptr(f float64) *float32 {
	val := float32(f)
	return &val
}

// intPtr 返回 int 指针
func intPtr(i int) *int {
	return &i
}

// Close 关闭服务
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	if s.provider != nil {
		s.provider.Hangup()
	}

	s.closed = true
	return nil
}
