package chunk

import (
	"context"
	"strings"
	"testing"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestNewStructuredRuleChunker(t *testing.T) {
	cfg := &Config{
		Provider: "rules_structured",
	}

	chunker := NewStructuredRuleChunker(cfg)

	if chunker == nil {
		t.Fatal("expected chunker, got nil")
	}

	if chunker.config != cfg {
		t.Error("expected config to be set")
	}
}

func TestStructuredRuleChunker_Provider(t *testing.T) {
	chunker := NewStructuredRuleChunker(&Config{})

	if chunker.Provider() != "rules_structured" {
		t.Errorf("expected provider 'rules_structured', got '%s'", chunker.Provider())
	}
}

func TestStructuredRuleChunker_ChunkEmptyText(t *testing.T) {
	chunker := NewStructuredRuleChunker(&Config{})

	_, err := chunker.Chunk(context.Background(), "", nil)
	if err != ErrEmptyText {
		t.Errorf("expected ErrEmptyText, got %v", err)
	}
}

func TestStructuredRuleChunker_ChunkWithHeadings(t *testing.T) {
	chunker := NewStructuredRuleChunker(&Config{})

	text := `# Main Title
This is introduction content with some text.

## Section 1
Section 1 content here with more text.

## Section 2
Section 2 content here with additional text.`

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

	for i, chunk := range chunks {
		if chunk.Index != i {
			t.Errorf("chunk %d: expected index %d, got %d", i, i, chunk.Index)
		}

		if len(chunk.Text) == 0 {
			t.Errorf("chunk %d: text is empty", i)
		}
	}
}

func TestStructuredRuleChunker_ChunkSmallText(t *testing.T) {
	chunker := NewStructuredRuleChunker(&Config{})

	// Test with very small text that results in no chunks
	_, err := chunker.Chunk(context.Background(), "x", &ChunkOptions{
		MaxChars: 100,
		MinChars: 50,
	})

	// Should return ErrNoChunks because text is too small
	if err != ErrNoChunks {
		t.Errorf("expected ErrNoChunks, got %v", err)
	}
}

func TestNewTableKVChunker(t *testing.T) {
	cfg := &Config{
		Provider: "rules_table_kv",
	}

	chunker := NewTableKVChunker(cfg)

	if chunker == nil {
		t.Fatal("expected chunker, got nil")
	}

	if chunker.config != cfg {
		t.Error("expected config to be set")
	}
}

func TestTableKVChunker_Provider(t *testing.T) {
	chunker := NewTableKVChunker(&Config{})

	if chunker.Provider() != "rules_table_kv" {
		t.Errorf("expected provider 'rules_table_kv', got '%s'", chunker.Provider())
	}
}

func TestTableKVChunker_ChunkEmptyText(t *testing.T) {
	chunker := NewTableKVChunker(&Config{})

	_, err := chunker.Chunk(context.Background(), "", nil)
	if err != ErrEmptyText {
		t.Errorf("expected ErrEmptyText, got %v", err)
	}
}

func TestTableKVChunker_ChunkKeyValuePairs(t *testing.T) {
	chunker := NewTableKVChunker(&Config{})

	text := `Name: John Doe
Age: 30
Email: john@example.com

Name: Jane Smith
Age: 28
Email: jane@example.com`

	chunks, err := chunker.Chunk(context.Background(), text, &ChunkOptions{
		MaxChars: 100,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("expected chunks, got none")
	}

	for i, chunk := range chunks {
		if chunk.Index != i {
			t.Errorf("chunk %d: expected index %d, got %d", i, i, chunk.Index)
		}

		if len(chunk.Text) == 0 {
			t.Errorf("chunk %d: text is empty", i)
		}
	}
}

func TestTableKVChunker_ChunkTable(t *testing.T) {
	chunker := NewTableKVChunker(&Config{})

	text := `| Name | Age | City |
| John | 30 | NYC |
| Jane | 28 | LA |`

	chunks, err := chunker.Chunk(context.Background(), text, &ChunkOptions{
		MaxChars: 100,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("expected chunks, got none")
	}
}

func TestDropTinyTrailing(t *testing.T) {
	chunks := []Chunk{
		{Index: 0, Text: "This is a large chunk with enough content to pass minimum size requirement."},
		{Index: 1, Text: "Small"},
	}

	result := dropTinyTrailing(chunks, 20)

	if len(result) != 1 {
		t.Errorf("expected 1 chunk after dropping tiny, got %d", len(result))
	}

	if !strings.Contains(result[0].Text, "Small") {
		t.Error("expected small chunk to be merged into previous")
	}
}

func TestDropTinyTrailing_NoTiny(t *testing.T) {
	chunks := []Chunk{
		{Index: 0, Text: "First chunk with enough content."},
		{Index: 1, Text: "Second chunk with enough content too."},
	}

	result := dropTinyTrailing(chunks, 10)

	if len(result) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(result))
	}
}
