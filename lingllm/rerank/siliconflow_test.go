package rerank

import (
	"testing"
	"time"
)

func TestSiliconFlowRerankClientProvider(t *testing.T) {
	config := &RerankClientConfig{
		APIKey: "test-key",
		Model:  "BAAI/bge-reranker-v2-m3",
	}

	client := NewSiliconFlowRerankClient(config)
	if client == nil {
		t.Error("NewSiliconFlowRerankClient should not return nil")
	}

	if client.Provider() != ProviderSiliconFlow {
		t.Errorf("Provider() = %s, want %s", client.Provider(), ProviderSiliconFlow)
	}
}

func TestNewSiliconFlowRerankClient(t *testing.T) {
	config := &RerankClientConfig{
		APIKey:  "test-api-key",
		Model:   "BAAI/bge-reranker-v2-m3",
		Timeout: 30 * time.Second,
	}

	client := NewSiliconFlowRerankClient(config)

	if client == nil {
		t.Error("NewSiliconFlowRerankClient should not return nil")
	}

	if client.APIKey != "test-api-key" {
		t.Errorf("APIKey = %s, want test-api-key", client.APIKey)
	}

	if client.Model != "BAAI/bge-reranker-v2-m3" {
		t.Errorf("Model = %s, want BAAI/bge-reranker-v2-m3", client.Model)
	}
}

func TestSiliconFlowRerankClientNilConfig(t *testing.T) {
	client := NewSiliconFlowRerankClient(nil)

	if client != nil {
		t.Error("NewSiliconFlowRerankClient with nil config should return nil")
	}
}
