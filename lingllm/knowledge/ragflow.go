package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/LingByte/lingllm/embedder"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: AGPL-3.0

// RAGFlowHandler implements KnowledgeHandler using RAGFlow.
// RAGFlow is an open-source RAG engine that provides document management,
// chunking, and vector search capabilities.
type RAGFlowHandler struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Embedder   embedder.Embedder
}

func (rh *RAGFlowHandler) Provider() string { return ProviderRAGFlow }

// Upsert adds or updates records in RAGFlow knowledge base
func (rh *RAGFlowHandler) Upsert(ctx context.Context, records []Record, opts *UpsertOptions) error {
	if rh == nil {
		return ErrHandlerNotFound
	}
	if len(records) == 0 {
		return nil
	}

	ns := "default"
	if opts != nil && opts.Namespace != "" {
		ns = opts.Namespace
	}

	// RAGFlow uses datasets as namespaces
	datasetID, err := rh.ensureDataset(ctx, ns)
	if err != nil {
		return err
	}

	// Upload documents to RAGFlow
	// According to RAGFlow API docs, we need to upload actual file content
	// We'll create temporary files with the document content and upload them
	uploadedDocIDs := make([]string, 0, len(records))

	for _, record := range records {
		if strings.TrimSpace(record.ID) == "" {
			return fmt.Errorf("record id cannot be empty")
		}

		// Upload document with file content using multipart/form-data
		uploadURL := fmt.Sprintf("%s/api/v1/datasets/%s/documents", rh.BaseURL, url.PathEscape(datasetID))

		// Create multipart form with file content
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Add file field with document content
		// Use record.ID as filename with .txt extension
		part, err := writer.CreateFormFile("file", record.ID+".txt")
		if err != nil {
			return fmt.Errorf("failed to create form file: %w", err)
		}

		// Write document content to the file part
		if _, err := part.Write([]byte(record.Content)); err != nil {
			return fmt.Errorf("failed to write document content: %w", err)
		}

		// Close the writer to finalize the form
		if err := writer.Close(); err != nil {
			return fmt.Errorf("failed to close multipart writer: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, body)
		if err != nil {
			return fmt.Errorf("failed to create request for document %s: %w", record.ID, err)
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rh.APIKey))

		resp, err := rh.HTTPClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to upload document %s: %w", record.ID, err)
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// Accept various success status codes
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("ragflow upload failed for document %s: status=%d, body=%s", record.ID, resp.StatusCode, string(respBody))
		}

		uploadedDocIDs = append(uploadedDocIDs, record.ID)
	}

	// Parse uploaded documents to make them searchable
	if len(uploadedDocIDs) > 0 {
		parseURL := fmt.Sprintf("%s/api/v1/datasets/%s/documents/run", rh.BaseURL, url.PathEscape(datasetID))

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, parseURL, bytes.NewReader([]byte("{}")))
		if err != nil {
			return fmt.Errorf("failed to create parse request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rh.APIKey))

		resp, err := rh.HTTPClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to parse documents: %w", err)
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("ragflow parse failed: status=%d, body=%s", resp.StatusCode, string(respBody))
		}
	}

	return nil
}

