package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RateLimiterConfig 限流配置
type RateLimiterConfig struct {
	// 全局限流配置
	GlobalRPS    int           // 全局每秒请求数
	GlobalBurst  int           // 全局突发请求数
	GlobalWindow time.Duration // 全局时间窗口

	// 用户限流配置
	UserRPS    int           // 用户每秒请求数
	UserBurst  int           // 用户突发请求数
	UserWindow time.Duration // 用户时间窗口

	// IP限流配置
	IPRPS    int           // IP每秒请求数
	IPBurst  int           // IP突发请求数
	IPWindow time.Duration // IP时间窗口

	// 接口级别限流配置
	EndpointLimits map[string]EndpointLimit // 特定接口的限流配置
}

// EndpointLimit 接口级别限流配置
type EndpointLimit struct {
	RPS    int           // 每秒请求数
	Burst  int           // 突发请求数
	Window time.Duration // 时间窗口
}

// TokenBucket 令牌桶实现
type TokenBucket struct {
	capacity     int           // 桶容量
	tokens       int           // 当前令牌数
	refillRate   int           // 每秒补充令牌数
	lastRefill   time.Time     // 上次补充时间
	mu           sync.Mutex    // 互斥锁
	window       time.Duration // 时间窗口
	requestCount int           // 窗口内请求计数
	windowStart  time.Time     // 窗口开始时间
}

// NewTokenBucket 创建新的令牌桶
func NewTokenBucket(capacity, refillRate int, window time.Duration) *TokenBucket {
	return &TokenBucket{
		capacity:    capacity,
		tokens:      capacity,
		refillRate:  refillRate,
		lastRefill:  time.Now(),
		window:      window,
		windowStart: time.Now(),
	}
}

// Allow 检查是否允许请求
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()

	// 检查时间窗口是否需要重置
	if now.Sub(tb.windowStart) >= tb.window {
		tb.requestCount = 0
		tb.windowStart = now
	}

	// 补充令牌
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int(elapsed.Seconds()) * tb.refillRate
	if tokensToAdd > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}

	// 检查是否有可用令牌
	if tb.tokens > 0 {
		tb.tokens--
		tb.requestCount++
		return true
	}

	return false
}

// GetStats 获取令牌桶统计信息
func (tb *TokenBucket) GetStats() (tokens, requestCount int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.tokens, tb.requestCount
}

// RateLimiter 限流器
type RateLimiter struct {
	config          RateLimiterConfig
	globalBucket    *TokenBucket
	userBuckets     sync.Map // map[uint]*TokenBucket
	ipBuckets       sync.Map // map[string]*TokenBucket
	endpointBuckets sync.Map // map[string]*TokenBucket
	mu              sync.RWMutex
}

// NewRateLimiter 创建新的限流器
func NewRateLimiter(config RateLimiterConfig) *RateLimiter {
	rl := &RateLimiter{
		config: config,
	}

	// 创建全局令牌桶
	if config.GlobalRPS > 0 {
		rl.globalBucket = NewTokenBucket(config.GlobalBurst, config.GlobalRPS, config.GlobalWindow)
	}

	return rl
}

// DefaultRateLimiterConfig 默认限流配置
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		// 全局限流：每秒1000个请求，突发2000个
		GlobalRPS:    1000,
		GlobalBurst:  2000,
		GlobalWindow: time.Minute,

		// 用户限流：每秒100个请求，突发200个
		UserRPS:    100,
		UserBurst:  200,
		UserWindow: time.Minute,

		// IP限流：每秒50个请求，突发100个
		IPRPS:    50,
		IPBurst:  100,
		IPWindow: time.Minute,

		// 特定接口限流
		EndpointLimits: map[string]EndpointLimit{
			// 登录接口：每分钟5次
			"/api/auth/login/password": {
				RPS:    1,
				Burst:  5,
				Window: time.Minute,
			},
			"/api/auth/login/email": {
				RPS:    1,
				Burst:  5,
				Window: time.Minute,
			},
			// 注册接口：每小时3次
			"/api/auth/register": {
				RPS:    1,
				Burst:  3,
				Window: time.Hour,
			},
			// 发送验证码：每分钟3次
			"/api/auth/send/email": {
				RPS:    1,
				Burst:  3,
				Window: time.Minute,
			},
			// 文件上传：每分钟10次
			"/api/upload": {
				RPS:    5,
				Burst:  10,
				Window: time.Minute,
			},
		},
	}
}

// getUserBucket 获取用户令牌桶
func (rl *RateLimiter) getUserBucket(userID uint) *TokenBucket {
	if bucket, ok := rl.userBuckets.Load(userID); ok {
		return bucket.(*TokenBucket)
	}

	bucket := NewTokenBucket(rl.config.UserBurst, rl.config.UserRPS, rl.config.UserWindow)
	rl.userBuckets.Store(userID, bucket)
	return bucket
}

