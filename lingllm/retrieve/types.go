package retrieve

import "context"

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Strategy selects how indexed knowledge is queried.
type Strategy string

const (
	// StrategyVector uses dense vector similarity (Qdrant/Milvus).
	StrategyVector Strategy = "vector"
	// StrategyKeyword uses full-text search (Bleve via the search package).
	StrategyKeyword Strategy = "keyword"
	// StrategyHybrid merges vector and keyword scores.
	StrategyHybrid Strategy = "hybrid"
)

// VectorRetriever runs dense vector queries against a knowledge store.
type VectorRetriever interface {
	Retrieve(ctx context.Context, query string, topK int) ([]*Document, error)
}

// Document represents a retrieved document.
type Document struct {
	ID       string
	Content  string
	Score    float64
	Metadata map[string]string
}

// Reranker re-scores candidate documents.
type Reranker interface {
	Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)
}

// RerankResult represents a reranked document.
type RerankResult struct {
	Index int
	Score float64
}
