package chunk

import (
	"context"
	"fmt"
	"sync"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// DefaultFactory 默认分块工厂
type DefaultFactory struct {
	mu        sync.RWMutex
	factories map[string]ChunkerFactory
}

// NewDefaultFactory 创建默认工厂
func NewDefaultFactory() *DefaultFactory {
	df := &DefaultFactory{
		factories: make(map[string]ChunkerFactory),
	}

	// 注册内置提供商
	df.Register(&LLMChunkerFactory{})
	df.Register(&StructuredChunkerFactory{})
	df.Register(&TableKVChunkerFactory{})
	df.Register(&RouterChunkerFactory{})

	return df
}

// Register 注册工厂
func (f *DefaultFactory) Register(factory ChunkerFactory) error {
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

// Create 创建分块器
func (f *DefaultFactory) Create(ctx context.Context, cfg *Config) (Chunker, error) {
	if cfg == nil {
		return nil, ErrInvalidConfig
	}

	if cfg.Provider == "" {
		return nil, ErrProviderNotFound
	}

	f.mu.RLock()
	factory, exists := f.factories[cfg.Provider]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, cfg.Provider)
	}

	return factory.Create(ctx, cfg)
}

// List 列出所有支持的提供商
func (f *DefaultFactory) List() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	providers := make([]string, 0, len(f.factories))
	for name := range f.factories {
		providers = append(providers, name)
	}
	return providers
}

// Supports 检查是否支持该提供商
func (f *DefaultFactory) Supports(provider string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	_, exists := f.factories[provider]
	return exists
}

// LLMChunkerFactory LLM 分块工厂
type LLMChunkerFactory struct{}

func (f *LLMChunkerFactory) Name() string {
	return "llm"
}

func (f *LLMChunkerFactory) Supports(provider string) bool {
	return provider == "llm"
}

func (f *LLMChunkerFactory) Create(ctx context.Context, cfg *Config) (Chunker, error) {
	if cfg.ChatModel == nil {
		return nil, fmt.Errorf("ChatModel is required for LLM chunker")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("Model is required for LLM chunker")
	}

	return NewLLMChunker(cfg), nil
}

// StructuredChunkerFactory 结构化分块工厂
type StructuredChunkerFactory struct{}

func (f *StructuredChunkerFactory) Name() string {
	return "rules_structured"
}

func (f *StructuredChunkerFactory) Supports(provider string) bool {
	return provider == "rules_structured"
}

func (f *StructuredChunkerFactory) Create(ctx context.Context, cfg *Config) (Chunker, error) {
	return NewStructuredRuleChunker(cfg), nil
}

// TableKVChunkerFactory 表格/键值对分块工厂
type TableKVChunkerFactory struct{}

func (f *TableKVChunkerFactory) Name() string {
	return "rules_table_kv"
}

func (f *TableKVChunkerFactory) Supports(provider string) bool {
	return provider == "rules_table_kv"
}

func (f *TableKVChunkerFactory) Create(ctx context.Context, cfg *Config) (Chunker, error) {
	return NewTableKVChunker(cfg), nil
}

// RouterChunkerFactory 路由分块工厂
type RouterChunkerFactory struct{}

func (f *RouterChunkerFactory) Name() string {
	return "router"
}

func (f *RouterChunkerFactory) Supports(provider string) bool {
	return provider == "router"
}

func (f *RouterChunkerFactory) Create(ctx context.Context, cfg *Config) (Chunker, error) {
	return NewRoutingChunker(cfg), nil
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

// Create 使用全局工厂创建分块器
func Create(ctx context.Context, cfg *Config) (Chunker, error) {
	return GetFactory().Create(ctx, cfg)
}

// Register 向全局工厂注册提供商
func Register(factory ChunkerFactory) error {
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
