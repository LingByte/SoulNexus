package knowledge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNvidiaEmbedClient_Embed_SendsRequestAndParses(t *testing.T) {
	var gotAuth string
	var gotModel string
	var gotInputs []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotModel, _ = body["model"].(string)
		if v, ok := body["input"].([]any); ok {
			for _, it := range v {
				gotInputs = append(gotInputs, it.(string))
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []any{
				map[string]any{"embedding": []float64{0.1, 0.2}},
				map[string]any{"embedding": []float64{0.3, 0.4}},
			},
		})
	}))
	defer srv.Close()

	c := &NvidiaEmbedClient{
		BaseURL:    srv.URL,
		APIKey:     "k",
		Model:      "m",
		HTTPClient: &http.Client{Timeout: 2 * time.Second},
	}

	vecs, err := c.Embed(context.Background(), []string{" a ", "b"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if gotAuth != "Bearer k" {
		t.Fatalf("expected auth header, got %q", gotAuth)
	}
	if gotModel != "m" {
		t.Fatalf("expected model m, got %q", gotModel)
	}
	if len(gotInputs) != 2 || strings.TrimSpace(gotInputs[0]) != "a" {
		t.Fatalf("unexpected inputs: %#v", gotInputs)
	}
	if len(vecs) != 2 || len(vecs[0]) != 2 {
		t.Fatalf("unexpected vecs: %#v", vecs)
	}
}

