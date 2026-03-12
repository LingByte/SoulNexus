package hardware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/cache"
	"github.com/LingByte/SoulNexus/pkg/hardware/constants"
	"github.com/LingByte/SoulNexus/pkg/hardware/protocol"
	"github.com/LingByte/SoulNexus/pkg/voiceprint"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HardwareOptions hardware options
type HardwareOptions struct {
	Conn                 *websocket.Conn        // websocket connection
	AssistantID          uint                   // assistant id
	Credential           *models.UserCredential // credential
	Language             string                 // language
	Speaker              string                 // speaker
	Temperature          float64                // temperature
	SystemPrompt         string                 // ai system prompt
	KnowledgeKey         string                 // knowledge key
	UserID               uint                   // user id
	DeviceID             *string                // device id
	MacAddress           string                 // mac address
	LLMModel             string                 // chat llm model for assistant
	MaxLLMToken          int                    // max llm token
	EnableVAD            bool                   // enable VAD
	VADThreshold         float64                // VAD threshold
	VADConsecutiveFrames int                    // VAD consecutive frames
	VoiceCloneID         *int                   // voice clone id (optional)
}

// HardwareHandler hardware handler
type HardwareHandler struct {
	logger *zap.Logger
	db     *gorm.DB
}

// NewHardwareHandler create hardware handler
func NewHardwareHandler(db *gorm.DB, logger *zap.Logger) *HardwareHandler {
	if logger == nil {
		logger = zap.L()
	}
	return &HardwareHandler{
		logger: logger,
		db:     db,
	}
}

// HandlerHardwareWebsocket handler hardware websocket
func (h *HardwareHandler) HandlerHardwareWebsocket(
	ctx context.Context,
	options *HardwareOptions) {
	if options == nil || options.AssistantID == 0 {
		h.logger.Error("options is nil or assistantID is 0 please check")
		return
	}
	options.loadConfigs()
	defer options.Conn.Close()
	h.logger.Info(fmt.Sprintf("create hardwareSession assistantID: %d", options.AssistantID))
	voiceprintConfig := voiceprint.DefaultConfig()
	if err := voiceprintConfig.Validate(); err != nil {
		h.logger.Warn("[Handler] --- 验证声纹识别服务配置失败", zap.Error(err))
	}
	voiceprintService, err := voiceprint.NewService(voiceprintConfig, cache.GetGlobalCache())
	if err != nil {
		h.logger.Warn("[Handler] --- 初始化声纹识别服务失败", zap.Error(err))
		voiceprintService = nil
	}
	session := protocol.NewHardwareSession(ctx, &protocol.HardwareSessionOption{
		Conn:                 options.Conn,
		Logger:               h.logger,
		AssistantID:          options.AssistantID,
		LLMModel:             options.LLMModel,
		Credential:           options.Credential,
		SystemPrompt:         options.SystemPrompt,
		MaxToken:             options.MaxLLMToken,
		Speaker:              options.Speaker,
		EnableVAD:            options.EnableVAD,
		VADThreshold:         options.VADThreshold,
		VADConsecutiveFrames: options.VADConsecutiveFrames,
		DB:                   h.db,
		VoiceprintService:    voiceprintService,
		DeviceID:             options.DeviceID,
		MacAddress:           options.MacAddress,
		VoiceCloneID:         options.VoiceCloneID,
	})
	if err := session.Start(); err != nil {
		h.logger.Error("[Handler] start session failed: ", zap.Error(err))
		return
	}
	<-ctx.Done()
	if err := session.Stop(); err != nil {
		h.logger.Error("[Handler] stop session failed: ", zap.Error(err))
	}
}

func (ho *HardwareOptions) loadConfigs() *HardwareOptions {
	if ho.LLMModel == "" {
		ho.LLMModel = constants.DefaultLLMModel
	}
	if ho.Temperature <= 0 {
		ho.Temperature = constants.DefaultTemperature
	}
	if ho.EnableVAD == false {
		ho.EnableVAD = constants.DefaultEnabledVAD
	}
	if ho.VADThreshold <= 0 {
		ho.VADThreshold = constants.DefaultVADThreshold
	}
	if ho.VADConsecutiveFrames <= 0 {
		ho.VADConsecutiveFrames = constants.DefaultVADConsecutiveFrames
	}
	if ho.MaxLLMToken <= 0 {
		ho.MaxLLMToken = constants.DefaultMaxLLMToken
	}
	return ho
}
