package cache

import (
	"context"
	"time"
)

// Cache 缓存接口
type Cache interface {
	// Get 获取缓存值
	Get(ctx context.Context, key string) (interface{}, bool)

	// Set 设置缓存值
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error

	// Delete 删除缓存
	Delete(ctx context.Context, key string) error

	// Exists 检查键是否存在
	Exists(ctx context.Context, key string) bool

	// Clear 清空所有缓存
	Clear(ctx context.Context) error

	// GetMulti 批量获取
	GetMulti(ctx context.Context, keys ...string) map[string]interface{}

	// SetMulti 批量设置
	SetMulti(ctx context.Context, data map[string]interface{}, expiration time.Duration) error

	// DeleteMulti 批量删除
	DeleteMulti(ctx context.Context, keys ...string) error

	// Increment 自增
	Increment(ctx context.Context, key string, value int64) (int64, error)

	// Decrement 自减
	Decrement(ctx context.Context, key string, value int64) (int64, error)

	// GetWithTTL 获取值并返回剩余TTL
	GetWithTTL(ctx context.Context, key string) (interface{}, time.Duration, bool)

	// Close 关闭缓存连接
	Close() error
}

// Config 缓存配置
type Config struct {
	// 缓存类型: "local" 或 "redis"
	Type string `json:"type" yaml:"type" env:"CACHE_TYPE" default:"local"`

	// Redis配置
	Redis RedisConfig `json:"redis" yaml:"redis"`

	// 本地缓存配置
	Local LocalConfig `json:"local" yaml:"local"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	// Redis地址
	Addr string `json:"addr" yaml:"addr" env:"REDIS_ADDR" default:"localhost:6379"`

	// Redis密码
	Password string `json:"password" yaml:"password" env:"REDIS_PASSWORD"`

	// Redis数据库
	DB int `json:"db" yaml:"db" env:"REDIS_DB" default:"0"`

	// 连接池大小
	PoolSize int `json:"pool_size" yaml:"pool_size" env:"REDIS_POOL_SIZE" default:"10"`

	// 最小空闲连接数
	MinIdleConns int `json:"min_idle_conns" yaml:"min_idle_conns" env:"REDIS_MIN_IDLE_CONNS" default:"5"`

	// 连接超时时间
	DialTimeout time.Duration `json:"dial_timeout" yaml:"dial_timeout" env:"REDIS_DIAL_TIMEOUT" default:"5s"`

	// 读取超时时间
	ReadTimeout time.Duration `json:"read_timeout" yaml:"read_timeout" env:"REDIS_READ_TIMEOUT" default:"3s"`

	// 写入超时时间
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" env:"REDIS_WRITE_TIMEOUT" default:"3s"`

	// 连接最大空闲时间
	IdleTimeout time.Duration `json:"idle_timeout" yaml:"idle_timeout" env:"REDIS_IDLE_TIMEOUT" default:"5m"`
}

// LocalConfig 本地缓存配置
type LocalConfig struct {
	// 最大缓存项数
	MaxSize int `json:"max_size" yaml:"max_size" env:"LOCAL_CACHE_MAX_SIZE" default:"1000"`

	// 默认过期时间
	DefaultExpiration time.Duration `json:"default_expiration" yaml:"default_expiration" env:"LOCAL_CACHE_DEFAULT_EXPIRATION" default:"5m"`

	// 清理间隔
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
