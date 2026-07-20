package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// OllamaEmbedder Ollama embedding 实现
type OllamaEmbedder struct {
	baseURL    string
	model      string
	dimension  int
	httpClient *http.Client
}

// NewOllamaEmbedder 创建 Ollama embedder
func NewOllamaEmbedder(cfg *Config) *OllamaEmbedder {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	dimension := cfg.Dimension
	if dimension <= 0 {
		dimension = 384 // Ollama 默认维度
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	return &OllamaEmbedder{
		baseURL:    baseURL,
		model:      cfg.Model,
		dimension:  dimension,
		httpClient: &http.Client{Timeout: time.Duration(timeout) * time.Second},
	}
}

func (e *OllamaEmbedder) Name() string {
	return "ollama"
}

func (e *OllamaEmbedder) Provider() string {
	return "ollama"
}

func (e *OllamaEmbedder) Dimension() int {
	return e.dimension
}

func (e *OllamaEmbedder) Close() error {
	return nil
}

// Embed 批量向量化
func (e *OllamaEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}

	vectors := make([][]float32, 0, len(texts))

	// Ollama 逐个处理文本
	for _, text := range texts {
		text = strings.TrimSpace(text)
		if text == "" {
			text = " "
		}

		// 构建请求
		reqBody := map[string]interface{}{
			"model":  e.model,
			"prompt": text,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
		}

		// 发送请求
		req, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/api/embeddings", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := e.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
		}
		defer resp.Body.Close()

		// 读取响应
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%w: status %d: %s", ErrEmbedFailed, resp.StatusCode, string(respBody))
		}

		// 解析响应
		var result struct {
			Embedding []float32 `json:"embedding"`
		}

		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
		}

		vectors = append(vectors, result.Embedding)
	}

	return vectors, nil
}

// EmbedSingle 单个文本向量化
func (e *OllamaEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
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
