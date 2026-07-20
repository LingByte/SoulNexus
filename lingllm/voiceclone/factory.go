package voiceclone

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"
)

// Factory 语音克隆服务工厂
type Factory struct{}

// NewFactory 创建工厂实例
func NewFactory() *Factory {
	return &Factory{}
}

// CreateService 根据配置创建语音克隆服务
func (f *Factory) CreateService(config *Config) (VoiceCloneService, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	switch config.Provider {
	case ProviderXunfei:
		return f.createXunfeiService(config.Options)
	case ProviderVolcengine:
		return f.createVolcengineService(config.Options)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", config.Provider)
	}
}

// createXunfeiService 创建讯飞克隆服务
func (f *Factory) createXunfeiService(options map[string]interface{}) (VoiceCloneService, error) {
	appID, _ := options["app_id"].(string)
	apiKey, _ := options["api_key"].(string)
	baseURL, _ := options["base_url"].(string)
	timeout, _ := options["timeout"].(int)
	wsAppID, _ := options["ws_app_id"].(string)
	wsAPIKey, _ := options["ws_api_key"].(string)
	wsAPISecret, _ := options["ws_api_secret"].(string)
	engineVersion, _ := options["engine_version"].(string)
	vcn, _ := options["vcn"].(string)

	if appID == "" || apiKey == "" {
		return nil, fmt.Errorf("xunfei app_id and api_key are required")
	}

	return NewXunfeiCloneService(XunfeiCloneConfig{
		AppID:              appID,
		APIKey:             apiKey,
		BaseURL:            baseURL,
		Timeout:            timeout,
		EngineVersion:      engineVersion,
		VCN:                vcn,
		WebSocketAppID:     wsAppID,
		WebSocketAPIKey:    wsAPIKey,
		WebSocketAPISecret: wsAPISecret,
	}), nil
}

// createVolcengineService 创建火山引擎克隆服务
func (f *Factory) createVolcengineService(options map[string]interface{}) (VoiceCloneService, error) {
	appID, _ := options["app_id"].(string)
	token, _ := options["token"].(string)
	cluster, _ := options["cluster"].(string)
	resourceID, _ := options["resource_id"].(string)
	modelType, _ := options["model_type"].(int)

	// Token 对于 HTTP API（训练和查询状态）和 WebSocket API（合成）都是必需的
	if token == "" {
		return nil, fmt.Errorf("volcengine token is required (for HTTP API: training/status query, and WebSocket API: synthesis)")
	}

	if cluster == "" {
		cluster = "volcano_icl"
	}

	// 解析其他可选参数
	voiceType, _ := options["voice_type"].(string)
	encoding, _ := options["encoding"].(string)
	sampleRate, _ := options["sample_rate"].(int)
	bitDepth, _ := options["bit_depth"].(int)
	channels, _ := options["channels"].(int)
	frameDuration, _ := options["frame_duration"].(string)
	speedRatio, _ := options["speed_ratio"].(float64)
	trainingTimes, _ := options["training_times"].(int)

	return NewVolcengineCloneService(VolcengineCloneConfig{
		AppID:         appID,
		Token:         token,
		Cluster:       cluster,
		ResourceID:    resourceID,
		ModelType:     modelType,
		VoiceType:     voiceType,
		Encoding:      encoding,
		SampleRate:    sampleRate,
		BitDepth:      bitDepth,
		Channels:      channels,
		FrameDuration: frameDuration,
		SpeedRatio:    speedRatio,
		TrainingTimes: trainingTimes,
	}), nil
}

// CreateServiceFromJSON 从JSON配置创建服务
func (f *Factory) CreateServiceFromJSON(jsonConfig string) (VoiceCloneService, error) {
	var config Config
	if err := json.Unmarshal([]byte(jsonConfig), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return f.CreateService(&config)
}

// ValidateConfig 验证配置
func (f *Factory) ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	switch config.Provider {
	case ProviderXunfei:
		appID, _ := config.Options["app_id"].(string)
		apiKey, _ := config.Options["api_key"].(string)
		if appID == "" || apiKey == "" {
			return fmt.Errorf("xunfei app_id and api_key are required")
		}
	case ProviderVolcengine:
		appID, _ := config.Options["app_id"].(string)
		token, _ := config.Options["token"].(string)
		if appID == "" || token == "" {
			return fmt.Errorf("volcengine app_id and token are required")
		}
	default:
		return fmt.Errorf("unsupported provider: %s", config.Provider)
	}

	return nil
}

// GetSupportedProviders 获取支持的提供商列表
func (f *Factory) GetSupportedProviders() []Provider {
	return []Provider{
		ProviderXunfei,
		ProviderVolcengine,
	}
}
