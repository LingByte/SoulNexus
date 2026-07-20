package knowledge

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// MockHandler implements KnowledgeHandler for testing
type MockHandler struct {
	records map[string][]Record
}

func (m *MockHandler) Provider() string {
	return "mock"
}

func (m *MockHandler) Upsert(ctx context.Context, records []Record, opts *UpsertOptions) error {
	if m.records == nil {
		m.records = make(map[string][]Record)
	}
	ns := opts.Namespace
	if ns == "" {
		ns = "default"
	}
	m.records[ns] = records
	return nil
}

func (m *MockHandler) Query(ctx context.Context, text string, opts *QueryOptions) ([]QueryResult, error) {
	ns := opts.Namespace
	if ns == "" {
		ns = "default"
	}
	records, ok := m.records[ns]
	if !ok {
		return nil, ErrRecordNotFound
	}
	results := make([]QueryResult, 0, len(records))
	for _, r := range records {
		results = append(results, QueryResult{
			Record: r,
			Score:  0.95,
		})
	}
	return results, nil
}

func (m *MockHandler) Get(ctx context.Context, ids []string, opts *GetOptions) ([]Record, error) {
	return nil, nil
}

func (m *MockHandler) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	return nil, nil
}

func (m *MockHandler) Delete(ctx context.Context, ids []string, opts *DeleteOptions) error {
	return nil
}

func (m *MockHandler) Ping(ctx context.Context) error {
	return nil
}

func (m *MockHandler) CreateNamespace(ctx context.Context, name string) error {
	return nil
}

func (m *MockHandler) DeleteNamespace(ctx context.Context, name string) error {
	if m.records != nil {
		delete(m.records, name)
	}
	return nil
}

func (m *MockHandler) ListNamespaces(ctx context.Context) ([]string, error) {
	if m.records == nil {
		return []string{}, nil
	}
	namespaces := make([]string, 0, len(m.records))
	for ns := range m.records {
		namespaces = append(namespaces, ns)
	}
	return namespaces, nil
}

// MockEmbedder implements embedder.Embedder for testing
type MockEmbedder struct{}

func (m *MockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, 384)
		for j := range result[i] {
			result[i][j] = 0.1
		}
	}
	return result, nil
}

func (m *MockEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	vec := make([]float32, 384)
	for i := range vec {
		vec[i] = 0.1
	}
	return vec, nil
}

func (m *MockEmbedder) Dimension() int {
	return 384
}

func (m *MockEmbedder) Name() string {
	return "mock"
}

func (m *MockEmbedder) Provider() string {
	return "mock"
}

func (m *MockEmbedder) Close() error {
	return nil
}

// MockChunker implements Chunker for testing
type MockChunker struct{}

func (m *MockChunker) Provider() string {
	return "mock"
}

func (m *MockChunker) Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error) {
	return []Chunk{
		{
			Index:    0,
			Title:    opts.DocumentTitle,
			Text:     text,
			Metadata: map[string]any{},
		},
	}, nil
}

func TestNewKnowledgeBase_RequiresHandler(t *testing.T) {
	_, err := NewKnowledgeBase(KnowledgeBaseConfig{})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Handler is required")
}

func TestNewKnowledgeBase_Success(t *testing.T) {
	handler := &MockHandler{}
	kb, err := NewKnowledgeBase(KnowledgeBaseConfig{
		Handler: handler,
	})
	assert.Nil(t, err)
	assert.NotNil(t, kb)
}

func TestKnowledgeBase_AddDocument_EmptyContent(t *testing.T) {
	handler := &MockHandler{}
	kb, _ := NewKnowledgeBase(KnowledgeBaseConfig{
		Handler: handler,
	})

	err := kb.AddDocument(context.Background(), "doc1", "Title", "", nil)
	assert.Equal(t, ErrEmptyText, err)
}

func TestKnowledgeBase_AddDocument_Success(t *testing.T) {
	handler := &MockHandler{}
	embedder := &MockEmbedder{}
	chunker := &MockChunker{}

	kb, _ := NewKnowledgeBase(KnowledgeBaseConfig{
		Handler:  handler,
		Embedder: embedder,
		Chunkers: map[DocumentType]Chunker{
			DocumentTypeStructured: chunker,
		},
	})

	err := kb.AddDocument(context.Background(), "doc1", "Test Document", "This is test content", nil)
	assert.Nil(t, err)

	// Verify document was stored
	records, ok := handler.records["doc1"]
	assert.True(t, ok)
	assert.Equal(t, 1, len(records))
	assert.Equal(t, "doc1#0", records[0].ID)
	assert.Equal(t, "Test Document", records[0].Title)
	assert.Equal(t, "This is test content", records[0].Content)
}

func TestKnowledgeBase_AddDocument_WithMetadata(t *testing.T) {
	handler := &MockHandler{}
	embedder := &MockEmbedder{}
	chunker := &MockChunker{}

	kb, _ := NewKnowledgeBase(KnowledgeBaseConfig{
		Handler:  handler,
		Embedder: embedder,
		Chunkers: map[DocumentType]Chunker{
			DocumentTypeStructured: chunker,
		},
	})

	metadata := map[string]any{
		"author": "John Doe",
		"year":   2024,
	}

	err := kb.AddDocument(context.Background(), "doc1", "Test", "Content", metadata)
	assert.Nil(t, err)

	records := handler.records["doc1"]
	assert.Equal(t, "John Doe", records[0].Metadata["author"])
	assert.Equal(t, 2024, records[0].Metadata["year"])
}

func TestKnowledgeBase_Query_EmptyQuery(t *testing.T) {
	handler := &MockHandler{}
	kb, _ := NewKnowledgeBase(KnowledgeBaseConfig{
		Handler: handler,
	})

	_, err := kb.Query(context.Background(), "", 10)
	assert.Equal(t, ErrEmptyQuery, err)
}

func TestKnowledgeBase_Query_Success(t *testing.T) {
	handler := &MockHandler{}
	handler.records = map[string][]Record{
		"default": {
			{
				ID:      "chunk1",
				Title:   "Test",
				Content: "Test content",
			},
		},
	}

	kb, _ := NewKnowledgeBase(KnowledgeBaseConfig{
		Handler: handler,
	})

	results, err := kb.Query(context.Background(), "test", 10)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(results))
	assert.Equal(t, "chunk1", results[0].Record.ID)
}

func TestKnowledgeBase_DeleteDocument_Success(t *testing.T) {
	handler := &MockHandler{}
	handler.records = map[string][]Record{
		"doc1": {
			{ID: "doc1#0", Content: "Test"},
		},
	}

	kb, _ := NewKnowledgeBase(KnowledgeBaseConfig{
		Handler: handler,
	})

	err := kb.DeleteDocument(context.Background(), "doc1")
	assert.Nil(t, err)

	// Verify document was deleted
	_, ok := handler.records["doc1"]
	assert.False(t, ok)
}

func TestKnowledgeBase_Health_Success(t *testing.T) {
	handler := &MockHandler{}
	kb, _ := NewKnowledgeBase(KnowledgeBaseConfig{
		Handler: handler,
	})

	err := kb.Health(context.Background())
	assert.Nil(t, err)
}

func TestKnowledgeBase_Close_Success(t *testing.T) {
	handler := &MockHandler{}
	embedder := &MockEmbedder{}

	kb, _ := NewKnowledgeBase(KnowledgeBaseConfig{
		Handler:  handler,
		Embedder: embedder,
	})

	err := kb.Close()
	assert.Nil(t, err)
}
