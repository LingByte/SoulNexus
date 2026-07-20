package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/errors"
	knconfig "github.com/LingByte/SoulNexus/pkg/knowledge/config"
	knsearch "github.com/LingByte/SoulNexus/pkg/knowledge/search"
	llmchunk "github.com/LingByte/lingllm/chunk"
	llmembed "github.com/LingByte/lingllm/embedder"
	llmknowledge "github.com/LingByte/lingllm/knowledge"
	llmprotocol "github.com/LingByte/lingllm/protocol"
	llmrerank "github.com/LingByte/lingllm/rerank"
	llmretrieve "github.com/LingByte/lingllm/retrieve"
	"github.com/LingByte/lingllm/retrieve/langfuse"
	"github.com/google/uuid"
)

// serviceDeps holds runtime dependencies (always non-nil after NewService).
type serviceDeps struct {
	embedder      llmembed.Embedder
	handler       llmknowledge.KnowledgeHandler
	chunker       llmchunk.Chunker
	searchIndexes *knsearch.SearchIndexManager
	reranker      llmrerank.Reranker
	observer      llmretrieve.Observer
}

// Service wires LingLLM embedder, chunker, and vector handler for tenant knowledge bases.
type Service struct {
	cfg  *knconfig.Config
	deps *serviceDeps
}

// Config returns the service configuration snapshot.
func (s *Service) Config() *knconfig.Config {
	if s == nil || s.deps == nil {
		return nil
	}
	return s.cfg
}

// NewService builds a knowledge-base service from VoiceServer env configuration.
func NewService(ctx context.Context, cfg *knconfig.Config) (*Service, error) {
	if err := validateVectorProviderConfig(cfg); err != nil {
		return nil, err
	}
	if err := validateEmbedConfig(cfg); err != nil {
		return nil, err
	}
	emb, err := llmembed.Create(ctx, &llmembed.Config{
		Provider:  cfg.EmbedType,
		Model:     cfg.EmbedModel,
		BaseURL:   cfg.EmbedBaseURL,
		APIKey:    cfg.EmbedAPIKey,
		Dimension: cfg.EmbedDim,
		Timeout:   cfg.EmbedTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("create embedder: %w", err)
	}
	// Cache query embeddings — remote embed APIs often dominate voice recall latency (1–3s+).
	emb = llmembed.NewCachingEmbedder(emb, 2048, 30*time.Minute)

	handler, err := newVectorHandler(cfg, emb)
	if err != nil {
		_ = emb.Close()
		return nil, err
	}

	chunkCfg := &llmchunk.Config{
		Provider:     "router",
		MaxChars:     cfg.ChunkMaxChars,
		MinChars:     cfg.ChunkMinChars,
		OverlapChars: cfg.ChunkOverlapChars,
		Detector:     &llmchunk.RuleBasedDocumentTypeDetector{},
	}
	if cfg.ChunkLLMEnabled {
		chatModel, err := newChunkChatModel(cfg)
		if err != nil {
			_ = emb.Close()
			return nil, err
		}
		chunkCfg.ChatModel = chatModel
		chunkCfg.Model = cfg.ChunkLLMModel
	}

	chunker, err := llmchunk.Create(ctx, chunkCfg)
	if err != nil {
		_ = emb.Close()
		return nil, fmt.Errorf("create chunker: %w", err)
	}

	searchIndexes, err := knsearch.NewSearchIndexManager(cfg.SearchIndexDir, cfg.SearchQueryTimeout)
	if err != nil {
		_ = emb.Close()
		return nil, fmt.Errorf("create search index manager: %w", err)
	}

	var reranker llmrerank.Reranker
	if cfg.RerankEnabled {
		r, err := llmrerank.Create(&llmrerank.RerankConfig{
			Provider: cfg.RerankProvider,
			Model:    cfg.RerankModel,
			BaseURL:  cfg.RerankBaseURL,
			APIKey:   cfg.RerankAPIKey,
			Timeout:  cfg.RerankTimeout,
		})
		if err != nil {
			_ = emb.Close()
			return nil, fmt.Errorf("create reranker: %w", err)
		}
		reranker = r
	}

	return &Service{
		cfg: cfg,
		deps: &serviceDeps{
			embedder:      emb,
			handler:       handler,
			chunker:       chunker,
			searchIndexes: searchIndexes,
			reranker:      reranker,
			observer:      newRetrievalObserver(cfg),
		},
	}, nil
}

func validateEmbedConfig(cfg *knconfig.Config) error {
	if cfg == nil {
		return errors.ErrEmptyEmbedApiKey
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.EmbedType))
	// Local / Ollama do not require a cloud API key.
	switch provider {
	case "ollama", "local":
		if strings.TrimSpace(cfg.EmbedModel) == "" {
			return fmt.Errorf("EMBED_MODEL is required for %s", provider)
		}
		return nil
	}
	if strings.TrimSpace(cfg.EmbedAPIKey) == "" {
		return errors.ErrEmptyEmbedApiKey
	}
	return nil
}

