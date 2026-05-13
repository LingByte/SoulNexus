package recognizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"strconv"
	"strings"

	"cloud.google.com/go/speech/apiv1/speechpb"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/utils"
)

// ConfigReader 配置读取器 - 简化配置读取逻辑
type ConfigReader struct {
	config map[string]interface{}
}

// NewConfigReader 创建配置读取器
func NewConfigReader(config map[string]interface{}) *ConfigReader {
	return &ConfigReader{config: config}
}

// String 获取字符串值，支持多个键名和默认值
// 用法: String("key1", "key2", "default_value")
func (r *ConfigReader) String(keysAndDefault ...string) string {
	if len(keysAndDefault) == 0 {
		return ""
	}

	// 最后一个可能是默认值
	defaultValue := ""
	keys := keysAndDefault
	if len(keysAndDefault) > 1 {
		// 如果最后一个不是key（可能是默认值），尝试获取
		lastKey := keysAndDefault[len(keysAndDefault)-1]
		if _, exists := r.config[lastKey]; !exists {
			// 最后一个不是配置中的key，作为默认值
			defaultValue = lastKey
			keys = keysAndDefault[:len(keysAndDefault)-1]
		}
	}

	for _, key := range keys {
		if val, ok := r.config[key].(string); ok && val != "" {
			return val
		}
	}
	return defaultValue
}

// Int 获取整数值，支持多个键名和默认值
// 用法: Int("key1", "key2", 100)
func (r *ConfigReader) Int(keys ...interface{}) int {
	var defaultValue int
	var keyStrings []string

	for _, k := range keys {
		switch v := k.(type) {
		case string:
			keyStrings = append(keyStrings, v)
		case int:
			defaultValue = v
		}
	}

	for _, key := range keyStrings {
		if val, ok := r.config[key]; ok {
			switch v := val.(type) {
			case int:
				return v
			case int64:
				return int(v)
			case float64:
				return int(v)
			case string:
				if intVal, err := strconv.Atoi(v); err == nil {
					return intVal
				}
			}
		}
	}
	return defaultValue
}

// GetVendor 获取vendor枚举值（公开函数，供其他包使用）
func GetVendor(provider string) Vendor {
	if provider == "tencent" {
		return VendorQCloud
	}
	return Vendor(provider)
}

// NewTranscriberConfigFromMap 从 map[string]interface{} 创建 TranscriberConfig
// 这是统一的配置入口，所有配置解析逻辑都在这里
func NewTranscriberConfigFromMap(
	provider string,
	config map[string]interface{},
	language string,
) (TranscriberConfig, error) {
	switch provider {
	case "tencent", "qcloud":
		return buildQCloudConfig(config)
	case "google":
		return buildGoogleConfig(config, language)
	case "qiniu":
		return buildQiniuConfig(config)
	case "funasr":
		return buildFunASRConfig(config)
	case "funasr_realtime":
		return buildFunASRRealtimeConfig(config)
	case "volcengine":
		return buildVolcengineConfig(config)
	case "volcllmasr", "volcengine_llm":
		return buildVolcengineLLMConfig(config)
	case "gladia":
		return buildGladiaConfig(config)
	case "deepgram":
		return buildDeepgramConfig(config, language)
	case "aws":
		return buildAwsConfig(config, language)
	case "baidu":
		return buildBaiduConfig(config)
	case "voiceapi":
		return buildVoiceAPIConfig(config)
	case "whisper":
		return buildWhisperConfig(config)
	case "local":
		return buildLocalConfig(config)
	default:
		return nil, fmt.Errorf("unsupported ASR provider: %s", provider)
	}
}

// buildQCloudConfig 构建腾讯云ASR配置
func buildQCloudConfig(config map[string]interface{}) (*QCloudASROption, error) {
	cfg := NewConfigReader(config)

	// 优先使用配置中的值，如果没有则使用环境变量
	appID := cfg.String("app_id", "appId")
	if appID == "" {
		appID = utils.GetEnv("QCLOUD_APP_ID")
	}

	secretID := cfg.String("secret_id", "secretId")
	if secretID == "" {
		secretID = utils.GetEnv("QCLOUD_SECRET_ID")
	}

	secretKey := cfg.String("secret_key", "secretKey", "secret")
	if secretKey == "" {
		secretKey = utils.GetEnv("QCLOUD_SECRET")
	}

	if appID == "" || secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("腾讯云ASR配置不完整：缺少appId、secretId或secretKey (配置: appId=%s, secretId=%s, secretKey=%s)",
			appID, secretID, secretKey)
	}

	opt := NewQcloudASROption(appID, secretID, secretKey)
	return &opt, nil
}

