package langfuse_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	llmretrieve "github.com/LingByte/lingllm/retrieve"
	"github.com/LingByte/lingllm/retrieve/langfuse"
)

func TestObserver_EndToEnd(t *testing.T) {
	var mu sync.Mutex
	count := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/public/ingestion", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := langfuse.NewClient(langfuse.Config{
		Enabled:    true,
		BaseURL:    srv.URL,
		PublicKey:  "pk",
		SecretKey:  "sk",
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	obs := langfuse.NewObserver(c, "kb-retrieval")
	ctx := obs.BeginRetrieval(context.Background(), "test query", map[string]string{"strategy": "hybrid"})
	obs.RecordStage(ctx, llmretrieve.StageRetrieve, llmretrieve.StageEvent{
		DurationMs:  12,
		InputCount:  10,
		OutputCount: 5,
	})
	obs.EndRetrieval(ctx, []*llmretrieve.Document{{ID: "a", Score: 0.9}}, nil)
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if count == 0 {
		t.Fatal("expected langfuse ingestion call")
	}
}

func TestNewClient_Disabled(t *testing.T) {
	c, err := langfuse.NewClient(langfuse.Config{Enabled: false})
	if err != nil || c != nil {
		t.Fatalf("client=%v err=%v", c, err)
	}
}
