package knowledge

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	llmchunk "github.com/LingByte/lingllm/chunk"
	llmknowledge "github.com/LingByte/lingllm/knowledge"
)

const chunkPreviewMaxRunes = 240

// ChunkSummary is stored on the document row (preview + full text for detail view).
type ChunkSummary struct {
	Index     int    `json:"index"`
	Title     string `json:"title"`
	Preview   string `json:"preview"`
	Content   string `json:"content,omitempty"`
	CharCount int    `json:"charCount"`
}

// DetectChunkStrategy mirrors lingllm router document-type routing.
func DetectChunkStrategy(ctx context.Context, text string, llmChunkEnabled bool) string {
	detector := &llmchunk.RuleBasedDocumentTypeDetector{}
	docType, err := detector.DetectDocumentType(ctx, text)
	if err != nil {
		return "rules_structured"
	}
	switch docType {
	case llmchunk.DocumentTypeTableKV:
		return "rules_table_kv"
	case llmchunk.DocumentTypeUnstructured:
		if llmChunkEnabled {
			return "llm"
		}
		return "rules_structured"
	case llmchunk.DocumentTypeStructured:
		return "rules_structured"
	default:
		return "rules_structured"
	}
}

func chunkStrategyLabel(strategy string) string {
	switch strings.TrimSpace(strategy) {
	case "rules_table_kv":
		return "rules_table_kv"
	case "llm":
		return "llm"
	case "router":
		return "router"
	default:
		return "rules_structured"
	}
}

// EncodeChunkSummaries serializes chunk previews for DB storage.
func EncodeChunkSummaries(items []ChunkSummary) string {
	if len(items) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(items)
	return string(b)
}

// ChunkSummaryByIndex finds one stored chunk by index.
func ChunkSummaryByIndex(raw string, index int) (ChunkSummary, bool) {
	for _, ch := range DecodeChunkSummaries(raw) {
		if ch.Index == index {
			return ch, true
		}
	}
	return ChunkSummary{}, false
}

// DecodeChunkSummaries parses stored chunk previews.
func DecodeChunkSummaries(raw string) []ChunkSummary {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var items []ChunkSummary
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}

// DocumentMetadata carries filterable fields stored on vector records.
type DocumentMetadata struct {
	DocType     string
	Tags        []string
	CampaignID  string
	ProductLine string
	CreatedAt   time.Time
}

// EncodeTagsJSON serializes tags for DB storage.
func EncodeTagsJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(tags)
	return string(b)
}

// DecodeTagsJSON parses stored tags.
func DecodeTagsJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(raw), &tags); err != nil {
		return nil
	}
	return tags
}

func (m DocumentMetadata) baseMetadata(docID, strategy, chunkLevel string, chunkIndex int, parentIndex int) map[string]any {
	meta := map[string]any{
		"chunk_index":    chunkIndex,
		"doc_id":         docID,
		"chunk_strategy": strategy,
		"chunk_level":    chunkLevel,
	}
	if parentIndex >= 0 {
		meta["parent_index"] = parentIndex
	}
	if v := strings.TrimSpace(m.DocType); v != "" {
		meta["doc_type"] = v
	}
	if v := strings.TrimSpace(m.CampaignID); v != "" {
		meta["campaign_id"] = v
	}
	if v := strings.TrimSpace(m.ProductLine); v != "" {
		meta["product_line"] = v
	}
	if len(m.Tags) > 0 {
		meta["tags"] = m.Tags
	}
	if !m.CreatedAt.IsZero() {
		meta["created_at"] = m.CreatedAt.UTC().Format(time.RFC3339)
	}
	return meta
}

// RecallFilter is the API-facing metadata filter for knowledge recall.
type RecallFilter struct {
	DocIDs      []string
	DocTypes    []string
	Tags        []string
	CampaignID  string
	ProductLine string
	CreatedFrom string
	CreatedTo   string
}

