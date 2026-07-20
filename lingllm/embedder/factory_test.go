package embedder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestNewDefaultFactory(t *testing.T) {
	factory := NewDefaultFactory()

	assert.NotNil(t, factory)
	assert.True(t, factory.Supports("openai"))
	assert.True(t, factory.Supports("ollama"))
	assert.True(t, factory.Supports("local"))
	assert.True(t, factory.Supports("nvidia"))
	assert.True(t, factory.Supports("dashscope"))
	assert.True(t, factory.Supports("aliyun"))
	assert.True(t, factory.Supports("jina"))
	assert.True(t, factory.Supports("gemini"))
	assert.True(t, factory.Supports("volcengine"))
	assert.True(t, factory.Supports("zhipu"))
	assert.True(t, factory.Supports("azure_openai"))
}

func TestDefaultFactory_Register(t *testing.T) {
	factory := NewDefaultFactory()

	customFactory := &LocalFactory{}
	err := factory.Register(customFactory)

	assert.Nil(t, err)
	assert.True(t, factory.Supports("local"))
}

func TestDefaultFactory_RegisterNil(t *testing.T) {
	factory := NewDefaultFactory()

	err := factory.Register(nil)

	assert.Equal(t, ErrInvalidConfig, err)
}

func TestDefaultFactory_Create_OpenAI(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "openai",
		APIKey:   "sk-test",
		Model:    "text-embedding-3-small",
	}

	embedder, err := factory.Create(context.Background(), cfg)

	assert.Nil(t, err)
	assert.NotNil(t, embedder)
	assert.Equal(t, "openai", embedder.Provider())
}

func TestDefaultFactory_Create_Ollama(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434",
		Model:    "nomic-embed-text",
	}

	embedder, err := factory.Create(context.Background(), cfg)

	assert.Nil(t, err)
	assert.NotNil(t, embedder)
	assert.Equal(t, "ollama", embedder.Provider())
}

func TestDefaultFactory_Create_Local(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "local",
		Model:    "local",
	}

	embedder, err := factory.Create(context.Background(), cfg)

	assert.Nil(t, err)
	assert.NotNil(t, embedder)
	assert.Equal(t, "local", embedder.Provider())
}

func TestDefaultFactory_Create_Nvidia(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "nvidia",
		APIKey:   "nvapi-test",
		Model:    "nvidia/nv-embed-qa-4",
	}

	embedder, err := factory.Create(context.Background(), cfg)

	assert.Nil(t, err)
	assert.NotNil(t, embedder)
	assert.Equal(t, "nvidia", embedder.Provider())
}

func TestDefaultFactory_Create_InvalidConfig(t *testing.T) {
	factory := NewDefaultFactory()

	_, err := factory.Create(context.Background(), nil)

	assert.Equal(t, ErrInvalidConfig, err)
}

func TestDefaultFactory_Create_NoProvider(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Model: "test",
	}

	_, err := factory.Create(context.Background(), cfg)

	assert.Equal(t, ErrProviderNotFound, err)
}

func TestDefaultFactory_Create_UnknownProvider(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "unknown",
		Model:    "test",
	}

	_, err := factory.Create(context.Background(), cfg)

	assert.NotNil(t, err)
}

func TestDefaultFactory_List(t *testing.T) {
	factory := NewDefaultFactory()

	providers := factory.List()

	assert.Equal(t, 11, len(providers))
	assert.Contains(t, providers, "openai")
	assert.Contains(t, providers, "ollama")
	assert.Contains(t, providers, "local")
	assert.Contains(t, providers, "nvidia")
	assert.Contains(t, providers, "dashscope")
	assert.Contains(t, providers, "aliyun")
	assert.Contains(t, providers, "jina")
	assert.Contains(t, providers, "gemini")
	assert.Contains(t, providers, "volcengine")
	assert.Contains(t, providers, "zhipu")
	assert.Contains(t, providers, "azure_openai")
}

func TestDefaultFactory_Supports(t *testing.T) {
	factory := NewDefaultFactory()

	assert.True(t, factory.Supports("openai"))
	assert.True(t, factory.Supports("ollama"))
	assert.True(t, factory.Supports("local"))
	assert.True(t, factory.Supports("nvidia"))
	assert.False(t, factory.Supports("unknown"))
}

func TestOpenAIFactory_Create_NoAPIKey(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "openai",
		Model:    "text-embedding-3-small",
	}

	_, err := factory.Create(context.Background(), cfg)

	assert.Equal(t, ErrAPIKeyRequired, err)
}

func TestOpenAIFactory_Create_NoModel(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "openai",
		APIKey:   "sk-test",
	}

	_, err := factory.Create(context.Background(), cfg)

	assert.Equal(t, ErrModelNotFound, err)
}

func TestOllamaFactory_Create_WithDefaultBaseURL(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "ollama",
		Model:    "nomic-embed-text",
	}

	// Ollama has default BaseURL, so this should succeed
	embedder, err := factory.Create(context.Background(), cfg)

	assert.Nil(t, err)
	assert.NotNil(t, embedder)
}

func TestOllamaFactory_Create_NoModel(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "ollama",
		BaseURL:  "http://localhost:11434",
	}

	_, err := factory.Create(context.Background(), cfg)

	assert.Equal(t, ErrModelNotFound, err)
}

func TestLocalFactory_Create_NoModel(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "local",
	}

	_, err := factory.Create(context.Background(), cfg)

	assert.Equal(t, ErrModelNotFound, err)
}

func TestNvidiaFactory_Create_NoAPIKey(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "nvidia",
		Model:    "nvidia/nv-embed-qa-4",
	}

	_, err := factory.Create(context.Background(), cfg)

	assert.Equal(t, ErrAPIKeyRequired, err)
}

func TestNvidiaFactory_Create_NoModel(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "nvidia",
		APIKey:   "nvapi-test",
	}

	_, err := factory.Create(context.Background(), cfg)

	assert.Equal(t, ErrModelNotFound, err)
}

func TestGetFactory(t *testing.T) {
	factory1 := GetFactory()
	factory2 := GetFactory()

	assert.Equal(t, factory1, factory2)
}

func TestGlobalCreate(t *testing.T) {
	cfg := &Config{
		Provider: "local",
		Model:    "local",
	}

	embedder, err := Create(context.Background(), cfg)

	assert.Nil(t, err)
	assert.NotNil(t, embedder)
}

func TestGlobalRegister(t *testing.T) {
	factory := &LocalFactory{}
	err := Register(factory)

	assert.Nil(t, err)
}

func TestGlobalList(t *testing.T) {
	providers := List()

	assert.True(t, len(providers) > 0)
	assert.Contains(t, providers, "local")
}

func TestGlobalSupports(t *testing.T) {
	assert.True(t, Supports("local"))
	assert.False(t, Supports("unknown"))
}

func TestDefaultFactory_Create_DashScope(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "dashscope",
		APIKey:   "sk-test",
		Model:    "text-embedding-v4",
	}

	embedder, err := factory.Create(context.Background(), cfg)

	assert.Nil(t, err)
	assert.NotNil(t, embedder)
	assert.Equal(t, "dashscope", embedder.Provider())
}

func TestDashScopeFactory_Create_NoAPIKey(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "dashscope",
		Model:    "text-embedding-v4",
	}

	_, err := factory.Create(context.Background(), cfg)

	assert.Equal(t, ErrAPIKeyRequired, err)
}

func TestDashScopeFactory_Create_NoModel(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "dashscope",
		APIKey:   "sk-test",
	}

	_, err := factory.Create(context.Background(), cfg)

	assert.Equal(t, ErrModelNotFound, err)
}