func validateVectorProviderConfig(cfg *knconfig.Config) error {
	switch strings.ToLower(strings.TrimSpace(cfg.VectorProvider)) {
	case llmknowledge.ProviderQdrant, "":
		if strings.TrimSpace(cfg.QdrantBaseURL) == "" {
			return errors.ErrEmptyQrantBaseUrl
		}
	case llmknowledge.ProviderMilvus:
		if strings.TrimSpace(cfg.MilvusAddress) == "" {
			return errors.ErrEmptyMilvusAddress
		}
	case llmknowledge.ProviderPostgres:
		if strings.TrimSpace(cfg.PostgresDSN) == "" {
			return errors.ErrEmptyPostgresDsn
		}
	case llmknowledge.ProviderElasticsearch:
		if strings.TrimSpace(cfg.ElasticsearchURL) == "" {
			return errors.ErrEmptyElasticSearchUrl
		}
	case llmknowledge.ProviderWeaviate:
		if strings.TrimSpace(cfg.WeaviateURL) == "" {
			return errors.ErrEmptyWeaviateUrl
		}
	}
	return nil
}

func newVectorHandler(cfg *knconfig.Config, emb llmembed.Embedder) (llmknowledge.KnowledgeHandler, error) {
	provider := strings.TrimSpace(cfg.VectorProvider)
	if provider == "" {
		provider = llmknowledge.ProviderQdrant
	}
	timeout := time.Duration(cfg.EmbedTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	handler, err := llmknowledge.NewKnowledgeHandler(llmknowledge.HandlerFactoryParams{
		Provider: provider,
		QdrantConfig: &llmknowledge.QdrantConfig{
			BaseURL: cfg.QdrantBaseURL,
			APIKey:  cfg.QdrantAPIKey,
			Timeout: timeout,
		},
		MilvusConfig: &llmknowledge.MilvusConfig{
			Address:  cfg.MilvusAddress,
			Username: cfg.MilvusUsername,
			Password: cfg.MilvusPassword,
			Token:    cfg.MilvusToken,
			DBName:   cfg.MilvusDBName,
		},
		PostgresConfig: &llmknowledge.PostgresConfig{
			DSN: cfg.PostgresDSN,
		},
		ElasticsearchConfig: &llmknowledge.ElasticsearchConfig{
			BaseURL:     cfg.ElasticsearchURL,
			APIKey:      cfg.ElasticsearchAPIKey,
			IndexPrefix: cfg.ElasticsearchIndexPrefix,
			Timeout:     timeout,
		},
		WeaviateConfig: &llmknowledge.WeaviateConfig{
			BaseURL: cfg.WeaviateURL,
			APIKey:  cfg.WeaviateAPIKey,
			Timeout: timeout,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create vector handler: %w", err)
	}

	attachEmbedder(handler, emb)
	return handler, nil
}

func attachEmbedder(handler llmknowledge.KnowledgeHandler, emb llmembed.Embedder) {
	switch h := handler.(type) {
	case *llmknowledge.QdrantHandler:
		h.Embedder = emb
	case *llmknowledge.MilvusHandler:
		h.Embedder = emb
	case *llmknowledge.PostgresHandler:
		h.Embedder = emb
	case *llmknowledge.ElasticsearchHandler:
		h.Embedder = emb
	case *llmknowledge.WeaviateHandler:
		h.Embedder = emb
	case *llmknowledge.RAGFlowHandler:
		h.Embedder = emb
	case *llmknowledge.AliyunHandler:
		h.Embedder = emb
	}
}

// VectorProvider returns the deployment-wide vector backend (KNOWLEDGE_VECTOR_PROVIDER).
func (s *Service) VectorProvider() string {
	if s == nil || s.cfg == nil {
		return llmknowledge.ProviderQdrant
	}
	p := strings.ToLower(strings.TrimSpace(s.cfg.VectorProvider))
	if p == "" {
		return llmknowledge.ProviderQdrant
	}
	return p
}

// EmbedModel returns the deployment-wide embedding model (EMBED_MODEL).
func (s *Service) EmbedModel() string {
	if s == nil || s.cfg == nil {
		return "text-embedding-v3"
	}
	if m := strings.TrimSpace(s.cfg.EmbedModel); m != "" {
		return m
	}
	return "text-embedding-v3"
}

// EmbedDim returns the deployment-wide embedding dimension (EMBED_DIMENSION).
func (s *Service) EmbedDim() int {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.EmbedDim
}

// Close releases embedder resources.
func (s *Service) Close() error {
	if s == nil || s.deps == nil {
		return nil
	}
	if s.deps.searchIndexes != nil {
		_ = s.deps.searchIndexes.Close()
	}
	if s.deps.embedder == nil {
		return nil
	}
	return s.deps.embedder.Close()
}

// Ping checks connectivity to the vector backend (Qdrant health /collections).
func (s *Service) Ping(ctx context.Context) error {
	if s == nil || s.deps == nil || s.deps.handler == nil {
		return fmt.Errorf("knowledge base service is not initialized")
	}
	return s.deps.handler.Ping(ctx)
}

// EnsureNamespace creates the Qdrant collection for a knowledge namespace.
func (s *Service) EnsureNamespace(ctx context.Context, collection string, vectorDim int) error {
	if s == nil || s.deps == nil || s.deps.handler == nil {
		return fmt.Errorf("knowledge base service is not initialized")
	}
	_ = vectorDim
	if err := s.deps.handler.CreateNamespace(ctx, collection); err != nil {
		return err
	}
	// Open/create the local keyword index so ingest/upsert and hybrid recall share the same path.
	if s.deps.searchIndexes != nil {
		if _, err := s.deps.searchIndexes.ForNamespace(collection); err != nil {
			return fmt.Errorf("ensure search index: %w", err)
		}
	}
	return nil
}

// DeleteNamespace removes the Qdrant collection backing a namespace.
func (s *Service) DeleteNamespace(ctx context.Context, collection string) error {
	if s == nil || s.deps == nil || s.deps.handler == nil {
		return fmt.Errorf("knowledge base service is not initialized")
	}
	if s.deps.searchIndexes != nil {
		if err := s.deps.searchIndexes.DeleteNamespace(collection); err != nil {
			return err
		}
	}
	return s.deps.handler.DeleteNamespace(ctx, collection)
}

// Qdrant accepts point IDs as uint64 or UUID strings only (not arbitrary strings like "doc#0").
var knowledgeChunkPointNS = uuid.MustParse("7c9e6679-7425-40de-944b-e07fc1f90ae7")

func chunkPointID(docID string, chunkIndex int) string {
	name := fmt.Sprintf("%s:%d", strings.TrimSpace(docID), chunkIndex)
	return uuid.NewSHA1(knowledgeChunkPointNS, []byte(name)).String()
}

// IngestResult holds vector-ingestion output for one document.
type IngestResult struct {
	ChunkCount    int
	ChunkStrategy string
	Chunks        []ChunkSummary
	RecordIDs     []string
	SummaryText   string
}

// DeleteVectors removes vector points by ID from a namespace collection.
func (s *Service) DeleteVectors(ctx context.Context, collection string, recordIDs []string) error {
	if s == nil || s.deps == nil || s.deps.handler == nil || len(recordIDs) == 0 {
		return nil
	}
	if err := s.deps.handler.Delete(ctx, recordIDs, &llmknowledge.DeleteOptions{Namespace: collection}); err != nil {
		return err
	}
	return s.deleteSearchChunks(ctx, collection, recordIDs)
}

// RetrieveStrategy returns the configured recall strategy label.
func (s *Service) RetrieveStrategy() string {
	if s == nil || s.cfg == nil {
		return "vector"
	}
	return strings.ToLower(strings.TrimSpace(s.cfg.RetrieveStrategy))
}

// RecallOptions configures vector/keyword/hybrid recall.
type RecallOptions struct {
	TopK     int
	MinScore float64
	Filter   RecallFilter
	// Strategy overrides the global KNOWLEDGE_RETRIEVE_STRATEGY when non-empty (vector|keyword|hybrid).
	Strategy string
	// SkipRerank disables remote rerank for this call (voice latency path).
	SkipRerank bool
	// Lean disables over-retrieve expansion (fetch exactly TopK).
	Lean bool
}

// Recall searches the namespace using vector, keyword, or hybrid retrieval.
func (s *Service) Recall(ctx context.Context, collection, query string, topK int, minScore float64) ([]llmknowledge.QueryResult, error) {
	return s.RecallWithOptions(ctx, collection, query, RecallOptions{TopK: topK, MinScore: minScore})
}

// RecallWithOptions searches with metadata filters and parent-context expansion.
func (s *Service) RecallWithOptions(ctx context.Context, collection, query string, opts RecallOptions) ([]llmknowledge.QueryResult, error) {
	if s == nil || s.deps == nil || s.deps.handler == nil {
		return nil, fmt.Errorf("knowledge base service is not initialized")
	}
	topK := opts.TopK
	if topK <= 0 {
		topK = 5
	}
	minScore := opts.MinScore
	filters := opts.Filter.ToQueryFilters()
	strategy := strings.ToLower(strings.TrimSpace(opts.Strategy))
	if strategy == "" {
		strategy = s.RetrieveStrategy()
	}

	var hits []llmknowledge.QueryResult
	var err error
	switch strategy {
	case "keyword":
		hits, err = s.keywordRecallFiltered(ctx, collection, query, topK, minScore, filters, opts.SkipRerank, opts.Lean)
	case "hybrid":
		hits, err = s.hybridRecallFiltered(ctx, collection, query, topK, minScore, filters, opts.SkipRerank, opts.Lean)
	default:
		hits, err = s.vectorRecallFiltered(ctx, collection, query, topK, minScore, filters, opts.SkipRerank, opts.Lean)
	}
	if err != nil {
		return nil, err
	}
	return expandRecallParentContext(hits), nil
}

// EncodeRecordIDs serializes vector point IDs for DB storage.
func EncodeRecordIDs(ids []string) string {
	if len(ids) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ids)
	return string(b)
}

// DecodeRecordIDs parses stored vector point IDs.
func DecodeRecordIDs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil
	}
	return ids
}

