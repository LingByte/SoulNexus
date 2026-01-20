package hardware

import (
	"context"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Handler WebSocket处理器
type Handler struct {
	logger *zap.Logger
}

// NewHandler 创建新的处理器
func NewHandler(logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.L()
	}
	return &Handler{
		logger: logger,
	}
}

// HandleWebSocket 处理WebSocket连接
func (h *Handler) HandleWebSocket(
	ctx context.Context,
	conn *websocket.Conn,
	credential *models.UserCredential,
	assistantID int,
	language, speaker string,
	temperature float64,
	systemPrompt string,
	knowledgeKey string,
	db *gorm.DB,
) {
	defer conn.Close()

	// 查询助手配置（获取LLM模型等）
	llmModel := DefaultLLMModel
	assistantTemperature := 0.6
	assistantMaxTokens := 70
	enableVAD := true
	vadThreshold := 500.0
	vadConsecutiveFrames := 2
	if assistantID > 0 && db != nil {
		var assistant models.Assistant
		if err := db.First(&assistant, assistantID).Error; err == nil {
			if assistant.LLMModel != "" {
				llmModel = assistant.LLMModel
			}
			if assistant.Temperature > 0 {
				assistantTemperature = float64(assistant.Temperature)
			}
			if assistant.MaxTokens > 0 {
				assistantMaxTokens = assistant.MaxTokens
			}
			// 读取 VAD 配置
			enableVAD = assistant.EnableVAD
			if assistant.VADThreshold > 0 {
				vadThreshold = assistant.VADThreshold
			}
			if assistant.VADConsecutiveFrames > 0 {
				vadConsecutiveFrames = assistant.VADConsecutiveFrames
			}
		}
	}

	// 如果temperature为0或未设置，使用assistant的temperature
	if temperature <= 0 {
		temperature = assistantTemperature
	}

	// 创建会话配置
	config := &SessionConfig{
		Conn:         conn,
		Credential:   credential,
		AssistantID:  assistantID,
		Language:     language,
		Speaker:      speaker,
		Temperature:  temperature,
		MaxTokens:    assistantMaxTokens,
		SystemPrompt: systemPrompt,
		KnowledgeKey: knowledgeKey,
		LLMModel:     llmModel,
		DB:           db,
		Logger:       h.logger,
		Context:      ctx,
		// VAD 配置
		EnableVAD:            enableVAD,
		VADThreshold:         vadThreshold,
		VADConsecutiveFrames: vadConsecutiveFrames,
	}

	// 创建会话
	session, err := NewSession(config)
	if err != nil {
		h.logger.Error("创建会话失败", zap.Error(err))
		return
	}

	// 启动会话
	if err := session.Start(); err != nil {
		h.logger.Error("启动会话失败", zap.Error(err))
		return
	}

	// 等待会话结束
	<-ctx.Done()

	// 停止会话
	if err := session.Stop(); err != nil {
		h.logger.Error("停止会话失败", zap.Error(err))
	}
}