// Query searches documents in RAGFlow knowledge base
func (rh *RAGFlowHandler) Query(ctx context.Context, text string, opts *QueryOptions) ([]QueryResult, error) {
	if rh == nil {
		return nil, ErrHandlerNotFound
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyQuery
	}

	topK := 10
	minScore := 0.0
	ns := "default"

	if opts != nil {
		if opts.TopK > 0 {
			topK = opts.TopK
		}
		if opts.MinScore > 0 {
			minScore = opts.MinScore
		}
		if opts.Namespace != "" {
			ns = opts.Namespace
		}
	}

	// Get or create dataset
	datasetID, err := rh.ensureDataset(ctx, ns)
	if err != nil {
		return nil, err
	}

	// Search in RAGFlow using the search endpoint
	// According to RAGFlow API docs, search uses POST with query and top_k parameters
	endpoint := fmt.Sprintf("%s/api/v1/datasets/%s/search", rh.BaseURL, url.PathEscape(datasetID))

	reqBody := map[string]any{
		"query": text,
		"top_k": topK,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rh.APIKey))

	resp, err := rh.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()

	var response *http.Response
	var respBody []byte
	var lastErr error

	if resp.StatusCode == http.StatusOK {
		response = resp
	} else {
		respBody, _ = io.ReadAll(resp.Body)
		lastErr = fmt.Errorf("search failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	if response == nil {
		if lastErr != nil {
			return nil, fmt.Errorf("ragflow search failed: %v", lastErr)
		}
		return nil, fmt.Errorf("ragflow search failed: no valid endpoint found")
	}

	// Read response body for error handling
	if respBody == nil {
		respBody, _ = io.ReadAll(response.Body)
	}

	// Try to parse as generic JSON first to understand the structure
	var rawResp map[string]any
	if err := json.Unmarshal(respBody, &rawResp); err != nil {
		return nil, fmt.Errorf("ragflow search decode failed: %w, body=%s", err, string(respBody))
	}

	// Handle different response formats
	var results []QueryResult

	// Check for code field
	code := int64(0)
	if codeVal, ok := rawResp["code"]; ok {
		switch v := codeVal.(type) {
		case float64:
			code = int64(v)
		case int:
			code = int64(v)
		}
	}

	if code != 0 {
		// code=101 usually means dataset not found or empty
		if code == 101 {
			return []QueryResult{}, nil
		}
		return nil, fmt.Errorf("ragflow search error: code=%d, body=%s", code, string(respBody))
	}

	// Extract chunks from various possible response formats
	var chunks []map[string]any

	if data, ok := rawResp["data"]; ok {
		switch dataVal := data.(type) {
		case map[string]any:
			// data is an object, look for chunks
			if chunksVal, ok := dataVal["chunks"]; ok {
				if chunksArray, ok := chunksVal.([]any); ok {
					for _, chunk := range chunksArray {
						if chunkMap, ok := chunk.(map[string]any); ok {
							chunks = append(chunks, chunkMap)
						}
					}
				}
			}
		case []any:
			// data is directly an array
			for _, chunk := range dataVal {
				if chunkMap, ok := chunk.(map[string]any); ok {
					chunks = append(chunks, chunkMap)
				}
			}
		}
	}

	// Parse chunks into results
	results = make([]QueryResult, 0, len(chunks))
	for _, chunk := range chunks {
		var score float64
		if scoreVal, ok := chunk["similarity"]; ok {
			if scoreFloat, ok := scoreVal.(float64); ok {
				score = scoreFloat
			}
		} else if scoreVal, ok := chunk["score"]; ok {
			if scoreFloat, ok := scoreVal.(float64); ok {
				score = scoreFloat
			}
		}

		if score < minScore {
			continue
		}

		var content string
		if contentVal, ok := chunk["content"]; ok {
			content, _ = contentVal.(string)
		}

		var docID string
		if docIDVal, ok := chunk["doc_id"]; ok {
			docID, _ = docIDVal.(string)
		} else if idVal, ok := chunk["id"]; ok {
			docID, _ = idVal.(string)
		}

		var metadata map[string]any
		if metaVal, ok := chunk["metadata"]; ok {
			metadata, _ = metaVal.(map[string]any)
		}

		record := Record{
			ID:       docID,
			Content:  content,
			Metadata: metadata,
		}

		results = append(results, QueryResult{
			Record: record,
			Score:  score,
		})
	}

	return results, nil
}

