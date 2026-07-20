package embedder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestNewOllamaEmbedder(t *testing.T) {
	cfg := &Config{
		BaseURL:   "http://localhost:11434",
		Model:     "nomic-embed-text",
		Dimension: 384,
		Timeout:   30,
	}

	embedder := NewOllamaEmbedder(cfg)

	assert.NotNil(t, embedder)
	assert.Equal(t, "ollama", embedder.Name())
	assert.Equal(t, "ollama", embedder.Provider())
	assert.Equal(t, 384, embedder.Dimension())
}

func TestNewOllamaEmbedder_DefaultValues(t *testing.T) {
	cfg := &Config{
		Model: "nomic-embed-text",
	}

	embedder := NewOllamaEmbedder(cfg)

	assert.Equal(t, "http://localhost:11434", embedder.baseURL)
	assert.Equal(t, 384, embedder.Dimension())
}

func TestOllamaEmbedder_Close(t *testing.T) {
	embedder := NewOllamaEmbedder(&Config{
		Model: "nomic-embed-text",
	})

	err := embedder.Close()
	assert.Nil(t, err)
}

func TestOllamaEmbedder_EmbedSingle_Empty(t *testing.T) {
	embedder := NewOllamaEmbedder(&Config{
		Model: "nomic-embed-text",
	})

	ctx := context.Background()
	_, err := embedder.EmbedSingle(ctx, "")

	assert.Equal(t, ErrEmptyInput, err)
}

func TestOllamaEmbedder_Embed_Empty(t *testing.T) {
	embedder := NewOllamaEmbedder(&Config{
		Model: "nomic-embed-text",
	})

	ctx := context.Background()
	_, err := embedder.Embed(ctx, []string{})

	assert.Equal(t, ErrEmptyInput, err)
}

func TestOllamaEmbedder_BaseURLTrimming(t *testing.T) {
	cfg := &Config{
		BaseURL: "http://localhost:11434/",
		Model:   "nomic-embed-text",
	}

	embedder := NewOllamaEmbedder(cfg)
	assert.Equal(t, "http://localhost:11434", embedder.baseURL)
}

func TestOllamaEmbedder_DefaultTimeout(t *testing.T) {
	cfg := &Config{
		Model: "nomic-embed-text",
	}

	embedder := NewOllamaEmbedder(cfg)
	assert.NotNil(t, embedder.httpClient)
}
