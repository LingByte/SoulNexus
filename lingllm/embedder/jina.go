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

// JinaEmbedder Jina AI embedding 实现
type JinaEmbedder struct {
	baseURL                   string
	apiKey                    string
	model                     string
	dimension                 int
	supportsDimensionOverride bool
	customHeaders             map[string]string
	httpClient                *http.Client
	maxRetries                int
}

type jinaEmbedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Truncate   bool     `json:"truncate,omitempty"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type jinaEmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// NewJinaEmbedder 创建 Jina embedder
func NewJinaEmbedder(cfg *Config) (*JinaEmbedder, error) {
	if cfg.Model == "" {
		return nil, ErrModelNotFound
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.jina.ai/v1"
	}

	dimension := cfg.Dimension
	if dimension <= 0 {
		dimension = 1024
	}

	return &JinaEmbedder{
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

func (e *JinaEmbedder) Name() string  { return ProviderJina }
func (e *JinaEmbedder) Provider() string { return ProviderJina }
func (e *JinaEmbedder) Dimension() int   { return e.dimension }
func (e *JinaEmbedder) Close() error     { return nil }

func (e *JinaEmbedder) supportsDimensionsParam() bool {
	return e.supportsDimensionOverride && e.dimension > 0
}

func (e *JinaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}

	reqBody := jinaEmbedRequest{
		Model:    e.model,
		Input:    SanitizeEmbedInputs(texts),
		Truncate: true,
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

	var response jinaEmbedResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	vectors := make([][]float32, 0, len(response.Data))
	for _, item := range response.Data {
		vectors = append(vectors, item.Embedding)
	}
	return vectors, nil
}

func (e *JinaEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
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
