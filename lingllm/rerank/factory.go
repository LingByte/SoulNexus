package rerank

import (
	"fmt"
	"strings"
	"time"
)

// Create creates a new reranker based on the provider.
func Create(cfg *RerankConfig) (Reranker, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	providerName := ProviderName(cfg.Provider)
	if providerName == "" {
		if strings.TrimSpace(cfg.BaseURL) == "" {
			return nil, fmt.Errorf("provider is required")
		}
		providerName = DetectProvider(cfg.BaseURL)
	}

	clientCfg := &RerankClientConfig{
		BaseURL:       cfg.BaseURL,
		APIKey:        cfg.APIKey,
		SecretKey:     cfg.SecretKey,
		Model:         cfg.Model,
		HTTPClient:    cfg.HTTPClient,
		CustomHeaders: cfg.CustomHeaders,
		ExtraConfig:   cfg.ExtraConfig,
	}
	if cfg.Timeout > 0 {
		clientCfg.Timeout = time.Duration(cfg.Timeout) * time.Second
	}

	switch providerName {
	case ProviderAliyun:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("APIKey is required for Aliyun")
		}
		if cfg.Model == "" {
			return nil, fmt.Errorf("Model is required for Aliyun")
		}
		return NewAliyunRerankClient(clientCfg), nil

	case ProviderZhipu:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("APIKey is required for Zhipu")
		}
		if cfg.Model == "" {
			return nil, fmt.Errorf("Model is required for Zhipu")
		}
		return NewZhipuRerankClient(clientCfg), nil

	case ProviderJina, ProviderJinaAI:
		if cfg.BaseURL == "" {
			return nil, fmt.Errorf("BaseURL is required for Jina")
		}
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("APIKey is required for Jina")
		}
		if cfg.Model == "" {
			return nil, fmt.Errorf("Model is required for Jina")
		}
		return NewJinaAIRerankClient(clientCfg), nil

	case ProviderNvidia:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("APIKey is required for NVIDIA")
		}
		if cfg.Model == "" {
			return nil, fmt.Errorf("Model is required for NVIDIA")
		}
		return NewNvidiaRerankClient(clientCfg), nil

	case ProviderLKEAP:
		return NewLKEAPRerankClient(clientCfg)

	case ProviderSiliconFlow:
		if cfg.BaseURL == "" {
			return nil, fmt.Errorf("BaseURL is required for SiliconFlow")
		}
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("APIKey is required for SiliconFlow")
		}
		if cfg.Model == "" {
			return nil, fmt.Errorf("Model is required for SiliconFlow")
		}
		return NewSiliconFlowRerankClient(clientCfg), nil

	case ProviderCohere, ProviderCohereAI:
		if cfg.BaseURL == "" {
			return nil, fmt.Errorf("BaseURL is required for Cohere")
		}
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("APIKey is required for Cohere")
		}
		if cfg.Model == "" {
			return nil, fmt.Errorf("Model is required for Cohere")
		}
		return NewCohereAIRerankClient(clientCfg), nil

	case ProviderLocal:
		return NewLocalRerankClient(clientCfg), nil

	case ProviderOpenAI:
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("APIKey is required for OpenAI")
		}
		if cfg.Model == "" {
			return nil, fmt.Errorf("Model is required for OpenAI")
		}
		return NewOpenAIRerankClient(clientCfg), nil

	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}