func newRetrievalObserver(cfg *knconfig.Config) llmretrieve.Observer {
	if !cfg.LangfuseEnabled {
		return nil
	}
	client, err := langfuse.NewClient(langfuse.Config{
		Enabled:   true,
		BaseURL:   cfg.LangfuseHost,
		PublicKey: cfg.LangfusePublicKey,
		SecretKey: cfg.LangfuseSecretKey,
	})
	if err != nil || client == nil {
		return nil
	}
	return langfuse.NewObserver(client, "knowledge-retrieval")
}

func newChunkChatModel(cfg *knconfig.Config) (llmprotocol.ChatModel, error) {
	if !cfg.ChunkLLMEnabled {
		return nil, nil
	}
	apiKey := strings.TrimSpace(cfg.ChunkLLMAPIKey)
	if apiKey == "" {
		return nil, fmt.Errorf("KNOWLEDGE_CHUNK_LLM_API_KEY is required when KNOWLEDGE_CHUNK_LLM_ENABLED=true")
	}
	provider := strings.TrimSpace(cfg.ChunkLLMProvider)
	if provider == "" {
		provider = string(llmprotocol.ProviderOpenAI)
	}
	if strings.TrimSpace(cfg.ChunkLLMModel) == "" {
		return nil, fmt.Errorf("KNOWLEDGE_CHUNK_LLM_MODEL is required when KNOWLEDGE_CHUNK_LLM_ENABLED=true")
	}
	baseURL := strings.TrimSpace(cfg.ChunkLLMBaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("KNOWLEDGE_CHUNK_LLM_BASEURL is required when KNOWLEDGE_CHUNK_LLM_ENABLED=true")
	}

	client, err := llmprotocol.NewClient(llmprotocol.ClientConfig{
		Provider: llmprotocol.ProviderType(provider),
		APIKey:   apiKey,
		BaseURL:  baseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("create chunk LLM client: %w", err)
	}
	return client, nil
}

// EmbedTexts returns embedding vectors for non-empty texts.
func (s *Service) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if s == nil || s.deps == nil || s.deps.embedder == nil {
		return nil, fmt.Errorf("embedder not available")
	}
	clean := make([]string, 0, len(texts))
	for _, t := range texts {
		if v := strings.TrimSpace(t); v != "" {
			clean = append(clean, v)
		}
	}
	if len(clean) == 0 {
		return nil, fmt.Errorf("no texts to embed")
	}
	return s.deps.embedder.Embed(ctx, clean)
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
