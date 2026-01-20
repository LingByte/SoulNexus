package config

import (
	"log"
	"os"
	"time"

	"github.com/LingByte/SoulNexus/pkg/cache"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingstorage-sdk-go"
)

// Config System  CommonConfig
type Config struct {
	MachineID        int64  `env:"MACHINE_ID"`
	ServerName       string `env:"SERVER_NAME"`
	ServerDesc       string `env:"SERVER_DESC"`
	ServerUrl        string `env:"SERVER_URL"`
	ServerLogo       string `env:"SERVER_LOGO"`
	ServerTermsUrl   string `env:"SERVER_TERMS_URL"`
	DBDriver         string `env:"DB_DRIVER"`
	DSN              string `env:"DSN"`
	Log              logger.LogConfig
	Mail             notification.MailConfig
	Addr             string `env:"ADDR"`
	Mode             string `env:"MODE"`
	DocsPrefix       string `env:"DOCS_PREFIX"`
	APIPrefix        string `env:"API_PREFIX"`
	AuthHeader       string `env:"AUTH_HEADER"`
	AdminPrefix      string `env:"ADMIN_PREFIX"`
	AuthPrefix       string `env:"AUTH_PREFIX"`
	SessionSecret    string `env:"SESSION_SECRET"`
	SecretExpireDays string `env:"SESSION_EXPIRE_DAYS"`
	LLMApiKey        string `env:"LLM_API_KEY"`
	LLMBaseURL       string `env:"LLM_BASE_URL"`
	LLMModel         string `env:"LLM_MODEL"`
	SearchEnabled    bool   `env:"SEARCH_ENABLED"`
	SearchPath       string `env:"SEARCH_PATH"`
	SearchBatchSize  int    `env:"SEARCH_BATCH_SIZE"`
	MonitorPrefix    string `env:"MONITOR_PREFIX"`
	LanguageEnabled  bool   `env:"LANGUAGE_ENABLED"`
	APISecretKey     string `env:"API_SECRET_KEY"`
	BackupEnabled    bool   `env:"BACKUP_ENABLED"`
	BackupPath       string `env:"BACKUP_PATH"`
	BackupSchedule   string `env:"BACKUP_SCHEDULE"`
	// ASR/TTS配置
	QiniuASRApiKey  string `env:"QINIU_ASR_API_KEY"`
	QiniuASRBaseURL string `env:"QINIU_ASR_BASE_URL"`
	QiniuTTSApiKey  string `env:"QINIU_TTS_API_KEY"`
	QiniuTTSBaseURL string `env:"QINIU_TTS_BASE_URL"`

	XunfeiWsAppId     string `env:"XUNFEI_WS_APP_ID"`
	XunfeiWsApiKey    string `env:"XUNFEI_WS_API_KEY"`
	XunfeiWsApiSecret string `env:"XUNFEI_WS_API_SECRET"`

	// 知识库配置
	KnowledgeBaseEnabled  bool   `env:"KNOWLEDGE_BASE_ENABLED"`  // 是否启用知识库功能
	KnowledgeBaseProvider string `env:"KNOWLEDGE_BASE_PROVIDER"` // 知识库提供者：aliyun, milvus, qdrant, elasticsearch, pinecone

	// 阿里云百炼配置
	BailianAccessKeyId     string `env:"BAILIAN_ACCESS_KEY_ID"`
	BailianAccessKeySecret string `env:"BAILIAN_ACCESS_KEY_SECRET"`
	BailianEndpoint        string `env:"BAILIAN_ENDPOINT"`
	BailianWorkspaceId     string `env:"BAILIAN_WORKSPACE_ID"`
	BailianCategoryId      string `env:"BAILIAN_CATEGORY_ID"`
	BailianSourceType      string `env:"BAILIAN_SOURCE_TYPE"`
	BailianParser          string `env:"BAILIAN_PARSER"`
	BailianStructType      string `env:"BAILIAN_STRUCT_TYPE"`
	BailianSinkType        string `env:"BAILIAN_SINK_TYPE"`

	// Milvus 配置
	MilvusAddress    string `env:"MILVUS_ADDRESS"`    // Milvus 服务器地址（默认: localhost:19530）
	MilvusUsername   string `env:"MILVUS_USERNAME"`   // 用户名（可选）
	MilvusPassword   string `env:"MILVUS_PASSWORD"`   // 密码（可选）
	MilvusCollection string `env:"MILVUS_COLLECTION"` // 集合名称
	MilvusDimension  int    `env:"MILVUS_DIMENSION"`  // 向量维度（默认: 768）

	// Qdrant 配置
	QdrantBaseURL    string `env:"QDRANT_BASE_URL"`   // Qdrant 服务器地址（默认: http://localhost:6333）
	QdrantApiKey     string `env:"QDRANT_API_KEY"`    // API Key（可选）
	QdrantCollection string `env:"QDRANT_COLLECTION"` // 集合名称
	QdrantDimension  int    `env:"QDRANT_DIMENSION"`  // 向量维度（默认: 384）

	// Elasticsearch 配置
	ElasticsearchBaseURL  string `env:"ELASTICSEARCH_BASE_URL"` // Elasticsearch 服务器地址（默认: http://localhost:9200）
	ElasticsearchUsername string `env:"ELASTICSEARCH_USERNAME"` // 用户名（可选）
	ElasticsearchPassword string `env:"ELASTICSEARCH_PASSWORD"` // 密码（可选）
	ElasticsearchIndex    string `env:"ELASTICSEARCH_INDEX"`    // 索引名称

	// Pinecone 配置
	PineconeApiKey    string `env:"PINECONE_API_KEY"`    // API Key（必需）
	PineconeBaseURL   string `env:"PINECONE_BASE_URL"`   // Base URL（默认: https://api.pinecone.io）
	PineconeIndexName string `env:"PINECONE_INDEX_NAME"` // 索引名称（必需）
	PineconeDimension int    `env:"PINECONE_DIMENSION"`  // 向量维度（默认: 1536）

	// 缓存配置
	Cache cache.Config

	// SSL/TLS配置
	SSLEnabled  bool   `env:"SSL_ENABLED"`
	SSLCertFile string `env:"SSL_CERT_FILE"`
	SSLKeyFile  string `env:"SSL_KEY_FILE"`

	// Neo4j 图数据库配置
	Neo4jEnabled         bool   `env:"NEO4J_ENABLED"`  // 是否启用 Neo4j
	Neo4jURI             string `env:"NEO4J_URI"`      // Neo4j 连接 URI（默认: bolt://localhost:7687）
	Neo4jUsername        string `env:"NEO4J_USERNAME"` // Neo4j 用户名（默认: neo4j）
	Neo4jPassword        string `env:"NEO4J_PASSWORD"` // Neo4j 密码
	Neo4jDatabase        string `env:"NEO4J_DATABASE"` // Neo4j 数据库名称（默认: neo4j）
	LingstorageBaseUrl   string `env:"LINGSTORAGE_BASE_URL"`
	LingstorageApiKey    string `env:"LINGSTORAGE_API_KEY"`
	LingstorageApiSecret string `env:"LINGSTORAGE_API_SECRET"`
	LingstorageBucket    string `env:"LINGSTORAGE_BUCKET"`

	// 中间件配置
	Middleware MiddlewareConfig
}

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	// 限流配置
	RateLimit RateLimiterConfig
	// 超时配置
	Timeout TimeoutConfig
	// 熔断器配置
	CircuitBreaker CircuitBreakerConfig
	// 是否启用各个中间件
	EnableRateLimit      bool `env:"ENABLE_RATE_LIMIT"`
	EnableTimeout        bool `env:"ENABLE_TIMEOUT"`
	EnableCircuitBreaker bool `env:"ENABLE_CIRCUIT_BREAKER"`
	EnableOperationLog   bool `env:"ENABLE_OPERATION_LOG"`
}

