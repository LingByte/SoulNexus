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

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/cache"
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
	JWT          JWTConfig
}

// JWTConfig JWT related configuration
type JWTConfig struct {
	Algorithm    string `env:"JWT_ALGORITHM"`
	KeyFile      string `env:"JWT_KEY_FILE"`
	RotationDays int    `env:"JWT_ROTATION_DAYS"`
	KeepOldKeys  int    `env:"JWT_KEEP_OLD_KEYS"`
}

// ServerConfig server configuration
type ServerConfig struct {
	Name        string `env:"SERVER_NAME"`
	Desc        string `env:"SERVER_DESC"`
	URL         string `env:"SERVER_URL"`
	Logo        string `env:"SERVER_LOGO"`
	TermsURL    string `env:"SERVER_TERMS_URL"`
	Addr        string `env:"ADDR"`
	Mode        string `env:"MODE"`
	DocsPrefix  string `env:"DOCS_PREFIX"`
	APIPrefix   string `env:"API_PREFIX"`
	AuthPrefix  string `env:"AUTH_PREFIX"`
	SSLEnabled  bool   `env:"SSL_ENABLED"`
	SSLCertFile string `env:"SSL_CERT_FILE"`
	SSLKeyFile  string `env:"SSL_KEY_FILE"`
}

// DatabaseConfig database configuration
type DatabaseConfig struct {
	Driver string `env:"DB_DRIVER"`
	DSN    string `env:"DSN"`
}

// ServicesConfig services configuration
type ServicesConfig struct {
	LLM     LLMConfig     `mapstructure:"llm"`
	Voice   VoiceConfig   `mapstructure:"voice"`
	Storage StorageConfig `mapstructure:"storage"`
}

// LLMConfig LLM service configuration
type LLMConfig struct {
	APIKey                  string `env:"LLM_API_KEY"`
	BaseURL                 string `env:"LLM_BASE_URL"`
	Model                   string `env:"LLM_MODEL"`
	MemoryCompressThreshold int    `env:"LLM_MEMORY_COMPRESS_THRESHOLD"`
	ShortTermMessageLimit   int    `env:"LLM_SHORT_TERM_MESSAGE_LIMIT"`
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
			Name:        getStringOrDefault("SERVER_NAME", ""),
			Desc:        getStringOrDefault("SERVER_DESC", ""),
			URL:         getStringOrDefault("SERVER_URL", ""),
			Logo:        getStringOrDefault("SERVER_LOGO", ""),
			TermsURL:    getStringOrDefault("SERVER_TERMS_URL", ""),
			Addr:        getStringOrDefault("ADDR", ":7072"),
			Mode:        getStringOrDefault("MODE", "development"),
			DocsPrefix:  getStringOrDefault("DOCS_PREFIX", "/api/docs"),
			APIPrefix:   getStringOrDefault("API_PREFIX", "/api"),
			AuthPrefix:  getStringOrDefault("AUTH_PREFIX", "/auth"),
			SSLEnabled:  getBoolOrDefault("SSL_ENABLED", false),
			SSLCertFile: getStringOrDefault("SSL_CERT_FILE", ""),
			SSLKeyFile:  getStringOrDefault("SSL_KEY_FILE", ""),
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
		JWT: JWTConfig{
			Algorithm:    getStringOrDefault("JWT_ALGORITHM", "RS256"),
			KeyFile:      getStringOrDefault("JWT_KEY_FILE", "./keys/jwks.json"),
			RotationDays: getIntOrDefault("JWT_ROTATION_DAYS", 30),
			KeepOldKeys:  getIntOrDefault("JWT_KEEP_OLD_KEYS", 2),
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
