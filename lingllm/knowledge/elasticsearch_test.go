package knowledge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestNewKnowledgeHandler_PostgresRequiresDSN(t *testing.T) {
	_, err := NewKnowledgeHandler(HandlerFactoryParams{
		Provider:       ProviderPostgres,
		PostgresConfig: &PostgresConfig{},
	})
	if err == nil {
		t.Fatal("expected error for empty DSN")
	}
}

func TestNewKnowledgeHandler_ElasticsearchAndWeaviate(t *testing.T) {
	es, err := NewKnowledgeHandler(HandlerFactoryParams{
		Provider: ProviderElasticsearch,
		ElasticsearchConfig: &ElasticsearchConfig{
			BaseURL: "http://127.0.0.1:9200",
		},
	})
	if err != nil || es == nil {
		t.Fatalf("elasticsearch handler: %v", err)
	}
	if es.Provider() != ProviderElasticsearch {
		t.Fatalf("provider=%s", es.Provider())
	}

	wv, err := NewKnowledgeHandler(HandlerFactoryParams{
		Provider: ProviderWeaviate,
		WeaviateConfig: &WeaviateConfig{
			BaseURL: "http://127.0.0.1:8080",
		},
	})
	if err != nil || wv == nil {
		t.Fatalf("weaviate handler: %v", err)
	}
}

type esMock struct {
	mu      sync.Mutex
	indices map[string]map[string]map[string]any
}

func newESMockServer() (*httptest.Server, *esMock) {
	mock := &esMock{indices: map[string]map[string]map[string]any{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
		case strings.HasPrefix(r.URL.Path, "/_cat/indices/"):
			_ = json.NewEncoder(w).Encode([]map[string]string{})
		case r.URL.Path == "/_bulk" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"errors": false})
		default:
			path := strings.TrimPrefix(r.URL.Path, "/")
			parts := strings.Split(path, "/")
			if len(parts) == 1 && r.Method == http.MethodPut {
				idx := parts[0]
				mock.mu.Lock()
				if mock.indices[idx] == nil {
					mock.indices[idx] = map[string]map[string]any{}
				}
				mock.mu.Unlock()
				w.WriteHeader(http.StatusOK)
				return
			}
			if len(parts) == 2 && parts[1] == "_search" && r.Method == http.MethodPost {
				idx := parts[0]
				mock.mu.Lock()
				defer mock.mu.Unlock()
				var hits []any
				for id, doc := range mock.indices[idx] {
					hits = append(hits, map[string]any{"_score": 0.92, "_source": doc, "_id": id})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"hits": map[string]any{"hits": hits},
				})
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
	})
	return httptest.NewServer(mux), mock
}

func TestElasticsearchHandler_Query(t *testing.T) {
	srv, mock := newESMockServer()
	defer srv.Close()

	idx := "lingkb_demo"
	mock.indices[idx] = map[string]map[string]any{
		"p1": {
			"id":      "p1",
			"source":  "doc1",
			"title":   "T",
			"content": "elastic search test",
		},
	}

	h := &ElasticsearchHandler{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		Embedder:   fakeEmbedder{dim: 4},
	}
	results, err := h.Query(context.Background(), "elastic", &QueryOptions{Namespace: "demo", TopK: 5})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) == 0 || results[0].Record.ID != "p1" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestElasticsearchHandler_IndexName(t *testing.T) {
	h := &ElasticsearchHandler{IndexPrefix: "lingkb"}
	name, err := h.indexName("Tenant-A")
	if err != nil || name != "lingkb_tenant_a" {
		t.Fatalf("indexName=%q err=%v", name, err)
	}
}

func TestEsHitToRecord(t *testing.T) {
	raw := map[string]json.RawMessage{
		"id":      []byte(`"x"`),
		"content": []byte(`"body"`),
		"title":   []byte(`"T"`),
	}
	rec := esHitToRecord(raw)
	if rec.ID != "x" || rec.Content != "body" {
		t.Fatalf("record=%+v", rec)
	}
}
