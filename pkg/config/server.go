package config

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
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
	RTCSFU       RTCSFUConfig       `mapstructure:"rtcs"`
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
	APIPrefix  string `env:"API_PREFIX"`
	AuthPrefix string `env:"AUTH_PREFIX"`
	SSLEnabled bool   `env:"SSL_ENABLED"`
	SSLCertFile   string `env:"SSL_CERT_FILE"`
	SSLKeyFile    string `env:"SSL_KEY_FILE"`
}

// RTCSFUConfig enables the hybrid multi-SFU control-plane HTTP API (pkg/rtcsfu) and embedded Pion SFU.
type RTCSFUConfig struct {
	Enabled   bool   `env:"RTCSFU_ENABLED"`
	NodesJSON string `env:"RTCSFU_NODES"`   // JSON array, see pkg/rtcsfu.ParseNodesJSON
	APIKey    string `env:"RTCSFU_API_KEY"` // optional; if set, join requires header X-RTCSFU-Key

	ICEServersJSON       string `env:"RTCSFU_ICE_SERVERS_JSON"`       // browser-style JSON array; empty = default STUN
	MaxRooms             int    `env:"RTCSFU_MAX_ROOMS"`              // 0 = unlimited
	MaxPeersPerRoom      int    `env:"RTCSFU_MAX_PEERS_PER_ROOM"`     // 0 = unlimited
	WSReadTimeoutSec     int    `env:"RTCSFU_WS_READ_TIMEOUT_SEC"`    // WebSocket read deadline (ping/pong refresh)
	WSPingIntervalSec    int    `env:"RTCSFU_WS_PING_INTERVAL_SEC"`   // 0 = disable WS ping
	WSMaxMessageBytes    int    `env:"RTCSFU_WS_MAX_MESSAGE_BYTES"`   // signaling frame size cap
	SignalAllowedOrigins string `env:"RTCSFU_SIGNAL_ALLOWED_ORIGINS"` // comma-separated Origin allowlist; empty: prod=same-host only
	SignalRequireAuth    bool   `env:"RTCSFU_SIGNAL_REQUIRE_AUTH"`    // if true, signal WS requires same auth as join when API key is set

	JoinTokenSecret            string `env:"RTCSFU_JOIN_TOKEN_SECRET"`              // HMAC secret for short-lived room tokens (optional)
	JoinTokenDefaultTTLSeconds int    `env:"RTCSFU_JOIN_TOKEN_DEFAULT_TTL_SECONDS"` // mint default when ttl_sec omitted
	JoinTokenMaxTTLSeconds     int    `env:"RTCSFU_JOIN_TOKEN_MAX_TTL_SECONDS"`     // cap mint ttl; 0 = use default cap in handler

	// ClusterRole is informational + gates some admin APIs: primary | replica | standalone (default standalone).
	ClusterRole string `env:"RTCSFU_CLUSTER_ROLE"`

	// Replica → primary registration (HTTP). Auth: X-RTCSFU-Key must match primary RTCSFU_API_KEY.
	PrimaryBaseURL      string `env:"RTCSFU_PRIMARY_BASE_URL"`      // e.g. https://primary.example.com:7075
	ReplicaSelfJSON     string `env:"RTCSFU_REPLICA_SELF_JSON"`     // one node object or array (same shape as RTCSFU_NODES items)
	ReplicaHeartbeatSec int    `env:"RTCSFU_REPLICA_HEARTBEAT_SEC"` // interval for re-register; 0 = default 30
	// ReplicaHeartbeatMode: register (full JSON each tick) or touch (POST .../replica/touch after one initial register).
	ReplicaHeartbeatMode string `env:"RTCSFU_REPLICA_HEARTBEAT_MODE"`

	ReplicaMTLSClientCAFile       string `env:"RTCSFU_REPLICA_MTLS_CA_FILE"`              // PEM CA to verify primary server cert
	ReplicaMTLSClientCertFile     string `env:"RTCSFU_REPLICA_MTLS_CERT_FILE"`            // client cert for mTLS
	ReplicaMTLSClientKeyFile      string `env:"RTCSFU_REPLICA_MTLS_KEY_FILE"`             // client key
	ReplicaMTLSInsecureSkipVerify bool   `env:"RTCSFU_REPLICA_MTLS_INSECURE_SKIP_VERIFY"` // dev only

	// Primary: if last_seen older than this many seconds, replica is treated as unhealthy for routing (0 = off).
	ReplicaStaleSeconds int `env:"RTCSFU_REPLICA_STALE_SECONDS"`
	// Primary: if set, POST .../replica/touch must include valid HMAC (Bearer or JSON touch_token) bound to id.
	ReplicaTouchHMACSecret string `env:"RTCSFU_REPLICA_TOUCH_HMAC_SECRET"`
	// Replica: TTL for minted touch tokens (seconds); 0 = default 120.
	ReplicaTouchTokenTTLSeconds int `env:"RTCSFU_REPLICA_TOUCH_TOKEN_TTL_SECONDS"`
}

