package chunk

import (
	"context"
	"errors"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// DocumentType 文档类型
type DocumentType int

const (
	DocumentTypeUnknown      DocumentType = iota
	DocumentTypeStructured                // 结构化文档（有标题、段落等）
	DocumentTypeTableKV                   // 表格/键值对文档
	DocumentTypeUnstructured              // 非结构化文档（OCR、噪声文本等）
)

// Chunk 分块结果
type Chunk struct {
	Index    int    // 分块索引
	Title    string // 分块标题（可选）
	Text     string // 分块文本
	Metadata map[string]interface{}
}

// ChunkOptions 分块选项
type ChunkOptions struct {
	// MaxChars 目标最大字符数
	MaxChars int
	// OverlapChars 相邻分块的重叠字符数
	OverlapChars int
	// MinChars 最小字符数
	MinChars int
	// DocumentTitle 文档标题
	DocumentTitle string
	// Strategy 分块策略：auto | heading | heuristic | recursive | legacy
	Strategy string
	// PreChunkClean 分块前的清理选项
	PreChunkClean map[string]interface{}
}

// Chunker 分块接口
type Chunker interface {
	// Provider 返回提供商名称
	Provider() string

	// Chunk 将文本分块
	Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error)
}

// DocumentTypeDetector 文档类型检测器
type DocumentTypeDetector interface {
	// DetectDocumentType 检测文档类型
	DetectDocumentType(ctx context.Context, text string) (DocumentType, error)
}

// ChunkerFactory 分块工厂接口
type ChunkerFactory interface {
	// Create 创建分块器
	Create(ctx context.Context, cfg *Config) (Chunker, error)

	// Name 返回工厂名称
	Name() string

	// Supports 检查是否支持该提供商
	Supports(provider string) bool
}

// Config 分块器配置
type Config struct {
	// Provider 提供商名称 (llm, rules_structured, rules_table_kv, router 等)
	Provider string

	// Model LLM 模型名称（仅用于 LLM 分块器）
	Model string

	// ChatModel LLM 聊天模型（仅用于 LLM 分块器）
	ChatModel interface{}

	// Detector 文档类型检测器（仅用于 router 分块器）
	Detector DocumentTypeDetector

	// MaxChars 最大字符数
	MaxChars int

	// MinChars 最小字符数
	MinChars int

	// OverlapChars 重叠字符数
	OverlapChars int

	// Strategy 分块策略：auto | heading | heuristic | recursive | legacy
	Strategy string

	// CustomConfig 自定义配置
	CustomConfig map[string]interface{}
}

var (
	ErrEmptyText        = errors.New("text is empty")
	ErrInvalidChunkOpt  = errors.New("invalid chunk options")
	ErrNoChunks         = errors.New("no chunks produced")
	ErrChunkerNotFound  = errors.New("chunker not found")
	ErrProviderNotFound = errors.New("provider not found")
	ErrInvalidConfig    = errors.New("invalid configuration")
	ErrChunkFailed      = errors.New("chunking failed")
	ErrDetectionFailed  = errors.New("document type detection failed")
)

// ChunkResult 分块结果
type ChunkResult struct {
	Chunks       []Chunk
	DocumentType DocumentType
	Duration     int64 // 毫秒
}

// ChunkStats 分块统计
type ChunkStats struct {
	TotalChunks  int
	TotalChars   int
	AvgChunkSize int
	MinChunkSize int
	MaxChunkSize int
	OverlapChars int
	DocumentType DocumentType
}
