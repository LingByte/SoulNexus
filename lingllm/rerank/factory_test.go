package rerank

import (
	"testing"
)

func TestCreate_SiliconFlow(t *testing.T) {
	cfg := &RerankConfig{
		Provider: ProviderSiliconFlow,
		BaseURL:  "https://api.siliconflow.cn",
		APIKey:   "test-key",
		Model:    "test-model",
	}

	reranker, err := Create(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reranker == nil {
		t.Fatal("expected reranker, got nil")
	}
	if reranker.Provider() != ProviderSiliconFlow {
		t.Errorf("expected provider %s, got %s", ProviderSiliconFlow, reranker.Provider())
	}
}

func TestCreate_Jina(t *testing.T) {
	cfg := &RerankConfig{
		Provider: ProviderJina,
		BaseURL:  "https://api.jina.ai",
		APIKey:   "test-key",
		Model:    "test-model",
	}

	reranker, err := Create(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reranker == nil {
		t.Fatal("expected reranker, got nil")
	}
	if reranker.Provider() != ProviderJina {
		t.Errorf("expected provider %s, got %s", ProviderJina, reranker.Provider())
	}
}

func TestCreate_JinaAI_Alias(t *testing.T) {
	cfg := &RerankConfig{
		Provider: ProviderJinaAI,
		BaseURL:  "https://api.jina.ai",
		APIKey:   "test-key",
		Model:    "test-model",
	}

	reranker, err := Create(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reranker.Provider() != ProviderJina {
		t.Errorf("expected provider %s, got %s", ProviderJina, reranker.Provider())
	}
}

func TestCreate_Cohere(t *testing.T) {
	cfg := &RerankConfig{
		Provider: ProviderCohere,
		BaseURL:  "https://api.cohere.ai",
		APIKey:   "test-key",
		Model:    "test-model",
	}

	reranker, err := Create(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reranker == nil {
		t.Fatal("expected reranker, got nil")
	}
	if reranker.Provider() != ProviderCohere {
		t.Errorf("expected provider %s, got %s", ProviderCohere, reranker.Provider())
	}
}

func TestCreate_OpenAI(t *testing.T) {
	cfg := &RerankConfig{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		Model:    "test-model",
	}

	reranker, err := Create(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reranker.Provider() != ProviderOpenAI {
		t.Errorf("expected provider %s, got %s", ProviderOpenAI, reranker.Provider())
	}
}

func TestCreate_OpenAI_DefaultFromURL(t *testing.T) {
	cfg := &RerankConfig{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "test-key",
		Model:   "test-model",
	}

	reranker, err := Create(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reranker.Provider() != ProviderOpenAI {
		t.Errorf("expected provider %s, got %s", ProviderOpenAI, reranker.Provider())
	}
}

func TestCreate_Aliyun(t *testing.T) {
	cfg := &RerankConfig{
		Provider: ProviderAliyun,
		APIKey:   "test-key",
		Model:    "gte-rerank",
	}

	reranker, err := Create(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reranker.Provider() != ProviderAliyun {
		t.Errorf("expected provider %s, got %s", ProviderAliyun, reranker.Provider())
	}
}

func TestCreate_Zhipu(t *testing.T) {
	cfg := &RerankConfig{
		Provider: ProviderZhipu,
		APIKey:   "test-key",
		Model:    "rerank",
	}

	reranker, err := Create(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reranker.Provider() != ProviderZhipu {
		t.Errorf("expected provider %s, got %s", ProviderZhipu, reranker.Provider())
	}
}

func TestCreate_Nvidia(t *testing.T) {
	cfg := &RerankConfig{
		Provider: ProviderNvidia,
		APIKey:   "test-key",
		Model:    "nv-rerankqa-mistral-4b-v3",
	}

	reranker, err := Create(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reranker.Provider() != ProviderNvidia {
		t.Errorf("expected provider %s, got %s", ProviderNvidia, reranker.Provider())
	}
}

func TestCreate_LKEAP(t *testing.T) {
	cfg := &RerankConfig{
		Provider:  ProviderLKEAP,
		APIKey:    "AKIDtest",
		SecretKey: "sk-test",
		Model:     "lke-reranker-base",
	}

	reranker, err := Create(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reranker.Provider() != ProviderLKEAP {
		t.Errorf("expected provider %s, got %s", ProviderLKEAP, reranker.Provider())
	}
}

func TestCreate_Local(t *testing.T) {
	cfg := &RerankConfig{
		Provider: ProviderLocal,
	}

	reranker, err := Create(cfg)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if reranker == nil {
		t.Fatal("expected reranker, got nil")
	}
	if reranker.Provider() != ProviderLocal {
		t.Errorf("expected provider %s, got %s", ProviderLocal, reranker.Provider())
	}
}

func TestCreate_MissingProvider(t *testing.T) {
	cfg := &RerankConfig{}

	_, err := Create(cfg)
	if err == nil {
		t.Fatal("expected error for missing provider")
	}
}

func TestCreate_UnsupportedProvider(t *testing.T) {
	cfg := &RerankConfig{
		Provider: "unsupported",
	}

	_, err := Create(cfg)
	if err == nil {
		t.Fatal("expected error for unsupported provider")
	}
}

func TestCreate_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *RerankConfig
		hasErr bool
	}{
		{
			name: "SiliconFlow missing BaseURL",
			cfg: &RerankConfig{
				Provider: ProviderSiliconFlow,
				APIKey:   "test",
				Model:    "test",
			},
			hasErr: true,
		},
		{
			name: "SiliconFlow missing APIKey",
			cfg: &RerankConfig{
				Provider: ProviderSiliconFlow,
				BaseURL:  "https://api.test.com",
				Model:    "test",
			},
			hasErr: true,
		},
		{
			name: "SiliconFlow missing Model",
			cfg: &RerankConfig{
				Provider: ProviderSiliconFlow,
				BaseURL:  "https://api.test.com",
				APIKey:   "test",
			},
			hasErr: true,
		},
		{
			name: "LKEAP missing SecretKey",
			cfg: &RerankConfig{
				Provider: ProviderLKEAP,
				APIKey:   "test",
			},
			hasErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Create(tt.cfg)
			if (err != nil) != tt.hasErr {
				t.Errorf("expected error=%v, got %v", tt.hasErr, err)
			}
		})
	}
}