// Get retrieves records by IDs
func (rh *RAGFlowHandler) Get(ctx context.Context, ids []string, opts *GetOptions) ([]Record, error) {
	if rh == nil {
		return nil, ErrHandlerNotFound
	}
	if len(ids) == 0 {
		return nil, nil
	}

	ns := "default"
	if opts != nil && opts.Namespace != "" {
		ns = opts.Namespace
	}

	datasetID, err := rh.ensureDataset(ctx, ns)
	if err != nil {
		return nil, err
	}

	results := make([]Record, 0, len(ids))

	for _, id := range ids {
		reqURL := fmt.Sprintf("%s/api/v1/datasets/%s/documents/%s", rh.BaseURL, url.PathEscape(datasetID), url.PathEscape(id))

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rh.APIKey))

		resp, err := rh.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("ragflow get failed: status=%d body=%s", resp.StatusCode, string(respBody))
		}

		var docResp struct {
			Code int `json:"code"`
			Data struct {
				ID       string         `json:"id"`
				Name     string         `json:"name"`
				Content  string         `json:"content"`
				Metadata map[string]any `json:"metadata"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&docResp); err != nil {
			return nil, err
		}

		if docResp.Code == 0 {
			record := Record{
				ID:       docResp.Data.ID,
				Title:    docResp.Data.Name,
				Content:  docResp.Data.Content,
				Metadata: docResp.Data.Metadata,
			}
			results = append(results, record)
		}
	}

	return results, nil
}

// List lists all records in a namespace
func (rh *RAGFlowHandler) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	if rh == nil {
		return nil, ErrHandlerNotFound
	}

	ns := "default"
	if opts != nil && opts.Namespace != "" {
		ns = opts.Namespace
	}

	datasetID, err := rh.ensureDataset(ctx, ns)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("%s/api/v1/datasets/%s/documents", rh.BaseURL, url.PathEscape(datasetID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rh.APIKey))

	resp, err := rh.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ragflow list failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var listResp struct {
		Code int `json:"code"`
		Data struct {
			Documents []struct {
				ID      string `json:"id"`
				Name    string `json:"name"`
				Content string `json:"content"`
			} `json:"documents"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}

	if listResp.Code != 0 {
		return nil, fmt.Errorf("ragflow list error: code=%d", listResp.Code)
	}

	records := make([]Record, 0, len(listResp.Data.Documents))
	for _, doc := range listResp.Data.Documents {
		records = append(records, Record{
			ID:      doc.ID,
			Title:   doc.Name,
			Content: doc.Content,
		})
	}

	return &ListResult{
		Records: records,
	}, nil
}

// Delete removes records by IDs
func (rh *RAGFlowHandler) Delete(ctx context.Context, ids []string, opts *DeleteOptions) error {
	if rh == nil {
		return ErrHandlerNotFound
	}
	if len(ids) == 0 {
		return nil
	}

	ns := "default"
	if opts != nil && opts.Namespace != "" {
		ns = opts.Namespace
	}

	datasetID, err := rh.ensureDataset(ctx, ns)
	if err != nil {
		return err
	}

	for _, id := range ids {
		reqURL := fmt.Sprintf("%s/api/v1/datasets/%s/documents/%s", rh.BaseURL, url.PathEscape(datasetID), url.PathEscape(id))

		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rh.APIKey))

		resp, err := rh.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			respBody, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("ragflow delete failed: status=%d body=%s", resp.StatusCode, string(respBody))
		}
	}

	return nil
}

// Ping checks the health of RAGFlow service
func (rh *RAGFlowHandler) Ping(ctx context.Context) error {
	if rh == nil {
		return ErrHandlerNotFound
	}

	// Try multiple health check endpoints
	healthEndpoints := []string{
		"/v1/system/healthz",
		"/health",
		"/api/health",
		"/api/v1/health",
	}

	var lastErr error
	for _, endpoint := range healthEndpoints {
		reqURL := fmt.Sprintf("%s%s", rh.BaseURL, endpoint)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rh.APIKey))

		resp, err := rh.HTTPClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return nil
		}
		lastErr = fmt.Errorf("status=%d", resp.StatusCode)
	}

	if lastErr != nil {
		return fmt.Errorf("ragflow health check failed: %v", lastErr)
	}
	return fmt.Errorf("ragflow health check failed: no valid endpoint found")
}

