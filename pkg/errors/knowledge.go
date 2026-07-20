package errors

import "errors"

// Copyright (c) 2026 LingByte
// SPDX-License-Identifier: MIT

var ErrEmptyQrantBaseUrl = errors.New("QDRANT_BASEURL is required for qdrant provider")
var ErrEmptyMilvusAddress = errors.New("MILVUS_ADDRESS is required for milvus provider")
var ErrEmptyPostgresDsn = errors.New("POSTGRES_DSN is required for postgres provider")
var ErrEmptyWeaviateUrl = errors.New("WEAVIATE_URL is required for weaviate provider")
var ErrEmptyElasticSearchUrl = errors.New("ELASTICSEARCH_URL is required for elasticsearch provider")
var ErrEmptyEmbedApiKey = errors.New("EMBED_API_KEY is required")
