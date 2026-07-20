package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/LingByte/SoulNexus/pkg/knowledge/constants"
	llmchunk "github.com/LingByte/lingllm/chunk"
	llmknowledge "github.com/LingByte/lingllm/knowledge"
	llmparser "github.com/LingByte/lingllm/parser"
	"github.com/LingByte/lingllm/utils"
	"strings"
)

// PreviewChunk is one chunk row in the pre-index preview payload.
type PreviewChunk struct {
	Index       int    `json:"index"`
	Title       string `json:"title"`
	Preview     string `json:"preview"`
	Content     string `json:"content"`
	CharCount   int    `json:"charCount"`
	ParentIndex int    `json:"parentIndex,omitempty"`
	Level       string `json:"level,omitempty"`
}

// DocumentPreview is stored on knowledge_documents.preview_json before confirm.
type DocumentPreview struct {
	Mode        string                      `json:"mode"`
	Strategy    string                      `json:"strategy"`
	CharCount   int                         `json:"charCount"`
	ParentCount int                         `json:"parentCount"`
	ChildCount  int                         `json:"childCount"`
	Summary     string                      `json:"summary,omitempty"`
	Parse       *llmparser.ParsePreviewMeta `json:"parse,omitempty"`
	Children    []PreviewChunk              `json:"children"`
	Parents     []PreviewChunk              `json:"parents,omitempty"`
}

// EncodeDocumentPreview serializes preview JSON for DB storage.
func EncodeDocumentPreview(p *DocumentPreview) string {
	if p == nil {
		return "{}"
	}
	b, _ := json.Marshal(p)
	return string(b)
}

// DecodeDocumentPreview parses stored preview JSON.
func DecodeDocumentPreview(raw string) *DocumentPreview {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var p DocumentPreview
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil
	}
	return &p
}

// BuildDocumentPreview produces parent-child or flat chunk preview via lingllm/chunk.
func (s *Service) BuildDocumentPreview(ctx context.Context, title, content string, indexMode string) (*DocumentPreview, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("empty content")
	}
	indexMode = normalizeIndexMode(indexMode, s.cfg.ParentChildEnabled)

	preview, err := s.buildLocalPreview(content, title, indexMode)
	if err != nil {
		return nil, err
	}
	if s.cfg.SummaryIndexEnabled && strings.TrimSpace(preview.Summary) == "" {
		preview.Summary = utils.PreviewText(content, 800)
	}
	return preview, nil
}

func normalizeIndexMode(mode string, parentChildEnabled bool) string {
	mode = strings.TrimSpace(mode)
	if mode == constants.KnowledgeIndexModeFlat {
		return constants.KnowledgeIndexModeFlat
	}
	if !parentChildEnabled {
		return constants.KnowledgeIndexModeFlat
	}
	if mode == "" {
		return constants.KnowledgeIndexModeParentChild
	}
	return mode
}

