package voiceclone

import (
	"encoding/json"
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/utils"
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

// CreateServiceFromEnv 从环境变量创建服务
func (f *Factory) CreateServiceFromEnv(provider Provider) (VoiceCloneService, error) {
	config := &Config{
		Provider: provider,
		Options:  make(map[string]interface{}),
	}

	switch provider {
	case ProviderXunfei:
		config.Options["app_id"] = utils.GetEnv("XUNFEI_APP_ID")
		config.Options["api_key"] = utils.GetEnv("XUNFEI_API_KEY")
		config.Options["base_url"] = utils.GetEnv("XUNFEI_BASE_URL")
		if config.Options["base_url"] == "" {
			// 使用讯飞默认值
			config.Options["base_url"] = "http://opentrain.xfyousheng.com"
		}
		config.Options["timeout"] = utils.GetIntEnv("XUNFEI_TIMEOUT")
		if config.Options["timeout"] == 0 {
			config.Options["timeout"] = 30
		}
		// WebSocket配置
		config.Options["ws_app_id"] = utils.GetEnv("XUNFEI_WS_APP_ID")
		config.Options["ws_api_key"] = utils.GetEnv("XUNFEI_WS_API_KEY")
		config.Options["ws_api_secret"] = utils.GetEnv("XUNFEI_WS_API_SECRET")

	case ProviderVolcengine:
		config.Options["app_id"] = utils.GetEnv("VOLCENGINE_CLONE_APP_ID")
		config.Options["token"] = utils.GetEnv("VOLCENGINE_CLONE_TOKEN")
		config.Options["cluster"] = utils.GetEnv("VOLCENGINE_CLONE_CLUSTER")
		config.Options["voice_type"] = utils.GetEnv("VOLCENGINE_CLONE_VOICE_TYPE")
		config.Options["encoding"] = utils.GetEnv("VOLCENGINE_CLONE_ENCODING")
		if sampleRate := utils.GetIntEnv("VOLCENGINE_CLONE_SAMPLE_RATE"); sampleRate > 0 {
			config.Options["sample_rate"] = sampleRate
		}
		if bitDepth := utils.GetIntEnv("VOLCENGINE_CLONE_BIT_DEPTH"); bitDepth > 0 {
			config.Options["bit_depth"] = bitDepth
		}
		if channels := utils.GetIntEnv("VOLCENGINE_CLONE_CHANNELS"); channels > 0 {
			config.Options["channels"] = channels
		}
		config.Options["frame_duration"] = utils.GetEnv("VOLCENGINE_CLONE_FRAME_DURATION")
		if speedRatio := utils.GetFloatEnv("VOLCENGINE_CLONE_SPEED_RATIO"); speedRatio > 0 {
			config.Options["speed_ratio"] = speedRatio
		}
		if trainingTimes := utils.GetIntEnv("VOLCENGINE_CLONE_TRAINING_TIMES"); trainingTimes > 0 {
			config.Options["training_times"] = trainingTimes
		}
		if config.Options["cluster"] == "" {
			config.Options["cluster"] = "volcano_icl"
		}

	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	return f.CreateService(config)
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

	if appID == "" || apiKey == "" {
		return nil, fmt.Errorf("xunfei app_id and api_key are required")
	}

	return NewXunfeiCloneService(XunfeiCloneConfig{
		AppID:              appID,
		APIKey:             apiKey,
		BaseURL:            baseURL,
		Timeout:            timeout,
		WebSocketAppID:     wsAppID,
		WebSocketAPIKey:    wsAPIKey,
		WebSocketAPISecret: wsAPISecret,
	}), nil
}

// createVolcengineService 创建火山引擎克隆服务
// 完全效仿 voiceserver-main，只支持 WebSocket，需要 token
func (f *Factory) createVolcengineService(options map[string]interface{}) (VoiceCloneService, error) {
	appID, _ := options["app_id"].(string)
	token, _ := options["token"].(string)
	cluster, _ := options["cluster"].(string)

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
