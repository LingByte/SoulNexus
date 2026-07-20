package embedder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestNewDashScopeEmbedder(t *testing.T) {
	cfg := &Config{
		BaseURL:   "https://dashscope.aliyuncs.com/api/v1",
		APIKey:    "sk-test",
		Model:     "text-embedding-v4",
		Dimension: 1024,
		Timeout:   30,
	}

	embedder := NewDashScopeEmbedder(cfg)

	assert.NotNil(t, embedder)
	assert.Equal(t, "dashscope", embedder.Name())
	assert.Equal(t, "dashscope", embedder.Provider())
	assert.Equal(t, 1024, embedder.Dimension())
}

func TestNewDashScopeEmbedder_DefaultValues(t *testing.T) {
	cfg := &Config{
		APIKey: "sk-test",
		Model:  "text-embedding-v4",
	}

	embedder := NewDashScopeEmbedder(cfg)

	assert.Equal(t, "https://dashscope.aliyuncs.com/api/v1", embedder.baseURL)
	assert.Equal(t, 1024, embedder.Dimension())
}

func TestDashScopeEmbedder_Close(t *testing.T) {
	embedder := NewDashScopeEmbedder(&Config{
		APIKey: "sk-test",
		Model:  "text-embedding-v4",
	})

	err := embedder.Close()
	assert.Nil(t, err)
}

func TestDashScopeEmbedder_EmbedSingle_Empty(t *testing.T) {
	embedder := NewDashScopeEmbedder(&Config{
		APIKey: "sk-test",
		Model:  "text-embedding-v4",
	})

	ctx := context.Background()
	_, err := embedder.EmbedSingle(ctx, "")

	assert.Equal(t, ErrEmptyInput, err)
}

func TestDashScopeEmbedder_Embed_Empty(t *testing.T) {
	embedder := NewDashScopeEmbedder(&Config{
		APIKey: "sk-test",
		Model:  "text-embedding-v4",
	})

	ctx := context.Background()
	_, err := embedder.Embed(ctx, []string{})

	assert.Equal(t, ErrEmptyInput, err)
}

func TestDashScopeEmbedder_BaseURLTrimming(t *testing.T) {
	cfg := &Config{
		BaseURL: "https://dashscope.aliyuncs.com/api/v1/",
		APIKey:  "sk-test",
		Model:   "text-embedding-v4",
	}

	embedder := NewDashScopeEmbedder(cfg)
	assert.Equal(t, "https://dashscope.aliyuncs.com/api/v1", embedder.baseURL)
}

func TestDashScopeEmbedder_DefaultTimeout(t *testing.T) {
	cfg := &Config{
		APIKey: "sk-test",
		Model:  "text-embedding-v4",
	}

	embedder := NewDashScopeEmbedder(cfg)
	assert.NotNil(t, embedder.httpClient)
}

func TestDashScopeEmbedder_DifferentDimensions(t *testing.T) {
	dimensions := []int{64, 128, 256, 512, 768, 1024, 1536, 2048}

	for _, dim := range dimensions {
		cfg := &Config{
			APIKey:    "sk-test",
			Model:     "text-embedding-v4",
			Dimension: dim,
		}

		embedder := NewDashScopeEmbedder(cfg)
		assert.Equal(t, dim, embedder.Dimension())
	}
}

func TestDashScopeEmbedder_Models(t *testing.T) {
	models := []string{"text-embedding-v4", "text-embedding-v3"}

	for _, model := range models {
		cfg := &Config{
			APIKey: "sk-test",
			Model:  model,
		}

		embedder := NewDashScopeEmbedder(cfg)
		assert.Equal(t, model, embedder.model)
	}
}
