package knowledge

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNvidiaEmbeddingProvider_RealAPI tests the NVIDIA embedding provider with real API calls
func TestNvidiaEmbeddingProvider_RealAPI(t *testing.T) {
	// Load environment variables
	apiKey := utils.GetEnv("EMBEDDING_API_KEY")
	if apiKey == "" {
		t.Skip("EMBEDDING_API_KEY not set, skipping real API test")
	}

	cfg := config.EmbeddingConfig{
		Provider:  "nvidia",
		BaseURL:   "https://integrate.api.nvidia.com/v1",
		APIKey:    apiKey,
		Model:     "nvidia/nv-embed-v1",
		Dimension: 4096,
	}

	provider := NewNvidiaEmbeddingProvider(cfg)
	require.NotNil(t, provider)

	t.Run("GenerateSingleEmbedding", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		text := "什么是人工智能？"
		embedding, err := provider.GenerateEmbedding(ctx, text)

		require.NoError(t, err, "Failed to generate embedding")
		assert.NotNil(t, embedding)
		assert.Equal(t, 4096, len(embedding), "Embedding dimension should be 4096")

		// Check that embedding is not all zeros
		hasNonZero := false
		for _, v := range embedding {
			if v != 0 {
				hasNonZero = true
				break
			}
		}
		assert.True(t, hasNonZero, "Embedding should contain non-zero values")

		t.Logf("✅ Generated embedding for text: '%s'", text)
		t.Logf("   Dimension: %d", len(embedding))
		t.Logf("   First 5 values: %v", embedding[:5])
	})

	t.Run("GenerateMultipleEmbeddings", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		texts := []string{
			"人工智能是什么？",
			"机器学习的基本原理",
			"深度学习神经网络",
		}

		embeddings, err := provider.GenerateEmbeddings(ctx, texts)

		require.NoError(t, err, "Failed to generate embeddings")
		assert.NotNil(t, embeddings)
		assert.Equal(t, len(texts), len(embeddings), "Should generate embeddings for all texts")

		for i, embedding := range embeddings {
			assert.Equal(t, 4096, len(embedding), "Each embedding should have dimension 4096")
			t.Logf("✅ Generated embedding %d for text: '%s'", i+1, texts[i])
			t.Logf("   First 5 values: %v", embedding[:5])
		}
	})

	t.Run("SemanticSimilarity", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Similar texts
		text1 := "人工智能技术"
		text2 := "AI技术"
		text3 := "今天天气很好"

		embeddings, err := provider.GenerateEmbeddings(ctx, []string{text1, text2, text3})
		require.NoError(t, err)

		// Calculate cosine similarity
		sim12 := cosineSimilarity(embeddings[0], embeddings[1])
		sim13 := cosineSimilarity(embeddings[0], embeddings[2])

		t.Logf("✅ Semantic similarity test:")
		t.Logf("   '%s' vs '%s': %.4f", text1, text2, sim12)
		t.Logf("   '%s' vs '%s': %.4f", text1, text3, sim13)

		// Similar texts should have higher similarity
		assert.Greater(t, sim12, sim13, "Similar texts should have higher similarity score")
	})

	t.Run("ChineseAndEnglish", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		texts := []string{
			"人工智能",
			"Artificial Intelligence",
			"机器学习",
			"Machine Learning",
		}

		embeddings, err := provider.GenerateEmbeddings(ctx, texts)
		require.NoError(t, err)

		// Calculate similarities
		simCN := cosineSimilarity(embeddings[0], embeddings[2])        // 人工智能 vs 机器学习
		simEN := cosineSimilarity(embeddings[1], embeddings[3])        // AI vs ML
		simCrossLang := cosineSimilarity(embeddings[0], embeddings[1]) // 人工智能 vs AI

		t.Logf("✅ Cross-language embedding test:")
		t.Logf("   Chinese similarity: %.4f", simCN)
		t.Logf("   English similarity: %.4f", simEN)
		t.Logf("   Cross-language similarity: %.4f", simCrossLang)

		// Cross-language similarity should be high for same concept
		assert.Greater(t, simCrossLang, 0.7, "Cross-language similarity should be high")
	})

	t.Run("EmptyText", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := provider.GenerateEmbedding(ctx, "")
		// API might handle empty text differently, just check it doesn't panic
		t.Logf("Empty text result: %v", err)
	})

	t.Run("LongText", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		longText := `人工智能（Artificial Intelligence，AI）是计算机科学的一个分支，
它企图了解智能的实质，并生产出一种新的能以人类智能相似的方式做出反应的智能机器。
该领域的研究包括机器人、语言识别、图像识别、自然语言处理和专家系统等。
人工智能从诞生以来，理论和技术日益成熟，应用领域也不断扩大。
可以设想，未来人工智能带来的科技产品，将会是人类智慧的"容器"。`

		embedding, err := provider.GenerateEmbedding(ctx, longText)
		require.NoError(t, err)
		assert.Equal(t, 4096, len(embedding))
		t.Logf("✅ Generated embedding for long text (%d chars)", len(longText))
	})
}

