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

// AzureOpenAIEmbedder Azure OpenAI embedding 实现
type AzureOpenAIEmbedder struct {
	baseURL                   string
	apiKey                    string
	model                     string
	dimension                 int
	apiVersion                string
	supportsDimensionOverride bool
	customHeaders             map[string]string
	httpClient                *http.Client
	maxRetries                int
}

type azureOpenAIEmbedRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format,omitempty"`
	Dimensions     int      `json:"dimensions,omitempty"`
}

// NewAzureOpenAIEmbedder 创建 Azure OpenAI embedder
func NewAzureOpenAIEmbedder(cfg *Config) (*AzureOpenAIEmbedder, error) {
	if cfg.BaseURL == "" {
		return nil, ErrBaseURLRequired
	}
	if cfg.Model == "" {
		return nil, ErrModelNotFound
	}

	apiVersion := CustomConfigString(cfg, "api_version")
	if apiVersion == "" {
		apiVersion = "2024-10-21"
	}

	dimension := cfg.Dimension
	if dimension <= 0 {
		dimension = 1536
	}

	return &AzureOpenAIEmbedder{
		baseURL:                   strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:                    cfg.APIKey,
		model:                     cfg.Model,
		dimension:                 dimension,
		apiVersion:                apiVersion,
		supportsDimensionOverride: cfg.SupportsDimensionOverride,
		customHeaders:             cfg.CustomHeaders,
		httpClient:                HTTPClientFromConfig(cfg),
		maxRetries:                maxRetriesFromConfig(cfg),
	}, nil
}

func (e *AzureOpenAIEmbedder) Name() string     { return ProviderAzureOpenAI }
func (e *AzureOpenAIEmbedder) Provider() string { return ProviderAzureOpenAI }
func (e *AzureOpenAIEmbedder) Dimension() int   { return e.dimension }
func (e *AzureOpenAIEmbedder) Close() error     { return nil }

func (e *AzureOpenAIEmbedder) supportsDimensionsParam() bool {
	return e.supportsDimensionOverride && e.dimension > 0
}

func (e *AzureOpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}

	reqBody := azureOpenAIEmbedRequest{
		Model:          e.model,
		Input:          SanitizeEmbedInputs(texts),
		EncodingFormat: "float",
	}
	if e.supportsDimensionsParam() {
		reqBody.Dimensions = e.dimension
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
	}

	url := fmt.Sprintf("%s/openai/deployments/%s/embeddings?api-version=%s",
		e.baseURL, e.model, e.apiVersion)

	resp, err := DoRequestWithRetry(ctx, e.httpClient, e.maxRetries, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
		if reqErr != nil {
			return nil, reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("api-key", e.apiKey)
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

	var response openAIEmbedResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	vectors := make([][]float32, 0, len(response.Data))
	for _, item := range response.Data {
		vectors = append(vectors, item.Embedding)
	}
	return vectors, nil
}

func (e *AzureOpenAIEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
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
