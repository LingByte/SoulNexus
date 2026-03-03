package cache

import (
	"context"
	"time"
)

// Cache defines the cache interface
type Cache interface {
	// Get retrieves a cached value
	Get(ctx context.Context, key string) (interface{}, bool)

	// Set stores a value in cache
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// Delete removes a cached value
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists
	Exists(ctx context.Context, key string) bool

	// Clear removes all cached values
	Clear(ctx context.Context) error

	// GetMulti retrieves multiple values at once
	GetMulti(ctx context.Context, keys ...string) map[string]interface{}

	// SetMulti stores multiple values at once
	SetMulti(ctx context.Context, data map[string]interface{}, expiration time.Duration) error

	// DeleteMulti removes multiple values at once
	DeleteMulti(ctx context.Context, keys ...string) error

	// Increment increments a numeric value
	Increment(ctx context.Context, key string, value int64) (int64, error)

	// Decrement decrements a numeric value
	Decrement(ctx context.Context, key string, value int64) (int64, error)

	// GetWithTTL retrieves value with remaining TTL
	GetWithTTL(ctx context.Context, key string) (interface{}, time.Duration, bool)

	// Close closes the cache connection
	Close() error
}

// Config defines cache configuration
type Config struct {
	// Cache type: "local" or "redis"
	Type string `json:"type" yaml:"type" env:"CACHE_TYPE" default:"local"`

	// Redis configuration
	Redis RedisConfig `json:"redis" yaml:"redis"`

	// Local cache configuration
	Local LocalConfig `json:"local" yaml:"local"`
}

// RedisConfig defines Redis configuration
type RedisConfig struct {
	// Redis server address
	Addr string `json:"addr" yaml:"addr" env:"REDIS_ADDR" default:"localhost:6379"`

	// Redis password
	Password string `json:"password" yaml:"password" env:"REDIS_PASSWORD"`

	// Redis database number
	DB int `json:"db" yaml:"db" env:"REDIS_DB" default:"0"`

	// Connection pool size
	PoolSize int `json:"pool_size" yaml:"pool_size" env:"REDIS_POOL_SIZE" default:"10"`

	// Minimum idle connections
	MinIdleConns int `json:"min_idle_conns" yaml:"min_idle_conns" env:"REDIS_MIN_IDLE_CONNS" default:"5"`

	// Connection dial timeout
	DialTimeout time.Duration `json:"dial_timeout" yaml:"dial_timeout" env:"REDIS_DIAL_TIMEOUT" default:"5s"`

	// Read timeout
	ReadTimeout time.Duration `json:"read_timeout" yaml:"read_timeout" env:"REDIS_READ_TIMEOUT" default:"3s"`

	// Write timeout
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" env:"REDIS_WRITE_TIMEOUT" default:"3s"`

	// Connection idle timeout
	IdleTimeout time.Duration `json:"idle_timeout" yaml:"idle_timeout" env:"REDIS_IDLE_TIMEOUT" default:"5m"`
}

// LocalConfig defines local cache configuration
type LocalConfig struct {
	// Maximum number of cache items
	MaxSize int `json:"max_size" yaml:"max_size" env:"LOCAL_CACHE_MAX_SIZE" default:"1000"`

	// Default expiration time
	DefaultExpiration time.Duration `json:"default_expiration" yaml:"default_expiration" env:"LOCAL_CACHE_DEFAULT_EXPIRATION" default:"5m"`

	// Cleanup interval
	CleanupInterval time.Duration `json:"cleanup_interval" yaml:"cleanup_interval" env:"LOCAL_CACHE_CLEANUP_INTERVAL" default:"10m"`
}

// Options 缓存选项
type Options struct {
	// 过期时间
	Expiration time.Duration

	// 是否使用本地缓存作为一级缓存
	UseLocalCache bool

	// 本地缓存过期时间（通常比分布式缓存短）
	LocalExpiration time.Duration
}

// DefaultOptions 默认选项
func DefaultOptions() *Options {
	return &Options{
		Expiration:      5 * time.Minute,
		UseLocalCache:   true,
		LocalExpiration: 1 * time.Minute,
	}
}
