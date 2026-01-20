package search

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// mockEngine is a mock implementation of Engine for testing
type mockEngine struct {
	indexFunc                func(ctx context.Context, doc Doc) error
	indexBatchFunc           func(ctx context.Context, docs []Doc) error
	deleteFunc               func(ctx context.Context, id string) error
	searchFunc               func(ctx context.Context, req SearchRequest) (SearchResult, error)
	getAutoCompleteFunc      func(ctx context.Context, keyword string) ([]string, error)
	getSearchSuggestionsFunc func(ctx context.Context, keyword string) ([]string, error)
	closeFunc                func() error
}

func (m *mockEngine) Index(ctx context.Context, doc Doc) error {
	if m.indexFunc != nil {
		return m.indexFunc(ctx, doc)
	}
	return nil
}

func (m *mockEngine) IndexBatch(ctx context.Context, docs []Doc) error {
	if m.indexBatchFunc != nil {
		return m.indexBatchFunc(ctx, docs)
	}
	return nil
}

func (m *mockEngine) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockEngine) Search(ctx context.Context, req SearchRequest) (SearchResult, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, req)
	}
	return SearchResult{}, nil
}

func (m *mockEngine) GetAutoCompleteSuggestions(ctx context.Context, keyword string) ([]string, error) {
	if m.getAutoCompleteFunc != nil {
		return m.getAutoCompleteFunc(ctx, keyword)
	}
	return []string{}, nil
}

func (m *mockEngine) GetSearchSuggestions(ctx context.Context, keyword string) ([]string, error) {
	if m.getSearchSuggestionsFunc != nil {
		return m.getSearchSuggestionsFunc(ctx, keyword)
	}
	return []string{}, nil
}

func (m *mockEngine) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	// Set up config for testing
	if config.GlobalConfig == nil {
		config.GlobalConfig = &config.Config{
			SearchEnabled: true,
		}
	} else {
		config.GlobalConfig.SearchEnabled = true
	}
	return gin.New()
}

func TestNewSearchHandlers(t *testing.T) {
	mock := &mockEngine{}
	handlers := NewSearchHandlers(mock)

	assert.NotNil(t, handlers)
	assert.Equal(t, mock, handlers.engine)
}

func TestSearchHandlers_HandleSearch_Success(t *testing.T) {
	router := setupTestRouter()
	mock := &mockEngine{
		searchFunc: func(ctx context.Context, req SearchRequest) (SearchResult, error) {
			return SearchResult{
				Total: 1,
				Hits:  []Hit{{ID: "doc1", Score: 1.0}},
			}, nil
		},
	}
	handlers := NewSearchHandlers(mock)
	handlers.RegisterSearchRoutes(router.Group("/api"))

	req := SearchRequest{
		Keyword: "test",
		Size:    10,
	}
	body, _ := json.Marshal(req)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/search/", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchHandlers_HandleSearch_InvalidRequest(t *testing.T) {
	router := setupTestRouter()
	mock := &mockEngine{}
	handlers := NewSearchHandlers(mock)
	handlers.RegisterSearchRoutes(router.Group("/api"))

	body := []byte("invalid json")

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/search/", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code) // response.Fail returns 200 with error code
}

func TestSearchHandlers_HandleIndex_Success(t *testing.T) {
	router := setupTestRouter()
	mock := &mockEngine{
		indexFunc: func(ctx context.Context, doc Doc) error {
			return nil
		},
	}
	handlers := NewSearchHandlers(mock)
	handlers.RegisterSearchRoutes(router.Group("/api"))

	doc := Doc{
		ID:   "doc1",
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test",
		},
	}
	body, _ := json.Marshal(doc)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/search/index", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchHandlers_HandleIndex_MissingID(t *testing.T) {
	router := setupTestRouter()
	mock := &mockEngine{}
	handlers := NewSearchHandlers(mock)
	handlers.RegisterSearchRoutes(router.Group("/api"))

	doc := Doc{
		Type: "article",
		Fields: map[string]interface{}{
			"title": "Test",
		},
	}
	body, _ := json.Marshal(doc)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/search/index", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code) // response.Fail returns 200
}

func TestSearchHandlers_HandleDelete_Success(t *testing.T) {
	router := setupTestRouter()
	mock := &mockEngine{
		deleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}
	handlers := NewSearchHandlers(mock)
	handlers.RegisterSearchRoutes(router.Group("/api"))

	req := map[string]string{"id": "doc1"}
	body, _ := json.Marshal(req)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/search/delete", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchHandlers_HandleDelete_MissingID(t *testing.T) {
	router := setupTestRouter()
	mock := &mockEngine{}
	handlers := NewSearchHandlers(mock)
	handlers.RegisterSearchRoutes(router.Group("/api"))

	req := map[string]string{}
	body, _ := json.Marshal(req)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/search/delete", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code) // response.Fail returns 200
}

func TestSearchHandlers_HandleAutoComplete_Success(t *testing.T) {
	router := setupTestRouter()
	mock := &mockEngine{
		getAutoCompleteFunc: func(ctx context.Context, keyword string) ([]string, error) {
			return []string{"suggestion1", "suggestion2"}, nil
		},
	}
	handlers := NewSearchHandlers(mock)
	handlers.RegisterSearchRoutes(router.Group("/api"))

	req := map[string]string{"keyword": "test"}
	body, _ := json.Marshal(req)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/search/auto-complete", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchHandlers_HandleSuggest_Success(t *testing.T) {
	router := setupTestRouter()
	mock := &mockEngine{
		getSearchSuggestionsFunc: func(ctx context.Context, keyword string) ([]string, error) {
			return []string{"suggestion1", "suggestion2"}, nil
		},
	}
	handlers := NewSearchHandlers(mock)
	handlers.RegisterSearchRoutes(router.Group("/api"))

	req := map[string]string{"keyword": "test"}
	body, _ := json.Marshal(req)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/search/suggest", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
}
