package chunk

import (
	"testing"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestChunk_Structure(t *testing.T) {
	chunk := Chunk{
		Index:    0,
		Title:    "Test Title",
		Text:     "Test content",
		Metadata: map[string]interface{}{"key": "value"},
	}

	if chunk.Index != 0 {
		t.Errorf("expected index 0, got %d", chunk.Index)
	}

	if chunk.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got '%s'", chunk.Title)
	}

	if chunk.Text != "Test content" {
		t.Errorf("expected text 'Test content', got '%s'", chunk.Text)
	}

	if chunk.Metadata["key"] != "value" {
		t.Errorf("expected metadata value 'value', got '%v'", chunk.Metadata["key"])
	}
}

func TestChunkOptions_Defaults(t *testing.T) {
	opts := &ChunkOptions{
		MaxChars:      600,
		MinChars:      40,
		OverlapChars:  80,
		DocumentTitle: "Test Doc",
	}

	if opts.MaxChars != 600 {
		t.Errorf("expected MaxChars 600, got %d", opts.MaxChars)
	}

	if opts.MinChars != 40 {
		t.Errorf("expected MinChars 40, got %d", opts.MinChars)
	}

	if opts.OverlapChars != 80 {
		t.Errorf("expected OverlapChars 80, got %d", opts.OverlapChars)
	}

	if opts.DocumentTitle != "Test Doc" {
		t.Errorf("expected DocumentTitle 'Test Doc', got '%s'", opts.DocumentTitle)
	}
}

func TestDocumentType_Constants(t *testing.T) {
	tests := []struct {
		docType DocumentType
		name    string
	}{
		{DocumentTypeUnknown, "Unknown"},
		{DocumentTypeStructured, "Structured"},
		{DocumentTypeTableKV, "TableKV"},
		{DocumentTypeUnstructured, "Unstructured"},
	}

	for _, test := range tests {
		if test.docType < 0 || test.docType > 3 {
			t.Errorf("invalid DocumentType: %v", test.docType)
		}
	}
}

func TestConfig_Structure(t *testing.T) {
	cfg := &Config{
		Provider:     "llm",
		Model:        "gpt-4",
		MaxChars:     600,
		MinChars:     40,
		OverlapChars: 80,
		CustomConfig: map[string]interface{}{"key": "value"},
	}

	if cfg.Provider != "llm" {
		t.Errorf("expected provider 'llm', got '%s'", cfg.Provider)
	}

	if cfg.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got '%s'", cfg.Model)
	}

	if cfg.MaxChars != 600 {
		t.Errorf("expected MaxChars 600, got %d", cfg.MaxChars)
	}

	if cfg.CustomConfig["key"] != "value" {
		t.Errorf("expected custom config value 'value', got '%v'", cfg.CustomConfig["key"])
	}
}

func TestChunkResult_Structure(t *testing.T) {
	result := &ChunkResult{
		Chunks: []Chunk{
			{Index: 0, Text: "chunk 1"},
			{Index: 1, Text: "chunk 2"},
		},
		DocumentType: DocumentTypeStructured,
		Duration:     100,
	}

	if len(result.Chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(result.Chunks))
	}

	if result.DocumentType != DocumentTypeStructured {
		t.Errorf("expected DocumentTypeStructured, got %v", result.DocumentType)
	}

	if result.Duration != 100 {
		t.Errorf("expected duration 100, got %d", result.Duration)
	}
}

func TestChunkStats_Structure(t *testing.T) {
	stats := &ChunkStats{
		TotalChunks:  10,
		TotalChars:   6000,
		AvgChunkSize: 600,
		MinChunkSize: 40,
		MaxChunkSize: 1200,
		OverlapChars: 80,
		DocumentType: DocumentTypeStructured,
	}

	if stats.TotalChunks != 10 {
		t.Errorf("expected TotalChunks 10, got %d", stats.TotalChunks)
	}

	if stats.TotalChars != 6000 {
		t.Errorf("expected TotalChars 6000, got %d", stats.TotalChars)
	}

	if stats.AvgChunkSize != 600 {
		t.Errorf("expected AvgChunkSize 600, got %d", stats.AvgChunkSize)
	}
}

func TestErrorTypes(t *testing.T) {
	errors := []error{
		ErrEmptyText,
		ErrInvalidChunkOpt,
		ErrNoChunks,
		ErrChunkerNotFound,
		ErrProviderNotFound,
		ErrInvalidConfig,
		ErrChunkFailed,
		ErrDetectionFailed,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("error should not be nil")
		}

		if err.Error() == "" {
			t.Error("error message should not be empty")
		}
	}
}
