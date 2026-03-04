package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestEngine(t *testing.T) (Engine, string) {
	tmpDir := os.TempDir()
	indexPath := filepath.Join(tmpDir, "test_search_index_"+t.Name())

	// Clean up any existing index
	os.RemoveAll(indexPath)

	cfg := Config{
		IndexPath:           indexPath,
		DefaultAnalyzer:     "standard",
		DefaultSearchFields: []string{"title", "body"},
		OpenTimeout:         5 * time.Second,
		QueryTimeout:        5 * time.Second,
		BatchSize:           100,
	}

	m := BuildIndexMapping("standard")
	engine, err := New(cfg, m)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	return engine, indexPath
}

func cleanupTestEngine(t *testing.T, engine Engine, indexPath string) {
	if engine != nil {
		_ = engine.Close()
	}
	os.RemoveAll(indexPath)
}

func TestNew_OpenExistingIndex(t *testing.T) {
	tmpDir := os.TempDir()
	indexPath := filepath.Join(tmpDir, "test_existing_index")
	defer os.RemoveAll(indexPath)

	cfg := Config{
		IndexPath:       indexPath,
		DefaultAnalyzer: "standard",
		QueryTimeout:    5 * time.Second,
	}

	m := BuildIndexMapping("standard")

	// Create first index
	engine1, err := New(cfg, m)
	if err != nil {
		t.Fatalf("Failed to create first engine: %v", err)
	}

	// Index a document
	doc := Doc{
		ID:   "test1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test Document",
		},
	}
	err = engine1.Index(context.Background(), doc)
	if err != nil {
		t.Fatalf("Failed to index document: %v", err)
	}
	engine1.Close()

	// Open existing index
	engine2, err := New(cfg, m)
	if err != nil {
		t.Fatalf("Failed to open existing index: %v", err)
	}
	defer engine2.Close()

	// Verify document exists
	req := SearchRequest{
		Keyword: "Test",
		Size:    10,
	}
	result, err := engine2.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}
	if result.Total == 0 {
		t.Fatalf("Expected to find document, but got 0 results")
	}
}

func TestBleveEngine_Index(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	doc := Doc{
		ID:   "doc1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test Article",
			"body":  "This is a test article body",
		},
	}

	err := engine.Index(context.Background(), doc)
	if err != nil {
		t.Fatalf("Index failed: %v", err)
	}
}

func TestBleveEngine_IndexBatch(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	docs := []Doc{
		{
			ID:   "doc1",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article 1",
			},
		},
		{
			ID:   "doc2",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article 2",
			},
		},
		{
			ID:   "doc3",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article 3",
			},
		},
	}

	err := engine.IndexBatch(context.Background(), docs)
	if err != nil {
		t.Fatalf("IndexBatch failed: %v", err)
	}

	// Verify documents were indexed
	req := SearchRequest{
		Keyword: "Article",
		Size:    10,
	}
	result, err := engine.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("Expected 3 documents, got %d", result.Total)
	}
}

func TestBleveEngine_Delete(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index a document
	doc := Doc{
		ID:   "doc1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test Article",
		},
	}
	err := engine.Index(context.Background(), doc)
	if err != nil {
		t.Fatalf("Index failed: %v", err)
	}

	// Delete the document
	err = engine.Delete(context.Background(), "doc1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify document is deleted
	req := SearchRequest{
		Keyword: "Test",
		Size:    10,
	}
	result, err := engine.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("Expected 0 documents after delete, got %d", result.Total)
	}
}

func TestBleveEngine_Search(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index documents
	docs := []Doc{
		{
			ID:   "doc1",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Go Programming",
				"body":  "Go is a programming language",
			},
		},
		{
			ID:   "doc2",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Python Programming",
				"body":  "Python is also a programming language",
			},
		},
	}
	for _, doc := range docs {
		err := engine.Index(context.Background(), doc)
		if err != nil {
			t.Fatalf("Index failed: %v", err)
		}
	}

	// Search
	req := SearchRequest{
		Keyword: "programming",
		Size:    10,
	}
	result, err := engine.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result.Total < 2 {
		t.Fatalf("Expected at least 2 results, got %d", result.Total)
	}
	if len(result.Hits) == 0 {
		t.Fatalf("Expected hits, got none")
	}
}

