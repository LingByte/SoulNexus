package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"sync"
)

// SynthesisConfig 统一的TTS配置接口
type SynthesisConfig interface {
	GetProvider() TTSProvider
}

// SynthesisFactory TTS工厂接口
type SynthesisFactory interface {
	// CreateEngine 根据配置创建 AudioSynthesisEngine
	CreateEngine(config SynthesisConfig) (AudioSynthesisEngine, error)
	// GetSupportedProviders 获取支持的提供商列表
	GetSupportedProviders() []TTSProvider
	// IsProviderSupported 检查提供商是否支持
	IsProviderSupported(provider TTSProvider) bool
	// RegisterCreator 注册创建函数
	RegisterCreator(provider TTSProvider, creator func(SynthesisConfig) (AudioSynthesisEngine, error))
}

// DefaultSynthesisFactory 默认TTS工厂实现
type DefaultSynthesisFactory struct {
	creators map[TTSProvider]func(SynthesisConfig) (AudioSynthesisEngine, error)
	mu       sync.RWMutex
}

// NewSynthesisFactory 创建新的TTS工厂实例
func NewSynthesisFactory() *DefaultSynthesisFactory {
	factory := &DefaultSynthesisFactory{
		creators: make(map[TTSProvider]func(SynthesisConfig) (AudioSynthesisEngine, error)),
	}
	factory.registerDefaultCreators()
	return factory
}

// RegisterCreator 注册创建函数
func (f *DefaultSynthesisFactory) RegisterCreator(provider TTSProvider, creator func(SynthesisConfig) (AudioSynthesisEngine, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creators[provider] = creator
}

// CreateEngine 创建 AudioSynthesisEngine
func (f *DefaultSynthesisFactory) CreateEngine(config SynthesisConfig) (AudioSynthesisEngine, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	provider := config.GetProvider()
	f.mu.RLock()
	creator, exists := f.creators[provider]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not supported", provider)
	}

	return creator(config)
}

// GetSupportedProviders 获取支持的提供商列表
func (f *DefaultSynthesisFactory) GetSupportedProviders() []TTSProvider {
	f.mu.RLock()
	defer f.mu.RUnlock()

	providers := make([]TTSProvider, 0, len(f.creators))
	for provider := range f.creators {
		providers = append(providers, provider)
	}
	return providers
}

// IsProviderSupported 检查提供商是否支持
func (f *DefaultSynthesisFactory) IsProviderSupported(provider TTSProvider) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, exists := f.creators[provider]
	return exists
}

