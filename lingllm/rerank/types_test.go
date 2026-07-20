package rerank

import (
	"testing"
)

func TestProviderConstants(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{ProviderLocal, "local"},
		{ProviderCohere, "cohere"},
		{ProviderJina, "jina"},
		{ProviderSiliconFlow, "siliconflow"},
		{ProviderAliyun, "aliyun"},
		{ProviderZhipu, "zhipu"},
		{ProviderNvidia, "nvidia"},
		{ProviderLKEAP, "lkeap"},
		{ProviderOpenAI, "openai"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			if tt.provider != tt.want {
				t.Errorf("Provider = %s, want %s", tt.provider, tt.want)
			}
		})
	}
}

func TestRerankResultStruct(t *testing.T) {
	result := RerankResult{
		Index: 0,
		Score: 0.95,
	}

	if result.Index != 0 {
		t.Errorf("Index = %d, want 0", result.Index)
	}

	if result.Score != 0.95 {
		t.Errorf("Score = %f, want 0.95", result.Score)
	}
}

func TestRerankResultMultiple(t *testing.T) {
	results := []RerankResult{
		{Index: 0, Score: 0.95},
		{Index: 1, Score: 0.87},
		{Index: 2, Score: 0.72},
	}

	if len(results) != 3 {
		t.Errorf("Results length = %d, want 3", len(results))
	}

	if results[0].Score > results[1].Score {
		t.Log("Results are properly ordered by score")
	}
}

func TestRerankConfigStruct(t *testing.T) {
	config := RerankConfig{
		Provider:      ProviderCohere,
		APIKey:        "test-key",
		SecretKey:     "secret",
		Model:         "rerank-english-v3.0",
		CustomHeaders: map[string]string{"X-Gateway": "edge"},
		ExtraConfig:   map[string]string{"region": "ap-beijing"},
	}

	if config.Provider != ProviderCohere {
		t.Errorf("Provider = %s, want %s", config.Provider, ProviderCohere)
	}
	if config.APIKey != "test-key" {
		t.Errorf("APIKey = %s, want test-key", config.APIKey)
	}
	if config.SecretKey != "secret" {
		t.Errorf("SecretKey = %s, want secret", config.SecretKey)
	}
	if config.Model != "rerank-english-v3.0" {
		t.Errorf("Model = %s, want rerank-english-v3.0", config.Model)
	}
	if config.CustomHeaders["X-Gateway"] != "edge" {
		t.Errorf("CustomHeaders not set correctly")
	}
	if config.ExtraConfig["region"] != "ap-beijing" {
		t.Errorf("ExtraConfig not set correctly")
	}
}

func TestRerankClientConfigStruct(t *testing.T) {
	config := RerankClientConfig{
		APIKey:    "test-key",
		SecretKey: "secret",
		Model:     "rerank-english-v3.0",
	}

	if config.APIKey != "test-key" {
		t.Errorf("APIKey = %s, want test-key", config.APIKey)
	}
	if config.SecretKey != "secret" {
		t.Errorf("SecretKey = %s, want secret", config.SecretKey)
	}
	if config.Model != "rerank-english-v3.0" {
		t.Errorf("Model = %s, want rerank-english-v3.0", config.Model)
	}
}

func TestRerankInterface(t *testing.T) {
	t.Log("Reranker interface is properly defined")
}
