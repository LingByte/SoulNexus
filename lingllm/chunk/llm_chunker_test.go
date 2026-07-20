package chunk

import (
	"context"
	"errors"
	"testing"

	"github.com/LingByte/lingllm/protocol"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestNewLLMChunker(t *testing.T) {
	cfg := &Config{
		Provider: "llm",
	}

	chunker := NewLLMChunker(cfg)

	if chunker == nil {
		t.Fatal("expected chunker, got nil")
	}

	if chunker.config != cfg {
		t.Error("expected config to be set")
	}
}

func TestLLMChunker_Provider(t *testing.T) {
	chunker := NewLLMChunker(&Config{})

	if chunker.Provider() != "llm" {
		t.Errorf("expected provider 'llm', got '%s'", chunker.Provider())
	}
}

func TestLLMChunker_ChunkEmptyText(t *testing.T) {
	chunker := NewLLMChunker(&Config{})

	_, err := chunker.Chunk(context.Background(), "", nil)
	if err != ErrEmptyText {
		t.Errorf("expected ErrEmptyText, got %v", err)
	}
}

type stubChatModel struct {
	lastReq protocol.ChatRequest
	resp    string
}

func (m *stubChatModel) Name() string { return "stub" }

func (m *stubChatModel) Chat(ctx context.Context, req protocol.ChatRequest) (*protocol.ChatResponse, error) {
	m.lastReq = req
	return &protocol.ChatResponse{
		Choices: []protocol.Choice{{
			Message: protocol.Message{Role: protocol.RoleAssistant, Content: m.resp},
		}},
	}, nil
}

func (m *stubChatModel) StreamChat(ctx context.Context, req protocol.ChatRequest) (protocol.ChatStream, error) {
	return nil, errors.New("not implemented")
}

func TestLLMChunker_UsesConfigModel(t *testing.T) {
	model := &stubChatModel{
		resp: `[{"text": "chunk 1", "title": "title 1"}]`,
	}
	chunker := NewLLMChunker(&Config{
		Model:     "claude-3-5-sonnet-20241022",
		ChatModel: model,
	})

	_, err := chunker.Chunk(context.Background(), "test text", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model.lastReq.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("expected configured model, got %q", model.lastReq.Model)
	}
}

func TestLLMChunker_MissingModel(t *testing.T) {
	chunker := NewLLMChunker(&Config{
		ChatModel: &stubChatModel{},
	})

	_, err := chunker.Chunk(context.Background(), "test text", nil)
	if err == nil {
		t.Fatal("expected error for missing model")
	}
	if err.Error() != "Model is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLLMChunker_ChunkNilChatModel(t *testing.T) {
	chunker := NewLLMChunker(&Config{
		ChatModel: nil,
	})

	_, err := chunker.Chunk(context.Background(), "test text", nil)
	if err == nil {
		t.Error("expected error for nil ChatModel")
	}
}

func TestBuildChunkPrompt(t *testing.T) {
	prompt := buildChunkPrompt("test text", "Test Title", 600, 80, 40)

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}

	if !contains(prompt, "600") {
		t.Error("expected maxChars in prompt")
	}

	if !contains(prompt, "80") {
		t.Error("expected overlap in prompt")
	}

	if !contains(prompt, "40") {
		t.Error("expected minChars in prompt")
	}

	if !contains(prompt, "Test Title") {
		t.Error("expected document title in prompt")
	}

	if !contains(prompt, "test text") {
		t.Error("expected text in prompt")
	}
}

func TestParseLLMChunks_ValidJSON(t *testing.T) {
	raw := `[
		{"text": "chunk 1", "title": "title 1"},
		{"text": "chunk 2", "title": "title 2"}
	]`

	chunks, err := parseLLMChunks(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}

	if chunks[0].Text != "chunk 1" {
		t.Errorf("expected text 'chunk 1', got '%s'", chunks[0].Text)
	}

	if chunks[0].Title != "title 1" {
		t.Errorf("expected title 'title 1', got '%s'", chunks[0].Title)
	}

	if chunks[1].Text != "chunk 2" {
		t.Errorf("expected text 'chunk 2', got '%s'", chunks[1].Text)
	}
}

