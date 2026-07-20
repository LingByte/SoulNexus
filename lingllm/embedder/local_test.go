package embedder

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestNewLocalEmbedder(t *testing.T) {
	cfg := &Config{
		Model:     "local",
		Dimension: 256,
	}

	embedder := NewLocalEmbedder(cfg)

	assert.NotNil(t, embedder)
	assert.Equal(t, 256, embedder.Dimension())
	assert.Equal(t, "local", embedder.Name())
	assert.Equal(t, "local", embedder.Provider())
}

func TestNewLocalEmbedder_DefaultDimension(t *testing.T) {
	cfg := &Config{
		Model: "local",
	}

	embedder := NewLocalEmbedder(cfg)

	assert.Equal(t, 384, embedder.Dimension())
}

func TestLocalEmbedder_Close(t *testing.T) {
	embedder := NewLocalEmbedder(&Config{Model: "local"})
	err := embedder.Close()
	assert.Nil(t, err)
}

func TestLocalEmbedder_EmbedSingle(t *testing.T) {
	embedder := NewLocalEmbedder(&Config{
		Model:     "local",
		Dimension: 128,
	})

	ctx := context.Background()
	vector, err := embedder.EmbedSingle(ctx, "test text")

	assert.Nil(t, err)
	assert.Equal(t, 128, len(vector))
	assert.True(t, isUnitVector(vector))
}

func TestLocalEmbedder_EmbedSingle_Empty(t *testing.T) {
	embedder := NewLocalEmbedder(&Config{Model: "local"})

	ctx := context.Background()
	_, err := embedder.EmbedSingle(ctx, "")

	assert.Equal(t, ErrEmptyInput, err)
}

func TestLocalEmbedder_EmbedSingle_Whitespace(t *testing.T) {
	embedder := NewLocalEmbedder(&Config{Model: "local"})

	ctx := context.Background()
	_, err := embedder.EmbedSingle(ctx, "   \n\t  ")

	assert.Equal(t, ErrEmptyInput, err)
}

func TestLocalEmbedder_Embed(t *testing.T) {
	embedder := NewLocalEmbedder(&Config{
		Model:     "local",
		Dimension: 128,
	})

	ctx := context.Background()
	texts := []string{"text1", "text2", "text3"}
	vectors, err := embedder.Embed(ctx, texts)

	assert.Nil(t, err)
	assert.Equal(t, 3, len(vectors))
	for _, vec := range vectors {
		assert.Equal(t, 128, len(vec))
		assert.True(t, isUnitVector(vec))
	}
}

func TestLocalEmbedder_Embed_Empty(t *testing.T) {
	embedder := NewLocalEmbedder(&Config{Model: "local"})

	ctx := context.Background()
	_, err := embedder.Embed(ctx, []string{})

	assert.Equal(t, ErrEmptyInput, err)
}

func TestLocalEmbedder_Embed_WithEmpty(t *testing.T) {
	embedder := NewLocalEmbedder(&Config{
		Model:     "local",
		Dimension: 128,
	})

	ctx := context.Background()
	texts := []string{"text1", "", "text3"}
	vectors, err := embedder.Embed(ctx, texts)

	assert.Nil(t, err)
	assert.Equal(t, 3, len(vectors))
	// Empty text should be converted to space
	assert.True(t, isUnitVector(vectors[1]))
}

func TestLocalEmbedder_Deterministic(t *testing.T) {
	embedder := NewLocalEmbedder(&Config{
		Model:     "local",
		Dimension: 128,
	})

	ctx := context.Background()
	text := "same text"

	vec1, _ := embedder.EmbedSingle(ctx, text)
	vec2, _ := embedder.EmbedSingle(ctx, text)

	// Same text should produce same vector
	for i := range vec1 {
		assert.Equal(t, vec1[i], vec2[i])
	}
}

func TestLocalEmbedder_Different(t *testing.T) {
	embedder := NewLocalEmbedder(&Config{
		Model:     "local",
		Dimension: 128,
	})

	ctx := context.Background()
	vec1, _ := embedder.EmbedSingle(ctx, "text1")
	vec2, _ := embedder.EmbedSingle(ctx, "text2")

	// Different texts should produce different vectors
	different := false
	for i := range vec1 {
		if vec1[i] != vec2[i] {
			different = true
			break
		}
	}
	assert.True(t, different)
}

func TestLocalEmbedder_Normalization(t *testing.T) {
	embedder := NewLocalEmbedder(&Config{
		Model:     "local",
		Dimension: 256,
	})

	ctx := context.Background()
	vector, _ := embedder.EmbedSingle(ctx, "test")

	// Calculate L2 norm
	var norm float32
	for _, v := range vector {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))

	// Should be approximately 1.0 (unit vector)
	assert.InDelta(t, 1.0, norm, 0.0001)
}

func TestLocalEmbedder_DifferentDimensions(t *testing.T) {
	dimensions := []int{64, 128, 256, 512}

	ctx := context.Background()
	text := "test text"

	for _, dim := range dimensions {
		embedder := NewLocalEmbedder(&Config{
			Model:     "local",
			Dimension: dim,
		})

		vector, err := embedder.EmbedSingle(ctx, text)

		assert.Nil(t, err)
		assert.Equal(t, dim, len(vector))
		assert.True(t, isUnitVector(vector))
	}
}

// Helper function to check if vector is unit vector
func isUnitVector(v []float32) bool {
	var norm float32
	for _, val := range v {
		norm += val * val
	}
	norm = float32(math.Sqrt(float64(norm)))
	return math.Abs(float64(norm-1.0)) < 0.0001
}