// buildGoogleConfig 构建Google ASR配置
func buildGoogleConfig(config map[string]interface{}, language string) (*GoogleASROption, error) {
	cfg := NewConfigReader(config)

	encoding := cfg.String("encoding", "LINEAR16")
	sampleRate := cfg.Int("sampleRate", "sample_rate", 16000)
	languageCode := cfg.String("languageCode", "language_code", language)
	if languageCode == "" {
		languageCode = "zh-CN"
	}

	// 转换为Google编码类型
	var googleEncoding speechpb.RecognitionConfig_AudioEncoding
	switch strings.ToUpper(encoding) {
	case "LINEAR16", "PCM":
		googleEncoding = speechpb.RecognitionConfig_LINEAR16
	case "FLAC":
		googleEncoding = speechpb.RecognitionConfig_FLAC
	case "MULAW":
		googleEncoding = speechpb.RecognitionConfig_MULAW
	case "AMR":
		googleEncoding = speechpb.RecognitionConfig_AMR
	case "AMR_WB":
		googleEncoding = speechpb.RecognitionConfig_AMR_WB
	case "OGG_OPUS":
		googleEncoding = speechpb.RecognitionConfig_OGG_OPUS
	case "SPEEX_WITH_HEADER_BYTE":
		googleEncoding = speechpb.RecognitionConfig_SPEEX_WITH_HEADER_BYTE
	default:
		googleEncoding = speechpb.RecognitionConfig_LINEAR16
	}

	opt := NewGoogleASROption(googleEncoding, int32(sampleRate), languageCode)
	return &opt, nil
}

// buildQiniuConfig 构建七牛云ASR配置
func buildQiniuConfig(config map[string]interface{}) (*QiniuASROption, error) {
	cfg := NewConfigReader(config)
	apiKey := cfg.String("apiKey", "api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("七牛云ASR配置不完整：缺少apiKey")
	}
	opt := NewQiniuASROption(apiKey)
	return &opt, nil
}

// buildFunASRConfig 构建FunASR配置
func buildFunASRConfig(config map[string]interface{}) (*FunASROption, error) {
	cfg := NewConfigReader(config)
	url := cfg.String("url", "wss://dashscope.aliyuncs.com/api-ws/v1/inference")
	opt := NewFunASROption(url)
	return &opt, nil
}

// buildFunASRRealtimeConfig 构建FunASR实时配置
func buildFunASRRealtimeConfig(config map[string]interface{}) (*FunAsrRealtimeOption, error) {
	cfg := NewConfigReader(config)
	opt := FunAsrRealtimeOption{
		Url:           cfg.String("url", "wss://dashscope.aliyuncs.com/api-ws/v1/inference"),
		ApiKey:        cfg.String("apiKey", "api_key"),
		Model:         cfg.String("model", "fun-asr-realtime"),
		SampleRate:    cfg.Int("sampleRate", "sample_rate", 16000),
		Format:        cfg.String("format", "pcm"),
		LanguageHints: cfg.String("languageHints", "language_hints", "zh"),
	}
	return &opt, nil
}

// buildVolcengineConfig 构建火山引擎标准ASR配置
func buildVolcengineConfig(config map[string]interface{}) (*VolcengineOption, error) {
	cfg := NewConfigReader(config)

	url := cfg.String("url", "wss://openspeech.bytedance.com/api/v2/asr")
	appID := cfg.String("appId", "app_id")
	token := cfg.String("token")
	cluster := cfg.String("cluster", "volcano_tts")
	format := cfg.String("format", "raw")

	if appID == "" || token == "" {
		return nil, fmt.Errorf("火山引擎ASR配置不完整：缺少appId或token")
	}

	opt := NewVolcengineOption(appID, token, cluster, format)
	opt.Url = url
	return &opt, nil
}