// CreateNamespace creates a new dataset in RAGFlow
func (rh *RAGFlowHandler) CreateNamespace(ctx context.Context, name string) error {
	if rh == nil {
		return ErrHandlerNotFound
	}

	if strings.TrimSpace(name) == "" {
		return ErrNamespaceNotFound
	}

	reqBody := map[string]any{
		"name": name,
	}

	reqURL := fmt.Sprintf("%s/api/v1/datasets", rh.BaseURL)
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rh.APIKey))

	resp, err := rh.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ragflow create dataset failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

// DeleteNamespace deletes a dataset from RAGFlow
func (rh *RAGFlowHandler) DeleteNamespace(ctx context.Context, name string) error {
	if rh == nil {
		return ErrHandlerNotFound
	}

	if strings.TrimSpace(name) == "" {
		return ErrNamespaceNotFound
	}

	datasetID, err := rh.getDatasetID(ctx, name)
	if err != nil {
		return err
	}

	if datasetID == "" {
		return nil
	}

	reqURL := fmt.Sprintf("%s/api/v1/datasets/%s", rh.BaseURL, url.PathEscape(datasetID))

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rh.APIKey))

	resp, err := rh.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ragflow delete dataset failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ListNamespaces lists all datasets in RAGFlow
func (rh *RAGFlowHandler) ListNamespaces(ctx context.Context) ([]string, error) {
	if rh == nil {
		return nil, ErrHandlerNotFound
	}

	reqURL := fmt.Sprintf("%s/api/v1/datasets", rh.BaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rh.APIKey))

	resp, err := rh.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ragflow list datasets failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}

	// First try to decode as generic JSON to handle flexible response format
	var rawResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("ragflow list datasets decode failed: %w", err)
	}

	// Extract datasets from response
	namespaces := make([]string, 0)

	// Handle different response formats
	if data, ok := rawResp["data"]; ok {
		if dataMap, ok := data.(map[string]interface{}); ok {
			// Try to get datasets array
			if datasets, ok := dataMap["datasets"]; ok {
				if datasetsArray, ok := datasets.([]interface{}); ok {
					for _, ds := range datasetsArray {
						if dsMap, ok := ds.(map[string]interface{}); ok {
							if name, ok := dsMap["name"].(string); ok {
								namespaces = append(namespaces, name)
							}
						}
					}
				}
			}
		} else if dataArray, ok := data.([]interface{}); ok {
			// If data is directly an array
			for _, ds := range dataArray {
				if dsMap, ok := ds.(map[string]interface{}); ok {
					if name, ok := dsMap["name"].(string); ok {
						namespaces = append(namespaces, name)
					}
				}
			}
		}
	}

	return namespaces, nil
}

// Helper functions

// ensureDataset gets or creates a dataset
func (rh *RAGFlowHandler) ensureDataset(ctx context.Context, name string) (string, error) {
	// Try to get existing dataset
	datasetID, err := rh.getDatasetID(ctx, name)
	if err == nil && datasetID != "" {
		return datasetID, nil
	}

	// Create new dataset if not found
	if err := rh.CreateNamespace(ctx, name); err != nil {
		return "", err
	}

	// Get the newly created dataset ID
	return rh.getDatasetID(ctx, name)
}

// getDatasetID retrieves the ID of a dataset by name
func (rh *RAGFlowHandler) getDatasetID(ctx context.Context, name string) (string, error) {
	datasets, err := rh.ListNamespaces(ctx)
	if err != nil {
		return "", err
	}

	// Case-insensitive comparison for dataset name
	lowerName := strings.ToLower(name)
	for _, ds := range datasets {
		if strings.ToLower(ds) == lowerName {
			// Return the actual dataset name from the server
			// This ensures we use the correct case
			return ds, nil
		}
	}

	return "", nil
}
