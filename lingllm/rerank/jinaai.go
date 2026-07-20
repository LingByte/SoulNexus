package rerank

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// JinaAIRerankClient is a reranker client for Jina AI API
type JinaAIRerankClient struct {
	BaseURL       string
	APIKey        string
	Model         string
	HTTPClient    *http.Client
	CustomHeaders map[string]string
}

// NewJinaAIRerankClient creates a new Jina AI reranker client
func NewJinaAIRerankClient(cfg *RerankClientConfig) *JinaAIRerankClient {
	if cfg == nil {
		return nil
	}

	return &JinaAIRerankClient{
		BaseURL:       cfg.BaseURL,
		APIKey:        cfg.APIKey,
		Model:         cfg.Model,
		HTTPClient:    newHTTPClient(cfg),
		CustomHeaders: cfg.CustomHeaders,
	}
}

// Provider returns the provider name
func (c *JinaAIRerankClient) Provider() string {
	return ProviderJina
}

// Rerank reranks documents based on query relevance using Jina AI API
func (c *JinaAIRerankClient) Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error) {
	if c == nil {
		return nil, errors.New(ErrNilClient)
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return nil, errors.New(ErrEmptyBaseURL)
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, errors.New(ErrEmptyAPIKey)
	}
	if strings.TrimSpace(c.Model) == "" {
		return nil, errors.New(ErrEmptyModel)
	}
	if strings.TrimSpace(query) == "" {
		return nil, errors.New(ErrEmptyQuery)
	}
	if len(documents) == 0 {
		return nil, errors.New(ErrEmptyDocuments)
	}
	topN = normalizeTopN(topN, len(documents))

	endpoint := rerankEndpoint(c.BaseURL, "/rerank")

	body := map[string]any{
		"model":            c.Model,
		"query":            query,
		"documents":        documents,
		"top_n":            topN,
		"return_documents": true,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	ApplyCustomHeaders(req, c.CustomHeaders)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("rerank request failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	// Parse Jina AI response format
	var parsed struct {
		Results []struct {
			Index          int     `json:"index"`
			RelevanceScore float64 `json:"relevance_score"`
			Score          float64 `json:"score"`
		} `json:"results"`
	}

	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(parsed.Results) == 0 {
		return nil, fmt.Errorf("no results in rerank response")
	}

	out := make([]RerankResult, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		s := r.Score
		if s == 0 {
			s = r.RelevanceScore
		}
		out = append(out, RerankResult{Index: r.Index, Score: s})
	}

	return out, nil
}
