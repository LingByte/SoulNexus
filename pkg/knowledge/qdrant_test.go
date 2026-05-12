package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNormalizeVec64InPlace(t *testing.T) {
	v := []float64{3, 4}
	normalizeVec64InPlace(v)
	n := math.Sqrt(v[0]*v[0] + v[1]*v[1])
	if math.Abs(n-1) > 1e-9 {
		t.Fatalf("expected normalized vector, norm=%v", n)
	}
}

func TestQdrantPointIDFromString(t *testing.T) {
	if got := qdrantPointIDFromString("123"); got.(uint64) != 123 {
		t.Fatalf("expected uint64 id, got %#v", got)
	}
	if got := qdrantPointIDFromString("550e8400-e29b-41d4-a716-446655440000"); got.(string) == "" {
		t.Fatalf("expected string id")
	}
}

func TestBuildQdrantFilter(t *testing.T) {
	f := buildQdrantFilter([]Filter{
		{Field: "doc_id", Operator: FilterOpEqual, Value: []any{"a"}},
		{Field: "tags", Operator: FilterOpContainsAny, Value: []any{"x", "y"}},
	})
	if f == nil {
		t.Fatalf("expected filter")
	}
}

func TestTruncateForErr(t *testing.T) {
	b := make([]byte, 3000)
	for i := range b {
		b[i] = 'a'
	}
	s := truncateForErr(b)
	if len(s) < 1000 || len(s) > 2500 {
		t.Fatalf("unexpected truncation length: %d", len(s))
	}
}

type fakeEmbedder struct {
	dim int
}

func (f fakeEmbedder) Embed(ctx context.Context, inputs []string) ([][]float64, error) {
	out := make([][]float64, 0, len(inputs))
	for _, in := range inputs {
		vec := make([]float64, f.dim)
		ln := float64(len(in))
		for i := 0; i < f.dim; i++ {
			vec[i] = ln + float64(i+1)
		}
		out = append(out, vec)
	}
	return out, nil
}

type inMemQdrant struct {
	collections map[string]map[string]Record
}

