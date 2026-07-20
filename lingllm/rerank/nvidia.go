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

// NvidiaRerankClient is a reranker client for NVIDIA retrieval reranking API.
type NvidiaRerankClient struct {
	BaseURL       string
	APIKey        string
	Model         string
	HTTPClient    *http.Client
	CustomHeaders map[string]string
}

// NewNvidiaRerankClient creates a new NVIDIA reranker client.
func NewNvidiaRerankClient(cfg *RerankClientConfig) *NvidiaRerankClient {
	if cfg == nil {
		return nil
	}

	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "https://ai.api.nvidia.com/v1/retrieval/nvidia/reranking"
	}

	client := newHTTPClient(cfg)
	if cfg.HTTPClient == nil {
		client.Transport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	}

	return &NvidiaRerankClient{
		BaseURL:       baseURL,
		APIKey:        cfg.APIKey,
		Model:         cfg.Model,
		HTTPClient:    client,
		CustomHeaders: cfg.CustomHeaders,
	}
}

func (c *NvidiaRerankClient) Provider() string {
	return ProviderNvidia
}

func (c *NvidiaRerankClient) Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error) {
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

	passages := make([]map[string]string, len(documents))
	for i, doc := range documents {
		passages[i] = map[string]string{"text": doc}
	}

	body := map[string]any{
		"model":    c.Model,
		"query":    map[string]string{"text": query},
		"passages": passages,
	}

	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL, bytes.NewReader(b))
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
		return nil, fmt.Errorf("nvidia rerank request failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		Rankings []struct {
			Index          int     `json:"index"`
			RelevanceScore float64 `json:"logit"`
		} `json:"rankings"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(parsed.Rankings) == 0 {
		return nil, fmt.Errorf("no results in rerank response")
	}

	out := make([]RerankResult, 0, len(parsed.Rankings))
	for _, r := range parsed.Rankings {
		out = append(out, RerankResult{Index: r.Index, Score: r.RelevanceScore})
	}
	return limitResults(out, topN), nil
}