func (s *Service) buildLocalPreview(content, title, indexMode string) (*DocumentPreview, error) {
	if indexMode == constants.KnowledgeIndexModeFlat {
		chunks, err := s.deps.chunker.Chunk(context.Background(), content, &llmchunk.ChunkOptions{
			MaxChars:      s.cfg.ChildChunkMaxChars,
			MinChars:      s.cfg.ChunkMinChars,
			OverlapChars:  s.cfg.ChunkOverlapChars,
			DocumentTitle: title,
		})
		if err != nil {
			return nil, err
		}
		prev := &DocumentPreview{
			Mode:      constants.KnowledgeIndexModeFlat,
			Strategy:  chunkStrategyLabel(DetectChunkStrategy(context.Background(), content, s.cfg.ChunkLLMEnabled)),
			CharCount: len([]rune(content)),
		}
		for i, ch := range chunks {
			text := strings.TrimSpace(ch.Text)
			prev.Children = append(prev.Children, PreviewChunk{
				Index:     i,
				Title:     ch.Title,
				Preview:   utils.PreviewText(text, chunkPreviewMaxRunes),
				Content:   text,
				CharCount: len([]rune(text)),
				Level:     "child",
			})
		}
		prev.ChildCount = len(prev.Children)
		return prev, nil
	}

	parentCfg := llmchunk.SplitterConfig{
		ChunkSize:    s.cfg.ParentChunkMaxChars,
		ChunkOverlap: int(float64(s.cfg.ParentChunkMaxChars) * 0.1),
		Strategy:     "auto",
	}
	childCfg := llmchunk.SplitterConfig{
		ChunkSize:    s.cfg.ChildChunkMaxChars,
		ChunkOverlap: s.cfg.ChunkOverlapChars,
		Strategy:     "auto",
	}
	res := llmchunk.SplitParentChild(content, parentCfg, childCfg)
	prev := &DocumentPreview{
		Mode:      constants.KnowledgeIndexModeParentChild,
		Strategy:  "parent_child",
		CharCount: len([]rune(content)),
	}
	for i, p := range res.Parents {
		text := strings.TrimSpace(p.Content)
		prev.Parents = append(prev.Parents, PreviewChunk{
			Index:     i,
			Title:     title,
			Preview:   utils.PreviewText(text, chunkPreviewMaxRunes),
			Content:   text,
			CharCount: len([]rune(text)),
			Level:     "parent",
		})
	}
	for _, ch := range res.Children {
		body := strings.TrimSpace(ch.Content)
		embed := strings.TrimSpace(ch.EmbeddingContent())
		prev.Children = append(prev.Children, PreviewChunk{
			Index:       ch.Seq,
			Title:       title,
			Preview:     utils.PreviewText(body, chunkPreviewMaxRunes),
			Content:     embed,
			CharCount:   len([]rune(body)),
			ParentIndex: ch.ParentIndex,
			Level:       "child",
		})
	}
	prev.ParentCount = len(prev.Parents)
	prev.ChildCount = len(prev.Children)
	return prev, nil
}

// PreviewToChunkSummaries converts preview children to stored chunk summaries.
func PreviewToChunkSummaries(p *DocumentPreview) []ChunkSummary {
	if p == nil {
		return nil
	}
	out := make([]ChunkSummary, 0, len(p.Children))
	for _, ch := range p.Children {
		out = append(out, ChunkSummary{
			Index:     ch.Index,
			Title:     ch.Title,
			Preview:   ch.Preview,
			Content:   ch.Content,
			CharCount: ch.CharCount,
		})
	}
	return out
}

// IngestFromPreview indexes a confirmed preview payload (parent-child + optional summary).
func (s *Service) IngestFromPreview(
	ctx context.Context,
	collection, docID, title string,
	preview *DocumentPreview,
	meta DocumentMetadata,
) (*IngestResult, error) {
	if s == nil || s.deps == nil || s.deps.handler == nil {
		return nil, fmt.Errorf("knowledge base service is not initialized")
	}
	if preview == nil || len(preview.Children) == 0 {
		return nil, llmknowledge.ErrNoChunks
	}

	strategy := strings.TrimSpace(preview.Strategy)
	if strategy == "" {
		strategy = "parent_child"
	}

	parentByIndex := map[int]string{}
	for _, p := range preview.Parents {
		parentByIndex[p.Index] = p.Content
	}

	records := make([]llmknowledge.Record, 0, len(preview.Children)+1)
	recordIDs := make([]string, 0, len(preview.Children)+1)

	for _, ch := range preview.Children {
		pointID := chunkPointID(docID, ch.Index)
		recordIDs = append(recordIDs, pointID)

		embedText := strings.TrimSpace(ch.Content)
		returnText := embedText
		if ch.ParentIndex >= 0 {
			if parentText, ok := parentByIndex[ch.ParentIndex]; ok && strings.TrimSpace(parentText) != "" {
				returnText = parentText
			}
		}

		chunkMeta := meta.baseMetadata(docID, strategy, "child", ch.Index, ch.ParentIndex)
		chunkMeta["return_content"] = returnText

		records = append(records, llmknowledge.Record{
			ID:       pointID,
			Source:   docID,
			Title:    ch.Title,
			Content:  embedText,
			Tags:     meta.Tags,
			Metadata: chunkMeta,
		})
	}

	if s.cfg.SummaryIndexEnabled {
		summary := strings.TrimSpace(preview.Summary)
		if summary != "" {
			summaryID := summaryPointID(docID)
			recordIDs = append(recordIDs, summaryID)
			sumMeta := meta.baseMetadata(docID, strategy, "summary", -1, -1)
			records = append(records, llmknowledge.Record{
				ID:       summaryID,
				Source:   docID,
				Title:    title,
				Content:  summary,
				Tags:     meta.Tags,
				Metadata: sumMeta,
			})
		}
	}

	if err := s.deps.handler.Upsert(ctx, records, &llmknowledge.UpsertOptions{
		Namespace: collection,
		Overwrite: true,
		BatchSize: 64,
	}); err != nil {
		return nil, fmt.Errorf("upsert vectors: %w", err)
	}
	if err := s.indexSearchRecords(ctx, collection, records); err != nil {
		_ = s.deps.handler.Delete(ctx, recordIDs, &llmknowledge.DeleteOptions{Namespace: collection})
		return nil, fmt.Errorf("index full-text search: %w", err)
	}

	summaries := PreviewToChunkSummaries(preview)
	return &IngestResult{
		ChunkCount:    len(preview.Children),
		ChunkStrategy: strategy,
		Chunks:        summaries,
		RecordIDs:     recordIDs,
		SummaryText:   preview.Summary,
	}, nil
}