func newQdrantMockServer() (*httptest.Server, *inMemQdrant) {
	mem := &inMemQdrant{collections: map[string]map[string]Record{}}
	mux := http.NewServeMux()

	mux.HandleFunc("/collections", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var names []any
		for k := range mem.collections {
			names = append(names, map[string]any{"name": k})
		}
		sort.Slice(names, func(i, j int) bool {
			return names[i].(map[string]any)["name"].(string) < names[j].(map[string]any)["name"].(string)
		})
		_ = json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{"collections": names},
			"status": "ok",
		})
	})

	mux.HandleFunc("/collections/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/collections/")
		parts := strings.Split(path, "/")
		if len(parts) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		coll := parts[0]
		rest := parts[1:]

		if len(rest) == 0 && r.Method == http.MethodGet {
			if _, ok := mem.collections[coll]; ok {
				_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"status": "green"}, "status": "ok"})
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "error"})
			return
		}

		if len(rest) == 0 && r.Method == http.MethodPut {
			if _, ok := mem.collections[coll]; !ok {
				mem.collections[coll] = map[string]Record{}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{}, "status": "ok"})
			return
		}

		if len(rest) == 0 && r.Method == http.MethodDelete {
			delete(mem.collections, coll)
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{}, "status": "ok"})
			return
		}

		if len(rest) == 1 && rest[0] == "points" && r.Method == http.MethodPut {
			var req struct {
				Points []struct {
					ID      any            `json:"id"`
					Vector  []float32      `json:"vector"`
					Payload map[string]any `json:"payload"`
				} `json:"points"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if _, ok := mem.collections[coll]; !ok {
				mem.collections[coll] = map[string]Record{}
			}
			for _, p := range req.Points {
				id := ""
				switch v := p.ID.(type) {
				case string:
					id = v
				case float64:
					id = strconv.FormatInt(int64(v), 10)
				default:
					id = fmt.Sprint(v)
				}
				rec := payloadToRecord(p.ID, p.Payload)
				rec.ID = id
				mem.collections[coll][id] = rec
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{}, "status": "ok"})
			return
		}

		if len(rest) == 2 && rest[0] == "points" && rest[1] == "search" && r.Method == http.MethodPost {
			var req map[string]any
			_ = json.NewDecoder(r.Body).Decode(&req)
			limit := 10
			if v, ok := req["limit"].(float64); ok && int(v) > 0 {
				limit = int(v)
			}
			var res []any
			for id, rec := range mem.collections[coll] {
				res = append(res, map[string]any{
					"id": id,
					"payload": map[string]any{
						"source":   rec.Source,
						"title":    rec.Title,
						"content":  rec.Content,
						"tags":     rec.Tags,
						"metadata": rec.Metadata,
					},
					"score": 1.0,
				})
			}
			sort.Slice(res, func(i, j int) bool {
				return res[i].(map[string]any)["id"].(string) < res[j].(map[string]any)["id"].(string)
			})
			if len(res) > limit {
				res = res[:limit]
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": res, "status": "ok"})
			return
		}

		if len(rest) == 2 && rest[0] == "points" && rest[1] == "lookup" && r.Method == http.MethodPost {
			var req struct {
				IDs []any `json:"ids"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			var res []any
			for _, it := range req.IDs {
				id := ""
				switch v := it.(type) {
				case string:
					id = v
				case float64:
					id = strconv.FormatInt(int64(v), 10)
				default:
					id = fmt.Sprint(it)
				}
				if rec, ok := mem.collections[coll][id]; ok {
					res = append(res, map[string]any{
						"id": id,
						"payload": map[string]any{
							"source":   rec.Source,
							"title":    rec.Title,
							"content":  rec.Content,
							"tags":     rec.Tags,
							"metadata": rec.Metadata,
						},
					})
				}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": res, "status": "ok"})
			return
		}

		if len(rest) == 2 && rest[0] == "points" && rest[1] == "scroll" && r.Method == http.MethodPost {
			var resp struct {
				Result struct {
					Points []any `json:"points"`
				} `json:"result"`
			}
			for id, rec := range mem.collections[coll] {
				resp.Result.Points = append(resp.Result.Points, map[string]any{
					"id": id,
					"payload": map[string]any{
						"source":   rec.Source,
						"title":    rec.Title,
						"content":  rec.Content,
						"tags":     rec.Tags,
						"metadata": rec.Metadata,
					},
				})
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if len(rest) == 2 && rest[0] == "points" && rest[1] == "delete" && r.Method == http.MethodPost {
			var req struct {
				Points []any `json:"points"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			for _, it := range req.Points {
				delete(mem.collections[coll], fmt.Sprint(it))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{}, "status": "ok"})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	return httptest.NewServer(mux), mem
}

func TestQdrant_Mock_UpsertQueryGetListDeleteNamespaces(t *testing.T) {
	srv, _ := newQdrantMockServer()
	defer srv.Close()

	qh := &QdrantHandler{
		BaseURL:    srv.URL,
		HTTPClient: &http.Client{Timeout: 2 * time.Second},
		Embedder:   fakeEmbedder{dim: 3},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ns := "c1"
	if err := qh.CreateNamespace(ctx, ns); err != nil {
		t.Fatalf("CreateNamespace: %v", err)
	}
	defer func() { _ = qh.DeleteNamespace(context.Background(), ns) }()

	now := time.Now().UTC()
	records := []Record{
		{ID: "1", Content: "hello", Source: "s1", Title: "t1", Tags: []string{"a"}, Metadata: map[string]any{"k": "v"}, CreatedAt: now, UpdatedAt: now},
		{ID: "2", Content: "world!", Source: "s2", Title: "t2", Tags: []string{"b"}, Metadata: map[string]any{"k": "v2"}, CreatedAt: now, UpdatedAt: now},
	}
	if err := qh.Upsert(ctx, records, &UpsertOptions{Namespace: ns, BatchSize: 1}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	qres, err := qh.Query(ctx, "hello", &QueryOptions{Namespace: ns, TopK: 10, MinScore: 0})
	if err != nil || len(qres) != 2 {
		t.Fatalf("Query: err=%v len=%d", err, len(qres))
	}

	got, err := qh.Get(ctx, []string{"1", "2"}, &GetOptions{Namespace: ns})
	if err != nil || len(got) != 2 {
		t.Fatalf("Get: err=%v len=%d", err, len(got))
	}

	ls, err := qh.List(ctx, &ListOptions{Namespace: ns, Limit: 10})
	if err != nil || ls == nil || len(ls.Records) != 2 {
		t.Fatalf("List: err=%v ls=%v", err, ls)
	}

	if err := qh.Delete(ctx, []string{"1"}, &DeleteOptions{Namespace: ns}); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if err := qh.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}

	nss, err := qh.ListNamespaces(ctx)
	if err != nil || len(nss) == 0 {
		t.Fatalf("ListNamespaces: err=%v nss=%v", err, nss)
	}
}

func TestQdrant_HandlerNilAndBadURL(t *testing.T) {
	ctx := context.Background()
	var qh *QdrantHandler
	if err := qh.Upsert(ctx, []Record{{ID: "1"}}, &UpsertOptions{Namespace: "n"}); err != ErrHandlerNotFound {
		t.Fatalf("want ErrHandlerNotFound, got %v", err)
	}
	q2 := &QdrantHandler{BaseURL: " ", HTTPClient: &http.Client{Timeout: time.Second}}
	if err := q2.Ping(ctx); err != ErrBaseURL {
		t.Fatalf("want ErrBaseURL, got %v", err)
	}
}

func TestToFloat32s(t *testing.T) {
	if _, err := (&QdrantHandler{}).toFloat32s(nil); err != ErrInvalidVectorDimension {
		t.Fatalf("want ErrInvalidVectorDimension")
	}
	v, err := (&QdrantHandler{}).toFloat32s([]float64{1, 2})
	if err != nil || len(v) != 2 {
		t.Fatalf("unexpected: %v %#v", err, v)
	}
}

