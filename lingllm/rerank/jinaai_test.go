package rerank

import (
	"testing"
	"time"
)

func TestJinaAIRerankClientProvider(t *testing.T) {
	config := &RerankClientConfig{
		APIKey: "test-key",
		Model:  "jina-reranker-v1-base-en",
	}

	client := NewJinaAIRerankClient(config)
	if client == nil {
		t.Error("NewJinaAIRerankClient should not return nil")
	}

	if client.Provider() != ProviderJina {
		t.Errorf("Provider() = %s, want %s", client.Provider(), ProviderJina)
	}
}

func TestNewJinaAIRerankClient(t *testing.T) {
	config := &RerankClientConfig{
		APIKey:  "test-api-key",
		Model:   "jina-reranker-v1-base-en",
		Timeout: 30 * time.Second,
	}

	client := NewJinaAIRerankClient(config)

	if client == nil {
		t.Error("NewJinaAIRerankClient should not return nil")
	}

	if client.APIKey != "test-api-key" {
		t.Errorf("APIKey = %s, want test-api-key", client.APIKey)
	}

	if client.Model != "jina-reranker-v1-base-en" {
		t.Errorf("Model = %s, want jina-reranker-v1-base-en", client.Model)
	}
}

func TestJinaAIRerankClientNilConfig(t *testing.T) {
	client := NewJinaAIRerankClient(nil)

	if client != nil {
		t.Error("NewJinaAIRerankClient with nil config should return nil")
	}
}
