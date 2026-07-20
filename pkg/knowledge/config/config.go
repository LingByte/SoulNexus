package config

import (
	"github.com/LingByte/SoulNexus/pkg/utils"
)

// Config holds global knowledge-base settings loaded from environment variables.
type Config struct {
	// vector module
	VectorProvider           string `json:"vectorProvider"`
	QdrantBaseURL            string `json:"qdrantBaseURL"`
	QdrantAPIKey             string `json:"qdrantAPIKey"`
	MilvusAddress            string `json:"milvusAddress"`
	MilvusUsername           string `json:"milvusUsername"`
	MilvusPassword           string `json:"milvusPassword"`
	MilvusToken              string `json:"milvusToken"`
	MilvusDBName             string `json:"milvusDBName"`
	PostgresDSN              string `json:"postgresDSN"`
	ElasticsearchURL         string `json:"elasticsearchURL"`
	ElasticsearchAPIKey      string `json:"elasticsearchAPIKey"`
	ElasticsearchIndexPrefix string `json:"elasticsearchIndexPrefix"`
	WeaviateURL              string `json:"weaviateURL"`
	WeaviateAPIKey           string `json:"weaviateAPIKey"`

	// embed module
	EmbedType    string `json:"embedType"`
	EmbedAPIKey  string `json:"embedAPIKey"`
	EmbedBaseURL string `json:"embedBaseURL"`
	EmbedModel   string `json:"embedModel"`
	EmbedDim     int    `json:"embedDim"`
	EmbedTimeout int    `json:"embedTimeout"`

	// chunk module
	ChunkMaxChars      int     `json:"chunkMaxChars"`
	ChunkMinChars      int     `json:"chunkMinChars"`
	ChunkOverlapChars  int     `json:"chunkOverlapChars"`
	ChunkLLMEnabled    bool    `json:"chunkLLMEnabled"`
	ChunkLLMProvider   string  `json:"chunkLLMProvider"`
	ChunkLLMModel      string  `json:"chunkLLMModel"`
	ChunkLLMBaseURL    string  `json:"chunkLLMBaseURL"`
	ChunkLLMAPIKey     string  `json:"chunkLLMAPIKey"`
	SearchIndexDir     string  `json:"searchIndexDir"`
	RetrieveStrategy   string  `json:"retrieveStrategy"`
	HybridVectorWeight float64 `json:"hybridVectorWeight"`
	SearchQueryTimeout int     `json:"searchQueryTimeout"`
	// Voice path defaults: vector-only, skip remote rerank (embed+rerank usually dominate latency).
	VoiceRetrieveStrategy string `json:"voiceRetrieveStrategy"`
	VoiceRerankEnabled    bool   `json:"voiceRerankEnabled"`

	// retrieval module
	RetrievalEmbeddingTopK         int     `json:"retrievalEmbeddingTopK"`
	RetrievalVectorThreshold       float64 `json:"retrievalVectorThreshold"`
	RetrievalKeywordThreshold      float64 `json:"retrievalKeywordThreshold"`
	RetrievalRerankTopK            int     `json:"retrievalRerankTopK"`
	RetrievalRerankThreshold       float64 `json:"retrievalRerankThreshold"`
	RetrievalRRFK                  int     `json:"retrievalRRFK"`
	RetrievalRRFVectorWeight       float64 `json:"retrievalRRFVectorWeight"`
	RetrievalRRFKeywordWeight      float64 `json:"retrievalRRFKeywordWeight"`
	RetrievalOverRetrieveMult      int     `json:"retrievalOverRetrieveMult"`
	RetrievalOverRetrieveMin       int     `json:"retrievalOverRetrieveMin"`
	RetrievalOverRetrieveMax       int     `json:"retrievalOverRetrieveMax"`
	RetrievalMMRLambda             float64 `json:"retrievalMMRLambda"`
	RetrievalEnableMMR             bool    `json:"retrievalEnableMMR"`
	RetrievalEnableDedup           bool    `json:"retrievalEnableDedup"`
	RetrievalOverlapThreshold      float64 `json:"retrievalOverlapThreshold"`
	RetrievalEnableCompositeRerank bool    `json:"retrievalEnableCompositeRerank"`
	RetrievalRerankModelWeight     float64 `json:"retrievalRerankModelWeight"`
	RetrievalRerankBaseWeight      float64 `json:"retrievalRerankBaseWeight"`

	// rerank module
	RerankEnabled  bool   `json:"rerankEnabled"`
	RerankProvider string `json:"rerankProvider"`
	RerankModel    string `json:"rerankModel"`
	RerankBaseURL  string `json:"rerankBaseURL"`
	RerankAPIKey   string `json:"rerankAPIKey"`
	RerankTimeout  int    `json:"rerankTimeout"`

	// langfuse module
	LangfuseEnabled   bool   `json:"langfuseEnabled"`
	LangfuseHost      string `json:"langfuseHost"`
	LangfusePublicKey string `json:"langfusePublicKey"`
	LangfuseSecretKey string `json:"langfuseSecretKey"`

	// worker module
	IngestWorkers        int  `json:"ingestWorkers"`
	IndexPreviewRequired bool `json:"indexPreviewRequired"`
	ParentChildEnabled   bool `json:"parentChildEnabled"`
	SummaryIndexEnabled  bool `json:"summaryIndexEnabled"`
	ParentChunkMaxChars  int  `json:"parentChunkMaxChars"`
	ChildChunkMaxChars   int  `json:"childChunkMaxChars"`
}

