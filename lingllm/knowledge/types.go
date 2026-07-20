package knowledge

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	bailian "github.com/alibabacloud-go/bailian-20231229/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	"github.com/alibabacloud-go/tea/tea"

	"github.com/LingByte/lingllm/utils"
)

const (
	// ProviderQdrant Qdrant Vector Database
	ProviderQdrant = "qdrant"

	// ProviderMilvus Milvus Vector Database
	ProviderMilvus = "milvus"

	// ProviderRAGFlow RAGFlow RAG Engine
	ProviderRAGFlow = "ragflow"

	// ProviderAliyun Alibaba Bailian Knowledge Base
	ProviderAliyun = "aliyun"

	// ProviderPostgres PostgreSQL + pgvector
	ProviderPostgres = "postgres"

	// ProviderElasticsearch Elasticsearch 8+ dense_vector kNN
	ProviderElasticsearch = "elasticsearch"

	// ProviderWeaviate Weaviate vector database
	ProviderWeaviate = "weaviate"
)

var (
	ErrHandlerNotFound        = errors.New("handler not be null")
	ErrBaseURL                = errors.New("BaseURL is required")
	ErrCollectionNotFound     = errors.New("Collection is required")
	ErrRecordNotFound         = errors.New("record not found")
	ErrNamespaceNotFound      = errors.New("namespace not found")
	ErrInvalidVectorDimension = errors.New("invalid vector dimension")
	ErrEmptyQuery             = errors.New("empty query text")
	ErrEmptyText              = errors.New("empty text")
	ErrInvalidChunkOpt        = errors.New("invalid chunk options")
	ErrNoChunks               = errors.New("no chunks generated")
	ErrChunkerNotFound        = errors.New("no suitable chunker for document type")
)

type DocumentType int

const (
	DocumentTypeUnknown      DocumentType = iota
	DocumentTypeStructured                // 有标题、章节、段落（手册、论文、markdown）
	DocumentTypeTableKV                   // 表格、键值对、表单、简历
	DocumentTypeUnstructured              // 杂乱、OCR、无标点、无段落（必须 LLM）
)

// Chunk is one retrieval-oriented segment produced by a Chunker.
type Chunk struct {
	Index    int
	Title    string
	Text     string
	Metadata map[string]any
}

// ChunkOptions controls chunk size, overlap and optional title metadata.
type ChunkOptions struct {
	// MaxChars is the target maximum characters per chunk. When 0, chunkers use their own defaults.
	MaxChars int
	// OverlapChars is the overlap size between consecutive chunks.
	// If set to -1, chunkers may disable overlap.
	OverlapChars int
	// MinChars is a lower bound; very small chunks may be dropped/merged.
	MinChars int

	DocumentTitle string

	// PreChunkClean is passed to base.CleanText before an LLM call.
	// If nil, some implementations will enable StripMarkdown and DedupLines by default.
	PreChunkClean *utils.Options
}

// Chunker splits long text into chunks (implementations may use deterministic rules or an LLM).
type Chunker interface {
	Provider() string
	Chunk(ctx context.Context, text string, opts *ChunkOptions) ([]Chunk, error)
}

// DocumentTypeDetector decides which chunking strategy should be used for a document.
type DocumentTypeDetector interface {
	DetectDocumentType(ctx context.Context, text string) (DocumentType, error)
}

type FilterOp string

const (
	FilterOpEqual       FilterOp = "$eq"
	FilterOpNotEqual    FilterOp = "$ne"
	FilterOpIn          FilterOp = "$in"
	FilterOpNotIn       FilterOp = "$nin"
	FilterOpGt          FilterOp = "$gt"
	FilterOpGte         FilterOp = "$gte"
	FilterOpLt          FilterOp = "$lt"
	FilterOpLte         FilterOp = "$lte"
	FilterOpContainsAll FilterOp = "$all"
	FilterOpContainsAny FilterOp = "$any"
)

type Filter struct {
	Field    string   `json:"field"`
	Operator FilterOp `json:"operator"`
	Value    []any    `json:"value"`
}

