package middleware

import (
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MiddlewareManager 中间件管理器
type MiddlewareManager struct {
	config            config.MiddlewareConfig
	rateLimiter       *RateLimiter
	timeoutCircuitMgr *TimeoutCircuitManager
}

// NewMiddlewareManager 创建中间件管理器
func NewMiddlewareManager(cfg config.MiddlewareConfig) *MiddlewareManager {
	mgr := &MiddlewareManager{
		config: cfg,
	}

	// 初始化限流器
	if cfg.EnableRateLimit {
		rateLimitConfig := RateLimiterConfig{
			GlobalRPS:      cfg.RateLimit.GlobalRPS,
			GlobalBurst:    cfg.RateLimit.GlobalBurst,
			GlobalWindow:   cfg.RateLimit.GlobalWindow,
			UserRPS:        cfg.RateLimit.UserRPS,
			UserBurst:      cfg.RateLimit.UserBurst,
			UserWindow:     cfg.RateLimit.UserWindow,
			IPRPS:          cfg.RateLimit.IPRPS,
			IPBurst:        cfg.RateLimit.IPBurst,
			IPWindow:       cfg.RateLimit.IPWindow,
			EndpointLimits: getDefaultEndpointLimits(),
		}
		mgr.rateLimiter = NewRateLimiter(rateLimitConfig)
		logger.Info("Rate limiter initialized",
			zap.Int("globalRPS", cfg.RateLimit.GlobalRPS),
			zap.Int("userRPS", cfg.RateLimit.UserRPS),
			zap.Int("ipRPS", cfg.RateLimit.IPRPS))
	}

	// 初始化超时熔断管理器
	if cfg.EnableTimeout || cfg.EnableCircuitBreaker {
		timeoutConfig := TimeoutConfig{
			DefaultTimeout:   cfg.Timeout.DefaultTimeout,
			EndpointTimeouts: getDefaultEndpointTimeouts(),
			FallbackResponse: cfg.Timeout.FallbackResponse,
		}
		circuitBreakerConfig := CircuitBreakerConfig{
			FailureThreshold:      cfg.CircuitBreaker.FailureThreshold,
			SuccessThreshold:      cfg.CircuitBreaker.SuccessThreshold,
			Timeout:               cfg.CircuitBreaker.Timeout,
			OpenTimeout:           cfg.CircuitBreaker.OpenTimeout,
			MaxConcurrentRequests: cfg.CircuitBreaker.MaxConcurrentRequests,
		}
		mgr.timeoutCircuitMgr = NewTimeoutCircuitManager(timeoutConfig, circuitBreakerConfig)
		logger.Info("Timeout and circuit breaker manager initialized",
			zap.Duration("defaultTimeout", cfg.Timeout.DefaultTimeout),
			zap.Int("failureThreshold", cfg.CircuitBreaker.FailureThreshold))
	}

	return mgr
}

// getDefaultEndpointLimits 获取默认的接口限流配置
func getDefaultEndpointLimits() map[string]EndpointLimit {
	return map[string]EndpointLimit{
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
	}
}

// getDefaultEndpointTimeouts 获取默认的接口超时配置
func getDefaultEndpointTimeouts() map[string]time.Duration {
	return map[string]time.Duration{
		// 登录接口：10秒超时
		"/api/auth/login/password": 10 * time.Second,
		"/api/auth/login/email":    10 * time.Second,

		// 文件上传：5分钟超时
		"/api/upload": 5 * time.Minute,

		// AI相关接口：60秒超时
		"/api/assistant/chat": 60 * time.Second,
		"/api/chat/send":      60 * time.Second,

		// 工作流执行：10分钟超时
		"/api/workflow/execute": 10 * time.Minute,

		// 语音相关：30秒超时
		"/api/voice/training/create": 30 * time.Second,
		"/api/voice/synthesis":       30 * time.Second,
	}
}

// ApplyMiddlewares 应用中间件到路由组
func (mgr *MiddlewareManager) ApplyMiddlewares(r *gin.RouterGroup) {
	logger.Info("Applying middlewares",
		zap.Bool("rateLimit", mgr.config.EnableRateLimit),
		zap.Bool("timeout", mgr.config.EnableTimeout),
		zap.Bool("circuitBreaker", mgr.config.EnableCircuitBreaker),
		zap.Bool("operationLog", mgr.config.EnableOperationLog))

	// 1. 限流中间件（最先执行）
	if mgr.config.EnableRateLimit && mgr.rateLimiter != nil {
		r.Use(RateLimitMiddleware())
		logger.Info("Rate limit middleware applied")
	}

	// 2. 超时和熔断中间件
	if (mgr.config.EnableTimeout || mgr.config.EnableCircuitBreaker) && mgr.timeoutCircuitMgr != nil {
		r.Use(CombinedTimeoutCircuitMiddleware())
		logger.Info("Timeout and circuit breaker middleware applied")
	}

	// 3. 操作日志中间件（最后执行，记录所有操作）
	if mgr.config.EnableOperationLog {
		r.Use(OperationLogMiddleware())
		logger.Info("Operation log middleware applied")
	}
}

// GetStats 获取所有中间件统计信息
func (mgr *MiddlewareManager) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 限流统计
	if mgr.rateLimiter != nil {
		stats["rate_limiter"] = mgr.rateLimiter.GetStats()
	}

	// 熔断器统计
	if mgr.timeoutCircuitMgr != nil {
		stats["circuit_breakers"] = GetCircuitBreakerStats()
	}

	return stats
}

