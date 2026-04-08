package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	maxEmbedInputChars  = 12000
	maxEmbedBatchInputs = 16
)

type NvidiaEmbedClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

func (c *NvidiaEmbedClient) Embed(ctx context.Context, inputs []string) ([][]float64, error) {
	if c == nil {
		return nil, errors.New("client is nil")
	}
	if c.BaseURL == "" {
		return nil, errors.New("BaseURL is required")
	}
	if c.APIKey == "" {
		return nil, errors.New("APIKey is required")
	}
	if c.Model == "" {
		return nil, errors.New("Model is required")
	}
	if len(inputs) == 0 {
		return nil, errors.New("inputs is empty")
	}
	cl := c.HTTPClient
	if cl == nil {
		cl = &http.Client{Timeout: 30 * time.Second}
	}

	endpoint := strings.TrimRight(c.BaseURL, "/")
	if !strings.HasSuffix(endpoint, "/embeddings") {
		endpoint += "/embeddings"
	}
	sanitized := make([]string, 0, len(inputs))
	for _, in := range inputs {
		in = strings.TrimSpace(in)
		if in == "" {
			in = "-"
		}
		if len(in) > maxEmbedInputChars {
			in = in[:maxEmbedInputChars]
		}
		sanitized = append(sanitized, in)
	}

	out := make([][]float64, 0, len(sanitized))
	for start := 0; start < len(sanitized); start += maxEmbedBatchInputs {
		end := start + maxEmbedBatchInputs
		if end > len(sanitized) {
			end = len(sanitized)
		}
		batch := sanitized[start:end]
		body := map[string]any{
			"model": c.Model,
			"input": batch,
		}
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.APIKey)

		resp, err := cl.Do(req)
		if err != nil {
			return nil, err
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("embeddings request failed: status=%d body=%s", resp.StatusCode, string(respBody))
		}

		var parsed struct {
			Data []struct {
				Embedding []float64 `json:"embedding"`
			} `json:"data"`
		}
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return nil, err
		}
		if len(parsed.Data) == 0 {
			return nil, errors.New("no embeddings returned")
		}
		for _, d := range parsed.Data {
			if len(d.Embedding) == 0 {
				return nil, errors.New("empty embedding returned")
			}
			out = append(out, d.Embedding)
		}
	}
	if len(out) != len(sanitized) {
		return nil, fmt.Errorf("embedding count mismatch: got=%d want=%d", len(out), len(sanitized))
	}
	return out, nil
}

type SiliconFlowRerankClient struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

type RerankResult struct {
	Index int
	Score float64
}

func (c *SiliconFlowRerankClient) Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error) {
	if c == nil {
		return nil, errors.New("client is nil")
	}
	if c.BaseURL == "" {
		return nil, errors.New("BaseURL is required")
	}
	if c.APIKey == "" {
		return nil, errors.New("APIKey is required")
	}
	if c.Model == "" {
		return nil, errors.New("Model is required")
	}
	if query == "" {
		return nil, errors.New("query is empty")
	}
	if len(documents) == 0 {
		return nil, errors.New("documents is empty")
	}
	if topN <= 0 {
		topN = 5
	}

	cl := c.HTTPClient
	if cl == nil {
		cl = &http.Client{Timeout: 30 * time.Second}
	}

	endpoint := strings.TrimRight(c.BaseURL, "/")
	if !strings.HasSuffix(endpoint, "/rerank") {
		endpoint += "/rerank"
	}
	body := map[string]any{
		"model":     c.Model,
		"query":     query,
		"documents": documents,
		"top_n":     topN,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("rerank request failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	// Try a few common response shapes.
	var parsed1 struct {
		Results []struct {
			Index          int     `json:"index"`
			RelevanceScore float64 `json:"relevance_score"`
			Score          float64 `json:"score"`
		} `json:"results"`
	}
	if err := json.Unmarshal(respBody, &parsed1); err == nil && len(parsed1.Results) > 0 {
		out := make([]RerankResult, 0, len(parsed1.Results))
		for _, r := range parsed1.Results {
			s := r.Score
			if s == 0 {
				s = r.RelevanceScore
			}
			out = append(out, RerankResult{Index: r.Index, Score: s})
		}
		return out, nil
	}

	var parsed2 struct {
		Data []struct {
			Index int     `json:"index"`
			Score float64 `json:"score"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed2); err == nil && len(parsed2.Data) > 0 {
		out := make([]RerankResult, 0, len(parsed2.Data))
		for _, r := range parsed2.Data {
			out = append(out, RerankResult{Index: r.Index, Score: r.Score})
		}
		return out, nil
	}

	return nil, fmt.Errorf("unrecognized rerank response: %s", string(respBody))
}