// Record 知识库记录
type Record struct {
	ID        string         `json:"id"`
	Source    string         `json:"source"` // 来源file/url/api etc.
	Title     string         `json:"title"`
	Content   string         `json:"content"` // 原文片段
	Vector    []float32      `json:"vector"`  // 向量
	Tags      []string       `json:"tags"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type UpsertOptions struct {
	Namespace string
	Overwrite bool
	BatchSize int
}

type QueryOptions struct {
	Namespace       string
	TopK            int
	MinScore        float64  // 分数阈值
	Filters         []Filter // 复杂过滤
	Model           string   // embedding 模型
	EnableReranking bool     // 是否启用重排序（仅 Aliyun 支持）
	ReturnMetadata  bool     // 是否返回完整元数据
}

type QueryResult struct {
	Record Record  `json:"record"`
	Score  float64 `json:"score"`
}

type GetOptions struct {
	Namespace string
}

type DeleteOptions struct {
	Namespace string
}

type ListOptions struct {
	Namespace string
	Limit     int
	Offset    string
	Filters   []Filter
	OrderBy   string // "created_at" "updated_at"
	OrderDir  string // "asc" "desc"
}

type ListResult struct {
	Records    []Record `json:"records"`
	NextOffset string   `json:"next_offset,omitempty"`
}

// QdrantConfig configuration for Qdrant provider
type QdrantConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// MilvusConfig configuration for Milvus provider
type MilvusConfig struct {
	Address  string
	Username string
	Password string
	Token    string
	DBName   string
}

// RAGFlowConfig configuration for RAGFlow provider
type RAGFlowConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// AliyunConfig configuration for Alibaba Bailian provider
type AliyunConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string
	WorkspaceID     string
	CategoryID      string
	Timeout         time.Duration
}

// PostgresConfig configuration for PostgreSQL + pgvector provider
type PostgresConfig struct {
	DSN string
}

// ElasticsearchConfig configuration for Elasticsearch provider
type ElasticsearchConfig struct {
	BaseURL     string
	APIKey      string
	IndexPrefix string
	Timeout     time.Duration
}

// WeaviateConfig configuration for Weaviate provider
type WeaviateConfig struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

// HandlerFactoryParams selects and configures a KnowledgeHandler.
type HandlerFactoryParams struct {
	// Provider is ProviderQdrant, ProviderMilvus, ProviderRAGFlow, or ProviderAliyun (see constants in this package).
	Provider string
	// Namespace is the Qdrant / Milvus collection name, RAGFlow dataset name, or Alibaba Bailian index name.
	Namespace string
	// QdrantConfig is required when Provider is ProviderQdrant
	QdrantConfig *QdrantConfig
	// MilvusConfig is required when Provider is ProviderMilvus
	MilvusConfig *MilvusConfig
	// RAGFlowConfig is required when Provider is ProviderRAGFlow
	RAGFlowConfig *RAGFlowConfig
	// AliyunConfig is required when Provider is ProviderAliyun
	AliyunConfig *AliyunConfig
	// PostgresConfig is required when Provider is ProviderPostgres
	PostgresConfig *PostgresConfig
	// ElasticsearchConfig is required when Provider is ProviderElasticsearch
	ElasticsearchConfig *ElasticsearchConfig
	// WeaviateConfig is required when Provider is ProviderWeaviate
	WeaviateConfig *WeaviateConfig
}

// NewKnowledgeHandler returns a backend implementation for the given namespace configuration.
func NewKnowledgeHandler(p HandlerFactoryParams) (KnowledgeHandler, error) {
	switch p.Provider {
	case ProviderQdrant:
		if p.QdrantConfig == nil {
			return nil, errors.New("QdrantConfig is required for Qdrant provider")
		}
		timeout := p.QdrantConfig.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		qh := &QdrantHandler{
			BaseURL: p.QdrantConfig.BaseURL,
			APIKey:  p.QdrantConfig.APIKey,
			// Reuse idle connections across voice turns (same process ↔ Qdrant).
			HTTPClient: &http.Client{
				Timeout: timeout,
				Transport: &http.Transport{
					Proxy:                 http.ProxyFromEnvironment,
					MaxIdleConns:          100,
					MaxIdleConnsPerHost:   16,
					IdleConnTimeout:       90 * time.Second,
					TLSHandshakeTimeout:   10 * time.Second,
					ExpectContinueTimeout: 1 * time.Second,
				},
			},
			Embedder: nil,
		}
		return qh, nil
	case ProviderMilvus:
		if p.MilvusConfig == nil {
			return nil, errors.New("MilvusConfig is required for Milvus provider")
		}
		mh := &MilvusHandler{
			Address:  p.MilvusConfig.Address,
			Username: p.MilvusConfig.Username,
			Password: p.MilvusConfig.Password,
			Token:    p.MilvusConfig.Token,
			DBName:   p.MilvusConfig.DBName,
			Embedder: nil,
			cli:      nil,
		}
		return mh, nil
	case ProviderRAGFlow:
		if p.RAGFlowConfig == nil {
			return nil, errors.New("RAGFlowConfig is required for RAGFlow provider")
		}
		timeout := p.RAGFlowConfig.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		rh := &RAGFlowHandler{
			BaseURL:    p.RAGFlowConfig.BaseURL,
			APIKey:     p.RAGFlowConfig.APIKey,
			HTTPClient: &http.Client{Timeout: timeout},
			Embedder:   nil,
		}
		return rh, nil
	case ProviderAliyun:
		if p.AliyunConfig == nil {
			return nil, errors.New("AliyunConfig is required for Aliyun provider")
		}

		// Create Alibaba Cloud client using official SDK
		openapiConfig := &openapi.Config{
			AccessKeyId:     tea.String(p.AliyunConfig.AccessKeyID),
			AccessKeySecret: tea.String(p.AliyunConfig.AccessKeySecret),
			Endpoint:        tea.String(p.AliyunConfig.Endpoint),
		}

		client, err := bailian.NewClient(openapiConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create alibaba bailian client: %w", err)
		}

		ah := &AliyunHandler{
			client:      client,
			WorkspaceID: p.AliyunConfig.WorkspaceID,
			CategoryID:  p.AliyunConfig.CategoryID,
			Embedder:    nil,
		}
		return ah, nil
	case ProviderPostgres:
		if p.PostgresConfig == nil || strings.TrimSpace(p.PostgresConfig.DSN) == "" {
			return nil, errors.New("PostgresConfig.DSN is required for Postgres provider")
		}
		return &PostgresHandler{DSN: strings.TrimSpace(p.PostgresConfig.DSN)}, nil
	case ProviderElasticsearch:
		if p.ElasticsearchConfig == nil {
			return nil, errors.New("ElasticsearchConfig is required for Elasticsearch provider")
		}
		timeout := p.ElasticsearchConfig.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		return &ElasticsearchHandler{
			BaseURL:     p.ElasticsearchConfig.BaseURL,
			APIKey:      p.ElasticsearchConfig.APIKey,
			IndexPrefix: p.ElasticsearchConfig.IndexPrefix,
			HTTPClient:  &http.Client{Timeout: timeout},
		}, nil
	case ProviderWeaviate:
		if p.WeaviateConfig == nil {
			return nil, errors.New("WeaviateConfig is required for Weaviate provider")
		}
		timeout := p.WeaviateConfig.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		return &WeaviateHandler{
			BaseURL:    p.WeaviateConfig.BaseURL,
			APIKey:     p.WeaviateConfig.APIKey,
			HTTPClient: &http.Client{Timeout: timeout},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported knowledge provider %q (use %s, %s, %s, %s, %s, %s, or %s)",
			p.Provider, ProviderQdrant, ProviderMilvus, ProviderPostgres, ProviderElasticsearch,
			ProviderWeaviate, ProviderRAGFlow, ProviderAliyun)
	}
}

// KnowledgeHandler abstract knowledge interface
type KnowledgeHandler interface {
	Provider() string

	// Upsert write and update files
	Upsert(ctx context.Context, records []Record, opts *UpsertOptions) error

	// Query Query for txt
	Query(ctx context.Context, text string, opts *QueryOptions) ([]QueryResult, error)

	// Get get by id
	Get(ctx context.Context, ids []string, opts *GetOptions) ([]Record, error)

	// List list query for page
	List(ctx context.Context, opts *ListOptions) (*ListResult, error)

	// Delete delete file document
	Delete(ctx context.Context, ids []string, opts *DeleteOptions) error

	// Ping health check
	Ping(ctx context.Context) error

	// CreateNamespace create new namespace
	CreateNamespace(ctx context.Context, name string) error

	// DeleteNamespace delete namespack
	DeleteNamespace(ctx context.Context, name string) error

	// ListNamespaces List database namespace
	ListNamespaces(ctx context.Context) ([]string, error)
}
