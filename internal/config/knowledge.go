// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/LingByte/SoulNexus/pkg/utils"
	lingchunk "github.com/LingByte/lingllm/chunk"
	lingembedder "github.com/LingByte/lingllm/embedder"
	lingknowledge "github.com/LingByte/lingllm/knowledge"
	"github.com/LingByte/lingllm/protocol"
	_ "github.com/LingByte/lingllm/protocol/openai"
)

// ---------------------------------------------------------------------------
// chunk: document chunking helpers
// ---------------------------------------------------------------------------

const llmChunkInputBatchRunes = 8000

// Default knowledge chunk target (~800–1000 Chinese chars per segment for RAG).
const defaultKnowledgeChunkMaxChars = 900

// ChunkMaxCharsFromEnv returns max characters per chunk (CHUNK_MAX_CHARS, default 900).
func ChunkMaxCharsFromEnv() int {
	raw := strings.TrimSpace(utils.GetEnv("CHUNK_MAX_CHARS"))
	if raw == "" {
		return defaultKnowledgeChunkMaxChars
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 200 {
		return defaultKnowledgeChunkMaxChars
	}
	if n > 4000 {
		return 4000
	}
	return n
}

// chunkModelOverride forces CHUNK_LLM_MODEL on each chat call (lingllm LLMChunker defaults to gpt-4).
type chunkModelOverride struct {
	inner protocol.ChatModel
	model string
}

func (m *chunkModelOverride) Name() string {
	if m == nil || m.inner == nil {
		return "chunk_llm"
	}
	return m.inner.Name()
}

func (m *chunkModelOverride) Chat(ctx context.Context, req protocol.ChatRequest) (*protocol.ChatResponse, error) {
	if m == nil || m.inner == nil {
		return nil, errors.New("chunk llm client is nil")
	}
	if m.model != "" {
		req.Model = m.model
	}
	return m.inner.Chat(ctx, req)
}

func (m *chunkModelOverride) StreamChat(ctx context.Context, req protocol.ChatRequest) (protocol.ChatStream, error) {
	if m == nil || m.inner == nil {
		return nil, errors.New("chunk llm client is nil")
	}
	if m.model != "" {
		req.Model = m.model
	}
	return m.inner.StreamChat(ctx, req)
}

// semanticLLMChunker uses an optimized prompt for RAG-friendly, self-contained segments.
type semanticLLMChunker struct {
	chat  protocol.ChatModel
	model string
}

func (c *semanticLLMChunker) Provider() string { return "semantic_llm" }

func (c *semanticLLMChunker) Chunk(ctx context.Context, text string, opts *lingchunk.ChunkOptions) ([]lingchunk.Chunk, error) {
	if c == nil || c.chat == nil {
		return nil, errors.New("semantic chunk llm client is nil")
	}
	text = strings.TrimSpace(strings.ToValidUTF8(text, ""))
	if text == "" {
		return nil, lingchunk.ErrEmptyText
	}

	maxChars := ChunkMaxCharsFromEnv()
	minChars := DefaultChunkMinChars
	docTitle := ""
	if opts != nil {
		if opts.MaxChars > 0 {
			maxChars = opts.MaxChars
		}
		if opts.MinChars > 0 {
			minChars = opts.MinChars
		}
		docTitle = strings.TrimSpace(opts.DocumentTitle)
	}

	batches := splitTextForLLMInput(text, llmChunkInputBatchRunes)
	out := make([]lingchunk.Chunk, 0, 16)
	idx := 0
	for _, batch := range batches {
		batchChunks, err := c.chunkBatch(ctx, batch, docTitle, maxChars, minChars)
		if err != nil {
			return nil, err
		}
		for _, ch := range batchChunks {
			txt := strings.TrimSpace(strings.ToValidUTF8(ch.Text, ""))
			if txt == "" || utf8.RuneCountInString(txt) < minChars {
				continue
			}
			ch.Index = idx
			ch.Text = txt
			idx++
			out = append(out, ch)
		}
	}
	if len(out) == 0 {
		return nil, lingchunk.ErrNoChunks
	}
	return out, nil
}

func (c *semanticLLMChunker) chunkBatch(ctx context.Context, text, docTitle string, maxChars, minChars int) ([]lingchunk.Chunk, error) {
	prompt := buildSemanticChunkPrompt(text, docTitle, maxChars, minChars)
	req := protocol.ChatRequest{
		Model: c.model,
		Messages: []protocol.Message{{
			Role:    protocol.RoleUser,
			Content: prompt,
		}},
	}
	resp, err := c.chat.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(resp.FirstContent())
	if raw == "" {
		return nil, errors.New("empty LLM chunk response")
	}
	return parseSemanticChunkJSON(raw)
}

func buildSemanticChunkPrompt(text, docTitle string, maxChars, minChars int) string {
	titleLine := docTitle
	if titleLine == "" {
		titleLine = "（未提供）"
	}
	return fmt.Sprintf(`你是通用知识库RAG文档分块处理专家，负责把原始文档切割为适配向量检索的独立语义片段，兼顾检索召回精度与片段可读性，无需过度精细拆分。

## 硬性约束（必须严格遵守）
1. 分块逻辑：优先按完整语义单元划分（段落、章节、主题模块、完整事件），禁止固定字数暴力截断；允许同一大类主题合并，不强制拆分成极小单元。
2. 完整性要求：每一段落语义闭环、能单独理解，不可拆分完整句子、专有名词、数字、专业术语，中文汉字不中途切断。
3. 字符长度限制：单片段总字符上限 %d（汉字、标点均计1字符），推荐单块合理区间500–1200字符；仅当单一主题远超上限时，才按子话题拆分，短主题尽量合并。
4. 内容去重规则：片段间不重复堆砌相同原文，不设置文本重叠；相近弱关联内容可整合为一块。
5. 原文保真：完整保留全部原始信息、数据、专有名词、编号、时间，禁止改写、删减、概括压缩原文内容。
6. 片段标题规范：每个片段配简短中文标题，概括整块核心主题，无需细分到极小条目。

## 输入信息
文档总标题：%s
完整文档正文：
%s

## 输出要求
仅输出标准JSON数组，禁止附带任何额外解释、注释、markdown标记、多余文字。
输出格式固定：
[{"text":"当前片段完整原文","title":"片段简短主题标题"}]`, maxChars, titleLine, text)
}
func parseSemanticChunkJSON(raw string) ([]lingchunk.Chunk, error) {
	raw = extractJSONPayload(raw)
	var parsed []struct {
		Text  string `json:"text"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("parse LLM chunk JSON: %w", err)
	}
	if len(parsed) == 0 {
		return nil, errors.New("LLM returned empty chunk array")
	}
	out := make([]lingchunk.Chunk, 0, len(parsed))
	for _, p := range parsed {
		txt := strings.TrimSpace(strings.ToValidUTF8(p.Text, ""))
		if txt == "" {
			continue
		}
		out = append(out, lingchunk.Chunk{
			Title:    strings.TrimSpace(p.Title),
			Text:     txt,
			Metadata: map[string]interface{}{"chunker": "semantic_llm"},
		})
	}
	if len(out) == 0 {
		return nil, errors.New("no valid chunks after parsing LLM response")
	}
	return out, nil
}

func extractJSONPayload(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSpace(raw)
		if i := strings.LastIndex(raw, "```"); i >= 0 {
			raw = strings.TrimSpace(raw[:i])
		}
	}
	// Trim to outermost JSON array if model added prose.
	if i := strings.Index(raw, "["); i > 0 {
		raw = raw[i:]
	}
	if j := strings.LastIndex(raw, "]"); j >= 0 && j < len(raw)-1 {
		raw = raw[:j+1]
	}
	return raw
}

