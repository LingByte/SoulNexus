package embedder

import (
	"context"
	"sync"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// DefaultFactory 默认工厂实现
type DefaultFactory struct {
	mu        sync.RWMutex
	factories map[string]EmbedderFactory
}

// NewDefaultFactory 创建默认工厂
func NewDefaultFactory() *DefaultFactory {
	df := &DefaultFactory{
		factories: make(map[string]EmbedderFactory),
	}

	df.Register(&OpenAIFactory{})
	df.Register(&OllamaFactory{})
	df.Register(&LocalFactory{})
	df.Register(&NvidiaFactory{})
	df.Register(&DashScopeFactory{})
	df.Register(&AliyunFactory{})
	df.Register(&JinaFactory{})
	df.Register(&GeminiFactory{})
	df.Register(&VolcengineFactory{})
	df.Register(&ZhipuFactory{})
	df.Register(&AzureOpenAIFactory{})

	return df
}

// Register 注册工厂
func (f *DefaultFactory) Register(factory EmbedderFactory) error {
	if factory == nil {
		return ErrInvalidConfig
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	name := factory.Name()
	if name == "" {
		return ErrInvalidConfig
	}

	f.factories[name] = factory
	return nil
}

// Create 创建 embedder
func (f *DefaultFactory) Create(ctx context.Context, cfg *Config) (Embedder, error) {
	if cfg == nil {
		return nil, ErrInvalidConfig
	}

	if cfg.Provider != "" {
		cfg.Provider = NormalizeProvider(cfg.Provider)
	}

	return createEmbedder(ctx, cfg)
}

// List 列出所有支持的提供商
func (f *DefaultFactory) List() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	providers := make([]string, 0, len(f.factories))
	seen := make(map[string]struct{}, len(f.factories))
	for name := range f.factories {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		providers = append(providers, name)
	}
	return providers
}

// Supports 检查是否支持该提供商
func (f *DefaultFactory) Supports(provider string) bool {
	provider = NormalizeProvider(provider)
	if provider == "" {
		return false
	}
	if isKnownProvider(provider) {
		return true
	}

	f.mu.RLock()
	defer f.mu.RUnlock()
	_, exists := f.factories[provider]
	return exists
}

// OpenAIFactory OpenAI embedder 工厂
type OpenAIFactory struct{}

func (f *OpenAIFactory) Name() string { return ProviderOpenAI }
func (f *OpenAIFactory) Supports(provider string) bool {
	return NormalizeProvider(provider) == ProviderOpenAI
}
func (f *OpenAIFactory) Create(ctx context.Context, cfg *Config) (Embedder, error) {
	return createEmbedder(ctx, cfg)
}

// OllamaFactory Ollama embedder 工厂
type OllamaFactory struct{}

func (f *OllamaFactory) Name() string { return ProviderOllama }
func (f *OllamaFactory) Supports(provider string) bool {
	return NormalizeProvider(provider) == ProviderOllama
}
func (f *OllamaFactory) Create(ctx context.Context, cfg *Config) (Embedder, error) {
	return createEmbedder(ctx, cfg)
}

// LocalFactory 本地 embedder 工厂
type LocalFactory struct{}

func (f *LocalFactory) Name() string { return ProviderLocal }
func (f *LocalFactory) Supports(provider string) bool {
	return NormalizeProvider(provider) == ProviderLocal
}
func (f *LocalFactory) Create(ctx context.Context, cfg *Config) (Embedder, error) {
	return createEmbedder(ctx, cfg)
}

// GlobalFactory 全局工厂实例
var globalFactory *DefaultFactory
var factoryOnce sync.Once

// GetFactory 获取全局工厂实例
func GetFactory() *DefaultFactory {
	factoryOnce.Do(func() {
		globalFactory = NewDefaultFactory()
	})
	return globalFactory
}

// Create 使用全局工厂创建 embedder
func Create(ctx context.Context, cfg *Config) (Embedder, error) {
	return GetFactory().Create(ctx, cfg)
}

// Register 向全局工厂注册提供商
func Register(factory EmbedderFactory) error {
	return GetFactory().Register(factory)
}

// List 列出所有支持的提供商
func List() []string {
	return GetFactory().List()
}

// Supports 检查是否支持该提供商
func Supports(provider string) bool {
	return GetFactory().Supports(provider)
}

// NvidiaFactory Nvidia embedder 工厂
type NvidiaFactory struct{}

func (f *NvidiaFactory) Name() string { return ProviderNvidia }
func (f *NvidiaFactory) Supports(provider string) bool {
	return NormalizeProvider(provider) == ProviderNvidia
}
func (f *NvidiaFactory) Create(ctx context.Context, cfg *Config) (Embedder, error) {
	return createEmbedder(ctx, cfg)
}

// DashScopeFactory DashScope embedder 工厂
type DashScopeFactory struct{}

func (f *DashScopeFactory) Name() string { return ProviderDashscope }
func (f *DashScopeFactory) Supports(provider string) bool {
	p := NormalizeProvider(provider)
	return p == ProviderDashscope || p == ProviderAliyun
}
func (f *DashScopeFactory) Create(ctx context.Context, cfg *Config) (Embedder, error) {
	return createEmbedder(ctx, cfg)
}

// AliyunFactory Aliyun embedder 工厂（dashscope 别名）
type AliyunFactory struct{}

func (f *AliyunFactory) Name() string { return ProviderAliyun }
func (f *AliyunFactory) Supports(provider string) bool {
	p := NormalizeProvider(provider)
	return p == ProviderAliyun || p == ProviderDashscope
}
func (f *AliyunFactory) Create(ctx context.Context, cfg *Config) (Embedder, error) {
	return createEmbedder(ctx, cfg)
}

// JinaFactory Jina embedder 工厂
type JinaFactory struct{}

func (f *JinaFactory) Name() string { return ProviderJina }
func (f *JinaFactory) Supports(provider string) bool {
	return NormalizeProvider(provider) == ProviderJina
}
func (f *JinaFactory) Create(ctx context.Context, cfg *Config) (Embedder, error) {
	return createEmbedder(ctx, cfg)
}

// GeminiFactory Gemini embedder 工厂
type GeminiFactory struct{}

func (f *GeminiFactory) Name() string { return ProviderGemini }
func (f *GeminiFactory) Supports(provider string) bool {
	return NormalizeProvider(provider) == ProviderGemini
}
func (f *GeminiFactory) Create(ctx context.Context, cfg *Config) (Embedder, error) {
	return createEmbedder(ctx, cfg)
}

// VolcengineFactory Volcengine embedder 工厂
type VolcengineFactory struct{}

func (f *VolcengineFactory) Name() string { return ProviderVolcengine }
func (f *VolcengineFactory) Supports(provider string) bool {
	return NormalizeProvider(provider) == ProviderVolcengine
}
func (f *VolcengineFactory) Create(ctx context.Context, cfg *Config) (Embedder, error) {
	return createEmbedder(ctx, cfg)
}

// ZhipuFactory Zhipu embedder 工厂
type ZhipuFactory struct{}

func (f *ZhipuFactory) Name() string { return ProviderZhipu }
func (f *ZhipuFactory) Supports(provider string) bool {
	return NormalizeProvider(provider) == ProviderZhipu
}
func (f *ZhipuFactory) Create(ctx context.Context, cfg *Config) (Embedder, error) {
	return createEmbedder(ctx, cfg)
}

// AzureOpenAIFactory Azure OpenAI embedder 工厂
type AzureOpenAIFactory struct{}

func (f *AzureOpenAIFactory) Name() string { return ProviderAzureOpenAI }
func (f *AzureOpenAIFactory) Supports(provider string) bool {
	return NormalizeProvider(provider) == ProviderAzureOpenAI
}
func (f *AzureOpenAIFactory) Create(ctx context.Context, cfg *Config) (Embedder, error) {
	return createEmbedder(ctx, cfg)
}
