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

// DashScopeEmbedder 阿里 DashScope embedding 实现
type DashScopeEmbedder struct {
	baseURL    string
	apiKey     string
	model      string
	dimension  int
	httpClient *http.Client
}

// NewDashScopeEmbedder 创建 DashScope embedder
func NewDashScopeEmbedder(cfg *Config) *DashScopeEmbedder {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/api/v1"
	}

	dimension := cfg.Dimension
	if dimension <= 0 {
		dimension = 1024 // DashScope 默认维度
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	return &DashScopeEmbedder{
		baseURL:   baseURL,
		apiKey:    cfg.APIKey,
		model:     cfg.Model,
		dimension: dimension,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   16,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				ForceAttemptHTTP2:     true,
			},
		},
	}
}

func (e *DashScopeEmbedder) Name() string {
	return "dashscope"
}

func (e *DashScopeEmbedder) Provider() string {
	return "dashscope"
}

func (e *DashScopeEmbedder) Dimension() int {
	return e.dimension
}

func (e *DashScopeEmbedder) Close() error {
	return nil
}

// Embed 批量向量化
func (e *DashScopeEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}

	// 清理输入
	cleanedTexts := make([]string, 0, len(texts))
	for _, text := range texts {
		text = strings.TrimSpace(text)
		if text == "" {
			text = " " // DashScope 不接受空字符串
		}
		cleanedTexts = append(cleanedTexts, text)
	}

	// 构建请求
	reqBody := map[string]interface{}{
		"model": e.model,
		"input": map[string]interface{}{
			"texts": cleanedTexts,
		},
		"parameters": map[string]interface{}{
			"dimension": e.dimension,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbedFailed, err)
	}

	// 发送请求
	endpoint := e.baseURL + "/services/embeddings/text-embedding/text-embedding"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

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
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, ErrRateLimited
		}
		return nil, fmt.Errorf("%w: status %d: %s", ErrEmbedFailed, resp.StatusCode, string(respBody))
	}

	// 解析响应
	var result struct {
		Output struct {
			Embeddings []struct {
				Embedding []float32 `json:"embedding"`
				TextIndex int       `json:"text_index"`
			} `json:"embeddings"`
		} `json:"output"`
		StatusCode int `json:"status_code"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}

	if result.StatusCode != 200 {
		return nil, fmt.Errorf("%w: status code %d", ErrEmbedFailed, result.StatusCode)
	}

	// 提取向量
	vectors := make([][]float32, len(cleanedTexts))
	for _, item := range result.Output.Embeddings {
		if item.TextIndex < len(vectors) {
			vectors[item.TextIndex] = item.Embedding
		}
	}

	return vectors, nil
}

// EmbedSingle 单个文本向量化
func (e *DashScopeEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
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
