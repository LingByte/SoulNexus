package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

// Default chunk sizing when ChunkOptions fields are zero or unset.
// Smaller chunks tend to improve vector retrieval precision (more segments per document).
const (
	DefaultChunkMaxChars     = 600
	DefaultChunkOverlapChars = 80
	DefaultChunkMinChars     = 40
)

// LLMChunker asks an LLM to return a JSON array of chunks.
type LLMChunker struct {
	LLM   llm.LLMHandler
	Model string
}

func (c *LLMChunker) Provider() string { return "llm" }

func (c *LLMChunker) Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error) {
	_ = ctx
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}
	if c == nil || c.LLM == nil {
		return nil, errors.New("LLM is required")
	}

	cleanOpts := defaultPreChunkCleanOptions(opts)
	text = utils.CleanText(text, cleanOpts)
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}

	model := strings.TrimSpace(c.Model)

	maxChars := DefaultChunkMaxChars
	overlap := DefaultChunkOverlapChars
	minChars := DefaultChunkMinChars
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

	prompt := buildChunkPrompt(text, docTitle, maxChars, overlap, minChars)
	qopts := &llm.QueryOptions{
		Model:                model,
		EnableJSONOutput:     true,
		Temperature:          0.2,
		EnableQueryRewrite:   false,
		EnableQueryExpansion: false,
	}
	resp, err := c.LLM.QueryWithOptions(prompt, qopts)
	if err != nil {
		return nil, err
	}
	if resp == nil || len(resp.Choices) == 0 {
		return nil, errors.New("empty LLM response")
	}
	raw := strings.TrimSpace(resp.Choices[0].Content)

	chunks, err := parseLLMChunks(raw)
	if err != nil {
		return nil, fmt.Errorf("%w (snippet: %s)", err, previewForError(raw, 180))
	}
	for i := range chunks {
		chunks[i].Index = i
		chunks[i].Text = strings.TrimSpace(chunks[i].Text)
	}
	return chunks, nil
}

var _ Chunker = (*LLMChunker)(nil)

func defaultPreChunkCleanOptions(opts *ChunkOptions) *utils.Options {
	if opts != nil && opts.PreChunkClean != nil {
		return opts.PreChunkClean
	}
	return &utils.Options{StripMarkdown: true, DedupLines: true}
}

func buildChunkPrompt(text string, title string, maxChars int, overlap int, minChars int) string {
	titleLine := ""
	if title != "" {
		titleLine = fmt.Sprintf("DocumentTitle: %s\n", title)
	}
	return fmt.Sprintf(`You are a text chunking engine.
%sSplit the input into semantically coherent chunks for retrieval.

STRICT OUTPUT RULES:
- Output MUST be one JSON object and nothing else (no markdown, no code fences, no commentary).
- The JSON object MUST have exactly one top-level key "chunks" whose value is an array.

JSON SCHEMA:
- Output: {"chunks":[{"title": string, "text": string, "metadata"?: object}, ...]}

CHUNKING RULES:
- Each chunk "text" length should be <= %d characters.
- Overlap between consecutive chunks should be about %d characters when helpful.
- Avoid chunks shorter than %d characters unless necessary.
- Prefer more, smaller coherent units (paragraph / list / code block scope) over fewer large chunks, as long as each stays within the max length.

INPUT:
%s
`, titleLine, maxChars, overlap, minChars, text)
}

type llmChunkItem struct {
	Title string         `json:"title"`
	Text  string         `json:"text"`
	Meta  map[string]any `json:"metadata"`
}

func parseLLMChunks(s string) ([]Chunk, error) {
	s = trimLLMJSONResponse(s)
	s = stripMarkdownCodeFence(s)

	// Prefer {"chunks":[...]} (matches response_format json_object on OpenAI-compatible APIs).
	if i := strings.IndexByte(s, '{'); i >= 0 {
		if frag, ok := extractBalancedDelimiters(s, i, '{', '}'); ok {
			var wrapper struct {
				Chunks []llmChunkItem `json:"chunks"`
			}
			if err := json.Unmarshal([]byte(frag), &wrapper); err == nil && len(wrapper.Chunks) > 0 {
				return chunksFromItems(wrapper.Chunks)
			}
		}
	}

	// Legacy: top-level JSON array (or models that ignore json_object).
	if i := strings.IndexByte(s, '['); i >= 0 {
		if frag, ok := extractBalancedDelimiters(s, i, '[', ']'); ok {
			var raw []llmChunkItem
			if err := json.Unmarshal([]byte(frag), &raw); err == nil && len(raw) > 0 {
				return chunksFromItems(raw)
			}
		}
	}

	// Last resort: old heuristic (unsafe if "]" appears inside strings — prefer paths above).
	s2 := extractJSONArrayStringLegacy(s)
	var raw []llmChunkItem
	if err := json.Unmarshal([]byte(s2), &raw); err != nil {
		return nil, fmt.Errorf("parse LLM chunk JSON: %w", err)
	}
	return chunksFromItems(raw)
}

func chunksFromItems(raw []llmChunkItem) ([]Chunk, error) {
	out := make([]Chunk, 0, len(raw))
	for _, r := range raw {
		out = append(out, Chunk{Title: r.Title, Text: r.Text, Metadata: r.Meta})
	}
	if len(out) == 0 {
		return nil, errors.New("no chunks returned")
	}
	return out, nil
}

func previewForError(s string, max int) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func trimLLMJSONResponse(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "\ufeff") {
		s = strings.TrimSpace(s[len("\ufeff"):])
	}
	if len(s) >= 3 && s[0] == '\xef' && s[1] == '\xbb' && s[2] == '\xbf' {
		s = strings.TrimSpace(s[3:])
	}
	return strings.TrimSpace(s)
}

func stripMarkdownCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || !strings.Contains(s, "```") {
		return s
	}
	lower := strings.ToLower(s)
	idx := strings.Index(lower, "```json")
	if idx < 0 {
		idx = strings.Index(lower, "```")
	}
	if idx < 0 {
		return s
	}
	endFence := strings.Index(lower[idx+3:], "```")
	if endFence < 0 {
		return s
	}
	inner := strings.TrimSpace(s[idx+3 : idx+3+endFence])
	innerLower := strings.ToLower(inner)
	if strings.HasPrefix(innerLower, "json") {
		inner = strings.TrimSpace(inner[4:])
	}
	return strings.TrimSpace(inner)
}

// extractBalancedDelimiters returns the substring from s[start] through its matching close byte,
// respecting JSON string escapes so brackets inside "text" fields do not break extraction.
func extractBalancedDelimiters(s string, start int, open, close byte) (string, bool) {
	if start < 0 || start >= len(s) || s[start] != open {
		return "", false
	}
	depth := 1
	inString := false
	escape := false
	for i := start + 1; i < len(s); i++ {
		b := s[i]
		if escape {
			escape = false
			continue
		}
		if inString {
			if b == '\\' {
				escape = true
				continue
			}
			if b == '"' {
				inString = false
			}
			continue
		}
		if b == '"' {
			inString = true
			continue
		}
		if b == open {
			depth++
			continue
		}
		if b == close {
			depth--
			if depth == 0 {
				return s[start : i+1], true
			}
		}
	}
	return "", false
}

func extractJSONArrayStringLegacy(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start >= 0 && end > start {
		return strings.TrimSpace(s[start : end+1])
	}
	return s
}