// RateLimiterConfig 限流配置
type RateLimiterConfig struct {
	// 全局限流配置
	GlobalRPS    int           `env:"RATE_LIMIT_GLOBAL_RPS"`   // 全局每秒请求数
	GlobalBurst  int           `env:"RATE_LIMIT_GLOBAL_BURST"` // 全局突发请求数
	GlobalWindow time.Duration // 全局时间窗口

	// 用户限流配置
	UserRPS    int           `env:"RATE_LIMIT_USER_RPS"`   // 用户每秒请求数
	UserBurst  int           `env:"RATE_LIMIT_USER_BURST"` // 用户突发请求数
	UserWindow time.Duration // 用户时间窗口

	// IP限流配置
	IPRPS    int           `env:"RATE_LIMIT_IP_RPS"`   // IP每秒请求数
	IPBurst  int           `env:"RATE_LIMIT_IP_BURST"` // IP突发请求数
	IPWindow time.Duration // IP时间窗口
}

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	// 默认超时时间
	DefaultTimeout time.Duration `env:"DEFAULT_TIMEOUT"`
	// 超时后的降级响应
	FallbackResponse interface{}
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	// 失败阈值
	FailureThreshold int `env:"CIRCUIT_BREAKER_FAILURE_THRESHOLD"`
	// 成功阈值（半开状态下）
	SuccessThreshold int `env:"CIRCUIT_BREAKER_SUCCESS_THRESHOLD"`
	// 超时时间
	Timeout time.Duration `env:"CIRCUIT_BREAKER_TIMEOUT"`
	// 熔断器打开后的等待时间
	OpenTimeout time.Duration `env:"CIRCUIT_BREAKER_OPEN_TIMEOUT"`
	// 最大并发请求数
	MaxConcurrentRequests int `env:"CIRCUIT_BREAKER_MAX_CONCURRENT"`
}

