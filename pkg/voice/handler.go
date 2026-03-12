package voice

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/voice/asr"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Handler WebSocket处理器
type Handler struct {
	logger           *zap.Logger
	asrPool          *asr.Pool // ASR连接池
	maxASRConcurrent int       // 最大ASR并发数
}

// NewHandler 创建新的处理器
func NewHandler(logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.L()
	}
	// 默认最大并发数为50，可以通过环境变量或配置调整
	maxConcurrent := 50
	pool := asr.GetGlobalPool(maxConcurrent, logger)
	return &Handler{
		logger:           logger,
		asrPool:          pool,
		maxASRConcurrent: maxConcurrent,
	}
}

// NewHandlerWithPool 创建带自定义连接池的处理器
func NewHandlerWithPool(logger *zap.Logger, maxASRConcurrent int) *Handler {
	if logger == nil {
		logger = zap.L()
	}
	if maxASRConcurrent <= 0 {
		maxASRConcurrent = 50
	}
	pool := asr.GetGlobalPool(maxASRConcurrent, logger)
	return &Handler{
		logger:           logger,
		asrPool:          pool,
		maxASRConcurrent: maxASRConcurrent,
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
	llmModel := "gpt-3.5-turbo"
	assistantTemperature := 0.6
	assistantMaxTokens := 0
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
		ASRPool:      h.asrPool, // 设置ASR连接池
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