// buildVolcengineLLMConfig 构建火山引擎LLM ASR配置
func buildVolcengineLLMConfig(config map[string]interface{}) (*VolcengineLLMOption, error) {
	cfg := NewConfigReader(config)
	token := cfg.String("token")
	appID := cfg.String("appId", "app_id")
	if token == "" || appID == "" {
		return nil, fmt.Errorf("火山引擎LLM ASR配置不完整：缺少token或appId")
	}
	opt := NewVolcengineLLMOption(token, appID)
	return &opt, nil
}

// buildGladiaConfig 构建Gladia ASR配置
func buildGladiaConfig(config map[string]interface{}) (*GladiaASROption, error) {
	cfg := NewConfigReader(config)
	apiKey := cfg.String("apiKey", "api_key")
	encoding := cfg.String("encoding", "WAV/PCM")
	if apiKey == "" {
		return nil, fmt.Errorf("Gladia ASR配置不完整：缺少apiKey")
	}
	opt := NewGladiaASROption(apiKey, encoding)
	return &opt, nil
}

// buildDeepgramConfig 构建Deepgram ASR配置
func buildDeepgramConfig(config map[string]interface{}, language string) (*DeepgramASROption, error) {
	cfg := NewConfigReader(config)
	apiKey := cfg.String("apiKey", "api_key")
	if apiKey == "" {
		apiKey = utils.GetEnv("DEEPGRAM_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("Deepgram ASR配置不完整：缺少apiKey")
	}

	model := cfg.String("model", "nova-2")
	lang := cfg.String("language", language)
	if lang == "" {
		lang = "en-US"
	}

	opt := NewDeepgramASROption(apiKey, model, lang)
	return &opt, nil
}

// buildAwsConfig 构建AWS ASR配置
func buildAwsConfig(config map[string]interface{}, language string) (*AwsASROption, error) {
	cfg := NewConfigReader(config)
	appID := cfg.String("appId", "app_id")
	region := cfg.String("region", "us-east-1")
	lang := cfg.String("language", language)
	if lang == "" {
		lang = "en-US"
	}

	opt := NewAwsASROption(appID, region, lang)
	return &opt, nil
}

// buildBaiduConfig 构建百度ASR配置
func buildBaiduConfig(config map[string]interface{}) (*BaiduASROption, error) {
	cfg := NewConfigReader(config)
	appID := cfg.Int("appId", "app_id")
	appKey := cfg.String("appKey", "app_key")
	devPid := cfg.Int("devPid", "dev_pid", 1537)
	format := cfg.String("format", "pcm")
	sampleRate := cfg.Int("sampleRate", "sample_rate", 16000)

	if appID == 0 || appKey == "" {
		return nil, fmt.Errorf("百度ASR配置不完整：缺少appId或appKey")
	}

	opt := NewBaiduASROption(appID, appKey, devPid, format, sampleRate)
	return &opt, nil
}

// buildVoiceAPIConfig 构建VoiceAPI ASR配置
func buildVoiceAPIConfig(config map[string]interface{}) (*VoiceapiASROption, error) {
	cfg := NewConfigReader(config)
	url := cfg.String("url", "")
	if url == "" {
		return nil, fmt.Errorf("VoiceAPI ASR配置不完整：缺少url")
	}

	opt := NewVoiceapiASROption(url)
	return &opt, nil
}

// buildWhisperConfig 构建Whisper ASR配置
func buildWhisperConfig(config map[string]interface{}) (*WhisperASROption, error) {
	cfg := NewConfigReader(config)
	url := cfg.String("url", "")
	model := cfg.String("model", "whisper-1")

	if url == "" {
		return nil, fmt.Errorf("Whisper ASR配置不完整：缺少url")
	}

	opt := NewWhisperASROption(url, model)
	return &opt, nil
}

// buildLocalConfig 构建本地ASR配置
func buildLocalConfig(config map[string]interface{}) (*LocalASRConfig, error) {
	cfg := NewConfigReader(config)
	modelPath := cfg.String("modelPath", "model_path")

	opt := &LocalASRConfig{
		ModelPath: modelPath,
	}
	return opt, nil
}
