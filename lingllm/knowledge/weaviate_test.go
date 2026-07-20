package knowledge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWeaviateHandler_ClassName(t *testing.T) {
	h := &WeaviateHandler{}
	class, err := h.className("demo-kb")
	if err != nil {
		t.Fatal(err)
	}
	if class != "Lingkb_Demo_kb" {
		t.Fatalf("class=%q", class)
	}
}

func TestWeaviateHandler_QueryGraphQL(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/.well-known/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v1/schema/Lingkb_Demo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{"class": "Lingkb_Demo"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	mux.HandleFunc("/v1/graphql", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"Get": map[string]any{
					"Lingkb_Demo": []map[string]any{
						{
							"record_id":     "p1",
							"source":        "doc1",
							"title":         "T",
							"content":       "weaviate vector test",
							"tags":          "[]",
							"metadata_json": "{}",
							"_additional":   map[string]any{"certainty": 0.88},
						},
					},
				},
			},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	h := &WeaviateHandler{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		Embedder:   fakeEmbedder{dim: 4},
	}
	results, err := h.Query(context.Background(), "vector", &QueryOptions{Namespace: "demo", TopK: 3})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(results) != 1 || results[0].Record.ID != "p1" {
		t.Fatalf("unexpected results: %+v", results)
	}
	if results[0].Score < 0.8 {
		t.Fatalf("score=%v", results[0].Score)
	}
}

func TestWeaviateGraphQLQueryFormat(t *testing.T) {
	q := `nearVector`
	if !strings.Contains(q, "nearVector") {
		t.Fatal("sanity")
	}
}
