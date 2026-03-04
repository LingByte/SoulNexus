package voiceprint

import (
	"strconv"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

// Config 声纹识别服务配置
type Config struct {
	// 服务基础配置
	Enabled             bool          `env:"VOICEPRINT_ENABLED" mapstructure:"enabled"`
	BaseURL             string        `env:"VOICEPRINT_BASE_URL" mapstructure:"base_url"`
	APIKey              string        `env:"VOICEPRINT_API_KEY" mapstructure:"api_key"`
	Timeout             time.Duration `env:"VOICEPRINT_TIMEOUT" mapstructure:"timeout"`
	ConnectTimeout      time.Duration `env:"VOICEPRINT_CONNECT_TIMEOUT" mapstructure:"connect_timeout"`
	MaxRetries          int           `env:"VOICEPRINT_MAX_RETRIES" mapstructure:"max_retries"`
	RetryInterval       time.Duration `env:"VOICEPRINT_RETRY_INTERVAL" mapstructure:"retry_interval"`
	SimilarityThreshold float64       `env:"VOICEPRINT_SIMILARITY_THRESHOLD" mapstructure:"similarity_threshold"`
	MaxCandidates       int           `env:"VOICEPRINT_MAX_CANDIDATES" mapstructure:"max_candidates"`
	CacheEnabled        bool          `env:"VOICEPRINT_CACHE_ENABLED" mapstructure:"cache_enabled"`
	CacheTTL            time.Duration `env:"VOICEPRINT_CACHE_TTL" mapstructure:"cache_ttl"`
	LogEnabled          bool          `env:"VOICEPRINT_LOG_ENABLED" mapstructure:"log_enabled"`
	LogLevel            string        `env:"VOICEPRINT_LOG_LEVEL" mapstructure:"log_level"`
}

// DefaultConfig 返回默认配置，从环境变量读取
func DefaultConfig() *Config {
	// 读取环境变量，如果不存在则使用默认值
	enabled := utils.GetEnv("VOICEPRINT_ENABLED") == "true"
	baseURL := utils.GetEnv("VOICEPRINT_BASE_URL")
	if baseURL == "" {
		baseURL = utils.GetEnv("VOICEPRINT_SERVICE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:8005"
		}
	}
	apiKey := utils.GetEnv("VOICEPRINT_API_KEY")

	// 解析超时时间
	timeoutStr := utils.GetEnv("VOICEPRINT_TIMEOUT")
	timeout := 30 * time.Second
	if timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}

	connectTimeoutStr := utils.GetEnv("VOICEPRINT_CONNECT_TIMEOUT")
	connectTimeout := 10 * time.Second
	if connectTimeoutStr != "" {
		if d, err := time.ParseDuration(connectTimeoutStr); err == nil {
			connectTimeout = d
		}
	}

	// 解析整数配置
	maxRetries := 3
	if maxRetriesStr := utils.GetEnv("VOICEPRINT_MAX_RETRIES"); maxRetriesStr != "" {
		if v, err := strconv.Atoi(maxRetriesStr); err == nil {
			maxRetries = v
		}
	}

	retryIntervalStr := utils.GetEnv("VOICEPRINT_RETRY_INTERVAL")
	retryInterval := 1 * time.Second
	if retryIntervalStr != "" {
		if d, err := time.ParseDuration(retryIntervalStr); err == nil {
			retryInterval = d
		}
	}

	// 解析浮点数配置
	similarityThreshold := 0.6
	if thresholdStr := utils.GetEnv("VOICEPRINT_SIMILARITY_THRESHOLD"); thresholdStr != "" {
		if v, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			similarityThreshold = v
		}
	}

	maxCandidates := 10
	if maxCandidatesStr := utils.GetEnv("VOICEPRINT_MAX_CANDIDATES"); maxCandidatesStr != "" {
		if v, err := strconv.Atoi(maxCandidatesStr); err == nil {
			maxCandidates = v
		}
	}

	// 解析布尔值配置
	cacheEnabled := true
	if cacheEnabledStr := utils.GetEnv("VOICEPRINT_CACHE_ENABLED"); cacheEnabledStr != "" {
		cacheEnabled = cacheEnabledStr == "true"
	}

	cacheTTLStr := utils.GetEnv("VOICEPRINT_CACHE_TTL")
	cacheTTL := 5 * time.Minute
	if cacheTTLStr != "" {
		if d, err := time.ParseDuration(cacheTTLStr); err == nil {
			cacheTTL = d
		}
	}

	logEnabled := true
	if logEnabledStr := utils.GetEnv("VOICEPRINT_LOG_ENABLED"); logEnabledStr != "" {
		logEnabled = logEnabledStr == "true"
	}

	logLevel := utils.GetEnv("VOICEPRINT_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	return &Config{
		Enabled:             enabled,
		BaseURL:             baseURL,
		APIKey:              apiKey,
		Timeout:             timeout,
		ConnectTimeout:      connectTimeout,
		MaxRetries:          maxRetries,
		RetryInterval:       retryInterval,
		SimilarityThreshold: similarityThreshold,
		MaxCandidates:       maxCandidates,
		CacheEnabled:        cacheEnabled,
		CacheTTL:            cacheTTL,
		LogEnabled:          logEnabled,
		LogLevel:            logLevel,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.BaseURL == "" {
		return ErrInvalidConfig("base_url is required")
	}
	if c.APIKey == "" {
		return ErrInvalidConfig("api_key is required")
	}
	if c.SimilarityThreshold < 0 || c.SimilarityThreshold > 1 {
		return ErrInvalidConfig("similarity_threshold must be between 0 and 1")
	}
	if c.MaxCandidates <= 0 {
		return ErrInvalidConfig("max_candidates must be greater than 0")
	}
	return nil
}