func summaryPointID(docID string) string {
	return chunkPointID(docID, -1)
}

// UpsertChunk embeds one slice and upserts vector + keyword index.
func (s *Service) UpsertChunk(ctx context.Context, collection, docID, recordID, title, content string, chunkIndex int) error {
	if s == nil || s.deps == nil || s.deps.handler == nil {
		return fmt.Errorf("knowledge base service is not initialized")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return llmknowledge.ErrEmptyText
	}
	if strings.TrimSpace(recordID) == "" {
		if docID != "" {
			recordID = chunkPointID(docID, chunkIndex)
		} else {
			return fmt.Errorf("record id is required for manual chunk")
		}
	}
	record := llmknowledge.Record{
		ID:      recordID,
		Source:  docID,
		Title:   strings.TrimSpace(title),
		Content: content,
		Metadata: map[string]any{
			"chunk_index": chunkIndex,
			"doc_id":      docID,
		},
		Tags: []string{},
	}
	if err := s.deps.handler.Upsert(ctx, []llmknowledge.Record{record}, &llmknowledge.UpsertOptions{
		Namespace: collection,
		Overwrite: true,
		BatchSize: 1,
	}); err != nil {
		return fmt.Errorf("upsert chunk vector: %w", err)
	}
	return s.indexSearchRecords(ctx, collection, []llmknowledge.Record{record})
}

// UpsertManualChunk creates a standalone slice with a synthetic doc id.
func (s *Service) UpsertManualChunk(ctx context.Context, collection, manualDocID, recordID, title, content string) (string, error) {
	if strings.TrimSpace(manualDocID) == "" {
		manualDocID = fmt.Sprintf("manual-%s", strings.TrimPrefix(recordID, ""))
	}
	if strings.TrimSpace(recordID) == "" {
		recordID = chunkPointID(manualDocID, 0)
	}
	if err := s.UpsertChunk(ctx, collection, manualDocID, recordID, title, content, 0); err != nil {
		return "", err
	}
	return recordID, nil
}

// DeleteChunk removes one vector point and its keyword index entry.
func (s *Service) DeleteChunk(ctx context.Context, collection string, recordIDs []string) error {
	return s.DeleteVectors(ctx, collection, recordIDs)
}

// ExportChunkRows builds flat rows for Excel/CSV export.
type ExportChunkRow struct {
	DocID      string `json:"docId"`
	ChunkIndex int    `json:"chunkIndex"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	RecordID   string `json:"recordId"`
	SourceType string `json:"sourceType"`
}