// getIPBucket 获取IP令牌桶
func (rl *RateLimiter) getIPBucket(ip string) *TokenBucket {
	if bucket, ok := rl.ipBuckets.Load(ip); ok {
		return bucket.(*TokenBucket)
	}

	bucket := NewTokenBucket(rl.config.IPBurst, rl.config.IPRPS, rl.config.IPWindow)
	rl.ipBuckets.Store(ip, bucket)
	return bucket
}

// getEndpointBucket 获取接口令牌桶
func (rl *RateLimiter) getEndpointBucket(endpoint string, userID uint) *TokenBucket {
	key := fmt.Sprintf("%s:%d", endpoint, userID)
	if bucket, ok := rl.endpointBuckets.Load(key); ok {
		return bucket.(*TokenBucket)
	}

	// 检查是否有特定接口的限流配置
	if limit, exists := rl.config.EndpointLimits[endpoint]; exists {
		bucket := NewTokenBucket(limit.Burst, limit.RPS, limit.Window)
		rl.endpointBuckets.Store(key, bucket)
		return bucket
	}

	return nil
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(userID uint, ip, endpoint string) (bool, string) {
	// 1. 检查全局限流
	if rl.globalBucket != nil && !rl.globalBucket.Allow() {
		return false, "global_rate_limit_exceeded"
	}

	// 2. 检查IP限流
	if rl.config.IPRPS > 0 {
		ipBucket := rl.getIPBucket(ip)
		if !ipBucket.Allow() {
			return false, "ip_rate_limit_exceeded"
		}
	}

	// 3. 检查用户限流
	if userID > 0 && rl.config.UserRPS > 0 {
		userBucket := rl.getUserBucket(userID)
		if !userBucket.Allow() {
			return false, "user_rate_limit_exceeded"
		}
	}

	// 4. 检查接口级别限流
	if userID > 0 {
		endpointBucket := rl.getEndpointBucket(endpoint, userID)
		if endpointBucket != nil && !endpointBucket.Allow() {
			return false, "endpoint_rate_limit_exceeded"
		}
	}

	return true, ""
}

// GetStats 获取限流统计信息
func (rl *RateLimiter) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if rl.globalBucket != nil {
		tokens, requests := rl.globalBucket.GetStats()
		stats["global"] = map[string]int{
			"tokens":   tokens,
			"requests": requests,
		}
	}

	// 统计用户桶数量
	userCount := 0
	rl.userBuckets.Range(func(key, value interface{}) bool {
		userCount++
		return true
	})
	stats["user_buckets"] = userCount

	// 统计IP桶数量
	ipCount := 0
	rl.ipBuckets.Range(func(key, value interface{}) bool {
		ipCount++
		return true
	})
	stats["ip_buckets"] = ipCount

	return stats
}

// 全局限流器实例
var globalRateLimiter *RateLimiter
var rateLimiterOnce sync.Once

// GetRateLimiter 获取全局限流器实例
func GetRateLimiter() *RateLimiter {
	rateLimiterOnce.Do(func() {
		globalRateLimiter = NewRateLimiter(DefaultRateLimiterConfig())
	})
	return globalRateLimiter
}

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware() gin.HandlerFunc {
	limiter := GetRateLimiter()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		endpoint := c.Request.URL.Path

		var userID uint = 0

		// 尝试获取用户ID
		if user := models.CurrentUser(c); user != nil {
			userID = user.ID
		}

		// 检查限流
		allowed, reason := limiter.Allow(userID, ip, endpoint)
		if !allowed {
			logger.Warn("Rate limit exceeded",
				zap.Uint("userID", userID),
				zap.String("ip", ip),
				zap.String("endpoint", endpoint),
				zap.String("reason", reason))

			// 设置限流响应头
			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.config.UserRPS))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))

			// 返回限流错误
			c.JSON(http.StatusTooManyRequests, map[string]interface{}{
				"error":   "rate_limit_exceeded",
				"message": getRateLimitMessage(reason),
				"reason":  reason,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// getRateLimitMessage 获取限流错误消息
func getRateLimitMessage(reason string) string {
	messages := map[string]string{
		"global_rate_limit_exceeded":   "系统繁忙，请稍后再试",
		"ip_rate_limit_exceeded":       "您的IP请求过于频繁，请稍后再试",
		"user_rate_limit_exceeded":     "您的请求过于频繁，请稍后再试",
		"endpoint_rate_limit_exceeded": "该接口请求过于频繁，请稍后再试",
	}

	if msg, exists := messages[reason]; exists {
		return msg
	}
	return "请求过于频繁，请稍后再试"
}

// min 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
