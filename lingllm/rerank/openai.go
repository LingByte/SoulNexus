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

// OpenAIRerankClient is a reranker client for OpenAI-compatible rerank APIs.
type OpenAIRerankClient struct {
	BaseURL       string
	APIKey        string
	Model         string
	HTTPClient    *http.Client
	CustomHeaders map[string]string
}

// NewOpenAIRerankClient creates a new OpenAI-compatible reranker client.
func NewOpenAIRerankClient(cfg *RerankClientConfig) *OpenAIRerankClient {
	if cfg == nil {
		return nil
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	return &OpenAIRerankClient{
		BaseURL:       baseURL,
		APIKey:        cfg.APIKey,
		Model:         cfg.Model,
		HTTPClient:    newHTTPClient(cfg),
		CustomHeaders: cfg.CustomHeaders,
	}
}

func (c *OpenAIRerankClient) Provider() string {
	return ProviderOpenAI
}

func (c *OpenAIRerankClient) Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error) {
	if c == nil {
		return nil, errors.New(ErrNilClient)
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

	body := map[string]any{
		"model":                  c.Model,
		"query":                  query,
		"documents":              documents,
		"truncate_prompt_tokens": 511,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := rerankEndpoint(c.BaseURL, "/rerank")
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("rerank request failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

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
		score := r.RelevanceScore
		if score == 0 {
			score = r.Score
		}
		out = append(out, RerankResult{Index: r.Index, Score: score})
	}
	return limitResults(out, topN), nil
}
