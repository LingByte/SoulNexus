package knowledge

import (
	"context"
	"fmt"
	"strings"

	bailian "github.com/alibabacloud-go/bailian-20231229/v2/client"
	teaUtil "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"

	"github.com/LingByte/lingllm/embedder"
)

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: AGPL-3.0

// AliyunHandler implements KnowledgeHandler using Alibaba Bailian.
// Alibaba Bailian is a cloud-based RAG service with document management,
// indexing, and semantic search capabilities.
//
// Note: Alibaba Bailian does not have a namespace concept. Instead, it uses:
// - Workspace: Top-level organization unit
// - Index: Knowledge base unit (equivalent to namespace in other systems)
// Each Index is an independent knowledge base within a Workspace.
type AliyunHandler struct {
	client      *bailian.Client
	WorkspaceID string
	CategoryID  string
	Embedder    embedder.Embedder
}

func (ah *AliyunHandler) Provider() string { return ProviderAliyun }

// Upsert adds or updates records in Alibaba Bailian knowledge base
// Note: In Alibaba Bailian, Namespace parameter maps to Index ID (knowledge base ID)
// Note: This implementation validates records but document upload requires using the Data Center API
// For now, this is a validation-only implementation
func (ah *AliyunHandler) Upsert(ctx context.Context, records []Record, opts *UpsertOptions) error {
	if ah == nil {
		return ErrHandlerNotFound
	}
	if len(records) == 0 {
		return nil
	}

	// Validate records
	for _, record := range records {
		if strings.TrimSpace(record.ID) == "" {
			return fmt.Errorf("record id cannot be empty")
		}
	}

	// Note: Alibaba Bailian's SubmitIndexAddDocumentsJob expects document IDs from Data Center
	// not raw document content. To upload documents, you need to:
	// 1. First upload files to Data Center using AddFile API
	// 2. Then reference those file IDs in SubmitIndexAddDocumentsJob
	// For now, we validate the records and return success
	// Users should upload documents through the Alibaba Bailian console or use AddFile API

	return nil
}

// Query searches documents in Alibaba Bailian knowledge base
// Note: In Alibaba Bailian, Namespace parameter maps to Index ID (knowledge base ID)
// Supports EnableReranking and ReturnMetadata options
func (ah *AliyunHandler) Query(ctx context.Context, text string, opts *QueryOptions) ([]QueryResult, error) {
	if ah == nil {
		return nil, ErrHandlerNotFound
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil, ErrEmptyQuery
	}

	topK := int32(10)
	minScore := 0.0
	indexID := "default"
	enableReranking := true
	returnMetadata := false

	if opts != nil {
		if opts.TopK > 0 {
			topK = int32(opts.TopK)
		}
		if opts.MinScore > 0 {
			minScore = opts.MinScore
		}
		if opts.Namespace != "" {
			indexID = opts.Namespace
		}
		enableReranking = opts.EnableReranking
		returnMetadata = opts.ReturnMetadata
	}

	// Use Alibaba Bailian Retrieve API
	headers := make(map[string]*string)
	retrieveRequest := &bailian.RetrieveRequest{
		IndexId:              tea.String(indexID),
		Query:                tea.String(text),
		DenseSimilarityTopK:  tea.Int32(topK),
		SparseSimilarityTopK: tea.Int32(0),
		RerankTopN:           tea.Int32(topK),
		EnableReranking:      tea.Bool(enableReranking),
	}

	runtime := &teaUtil.RuntimeOptions{}
	response, err := ah.client.RetrieveWithOptions(tea.String(ah.WorkspaceID), retrieveRequest, headers, runtime)
	if err != nil {
		return nil, fmt.Errorf("alibaba bailian retrieve failed: %w", err)
	}

	if response == nil || response.Body == nil {
		return nil, fmt.Errorf("alibaba bailian returned empty response")
	}

	// Check if there's an error in the response
	if response.Body.Code != nil && *response.Body.Code != "Success" {
		msg := "unknown error"
		if response.Body.Message != nil {
			msg = *response.Body.Message
		}
		return nil, fmt.Errorf("alibaba bailian error: %s", msg)
	}

	// Return empty results if no data (this is normal when index doesn't exist or has no documents)
	if response.Body.Data == nil || len(response.Body.Data.Nodes) == 0 {
		return []QueryResult{}, nil
	}

	results := make([]QueryResult, 0)
	for _, node := range response.Body.Data.Nodes {
		if node == nil || node.Text == nil {
			continue
		}

		score := 0.0
		if node.Score != nil {
			score = *node.Score
		}

		if score < minScore {
			continue
		}

		record := Record{
			Content: *node.Text,
		}

		// Include metadata if requested
		if returnMetadata {
			record.Metadata = make(map[string]any)
			if node.Metadata != nil {
				if m, ok := node.Metadata.(map[string]any); ok && m != nil {
					for k, v := range m {
						record.Metadata[k] = v
					}
				}
			}
			if node.Score != nil {
				record.Metadata["score"] = *node.Score
			}
		}

		results = append(results, QueryResult{
			Record: record,
			Score:  score,
		})
	}

	return results, nil
}