// LoadConfig reads knowledge-base settings from the process environment.
func LoadConfig() *Config {
	embedDim := 0
	if n, ok := utils.PositiveIntEnv("EMBED_DIMENSION"); ok {
		embedDim = n
	}

	chunkMax := 1200
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_CHUNK_MAX_CHARS"); ok {
		chunkMax = n
	}
	chunkMin := 80
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_CHUNK_MIN_CHARS"); ok {
		chunkMin = n
	}
	chunkOverlap := int(float64(chunkMax) * 0.12)
	if n, ok := utils.EnvInt("KNOWLEDGE_CHUNK_OVERLAP_CHARS"); ok && n >= 0 {
		chunkOverlap = n
	}

	embedType := utils.GetStringOrDefault("EMBED_TYPE", "openai")
	embedModel := utils.GetStringOrDefault("EMBED_MODEL", "text-embedding-v3")
	embedTimeout := 30
	if n, ok := utils.PositiveIntEnv("EMBED_TIMEOUT_SECONDS"); ok {
		embedTimeout = n
	}

	vectorProvider := utils.GetStringOrDefault("KNOWLEDGE_VECTOR_PROVIDER", "qdrant")

	ingestWorkers := 4
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_INGEST_WORKERS"); ok {
		ingestWorkers = n
	}

	chunkLLMEnabled := utils.GetBoolEnv("KNOWLEDGE_CHUNK_LLM_ENABLED")
	chunkLLMProvider := utils.GetStringOrDefault("KNOWLEDGE_CHUNK_LLM_PROVIDER", "openai")
	chunkLLMModel := utils.GetEnv("KNOWLEDGE_CHUNK_LLM_MODEL")
	chunkLLMBaseURL := utils.GetEnv("KNOWLEDGE_CHUNK_LLM_BASEURL")
	chunkLLMAPIKey := utils.GetEnv("KNOWLEDGE_CHUNK_LLM_API_KEY")
	searchIndexDir := utils.GetStringOrDefault("KNOWLEDGE_SEARCH_INDEX_DIR", "./data/knowledge-search")
	retrieveStrategy := utils.GetStringOrDefault("KNOWLEDGE_RETRIEVE_STRATEGY", "hybrid")
	voiceRetrieveStrategy := utils.GetStringOrDefault("KNOWLEDGE_VOICE_RETRIEVE_STRATEGY", "vector")
	voiceRerankEnabled := utils.GetBoolEnv("KNOWLEDGE_VOICE_RERANK_ENABLED")

	hybridWeight := 0.65
	if f, ok := utils.EnvFloat("KNOWLEDGE_HYBRID_VECTOR_WEIGHT"); ok && f > 0 && f <= 1 {
		hybridWeight = f
	}

	searchQueryTimeout := 3
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_SEARCH_QUERY_TIMEOUT_SECONDS"); ok {
		searchQueryTimeout = n
	}

	retrieval := loadRetrievalConfig(hybridWeight)
	rerankCfg := loadRerankConfig()
	langfuseCfg := loadLangfuseConfig()

	parentChunkMax := chunkMax
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_PARENT_CHUNK_MAX_CHARS"); ok {
		parentChunkMax = n
	}
	childChunkMax := 400
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_CHILD_CHUNK_MAX_CHARS"); ok {
		childChunkMax = n
	}

	return &Config{
		VectorProvider:                 vectorProvider,
		QdrantBaseURL:                  utils.GetEnv("QDRANT_BASEURL"),
		QdrantAPIKey:                   utils.GetEnv("QDRANT_API_KEY"),
		MilvusAddress:                  utils.GetEnv("MILVUS_ADDRESS"),
		MilvusUsername:                 utils.GetEnv("MILVUS_USERNAME"),
		MilvusPassword:                 utils.GetEnv("MILVUS_PASSWORD"),
		MilvusToken:                    utils.GetEnv("MILVUS_TOKEN"),
		MilvusDBName:                   utils.GetEnv("MILVUS_DB"),
		PostgresDSN:                    utils.GetEnv("POSTGRES_DSN"),
		ElasticsearchURL:               utils.GetEnv("ELASTICSEARCH_URL"),
		ElasticsearchAPIKey:            utils.GetEnv("ELASTICSEARCH_API_KEY"),
		ElasticsearchIndexPrefix:       utils.GetEnv("ELASTICSEARCH_INDEX_PREFIX"),
		WeaviateURL:                    utils.GetEnv("WEAVIATE_URL"),
		WeaviateAPIKey:                 utils.GetEnv("WEAVIATE_API_KEY"),
		EmbedType:                      embedType,
		EmbedAPIKey:                    utils.GetEnv("EMBED_API_KEY"),
		EmbedBaseURL:                   utils.GetEnv("EMBED_BASEURL"),
		EmbedModel:                     embedModel,
		EmbedDim:                       embedDim,
		EmbedTimeout:                   embedTimeout,
		ChunkMaxChars:                  chunkMax,
		ChunkMinChars:                  chunkMin,
		ChunkOverlapChars:              chunkOverlap,
		ChunkLLMEnabled:                chunkLLMEnabled,
		ChunkLLMProvider:               chunkLLMProvider,
		ChunkLLMModel:                  chunkLLMModel,
		ChunkLLMBaseURL:                chunkLLMBaseURL,
		ChunkLLMAPIKey:                 chunkLLMAPIKey,
		SearchIndexDir:                 searchIndexDir,
		RetrieveStrategy:               retrieveStrategy,
		HybridVectorWeight:             hybridWeight,
		SearchQueryTimeout:             searchQueryTimeout,
		VoiceRetrieveStrategy:          voiceRetrieveStrategy,
		VoiceRerankEnabled:             voiceRerankEnabled,
		RetrievalEmbeddingTopK:         retrieval.embeddingTopK,
		RetrievalVectorThreshold:       retrieval.vectorThreshold,
		RetrievalKeywordThreshold:      retrieval.keywordThreshold,
		RetrievalRerankTopK:            retrieval.rerankTopK,
		RetrievalRerankThreshold:       retrieval.rerankThreshold,
		RetrievalRRFK:                  retrieval.rrfK,
		RetrievalRRFVectorWeight:       retrieval.rrfVectorWeight,
		RetrievalRRFKeywordWeight:      retrieval.rrfKeywordWeight,
		RetrievalOverRetrieveMult:      retrieval.overRetrieveMult,
		RetrievalOverRetrieveMin:       retrieval.overRetrieveMin,
		RetrievalOverRetrieveMax:       retrieval.overRetrieveMax,
		RetrievalMMRLambda:             retrieval.mmrLambda,
		RetrievalEnableMMR:             retrieval.enableMMR,
		RetrievalEnableDedup:           retrieval.enableDedup,
		RetrievalOverlapThreshold:      retrieval.overlapThreshold,
		RetrievalEnableCompositeRerank: retrieval.enableCompositeRerank,
		RetrievalRerankModelWeight:     retrieval.rerankModelWeight,
		RetrievalRerankBaseWeight:      retrieval.rerankBaseWeight,
		RerankEnabled:                  rerankCfg.enabled,
		RerankProvider:                 rerankCfg.provider,
		RerankModel:                    rerankCfg.model,
		RerankBaseURL:                  rerankCfg.baseURL,
		RerankAPIKey:                   rerankCfg.apiKey,
		RerankTimeout:                  rerankCfg.timeout,
		LangfuseEnabled:                langfuseCfg.enabled,
		LangfuseHost:                   langfuseCfg.host,
		LangfusePublicKey:              langfuseCfg.publicKey,
		LangfuseSecretKey:              langfuseCfg.secretKey,
		IngestWorkers:                  ingestWorkers,
		IndexPreviewRequired:           !utils.GetBoolEnv("KNOWLEDGE_INDEX_PREVIEW_SKIP"),
		ParentChildEnabled:             !utils.GetBoolEnv("KNOWLEDGE_PARENT_CHILD_DISABLED"),
		SummaryIndexEnabled:            !utils.GetBoolEnv("KNOWLEDGE_SUMMARY_INDEX_DISABLED"),
		ParentChunkMaxChars:            parentChunkMax,
		ChildChunkMaxChars:             childChunkMax,
	}
}