var GlobalConfig *Config

var GlobalStore *lingstorage.Client

func Load() error {
	// 1. 根据环境加载 .env 文件（如果不存在也不报错，使用默认值）s
	env := os.Getenv("APP_ENV")
	err := utils.LoadEnv(env)
	if err != nil {
		// .env文件不存在时只记录日志，不影响启动
		log.Printf("Note: .env file not found or failed to load: %v (using default values)", err)
	}

	// 2. 加载全局配置（所有配置都有默认值，确保无.env文件也能启动）
	GlobalConfig = &Config{
		MachineID:        utils.GetIntEnv("MACHINE_ID"),
		ServerName:       getStringOrDefault("SERVER_NAME", ""),
		ServerDesc:       getStringOrDefault("SERVER_DESC", ""),
		ServerUrl:        getStringOrDefault("SERVER_URL", ""),
		ServerLogo:       getStringOrDefault("SERVER_LOGO", ""),
		ServerTermsUrl:   getStringOrDefault("SERVER_TERMS_URL", ""),
		DBDriver:         getStringOrDefault("DB_DRIVER", "sqlite"),
		DSN:              getStringOrDefault("DSN", "./ling.db"),
		Addr:             getStringOrDefault("ADDR", ":7072"),
		Mode:             getStringOrDefault("MODE", "development"),
		DocsPrefix:       getStringOrDefault("DOCS_PREFIX", "/api/docs"),
		AuthHeader:       getStringOrDefault("AUTH_HEADER", "Authorization"),
		APIPrefix:        getStringOrDefault("API_PREFIX", "/api"),
		AdminPrefix:      getStringOrDefault("ADMIN_PREFIX", "/admin"),
		AuthPrefix:       getStringOrDefault("AUTH_PREFIX", "/auth"),
		SecretExpireDays: getStringOrDefault("SESSION_EXPIRE_DAYS", "7"),
		SessionSecret:    getStringOrDefault("SESSION_SECRET", generateDefaultSessionSecret()),
		Log: logger.LogConfig{
			Level:      getStringOrDefault("LOG_LEVEL", "info"),
			Filename:   getStringOrDefault("LOG_FILENAME", "./logs/app.log"),
			MaxSize:    getIntOrDefault("LOG_MAX_SIZE", 100),
			MaxAge:     getIntOrDefault("LOG_MAX_AGE", 30),
			MaxBackups: getIntOrDefault("LOG_MAX_BACKUPS", 5),
			Daily:      getBoolOrDefault("LOG_DAILY", true),
		},
		Mail: notification.MailConfig{
			Host:     getStringOrDefault("MAIL_HOST", ""),
			Username: getStringOrDefault("MAIL_USERNAME", ""),
			Password: getStringOrDefault("MAIL_PASSWORD", ""),
			Port:     int64(getIntOrDefault("MAIL_PORT", 587)),
			From:     getStringOrDefault("MAIL_FROM", ""),
		},
		LLMApiKey:       getStringOrDefault("LLM_API_KEY", ""),
		LLMBaseURL:      getStringOrDefault("LLM_BASE_URL", "https://api.openai.com/v1"),
		LLMModel:        getStringOrDefault("LLM_MODEL", "gpt-3.5-turbo"),
		SearchEnabled:   getBoolOrDefault("SEARCH_ENABLED", false),
		SearchPath:      getStringOrDefault("SEARCH_PATH", "./search"),
		SearchBatchSize: getIntOrDefault("SEARCH_BATCH_SIZE", 100),
		MonitorPrefix:   getStringOrDefault("MONITOR_PREFIX", "/metrics"),
		LanguageEnabled: getBoolOrDefault("LANGUAGE_ENABLED", true),
		APISecretKey:    getStringOrDefault("API_SECRET_KEY", generateDefaultSessionSecret()),
		BackupEnabled:   getBoolOrDefault("BACKUP_ENABLED", false),
		BackupPath:      getStringOrDefault("BACKUP_PATH", "./backups"),
		BackupSchedule:  getStringOrDefault("BACKUP_SCHEDULE", "0 2 * * *"),
		// ASR/TTS配置
		QiniuASRApiKey:    getStringOrDefault("QINIU_ASR_API_KEY", ""),
		QiniuASRBaseURL:   getStringOrDefault("QINIU_ASR_BASE_URL", ""),
		QiniuTTSApiKey:    getStringOrDefault("QINIU_TTS_API_KEY", ""),
		QiniuTTSBaseURL:   getStringOrDefault("QINIU_TTS_BASE_URL", ""),
		XunfeiWsAppId:     getStringOrDefault("XUNFEI_WS_APP_ID", ""),
		XunfeiWsApiKey:    getStringOrDefault("XUNFEI_WS_API_KEY", ""),
		XunfeiWsApiSecret: getStringOrDefault("XUNFEI_WS_API_SECRET", ""),
		// 知识库配置
		KnowledgeBaseEnabled:  getBoolOrDefault("KNOWLEDGE_BASE_ENABLED", false),
		KnowledgeBaseProvider: getStringOrDefault("KNOWLEDGE_BASE_PROVIDER", "aliyun"),
		// 阿里云百炼配置
		BailianAccessKeyId:     getStringOrDefault("BAILIAN_ACCESS_KEY_ID", ""),
		BailianAccessKeySecret: getStringOrDefault("BAILIAN_ACCESS_KEY_SECRET", ""),
		BailianEndpoint:        getStringOrDefault("BAILIAN_ENDPOINT", ""),
		BailianWorkspaceId:     getStringOrDefault("BAILIAN_WORKSPACE_ID", ""),
		BailianCategoryId:      getStringOrDefault("BAILIAN_CATEGORY_ID", ""),
		BailianSourceType:      getStringOrDefault("BAILIAN_SOURCE_TYPE", ""),
		BailianParser:          getStringOrDefault("BAILIAN_PARSER", ""),
		BailianStructType:      getStringOrDefault("BAILIAN_STRUCT_TYPE", ""),
		BailianSinkType:        getStringOrDefault("BAILIAN_SINK_TYPE", ""),
		// Milvus 配置
		MilvusAddress:    getStringOrDefault("MILVUS_ADDRESS", "localhost:19530"),
		MilvusUsername:   getStringOrDefault("MILVUS_USERNAME", ""),
		MilvusPassword:   getStringOrDefault("MILVUS_PASSWORD", ""),
		MilvusCollection: getStringOrDefault("MILVUS_COLLECTION", ""),
		MilvusDimension:  getIntOrDefault("MILVUS_DIMENSION", 768),
		// Qdrant 配置
		QdrantBaseURL:    getStringOrDefault("QDRANT_BASE_URL", "http://localhost:6333"),
		QdrantApiKey:     getStringOrDefault("QDRANT_API_KEY", ""),
		QdrantCollection: getStringOrDefault("QDRANT_COLLECTION", ""),
		QdrantDimension:  getIntOrDefault("QDRANT_DIMENSION", 384),
		// Elasticsearch 配置
		ElasticsearchBaseURL:  getStringOrDefault("ELASTICSEARCH_BASE_URL", "http://localhost:9200"),
		ElasticsearchUsername: getStringOrDefault("ELASTICSEARCH_USERNAME", ""),
		ElasticsearchPassword: getStringOrDefault("ELASTICSEARCH_PASSWORD", ""),
		ElasticsearchIndex:    getStringOrDefault("ELASTICSEARCH_INDEX", ""),
		// Pinecone 配置
		PineconeApiKey:    getStringOrDefault("PINECONE_API_KEY", ""),
		PineconeBaseURL:   getStringOrDefault("PINECONE_BASE_URL", "https://api.pinecone.io"),
		PineconeIndexName: getStringOrDefault("PINECONE_INDEX_NAME", ""),
		PineconeDimension: getIntOrDefault("PINECONE_DIMENSION", 1536),
		// 缓存配置
		Cache: loadCacheConfig(),
		// SSL/TLS配置（默认禁用）
		SSLEnabled:  getBoolOrDefault("SSL_ENABLED", false),
		SSLCertFile: getStringOrDefault("SSL_CERT_FILE", ""),
		SSLKeyFile:  getStringOrDefault("SSL_KEY_FILE", ""),
		// Neo4j 图数据库配置（默认禁用）
		Neo4jEnabled:         getBoolOrDefault("NEO4J_ENABLED", false),
		Neo4jURI:             getStringOrDefault("NEO4J_URI", "bolt://localhost:7687"),
		Neo4jUsername:        getStringOrDefault("NEO4J_USERNAME", "neo4j"),
		Neo4jPassword:        getStringOrDefault("NEO4J_PASSWORD", ""),
		Neo4jDatabase:        getStringOrDefault("NEO4J_DATABASE", "neo4j"),
		LingstorageBaseUrl:   getStringOrDefault("LINGSTORAGE_BASE_URL", "https://api.lingstorage.com"),
		LingstorageApiKey:    getStringOrDefault("LINGSTORAGE_API_KEY", ""),
		LingstorageApiSecret: getStringOrDefault("LINGSTORAGE_API_SECRET", ""),
		LingstorageBucket:    getStringOrDefault("LINGSTORAGE_BUCKET", "default"),
		// 中间件配置
		Middleware: loadMiddlewareConfig(),
	}
	GlobalStore = lingstorage.NewClient(&lingstorage.Config{
		BaseURL:   GlobalConfig.LingstorageBaseUrl,
		APIKey:    GlobalConfig.LingstorageApiKey,
		APISecret: GlobalConfig.LingstorageApiSecret,
	})

	return nil
}