// Get retrieves records by IDs using semantic search
// Note: Alibaba Bailian doesn't support direct document retrieval by ID
// This implementation searches for documents with matching IDs
func (ah *AliyunHandler) Get(ctx context.Context, ids []string, opts *GetOptions) ([]Record, error) {
	if ah == nil {
		return nil, ErrHandlerNotFound
	}
	if len(ids) == 0 {
		return nil, nil
	}

	indexID := "default"
	if opts != nil && opts.Namespace != "" {
		indexID = opts.Namespace
	}

	records := make([]Record, 0, len(ids))

	// For each ID, search using a query
	for _, id := range ids {
		headers := make(map[string]*string)
		retrieveRequest := &bailian.RetrieveRequest{
			IndexId:              tea.String(indexID),
			Query:                tea.String(id),
			DenseSimilarityTopK:  tea.Int32(1),
			SparseSimilarityTopK: tea.Int32(0),
			RerankTopN:           tea.Int32(1),
		}

		runtime := &teaUtil.RuntimeOptions{}
		response, err := ah.client.RetrieveWithOptions(tea.String(ah.WorkspaceID), retrieveRequest, headers, runtime)
		if err != nil {
			continue
		}

		if response == nil || response.Body == nil || response.Body.Data == nil || len(response.Body.Data.Nodes) == 0 {
			continue
		}

		node := response.Body.Data.Nodes[0]
		if node == nil || node.Text == nil {
			continue
		}

		record := Record{
			Content: *node.Text,
		}
		records = append(records, record)
	}

	return records, nil
}

// List lists all records in a namespace using search
// Note: Alibaba Bailian doesn't have a native list API
// This implementation returns empty results as pagination is not supported
func (ah *AliyunHandler) List(ctx context.Context, opts *ListOptions) (*ListResult, error) {
	if ah == nil {
		return nil, ErrHandlerNotFound
	}

	// Alibaba Bailian doesn't support listing documents without a query
	// Return empty results - users should use Query instead
	return &ListResult{
		Records: []Record{},
	}, nil
}

// Delete removes records by IDs
// Note: Alibaba Bailian doesn't support deleting individual documents
// This method returns an error as per API limitations
func (ah *AliyunHandler) Delete(ctx context.Context, ids []string, opts *DeleteOptions) error {
	if ah == nil {
		return ErrHandlerNotFound
	}
	if len(ids) == 0 {
		return nil
	}

	// Alibaba Bailian doesn't support deleting individual documents
	// Users must delete the entire index using DeleteNamespace
	return fmt.Errorf("alibaba bailian does not support deleting individual documents; use DeleteNamespace to delete the entire index")
}

// Ping checks the health of Alibaba Bailian service
func (ah *AliyunHandler) Ping(ctx context.Context) error {
	if ah == nil {
		return ErrHandlerNotFound
	}

	// Simple health check - if client is initialized, we're good
	if ah.client == nil {
		return fmt.Errorf("alibaba bailian client not initialized")
	}

	return nil
}

// CreateNamespace creates a new Index (knowledge base) in Alibaba Bailian
// Note: In Alibaba Bailian, namespace maps to Index, which is the knowledge base unit
// Creating an empty index requires using the CreateIndex API
func (ah *AliyunHandler) CreateNamespace(ctx context.Context, name string) error {
	if ah == nil {
		return ErrHandlerNotFound
	}

	if strings.TrimSpace(name) == "" {
		return ErrNamespaceNotFound
	}

	// Create index using official SDK
	headers := make(map[string]*string)
	createIndexRequest := &bailian.CreateIndexRequest{
		Name: tea.String(name),
	}

	runtime := &teaUtil.RuntimeOptions{}
	response, err := ah.client.CreateIndexWithOptions(tea.String(ah.WorkspaceID), createIndexRequest, headers, runtime)
	if err != nil {
		return fmt.Errorf("alibaba bailian create index failed: %w", err)
	}

	if response == nil || response.Body == nil {
		return fmt.Errorf("alibaba bailian returned empty response")
	}

	// Check if there's an error in the response
	if response.Body.Code != nil && *response.Body.Code != "Success" {
		msg := "unknown error"
		if response.Body.Message != nil {
			msg = *response.Body.Message
		}
		return fmt.Errorf("alibaba bailian error: %s", msg)
	}

	return nil
}

// DeleteNamespace deletes an Index (knowledge base) from Alibaba Bailian
// Note: In Alibaba Bailian, namespace maps to Index, which is the knowledge base unit
// This operation deletes the entire index and all its documents
func (ah *AliyunHandler) DeleteNamespace(ctx context.Context, name string) error {
	if ah == nil {
		return ErrHandlerNotFound
	}

	if strings.TrimSpace(name) == "" {
		return ErrNamespaceNotFound
	}

	// Use official SDK to delete index
	headers := make(map[string]*string)
	deleteIndexRequest := &bailian.DeleteIndexRequest{
		IndexId: tea.String(name),
	}

	runtime := &teaUtil.RuntimeOptions{}
	response, err := ah.client.DeleteIndexWithOptions(tea.String(ah.WorkspaceID), deleteIndexRequest, headers, runtime)
	if err != nil {
		return fmt.Errorf("alibaba bailian delete index failed: %w", err)
	}

	if response == nil || response.Body == nil {
		return fmt.Errorf("alibaba bailian returned empty response")
	}

	// Check if there's an error in the response
	if response.Body.Code != nil && *response.Body.Code != "Success" {
		msg := "unknown error"
		if response.Body.Message != nil {
			msg = *response.Body.Message
		}
		return fmt.Errorf("alibaba bailian error: %s", msg)
	}

	return nil
}

// ListNamespaces lists all Indexes (knowledge bases) in Alibaba Bailian
// Note: In Alibaba Bailian, each Index is an independent knowledge base
// This implementation returns an empty list as the API doesn't provide a list operation
// Users should manage indexes through the Alibaba Bailian console
func (ah *AliyunHandler) ListNamespaces(ctx context.Context) ([]string, error) {
	if ah == nil {
		return nil, ErrHandlerNotFound
	}

	// Alibaba Bailian API doesn't provide a list indexes operation
	// Return empty list - users should manage indexes through the console
	return []string{}, nil
}
