package embedder

import (
	"context"
	"fmt"
	"strings"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func createEmbedder(ctx context.Context, cfg *Config) (Embedder, error) {
	if cfg == nil {
		return nil, ErrInvalidConfig
	}

	providerName := NormalizeProvider(cfg.Provider)
	if providerName == "" && strings.TrimSpace(cfg.BaseURL) != "" {
		providerName = DetectProvider(cfg.BaseURL)
	}
	if providerName == "" {
		return nil, ErrProviderNotFound
	}
	if !isKnownProvider(providerName) {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, providerName)
	}

	switch providerName {
	case ProviderOllama:
		if cfg.Model == "" {
			return nil, ErrModelNotFound
		}
		return NewOllamaEmbedder(cfg), nil

	case ProviderLocal:
		if cfg.Model == "" {
			return nil, ErrModelNotFound
		}
		return NewLocalEmbedder(cfg), nil

	case ProviderAliyun, ProviderDashscope:
		return createAliyunEmbedder(cfg)

	case ProviderVolcengine:
		if cfg.APIKey == "" {
			return nil, ErrAPIKeyRequired
		}
		if cfg.Model == "" {
			return nil, ErrModelNotFound
		}
		return NewVolcengineEmbedder(cfg)

	case ProviderJina:
		if cfg.APIKey == "" {
			return nil, ErrAPIKeyRequired
		}
		if cfg.Model == "" {
			return nil, ErrModelNotFound
		}
		return NewJinaEmbedder(cfg)

	case ProviderAzureOpenAI:
		if cfg.APIKey == "" {
			return nil, ErrAPIKeyRequired
		}
		if cfg.BaseURL == "" {
			return nil, ErrBaseURLRequired
		}
		if cfg.Model == "" {
			return nil, ErrModelNotFound
		}
		return NewAzureOpenAIEmbedder(cfg)

	case ProviderNvidia:
		if cfg.APIKey == "" {
			return nil, ErrAPIKeyRequired
		}
		if cfg.Model == "" {
			return nil, ErrModelNotFound
		}
		return NewNvidiaEmbedder(cfg), nil

	case ProviderGemini:
		if cfg.APIKey == "" {
			return nil, ErrAPIKeyRequired
		}
		if cfg.Model == "" {
			return nil, ErrModelNotFound
		}
		return NewGeminiEmbedder(cfg)

	case ProviderZhipu:
		if cfg.APIKey == "" {
			return nil, ErrAPIKeyRequired
		}
		if cfg.Model == "" {
			return nil, ErrModelNotFound
		}
		return NewZhipuEmbedder(cfg)

	case ProviderOpenAI:
		if cfg.APIKey == "" {
			return nil, ErrAPIKeyRequired
		}
		if cfg.Model == "" {
			return nil, ErrModelNotFound
		}
		return NewOpenAIEmbedder(cfg), nil

	default:
		if cfg.APIKey == "" {
			return nil, ErrAPIKeyRequired
		}
		if cfg.Model == "" {
			return nil, ErrModelNotFound
		}
		return NewOpenAIEmbedder(cfg), nil
	}
}

func createAliyunEmbedder(cfg *Config) (Embedder, error) {
	if cfg.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}
	if cfg.Model == "" {
		return nil, ErrModelNotFound
	}

	modelLower := strings.ToLower(cfg.Model)
	isMultimodalModel := strings.Contains(modelLower, "vision") ||
		strings.Contains(modelLower, "multimodal")

	if isMultimodalModel {
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://dashscope.aliyuncs.com"
		} else if strings.Contains(baseURL, "/compatible-mode/") {
			baseURL = strings.Replace(baseURL, "/compatible-mode/v1", "", 1)
			baseURL = strings.Replace(baseURL, "/compatible-mode", "", 1)
		}
		cfgCopy := *cfg
		cfgCopy.BaseURL = baseURL
		return NewAliyunMultimodalEmbedder(&cfgCopy)
	}

	baseURL := cfg.BaseURL
	if baseURL == "" || !strings.Contains(baseURL, "/compatible-mode/") {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	cfgCopy := *cfg
	cfgCopy.BaseURL = baseURL
	return NewOpenAIEmbedder(&cfgCopy), nil
}
