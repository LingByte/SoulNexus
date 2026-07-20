package knowledge

import (
	"context"
	"fmt"

	"github.com/LingByte/lingllm/embedder"
	"github.com/LingByte/lingllm/retrieve"
	"github.com/LingByte/lingllm/search"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// KnowledgeBase integrates embedder, search, retrieve, and vector database
// to provide a complete knowledge management solution.
type KnowledgeBase struct {
	handler     KnowledgeHandler
	embedder    embedder.Embedder
	searcher    search.Engine
	retriever   retrieve.StrategyRetriever
	detector    DocumentTypeDetector
	chunkers    map[DocumentType]Chunker
	namespace   string // Default namespace for queries
	queryCache  *QueryCache
	vectorCache *VectorCache
	enableCache bool
}

// KnowledgeBaseConfig configuration for KnowledgeBase
type KnowledgeBaseConfig struct {
	// Vector database handler
	Handler KnowledgeHandler

	// Text embedder for semantic search
	Embedder embedder.Embedder

	// Full-text search engine
	Searcher search.Engine

	// Multi-strategy retriever
	Retriever retrieve.StrategyRetriever

	// Document type detector
	Detector DocumentTypeDetector

	// Chunkers for different document types
	Chunkers map[DocumentType]Chunker

	// Default namespace for queries (optional)
	Namespace string

	// Enable query result caching (optional, default: true)
	EnableCache bool

	// Query cache size (optional, default: 1000)
	QueryCacheSize int

	// Vector cache size (optional, default: 10000)
	VectorCacheSize int
}

// NewKnowledgeBase creates a new knowledge base instance
func NewKnowledgeBase(cfg KnowledgeBaseConfig) (*KnowledgeBase, error) {
	if cfg.Handler == nil {
		return nil, fmt.Errorf("Handler is required")
	}

	// Default cache settings
	enableCache := cfg.EnableCache
	queryCacheSize := cfg.QueryCacheSize
	if queryCacheSize <= 0 {
		queryCacheSize = 1000
	}
	vectorCacheSize := cfg.VectorCacheSize
	if vectorCacheSize <= 0 {
		vectorCacheSize = 10000
	}

	kb := &KnowledgeBase{
		handler:     cfg.Handler,
		embedder:    cfg.Embedder,
		searcher:    cfg.Searcher,
		retriever:   cfg.Retriever,
		detector:    cfg.Detector,
		chunkers:    cfg.Chunkers,
		namespace:   cfg.Namespace,
		enableCache: enableCache,
	}

	// Initialize caches if enabled
	if enableCache {
		kb.queryCache = NewQueryCache(queryCacheSize, 0)
		kb.vectorCache = NewVectorCache(vectorCacheSize)
	}

	if kb.chunkers == nil {
		kb.chunkers = make(map[DocumentType]Chunker)
	}

	return kb, nil
}

// AddDocument adds a document to the knowledge base
// It chunks the document, generates embeddings, and stores in both vector and search engines
func (kb *KnowledgeBase) AddDocument(ctx context.Context, docID, title, content string, metadata map[string]any) error {
	if content == "" {
		return ErrEmptyText
	}

	// Detect document type
	docType := DocumentTypeStructured
	if kb.detector != nil {
		detected, err := kb.detector.DetectDocumentType(ctx, content)
		if err == nil {
			docType = detected
		}
	}

	// Chunk document
	chunks, err := kb.chunkDocument(ctx, content, title, docType)
	if err != nil {
		return fmt.Errorf("failed to chunk document: %w", err)
	}

	if len(chunks) == 0 {
		return ErrNoChunks
	}

	// Generate embeddings and create records
	records := make([]Record, 0, len(chunks))
	for i, chunk := range chunks {
		chunkID := fmt.Sprintf("%s#%d", docID, i)

		// Generate embedding if embedder is available
		var vector []float32
		if kb.embedder != nil {
			vec, err := kb.embedder.EmbedSingle(ctx, chunk.Text)
			if err != nil {
				return fmt.Errorf("failed to embed chunk %d: %w", i, err)
			}
			vector = vec
		}

		record := Record{
			ID:       chunkID,
			Source:   docID,
			Title:    chunk.Title,
			Content:  chunk.Text,
			Vector:   vector,
			Tags:     []string{},
			Metadata: chunk.Metadata,
		}

		if metadata != nil {
			if record.Metadata == nil {
				record.Metadata = make(map[string]any)
			}
			for k, v := range metadata {
				record.Metadata[k] = v
			}
		}

		records = append(records, record)
	}

	// Upsert to vector database
	err = kb.handler.Upsert(ctx, records, &UpsertOptions{
		Namespace: docID,
		Overwrite: true,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert records: %w", err)
	}

	// Index in search engine if available
	if kb.searcher != nil {
		searchDocs := make([]search.Doc, 0, len(records))
		for _, record := range records {
			searchDocs = append(searchDocs, search.Doc{
				ID:   record.ID,
				Type: "chunk",
				Fields: map[string]interface{}{
					"title":    record.Title,
					"content":  record.Content,
					"source":   record.Source,
					"metadata": record.Metadata,
				},
			})
		}

		err = kb.searcher.IndexBatch(ctx, searchDocs)
		if err != nil {
			return fmt.Errorf("failed to index in search engine: %w", err)
		}
	}

	return nil
}

// Query searches the knowledge base using hybrid retrieval
func (kb *KnowledgeBase) Query(ctx context.Context, query string, topK int) ([]QueryResult, error) {
	if query == "" {
		return nil, ErrEmptyQuery
	}

	if topK <= 0 {
		topK = 10
	}

	// Check query cache first
	if kb.enableCache && kb.queryCache != nil {
		if results, ok := kb.queryCache.Get(ctx, query); ok {
			return results, nil
		}
	}

	// Use vector database for semantic search (primary method)
	if kb.handler != nil {
		results, err := kb.handler.Query(ctx, query, &QueryOptions{
			TopK:      topK,
			Namespace: kb.namespace,
		})
		if err != nil {
			// If handler returns error, try fallback to search engine
			// Otherwise return the error from handler
			if kb.searcher == nil {
				return nil, err
			}
		} else if len(results) > 0 {
			// Cache the results
			if kb.enableCache && kb.queryCache != nil {
				kb.queryCache.Set(ctx, query, results)
			}
			return results, nil
		}
	}

	// Fallback to search engine if available
	if kb.searcher != nil {
		result, err := kb.searcher.Search(ctx, search.SearchRequest{
			Keyword: query,
			Size:    topK,
		})
		if err == nil && result.Total > 0 {
			queryResults := make([]QueryResult, 0, len(result.Hits))
			for _, hit := range result.Hits {
				record := Record{
					ID:       hit.ID,
					Title:    hit.Fields["title"].(string),
					Content:  hit.Fields["content"].(string),
					Metadata: hit.Fields,
				}
				queryResults = append(queryResults, QueryResult{
					Record: record,
					Score:  hit.Score,
				})
			}
			return queryResults, nil
		}
	}

	// If we have a handler but no results and no searcher, return empty results
	if kb.handler != nil {
		return []QueryResult{}, nil
	}

	return nil, fmt.Errorf("no search engine available")
}

// chunkDocument chunks a document based on its type
func (kb *KnowledgeBase) chunkDocument(ctx context.Context, content, title string, docType DocumentType) ([]Chunk, error) {
	chunker, ok := kb.chunkers[docType]
	if !ok {
		// Return single chunk if no specific chunker
		return []Chunk{
			{
				Index:    0,
				Title:    title,
				Text:     content,
				Metadata: map[string]any{"type": docType},
			},
		}, nil
	}

	opts := &ChunkOptions{
		DocumentTitle: title,
	}

	chunks, err := chunker.Chunk(ctx, content, opts)
	if err != nil {
		return nil, err
	}

	return chunks, nil
}

// DeleteDocument removes a document from the knowledge base
func (kb *KnowledgeBase) DeleteDocument(ctx context.Context, docID string) error {
	if docID == "" {
		return fmt.Errorf("docID is required")
	}

	// Delete from vector database
	if kb.handler != nil {
		err := kb.handler.DeleteNamespace(ctx, docID)
		if err != nil && err != ErrNamespaceNotFound {
			return fmt.Errorf("failed to delete from vector database: %w", err)
		}
	}

	// Delete from search engine
	if kb.searcher != nil {
		// Search for all chunks with this source
		result, err := kb.searcher.Search(ctx, search.SearchRequest{
			MustTerms: map[string][]string{
				"source": {docID},
			},
			Size: 1000,
		})
		if err == nil && result.Total > 0 {
			ids := make([]string, 0, len(result.Hits))
			for _, hit := range result.Hits {
				ids = append(ids, hit.ID)
			}
			// Note: search engine may not support batch delete, handle individually
			for _, id := range ids {
				_ = kb.searcher.Delete(ctx, id)
			}
		}
	}

	return nil
}

// Health checks the health of all components
func (kb *KnowledgeBase) Health(ctx context.Context) error {
	if kb.handler != nil {
		if err := kb.handler.Ping(ctx); err != nil {
			return fmt.Errorf("vector database health check failed: %w", err)
		}
	}

	return nil
}

// Close closes all resources
func (kb *KnowledgeBase) Close() error {
	// Clear caches
	if kb.queryCache != nil {
		kb.queryCache.Clear()
	}
	if kb.vectorCache != nil {
		kb.vectorCache.Clear()
	}

	if kb.embedder != nil {
		if err := kb.embedder.Close(); err != nil {
			return fmt.Errorf("failed to close embedder: %w", err)
		}
	}

	if kb.searcher != nil {
		if err := kb.searcher.Close(); err != nil {
			return fmt.Errorf("failed to close search engine: %w", err)
		}
	}

	return nil
}

// ClearCache clears all cached data
func (kb *KnowledgeBase) ClearCache() {
	if kb.queryCache != nil {
		kb.queryCache.Clear()
	}
	if kb.vectorCache != nil {
		kb.vectorCache.Clear()
	}
}

// CacheStats returns cache statistics
func (kb *KnowledgeBase) CacheStats() map[string]any {
	stats := make(map[string]any)
	if kb.queryCache != nil {
		stats["query_cache"] = kb.queryCache.Stats()
	}
	if kb.vectorCache != nil {
		stats["vector_cache"] = map[string]any{
			"size":     kb.vectorCache.Size(),
			"max_size": 10000,
		}
	}
	return stats
}
