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

const volcengineMultimodalEmbeddingPath = "/api/v3/embeddings/multimodal"

// VolcengineEmbedder 火山引擎 Ark 多模态 embedding 实现
type VolcengineEmbedder struct {
	baseURL                   string
	apiKey                    string
	model                     string
	dimension                 int
	supportsDimensionOverride bool
	customHeaders             map[string]string
	httpClient                *http.Client
	maxRetries                int
}

type volcengineEmbedRequest struct {
	Model      string                   `json:"model"`
	Input      []volcengineInputContent `json:"input"`
	Dimensions int                      `json:"dimensions,omitempty"`
}

type volcengineInputContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type volcengineEmbedResponse struct {
	Data struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

type volcengineErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewVolcengineEmbedder 创建 Volcengine embedder
func NewVolcengineEmbedder(cfg *Config) (*VolcengineEmbedder, error) {
	if cfg.Model == "" {
		return nil, ErrModelNotFound
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com"
	}
	if strings.Contains(baseURL, "/embeddings/multimodal") {
		if idx := strings.Index(baseURL, "/api/"); idx != -1 {
			baseURL = baseURL[:idx]
		}
	} else if strings.HasSuffix(baseURL, "/api/v3") {
		baseURL = strings.TrimSuffix(baseURL, "/api/v3")
	}

	dimension := cfg.Dimension
	if dimension <= 0 {
		dimension = 2048
	}

	return &VolcengineEmbedder{
		baseURL:                   baseURL,
		apiKey:                    cfg.APIKey,
		model:                     cfg.Model,
		dimension:                 dimension,
		supportsDimensionOverride: cfg.SupportsDimensionOverride,
		customHeaders:             cfg.CustomHeaders,
		httpClient:                HTTPClientFromConfig(cfg),
		maxRetries:                maxRetriesFromConfig(cfg),
	}, nil
}

func (e *VolcengineEmbedder) Name() string     { return ProviderVolcengine }
func (e *VolcengineEmbedder) Provider() string { return ProviderVolcengine }
func (e *VolcengineEmbedder) Dimension() int   { return e.dimension }
func (e *VolcengineEmbedder) Close() error     { return nil }

func (e *VolcengineEmbedder) supportsDimensionsParam() bool {
	return e.supportsDimensionOverride && e.dimension > 0
}

func (e *VolcengineEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}

	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		vector, err := e.embedSingle(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings[i] = vector
	}
	return embeddings, nil
}

func (e *VolcengineEmbedder) embedSingle(ctx context.Context, text string) ([]float32, error) {
	reqBody := volcengineEmbedRequest{
		Model: e.model,
		Input: []volcengineInputContent{{Type: "text", Text: text}},
	}
	if e.supportsDimensionsParam() {
		reqBody.Dimensions = e.dimension
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
	}

	url := e.baseURL + volcengineMultimodalEmbeddingPath
	resp, err := DoRequestWithRetry(ctx, e.httpClient, e.maxRetries, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
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

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
	}
	if resp.StatusCode != http.StatusOK {
		var errResp volcengineErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("%w: %s - %s", ErrEmbedFailed, errResp.Error.Code, errResp.Error.Message)
		}
		return nil, fmt.Errorf("%w: status %d: %s", ErrEmbedFailed, resp.StatusCode, string(body))
	}

	var response volcengineEmbedResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return response.Data.Embedding, nil
}

func (e *VolcengineEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyInput
	}
	return e.embedSingle(ctx, text)
}
