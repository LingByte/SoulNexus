package chunk

import (
	"context"
	"errors"
	"strings"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// RoutingChunker 路由分块器 - 根据文档类型选择合适的分块策略
type RoutingChunker struct {
	detector   DocumentTypeDetector
	structured Chunker
	tableKV    Chunker
	llm        Chunker
	config     *Config
}

// NewRoutingChunker 创建路由分块器
func NewRoutingChunker(cfg *Config) *RoutingChunker {
	rc := &RoutingChunker{
		config: cfg,
	}

	if cfg != nil {
		rc.detector = cfg.Detector
	}

	// 初始化默认的分块器
	if rc.detector == nil {
		rc.detector = &RuleBasedDocumentTypeDetector{}
	}

	rc.structured = NewStructuredRuleChunker(cfg)
	rc.tableKV = NewTableKVChunker(cfg)

	// 如果配置中有 ChatModel，创建 LLM 分块器
	if cfg != nil && cfg.ChatModel != nil {
		rc.llm = NewLLMChunker(cfg)
	}

	return rc
}

func (c *RoutingChunker) Provider() string {
	return "router"
}

func (c *RoutingChunker) Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}

	if c == nil {
		return nil, errors.New("chunker is nil")
	}

	// 检测文档类型
	detector := c.detector
	if detector == nil {
		detector = &RuleBasedDocumentTypeDetector{}
	}

	docType, err := detector.DetectDocumentType(ctx, text)
	if err != nil {
		// 如果检测失败，使用默认的结构化分块器
		docType = DocumentTypeStructured
	}

	// 根据文档类型选择分块器
	var chunker Chunker
	switch docType {
	case DocumentTypeStructured:
		chunker = c.structured
	case DocumentTypeTableKV:
		chunker = c.tableKV
	case DocumentTypeUnstructured:
		chunker = c.llm
		// 如果没有 LLM 分块器，使用结构化分块器
		if chunker == nil {
			chunker = c.structured
		}
	default:
		chunker = c.structured
	}

	if chunker == nil {
		return nil, ErrChunkerNotFound
	}

	return chunker.Chunk(ctx, text, opts)
}

// RuleBasedDocumentTypeDetector 基于规则的文档类型检测器
type RuleBasedDocumentTypeDetector struct{}

func (d *RuleBasedDocumentTypeDetector) DetectDocumentType(ctx context.Context, text string) (DocumentType, error) {
	if text == "" {
		return DocumentTypeUnknown, ErrEmptyText
	}

	// 检测表格/键值对特征
	if isTableKVDocument(text) {
		return DocumentTypeTableKV, nil
	}

	// 检测结构化特征（标题、段落等）
	if isStructuredDocument(text) {
		return DocumentTypeStructured, nil
	}

	// 默认为非结构化
	return DocumentTypeUnstructured, nil
}

// isTableKVDocument 检测是否为表格/键值对文档
func isTableKVDocument(text string) bool {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return false
	}

	// 检测表格特征：包含 | 符号
	tableCount := 0
	for _, line := range lines {
		if strings.Contains(line, "|") {
			tableCount++
		}
	}

	if tableCount > len(lines)/3 {
		return true
	}

	// 检测键值对特征：包含 : 或 =
	kvCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") || strings.Contains(line, "=") {
			kvCount++
		}
	}

	if kvCount > len(lines)/2 {
		return true
	}

	return false
}

// isStructuredDocument 检测是否为结构化文档
func isStructuredDocument(text string) bool {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return false
	}

	// 检测标题特征：以 # 开头
	headingCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			headingCount++
		}
	}

	if headingCount > 0 {
		return true
	}

	// 检测段落结构：有多个空行分隔
	emptyLineCount := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			emptyLineCount++
		}
	}

	if emptyLineCount > len(lines)/4 {
		return true
	}

	return false
}
