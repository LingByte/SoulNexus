package config

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/pkg/cache"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingstorage-sdk-go"
)

// Config main configuration structure
type Config struct {
	MachineID    int64              `env:"MACHINE_ID"`
	Server       ServerConfig       `mapstructure:"server"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Log          logger.LogConfig   `mapstructure:"log"`
	Cache        cache.Config       `mapstructure:"cache"`
	Auth         AuthConfig         `mapstructure:"auth"`
	Services     ServicesConfig     `mapstructure:"services"`
	Integrations IntegrationsConfig `mapstructure:"integrations"`
	Features     FeaturesConfig     `mapstructure:"features"`
	Middleware   MiddlewareConfig   `mapstructure:"middleware"`
}

// ServerConfig server configuration
type ServerConfig struct {
	Name          string `env:"SERVER_NAME"`
	Desc          string `env:"SERVER_DESC"`
	URL           string `env:"SERVER_URL"`
	Logo          string `env:"SERVER_LOGO"`
	TermsURL      string `env:"SERVER_TERMS_URL"`
	Addr          string `env:"ADDR"`
	Mode          string `env:"MODE"`
	DocsPrefix    string `env:"DOCS_PREFIX"`
	APIPrefix     string `env:"API_PREFIX"`
	AdminPrefix   string `env:"ADMIN_PREFIX"`
	AuthPrefix    string `env:"AUTH_PREFIX"`
	MonitorPrefix string `env:"MONITOR_PREFIX"`
	SSLEnabled    bool   `env:"SSL_ENABLED"`
	SSLCertFile   string `env:"SSL_CERT_FILE"`
	SSLKeyFile    string `env:"SSL_KEY_FILE"`
}

// DatabaseConfig database configuration
type DatabaseConfig struct {
	Driver string `env:"DB_DRIVER"`
	DSN    string `env:"DSN"`
}

// AuthConfig authentication configuration
type AuthConfig struct {
	Header           string `env:"AUTH_HEADER"`
	SessionSecret    string `env:"SESSION_SECRET"`
	SecretExpireDays string `env:"SESSION_EXPIRE_DAYS"`
	APISecretKey     string `env:"API_SECRET_KEY"`
}

// ServicesConfig services configuration
type ServicesConfig struct {
	LLM           LLMConfig               `mapstructure:"llm"`
	Mail          notification.MailConfig `mapstructure:"mail"`
	KnowledgeBase KnowledgeBaseConfig     `mapstructure:"knowledge_base"`
	Voice         VoiceConfig             `mapstructure:"voice"`
	Storage       StorageConfig           `mapstructure:"storage"`
}

// LLMConfig LLM service configuration
type LLMConfig struct {
	APIKey  string `env:"LLM_API_KEY"`
	BaseURL string `env:"LLM_BASE_URL"`
	Model   string `env:"LLM_MODEL"`
}

// KnowledgeBaseConfig knowledge base configuration
type KnowledgeBaseConfig struct {
	Enabled       bool                `env:"KNOWLEDGE_BASE_ENABLED"`
	Bailian       BailianConfig       `mapstructure:"bailian"`
	Milvus        MilvusConfig        `mapstructure:"milvus"`
	Qdrant        QdrantConfig        `mapstructure:"qdrant"`
	Elasticsearch ElasticsearchConfig `mapstructure:"elasticsearch"`
	Pinecone      PineconeConfig      `mapstructure:"pinecone"`
	Neo4j         Neo4jConfig         `mapstructure:"neo4j"`
	Embedding     EmbeddingConfig     `mapstructure:"embedding"`
	Rerank        RerankConfig        `mapstructure:"rerank"`
}

// BailianConfig Bailian configuration
type BailianConfig struct {
	AccessKeyId     string `env:"BAILIAN_ACCESS_KEY_ID"`
	AccessKeySecret string `env:"BAILIAN_ACCESS_KEY_SECRET"`
	Endpoint        string `env:"BAILIAN_ENDPOINT"`
	WorkspaceId     string `env:"BAILIAN_WORKSPACE_ID"`
	CategoryId      string `env:"BAILIAN_CATEGORY_ID"`
	SourceType      string `env:"BAILIAN_SOURCE_TYPE"`
	Parser          string `env:"BAILIAN_PARSER"`
	StructType      string `env:"BAILIAN_STRUCT_TYPE"`
	SinkType        string `env:"BAILIAN_SINK_TYPE"`
}

// MilvusConfig Milvus configuration
type MilvusConfig struct {
	Address    string `env:"MILVUS_ADDRESS"`
	Username   string `env:"MILVUS_USERNAME"`
	Password   string `env:"MILVUS_PASSWORD"`
	Collection string `env:"MILVUS_COLLECTION"`
	Dimension  int    `env:"MILVUS_DIMENSION"`
}

// QdrantConfig Qdrant configuration
type QdrantConfig struct {
	BaseURL    string `env:"QDRANT_BASE_URL"`
	APIKey     string `env:"QDRANT_API_KEY"`
	Collection string `env:"QDRANT_COLLECTION"`
	Dimension  int    `env:"QDRANT_DIMENSION"`
}

// ElasticsearchConfig Elasticsearch configuration
type ElasticsearchConfig struct {
	BaseURL  string `env:"ELASTICSEARCH_BASE_URL"`
	Username string `env:"ELASTICSEARCH_USERNAME"`
	Password string `env:"ELASTICSEARCH_PASSWORD"`
	Index    string `env:"ELASTICSEARCH_INDEX"`
}

// PineconeConfig Pinecone configuration
type PineconeConfig struct {
	APIKey    string `env:"PINECONE_API_KEY"`
	BaseURL   string `env:"PINECONE_BASE_URL"`
	IndexName string `env:"PINECONE_INDEX_NAME"`
	Dimension int    `env:"PINECONE_DIMENSION"`
}

// Neo4jConfig Neo4j configuration
type Neo4jConfig struct {
	Enabled  bool   `env:"NEO4J_ENABLED"`
	URI      string `env:"NEO4J_URI"`
	Username string `env:"NEO4J_USERNAME"`
	Password string `env:"NEO4J_PASSWORD"`
	Database string `env:"NEO4J_DATABASE"`
}

// EmbeddingConfig embedding service configuration
type EmbeddingConfig struct {
	Provider  string `env:"EMBEDDING_PROVIDER"`  // Provider name (e.g., "nvidia", "openai")
	BaseURL   string `env:"EMBEDDING_BASE_URL"`  // API base URL
	APIKey    string `env:"EMBEDDING_API_KEY"`   // API key
	Model     string `env:"EMBEDDING_MODEL"`     // Model name
	Dimension int    `env:"EMBEDDING_DIMENSION"` // Embedding dimension
}

// RerankConfig rerank service configuration
type RerankConfig struct {
	Provider string `env:"RERANK_PROVIDER"` // Provider name (e.g., "siliconflow", "cohere")
	BaseURL  string `env:"RERANK_BASE_URL"` // API base URL
	APIKey   string `env:"RERANK_API_KEY"`  // API key
	Model    string `env:"RERANK_MODEL"`    // Model name
	TopN     int    `env:"RERANK_TOP_N"`    // Number of top results to return after reranking
}

// VoiceConfig voice service configuration
type VoiceConfig struct {
	Qiniu      QiniuVoiceConfig  `mapstructure:"qiniu"`
	Xunfei     XunfeiVoiceConfig `mapstructure:"xunfei"`
	Voiceprint VoiceprintConfig  `mapstructure:"voiceprint"`
}

// QiniuVoiceConfig Qiniu voice configuration
type QiniuVoiceConfig struct {
	ASRAPIKey  string `env:"QINIU_ASR_API_KEY"`
	ASRBaseURL string `env:"QINIU_ASR_BASE_URL"`
	TTSAPIKey  string `env:"QINIU_TTS_API_KEY"`
	TTSBaseURL string `env:"QINIU_TTS_BASE_URL"`
}

// XunfeiVoiceConfig Xunfei voice configuration
type XunfeiVoiceConfig struct {
	WSAppId     string `env:"XUNFEI_WS_APP_ID"`
	WSAPIKey    string `env:"XUNFEI_WS_API_KEY"`
	WSAPISecret string `env:"XUNFEI_WS_API_SECRET"`
}

// VoiceprintConfig Voiceprint service configuration
type VoiceprintConfig struct {
	Enabled             bool          `env:"VOICEPRINT_ENABLED"`
	BaseURL             string        `env:"VOICEPRINT_BASE_URL"`
	APIKey              string        `env:"VOICEPRINT_API_KEY"`
	Timeout             time.Duration `env:"VOICEPRINT_TIMEOUT"`
	ConnectTimeout      time.Duration `env:"VOICEPRINT_CONNECT_TIMEOUT"`
	MaxRetries          int           `env:"VOICEPRINT_MAX_RETRIES"`
	RetryInterval       time.Duration `env:"VOICEPRINT_RETRY_INTERVAL"`
	SimilarityThreshold float64       `env:"VOICEPRINT_SIMILARITY_THRESHOLD"`
	MaxCandidates       int           `env:"VOICEPRINT_MAX_CANDIDATES"`
	CacheEnabled        bool          `env:"VOICEPRINT_CACHE_ENABLED"`
	CacheTTL            time.Duration `env:"VOICEPRINT_CACHE_TTL"`
	LogEnabled          bool          `env:"VOICEPRINT_LOG_ENABLED"`
	LogLevel            string        `env:"VOICEPRINT_LOG_LEVEL"`
}

// StorageConfig storage configuration
type StorageConfig struct {
	BaseURL   string `env:"LINGSTORAGE_BASE_URL"`
	APIKey    string `env:"LINGSTORAGE_API_KEY"`
	APISecret string `env:"LINGSTORAGE_API_SECRET"`
	Bucket    string `env:"LINGSTORAGE_BUCKET"`
}

// IntegrationsConfig integrations configuration
type IntegrationsConfig struct {
	// Other third-party integration configurations can be added here
}

// FeaturesConfig feature flags configuration
type FeaturesConfig struct {
	SearchEnabled   bool   `env:"SEARCH_ENABLED"`
	SearchPath      string `env:"SEARCH_PATH"`
	SearchBatchSize int    `env:"SEARCH_BATCH_SIZE"`
	LanguageEnabled bool   `env:"LANGUAGE_ENABLED"`
	BackupEnabled   bool   `env:"BACKUP_ENABLED"`
	BackupPath      string `env:"BACKUP_PATH"`
	BackupSchedule  string `env:"BACKUP_SCHEDULE"`
}

// MiddlewareConfig middleware configuration
type MiddlewareConfig struct {
	// Rate limiting configuration
	RateLimit RateLimiterConfig
	// Timeout configuration
	Timeout TimeoutConfig
	// Circuit breaker configuration
	CircuitBreaker CircuitBreakerConfig
	// Whether to enable each middleware
	EnableRateLimit      bool `env:"ENABLE_RATE_LIMIT"`
	EnableTimeout        bool `env:"ENABLE_TIMEOUT"`
	EnableCircuitBreaker bool `env:"ENABLE_CIRCUIT_BREAKER"`
	EnableOperationLog   bool `env:"ENABLE_OPERATION_LOG"`
}

// RateLimiterConfig rate limiting configuration
type RateLimiterConfig struct {
	GlobalRPS    int           `env:"RATE_LIMIT_GLOBAL_RPS"`   // Global requests per second
	GlobalBurst  int           `env:"RATE_LIMIT_GLOBAL_BURST"` // Global burst requests
	GlobalWindow time.Duration // Global time window
	UserRPS      int           `env:"RATE_LIMIT_USER_RPS"`   // User requests per second
	UserBurst    int           `env:"RATE_LIMIT_USER_BURST"` // User burst requests
	UserWindow   time.Duration // User time window
	IPRPS        int           `env:"RATE_LIMIT_IP_RPS"`   // IP requests per second
	IPBurst      int           `env:"RATE_LIMIT_IP_BURST"` // IP burst requests
	IPWindow     time.Duration // IP time window
}

// TimeoutConfig timeout configuration
type TimeoutConfig struct {
	DefaultTimeout   time.Duration `env:"DEFAULT_TIMEOUT"`
	FallbackResponse interface{}
}

// CircuitBreakerConfig circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold      int           `env:"CIRCUIT_BREAKER_FAILURE_THRESHOLD"`
	SuccessThreshold      int           `env:"CIRCUIT_BREAKER_SUCCESS_THRESHOLD"`
	Timeout               time.Duration `env:"CIRCUIT_BREAKER_TIMEOUT"`
	OpenTimeout           time.Duration `env:"CIRCUIT_BREAKER_OPEN_TIMEOUT"`
	MaxConcurrentRequests int           `env:"CIRCUIT_BREAKER_MAX_CONCURRENT"`
}

var GlobalConfig *Config

var GlobalStore *lingstorage.Client

func Load() error {
	// 1. Load .env file based on environment (don't error if it doesn't exist, use default values)
	env := os.Getenv("APP_ENV")
	err := utils.LoadEnv(env)
	if err != nil {
		// Only log when .env file doesn't exist, don't affect startup
		log.Printf("Note: .env file not found or failed to load: %v (using default values)", err)
	}

	// 2. Load global configuration
	GlobalConfig = &Config{
		MachineID: utils.GetIntEnv("MACHINE_ID"),
		Server: ServerConfig{
			Name:          getStringOrDefault("SERVER_NAME", ""),
			Desc:          getStringOrDefault("SERVER_DESC", ""),
			URL:           getStringOrDefault("SERVER_URL", ""),
			Logo:          getStringOrDefault("SERVER_LOGO", ""),
			TermsURL:      getStringOrDefault("SERVER_TERMS_URL", ""),
			Addr:          getStringOrDefault("ADDR", ":7072"),
			Mode:          getStringOrDefault("MODE", "development"),
			DocsPrefix:    getStringOrDefault("DOCS_PREFIX", "/api/docs"),
			APIPrefix:     getStringOrDefault("API_PREFIX", "/api"),
			AdminPrefix:   getStringOrDefault("ADMIN_PREFIX", "/admin"),
			AuthPrefix:    getStringOrDefault("AUTH_PREFIX", "/auth"),
			MonitorPrefix: getStringOrDefault("MONITOR_PREFIX", "/metrics"),
			SSLEnabled:    getBoolOrDefault("SSL_ENABLED", false),
			SSLCertFile:   getStringOrDefault("SSL_CERT_FILE", ""),
			SSLKeyFile:    getStringOrDefault("SSL_KEY_FILE", ""),
		},
		Database: DatabaseConfig{
			Driver: getStringOrDefault("DB_DRIVER", "sqlite"),
			DSN:    getStringOrDefault("DSN", "./ling.db"),
		},
		Log: logger.LogConfig{
			Level:      getStringOrDefault("LOG_LEVEL", "info"),
			Filename:   getStringOrDefault("LOG_FILENAME", "./logs/app.log"),
			MaxSize:    getIntOrDefault("LOG_MAX_SIZE", 100),
			MaxAge:     getIntOrDefault("LOG_MAX_AGE", 30),
			MaxBackups: getIntOrDefault("LOG_MAX_BACKUPS", 5),
			Daily:      getBoolOrDefault("LOG_DAILY", true),
		},
		Cache: loadCacheConfig(),
		Auth: AuthConfig{
			Header:           getStringOrDefault("AUTH_HEADER", "Authorization"),
			SessionSecret:    getStringOrDefault("SESSION_SECRET", generateDefaultSessionSecret()),
			SecretExpireDays: getStringOrDefault("SESSION_EXPIRE_DAYS", "7"),
			APISecretKey:     getStringOrDefault("API_SECRET_KEY", generateDefaultSessionSecret()),
		},
		Services: ServicesConfig{
			LLM: LLMConfig{
				APIKey:  getStringOrDefault("LLM_API_KEY", ""),
				BaseURL: getStringOrDefault("LLM_BASE_URL", "https://api.openai.com/v1"),
				Model:   getStringOrDefault("LLM_MODEL", "gpt-3.5-turbo"),
			},
			Mail: loadMailConfig(),
			KnowledgeBase: KnowledgeBaseConfig{
				Enabled: getBoolOrDefault("KNOWLEDGE_BASE_ENABLED", false),
				Bailian: BailianConfig{
					AccessKeyId:     getStringOrDefault("BAILIAN_ACCESS_KEY_ID", ""),
					AccessKeySecret: getStringOrDefault("BAILIAN_ACCESS_KEY_SECRET", ""),
					Endpoint:        getStringOrDefault("BAILIAN_ENDPOINT", ""),
					WorkspaceId:     getStringOrDefault("BAILIAN_WORKSPACE_ID", ""),
					CategoryId:      getStringOrDefault("BAILIAN_CATEGORY_ID", ""),
					SourceType:      getStringOrDefault("BAILIAN_SOURCE_TYPE", ""),
					Parser:          getStringOrDefault("BAILIAN_PARSER", ""),
					StructType:      getStringOrDefault("BAILIAN_STRUCT_TYPE", ""),
					SinkType:        getStringOrDefault("BAILIAN_SINK_TYPE", ""),
				},
				Milvus: MilvusConfig{
					Address:    getStringOrDefault("MILVUS_ADDRESS", "localhost:19530"),
					Username:   getStringOrDefault("MILVUS_USERNAME", ""),
					Password:   getStringOrDefault("MILVUS_PASSWORD", ""),
					Collection: getStringOrDefault("MILVUS_COLLECTION", ""),
					Dimension:  getIntOrDefault("MILVUS_DIMENSION", 768),
				},
				Qdrant: QdrantConfig{
					BaseURL:    getStringOrDefault("QDRANT_BASE_URL", "http://localhost:6333"),
					APIKey:     getStringOrDefault("QDRANT_API_KEY", ""),
					Collection: getStringOrDefault("QDRANT_COLLECTION", ""),
					Dimension:  getIntOrDefault("QDRANT_DIMENSION", 384),
				},
				Elasticsearch: ElasticsearchConfig{
					BaseURL:  getStringOrDefault("ELASTICSEARCH_BASE_URL", "http://localhost:9200"),
					Username: getStringOrDefault("ELASTICSEARCH_USERNAME", ""),
					Password: getStringOrDefault("ELASTICSEARCH_PASSWORD", ""),
					Index:    getStringOrDefault("ELASTICSEARCH_INDEX", ""),
				},
				Pinecone: PineconeConfig{
					APIKey:    getStringOrDefault("PINECONE_API_KEY", ""),
					BaseURL:   getStringOrDefault("PINECONE_BASE_URL", "https://api.pinecone.io"),
					IndexName: getStringOrDefault("PINECONE_INDEX_NAME", ""),
					Dimension: getIntOrDefault("PINECONE_DIMENSION", 1536),
				},
				Neo4j: Neo4jConfig{
					Enabled:  getBoolOrDefault("NEO4J_ENABLED", false),
					URI:      getStringOrDefault("NEO4J_URI", "bolt://localhost:7687"),
					Username: getStringOrDefault("NEO4J_USERNAME", "neo4j"),
					Password: getStringOrDefault("NEO4J_PASSWORD", ""),
					Database: getStringOrDefault("NEO4J_DATABASE", "neo4j"),
				},
				Embedding: EmbeddingConfig{
					Provider:  getStringOrDefault("EMBEDDING_PROVIDER", "nvidia"),
					BaseURL:   getStringOrDefault("EMBEDDING_BASE_URL", "https://integrate.api.nvidia.com/v1"),
					APIKey:    getStringOrDefault("EMBEDDING_API_KEY", ""),
					Model:     getStringOrDefault("EMBEDDING_MODEL", "nvidia/nv-embedqa-e5-v5"),
					Dimension: getIntOrDefault("EMBEDDING_DIMENSION", 384),
				},
				Rerank: RerankConfig{
					Provider: getStringOrDefault("RERANK_PROVIDER", "siliconflow"),
					BaseURL:  getStringOrDefault("RERANK_BASE_URL", "https://api.siliconflow.cn/v1"),
					APIKey:   getStringOrDefault("RERANK_API_KEY", ""),
					Model:    getStringOrDefault("RERANK_MODEL", "BAAI/bge-reranker-v2-m3"),
					TopN:     getIntOrDefault("RERANK_TOP_N", 5),
				},
			},
			Voice: VoiceConfig{
				Qiniu: QiniuVoiceConfig{
					ASRAPIKey:  getStringOrDefault("QINIU_ASR_API_KEY", ""),
					ASRBaseURL: getStringOrDefault("QINIU_ASR_BASE_URL", ""),
					TTSAPIKey:  getStringOrDefault("QINIU_TTS_API_KEY", ""),
					TTSBaseURL: getStringOrDefault("QINIU_TTS_BASE_URL", ""),
				},
				Xunfei: XunfeiVoiceConfig{
					WSAppId:     getStringOrDefault("XUNFEI_WS_APP_ID", ""),
					WSAPIKey:    getStringOrDefault("XUNFEI_WS_API_KEY", ""),
					WSAPISecret: getStringOrDefault("XUNFEI_WS_API_SECRET", ""),
				},
				Voiceprint: VoiceprintConfig{
					Enabled:             getBoolOrDefault("VOICEPRINT_ENABLED", false),
					BaseURL:             getStringOrDefault("VOICEPRINT_BASE_URL", "http://localhost:8005"),
					APIKey:              getStringOrDefault("VOICEPRINT_API_KEY", ""),
					Timeout:             parseDuration(getStringOrDefault("VOICEPRINT_TIMEOUT", "30s"), 30*time.Second),
					ConnectTimeout:      parseDuration(getStringOrDefault("VOICEPRINT_CONNECT_TIMEOUT", "10s"), 10*time.Second),
					MaxRetries:          getIntOrDefault("VOICEPRINT_MAX_RETRIES", 3),
					RetryInterval:       parseDuration(getStringOrDefault("VOICEPRINT_RETRY_INTERVAL", "1s"), 1*time.Second),
					SimilarityThreshold: getFloatOrDefault("VOICEPRINT_SIMILARITY_THRESHOLD", 0.6),
					MaxCandidates:       getIntOrDefault("VOICEPRINT_MAX_CANDIDATES", 10),
					CacheEnabled:        getBoolOrDefault("VOICEPRINT_CACHE_ENABLED", true),
					CacheTTL:            parseDuration(getStringOrDefault("VOICEPRINT_CACHE_TTL", "5m"), 5*time.Minute),
					LogEnabled:          getBoolOrDefault("VOICEPRINT_LOG_ENABLED", true),
					LogLevel:            getStringOrDefault("VOICEPRINT_LOG_LEVEL", "info"),
				},
			},
			Storage: StorageConfig{
				BaseURL:   getStringOrDefault("LINGSTORAGE_BASE_URL", "https://api.lingstorage.com"),
				APIKey:    getStringOrDefault("LINGSTORAGE_API_KEY", ""),
				APISecret: getStringOrDefault("LINGSTORAGE_API_SECRET", ""),
				Bucket:    getStringOrDefault("LINGSTORAGE_BUCKET", "default"),
			},
		},
		Features: FeaturesConfig{
			SearchEnabled:   getBoolOrDefault("SEARCH_ENABLED", false),
			SearchPath:      getStringOrDefault("SEARCH_PATH", "./search"),
			SearchBatchSize: getIntOrDefault("SEARCH_BATCH_SIZE", 100),
			LanguageEnabled: getBoolOrDefault("LANGUAGE_ENABLED", true),
			BackupEnabled:   getBoolOrDefault("BACKUP_ENABLED", false),
			BackupPath:      getStringOrDefault("BACKUP_PATH", "./backups"),
			BackupSchedule:  getStringOrDefault("BACKUP_SCHEDULE", "0 2 * * *"),
		},
		Middleware: loadMiddlewareConfig(),
	}
	GlobalStore = lingstorage.NewClient(&lingstorage.Config{
		BaseURL:   GlobalConfig.Services.Storage.BaseURL,
		APIKey:    GlobalConfig.Services.Storage.APIKey,
		APISecret: GlobalConfig.Services.Storage.APISecret,
	})

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate database configuration
	if c.Database.DSN == "" {
		return errors.New("database DSN is required")
	}

	// Validate server configuration
	if c.Server.Addr == "" {
		return errors.New("server address is required")
	}

	// Validate knowledge base configuration
	if c.Services.KnowledgeBase.Enabled {
		// Provider is now selected from frontend, so we don't validate it here
		// But we still validate individual provider configurations if they are configured

		// Validate Aliyun if configured
		if c.Services.KnowledgeBase.Bailian.AccessKeyId != "" {
			if c.Services.KnowledgeBase.Bailian.AccessKeySecret == "" {
				return errors.New("bailian access key secret is required when access key ID is set")
			}
		}

		// Validate Milvus if configured
		if c.Services.KnowledgeBase.Milvus.Address != "" {
			// Milvus is configured, validation passed
		}

		// Validate Qdrant if configured
		if c.Services.KnowledgeBase.Qdrant.BaseURL != "" {
			// Qdrant is configured, validation passed
		}

		// Validate Elasticsearch if configured
		if c.Services.KnowledgeBase.Elasticsearch.BaseURL != "" {
			// Elasticsearch is configured, validation passed
		}

		// Validate Pinecone if configured
		if c.Services.KnowledgeBase.Pinecone.APIKey != "" {
			if c.Services.KnowledgeBase.Pinecone.IndexName == "" {
				return errors.New("pinecone index name is required when API key is set")
			}
		}
	}

	// Validate Neo4j configuration
	if c.Services.KnowledgeBase.Neo4j.Enabled {
		if c.Services.KnowledgeBase.Neo4j.URI == "" {
			return errors.New("neo4j URI is required when enabled")
		}
	}

	return nil
}

// getStringOrDefault gets environment variable value, returns default if empty
func getStringOrDefault(key, defaultValue string) string {
	value := utils.GetEnv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getBoolOrDefault gets boolean environment variable value, returns default if empty
func getBoolOrDefault(key string, defaultValue bool) bool {
	value := utils.GetEnv(key)
	if value == "" {
		return defaultValue
	}
	return utils.GetBoolEnv(key)
}

// getIntOrDefault gets integer environment variable value, returns default if empty
func getIntOrDefault(key string, defaultValue int) int {
	value := utils.GetIntEnv(key)
	if value == 0 {
		return defaultValue
	}
	return int(value)
}

// getFloatOrDefault gets float environment variable value, returns default if empty
func getFloatOrDefault(key string, defaultValue float64) float64 {
	value := utils.GetEnv(key)
	if value == "" {
		return defaultValue
	}
	// 简单的字符串到float64转换
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}
	return defaultValue
}

// parseDuration parses duration string with default fallback
func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}

// generateDefaultSessionSecret generates default session secret (for development only)
func generateDefaultSessionSecret() string {
	if secret := utils.GetEnv("SESSION_SECRET"); secret != "" {
		return secret
	}
	return "default-secret-key-change-in-production-" + utils.RandText(16)
}

// loadCacheConfig loads cache configuration with all default values
func loadCacheConfig() cache.Config {
	cacheType := utils.GetEnv("CACHE_TYPE")
	if cacheType == "" {
		cacheType = "local"
	}
	redisAddr := utils.GetEnv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisDB := int(utils.GetIntEnv("REDIS_DB"))
	if redisDB == 0 {
		redisDB = 0
	}
	redisPoolSize := int(utils.GetIntEnv("REDIS_POOL_SIZE"))
	if redisPoolSize == 0 {
		redisPoolSize = 10
	}
	redisMinIdleConns := int(utils.GetIntEnv("REDIS_MIN_IDLE_CONNS"))
	if redisMinIdleConns == 0 {
		redisMinIdleConns = 5
	}
	localMaxSize := int(utils.GetIntEnv("LOCAL_CACHE_MAX_SIZE"))
	if localMaxSize == 0 {
		localMaxSize = 1000
	}
	localDefaultExpiration := parseDuration(utils.GetEnv("LOCAL_CACHE_DEFAULT_EXPIRATION"), 5*time.Minute)
	localCleanupInterval := parseDuration(utils.GetEnv("LOCAL_CACHE_CLEANUP_INTERVAL"), 10*time.Minute)
	return cache.Config{
		Type: cacheType,
		Redis: cache.RedisConfig{
			Addr:         redisAddr,
			Password:     utils.GetEnv("REDIS_PASSWORD"),
			DB:           redisDB,
			PoolSize:     redisPoolSize,
			MinIdleConns: redisMinIdleConns,
			DialTimeout:  parseDuration(utils.GetEnv("REDIS_DIAL_TIMEOUT"), 5*time.Second),
			ReadTimeout:  parseDuration(utils.GetEnv("REDIS_READ_TIMEOUT"), 3*time.Second),
			WriteTimeout: parseDuration(utils.GetEnv("REDIS_WRITE_TIMEOUT"), 3*time.Second),
			IdleTimeout:  parseDuration(utils.GetEnv("REDIS_IDLE_TIMEOUT"), 5*time.Minute),
		},
		Local: cache.LocalConfig{
			MaxSize:           localMaxSize,
			DefaultExpiration: localDefaultExpiration,
			CleanupInterval:   localCleanupInterval,
		},
	}
}

// loadMiddlewareConfig loads middleware configuration
func loadMiddlewareConfig() MiddlewareConfig {
	mode := getStringOrDefault("MODE", "development")
	var defaultConfig MiddlewareConfig

	if mode == "production" {
		defaultConfig = MiddlewareConfig{
			RateLimit: RateLimiterConfig{
				GlobalRPS:    2000,
				GlobalBurst:  4000,
				GlobalWindow: time.Minute,
				UserRPS:      200,
				UserBurst:    400,
				UserWindow:   time.Minute,
				IPRPS:        100,
				IPBurst:      200,
				IPWindow:     time.Minute,
			},
			Timeout: TimeoutConfig{
				DefaultTimeout: 30 * time.Second,
				FallbackResponse: map[string]interface{}{
					"error":   "service_unavailable",
					"message": "Service temporarily unavailable, please try again later",
					"code":    503,
				},
			},
			CircuitBreaker: CircuitBreakerConfig{
				FailureThreshold:      3,
				SuccessThreshold:      2,
				Timeout:               30 * time.Second,
				OpenTimeout:           30 * time.Second,
				MaxConcurrentRequests: 200,
			},
			EnableRateLimit:      true,
			EnableTimeout:        true,
			EnableCircuitBreaker: true,
			EnableOperationLog:   true,
		}
	} else {
		defaultConfig = MiddlewareConfig{
			RateLimit: RateLimiterConfig{
				GlobalRPS:    10000,
				GlobalBurst:  20000,
				GlobalWindow: time.Minute,
				UserRPS:      1000,
				UserBurst:    2000,
				UserWindow:   time.Minute,
				IPRPS:        500,
				IPBurst:      1000,
				IPWindow:     time.Minute,
			},
			Timeout: TimeoutConfig{
				DefaultTimeout: 60 * time.Second,
				FallbackResponse: map[string]interface{}{
					"error":   "service_unavailable",
					"message": "Service temporarily unavailable, please try again later",
					"code":    503,
				},
			},
			CircuitBreaker: CircuitBreakerConfig{
				FailureThreshold:      10,
				SuccessThreshold:      5,
				Timeout:               60 * time.Second,
				OpenTimeout:           60 * time.Second,
				MaxConcurrentRequests: 1000,
			},
			EnableRateLimit:      true,
			EnableTimeout:        true,
			EnableCircuitBreaker: false,
			EnableOperationLog:   true,
		}
	}
	return MiddlewareConfig{
		RateLimit: RateLimiterConfig{
			GlobalRPS:    getIntOrDefault("RATE_LIMIT_GLOBAL_RPS", defaultConfig.RateLimit.GlobalRPS),
			GlobalBurst:  getIntOrDefault("RATE_LIMIT_GLOBAL_BURST", defaultConfig.RateLimit.GlobalBurst),
			GlobalWindow: parseDuration(getStringOrDefault("RATE_LIMIT_GLOBAL_WINDOW", "1m"), defaultConfig.RateLimit.GlobalWindow),
			UserRPS:      getIntOrDefault("RATE_LIMIT_USER_RPS", defaultConfig.RateLimit.UserRPS),
			UserBurst:    getIntOrDefault("RATE_LIMIT_USER_BURST", defaultConfig.RateLimit.UserBurst),
			UserWindow:   parseDuration(getStringOrDefault("RATE_LIMIT_USER_WINDOW", "1m"), defaultConfig.RateLimit.UserWindow),
			IPRPS:        getIntOrDefault("RATE_LIMIT_IP_RPS", defaultConfig.RateLimit.IPRPS),
			IPBurst:      getIntOrDefault("RATE_LIMIT_IP_BURST", defaultConfig.RateLimit.IPBurst),
			IPWindow:     parseDuration(getStringOrDefault("RATE_LIMIT_IP_WINDOW", "1m"), defaultConfig.RateLimit.IPWindow),
		},
		Timeout: TimeoutConfig{
			DefaultTimeout:   parseDuration(getStringOrDefault("DEFAULT_TIMEOUT", "30s"), defaultConfig.Timeout.DefaultTimeout),
			FallbackResponse: defaultConfig.Timeout.FallbackResponse,
		},
		CircuitBreaker: CircuitBreakerConfig{
			FailureThreshold:      getIntOrDefault("CIRCUIT_BREAKER_FAILURE_THRESHOLD", defaultConfig.CircuitBreaker.FailureThreshold),
			SuccessThreshold:      getIntOrDefault("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", defaultConfig.CircuitBreaker.SuccessThreshold),
			Timeout:               parseDuration(getStringOrDefault("CIRCUIT_BREAKER_TIMEOUT", "30s"), defaultConfig.CircuitBreaker.Timeout),
			OpenTimeout:           parseDuration(getStringOrDefault("CIRCUIT_BREAKER_OPEN_TIMEOUT", "30s"), defaultConfig.CircuitBreaker.OpenTimeout),
			MaxConcurrentRequests: getIntOrDefault("CIRCUIT_BREAKER_MAX_CONCURRENT", defaultConfig.CircuitBreaker.MaxConcurrentRequests),
		},
		EnableRateLimit:      getBoolOrDefault("ENABLE_RATE_LIMIT", defaultConfig.EnableRateLimit),
		EnableTimeout:        getBoolOrDefault("ENABLE_TIMEOUT", defaultConfig.EnableTimeout),
		EnableCircuitBreaker: getBoolOrDefault("ENABLE_CIRCUIT_BREAKER", defaultConfig.EnableCircuitBreaker),
		EnableOperationLog:   getBoolOrDefault("ENABLE_OPERATION_LOG", defaultConfig.EnableOperationLog),
	}
}

// loadMailConfig loads mail configuration from environment variables
// Supports both SMTP (legacy MAIL_* vars) and SendCloud (SENDCLOUD_* vars)
func loadMailConfig() notification.MailConfig {
	provider := getStringOrDefault("MAIL_PROVIDER", "")

	// Auto-detect provider based on available environment variables
	if provider == "" {
		// If SendCloud credentials are set, use SendCloud
		if getStringOrDefault("SENDCLOUD_API_USER", "") != "" {
			provider = "sendcloud"
		} else if getStringOrDefault("MAIL_HOST", "") != "" {
			// If MAIL_HOST is set, use SMTP
			provider = "smtp"
		} else {
			// Default to SendCloud
			provider = "sendcloud"
		}
	}

	config := notification.MailConfig{
		Provider: provider,
	}

	if provider == "smtp" {
		// Load SMTP configuration (supports both MAIL_* and SMTP_* vars)
		config.Host = getStringOrDefault("SMTP_HOST", getStringOrDefault("MAIL_HOST", ""))
		config.Username = getStringOrDefault("SMTP_USERNAME", getStringOrDefault("MAIL_USERNAME", ""))
		config.Password = getStringOrDefault("SMTP_PASSWORD", getStringOrDefault("MAIL_PASSWORD", ""))
		config.Port = int64(getIntOrDefault("SMTP_PORT", getIntOrDefault("MAIL_PORT", 587)))
		config.From = getStringOrDefault("MAIL_FROM_EMAIL", getStringOrDefault("MAIL_FROM", ""))
	} else {
		// Load SendCloud configuration
		config.APIUser = getStringOrDefault("SENDCLOUD_API_USER", "")
		config.APIKey = getStringOrDefault("SENDCLOUD_API_KEY", "")
		config.From = getStringOrDefault("MAIL_FROM_EMAIL", getStringOrDefault("SENDCLOUD_FROM_EMAIL", ""))
	}

	return config
}
