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

const (
	maxEmbedInputChars  = 12000
	maxEmbedBatchInputs = 16
)

// NvidiaEmbedder Nvidia embedding 实现
type NvidiaEmbedder struct {
	baseURL                   string
	apiKey                    string
	model                     string
	inputKey                  string
	embeddingsPath            string
	dimension                 int
	supportsDimensionOverride bool
	customHeaders             map[string]string
	httpClient                *http.Client
	maxRetries                int
}

// NewNvidiaEmbedder 创建 Nvidia embedder
func NewNvidiaEmbedder(cfg *Config) *NvidiaEmbedder {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.nvcf.nvidia.com/v2"
	}

	dimension := cfg.Dimension
	if dimension <= 0 {
		dimension = 1024 // Nvidia 默认维度
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	inputKey := "input"
	embeddingsPath := ""
	if cfg.CustomConfig != nil {
		if customCfg, ok := cfg.CustomConfig["input_key"].(string); ok {
			inputKey = customCfg
		}
		if customCfg, ok := cfg.CustomConfig["embeddings_path"].(string); ok {
			embeddingsPath = customCfg
		}
	}

	return &NvidiaEmbedder{
		baseURL:                   baseURL,
		apiKey:                    cfg.APIKey,
		model:                     cfg.Model,
		inputKey:                  inputKey,
		embeddingsPath:            embeddingsPath,
		dimension:                 dimension,
		supportsDimensionOverride: cfg.SupportsDimensionOverride,
		customHeaders:             cfg.CustomHeaders,
		httpClient:                HTTPClientFromConfig(cfg),
		maxRetries:                maxRetries,
	}
}

func (e *NvidiaEmbedder) Name() string {
	return "nvidia"
}

func (e *NvidiaEmbedder) Provider() string {
	return "nvidia"
}

func (e *NvidiaEmbedder) Dimension() int {
	return e.dimension
}

func (e *NvidiaEmbedder) Close() error {
	return nil
}

func (e *NvidiaEmbedder) supportsDimensionsParam() bool {
	return e.supportsDimensionOverride && e.dimension > 0
}

// Embed 批量向量化
func (e *NvidiaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}

	sanitized := SanitizeEmbedInputs(texts)

	// 构建端点
	endpoint := e.buildEndpoint()

	// 批量处理
	var allVectors [][]float32

	for start := 0; start < len(sanitized); start += maxEmbedBatchInputs {
		end := start + maxEmbedBatchInputs
		if end > len(sanitized) {
			end = len(sanitized)
		}
		batch := sanitized[start:end]

		vectors, err := e.embedBatch(ctx, endpoint, batch)
		if err != nil {
			return nil, err
		}

		allVectors = append(allVectors, vectors...)
	}

	return allVectors, nil
}

// EmbedSingle 单个文本向量化
func (e *NvidiaEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
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

// buildEndpoint 构建 API 端点
func (e *NvidiaEmbedder) buildEndpoint() string {
	endpoint := e.baseURL
	if strings.TrimSpace(e.embeddingsPath) != "" {
		p := strings.TrimSpace(e.embeddingsPath)
		p = strings.TrimLeft(p, "/")
		endpoint += "/" + p
	} else {
		if !strings.HasSuffix(endpoint, "/embeddings") {
			endpoint += "/embeddings"
		}
	}
	return endpoint
}

// embedBatch 处理单个批次
func (e *NvidiaEmbedder) embedBatch(ctx context.Context, endpoint string, batch []string) ([][]float32, error) {
	body := map[string]interface{}{
		"model":           e.model,
		e.inputKey:        batch,
		"encoding_format": "float",
		"input_type":      "passage",
	}
	if e.supportsDimensionsParam() {
		body["dimensions"] = e.dimension
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
	}

	resp, err := DoRequestWithRetry(ctx, e.httpClient, e.maxRetries, func() (*http.Request, error) {
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, ErrRateLimited
		}
		return nil, fmt.Errorf("%w: status %d: %s", ErrEmbedFailed, resp.StatusCode, truncateForError(respBody, 200))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("%w: no embeddings returned", ErrInvalidResponse)
	}

	vectors := make([][]float32, 0, len(result.Data))
	for _, item := range result.Data {
		if len(item.Embedding) == 0 {
			return nil, fmt.Errorf("%w: empty embedding returned", ErrInvalidResponse)
		}
		vectors = append(vectors, item.Embedding)
	}
	return vectors, nil
}

// truncateForError 截断错误消息
func truncateForError(data []byte, maxLen int) string {
	s := string(data)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
