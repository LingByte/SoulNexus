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

const geminiEmbeddingBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// GeminiEmbedder Gemini embedding 实现
type GeminiEmbedder struct {
	baseURL                   string
	apiKey                    string
	model                     string
	dimension                 int
	supportsDimensionOverride bool
	customHeaders             map[string]string
	httpClient                *http.Client
	maxRetries                int
}

type geminiBatchEmbedRequest struct {
	Requests []geminiEmbedRequest `json:"requests"`
}

type geminiEmbedRequest struct {
	Model                string        `json:"model"`
	Content              geminiContent `json:"content"`
	OutputDimensionality int           `json:"output_dimensionality,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiBatchEmbedResponse struct {
	Embeddings []geminiEmbedding `json:"embeddings"`
}

type geminiEmbedding struct {
	Values []float32 `json:"values"`
}

// NewGeminiEmbedder 创建 Gemini embedder
func NewGeminiEmbedder(cfg *Config) (*GeminiEmbedder, error) {
	if cfg.Model == "" {
		return nil, ErrModelNotFound
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = geminiEmbeddingBaseURL
	}
	if strings.HasSuffix(baseURL, "/openai") {
		baseURL = strings.TrimSuffix(baseURL, "/openai")
	}

	dimension := cfg.Dimension
	if dimension <= 0 {
		dimension = 768
	}

	return &GeminiEmbedder{
		baseURL:                   baseURL,
		apiKey:                    cfg.APIKey,
		model:                     strings.TrimPrefix(cfg.Model, "models/"),
		dimension:                 dimension,
		supportsDimensionOverride: cfg.SupportsDimensionOverride,
		customHeaders:             cfg.CustomHeaders,
		httpClient:                HTTPClientFromConfig(cfg),
		maxRetries:                maxRetriesFromConfig(cfg),
	}, nil
}

func (e *GeminiEmbedder) Name() string     { return ProviderGemini }
func (e *GeminiEmbedder) Provider() string { return ProviderGemini }
func (e *GeminiEmbedder) Dimension() int   { return e.dimension }
func (e *GeminiEmbedder) Close() error     { return nil }

func (e *GeminiEmbedder) supportsDimensionsParam() bool {
	return e.supportsDimensionOverride && e.dimension > 0
}

func (e *GeminiEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}

	requests := make([]geminiEmbedRequest, 0, len(texts))
	for _, text := range texts {
		req := geminiEmbedRequest{
			Model: "models/" + e.model,
			Content: geminiContent{Parts: []geminiPart{
				{Text: text},
			}},
		}
		if e.supportsDimensionsParam() {
			req.OutputDimensionality = e.dimension
		}
		requests = append(requests, req)
	}

	jsonData, err := json.Marshal(geminiBatchEmbedRequest{Requests: requests})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
	}

	url := fmt.Sprintf("%s/models/%s:batchEmbedContents", e.baseURL, e.model)
	resp, err := DoRequestWithRetry(ctx, e.httpClient, e.maxRetries, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-goog-api-key", e.apiKey)
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

	var response geminiBatchEmbedResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if len(response.Embeddings) != len(texts) {
		return nil, fmt.Errorf("%w: got %d embeddings for %d inputs", ErrInvalidResponse, len(response.Embeddings), len(texts))
	}

	vectors := make([][]float32, 0, len(response.Embeddings))
	for _, embedding := range response.Embeddings {
		vectors = append(vectors, embedding.Values)
	}
	return vectors, nil
}

func (e *GeminiEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
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
