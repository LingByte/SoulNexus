package knowledge

import (
	"context"
	"encoding/json"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestQdrantHandler_Integration_UpsertQueryDelete(t *testing.T) {
	if utils.GetEnv("KNOWLEDGE_INTEGRATION_TESTS") != "1" {
		t.Skip("set KNOWLEDGE_INTEGRATION_TESTS=1 to enable integration tests")
	}
	qdrantURL := utils.GetEnv("QDRANT_URL")
	qdrantKey := utils.GetEnv("QDRANT_API_KEY")
	collection := utils.GetEnv("QDRANT_COLLECTION")
	if qdrantURL == "" || collection == "" {
		t.Skip("missing QDRANT_URL/QDRANT_COLLECTION")
	}

	nvKey := utils.GetEnv("NVIDIA_API_KEY")
	nvURL := utils.GetEnv("NVIDIA_EMBEDDINGS_URL")
	nvModel := utils.GetEnv("NVIDIA_EMBEDDINGS_MODEL")
	if nvKey == "" || nvURL == "" || nvModel == "" {
		t.Skip("missing NVIDIA_API_KEY/NVIDIA_EMBEDDINGS_URL/NVIDIA_EMBEDDINGS_MODEL")
	}

	emb := &NvidiaEmbedClient{BaseURL: nvURL, APIKey: nvKey, Model: nvModel}
	h := &QdrantHandler{BaseURL: qdrantURL, APIKey: qdrantKey, Collection: collection, Embedder: emb}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	records := []Record{
		{ID: "kb_test_1", Source: "test", Title: "France", Content: "The capital of France is Paris.", Tags: []string{"geo"}, Metadata: map[string]any{"k": "v"}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "kb_test_2", Source: "test", Title: "Go", Content: "Go is a programming language.", Tags: []string{"tech"}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	assert.NoError(t, h.Upsert(ctx, records, &UpsertOptions{}))

	res, err := h.Query(ctx, "What is the capital of France?", &QueryOptions{TopK: 3})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(res), 1)

	// Cleanup
	assert.NoError(t, h.Delete(ctx, []string{"kb_test_1", "kb_test_2"}, &DeleteOptions{}))
}

func TestKnowledgeFullFlow_Integration_EmbedUpsertQueryRerank(t *testing.T) {
	qdrantURL := utils.GetEnv("QDRANT_URL")
	qdrantKey := utils.GetEnv("QDRANT_API_KEY")
	collection := utils.GetEnv("QDRANT_COLLECTION")
	if qdrantURL == "" || collection == "" {
		t.Skip("missing QDRANT_URL/QDRANT_COLLECTION")
	}

	// Embeddings
	nvKey := utils.GetEnv("NVIDIA_API_KEY")
	nvURL := utils.GetEnv("NVIDIA_EMBEDDINGS_URL")
	nvModel := utils.GetEnv("NVIDIA_EMBEDDINGS_MODEL")
	if nvKey == "" || nvURL == "" || nvModel == "" {
		t.Skip("missing NVIDIA_API_KEY/NVIDIA_EMBEDDINGS_URL/NVIDIA_EMBEDDINGS_MODEL")
	}

	// Rerank
	rKey := utils.GetEnv("SILICONFLOW_API_KEY")
	rURL := utils.GetEnv("SILICONFLOW_RERANK_URL")
	rModel := utils.GetEnv("SILICONFLOW_RERANK_MODEL")
	if rKey == "" || rURL == "" || rModel == "" {
		t.Skip("missing SILICONFLOW_API_KEY/SILICONFLOW_RERANK_URL/SILICONFLOW_RERANK_MODEL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	emb := &NvidiaEmbedClient{BaseURL: nvURL, APIKey: nvKey, Model: nvModel}
	qh := &QdrantHandler{BaseURL: qdrantURL, APIKey: qdrantKey, Collection: collection, Embedder: emb}
	rr := &SiliconFlowRerankClient{BaseURL: rURL, APIKey: rKey, Model: rModel}

	records := []Record{
		{ID: "ff_1", Source: "full_flow", Title: "France", Content: "The capital of France is Paris.", Tags: []string{"geo"}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "ff_2", Source: "full_flow", Title: "Germany", Content: "The capital of Germany is Berlin.", Tags: []string{"geo"}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: "ff_3", Source: "full_flow", Title: "Go", Content: "Go is a programming language created at Google.", Tags: []string{"tech"}, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	assert.NoError(t, qh.Upsert(ctx, records, &UpsertOptions{}))
	defer func() {
		_ = qh.Delete(context.Background(), []string{"ff_1", "ff_2", "ff_3"}, &DeleteOptions{})
	}()

	query := "What is the capital of France?"
	res, err := qh.Query(ctx, query, &QueryOptions{TopK: 5})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(res), 2)

	// Recall check: France doc should be present in TopK.
	foundFrance := false

	docs := make([]string, 0, len(res))
	for _, r := range res {
		d := r.Record.Title + "\n" + r.Record.Content
		docs = append(docs, d)
		t.Logf("vector_recall: score=%.6f id=%s title=%q", r.Score, r.Record.ID, r.Record.Title)
		if strings.Contains(strings.ToLower(d), "france") || strings.Contains(strings.ToLower(d), "paris") {
			foundFrance = true
		}
	}
	assert.True(t, foundFrance, "expected France-related doc in vector TopK recall")

	ranked, err := rr.Rerank(ctx, query, docs, 3)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(ranked), 1)
	for i, r := range ranked {
		t.Logf("rerank[%d]: index=%d score=%.6f", i, r.Index, r.Score)
	}

	// Best-effort: the top rerank result should most likely point to the France doc.
	top := ranked[0]
	assert.GreaterOrEqual(t, top.Index, 0)
	assert.Less(t, top.Index, len(res))
	assert.Contains(t, docs[top.Index], "France")
}

type stubEmbedder struct{}

func (s *stubEmbedder) Embed(ctx context.Context, inputs []string) ([][]float64, error) {
	_ = ctx
	out := make([][]float64, 0, len(inputs))
	for range inputs {
		out = append(out, []float64{0.1, 0.2, 0.3})
	}
	return out, nil
}

func TestQdrantHandler_Get(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/points/retrieve") {
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &gotBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":[{"id":"x","payload":{"record_id":"rid","source":"s","title":"t","content":"c","tags":["a"],"metadata":{"k":"v"}}}]}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	h := &QdrantHandler{BaseURL: srv.URL, Collection: "col", Embedder: &stubEmbedder{}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	recs, err := h.Get(ctx, []string{"rid"}, &GetOptions{})
	assert.NoError(t, err)
	assert.Len(t, recs, 1)
	assert.Equal(t, "rid", recs[0].ID)
	assert.Equal(t, "c", recs[0].Content)

	ids, ok := gotBody["ids"].([]any)
	assert.True(t, ok)
	assert.Equal(t, qdrantPointIDFromString("rid"), ids[0].(string))
}

func TestQdrantHandler_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/points/scroll") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"result":{"points":[{"id":"x","payload":{"record_id":"rid1","source":"s","title":"t","content":"c1"}}],"next_page_offset":"next"}}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	h := &QdrantHandler{BaseURL: srv.URL, Collection: "col", Embedder: &stubEmbedder{}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := h.List(ctx, &ListOptions{Limit: 10, Offset: "", Filters: nil})
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Len(t, res.Records, 1)
	assert.Equal(t, "rid1", res.Records[0].ID)
	assert.Equal(t, "next", res.NextOffset)
}