func TestBleveEngine_SearchWithPagination(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index multiple documents
	for i := 0; i < 5; i++ {
		doc := Doc{
			ID:   "doc" + string(rune('0'+i)),
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article " + string(rune('0'+i)),
			},
		}
		err := engine.Index(context.Background(), doc)
		if err != nil {
			t.Fatalf("Index failed: %v", err)
		}
	}

	// Search with pagination
	req := SearchRequest{
		Keyword: "Article",
		From:    0,
		Size:    2,
	}
	result, err := engine.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(result.Hits) > 2 {
		t.Fatalf("Expected at most 2 hits, got %d", len(result.Hits))
	}
}

func TestBleveEngine_SearchWithFacets(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index documents with tags
	docs := []Doc{
		{
			ID:   "doc1",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article 1",
				"tags":  "go",
			},
		},
		{
			ID:   "doc2",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Article 2",
				"tags":  "python",
			},
		},
	}
	for _, doc := range docs {
		err := engine.Index(context.Background(), doc)
		if err != nil {
			t.Fatalf("Index failed: %v", err)
		}
	}

	// Search with facets
	req := SearchRequest{
		Keyword: "Article",
		Size:    10,
		Facets: []FacetRequest{
			{
				Name:  "tags",
				Field: "tags",
				Size:  10,
			},
		},
	}
	result, err := engine.Search(context.Background(), req)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result.Facets == nil {
		t.Fatalf("Expected facets, got nil")
	}
}

func TestBleveEngine_GetAutoCompleteSuggestions(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index documents
	docs := []Doc{
		{
			ID:   "apple",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Apple",
			},
		},
		{
			ID:   "application",
			Type: "article",
			Fields: map[string]interface{}{
				"title": "Application",
			},
		},
	}
	for _, doc := range docs {
		err := engine.Index(context.Background(), doc)
		if err != nil {
			t.Fatalf("Index failed: %v", err)
		}
	}

	// Get suggestions
	suggestions, err := engine.GetAutoCompleteSuggestions(context.Background(), "app")
	if err != nil {
		t.Fatalf("GetAutoCompleteSuggestions failed: %v", err)
	}

	if len(suggestions) == 0 {
		t.Fatalf("Expected suggestions, got none")
	}
}

func TestBleveEngine_GetAutoCompleteSuggestions_EmptyKeyword(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	suggestions, err := engine.GetAutoCompleteSuggestions(context.Background(), "")
	if err != nil {
		t.Fatalf("GetAutoCompleteSuggestions failed: %v", err)
	}

	if len(suggestions) != 0 {
		t.Fatalf("Expected empty suggestions for empty keyword, got %d", len(suggestions))
	}
}

func TestBleveEngine_GetSearchSuggestions(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Index documents
	doc := Doc{
		ID:   "test1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test Document",
		},
	}
	err := engine.Index(context.Background(), doc)
	if err != nil {
		t.Fatalf("Index failed: %v", err)
	}

	// Get suggestions
	suggestions, err := engine.GetSearchSuggestions(context.Background(), "test")
	if err != nil {
		t.Fatalf("GetSearchSuggestions failed: %v", err)
	}

	if len(suggestions) == 0 {
		t.Fatalf("Expected suggestions, got none")
	}
}

func TestBleveEngine_GetSearchSuggestions_EmptyKeyword(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	suggestions, err := engine.GetSearchSuggestions(context.Background(), "")
	if err != nil {
		t.Fatalf("GetSearchSuggestions failed: %v", err)
	}

	if len(suggestions) != 0 {
		t.Fatalf("Expected empty suggestions for empty keyword, got %d", len(suggestions))
	}
}

func TestBleveEngine_Close(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer os.RemoveAll(indexPath)

	err := engine.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Try to use closed engine
	err = engine.Index(context.Background(), Doc{ID: "test"})
	if err != ErrClosed {
		t.Fatalf("Expected ErrClosed, got %v", err)
	}
}

func TestBleveEngine_Close_MultipleTimes(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer os.RemoveAll(indexPath)

	err := engine.Close()
	if err != nil {
		t.Fatalf("First Close failed: %v", err)
	}

	// Close again should not error
	err = engine.Close()
	if err != nil {
		t.Fatalf("Second Close failed: %v", err)
	}
}

func TestBleveEngine_WithDeadline_NilContext(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	// Test with nil context
	be := engine.(*bleveEngine)
	err := be.withDeadline(nil, 0, func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("withDeadline with nil context failed: %v", err)
	}
}

func TestBleveEngine_WithDeadline_Timeout(t *testing.T) {
	engine, indexPath := setupTestEngine(t)
	defer cleanupTestEngine(t, engine, indexPath)

	be := engine.(*bleveEngine)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := be.withDeadline(ctx, 1*time.Millisecond, func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})
	if err == nil {
		t.Fatalf("Expected timeout error, got nil")
	}
}
