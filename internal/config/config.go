package config

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	apperror "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/LingByte/SoulNexus/pkg/utils/common"

	"github.com/LingByte/SoulNexus/internal/constants"
	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	llmcache "github.com/LingByte/lingllm/cache"
)

// Config main configuration structure
type Config struct {
	MachineID  int64            `env:"MACHINE_ID"`
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Log        logger.LogConfig `mapstructure:"log"`
	Auth       AuthConfig       `mapstructure:"auth"`
	Middleware MiddlewareConfig `mapstructure:"middleware"`
	JWT        JWTConfig        `mapstructure:"jwt"`
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
	SSLEnabled  bool   `env:"SSL_ENABLED"`
	SSLCertFile string `env:"SSL_CERT_FILE"`
	SSLKeyFile  string `env:"SSL_KEY_FILE"`
}

// DatabaseConfig database configuration
type DatabaseConfig struct {
	Driver string `env:"DB_DRIVER"`
	DSN    string `env:"DSN"`
}

// AuthConfig authentication configuration
type AuthConfig struct {
	Header               string `env:"AUTH_HEADER"`
	SessionSecret        string `env:"SESSION_SECRET"`
	SecretExpireDays     string `env:"SESSION_EXPIRE_DAYS"`
	APISecretKey         string `env:"API_SECRET_KEY"`
	GitHubClientID       string `env:"GITHUB_CLIENT_ID"`
	GitHubClientSecret   string `env:"GITHUB_CLIENT_SECRET"`
	GitHubRedirectURL    string `env:"GITHUB_OAUTH_REDIRECT_URL"`
	OAuthFrontendBaseURL string `env:"OAUTH_FRONTEND_BASE_URL"`
}

// LLMConfig LLM service configuration
type LLMConfig struct {
	APIKey  string `env:"LLM_API_KEY"`
	BaseURL string `env:"LLM_BASE_URL"`
	Model   string `env:"LLM_MODEL"`
}

// MiddlewareConfig middleware configuration
type MiddlewareConfig struct {
	// Rate limiting configuration
	RateLimit RateLimiterConfig
	// Timeout configuration
	Timeout TimeoutConfig
	// Circuit breaker configuration
	CircuitBreaker CircuitBreakerConfig
	// IP ACL for HTTP API (global + per-route overlays)
	IPACL IPACLConfig
	// Degrade / fallback when circuit open or overload
	Degrade              DegradeConfig
	EnableRateLimit      bool `env:"ENABLE_RATE_LIMIT"`
	EnableTimeout        bool `env:"ENABLE_TIMEOUT"`
	EnableCircuitBreaker bool `env:"ENABLE_CIRCUIT_BREAKER"`
}

// RateLimitRule is one token-bucket limit (used for module/api overlays).
type RateLimitRule struct {
	RPS    int           `json:"rps"`
	Burst  int           `json:"burst"`
	Window time.Duration `json:"window"`
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
	// Module dimension defaults (per first path segment under API prefix)
	ModuleDefaultRPS   int    `env:"RATE_LIMIT_MODULE_DEFAULT_RPS"`
	ModuleDefaultBurst int    `env:"RATE_LIMIT_MODULE_DEFAULT_BURST"`
	ModuleRulesJSON    string `env:"RATE_LIMIT_MODULE_RULES"` // {"knowledge":{"rps":50,"burst":100},...}
	// API dimension (path prefix or exact path → rule)
	APIRulesJSON string `env:"RATE_LIMIT_API_RULES"` // {"/api/knowledge/":{"rps":30,"burst":60},...}
}

// IPACLConfig HTTP IP allow/block lists (global + RouteIPACL overlays).
type IPACLConfig struct {
	Enable     bool   `env:"HTTP_IP_ACL_ENABLE"`
	AllowedIPs string `env:"HTTP_ALLOWED_IPS"` // comma IP/CIDR; empty = allow all (except blocked)
	BlockedIPs string `env:"HTTP_BLOCKED_IPS"` // comma IP/CIDR; checked first
}

