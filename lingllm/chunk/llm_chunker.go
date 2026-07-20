package chunk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/LingByte/lingllm/protocol"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

const (
	DefaultLLMChunkMaxChars     = 600
	DefaultLLMChunkOverlapChars = 80
	DefaultLLMChunkMinChars     = 40
)

// LLMChunker 使用 LLM 进行分块
type LLMChunker struct {
	chatModel protocol.ChatModel
	config    *Config
}

// NewLLMChunker 创建 LLM 分块器
func NewLLMChunker(cfg *Config) *LLMChunker {
	var chatModel protocol.ChatModel
	if cfg != nil && cfg.ChatModel != nil {
		if cm, ok := cfg.ChatModel.(protocol.ChatModel); ok {
			chatModel = cm
		}
	}
	return &LLMChunker{
		chatModel: chatModel,
		config:    cfg,
	}
}

func (c *LLMChunker) Provider() string {
	return "llm"
}

func (c *LLMChunker) Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}

	if c.chatModel == nil {
		return nil, errors.New("ChatModel is required")
	}

	// 获取分块参数
	maxChars := DefaultLLMChunkMaxChars
	overlap := DefaultLLMChunkOverlapChars
	minChars := DefaultLLMChunkMinChars
	docTitle := ""

	if opts != nil {
		if opts.MaxChars > 0 {
			maxChars = opts.MaxChars
		}
		if opts.OverlapChars >= 0 {
			overlap = opts.OverlapChars
		}
		if opts.MinChars > 0 {
			minChars = opts.MinChars
		}
		docTitle = strings.TrimSpace(opts.DocumentTitle)
	}

	model, err := c.resolveModel()
	if err != nil {
		return nil, err
	}

	// 构建提示词
	prompt := buildChunkPrompt(text, docTitle, maxChars, overlap, minChars)

	// 调用 LLM
	req := protocol.ChatRequest{
		Model: model,
		Messages: []protocol.Message{
			{
				Role:    protocol.RoleUser,
				Content: prompt,
			},
		},
	}

	resp, err := c.chatModel.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	raw := strings.TrimSpace(resp.FirstContent())
	if raw == "" {
		return nil, errors.New("empty LLM response")
	}

	// 解析 LLM 响应
	chunks, err := parseLLMChunks(raw)
	if err != nil {
		return nil, fmt.Errorf("%w (snippet: %s)", err, previewForError([]byte(raw), 180))
	}

	for i := range chunks {
		chunks[i].Index = i
		chunks[i].Text = strings.TrimSpace(chunks[i].Text)
	}

	return chunks, nil
}

func (c *LLMChunker) resolveModel() (string, error) {
	if c.config != nil {
		if model := strings.TrimSpace(c.config.Model); model != "" {
			return model, nil
		}
	}
	return "", errors.New("Model is required")
}

// buildChunkPrompt 构建分块提示词
func buildChunkPrompt(text, docTitle string, maxChars, overlap, minChars int) string {
	prompt := fmt.Sprintf(`You are a text chunking assistant. Split the following text into logical chunks.

Requirements:
- Each chunk should be approximately %d characters (max)
- Overlap between chunks: %d characters
- Minimum chunk size: %d characters
- Preserve sentence boundaries when possible
- Return a JSON array of chunks with "text" and "title" fields

Document Title: %s

Text to chunk:
%s

Return ONLY valid JSON array, no other text. Example format:
[
  {"text": "chunk content here", "title": "optional title"},
  {"text": "next chunk", "title": "optional title"}
]`, maxChars, overlap, minChars, docTitle, text)

	return prompt
}

// parseLLMChunks 解析 LLM 返回的分块
func parseLLMChunks(raw string) ([]Chunk, error) {
	var parsed []struct {
		Text  string `json:"text"`
		Title string `json:"title"`
	}

	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if len(parsed) == 0 {
		return nil, errors.New("no chunks returned from LLM")
	}

	chunks := make([]Chunk, 0, len(parsed))
	for _, p := range parsed {
		text := strings.TrimSpace(p.Text)
		if text == "" {
			continue
		}

		chunks = append(chunks, Chunk{
			Title:    strings.TrimSpace(p.Title),
			Text:     text,
			Metadata: make(map[string]interface{}),
		})
	}

	if len(chunks) == 0 {
		return nil, errors.New("no valid chunks after filtering")
	}

	return chunks, nil
}

// previewForError 获取错误预览
func previewForError(data []byte, maxLen int) string {
	s := string(data)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
