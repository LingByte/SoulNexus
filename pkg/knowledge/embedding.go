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

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

const (
	maxEmbedInputChars  = 12000
	maxEmbedBatchInputs = 16
)

type NvidiaEmbedClient struct {
	BaseURL        string
	APIKey         string
	Model          string
	InputKey       string // request JSON key for inputs; default: "input"
	EmbeddingsPath string
	HTTPClient     *http.Client
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
	if strings.TrimSpace(c.EmbeddingsPath) != "" {
		p := strings.TrimSpace(c.EmbeddingsPath)
		p = strings.TrimLeft(p, "/")
		endpoint += "/" + p
	} else {
		if !strings.HasSuffix(endpoint, "/embeddings") {
			endpoint += "/embeddings"
		}
	}

	inputKey := strings.TrimSpace(c.InputKey)
	if inputKey == "" {
		inputKey = "input"
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
		var respBody []byte
		var statusCode int
		var lastErr error
		var parsed struct {
			Data []struct {
				Embedding []float64 `json:"embedding"`
			} `json:"data"`
		}
		for attempt := 1; attempt <= 3; attempt++ {
			body := map[string]any{
				"model":  c.Model,
				inputKey: batch,
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
				lastErr = err
				// network error; retry
				time.Sleep(time.Duration(attempt*200) * time.Millisecond)
				continue
			}
			statusCode = resp.StatusCode
			respBody, _ = io.ReadAll(resp.Body)
			resp.Body.Close()
			if statusCode < 200 || statusCode >= 300 {
				return nil, fmt.Errorf(
					"embeddings request failed: endpoint=%s status=%d model=%s body=%s",
					endpoint, statusCode, c.Model, truncateForErr(respBody),
				)
			}
			if len(respBody) == 0 {
				lastErr = errors.New("empty response body")
				time.Sleep(time.Duration(attempt*200) * time.Millisecond)
				continue
			}
			parsed = struct {
				Data []struct {
					Embedding []float64 `json:"embedding"`
				} `json:"data"`
			}{}
			if err := json.Unmarshal(respBody, &parsed); err != nil {
				lastErr = fmt.Errorf("parse_failed: body_len=%d body=%s err=%v", len(respBody), truncateForErr(respBody), err)
				time.Sleep(time.Duration(attempt*300) * time.Millisecond)
				continue
			}
			if len(parsed.Data) == 0 {
				lastErr = fmt.Errorf("no_embeddings_returned: body_len=%d body=%s", len(respBody), truncateForErr(respBody))
				time.Sleep(time.Duration(attempt*300) * time.Millisecond)
				continue
			}
			lastErr = nil
			break
		}
		if lastErr != nil {
			return nil, fmt.Errorf("embeddings response invalid: endpoint=%s status=%d model=%s err=%v", endpoint, statusCode, c.Model, lastErr)
		}
		for _, d := range parsed.Data {
			if len(d.Embedding) == 0 {
				return nil, fmt.Errorf("empty embedding returned: endpoint=%s status=%d model=%s", endpoint, statusCode, c.Model)
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