// DegradeConfig controls fallback responses when circuit breaker blocks.
type DegradeConfig struct {
	Enable bool `env:"ENABLE_DEGRADE"`
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

func Load() error {
	// 1. Load .env file based on environment (don't error if it doesn't exist, use default values)
	envName := os.Getenv("MODE")
	err := common.LoadEnv(envName)
	if err != nil {
		// Only log when .env file doesn't exist, don't affect startup
		log.Printf("Note: .env file not found or failed to load: %v (using default values)", err)
	}

	// 2. Load global configuration
	GlobalConfig = &Config{
		MachineID: common.GetIntEnv("MACHINE_ID"),
		Server: ServerConfig{
			Name:        common.GetStringOrDefault("SERVER_NAME", ""),
			Desc:        common.GetStringOrDefault("SERVER_DESC", ""),
			URL:         common.GetStringOrDefault("SERVER_URL", ""),
			Logo:        common.GetStringOrDefault("SERVER_LOGO", ""),
			TermsURL:    common.GetStringOrDefault("SERVER_TERMS_URL", ""),
			Addr:        common.GetStringOrDefault("ADDR", ":8082"),
			Mode:        common.GetStringOrDefault("MODE", pkgconst.ENV_DEV),
			DocsPrefix:  common.GetStringOrDefault("DOCS_PREFIX", "/api/docs"),
			APIPrefix:   common.GetStringOrDefault("API_PREFIX", "/api"),
			SSLEnabled:  common.GetBoolOrDefault("SSL_ENABLED", false),
			SSLCertFile: common.GetStringOrDefault("SSL_CERT_FILE", ""),
			SSLKeyFile:  common.GetStringOrDefault("SSL_KEY_FILE", ""),
		},
		Database: DatabaseConfig{
			Driver: common.GetStringOrDefault(constants.ENV_DB_DRIVER, pkgconst.DBDriverSQLite),
			DSN:    common.GetStringOrDefault(constants.ENV_DSN, "./ling.db"),
		},
		Log: func() logger.LogConfig {
			retentionDays := common.GetIntOrDefault("LOG_RETENTION_DAYS", 7)
			maxAge := retentionDays
			if v := os.Getenv("LOG_MAX_AGE"); v != "" {
				maxAge = common.GetIntOrDefault("LOG_MAX_AGE", retentionDays)
			}
			return logger.LogConfig{
				Level:           common.GetStringOrDefault("LOG_LEVEL", "info"),
				Filename:        common.GetStringOrDefault("LOG_FILENAME", "./logs/app.log"),
				MaxSize:         common.GetIntOrDefault("LOG_MAX_SIZE", 100),
				MaxAge:          maxAge,
				RetentionDays:   retentionDays,
				MaxBackups:      common.GetIntOrDefault("LOG_MAX_BACKUPS", 5),
				Daily:           common.GetBoolOrDefault("LOG_DAILY", true),
				SensitiveFields: common.GetStringOrDefault("LOG_SENSITIVE_FIELDS", logger.DefaultSensitiveFields),
			}
		}(),
		Auth: AuthConfig{
			Header:               common.GetStringOrDefault("AUTH_HEADER", "Authorization"),
			SessionSecret:        common.GetStringOrDefault("SESSION_SECRET", ""),
			SecretExpireDays:     common.GetStringOrDefault("SESSION_EXPIRE_DAYS", "7"),
			APISecretKey:         common.GetStringOrDefault("API_SECRET_KEY", ""),
			GitHubClientID:       common.GetStringOrDefault("GITHUB_CLIENT_ID", ""),
			GitHubClientSecret:   common.GetStringOrDefault("GITHUB_CLIENT_SECRET", ""),
			GitHubRedirectURL:    common.GetStringOrDefault("GITHUB_OAUTH_REDIRECT_URL", ""),
			OAuthFrontendBaseURL: common.GetStringOrDefault("OAUTH_FRONTEND_BASE_URL", ""),
		},
		Middleware: loadMiddlewareConfig(),
		JWT: JWTConfig{
			Algorithm:    common.GetStringOrDefault("JWT_ALGORITHM", "RS256"),
			KeyFile:      common.GetStringOrDefault("JWT_KEY_FILE", "./data/keys/jwks.json"),
			RotationDays: common.GetIntOrDefault("JWT_ROTATION_DAYS", 30),
			KeepOldKeys:  common.GetIntOrDefault("JWT_KEEP_OLD_KEYS", 2),
		},
	}
	timeutil.Init(common.GetStringOrDefault("SYSTEM_TIMEZONE", timeutil.DefaultTimezone))
	err = InitFromEnv()
	if err != nil {
		return err
	}
	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate database configuration
	if c.Database.DSN == "" {
		return apperror.ErrDSNRequired
	}
	// Validate server configuration
	if c.Server.Addr == "" {
		return apperror.ErrServerAddrRequired
	}

	return nil
}

// loadMiddlewareConfig loads middleware configuration
func loadMiddlewareConfig() MiddlewareConfig {
	mode := common.GetStringOrDefault("MODE", pkgconst.ENV_DEV)
	var defaultConfig MiddlewareConfig

	if mode == pkgconst.ENV_PROD {
		defaultConfig = MiddlewareConfig{
			RateLimit: RateLimiterConfig{
				GlobalRPS:          2000,
				GlobalBurst:        4000,
				GlobalWindow:       time.Minute,
				UserRPS:            200,
				UserBurst:          400,
				UserWindow:         time.Minute,
				IPRPS:              100,
				IPBurst:            200,
				IPWindow:           time.Minute,
				ModuleDefaultRPS:   500,
				ModuleDefaultBurst: 1000,
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
		}
	} else {
		defaultConfig = MiddlewareConfig{
			RateLimit: RateLimiterConfig{
				GlobalRPS:          10000,
				GlobalBurst:        20000,
				GlobalWindow:       time.Minute,
				UserRPS:            1000,
				UserBurst:          2000,
				UserWindow:         time.Minute,
				IPRPS:              500,
				IPBurst:            1000,
				IPWindow:           time.Minute,
				ModuleDefaultRPS:   2000,
				ModuleDefaultBurst: 4000,
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
		}
	}
	return MiddlewareConfig{
		RateLimit: RateLimiterConfig{
			GlobalRPS:          common.GetIntOrDefault("RATE_LIMIT_GLOBAL_RPS", defaultConfig.RateLimit.GlobalRPS),
			GlobalBurst:        common.GetIntOrDefault("RATE_LIMIT_GLOBAL_BURST", defaultConfig.RateLimit.GlobalBurst),
			GlobalWindow:       common.ParseDuration(common.GetStringOrDefault("RATE_LIMIT_GLOBAL_WINDOW", "1m"), defaultConfig.RateLimit.GlobalWindow),
			UserRPS:            common.GetIntOrDefault("RATE_LIMIT_USER_RPS", defaultConfig.RateLimit.UserRPS),
			UserBurst:          common.GetIntOrDefault("RATE_LIMIT_USER_BURST", defaultConfig.RateLimit.UserBurst),
			UserWindow:         common.ParseDuration(common.GetStringOrDefault("RATE_LIMIT_USER_WINDOW", "1m"), defaultConfig.RateLimit.UserWindow),
			IPRPS:              common.GetIntOrDefault("RATE_LIMIT_IP_RPS", defaultConfig.RateLimit.IPRPS),
			IPBurst:            common.GetIntOrDefault("RATE_LIMIT_IP_BURST", defaultConfig.RateLimit.IPBurst),
			IPWindow:           common.ParseDuration(common.GetStringOrDefault("RATE_LIMIT_IP_WINDOW", "1m"), defaultConfig.RateLimit.IPWindow),
			ModuleDefaultRPS:   common.GetIntOrDefault("RATE_LIMIT_MODULE_DEFAULT_RPS", defaultConfig.RateLimit.ModuleDefaultRPS),
			ModuleDefaultBurst: common.GetIntOrDefault("RATE_LIMIT_MODULE_DEFAULT_BURST", defaultConfig.RateLimit.ModuleDefaultBurst),
			ModuleRulesJSON:    common.GetStringOrDefault("RATE_LIMIT_MODULE_RULES", ""),
			APIRulesJSON:       common.GetStringOrDefault("RATE_LIMIT_API_RULES", ""),
		},
		Timeout: TimeoutConfig{
			DefaultTimeout:   common.ParseDuration(common.GetStringOrDefault("DEFAULT_TIMEOUT", "30s"), defaultConfig.Timeout.DefaultTimeout),
			FallbackResponse: defaultConfig.Timeout.FallbackResponse,
		},
		CircuitBreaker: CircuitBreakerConfig{
			FailureThreshold:      common.GetIntOrDefault("CIRCUIT_BREAKER_FAILURE_THRESHOLD", defaultConfig.CircuitBreaker.FailureThreshold),
			SuccessThreshold:      common.GetIntOrDefault("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", defaultConfig.CircuitBreaker.SuccessThreshold),
			Timeout:               common.ParseDuration(common.GetStringOrDefault("CIRCUIT_BREAKER_TIMEOUT", "30s"), defaultConfig.CircuitBreaker.Timeout),
			OpenTimeout:           common.ParseDuration(common.GetStringOrDefault("CIRCUIT_BREAKER_OPEN_TIMEOUT", "30s"), defaultConfig.CircuitBreaker.OpenTimeout),
			MaxConcurrentRequests: common.GetIntOrDefault("CIRCUIT_BREAKER_MAX_CONCURRENT", defaultConfig.CircuitBreaker.MaxConcurrentRequests),
		},
		IPACL: IPACLConfig{
			Enable:     common.GetBoolOrDefault("HTTP_IP_ACL_ENABLE", false),
			AllowedIPs: common.GetStringOrDefault("HTTP_ALLOWED_IPS", ""),
			BlockedIPs: common.GetStringOrDefault("HTTP_BLOCKED_IPS", ""),
		},
		Degrade: DegradeConfig{
			Enable: common.GetBoolOrDefault("ENABLE_DEGRADE", defaultConfig.EnableCircuitBreaker),
		},
		EnableRateLimit:      common.GetBoolOrDefault("ENABLE_RATE_LIMIT", defaultConfig.EnableRateLimit),
		EnableTimeout:        common.GetBoolOrDefault("ENABLE_TIMEOUT", defaultConfig.EnableTimeout),
		EnableCircuitBreaker: common.GetBoolOrDefault("ENABLE_CIRCUIT_BREAKER", defaultConfig.EnableCircuitBreaker),
	}
}

// InitFromEnv configures the global lingllm cache from process environment.
func InitFromEnv() error {
	cfg := llmcache.Config{
		Type: strings.TrimSpace(common.GetEnv("CACHE_TYPE")),
		Local: llmcache.LocalConfig{
			MaxSize:           envInt("LOCAL_CACHE_MAX_SIZE", 1000),
			DefaultExpiration: envDuration("LOCAL_CACHE_DEFAULT_EXPIRATION", 5*time.Minute),
			CleanupInterval:   envDuration("LOCAL_CACHE_CLEANUP_INTERVAL", 10*time.Minute),
		},
		Redis: llmcache.RedisConfig{
			Addr:         common.GetEnv("REDIS_ADDR"),
			Password:     common.GetEnv("REDIS_PASSWORD"),
			DB:           envInt("REDIS_DB", 0),
			PoolSize:     envInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: envInt("REDIS_MIN_IDLE_CONNS", 5),
			DialTimeout:  envDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  envDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: envDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			IdleTimeout:  envDuration("REDIS_IDLE_TIMEOUT", 5*time.Minute),
		},
	}
	if cfg.Type == "" {
		cfg.Type = "local"
	}
	return llmcache.InitGlobalCache(cfg)
}

func envInt(key string, def int) int {
	s := strings.TrimSpace(common.GetEnv(key))
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func envDuration(key string, def time.Duration) time.Duration {
	s := strings.TrimSpace(common.GetEnv(key))
	if s == "" {
		return def
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}
