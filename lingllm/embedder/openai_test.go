package embedder

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestNewOpenAIEmbedder(t *testing.T) {
	cfg := &Config{
		BaseURL:   "https://api.openai.com/v1",
		APIKey:    "sk-test",
		Model:     "text-embedding-3-small",
		Dimension: 1536,
		Timeout:   30,
	}

	embedder := NewOpenAIEmbedder(cfg)

	assert.NotNil(t, embedder)
	assert.Equal(t, "openai", embedder.Name())
	assert.Equal(t, "openai", embedder.Provider())
	assert.Equal(t, 1536, embedder.Dimension())
}

func TestNewOpenAIEmbedder_DefaultValues(t *testing.T) {
	cfg := &Config{
		APIKey: "sk-test",
		Model:  "text-embedding-3-small",
	}

	embedder := NewOpenAIEmbedder(cfg)

	assert.Equal(t, "https://api.openai.com/v1", embedder.baseURL)
	assert.Equal(t, 1536, embedder.Dimension())
}

func TestOpenAIEmbedder_Close(t *testing.T) {
	embedder := NewOpenAIEmbedder(&Config{
		APIKey: "sk-test",
		Model:  "text-embedding-3-small",
	})

	err := embedder.Close()
	assert.Nil(t, err)
}

func TestOpenAIEmbedder_EmbedSingle_Empty(t *testing.T) {
	embedder := NewOpenAIEmbedder(&Config{
		APIKey: "sk-test",
		Model:  "text-embedding-3-small",
	})

	ctx := context.Background()
	_, err := embedder.EmbedSingle(ctx, "")

	assert.Equal(t, ErrEmptyInput, err)
}

func TestOpenAIEmbedder_Embed_Empty(t *testing.T) {
	embedder := NewOpenAIEmbedder(&Config{
		APIKey: "sk-test",
		Model:  "text-embedding-3-small",
	})

	ctx := context.Background()
	_, err := embedder.Embed(ctx, []string{})

	assert.Equal(t, ErrEmptyInput, err)
}

func TestOpenAIEmbedder_BaseURLTrimming(t *testing.T) {
	cfg := &Config{
		BaseURL: "https://api.openai.com/v1/",
		APIKey:  "sk-test",
		Model:   "text-embedding-3-small",
	}

	embedder := NewOpenAIEmbedder(cfg)
	assert.Equal(t, "https://api.openai.com/v1", embedder.baseURL)
}

func TestOpenAIEmbedder_DefaultTimeout(t *testing.T) {
	cfg := &Config{
		APIKey: "sk-test",
		Model:  "text-embedding-3-small",
	}

	embedder := NewOpenAIEmbedder(cfg)
	assert.NotNil(t, embedder.httpClient)
}

func TestOpenAIEmbedder_StringInputAutoDetect(t *testing.T) {
	embedder := NewOpenAIEmbedder(&Config{
		BaseURL: "https://ai.gitee.com/v1",
		APIKey:  "sk-test",
		Model:   "text-embedding-v3",
	})
	assert.True(t, embedder.stringInput)
}

func TestOpenAIEmbedder_Embed_StringInputRequestBody(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(body, &payload))

		var input string
		require.NoError(t, json.Unmarshal(payload["input"], &input))
		assert.Contains(t, []string{"hello", "world"}, input)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2],"index":0}]}`))
	}))
	t.Cleanup(server.Close)

	embedder := NewOpenAIEmbedder(&Config{
		BaseURL:     server.URL,
		APIKey:      "sk-test",
		Model:       "text-embedding-v3",
		StringInput: true,
		MaxRetries:  0,
	})

	vectors, err := embedder.Embed(context.Background(), []string{"hello", "world"})
	require.NoError(t, err)
	assert.Len(t, vectors, 2)
	assert.Equal(t, int32(2), atomic.LoadInt32(&requestCount))
}

func TestOpenAIEmbedder_Embed_ArrayInputRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var payload map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(body, &payload))

		var input []string
		require.NoError(t, json.Unmarshal(payload["input"], &input))
		assert.Equal(t, []string{"hello", "world"}, input)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2],"index":0},{"embedding":[0.3,0.4],"index":1}]}`))
	}))
	t.Cleanup(server.Close)

	embedder := NewOpenAIEmbedder(&Config{
		BaseURL:    server.URL,
		APIKey:     "sk-test",
		Model:      "text-embedding-3-small",
		MaxRetries: 0,
	})

	vectors, err := embedder.Embed(context.Background(), []string{"hello", "world"})
	require.NoError(t, err)
	assert.Len(t, vectors, 2)
}