// UpdateRateLimitConfig 动态更新限流配置
func (mgr *MiddlewareManager) UpdateRateLimitConfig(cfg config.RateLimiterConfig) {
	if mgr.rateLimiter != nil {
		mgr.config.RateLimit = cfg
		// 创建新的限流器实例
		rateLimitConfig := RateLimiterConfig{
			GlobalRPS:      cfg.GlobalRPS,
			GlobalBurst:    cfg.GlobalBurst,
			GlobalWindow:   cfg.GlobalWindow,
			UserRPS:        cfg.UserRPS,
			UserBurst:      cfg.UserBurst,
			UserWindow:     cfg.UserWindow,
			IPRPS:          cfg.IPRPS,
			IPBurst:        cfg.IPBurst,
			IPWindow:       cfg.IPWindow,
			EndpointLimits: getDefaultEndpointLimits(),
		}
		mgr.rateLimiter = NewRateLimiter(rateLimitConfig)
		logger.Info("Rate limit configuration updated",
			zap.Int("globalRPS", cfg.GlobalRPS),
			zap.Int("userRPS", cfg.UserRPS))
	}
}

// UpdateTimeoutConfig 动态更新超时配置
func (mgr *MiddlewareManager) UpdateTimeoutConfig(cfg config.TimeoutConfig) {
	if mgr.timeoutCircuitMgr != nil {
		mgr.config.Timeout = cfg
		timeoutConfig := TimeoutConfig{
			DefaultTimeout:   cfg.DefaultTimeout,
			EndpointTimeouts: getDefaultEndpointTimeouts(),
			FallbackResponse: cfg.FallbackResponse,
		}
		mgr.timeoutCircuitMgr.timeoutConfig = timeoutConfig
		logger.Info("Timeout configuration updated",
			zap.Duration("defaultTimeout", cfg.DefaultTimeout))
	}
}

// UpdateCircuitBreakerConfig 动态更新熔断器配置
func (mgr *MiddlewareManager) UpdateCircuitBreakerConfig(cfg config.CircuitBreakerConfig) {
	if mgr.timeoutCircuitMgr != nil {
		mgr.config.CircuitBreaker = cfg
		circuitBreakerConfig := CircuitBreakerConfig{
			FailureThreshold:      cfg.FailureThreshold,
			SuccessThreshold:      cfg.SuccessThreshold,
			Timeout:               cfg.Timeout,
			OpenTimeout:           cfg.OpenTimeout,
			MaxConcurrentRequests: cfg.MaxConcurrentRequests,
		}
		mgr.timeoutCircuitMgr.circuitBreakerConfig = circuitBreakerConfig
		// 清空现有熔断器，让它们重新创建
		mgr.timeoutCircuitMgr.circuitBreakers = sync.Map{}
		logger.Info("Circuit breaker configuration updated",
			zap.Int("failureThreshold", cfg.FailureThreshold))
	}
}

// 全局中间件管理器
var globalMiddlewareManager *MiddlewareManager

// InitGlobalMiddlewareManager 初始化全局中间件管理器
func InitGlobalMiddlewareManager(cfg config.MiddlewareConfig) {
	globalMiddlewareManager = NewMiddlewareManager(cfg)
	logger.Info("Global middleware manager initialized")
}

// GetGlobalMiddlewareManager 获取全局中间件管理器
func GetGlobalMiddlewareManager() *MiddlewareManager {
	if globalMiddlewareManager == nil {
		// 使用默认配置创建
		if config.GlobalConfig != nil {
			globalMiddlewareManager = NewMiddlewareManager(config.GlobalConfig.Middleware)
		} else {
			// 如果全局配置未初始化，使用默认值
			defaultConfig := config.MiddlewareConfig{
				RateLimit: config.RateLimiterConfig{
					GlobalRPS:    1000,
					GlobalBurst:  2000,
					GlobalWindow: time.Minute,
					UserRPS:      100,
					UserBurst:    200,
					UserWindow:   time.Minute,
					IPRPS:        50,
					IPBurst:      100,
					IPWindow:     time.Minute,
				},
				Timeout: config.TimeoutConfig{
					DefaultTimeout: 30 * time.Second,
					FallbackResponse: map[string]interface{}{
						"error":   "service_unavailable",
						"message": "服务暂时不可用，请稍后重试",
						"code":    503,
					},
				},
				CircuitBreaker: config.CircuitBreakerConfig{
					FailureThreshold:      5,
					SuccessThreshold:      3,
					Timeout:               30 * time.Second,
					OpenTimeout:           60 * time.Second,
					MaxConcurrentRequests: 100,
				},
				EnableRateLimit:      true,
				EnableTimeout:        true,
				EnableCircuitBreaker: true,
				EnableOperationLog:   true,
			}
			globalMiddlewareManager = NewMiddlewareManager(defaultConfig)
		}
		logger.Info("Global middleware manager created with default config")
	}
	return globalMiddlewareManager
}

// ApplyGlobalMiddlewares 应用全局中间件
func ApplyGlobalMiddlewares(r *gin.RouterGroup) {
	mgr := GetGlobalMiddlewareManager()
	mgr.ApplyMiddlewares(r)
}

// GetGlobalMiddlewareStats 获取全局中间件统计信息
func GetGlobalMiddlewareStats() map[string]interface{} {
	mgr := GetGlobalMiddlewareManager()
	return mgr.GetStats()
}
