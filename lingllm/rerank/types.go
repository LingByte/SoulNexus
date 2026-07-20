package rerank

import (
	"context"
	"net/http"
	"time"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// RerankResult represents a reranked document with its score
type RerankResult struct {
	Index int
	Score float64
}

// Reranker is the interface for reranking documents
type Reranker interface {
	// Rerank reranks documents based on query relevance
	Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error)

	// Provider returns the provider name
	Provider() string
}

// RerankConfig is the configuration for creating a reranker
type RerankConfig struct {
	Provider      string
	BaseURL       string
	APIKey        string
	SecretKey     string
	Model         string
	Timeout       int
	HTTPClient    *http.Client
	CustomHeaders map[string]string
	ExtraConfig   map[string]string
}

// RerankClientConfig holds common configuration for rerank clients
type RerankClientConfig struct {
	BaseURL       string
	APIKey        string
	SecretKey     string
	Model         string
	Timeout       time.Duration
	HTTPClient    *http.Client
	CustomHeaders map[string]string
	ExtraConfig   map[string]string
}

// DefaultTimeout is the default timeout for rerank requests
const DefaultTimeout = 30 * time.Second

// Common errors
var (
	ErrNilClient      = "client is nil"
	ErrEmptyBaseURL   = "BaseURL is required"
	ErrEmptyAPIKey    = "APIKey is required"
	ErrEmptyModel     = "Model is required"
	ErrEmptyQuery     = "query is empty"
	ErrEmptyDocuments = "documents is empty"
	ErrInvalidTopN    = "topN must be greater than 0"
)
