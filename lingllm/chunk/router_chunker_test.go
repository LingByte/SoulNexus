package chunk

import (
	"context"
	"testing"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestNewRoutingChunker(t *testing.T) {
	cfg := &Config{
		Provider: "router",
	}

	chunker := NewRoutingChunker(cfg)

	if chunker == nil {
		t.Fatal("expected chunker, got nil")
	}

	if chunker.config != cfg {
		t.Error("expected config to be set")
	}

	if chunker.detector == nil {
		t.Error("expected detector to be initialized")
	}
}

func TestRoutingChunker_Provider(t *testing.T) {
	chunker := NewRoutingChunker(&Config{})

	if chunker.Provider() != "router" {
		t.Errorf("expected provider 'router', got '%s'", chunker.Provider())
	}
}

func TestRoutingChunker_ChunkEmptyText(t *testing.T) {
	chunker := NewRoutingChunker(&Config{})

	_, err := chunker.Chunk(context.Background(), "", nil)
	if err != ErrEmptyText {
		t.Errorf("expected ErrEmptyText, got %v", err)
	}
}

func TestRoutingChunker_ChunkNil(t *testing.T) {
	var chunker *RoutingChunker

	_, err := chunker.Chunk(context.Background(), "test", nil)
	if err == nil {
		t.Error("expected error for nil chunker")
	}
}

func TestRoutingChunker_ChunkStructured(t *testing.T) {
	chunker := NewRoutingChunker(&Config{
		Provider: "router",
	})

	text := `# Document Title
This is a structured document.

## Section 1
Content here with more text.

## Section 2
More content for testing.`

	chunks, err := chunker.Chunk(context.Background(), text, &ChunkOptions{
		MaxChars: 100,
		MinChars: 10,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("expected chunks, got none")
	}
}

func TestRoutingChunker_ChunkTableKV(t *testing.T) {
	chunker := NewRoutingChunker(&Config{
		Provider: "router",
	})

	text := `Name: John
Age: 30
Email: john@example.com

Name: Jane
Age: 28
Email: jane@example.com`

	chunks, err := chunker.Chunk(context.Background(), text, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("expected chunks, got none")
	}
}

func TestRoutingChunker_ChunkUnstructured(t *testing.T) {
	chunker := NewRoutingChunker(&Config{
		Provider: "router",
	})

	text := `This is just some random unstructured text without any special formatting.
It doesn't have headings or tables or key-value pairs.
It's just plain text that needs to be chunked.`

	chunks, err := chunker.Chunk(context.Background(), text, &ChunkOptions{
		MaxChars: 100,
		MinChars: 10,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("expected chunks, got none")
	}
}

func TestRuleBasedDocumentTypeDetector_Structured(t *testing.T) {
	detector := &RuleBasedDocumentTypeDetector{}

	text := `# Title
Content here

## Section
More content`

	docType, err := detector.DetectDocumentType(context.Background(), text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if docType != DocumentTypeStructured {
		t.Errorf("expected DocumentTypeStructured, got %v", docType)
	}
}

func TestRuleBasedDocumentTypeDetector_TableKV(t *testing.T) {
	detector := &RuleBasedDocumentTypeDetector{}

	text := `Name: John
Age: 30
Email: john@example.com`

	docType, err := detector.DetectDocumentType(context.Background(), text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if docType != DocumentTypeTableKV {
		t.Errorf("expected DocumentTypeTableKV, got %v", docType)
	}
}

func TestRuleBasedDocumentTypeDetector_Table(t *testing.T) {
	detector := &RuleBasedDocumentTypeDetector{}

	text := `| Name | Age |
| John | 30 |
| Jane | 28 |`

	docType, err := detector.DetectDocumentType(context.Background(), text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if docType != DocumentTypeTableKV {
		t.Errorf("expected DocumentTypeTableKV, got %v", docType)
	}
}

func TestRuleBasedDocumentTypeDetector_Unstructured(t *testing.T) {
	detector := &RuleBasedDocumentTypeDetector{}

	text := `This is just some random text without any special structure.
It doesn't have headings or tables or key-value pairs.`

	docType, err := detector.DetectDocumentType(context.Background(), text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if docType != DocumentTypeUnstructured {
		t.Errorf("expected DocumentTypeUnstructured, got %v", docType)
	}
}

func TestRuleBasedDocumentTypeDetector_EmptyText(t *testing.T) {
	detector := &RuleBasedDocumentTypeDetector{}

	_, err := detector.DetectDocumentType(context.Background(), "")
	if err != ErrEmptyText {
		t.Errorf("expected ErrEmptyText, got %v", err)
	}
}

func TestIsTableKVDocument_WithPipes(t *testing.T) {
	text := `| Header 1 | Header 2 |
| Value 1 | Value 2 |
| Value 3 | Value 4 |`

	if !isTableKVDocument(text) {
		t.Error("expected to detect table document")
	}
}

func TestIsTableKVDocument_WithColons(t *testing.T) {
	text := `Name: John
Age: 30
Email: john@example.com`

	if !isTableKVDocument(text) {
		t.Error("expected to detect KV document")
	}
}

func TestIsTableKVDocument_WithEquals(t *testing.T) {
	text := `key1=value1
key2=value2
key3=value3`

	if !isTableKVDocument(text) {
		t.Error("expected to detect KV document with equals")
	}
}

func TestIsTableKVDocument_NotTableKV(t *testing.T) {
	text := `This is just regular text.
No special formatting here.`

	if isTableKVDocument(text) {
		t.Error("expected not to detect as table/KV document")
	}
}

func TestIsStructuredDocument_WithHeadings(t *testing.T) {
	text := `# Title
Content

## Section
More content`

	if !isStructuredDocument(text) {
		t.Error("expected to detect structured document")
	}
}

func TestIsStructuredDocument_WithParagraphs(t *testing.T) {
	text := `Paragraph 1 with content.

Paragraph 2 with more content.

Paragraph 3 with even more content.`

	if !isStructuredDocument(text) {
		t.Error("expected to detect structured document with paragraphs")
	}
}

func TestIsStructuredDocument_NotStructured(t *testing.T) {
	text := `Single line of text without any structure.`

	if isStructuredDocument(text) {
		t.Error("expected not to detect as structured document")
	}
}
