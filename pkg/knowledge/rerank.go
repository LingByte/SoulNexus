package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/LingByte/SoulNexus/pkg/config"
)

// RerankProvider interface for rerank services
type RerankProvider interface {
	// Rerank reranks documents based on query relevance
	Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
}

// RerankResult represents a reranked document with score
type RerankResult struct {
	Index          int     `json:"index"`           // Original index in input documents
	Document       string  `json:"document"`        // Document text
	RelevanceScore float64 `json:"relevance_score"` // Relevance score (higher is better)
}

// SiliconFlowRerankProvider SiliconFlow BGE reranker provider
type SiliconFlowRerankProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewSiliconFlowRerankProvider creates a new SiliconFlow rerank provider
func NewSiliconFlowRerankProvider(cfg config.RerankConfig) *SiliconFlowRerankProvider {
	return &SiliconFlowRerankProvider{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// siliconFlowRerankRequest SiliconFlow API request structure
type siliconFlowRerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

// siliconFlowRerankResponse SiliconFlow API response structure
type siliconFlowRerankResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"results"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// Rerank reranks documents based on query relevance
func (p *SiliconFlowRerankProvider) Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error) {
	if len(documents) == 0 {
		return nil, fmt.Errorf("no documents provided")
	}

	// Build request
	reqBody := siliconFlowRerankRequest{
		Model:     p.model,
		Query:     query,
		Documents: documents,
		TopN:      topN,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/rerank", p.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var rerankResp siliconFlowRerankResponse
	if err := json.Unmarshal(body, &rerankResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Build results
	results := make([]RerankResult, 0, len(rerankResp.Results))
	for _, result := range rerankResp.Results {
		if result.Index >= 0 && result.Index < len(documents) {
			results = append(results, RerankResult{
				Index:          result.Index,
				Document:       documents[result.Index],
				RelevanceScore: result.RelevanceScore,
			})
		}
	}

	// Sort by relevance score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].RelevanceScore > results[j].RelevanceScore
	})

	return results, nil
}

// GetRerankProvider gets the configured rerank provider
func GetRerankProvider() (RerankProvider, error) {
	cfg := config.GlobalConfig.Services.KnowledgeBase.Rerank

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("rerank API key not configured")
	}

	switch cfg.Provider {
	case "siliconflow":
		return NewSiliconFlowRerankProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported rerank provider: %s", cfg.Provider)
	}
}

// RerankSearchResults reranks search results using configured provider
func RerankSearchResults(query string, results []SearchResult, topN int) ([]SearchResult, error) {
	if len(results) == 0 {
		return results, nil
	}

	provider, err := GetRerankProvider()
	if err != nil {
		// If rerank not configured, return original results
		return results, nil
	}

	// Extract documents
	documents := make([]string, len(results))
	for i, result := range results {
		documents[i] = result.Content
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Rerank
	reranked, err := provider.Rerank(ctx, query, documents, topN)
	if err != nil {
		// On error, return original results
		return results, nil
	}

	// Build reranked results
	rerankedResults := make([]SearchResult, 0, len(reranked))
	for _, rr := range reranked {
		if rr.Index >= 0 && rr.Index < len(results) {
			result := results[rr.Index]
			// Update score with rerank score
			result.Score = rr.RelevanceScore
			rerankedResults = append(rerankedResults, result)
		}
	}

	return rerankedResults, nil
}
