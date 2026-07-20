package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

const zhipuEmbeddingBaseURL = "https://open.bigmodel.cn/api/paas/v4"

// ZhipuEmbedder 智谱 embedding 实现
type ZhipuEmbedder struct {
	baseURL                   string
	apiKey                    string
	model                     string
	dimension                 int
	truncatePromptTokens      int
	supportsDimensionOverride bool
	customHeaders             map[string]string
	httpClient                *http.Client
	maxRetries                int
}

type zhipuEmbedRequest struct {
	Model                string   `json:"model"`
	Input                []string `json:"input"`
	Dimensions           int      `json:"dimensions,omitempty"`
	TruncatePromptTokens int      `json:"truncate_prompt_tokens,omitempty"`
}

type zhipuEmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// NewZhipuEmbedder 创建 Zhipu embedder
func NewZhipuEmbedder(cfg *Config) (*ZhipuEmbedder, error) {
	if cfg.Model == "" {
		return nil, ErrModelNotFound
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = zhipuEmbeddingBaseURL
	}

	dimension := cfg.Dimension
	if dimension <= 0 {
		dimension = 1024
	}

	return &ZhipuEmbedder{
		baseURL:                   baseURL,
		apiKey:                    cfg.APIKey,
		model:                     cfg.Model,
		dimension:                 dimension,
		truncatePromptTokens:      truncatePromptTokensFromConfig(cfg),
		supportsDimensionOverride: cfg.SupportsDimensionOverride,
		customHeaders:             cfg.CustomHeaders,
		httpClient:                HTTPClientFromConfig(cfg),
		maxRetries:                maxRetriesFromConfig(cfg),
	}, nil
}

func (e *ZhipuEmbedder) Name() string     { return ProviderZhipu }
func (e *ZhipuEmbedder) Provider() string { return ProviderZhipu }
func (e *ZhipuEmbedder) Dimension() int   { return e.dimension }
func (e *ZhipuEmbedder) Close() error     { return nil }

func (e *ZhipuEmbedder) supportsDimensionsParam() bool {
	return e.supportsDimensionOverride && e.dimension > 0
}

func (e *ZhipuEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}

	reqBody := zhipuEmbedRequest{
		Model:                e.model,
		Input:                SanitizeEmbedInputs(texts),
		TruncatePromptTokens: e.truncatePromptTokens,
	}
	if e.supportsDimensionsParam() {
		reqBody.Dimensions = e.dimension
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
	}

	resp, err := DoRequestWithRetry(ctx, e.httpClient, e.maxRetries, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(jsonData))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+e.apiKey)
		ApplyCustomHeaders(req, e.customHeaders)
		return req, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d: %s", ErrEmbedFailed, resp.StatusCode, string(body))
	}

	var response zhipuEmbedResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	vectors := make([][]float32, 0, len(response.Data))
	for _, item := range response.Data {
		vectors = append(vectors, item.Embedding)
	}
	return vectors, nil
}

func (e *ZhipuEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyInput
	}
	vectors, err := e.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, ErrInvalidResponse
	}
	return vectors[0], nil
}