func splitTextForLLMInput(text string, maxRunes int) []string {
	text = strings.TrimSpace(text)
	if text == "" || maxRunes <= 0 {
		return nil
	}
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return []string{text}
	}
	var batches []string
	start := 0
	for start < len(runes) {
		end := start + maxRunes
		if end >= len(runes) {
			batches = append(batches, string(runes[start:]))
			break
		}
		// Prefer breaking at paragraph boundary in the tail window.
		cut := end
		for i := end - 1; i > start+maxRunes/2; i-- {
			if runes[i] == '\n' && i+1 < len(runes) && runes[i+1] == '\n' {
				cut = i + 2
				break
			}
		}
		batches = append(batches, string(runes[start:cut]))
		start = cut
	}
	return batches
}

// ChunkLLMFromEnv builds an optional ChatModel for document chunking.
// Env: CHUNK_LLM_PROVIDER, CHUNK_LLM_BASEURL, CHUNK_LLM_APIKEY, CHUNK_LLM_MODEL.
func ChunkLLMFromEnv() (protocol.ChatModel, string, error) {
	provider := strings.TrimSpace(strings.ToLower(utils.GetEnv("CHUNK_LLM_PROVIDER")))
	if provider == "" {
		return nil, "", nil
	}
	apiKey := strings.TrimSpace(utils.GetEnv("CHUNK_LLM_APIKEY"))
	baseURL := strings.TrimSpace(utils.GetEnv("CHUNK_LLM_BASEURL"))
	model := strings.TrimSpace(utils.GetEnv("CHUNK_LLM_MODEL"))

	switch provider {
	case "openai":
		if apiKey == "" {
			return nil, "", errors.New("CHUNK_LLM_APIKEY is required when CHUNK_LLM_PROVIDER is set")
		}
		client, err := protocol.NewClient(protocol.ClientConfig{
			Provider: protocol.ProviderOpenAI,
			APIKey:   apiKey,
			BaseURL:  baseURL,
		})
		if err != nil {
			return nil, "", fmt.Errorf("chunk llm client: %w", err)
		}
		if model == "" {
			model = "gpt-4"
		}
		return &chunkModelOverride{inner: client, model: model}, model, nil
	case "openai-response":
		if apiKey == "" {
			return nil, "", errors.New("CHUNK_LLM_APIKEY is required when CHUNK_LLM_PROVIDER is set")
		}
		client, err := protocol.NewClient(protocol.ClientConfig{
			Provider: protocol.ProviderOpenAIResponse,
			APIKey:   apiKey,
			BaseURL:  baseURL,
		})
		if err != nil {
			return nil, "", fmt.Errorf("chunk llm client: %w", err)
		}
		if model == "" {
			model = "gpt-4"
		}
		return &chunkModelOverride{inner: client, model: model}, model, nil
	default:
		return nil, "", fmt.Errorf("unsupported CHUNK_LLM_PROVIDER %q (use openai)", provider)
	}
}

