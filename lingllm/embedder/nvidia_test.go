package embedder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestNewNvidiaEmbedder(t *testing.T) {
	cfg := &Config{
		BaseURL:    "https://api.nvcf.nvidia.com/v2",
		APIKey:     "nvapi-test",
		Model:      "nvidia/nv-embed-qa-4",
		Dimension:  1024,
		Timeout:    30,
		MaxRetries: 3,
	}

	embedder := NewNvidiaEmbedder(cfg)

	assert.NotNil(t, embedder)
	assert.Equal(t, "nvidia", embedder.Name())
	assert.Equal(t, "nvidia", embedder.Provider())
	assert.Equal(t, 1024, embedder.Dimension())
}

func TestNewNvidiaEmbedder_DefaultValues(t *testing.T) {
	cfg := &Config{
		APIKey: "nvapi-test",
		Model:  "nvidia/nv-embed-qa-4",
	}

	embedder := NewNvidiaEmbedder(cfg)

	assert.Equal(t, "https://api.nvcf.nvidia.com/v2", embedder.baseURL)
	assert.Equal(t, 1024, embedder.Dimension())
	assert.Equal(t, 3, embedder.maxRetries)
}

func TestNvidiaEmbedder_Close(t *testing.T) {
	embedder := NewNvidiaEmbedder(&Config{
		APIKey: "nvapi-test",
		Model:  "nvidia/nv-embed-qa-4",
	})

	err := embedder.Close()
	assert.Nil(t, err)
}

func TestNvidiaEmbedder_EmbedSingle_Empty(t *testing.T) {
	embedder := NewNvidiaEmbedder(&Config{
		APIKey: "nvapi-test",
		Model:  "nvidia/nv-embed-qa-4",
	})

	ctx := context.Background()
	_, err := embedder.EmbedSingle(ctx, "")

	assert.Equal(t, ErrEmptyInput, err)
}

func TestNvidiaEmbedder_Embed_Empty(t *testing.T) {
	embedder := NewNvidiaEmbedder(&Config{
		APIKey: "nvapi-test",
		Model:  "nvidia/nv-embed-qa-4",
	})

	ctx := context.Background()
	_, err := embedder.Embed(ctx, []string{})

	assert.Equal(t, ErrEmptyInput, err)
}

func TestNvidiaEmbedder_SanitizeInputs(t *testing.T) {
	texts := []string{"text1", "", "   ", "text2"}
	sanitized := SanitizeEmbedInputs(texts)

	assert.Equal(t, 4, len(sanitized))
	assert.Equal(t, "text1", sanitized[0])
	assert.Equal(t, " ", sanitized[1])
	assert.Equal(t, " ", sanitized[2])
	assert.Equal(t, "text2", sanitized[3])
}

func TestNvidiaEmbedder_BuildEndpoint_Default(t *testing.T) {
	embedder := NewNvidiaEmbedder(&Config{
		APIKey: "nvapi-test",
		Model:  "nvidia/nv-embed-qa-4",
	})

	endpoint := embedder.buildEndpoint()
	assert.Equal(t, "https://api.nvcf.nvidia.com/v2/embeddings", endpoint)
}

func TestNvidiaEmbedder_BuildEndpoint_Custom(t *testing.T) {
	cfg := &Config{
		BaseURL: "https://api.nvcf.nvidia.com/v2",
		APIKey:  "nvapi-test",
		Model:   "nvidia/nv-embed-qa-4",
		CustomConfig: map[string]interface{}{
			"embeddings_path": "/custom/embeddings",
		},
	}

	embedder := NewNvidiaEmbedder(cfg)
	endpoint := embedder.buildEndpoint()

	assert.Equal(t, "https://api.nvcf.nvidia.com/v2/custom/embeddings", endpoint)
}

func TestNvidiaEmbedder_CustomInputKey(t *testing.T) {
	cfg := &Config{
		APIKey: "nvapi-test",
		Model:  "nvidia/nv-embed-qa-4",
		CustomConfig: map[string]interface{}{
			"input_key": "texts",
		},
	}

	embedder := NewNvidiaEmbedder(cfg)
	assert.Equal(t, "texts", embedder.inputKey)
}

func TestNvidiaEmbedder_BaseURLTrimming(t *testing.T) {
	cfg := &Config{
		BaseURL: "https://api.nvcf.nvidia.com/v2/",
		APIKey:  "nvapi-test",
		Model:   "nvidia/nv-embed-qa-4",
	}

	embedder := NewNvidiaEmbedder(cfg)
	assert.Equal(t, "https://api.nvcf.nvidia.com/v2", embedder.baseURL)
}
