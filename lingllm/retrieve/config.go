package retrieve

import (
	"context"
	"fmt"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// SearchEngine defines the interface for full-text search.
type SearchEngine interface {
	Search(ctx context.Context, query string, fields []string, size int) ([]SearchHit, error)
}

// SearchHit represents a search result.
type SearchHit struct {
	ID     string
	Score  float64
	Fields map[string]interface{}
}

// Config configures strategy-based retrieval over vector store + search engine.
type Config struct {
	Strategy Strategy
	Vector   VectorRetriever
	Search   SearchEngine
	TopK     int
	MinScore float64
	// KeywordFields passed to search engine when StrategyKeyword or StrategyHybrid.
	KeywordFields []string
	// VectorWeight is the vector score weight in hybrid mode (0–1, default 0.65).
	VectorWeight float64
	// Reranker optionally re-scores candidate documents after the base strategy.
	Reranker Reranker
	// RerankCandidates is how many docs to fetch before reranking (default TopK*3).
	RerankCandidates int
	// Retrieval holds RRF / threshold / over-retrieval tuning (WeKnora-style).
	Retrieval RetrievalParams
	// Observer receives pipeline stage events for tracing (e.g. Langfuse).
	Observer Observer
}

// StrategyRetriever implements retriever with vector/keyword/hybrid strategies.
type StrategyRetriever struct {
	cfg Config
}

// New builds a strategy retriever.
func New(cfg Config) (*StrategyRetriever, error) {
	if cfg.Strategy == "" {
		cfg.Strategy = StrategyVector
	}
	if cfg.TopK <= 0 {
		cfg.TopK = 3
	}
	if cfg.VectorWeight <= 0 {
		cfg.VectorWeight = 0.65
	}
	if len(cfg.KeywordFields) == 0 {
		cfg.KeywordFields = []string{"title", "content", "body"}
	}
	if cfg.Retrieval == (RetrievalParams{}) {
		cfg.Retrieval = DefaultRetrievalParams()
	}

	switch cfg.Strategy {
	case StrategyVector:
		if cfg.Vector == nil {
			return nil, fmt.Errorf("retrieve: vector retriever required for vector strategy")
		}
	case StrategyKeyword:
		if cfg.Search == nil {
			return nil, fmt.Errorf("retrieve: search engine required for keyword strategy")
		}
	case StrategyHybrid:
		if cfg.Vector == nil || cfg.Search == nil {
			return nil, fmt.Errorf("retrieve: hybrid strategy requires both vector retriever and search engine")
		}
	default:
		return nil, fmt.Errorf("retrieve: unsupported strategy %q", cfg.Strategy)
	}

	return &StrategyRetriever{cfg: cfg}, nil
}