type retrievalEnv struct {
	embeddingTopK, rrfK, rerankTopK, overRetrieveMult, overRetrieveMin, overRetrieveMax int
	vectorThreshold, keywordThreshold, rerankThreshold, overlapThreshold                float64
	rrfVectorWeight, rrfKeywordWeight, mmrLambda                                        float64
	enableMMR, enableDedup                                                              bool
	enableCompositeRerank                                                               bool
	rerankModelWeight, rerankBaseWeight                                                 float64
}

func loadRetrievalConfig(hybridWeight float64) retrievalEnv {
	def := retrievalEnv{
		embeddingTopK:         50,
		vectorThreshold:       0.15,
		// Bleve BM25 scores for short CJK queries are often << 0.3; keep absolute cutoff off by default.
		keywordThreshold: 0,
		rerankTopK:            10,
		rerankThreshold:       0.2,
		rrfK:                  60,
		rrfVectorWeight:       hybridWeight,
		rrfKeywordWeight:      1 - hybridWeight,
		overRetrieveMult:      5,
		overRetrieveMin:       50,
		overRetrieveMax:       500,
		mmrLambda:             0.7,
		enableMMR:             true,
		enableDedup:           true,
		overlapThreshold:      0.85,
		enableCompositeRerank: true,
		rerankModelWeight:     0.6,
		rerankBaseWeight:      0.3,
	}
	if hybridWeight <= 0 || hybridWeight > 1 {
		def.rrfVectorWeight = 0.7
		def.rrfKeywordWeight = 0.3
	}
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_EMBEDDING_TOP_K"); ok {
		def.embeddingTopK = n
	}
	if f, ok := utils.EnvFloat("KNOWLEDGE_VECTOR_THRESHOLD"); ok {
		def.vectorThreshold = f
	}
	if f, ok := utils.EnvFloat("KNOWLEDGE_KEYWORD_THRESHOLD"); ok {
		def.keywordThreshold = f
	}
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_RERANK_TOP_K"); ok {
		def.rerankTopK = n
	}
	if f, ok := utils.EnvFloat("KNOWLEDGE_RERANK_THRESHOLD"); ok {
		def.rerankThreshold = f
	}
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_RRF_K"); ok {
		def.rrfK = n
	}
	if f, ok := utils.EnvFloat("KNOWLEDGE_RRF_VECTOR_WEIGHT"); ok {
		def.rrfVectorWeight = f
	}
	if f, ok := utils.EnvFloat("KNOWLEDGE_RRF_KEYWORD_WEIGHT"); ok {
		def.rrfKeywordWeight = f
	}
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_OVER_RETRIEVE_MULTIPLIER"); ok {
		def.overRetrieveMult = n
	}
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_OVER_RETRIEVE_MIN"); ok {
		def.overRetrieveMin = n
	}
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_OVER_RETRIEVE_MAX"); ok {
		def.overRetrieveMax = n
	}
	if f, ok := utils.EnvFloat("KNOWLEDGE_MMR_LAMBDA"); ok {
		def.mmrLambda = f
	}
	if f, ok := utils.EnvFloat("KNOWLEDGE_OVERLAP_THRESHOLD"); ok {
		def.overlapThreshold = f
	}
	def.enableMMR = !utils.GetBoolEnv("KNOWLEDGE_MMR_DISABLED")
	def.enableDedup = !utils.GetBoolEnv("KNOWLEDGE_DEDUP_DISABLED")
	def.enableCompositeRerank = !utils.GetBoolEnv("KNOWLEDGE_RERANK_COMPOSITE_DISABLED")
	if f, ok := utils.EnvFloat("KNOWLEDGE_RERANK_MODEL_WEIGHT"); ok {
		def.rerankModelWeight = f
	}
	if f, ok := utils.EnvFloat("KNOWLEDGE_RERANK_BASE_WEIGHT"); ok {
		def.rerankBaseWeight = f
	}
	return def
}

