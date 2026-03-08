package knowledge

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

// TestSiliconFlowRerankProvider_RealAPI tests the SiliconFlow rerank provider with real API calls
func TestSiliconFlowRerankProvider_RealAPI(t *testing.T) {
	apiKey := utils.GetEnv("RERANK_API_KEY")
	if apiKey == "" {
		t.Skip("RERANK_API_KEY not set, skipping real API test")
	}

	cfg := config.RerankConfig{
		Provider: "siliconflow",
		BaseURL:  "https://api.siliconflow.cn/v1",
		APIKey:   apiKey,
		Model:    "BAAI/bge-reranker-v2-m3",
		TopN:     5,
	}

	provider := NewSiliconFlowRerankProvider(cfg)
	require.NotNil(t, provider)

	t.Run("BasicRerank", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		query := "什么是人工智能？"
		documents := []string{
			"人工智能（AI）是计算机科学的一个分支，致力于创建能够执行通常需要人类智能的任务的系统。",
			"今天天气很好，适合出去散步。",
			"机器学习是人工智能的一个子领域，专注于让计算机从数据中学习。",
			"我喜欢吃披萨和意大利面。",
			"深度学习使用神经网络来模拟人脑的工作方式。",
		}

		results, err := provider.Rerank(ctx, query, documents, 3)
		require.NoError(t, err)
		assert.NotNil(t, results)
		assert.LessOrEqual(t, len(results), 3, "Should return at most 3 results")

		t.Logf("✅ Rerank results for query: '%s'", query)
		for i, result := range results {
			t.Logf("   %d. [Score: %.4f] %s", i+1, result.RelevanceScore, result.Document[:50]+"...")
		}

		// First result should be most relevant (about AI)
		if len(results) > 0 {
			assert.Contains(t, results[0].Document, "人工智能", "Most relevant document should be about AI")
			assert.Greater(t, results[0].RelevanceScore, 0.5, "Top result should have high relevance score")
		}
	})

	t.Run("RerankWithKnowledgeBase", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		query := "如何使用知识库？"
		documents := []string{
			"知识库是一个存储和检索信息的系统，可以帮助用户快速找到所需的知识。",
			"要使用知识库，首先需要创建一个知识库实例，然后上传文档。",
			"知识库支持多种文档格式，包括PDF、Word、文本文件等。",
			"搜索知识库时，系统会使用向量相似度来找到最相关的文档。",
			"重排序（Rerank）可以进一步提高搜索结果的准确性。",
		}

		results, err := provider.Rerank(ctx, query, documents, 5)
		require.NoError(t, err)
		assert.NotNil(t, results)

		t.Logf("✅ Knowledge base rerank results:")
		for i, result := range results {
			t.Logf("   %d. [Score: %.4f] %s", i+1, result.RelevanceScore, result.Document)
		}

		// Results should be sorted by relevance score
		for i := 1; i < len(results); i++ {
			assert.GreaterOrEqual(t, results[i-1].RelevanceScore, results[i].RelevanceScore,
				"Results should be sorted by relevance score in descending order")
		}
	})

	t.Run("RerankWithDifferentLanguages", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		query := "machine learning algorithms"
		documents := []string{
			"机器学习算法包括监督学习、无监督学习和强化学习。",
			"Machine learning algorithms can be categorized into supervised, unsupervised, and reinforcement learning.",
			"今天的午餐很美味。",
			"The weather is nice today.",
			"深度学习是机器学习的一个重要分支。",
		}

		results, err := provider.Rerank(ctx, query, documents, 3)
		require.NoError(t, err)

		t.Logf("✅ Cross-language rerank results:")
		for i, result := range results {
			t.Logf("   %d. [Score: %.4f] %s", i+1, result.RelevanceScore, result.Document)
		}

		// Should rank relevant documents higher regardless of language
		if len(results) > 0 {
			assert.Greater(t, results[0].RelevanceScore, 0.3, "Top result should be relevant")
		}
	})

	t.Run("EmptyDocuments", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		query := "test query"
		documents := []string{}

		_, err := provider.Rerank(ctx, query, documents, 5)
		assert.Error(t, err, "Should return error for empty documents")
	})

	t.Run("SingleDocument", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		query := "人工智能"
		documents := []string{"人工智能是计算机科学的一个分支。"}

		results, err := provider.Rerank(ctx, query, documents, 5)
		require.NoError(t, err)
		assert.Equal(t, 1, len(results))
		t.Logf("✅ Single document rerank score: %.4f", results[0].RelevanceScore)
	})
}

