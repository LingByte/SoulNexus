package recognizer

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// Vendor 供应商类型
type Vendor string

const (
	// VendorQCloud 腾讯云
	VendorQCloud Vendor = "qcloud"
	// VendorGoogle Google
	VendorGoogle Vendor = "google"
	// VendorAliyun 阿里云
	VendorAliyun Vendor = "aliyun"
	// VendorQiniu 七牛云
	VendorQiniu Vendor = "qiniu"
	// VendorFunASR FunASR
	VendorFunASR Vendor = "funasr"
	// VendorVolcengine 火山引擎
	VendorVolcengine Vendor = "volcengine"
	// VendorVolcengineLLM 火山引擎LLM
	VendorVolcengineLLM Vendor = "volcllmasr"
	// VendorXfyunMul 科大讯飞多语言
	// 注意：Xfyun (非Mul版本) 未实现 TranscribeService 接口，只有 WithXfyunASR 函数
	VendorXfyunMul Vendor = "xfyun_mul"
	// VendorGladia Gladia
	VendorGladia Vendor = "gladia"
	// VendorFunASRRealtime FunASR实时
	VendorFunASRRealtime Vendor = "funasr_realtime"
	// VendorWhisper Whisper
	VendorWhisper Vendor = "whisper"
	// VendorDeepgram Deepgram
	VendorDeepgram Vendor = "deepgram"
	// VendorAWS AWS
	VendorAWS Vendor = "aws"
	// VendorBaidu 百度
	VendorBaidu Vendor = "baidu"
	// VendorVoiceAPI VoiceAPI
	VendorVoiceAPI Vendor = "voiceapi"
)

// TranscriberConfig 统一的配置接口
type TranscriberConfig interface {
	GetVendor() Vendor
}

// TranscriberFactory 工厂接口
type TranscriberFactory interface {
	// CreateTranscriber 根据配置创建 TranscribeService
	CreateTranscriber(config TranscriberConfig) (TranscribeService, error)
	// GetSupportedVendors 获取支持的供应商列表
	GetSupportedVendors() []Vendor
	// IsVendorSupported 检查供应商是否支持
	IsVendorSupported(vendor Vendor) bool
}

// DefaultTranscriberFactory 默认工厂实现
type DefaultTranscriberFactory struct {
	creators map[Vendor]func(TranscriberConfig) (TranscribeService, error)
	mu       sync.RWMutex
}

// NewTranscriberFactory 创建新的工厂实例
func NewTranscriberFactory() *DefaultTranscriberFactory {
	factory := &DefaultTranscriberFactory{
		creators: make(map[Vendor]func(TranscriberConfig) (TranscribeService, error)),
	}
	factory.registerDefaultCreators()
	return factory
}

// RegisterCreator 注册创建函数
func (f *DefaultTranscriberFactory) RegisterCreator(vendor Vendor, creator func(TranscriberConfig) (TranscribeService, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creators[vendor] = creator
}

// CreateTranscriber 创建 TranscribeService
func (f *DefaultTranscriberFactory) CreateTranscriber(config TranscriberConfig) (TranscribeService, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	vendor := config.GetVendor()
	f.mu.RLock()
	creator, exists := f.creators[vendor]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("vendor %s not supported", vendor)
	}

	return creator(config)
}

// GetSupportedVendors 获取支持的供应商列表
func (f *DefaultTranscriberFactory) GetSupportedVendors() []Vendor {
	f.mu.RLock()
	defer f.mu.RUnlock()

	vendors := make([]Vendor, 0, len(f.creators))
	for vendor := range f.creators {
		vendors = append(vendors, vendor)
	}
	return vendors
}

// IsVendorSupported 检查供应商是否支持
func (f *DefaultTranscriberFactory) IsVendorSupported(vendor Vendor) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, exists := f.creators[vendor]
	return exists
}