// DatabaseConfig database configuration
type DatabaseConfig struct {
	Driver string `env:"DB_DRIVER"`
	DSN    string `env:"DSN"`
}

// ServicesConfig services configuration
type ServicesConfig struct {
	LLM         LLMConfig               `mapstructure:"llm"`
	Mail        notification.MailConfig `mapstructure:"mail"`
	GraphMemory GraphMemoryConfig       `mapstructure:"graph_memory"`
	Voice       VoiceConfig             `mapstructure:"voice"`
	Storage     StorageConfig           `mapstructure:"storage"`
}

// LLMConfig LLM service configuration
type LLMConfig struct {
	APIKey                  string `env:"LLM_API_KEY"`
	BaseURL                 string `env:"LLM_BASE_URL"`
	Model                   string `env:"LLM_MODEL"`
	MemoryCompressThreshold int    `env:"LLM_MEMORY_COMPRESS_THRESHOLD"`
	ShortTermMessageLimit   int    `env:"LLM_SHORT_TERM_MESSAGE_LIMIT"`
}

// GraphMemoryConfig graph memory configuration
type GraphMemoryConfig struct {
	Neo4j Neo4jConfig `mapstructure:"neo4j"`
}

// Neo4jConfig Neo4j configuration
type Neo4jConfig struct {
	Enabled  bool   `env:"NEO4J_ENABLED"`
	URI      string `env:"NEO4J_URI"`
	Username string `env:"NEO4J_USERNAME"`
	Password string `env:"NEO4J_PASSWORD"`
	Database string `env:"NEO4J_DATABASE"`
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

var GlobalConfig *Config

var GlobalStore *lingstorage.Client

// resolveAppEnv returns APP_ENV or MODE for layered files like .env-server.development.
func resolveAppEnv() string {
	if v := strings.TrimSpace(os.Getenv("APP_ENV")); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("MODE"))
}

// Load loads `.env-server` then `.env-server.{APP_ENV|MODE}` and builds GlobalConfig (cmd/server, cmd/sip, tests).
func Load() error {
	if err := utils.LoadModuleEnvs("server", resolveAppEnv()); err != nil {
		log.Printf("Note: module env load: %v", err)
	}
	return buildGlobalConfig()
}

