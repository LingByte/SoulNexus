package knowledge

import (
	"context"
	"io"
	"mime/multipart"
)

const (
	// ProviderAliyun Aliyun Bailian Knowledge Base
	ProviderAliyun = "aliyun"
	// ProviderZilliz Milvus/Zilliz Vector Database
	ProviderZilliz = "zilliz"
	// ProviderPinecone Pinecone Vector Database
	ProviderPinecone = "pinecone"
	// ProviderQdrant Qdrant Vector Database
	ProviderQdrant = "qdrant"
	// ProviderElasticsearch Elasticsearch Full-Text Search
	ProviderElasticsearch = "elasticsearch"
)

var DefaultProvider = ProviderAliyun

// SearchResult knowledge base search result
type SearchResult struct {
	// Content retrieved content (text)
	Content string `json:"content"`
	// Score relevance score (0-1)
	Score float64 `json:"score"`
	// Metadata metadata (optional)
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// Source source identifier (document ID, filename, etc.)
	Source string `json:"source,omitempty"`
}

// SearchOptions search options
type SearchOptions struct {
	// TopK returns top K most relevant results
	TopK int `json:"top_k"`
	// Threshold relevance threshold (0-1), results below this threshold will be filtered
	Threshold float64 `json:"threshold,omitempty"`
	// Query query text
	Query string `json:"query"`
	// Filter filter conditions (optional, different providers support different formats)
	Filter map[string]interface{} `json:"filter,omitempty"`
}

// KnowledgeBase unified knowledge base interface
type KnowledgeBase interface {
	// Provider returns knowledge base provider name
	Provider() string

	// Search searches for relevant information in the knowledge base
	// knowledgeKey: knowledge base identifier (e.g., Aliyun's indexId)
	// options: search options
	// Returns: search result list, sorted by relevance
	Search(ctx context.Context, knowledgeKey string, options SearchOptions) ([]SearchResult, error)

	// CreateIndex creates knowledge base index
	// name: knowledge base name
	// config: configuration information (different providers need different configs)
	// Returns: knowledge base identifier (knowledgeKey)
	CreateIndex(ctx context.Context, name string, config map[string]interface{}) (string, error)

	// DeleteIndex deletes knowledge base index
	// knowledgeKey: knowledge base identifier
	DeleteIndex(ctx context.Context, knowledgeKey string) error

	// UploadDocument uploads document to knowledge base
	// knowledgeKey: knowledge base identifier
	// file: file content
	// metadata: document metadata (filename, type, etc.)
	UploadDocument(ctx context.Context, knowledgeKey string, file multipart.File, header *multipart.FileHeader, metadata map[string]interface{}) error

	// DeleteDocument deletes document from knowledge base
	// knowledgeKey: knowledge base identifier
	// documentID: document ID
	DeleteDocument(ctx context.Context, knowledgeKey string, documentID string) error

	// ListDocuments lists all documents in the knowledge base
	// knowledgeKey: knowledge base identifier
	// Returns: document ID list
	ListDocuments(ctx context.Context, knowledgeKey string) ([]string, error)

	// GetDocument gets document content
	// knowledgeKey: knowledge base identifier
	// documentID: document ID
	// Returns: document content stream
	GetDocument(ctx context.Context, knowledgeKey string, documentID string) (io.ReadCloser, error)
}

// Manager knowledge base manager for creating and managing knowledge base instances based on config
type Manager interface {
	// GetKnowledgeBase gets knowledge base instance by provider type (with cache)
	// provider: provider name (e.g., "aliyun", "zilliz", etc.)
	// config: configuration information (different providers need different configs)
	// Note: same provider + same config will reuse cached instance
	GetKnowledgeBase(provider string, config map[string]interface{}) (KnowledgeBase, error)

	// RegisterProvider registers a new knowledge base provider
	RegisterProvider(name string, factory func(config map[string]interface{}) (KnowledgeBase, error))

	// ListProviders lists all registered providers
	ListProviders() []string

	// ClearCache clears all cached instances
	ClearCache()

	// RemoveCachedInstance removes cached instance for specified provider and config
	RemoveCachedInstance(provider string, config map[string]interface{})
}