// TestRerankSearchResults tests the wrapper function
func TestRerankSearchResults(t *testing.T) {
	err := config.Load()
	require.NoError(t, err)

	apiKey := os.Getenv("RERANK_API_KEY")
	if apiKey == "" {
		t.Skip("RERANK_API_KEY not set, skipping test")
	}

	t.Run("RerankWithSearchResults", func(t *testing.T) {
		query := "人工智能技术"
		results := []SearchResult{
			{Content: "人工智能是计算机科学的一个分支。", Score: 0.8},
			{Content: "今天天气很好。", Score: 0.7},
			{Content: "机器学习是AI的核心技术。", Score: 0.75},
			{Content: "我喜欢吃水果。", Score: 0.6},
		}

		reranked, err := RerankSearchResults(query, results, 2)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(reranked), 2)

		t.Logf("✅ Reranked search results:")
		for i, result := range reranked {
			t.Logf("   %d. [Score: %.4f] %s", i+1, result.Score, result.Content)
		}

		// Scores should be updated with rerank scores
		if len(reranked) > 0 {
			assert.NotEqual(t, 0.8, reranked[0].Score, "Score should be updated by rerank")
		}
	})

	t.Run("RerankWithEmptyResults", func(t *testing.T) {
		query := "test"
		results := []SearchResult{}

		reranked, err := RerankSearchResults(query, results, 5)
		require.NoError(t, err)
		assert.Equal(t, 0, len(reranked))
	})

	t.Run("RerankFallback", func(t *testing.T) {
		// Temporarily clear API key to trigger fallback
		originalKey := config.GlobalConfig.Services.KnowledgeBase.Rerank.APIKey
		config.GlobalConfig.Services.KnowledgeBase.Rerank.APIKey = ""

		query := "test"
		results := []SearchResult{
			{Content: "content 1", Score: 0.8},
			{Content: "content 2", Score: 0.7},
		}

		reranked, err := RerankSearchResults(query, results, 5)
		require.NoError(t, err)
		assert.Equal(t, len(results), len(reranked), "Should return original results on fallback")

		// Restore API key
		config.GlobalConfig.Services.KnowledgeBase.Rerank.APIKey = originalKey
	})
}

// TestGetRerankProvider tests provider initialization
func TestGetRerankProvider(t *testing.T) {
	err := config.Load()
	require.NoError(t, err)

	apiKey := os.Getenv("RERANK_API_KEY")
	if apiKey == "" {
		t.Skip("RERANK_API_KEY not set, skipping test")
	}

	provider, err := GetRerankProvider()
	require.NoError(t, err)
	assert.NotNil(t, provider)

	t.Logf("✅ Successfully initialized rerank provider")
}

// Benchmark tests
func BenchmarkSiliconFlowRerank(b *testing.B) {
	apiKey := os.Getenv("RERANK_API_KEY")
	if apiKey == "" {
		b.Skip("RERANK_API_KEY not set")
	}

	cfg := config.RerankConfig{
		Provider: "siliconflow",
		BaseURL:  "https://api.siliconflow.cn/v1",
		APIKey:   apiKey,
		Model:    "BAAI/bge-reranker-v2-m3",
		TopN:     5,
	}

	provider := NewSiliconFlowRerankProvider(cfg)
	ctx := context.Background()

	query := "人工智能"
	documents := []string{
		"人工智能是计算机科学的一个分支。",
		"机器学习是AI的核心技术。",
		"深度学习使用神经网络。",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.Rerank(ctx, query, documents, 3)
	}
}