// ChunkerFromEnv returns semantic LLM chunker when CHUNK_LLM_* is set; otherwise rule-based routing.
func ChunkerFromEnv() (lingchunk.Chunker, error) {
	chatModel, model, err := ChunkLLMFromEnv()
	if err != nil {
		return nil, err
	}
	if chatModel != nil {
		return &semanticLLMChunker{chat: chatModel, model: model}, nil
	}
	cfg := &lingchunk.Config{
		MaxChars:     ChunkMaxCharsFromEnv(),
		MinChars:     DefaultChunkMinChars,
		OverlapChars: DefaultChunkOverlapChars,
	}
	return lingchunk.NewRoutingChunker(cfg), nil
}

// ---------------------------------------------------------------------------
// embed: embedding helpers
// ---------------------------------------------------------------------------

const (
	defaultEmbedMaxInputChars = 12000
	defaultEmbedBatchSize     = 16
)

// EmbedMaxInputChars caps each embedding request body (EMBED_MAX_INPUT_CHARS).
func EmbedMaxInputChars() int {
	raw := strings.TrimSpace(utils.GetEnv("EMBED_MAX_INPUT_CHARS"))
	if raw == "" {
		return defaultEmbedMaxInputChars
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultEmbedMaxInputChars
	}
	return n
}

// EmbedBatchSize is how many texts to send per Embed call (EMBED_BATCH_SIZE).
func EmbedBatchSize() int {
	raw := strings.TrimSpace(utils.GetEnv("EMBED_BATCH_SIZE"))
	if raw == "" {
		return defaultEmbedBatchSize
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultEmbedBatchSize
	}
	return n
}

// ClipEmbedInput shortens a single string before embedding (by runes, safe for UTF-8).
func ClipEmbedInput(text string, maxChars int) string {
	text = strings.TrimSpace(text)
	if maxChars <= 0 || len(text) <= maxChars {
		return text
	}
	r := []rune(text)
	if len(r) <= maxChars {
		return text
	}
	return string(r[:maxChars])
}

// EmbedTextsBatched calls the embedder in batches; each input is clipped to EmbedMaxInputChars.
func EmbedTextsBatched(ctx context.Context, emb lingembedder.Embedder, inputs []string) ([][]float32, error) {
	if emb == nil {
		return nil, fmt.Errorf("embedder is nil")
	}
	if len(inputs) == 0 {
		return nil, fmt.Errorf("no texts to embed")
	}
	maxChars := EmbedMaxInputChars()
	batchSize := EmbedBatchSize()
	out := make([][]float32, 0, len(inputs))
	for start := 0; start < len(inputs); start += batchSize {
		end := start + batchSize
		if end > len(inputs) {
			end = len(inputs)
		}
		batch := make([]string, end-start)
		for i, in := range inputs[start:end] {
			batch[i] = ClipEmbedInput(in, maxChars)
		}
		vecs, err := emb.Embed(ctx, batch)
		if err != nil {
			return nil, err
		}
		if len(vecs) != len(batch) {
			return nil, fmt.Errorf("embedder returned %d vectors for %d inputs", len(vecs), len(batch))
		}
		out = append(out, vecs...)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// qdrant: vector collection helpers
// ---------------------------------------------------------------------------

// QdrantCollectionVectorDim reads the vector size of an existing Qdrant collection (0 if missing).
func QdrantCollectionVectorDim(ctx context.Context, qh *lingknowledge.QdrantHandler, collection string) (int, error) {
	if qh == nil {
		return 0, fmt.Errorf("qdrant handler is nil")
	}
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return 0, fmt.Errorf("collection name is empty")
	}
	base := strings.TrimRight(strings.TrimSpace(qh.BaseURL), "/")
	if base == "" {
		return 0, fmt.Errorf("qdrant base URL is empty")
	}
	reqURL := base + "/collections/" + url.PathEscape(collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(qh.APIKey) != "" {
		req.Header.Set("api-key", strings.TrimSpace(qh.APIKey))
	}
	cl := qh.HTTPClient
	if cl == nil {
		cl = http.DefaultClient
	}
	resp, err := cl.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("qdrant get collection: status=%d body=%s", resp.StatusCode, truncateErrBody(body))
	}
	var parsed struct {
		Result struct {
			Config struct {
				Params struct {
					Vectors struct {
						Size int `json:"size"`
					} `json:"vectors"`
				} `json:"params"`
			} `json:"config"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}
	return parsed.Result.Config.Params.Vectors.Size, nil
}

func truncateErrBody(b []byte) string {
	const max = 512
	s := strings.TrimSpace(string(b))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
