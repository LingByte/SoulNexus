package factory

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/LingByte/SoulNexus/pkg/voice/errhandler"
	"go.uber.org/zap"
)

const (
	// DefaultTTSSpeedRatio 默认TTS语速倍率
	DefaultTTSSpeedRatio = 1.2
)

// ServiceFactory 服务工厂实现
type ServiceFactory struct {
	transcriberFactory *recognizer.DefaultTranscriberFactory
	logger             *zap.Logger
}

// NewServiceFactory 创建服务工厂
func NewServiceFactory(transcriberFactory *recognizer.DefaultTranscriberFactory, logger *zap.Logger) *ServiceFactory {
	return &ServiceFactory{
		transcriberFactory: transcriberFactory,
		logger:             logger,
	}
}

// CreateASR 创建ASR服务
func (f *ServiceFactory) CreateASR(credential *models.UserCredential, language string) (recognizer.TranscribeService, error) {
	asrProvider := credential.GetASRProvider()
	if asrProvider == "" {
		return nil, errhandler.NewRecoverableError("Factory", "ASR provider未配置", nil)
	}
	asrConfig := make(map[string]interface{})
	asrConfig["provider"] = asrProvider
	asrConfig["language"] = language

	if credential.AsrConfig != nil {
		for key, value := range credential.AsrConfig {
			asrConfig[key] = value
		}
	}
	vendor := recognizer.GetVendor(asrProvider)
	if f.transcriberFactory != nil && !f.transcriberFactory.IsVendorSupported(vendor) {
		supported := f.transcriberFactory.GetSupportedVendors()
		return nil, errhandler.NewRecoverableError("Factory", fmt.Sprintf("不支持的ASR提供商: %s, 支持的提供商: %v", asrProvider, supported), nil)
	}
	config, err := recognizer.NewTranscriberConfigFromMap(asrProvider, asrConfig, language)
	if err != nil {
		return nil, errhandler.NewRecoverableError("Factory", "解析ASR配置失败", err)
	}
	if f.transcriberFactory == nil {
		f.transcriberFactory = recognizer.GetGlobalFactory()
	}
	asrService, err := f.transcriberFactory.CreateTranscriber(config)
	if err != nil {
		return nil, errhandler.NewRecoverableError("Factory", "创建ASR服务失败", err)
	}

	return asrService, nil
}

// CreateTTS 创建TTS服务
func (f *ServiceFactory) CreateTTS(credential *models.UserCredential, speaker string) (synthesizer.SynthesisService, error) {
	ttsProvider := credential.GetTTSProvider()
	if ttsProvider == "" {
		return nil, errhandler.NewRecoverableError("Factory", "TTS provider未配置", nil)
	}
	ttsConfig := make(synthesizer.TTSCredentialConfig)
	ttsConfig["provider"] = ttsProvider

	if credential.TtsConfig != nil {
		for key, value := range credential.TtsConfig {
			ttsConfig[key] = value
		}
	}

	if _, exists := ttsConfig["voiceType"]; !exists && speaker != "" {
		ttsConfig["voiceType"] = speaker
	}
	if _, exists := ttsConfig["voice_type"]; !exists && speaker != "" {
		ttsConfig["voice_type"] = speaker
	}

	// 设置默认语速
	setDefaultTTSSpeed(ttsConfig, ttsProvider)

	ttsService, err := synthesizer.NewSynthesisServiceFromCredential(ttsConfig)
	if err != nil {
		return nil, errhandler.NewRecoverableError("Factory", "创建TTS服务失败", err)
	}

	return ttsService, nil
}

// CreateLLM 创建LLM服务
func (f *ServiceFactory) CreateLLM(ctx context.Context, credential *models.UserCredential, systemPrompt string) (llm.LLMProvider, error) {
	provider, err := llm.NewLLMProvider(ctx, credential.LLMProvider, credential.LLMApiKey, credential.LLMApiURL, systemPrompt)
	if err != nil {
		return nil, errhandler.NewRecoverableError("Factory", "创建LLM服务失败", err)
	}

	return provider, nil
}

// setDefaultTTSSpeed 设置默认TTS语速
func setDefaultTTSSpeed(ttsConfig synthesizer.TTSCredentialConfig, provider string) {
	// 检查是否已经设置了语速
	if _, exists := ttsConfig["speedRatio"]; exists {
		return
	}
	if _, exists := ttsConfig["speed_ratio"]; exists {
		return
	}
	if _, exists := ttsConfig["speed"]; exists {
		return
	}

	// 根据提供商设置默认语速
	switch provider {
	case "openai":
		ttsConfig["speed"] = DefaultTTSSpeedRatio
	default:
		// 大多数提供商使用 speedRatio
		ttsConfig["speedRatio"] = DefaultTTSSpeedRatio
	}
}
