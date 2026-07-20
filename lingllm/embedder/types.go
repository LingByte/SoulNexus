package embedder

import (
	"context"
	"errors"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Embedder 文本向量化接口
type Embedder interface {
	// Embed 将文本列表转换为向量
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedSingle 将单个文本转换为向量
	EmbedSingle(ctx context.Context, text string) ([]float32, error)

	// Dimension 返回向量维度
	Dimension() int

	// Name 返回 embedder 名称
	Name() string

	// Provider 返回提供商名称
	Provider() string

	// Close 关闭连接
	Close() error
}

// Config 通用 embedder 配置
type Config struct {
	// Provider 提供商名称 (openai, nvidia, ollama, local 等)
	Provider string

	// Model 模型名称
	Model string

	// BaseURL API 基础 URL
	BaseURL string

	// APIKey API 密钥
	APIKey string

	// Dimension 向量维度
	Dimension int

	// BatchSize 批处理大小
	BatchSize int

	// Timeout 超时时间（秒）
	Timeout int

	// MaxRetries 最大重试次数
	MaxRetries int

	// CustomConfig 自定义配置（用于特定提供商）
	CustomConfig map[string]interface{}

	// CustomHeaders 远程 API 调用时附加的自定义 HTTP 请求头
	CustomHeaders map[string]string

	// TruncatePromptTokens 截断 prompt token 数（OpenAI 兼容接口）
	TruncatePromptTokens int

	// SupportsDimensionOverride 是否允许通过 dimensions 参数覆盖输出维度
	SupportsDimensionOverride bool

	// StringInput sends OpenAI-compatible /embeddings "input" as a string instead of
	// a string array. Some gateways (e.g. Gitee AI) reject array input.
	StringInput bool
}

// EmbedderFactory 工厂接口
type EmbedderFactory interface {
	// Create 创建 embedder 实例
	Create(ctx context.Context, cfg *Config) (Embedder, error)

	// Name 返回工厂名称
	Name() string

	// Supports 检查是否支持该提供商
	Supports(provider string) bool
}

var (
	ErrEmptyInput       = errors.New("input is empty")
	ErrInvalidDimension = errors.New("invalid vector dimension")
	ErrInvalidConfig    = errors.New("invalid configuration")
	ErrProviderNotFound = errors.New("provider not found")
	ErrModelNotFound    = errors.New("model not found")
	ErrAPIKeyRequired   = errors.New("API key is required")
	ErrBaseURLRequired  = errors.New("BaseURL is required")
	ErrEmbedFailed      = errors.New("embedding failed")
	ErrConnectionFailed = errors.New("connection failed")
	ErrRateLimited      = errors.New("rate limited")
	ErrInvalidResponse  = errors.New("invalid response")
)

// EmbedResult 向量化结果
type EmbedResult struct {
	Text      string
	Vector    []float32
	Dimension int
	Error     error
}

// BatchEmbedResult 批量向量化结果
type BatchEmbedResult struct {
	Results   []EmbedResult
	Dimension int
	Duration  int64 // 毫秒
}

// EmbedderOptions 向量化选项
type EmbedderOptions struct {
	// BatchSize 批处理大小
	BatchSize int

	// Normalize 是否归一化向量
	Normalize bool

	// ReturnTokens 是否返回 token 数
	ReturnTokens bool

	// Timeout 超时时间（秒）
	Timeout int
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Healthy  bool
	Message  string
	Latency  int64 // 毫秒
	LastErr  error
	Provider string
	Model    string
}
