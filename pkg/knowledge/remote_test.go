package knowledge

import (
	"context"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestNvidiaEmbedClient_Integration(t *testing.T) {
	if utils.GetEnv("KNOWLEDGE_INTEGRATION_TESTS") != "1" {
		t.Skip("set KNOWLEDGE_INTEGRATION_TESTS=1 to enable integration tests")
	}
	apiKey := utils.GetEnv("NVIDIA_API_KEY")
	baseURL := utils.GetEnv("NVIDIA_EMBEDDINGS_URL")
	model := utils.GetEnv("NVIDIA_EMBEDDINGS_MODEL")
	if apiKey == "" || baseURL == "" || model == "" {
		t.Skip("missing NVIDIA_API_KEY/NVIDIA_EMBEDDINGS_URL/NVIDIA_EMBEDDINGS_MODEL")
	}

	c := &NvidiaEmbedClient{BaseURL: baseURL, APIKey: apiKey, Model: model}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vecs, err := c.Embed(ctx, []string{"hello world", "你好世界"})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(vecs), 1)
	assert.Greater(t, len(vecs[0]), 0)
}

func TestSiliconFlowRerankClient_Integration(t *testing.T) {
	if utils.GetEnv("KNOWLEDGE_INTEGRATION_TESTS") != "1" {
		t.Skip("set KNOWLEDGE_INTEGRATION_TESTS=1 to enable integration tests")
	}
	apiKey := utils.GetEnv("SILICONFLOW_API_KEY")
	baseURL := utils.GetEnv("SILICONFLOW_RERANK_URL")
	model := utils.GetEnv("SILICONFLOW_RERANK_MODEL")
	if apiKey == "" || baseURL == "" || model == "" {
		t.Skip("missing SILICONFLOW_API_KEY/SILICONFLOW_RERANK_URL/SILICONFLOW_RERANK_MODEL")
	}

	c := &SiliconFlowRerankClient{BaseURL: baseURL, APIKey: apiKey, Model: model}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	docs := []string{
		"Cats are small domesticated animals.",
		"The capital of France is Paris.",
		"Go is a programming language designed at Google.",
	}
	res, err := c.Rerank(ctx, "What is the capital of France?", docs, 2)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(res), 1)
}
