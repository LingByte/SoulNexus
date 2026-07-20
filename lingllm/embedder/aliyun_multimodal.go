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

const aliyunMultimodalEmbeddingEndpoint = "/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding"

// AliyunMultimodalEmbedder 阿里云 DashScope 多模态 embedding 实现
type AliyunMultimodalEmbedder struct {
	baseURL                   string
	apiKey                    string
	model                     string
	dimension                 int
	supportsDimensionOverride bool
	customHeaders             map[string]string
	httpClient                *http.Client
	maxRetries                int
}

type aliyunEmbedParameters struct {
	Dimension int `json:"dimension,omitempty"`
}

type aliyunEmbedRequest struct {
	Model      string                 `json:"model"`
	Input      aliyunEmbedInput       `json:"input"`
	Parameters *aliyunEmbedParameters `json:"parameters,omitempty"`
}

type aliyunEmbedInput struct {
	Contents []aliyunContent `json:"contents"`
}

type aliyunContent struct {
	Text string `json:"text,omitempty"`
}

type aliyunEmbedResponse struct {
	Output struct {
		Embeddings []struct {
			Embedding []float32 `json:"embedding"`
			TextIndex int       `json:"text_index"`
		} `json:"embeddings"`
	} `json:"output"`
}

type aliyunErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewAliyunMultimodalEmbedder 创建阿里云多模态 embedder
func NewAliyunMultimodalEmbedder(cfg *Config) (*AliyunMultimodalEmbedder, error) {
	if cfg.Model == "" {
		return nil, ErrModelNotFound
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com"
	}
	if strings.Contains(baseURL, "/compatible-mode/v1") {
		baseURL = strings.Replace(baseURL, "/compatible-mode/v1", "", 1)
	}

	dimension := cfg.Dimension
	if dimension <= 0 {
		dimension = 1024
	}

	return &AliyunMultimodalEmbedder{
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

func (e *AliyunMultimodalEmbedder) Name() string     { return ProviderAliyun }
func (e *AliyunMultimodalEmbedder) Provider() string { return ProviderAliyun }
func (e *AliyunMultimodalEmbedder) Dimension() int   { return e.dimension }
func (e *AliyunMultimodalEmbedder) Close() error     { return nil }

func (e *AliyunMultimodalEmbedder) supportsDimensionsParam() bool {
	return e.supportsDimensionOverride && e.dimension > 0
}

func (e *AliyunMultimodalEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}

	contents := make([]aliyunContent, 0, len(texts))
	for _, text := range texts {
		contents = append(contents, aliyunContent{Text: text})
	}

	reqBody := aliyunEmbedRequest{
		Model: e.model,
		Input: aliyunEmbedInput{Contents: contents},
	}
	if e.supportsDimensionsParam() {
		reqBody.Parameters = &aliyunEmbedParameters{Dimension: e.dimension}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
	}

	url := e.baseURL + aliyunMultimodalEmbeddingEndpoint
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
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
	}
	if resp.StatusCode != http.StatusOK {
		var errResp aliyunErrorResponse
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("%w: %s - %s", ErrEmbedFailed, errResp.Code, errResp.Message)
		}
		return nil, fmt.Errorf("%w: status %d: %s", ErrEmbedFailed, resp.StatusCode, string(body))
	}

	var response aliyunEmbedResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	embeddings := make([][]float32, len(texts))
	for _, emb := range response.Output.Embeddings {
		if emb.TextIndex >= 0 && emb.TextIndex < len(embeddings) {
			embeddings[emb.TextIndex] = emb.Embedding
		}
	}
	return embeddings, nil
}

func (e *AliyunMultimodalEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
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
