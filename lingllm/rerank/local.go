package rerank

import (
	"context"
	"errors"
	"sort"
	"strings"
)

// LocalRerankClient is a simple local reranker based on string similarity
type LocalRerankClient struct {
	Model string
}

// NewLocalRerankClient creates a new local reranker client
func NewLocalRerankClient(cfg *RerankClientConfig) *LocalRerankClient {
	model := "local"
	if cfg != nil && cfg.Model != "" {
		model = cfg.Model
	}
	return &LocalRerankClient{
		Model: model,
	}
}

// Provider returns the provider name
func (c *LocalRerankClient) Provider() string {
	return ProviderLocal
}

// Rerank reranks documents based on query relevance using local similarity
func (c *LocalRerankClient) Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error) {
	if c == nil {
		return nil, errors.New(ErrNilClient)
	}
	if strings.TrimSpace(query) == "" {
		return nil, errors.New(ErrEmptyQuery)
	}
	if len(documents) == 0 {
		return nil, errors.New(ErrEmptyDocuments)
	}
	if topN <= 0 {
		topN = 5
	}

	// Calculate similarity scores for each document
	type scoreEntry struct {
		index int
		score float64
	}

	scores := make([]scoreEntry, 0, len(documents))
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	for i, doc := range documents {
		score := calculateSimilarity(queryLower, queryWords, strings.ToLower(doc))
		scores = append(scores, scoreEntry{index: i, score: score})
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Return top N results
	if topN > len(scores) {
		topN = len(scores)
	}

	out := make([]RerankResult, topN)
	for i := 0; i < topN; i++ {
		out[i] = RerankResult{
			Index: scores[i].index,
			Score: scores[i].score,
		}
	}

	return out, nil
}

// calculateSimilarity calculates similarity between query and document
// using word overlap and length-based scoring
func calculateSimilarity(queryLower string, queryWords []string, docLower string) float64 {
	if len(queryWords) == 0 {
		return 0.0
	}

	docWords := strings.Fields(docLower)
	if len(docWords) == 0 {
		return 0.0
	}

	// Count matching words
	matchCount := 0
	for _, qWord := range queryWords {
		for _, dWord := range docWords {
			if qWord == dWord {
				matchCount++
				break
			}
		}
	}

	// Calculate word overlap ratio
	wordOverlap := float64(matchCount) / float64(len(queryWords))

	// Calculate length similarity (prefer documents similar in length to query)
	queryLen := len(queryLower)
	docLen := len(docLower)
	var lengthSim float64
	if queryLen > 0 {
		if docLen >= queryLen {
			lengthSim = float64(queryLen) / float64(docLen)
		} else {
			lengthSim = float64(docLen) / float64(queryLen)
		}
	}

	// Calculate substring match score
	substringScore := 0.0
	if strings.Contains(docLower, queryLower) {
		substringScore = 1.0
	} else {
		// Check for partial matches
		for _, qWord := range queryWords {
			if strings.Contains(docLower, qWord) {
				substringScore += 0.1
			}
		}
		substringScore = min(substringScore, 0.9)
	}

	// Combine scores with weights
	// 50% word overlap, 30% substring match, 20% length similarity
	score := wordOverlap*0.5 + substringScore*0.3 + lengthSim*0.2

	return score
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