// getStringOrDefault 获取环境变量值，如果为空则返回默认值
func getStringOrDefault(key, defaultValue string) string {
	value := utils.GetEnv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getBoolOrDefault 获取布尔环境变量值，如果为空则返回默认值
func getBoolOrDefault(key string, defaultValue bool) bool {
	value := utils.GetEnv(key)
	if value == "" {
		return defaultValue
	}
	return utils.GetBoolEnv(key)
}

// getIntOrDefault 获取整数环境变量值，如果为空则返回默认值
func getIntOrDefault(key string, defaultValue int) int {
	value := utils.GetIntEnv(key)
	if value == 0 {
		return defaultValue
	}
	return int(value)
}

// generateDefaultSessionSecret 生成默认的会话密钥（仅用于开发环境）
func generateDefaultSessionSecret() string {
	// 如果环境变量中已有值，使用环境变量
	if secret := utils.GetEnv("SESSION_SECRET"); secret != "" {
		return secret
	}
	// 否则生成一个随机字符串（仅用于开发）
	return "default-secret-key-change-in-production-" + utils.RandText(16)
}

// loadCacheConfig 加载缓存配置，设置所有默认值
func loadCacheConfig() cache.Config {
	cacheType := utils.GetEnv("CACHE_TYPE")
	if cacheType == "" {
		cacheType = "local"
	}

	// 解析时间字符串的辅助函数
	parseDuration := func(s string, defaultVal time.Duration) time.Duration {
		if s == "" {
			return defaultVal
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return defaultVal
		}
		return d
	}

	// Redis 配置
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

	// 本地缓存配置
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

// loadMiddlewareConfig 加载中间件配置
func loadMiddlewareConfig() MiddlewareConfig {
	// 解析时间字符串的辅助函数
	parseDuration := func(s string, defaultVal time.Duration) time.Duration {
		if s == "" {
			return defaultVal
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return defaultVal
		}
		return d
	}

	// 根据环境模式设置默认值
	mode := getStringOrDefault("MODE", "development")
	var defaultConfig MiddlewareConfig

	if mode == "production" {
		// 生产环境配置
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
					"message": "服务暂时不可用，请稍后重试",
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
		// 开发环境配置
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
					"message": "服务暂时不可用，请稍后重试",
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
			EnableCircuitBreaker: false, // 开发环境关闭熔断器
			EnableOperationLog:   true,
		}
	}

	// 从环境变量覆盖配置
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