func buildGlobalConfig() error {
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
			APIPrefix:  getStringOrDefault("API_PREFIX", "/api"),
			AuthPrefix: getStringOrDefault("AUTH_PREFIX", "/auth"),
			SSLEnabled: getBoolOrDefault("SSL_ENABLED", false),
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
				APIKey:                  getStringOrDefault("LLM_API_KEY", ""),
				BaseURL:                 getStringOrDefault("LLM_BASE_URL", "https://api.openai.com/v1"),
				Model:                   getStringOrDefault("LLM_MODEL", "gpt-3.5-turbo"),
				MemoryCompressThreshold: getIntOrDefault("LLM_MEMORY_COMPRESS_THRESHOLD", 20),
				ShortTermMessageLimit:   getIntOrDefault("LLM_SHORT_TERM_MESSAGE_LIMIT", 20),
			},
			Mail: loadMailConfig(),
			GraphMemory: GraphMemoryConfig{
				Neo4j: Neo4jConfig{
					Enabled:  getBoolOrDefault("NEO4J_ENABLED", false),
					URI:      getStringOrDefault("NEO4J_URI", "bolt://localhost:7687"),
					Username: getStringOrDefault("NEO4J_USERNAME", "neo4j"),
					Password: getStringOrDefault("NEO4J_PASSWORD", ""),
					Database: getStringOrDefault("NEO4J_DATABASE", "neo4j"),
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
		RTCSFU: RTCSFUConfig{
			Enabled:                       getBoolOrDefault("RTCSFU_ENABLED", false),
			NodesJSON:                     getStringOrDefault("RTCSFU_NODES", ""),
			APIKey:                        getStringOrDefault("RTCSFU_API_KEY", ""),
			ICEServersJSON:                getStringOrDefault("RTCSFU_ICE_SERVERS_JSON", ""),
			MaxRooms:                      getIntOrDefault("RTCSFU_MAX_ROOMS", 5000),
			MaxPeersPerRoom:               getIntOrDefault("RTCSFU_MAX_PEERS_PER_ROOM", 50),
			WSReadTimeoutSec:              getIntOrDefault("RTCSFU_WS_READ_TIMEOUT_SEC", 70),
			WSPingIntervalSec:             getIntOrDefault("RTCSFU_WS_PING_INTERVAL_SEC", 25),
			WSMaxMessageBytes:             getIntOrDefault("RTCSFU_WS_MAX_MESSAGE_BYTES", 786432),
			SignalAllowedOrigins:          getStringOrDefault("RTCSFU_SIGNAL_ALLOWED_ORIGINS", ""),
			SignalRequireAuth:             getBoolOrDefault("RTCSFU_SIGNAL_REQUIRE_AUTH", false),
			JoinTokenSecret:               getStringOrDefault("RTCSFU_JOIN_TOKEN_SECRET", ""),
			JoinTokenDefaultTTLSeconds:    getIntOrDefault("RTCSFU_JOIN_TOKEN_DEFAULT_TTL_SECONDS", 3600),
			JoinTokenMaxTTLSeconds:        getIntOrDefault("RTCSFU_JOIN_TOKEN_MAX_TTL_SECONDS", 0),
			ClusterRole:                   getStringOrDefault("RTCSFU_CLUSTER_ROLE", "standalone"),
			PrimaryBaseURL:                getStringOrDefault("RTCSFU_PRIMARY_BASE_URL", ""),
			ReplicaSelfJSON:               getStringOrDefault("RTCSFU_REPLICA_SELF_JSON", ""),
			ReplicaHeartbeatSec:           getIntOrDefault("RTCSFU_REPLICA_HEARTBEAT_SEC", 30),
			ReplicaHeartbeatMode:          getStringOrDefault("RTCSFU_REPLICA_HEARTBEAT_MODE", "register"),
			ReplicaMTLSClientCAFile:       getStringOrDefault("RTCSFU_REPLICA_MTLS_CA_FILE", ""),
			ReplicaMTLSClientCertFile:     getStringOrDefault("RTCSFU_REPLICA_MTLS_CERT_FILE", ""),
			ReplicaMTLSClientKeyFile:      getStringOrDefault("RTCSFU_REPLICA_MTLS_KEY_FILE", ""),
			ReplicaMTLSInsecureSkipVerify: getBoolOrDefault("RTCSFU_REPLICA_MTLS_INSECURE_SKIP_VERIFY", false),
			ReplicaStaleSeconds:           getIntOrDefault("RTCSFU_REPLICA_STALE_SECONDS", 0),
			ReplicaTouchHMACSecret:        getStringOrDefault("RTCSFU_REPLICA_TOUCH_HMAC_SECRET", ""),
			ReplicaTouchTokenTTLSeconds:   getIntOrDefault("RTCSFU_REPLICA_TOUCH_TOKEN_TTL_SECONDS", 0),
		},
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

	// Validate Neo4j configuration
	if c.Services.GraphMemory.Neo4j.Enabled {
		if c.Services.GraphMemory.Neo4j.URI == "" {
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
