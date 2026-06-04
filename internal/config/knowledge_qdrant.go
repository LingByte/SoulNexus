// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	lingknowledge "github.com/LingByte/lingllm/knowledge"
)

// QdrantCollectionVectorDim reads the vector size of an existing Qdrant collection (0 if missing).
func QdrantCollectionVectorDim(ctx context.Context, qh *lingknowledge.QdrantHandler, collection string) (int, error) {
	if qh == nil {
		return 0, fmt.Errorf("qdrant handler is nil")
	}
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return 0, fmt.Errorf("collection name is empty")
	}
	base := strings.TrimRight(strings.TrimSpace(qh.BaseURL), "/")
	if base == "" {
		return 0, fmt.Errorf("qdrant base URL is empty")
	}
	reqURL := base + "/collections/" + url.PathEscape(collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(qh.APIKey) != "" {
		req.Header.Set("api-key", strings.TrimSpace(qh.APIKey))
	}
	cl := qh.HTTPClient
	if cl == nil {
		cl = http.DefaultClient
	}
	resp, err := cl.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("qdrant get collection: status=%d body=%s", resp.StatusCode, truncateErrBody(body))
	}
	var parsed struct {
		Result struct {
			Config struct {
				Params struct {
					Vectors struct {
						Size int `json:"size"`
					} `json:"vectors"`
				} `json:"params"`
			} `json:"config"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}
	return parsed.Result.Config.Params.Vectors.Size, nil
}

func truncateErrBody(b []byte) string {
	const max = 512
	s := strings.TrimSpace(string(b))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
