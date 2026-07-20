package embedder

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestConfig_Structure(t *testing.T) {
	cfg := &Config{
		Provider:     "openai",
		Model:        "text-embedding-3-small",
		BaseURL:      "https://api.openai.com/v1",
		APIKey:       "sk-test",
		Dimension:    1536,
		BatchSize:    10,
		Timeout:      30,
		MaxRetries:   3,
		CustomConfig: map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "openai", cfg.Provider)
	assert.Equal(t, "text-embedding-3-small", cfg.Model)
	assert.Equal(t, 1536, cfg.Dimension)
	assert.Equal(t, 30, cfg.Timeout)
}

func TestEmbedResult_Structure(t *testing.T) {
	result := &EmbedResult{
		Text:      "test",
		Vector:    []float32{0.1, 0.2, 0.3},
		Dimension: 3,
		Error:     nil,
	}

	assert.Equal(t, "test", result.Text)
	assert.Equal(t, 3, len(result.Vector))
	assert.Nil(t, result.Error)
}

func TestBatchEmbedResult_Structure(t *testing.T) {
	results := []EmbedResult{
		{Text: "test1", Vector: []float32{0.1, 0.2}},
		{Text: "test2", Vector: []float32{0.3, 0.4}},
	}

	batchResult := &BatchEmbedResult{
		Results:   results,
		Dimension: 2,
		Duration:  100,
	}

	assert.Equal(t, 2, len(batchResult.Results))
	assert.Equal(t, 2, batchResult.Dimension)
	assert.Equal(t, int64(100), batchResult.Duration)
}

func TestEmbedderOptions_Structure(t *testing.T) {
	opts := &EmbedderOptions{
		BatchSize:    10,
		Normalize:    true,
		ReturnTokens: true,
		Timeout:      30,
	}

	assert.Equal(t, 10, opts.BatchSize)
	assert.True(t, opts.Normalize)
	assert.True(t, opts.ReturnTokens)
}

func TestHealthCheckResult_Structure(t *testing.T) {
	result := &HealthCheckResult{
		Healthy:  true,
		Message:  "OK",
		Latency:  50,
		LastErr:  nil,
		Provider: "openai",
		Model:    "text-embedding-3-small",
	}

	assert.True(t, result.Healthy)
	assert.Equal(t, "OK", result.Message)
	assert.Equal(t, int64(50), result.Latency)
}

func TestErrorTypes(t *testing.T) {
	assert.NotNil(t, ErrEmptyInput)
	assert.NotNil(t, ErrInvalidDimension)
	assert.NotNil(t, ErrInvalidConfig)
	assert.NotNil(t, ErrProviderNotFound)
	assert.NotNil(t, ErrModelNotFound)
	assert.NotNil(t, ErrAPIKeyRequired)
	assert.NotNil(t, ErrBaseURLRequired)
	assert.NotNil(t, ErrEmbedFailed)
	assert.NotNil(t, ErrConnectionFailed)
	assert.NotNil(t, ErrRateLimited)
	assert.NotNil(t, ErrInvalidResponse)
}
