package rerank

import (
	"context"
	"testing"
)

func TestLocalRerankClient_Provider(t *testing.T) {
	client := NewLocalRerankClient(nil)
	if client.Provider() != ProviderLocal {
		t.Errorf("expected provider %s, got %s", ProviderLocal, client.Provider())
	}
}

func TestLocalRerankClient_Rerank(t *testing.T) {
	client := NewLocalRerankClient(&RerankClientConfig{
		Model: "local",
	})

	query := "machine learning"
	documents := []string{
		"Machine learning is a subset of artificial intelligence",
		"Deep learning uses neural networks",
		"Natural language processing with transformers",
		"Computer vision for image recognition",
	}

	results, err := client.Rerank(context.Background(), query, documents, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// First result should be the most relevant
	if results[0].Index != 0 {
		t.Errorf("expected first result to be index 0, got %d", results[0].Index)
	}

	// Scores should be in descending order
	if results[0].Score < results[1].Score {
		t.Error("expected scores in descending order")
	}
}

func TestLocalRerankClient_EmptyQuery(t *testing.T) {
	client := NewLocalRerankClient(nil)

	_, err := client.Rerank(context.Background(), "", []string{"doc1"}, 5)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestLocalRerankClient_EmptyDocuments(t *testing.T) {
	client := NewLocalRerankClient(nil)

	_, err := client.Rerank(context.Background(), "query", []string{}, 5)
	if err == nil {
		t.Fatal("expected error for empty documents")
	}
}

func TestLocalRerankClient_TopNGreaterThanDocuments(t *testing.T) {
	client := NewLocalRerankClient(nil)

	query := "test"
	documents := []string{"doc1", "doc2"}

	results, err := client.Rerank(context.Background(), query, documents, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results (all documents), got %d", len(results))
	}
}

func TestLocalRerankClient_DefaultTopN(t *testing.T) {
	client := NewLocalRerankClient(nil)

	query := "test"
	documents := make([]string, 10)
	for i := 0; i < 10; i++ {
		documents[i] = "document " + string(rune(i))
	}

	results, err := client.Rerank(context.Background(), query, documents, 0)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(results) != 5 {
		t.Errorf("expected 5 results (default topN), got %d", len(results))
	}
}

func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		queryWords []string
		doc        string
		minScore   float64
	}{
		{
			name:       "exact match",
			query:      "machine learning",
			queryWords: []string{"machine", "learning"},
			doc:        "machine learning algorithms",
			minScore:   0.5,
		},
		{
			name:       "partial match",
			query:      "deep learning",
			queryWords: []string{"deep", "learning"},
			doc:        "machine learning",
			minScore:   0.1,
		},
		{
			name:       "no match",
			query:      "python",
			queryWords: []string{"python"},
			doc:        "java programming",
			minScore:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateSimilarity(tt.query, tt.queryWords, tt.doc)
			if score < tt.minScore {
				t.Errorf("expected score >= %f, got %f", tt.minScore, score)
			}
		})
	}
}
