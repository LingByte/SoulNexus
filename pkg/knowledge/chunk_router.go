package knowledge

import (
	"context"
	"errors"
	"strings"
)

// RoutingChunker chooses a chunking strategy based on detected DocumentType.
//
// - Structured: deterministic rule chunking (headings -> paragraphs -> sentences -> fallback)
// - Table/KV: table-preserving record chunking
// - Unstructured: LLM chunking (existing implementation)
type RoutingChunker struct {
	Detector DocumentTypeDetector

	Structured Chunker
	TableKV    Chunker
	LLM        Chunker
}

func (c *RoutingChunker) Provider() string { return "router" }

func (c *RoutingChunker) Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyText
	}
	if c == nil {
		return nil, errors.New("chunker is nil")
	}
	d := c.Detector
	if d == nil {
		d = &RuleBasedDocumentTypeDetector{}
	}
	dt, err := d.DetectDocumentType(ctx, text)
	if err != nil {
		return nil, err
	}
	var ch Chunker
	switch dt {
	case DocumentTypeStructured:
		ch = c.Structured
	case DocumentTypeTableKV:
		ch = c.TableKV
	case DocumentTypeUnstructured:
		ch = c.LLM
		// Without an LLM client, still chunk noisy/OCR-like text via deterministic rules.
		if ch == nil {
			ch = c.Structured
		}
	default:
		ch = c.Structured
	}

	if ch == nil {
		return nil, ErrChunkerNotFound
	}
	out, err := ch.Chunk(ctx, text, opts)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, ErrNoChunks
	}
	for i := range out {
		out[i].Index = i
		out[i].Text = strings.TrimSpace(out[i].Text)
	}
	return out, nil
}

// DefaultRoutingChunker builds a [RoutingChunker] with [RuleBasedDocumentTypeDetector],
// [StructuredRuleChunker], [TableKVChunker], and an optional LLM arm for unstructured text.
// When llm is nil, unstructured documents fall back to structured rule chunking.
func DefaultRoutingChunker(llm Chunker) *RoutingChunker {
	return &RoutingChunker{
		Detector:   &RuleBasedDocumentTypeDetector{},
		Structured: &StructuredRuleChunker{},
		TableKV:    &TableKVChunker{},
		LLM:        llm,
	}
}

var _ Chunker = (*RoutingChunker)(nil)

