package rerank

import (
	"testing"
	"time"
)

func TestCohereAIRerankClientProvider(t *testing.T) {
	config := &RerankClientConfig{
		APIKey: "test-key",
		Model:  "rerank-english-v3.0",
	}

	client := NewCohereAIRerankClient(config)
	if client == nil {
		t.Error("NewCohereAIRerankClient should not return nil")
	}

	if client.Provider() != ProviderCohere {
		t.Errorf("Provider() = %s, want %s", client.Provider(), ProviderCohere)
	}
}

func TestNewCohereAIRerankClient(t *testing.T) {
	config := &RerankClientConfig{
		APIKey:  "test-api-key",
		Model:   "rerank-english-v3.0",
		Timeout: 30 * time.Second,
	}

	client := NewCohereAIRerankClient(config)

	if client == nil {
		t.Error("NewCohereAIRerankClient should not return nil")
	}

	if client.APIKey != "test-api-key" {
		t.Errorf("APIKey = %s, want test-api-key", client.APIKey)
	}

	if client.Model != "rerank-english-v3.0" {
		t.Errorf("Model = %s, want rerank-english-v3.0", client.Model)
	}
}

func TestCohereAIRerankClientNilConfig(t *testing.T) {
	client := NewCohereAIRerankClient(nil)

	if client != nil {
		t.Error("NewCohereAIRerankClient with nil config should return nil")
	}
}