// registerDefaultCreators 注册默认创建函数
func (f *DefaultSynthesisFactory) registerDefaultCreators() {
	// 注册QCloud
	f.RegisterCreator(ProviderTencent, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		qcloudConfig, ok := config.(*QCloudTTSConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for qcloud, expected *QCloudTTSConfig")
		}
		return NewQCloudService(*qcloudConfig), nil
	})

	// 注册Xunfei
	f.RegisterCreator(ProviderXunfei, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		xunfeiConfig, ok := config.(*XunfeiTTSConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for xunfei, expected *XunfeiTTSConfig")
		}
		return NewXunfeiService(*xunfeiConfig), nil
	})

	// 注册Qiniu
	f.RegisterCreator(ProviderQiniu, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		qiniuConfig, ok := config.(*QiniuTTSConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for qiniu, expected *QiniuTTSConfig")
		}
		return NewQiniuService(*qiniuConfig), nil
	})

	// 注册AWS
	f.RegisterCreator(ProviderAWS, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		awsConfig, ok := config.(*AmazonTTSConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for aws, expected *AmazonTTSConfig")
		}
		return NewAmazonService(*awsConfig), nil
	})

	// 注册Baidu
	f.RegisterCreator(ProviderBaidu, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		baiduConfig, ok := config.(*BaiduTTSConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for baidu, expected *BaiduTTSConfig")
		}
		return NewBaiduService(*baiduConfig), nil
	})

	// 注册Google
	f.RegisterCreator(ProviderGoogle, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		googleConfig, ok := config.(*GoogleTTSOption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for google, expected *GoogleTTSOption")
		}
		return NewGoogleService(*googleConfig), nil
	})

	// 注册Azure
	f.RegisterCreator(ProviderAzure, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		azureConfig, ok := config.(*AzureConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for azure, expected *AzureConfig")
		}
		return NewAzureService(*azureConfig), nil
	})

	// 注册OpenAI
	f.RegisterCreator(ProviderOpenAI, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		openaiConfig, ok := config.(*OpenAIConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for openai, expected *OpenAIConfig")
		}
		return NewOpenAIService(*openaiConfig), nil
	})

	// 注册ElevenLabs
	f.RegisterCreator(ProviderElevenLabs, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		elevenlabsConfig, ok := config.(*ElevenLabsConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for elevenlabs, expected *ElevenLabsConfig")
		}
		return NewElevenLabsService(*elevenlabsConfig), nil
	})

	// 注册Local
	f.RegisterCreator(ProviderLocal, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		localConfig, ok := config.(*LocalTTSConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for local, expected *LocalTTSConfig")
		}
		return NewLocalService(*localConfig), nil
	})

	// 注册LocalGoSpeech
	f.RegisterCreator(ProviderLocalGoSpeech, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		localGoSpeechConfig, ok := config.(*LocalGoSpeechConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for local_gospeech, expected *LocalGoSpeechConfig")
		}
		return NewLocalGoSpeechService(localGoSpeechConfig)
	})

	// 注册FishSpeech
	f.RegisterCreator(ProviderFishSpeech, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		fishspeechConfig, ok := config.(*FishSpeechConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for fishspeech, expected *FishSpeechConfig")
		}
		return NewFishSpeechService(*fishspeechConfig), nil
	})

	// 注册FishAudio
	f.RegisterCreator(ProviderFishAudio, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		fishaudioConfig, ok := config.(*FishAudioConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for fishaudio, expected *FishAudioConfig")
		}
		return NewFishAudioService(*fishaudioConfig), nil
	})

	// 注册Coqui
	f.RegisterCreator(ProviderCoqui, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		coquiConfig, ok := config.(*CoquiTTSOption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for coqui, expected *CoquiTTSOption")
		}
		return NewCoquiService(*coquiConfig), nil
	})

	// 注册Volcengine
	f.RegisterCreator(ProviderVolcengine, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		volcengineConfig, ok := config.(*VolcengineTTSOption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for volcengine, expected *VolcengineTTSOption")
		}
		return NewVolcengineService(*volcengineConfig), nil
	})

	f.RegisterCreator(ProviderVolcengineClone, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		cloneConfig, ok := config.(*VolcengineCloneOption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for volcengine_clone, expected *VolcengineCloneOption")
		}
		return NewVolcengineCloneEngine(*cloneConfig)
	})

	// 注册Minimax
	f.RegisterCreator(ProviderMinimax, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		minimaxConfig, ok := config.(*MinimaxOption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for minimax, expected *MinimaxOption")
		}
		return NewMinimaxService(*minimaxConfig), nil
	})

	// 注册Aliyun
	f.RegisterCreator(ProviderAliyun, func(config SynthesisConfig) (AudioSynthesisEngine, error) {
		aliyunConfig, ok := config.(*AliyunTTSConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for aliyun, expected *AliyunTTSConfig")
		}
		return NewAliyunService(*aliyunConfig), nil
	})
}

// GlobalSynthesisFactory 全局TTS工厂实例
var (
	globalSynthesisFactory SynthesisFactory
	factoryMutex           sync.Mutex
)

// GetGlobalSynthesisFactory 获取全局TTS工厂实例
func GetGlobalSynthesisFactory() SynthesisFactory {
	factoryMutex.Lock()
	defer factoryMutex.Unlock()

	if globalSynthesisFactory == nil {
		globalSynthesisFactory = NewSynthesisFactory()
	}
	return globalSynthesisFactory
}

// SetGlobalSynthesisFactory 设置全局TTS工厂实例
func SetGlobalSynthesisFactory(factory SynthesisFactory) {
	factoryMutex.Lock()
	defer factoryMutex.Unlock()
	globalSynthesisFactory = factory
}