// TestGenerateEmbedding tests the wrapper function
func TestGenerateEmbedding(t *testing.T) {
	// Load config
	err := config.Load()
	require.NoError(t, err)

	apiKey := os.Getenv("EMBEDDING_API_KEY")
	if apiKey == "" {
		t.Skip("EMBEDDING_API_KEY not set, skipping test")
	}

	t.Run("WithConfiguredProvider", func(t *testing.T) {
		text := "测试文本向量化"
		embedding := GenerateEmbedding(text, 4096)

		assert.NotNil(t, embedding)
		assert.Equal(t, 4096, len(embedding))

		t.Logf("✅ Generated embedding via wrapper function")
		t.Logf("   Dimension: %d", len(embedding))
		t.Logf("   First 5 values: %v", embedding[:5])
	})

	t.Run("FallbackToSimpleEmbedding", func(t *testing.T) {
		// Temporarily clear API key to trigger fallback
		originalKey := config.GlobalConfig.Services.KnowledgeBase.Embedding.APIKey
		config.GlobalConfig.Services.KnowledgeBase.Embedding.APIKey = ""

		text := "测试降级方案"
		embedding := GenerateEmbedding(text, 384)

		assert.NotNil(t, embedding)
		assert.Equal(t, 384, len(embedding))

		t.Logf("✅ Fallback to simple embedding works")
		t.Logf("   Dimension: %d", len(embedding))

		// Restore API key
		config.GlobalConfig.Services.KnowledgeBase.Embedding.APIKey = originalKey
	})
}

// TestEmbeddingProvider_Dimension tests the Dimension method
func TestEmbeddingProvider_Dimension(t *testing.T) {
	cfg := config.EmbeddingConfig{
		Provider:  "nvidia",
		BaseURL:   "https://integrate.api.nvidia.com/v1",
		APIKey:    "test-key",
		Model:     "nvidia/nv-embed-v1",
		Dimension: 4096,
	}

	provider := NewNvidiaEmbeddingProvider(cfg)
	assert.Equal(t, 4096, provider.Dimension())
}

// TestGetEmbeddingProvider tests provider initialization
func TestGetEmbeddingProvider(t *testing.T) {
	err := config.Load()
	require.NoError(t, err)

	apiKey := os.Getenv("EMBEDDING_API_KEY")
	if apiKey == "" {
		t.Skip("EMBEDDING_API_KEY not set, skipping test")
	}

	provider, err := GetEmbeddingProvider()
	require.NoError(t, err)
	assert.NotNil(t, provider)
	assert.Equal(t, 4096, provider.Dimension())

	t.Logf("✅ Successfully initialized embedding provider")
	t.Logf("   Dimension: %d", provider.Dimension())
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

func sqrt(x float64) float64 {
	if x < 0 {
		return 0
	}
	// Simple Newton's method for square root
	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}

// Benchmark tests
func BenchmarkNvidiaEmbedding_Single(b *testing.B) {
	apiKey := os.Getenv("EMBEDDING_API_KEY")
	if apiKey == "" {
		b.Skip("EMBEDDING_API_KEY not set")
	}

	cfg := config.EmbeddingConfig{
		Provider:  "nvidia",
		BaseURL:   "https://integrate.api.nvidia.com/v1",
		APIKey:    apiKey,
		Model:     "nvidia/nv-embed-v1",
		Dimension: 4096,
	}

	provider := NewNvidiaEmbeddingProvider(cfg)
	ctx := context.Background()
	text := "人工智能技术"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.GenerateEmbedding(ctx, text)
	}
}

func BenchmarkSimpleEmbedding(b *testing.B) {
	text := "人工智能技术"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateSimpleEmbedding(text, 384)
	}
}