// ToQueryFilters converts RecallFilter to lingllm vector filters.
func (f RecallFilter) ToQueryFilters() []llmknowledge.Filter {
	if f.isEmpty() {
		return nil
	}
	var filters []llmknowledge.Filter
	if len(f.DocIDs) == 1 {
		filters = append(filters, llmknowledge.Filter{
			Field: "doc_id", Operator: llmknowledge.FilterOpEqual, Value: []any{f.DocIDs[0]},
		})
	} else if len(f.DocIDs) > 1 {
		vals := make([]any, 0, len(f.DocIDs))
		for _, id := range f.DocIDs {
			vals = append(vals, id)
		}
		filters = append(filters, llmknowledge.Filter{
			Field: "doc_id", Operator: llmknowledge.FilterOpIn, Value: vals,
		})
	}
	if len(f.DocTypes) == 1 {
		filters = append(filters, llmknowledge.Filter{
			Field: "doc_type", Operator: llmknowledge.FilterOpEqual, Value: []any{f.DocTypes[0]},
		})
	} else if len(f.DocTypes) > 1 {
		vals := make([]any, 0, len(f.DocTypes))
		for _, v := range f.DocTypes {
			vals = append(vals, v)
		}
		filters = append(filters, llmknowledge.Filter{
			Field: "doc_type", Operator: llmknowledge.FilterOpIn, Value: vals,
		})
	}
	if len(f.Tags) > 0 {
		vals := make([]any, 0, len(f.Tags))
		for _, t := range f.Tags {
			vals = append(vals, t)
		}
		filters = append(filters, llmknowledge.Filter{
			Field: "tags", Operator: llmknowledge.FilterOpContainsAny, Value: vals,
		})
	}
	if v := strings.TrimSpace(f.CampaignID); v != "" {
		filters = append(filters, llmknowledge.Filter{
			Field: "campaign_id", Operator: llmknowledge.FilterOpEqual, Value: []any{v},
		})
	}
	if v := strings.TrimSpace(f.ProductLine); v != "" {
		filters = append(filters, llmknowledge.Filter{
			Field: "product_line", Operator: llmknowledge.FilterOpEqual, Value: []any{v},
		})
	}
	if v := strings.TrimSpace(f.CreatedFrom); v != "" {
		filters = append(filters, llmknowledge.Filter{
			Field: "created_at", Operator: llmknowledge.FilterOpGte, Value: []any{v},
		})
	}
	if v := strings.TrimSpace(f.CreatedTo); v != "" {
		filters = append(filters, llmknowledge.Filter{
			Field: "created_at", Operator: llmknowledge.FilterOpLte, Value: []any{v},
		})
	}
	return filters
}

func (f RecallFilter) isEmpty() bool {
	return len(f.DocIDs) == 0 && len(f.DocTypes) == 0 && len(f.Tags) == 0 &&
		strings.TrimSpace(f.CampaignID) == "" && strings.TrimSpace(f.ProductLine) == "" &&
		strings.TrimSpace(f.CreatedFrom) == "" && strings.TrimSpace(f.CreatedTo) == ""
}

const maxIndexErrorLen = 1024

// FormatIndexError turns an ingest/index error into a user-facing message.
func FormatIndexError(err error) string {
	msg := "index failed"
	if err != nil {
		msg = strings.TrimSpace(err.Error())
	}
	msg = shortenEmbedAPIError(msg)
	if len(msg) > maxIndexErrorLen {
		msg = msg[:maxIndexErrorLen]
	}
	return msg
}

func shortenEmbedAPIError(msg string) string {
	const marker = `{"error":`
	idx := strings.Index(msg, marker)
	if idx < 0 {
		return msg
	}
	var payload struct {
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(msg[idx:]), &payload); err != nil {
		return msg
	}
	if payload.Error.Message != "" {
		if payload.Error.Code != "" && payload.Error.Code != payload.Error.Message {
			return payload.Error.Code + ": " + payload.Error.Message
		}
		return payload.Error.Message
	}
	return msg
}