// registerDefaultCreators 注册默认创建函数
func (f *DefaultTranscriberFactory) registerDefaultCreators() {
	// 注册腾讯云
	f.RegisterCreator(VendorQCloud, func(config TranscriberConfig) (TranscribeService, error) {
		qcloudConfig, ok := config.(*QCloudASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for qcloud")
		}
		return NewQcloudASR(*qcloudConfig), nil
	})

	// 注册Google
	f.RegisterCreator(VendorGoogle, func(config TranscriberConfig) (TranscribeService, error) {
		googleConfig, ok := config.(*GoogleASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for google")
		}
		asr := NewGoogleASR(*googleConfig)
		return &asr, nil
	})

	// 注册七牛云
	f.RegisterCreator(VendorQiniu, func(config TranscriberConfig) (TranscribeService, error) {
		qiniuConfig, ok := config.(*QiniuASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for qiniu")
		}
		return NewQiniuASR(*qiniuConfig), nil
	})

	// 注册FunASR
	f.RegisterCreator(VendorFunASR, func(config TranscriberConfig) (TranscribeService, error) {
		funasrConfig, ok := config.(*FunASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for funasr")
		}
		callback := NewFunASR(*funasrConfig)
		return &callback, nil
	})

	// 注册火山引擎LLM
	f.RegisterCreator(VendorVolcengineLLM, func(config TranscriberConfig) (TranscribeService, error) {
		volcengineLLMConfig, ok := config.(*VolcengineLLMOption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for volcengine_llm")
		}
		asr := NewVolcengineLLM(*volcengineLLMConfig)
		return &asr, nil
	})

	// 注册科大讯飞多语言
	// 注意：XfyunMul 需要 pipeline.SessionHandler 参数，需要通过 NewXfyunMul 创建
	// 如果需要在工厂中创建，需要特殊处理
	f.RegisterCreator(VendorXfyunMul, func(config TranscriberConfig) (TranscribeService, error) {
		return nil, fmt.Errorf("xfyun_mul requires pipeline.SessionHandler, use NewXfyunMul directly")
	})

	// 注册Gladia
	f.RegisterCreator(VendorGladia, func(config TranscriberConfig) (TranscribeService, error) {
		gladiaConfig, ok := config.(*GladiaASROption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for gladia")
		}
		asr := NewGladiaASR(*gladiaConfig)
		return &asr, nil
	})

	// 注册FunASR实时
	f.RegisterCreator(VendorFunASRRealtime, func(config TranscriberConfig) (TranscribeService, error) {
		funasrRealtimeConfig, ok := config.(*FunAsrRealtimeOption)
		if !ok {
			return nil, fmt.Errorf("invalid config type for funasr_realtime")
		}
		realtime := NewFunAsrRealtime(*funasrRealtimeConfig)
		return &realtime, nil
	})

	// 注意：以下 transcriber 没有实现 TranscribeService 接口，只有 With 函数
	// Whisper, Deepgram, AWS, Baidu, VoiceAPI 等需要通过 With*ASR 函数使用
	// 如果需要使用这些，请直接调用相应的 With*ASR 函数

	logrus.WithFields(logrus.Fields{
		"vendors": f.GetSupportedVendors(),
	}).Info("transcriber factory initialized")
}

// 为各个配置类型实现 GetVendor 方法

func (opt *QCloudASROption) GetVendor() Vendor {
	return VendorQCloud
}

func (opt *GoogleASROption) GetVendor() Vendor {
	return VendorGoogle
}

func (opt *QiniuASROption) GetVendor() Vendor {
	return VendorQiniu
}

func (opt *FunASROption) GetVendor() Vendor {
	return VendorFunASR
}

func (opt *VolcengineOption) GetVendor() Vendor {
	return VendorVolcengine
}

func (opt *VolcengineLLMOption) GetVendor() Vendor {
	return VendorVolcengineLLM
}

func (opt *GladiaASROption) GetVendor() Vendor {
	return VendorGladia
}

func (opt *FunAsrRealtimeOption) GetVendor() Vendor {
	return VendorFunASRRealtime
}

// 全局工厂实例
var (
	globalFactory *DefaultTranscriberFactory
	factoryOnce   sync.Once
)

// GetGlobalFactory 获取全局工厂实例（单例模式）
func GetGlobalFactory() *DefaultTranscriberFactory {
	factoryOnce.Do(func() {
		globalFactory = NewTranscriberFactory()
	})
	return globalFactory
}