type rerankEnv struct {
	enabled  bool
	provider string
	model    string
	baseURL  string
	apiKey   string
	timeout  int
}

func loadRerankConfig() rerankEnv {
	cfg := rerankEnv{
		enabled:  utils.GetBoolEnv("KNOWLEDGE_RERANK_ENABLED"),
		provider: utils.GetEnv("KNOWLEDGE_RERANK_PROVIDER"),
		model:    utils.GetEnv("KNOWLEDGE_RERANK_MODEL"),
		baseURL:  utils.GetEnv("KNOWLEDGE_RERANK_BASEURL"),
		apiKey:   utils.GetEnv("KNOWLEDGE_RERANK_API_KEY"),
		timeout:  30,
	}
	if n, ok := utils.PositiveIntEnv("KNOWLEDGE_RERANK_TIMEOUT_SECONDS"); ok {
		cfg.timeout = n
	}
	return cfg
}

type langfuseEnv struct {
	enabled   bool
	host      string
	publicKey string
	secretKey string
}

func loadLangfuseConfig() langfuseEnv {
	enabled := utils.GetBoolEnv("KNOWLEDGE_TRACE_LANGFUSE_ENABLED")
	if !enabled {
		enabled = utils.GetBoolEnv("LANGFUSE_ENABLED")
	}
	return langfuseEnv{
		enabled:   enabled,
		host:      utils.GetEnv("LANGFUSE_HOST"),
		publicKey: utils.GetEnv("LANGFUSE_PUBLIC_KEY"),
		secretKey: utils.GetEnv("LANGFUSE_SECRET_KEY"),
	}
}