func TestParseLLMChunks_InvalidJSON(t *testing.T) {
	raw := `invalid json`

	_, err := parseLLMChunks(raw)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseLLMChunks_EmptyArray(t *testing.T) {
	raw := `[]`

	_, err := parseLLMChunks(raw)
	if err == nil {
		t.Error("expected error for empty array")
	}
}

func TestParseLLMChunks_EmptyText(t *testing.T) {
	raw := `[
		{"text": "", "title": "title"},
		{"text": "valid chunk", "title": ""}
	]`

	chunks, err := parseLLMChunks(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have the valid chunk
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk (empty text filtered), got %d", len(chunks))
	}

	if chunks[0].Text != "valid chunk" {
		t.Errorf("expected text 'valid chunk', got '%s'", chunks[0].Text)
	}
}

func TestParseLLMChunks_WhitespaceHandling(t *testing.T) {
	raw := `[
		{"text": "  chunk with spaces  ", "title": "  title  "}
	]`

	chunks, err := parseLLMChunks(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}

	// Text should be trimmed
	if chunks[0].Text != "chunk with spaces" {
		t.Errorf("expected trimmed text 'chunk with spaces', got '%s'", chunks[0].Text)
	}

	// Title should be trimmed
	if chunks[0].Title != "title" {
		t.Errorf("expected trimmed title 'title', got '%s'", chunks[0].Title)
	}
}

func TestPreviewForError(t *testing.T) {
	tests := []struct {
		data     []byte
		maxLen   int
		expected string
	}{
		{[]byte("short"), 10, "short"},
		{[]byte("this is a long string"), 10, "this is a ..."},
		{[]byte(""), 10, ""},
		{[]byte("exactly10c"), 10, "exactly10c"},
	}

	for _, test := range tests {
		result := previewForError(test.data, test.maxLen)
		if result != test.expected {
			t.Errorf("expected '%s', got '%s'", test.expected, result)
		}
	}
}

func TestLLMChunker_ChunkWithOptions(t *testing.T) {
	chunker := NewLLMChunker(&Config{})

	opts := &ChunkOptions{
		MaxChars:      800,
		OverlapChars:  100,
		MinChars:      50,
		DocumentTitle: "Test Doc",
	}

	// This will fail because chatModel is nil, but it tests the options path
	_, err := chunker.Chunk(context.Background(), "test text", opts)
	if err == nil {
		t.Error("expected error for nil ChatModel")
	}
}

func TestLLMChunker_ChunkWhitespaceOnlyText(t *testing.T) {
	chunker := NewLLMChunker(&Config{})

	_, err := chunker.Chunk(context.Background(), "   \n\t  ", nil)
	if err != ErrEmptyText {
		t.Errorf("expected ErrEmptyText, got %v", err)
	}
}

func TestBuildChunkPrompt_EmptyTitle(t *testing.T) {
	prompt := buildChunkPrompt("test text", "", 600, 80, 40)

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}

	if !contains(prompt, "test text") {
		t.Error("expected text in prompt")
	}
}

func TestBuildChunkPrompt_CustomValues(t *testing.T) {
	prompt := buildChunkPrompt("content", "Title", 1000, 200, 100)

	if !contains(prompt, "1000") {
		t.Error("expected custom maxChars in prompt")
	}

	if !contains(prompt, "200") {
		t.Error("expected custom overlap in prompt")
	}

	if !contains(prompt, "100") {
		t.Error("expected custom minChars in prompt")
	}
}

func TestParseLLMChunks_AllEmptyText(t *testing.T) {
	raw := `[
		{"text": "", "title": "title1"},
		{"text": "   ", "title": "title2"}
	]`

	_, err := parseLLMChunks(raw)
	if err == nil {
		t.Error("expected error when all chunks have empty text")
	}
}

func TestParseLLMChunks_MissingFields(t *testing.T) {
	raw := `[
		{"text": "chunk 1"},
		{"title": "title 2"}
	]`

	chunks, err := parseLLMChunks(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk (missing text filtered), got %d", len(chunks))
	}

	if chunks[0].Text != "chunk 1" {
		t.Errorf("expected text 'chunk 1', got '%s'", chunks[0].Text)
	}
}

func TestPreviewForError_ExactLength(t *testing.T) {
	data := []byte("1234567890")
	result := previewForError(data, 10)
	if result != "1234567890" {
		t.Errorf("expected '1234567890', got '%s'", result)
	}
}

func TestPreviewForError_LongString(t *testing.T) {
	data := []byte("this is a very long string that exceeds max length")
	result := previewForError(data, 20)
	expected := "this is a very long ..."
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
