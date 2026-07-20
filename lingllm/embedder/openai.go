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

// OpenAIEmbedder OpenAI embedding 实现
type OpenAIEmbedder struct {
	baseURL                   string
	apiKey                    string
	model                     string
	providerName              string
	dimension                 int
	truncatePromptTokens      int
	supportsDimensionOverride bool
	stringInput               bool
	customHeaders             map[string]string
	httpClient                *http.Client
	maxRetries                int
}

type openAIEmbedRequest struct {
	Model                string   `json:"model"`
	Input                []string `json:"input"`
	EncodingFormat       string   `json:"encoding_format,omitempty"`
	Dimensions           int      `json:"dimensions,omitempty"`
	TruncatePromptTokens int      `json:"truncate_prompt_tokens,omitempty"`
}

type openAIEmbedStringRequest struct {
	Model      string `json:"model"`
	Input      string `json:"input"`
	Dimensions int    `json:"dimensions,omitempty"`
}

type openAIEmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

// NewOpenAIEmbedder 创建 OpenAI embedder
func NewOpenAIEmbedder(cfg *Config) *OpenAIEmbedder {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	dimension := cfg.Dimension
	if dimension <= 0 {
		dimension = 1536
	}

	stringInput := cfg.StringInput || CustomConfigBool(cfg, "string_input") || OpenAICompatStringInputBaseURL(baseURL)

	return &OpenAIEmbedder{
		baseURL:                   baseURL,
		apiKey:                    cfg.APIKey,
		model:                     cfg.Model,
		providerName:              providerNameFromConfig(cfg),
		dimension:                 dimension,
		truncatePromptTokens:      truncatePromptTokensFromConfig(cfg),
		supportsDimensionOverride: cfg.SupportsDimensionOverride,
		stringInput:               stringInput,
		customHeaders:             cfg.CustomHeaders,
		httpClient:                HTTPClientFromConfig(cfg),
		maxRetries:                maxRetriesFromConfig(cfg),
	}
}

func (e *OpenAIEmbedder) Name() string {
	return e.providerName
}

func (e *OpenAIEmbedder) Provider() string {
	return e.providerName
}

func (e *OpenAIEmbedder) Dimension() int {
	return e.dimension
}

func (e *OpenAIEmbedder) Close() error {
	return nil
}

func (e *OpenAIEmbedder) supportsDimensionsParam() bool {
	return e.supportsDimensionOverride && e.dimension > 0
}

// Embed 批量向量化
func (e *OpenAIEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}

	if e.stringInput {
		inputs := SanitizeEmbedInputs(texts)
		vectors := make([][]float32, 0, len(inputs))
		for _, text := range inputs {
			vec, err := e.embedStringInput(ctx, text)
			if err != nil {
				return nil, err
			}
			vectors = append(vectors, vec)
		}
		return vectors, nil
	}

	reqBody := openAIEmbedRequest{
		Model:                e.model,
		Input:                SanitizeEmbedInputs(texts),
		EncodingFormat:       "float",
		TruncatePromptTokens: e.truncatePromptTokens,
	}
	if e.supportsDimensionsParam() {
		reqBody.Dimensions = e.dimension
	}

	return e.doEmbedRequest(ctx, reqBody)
}

// EmbedSingle 单个文本向量化
func (e *OpenAIEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyInput
	}

	if e.stringInput {
		return e.embedStringInput(ctx, text)
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

func (e *OpenAIEmbedder) embedStringInput(ctx context.Context, text string) ([]float32, error) {
	reqBody := openAIEmbedStringRequest{
		Model: e.model,
		Input: text,
	}
	if e.supportsDimensionsParam() {
		reqBody.Dimensions = e.dimension
	}

	vectors, err := e.doEmbedRequest(ctx, reqBody)
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, ErrInvalidResponse
	}
	return vectors[0], nil
}

func (e *OpenAIEmbedder) doEmbedRequest(ctx context.Context, reqBody any) ([][]float32, error) {
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, ErrRateLimited
		}
		return nil, fmt.Errorf("%w: status %d: %s", ErrEmbedFailed, resp.StatusCode, string(respBody))
	}

	var result openAIEmbedResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	vectors := make([][]float32, 0, len(result.Data))
	for _, item := range result.Data {
		vectors = append(vectors, item.Embedding)
	}
	return vectors, nil
}
