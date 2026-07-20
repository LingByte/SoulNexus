package middleware

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	// 默认超时时间
	DefaultTimeout time.Duration
	// 接口级别超时配置
	EndpointTimeouts map[string]time.Duration
	// 超时后的降级响应
	FallbackResponse interface{}
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	// 失败阈值
	FailureThreshold int
	// 成功阈值（半开状态下）
	SuccessThreshold int
	// 超时时间
	Timeout time.Duration
	// 熔断器打开后的等待时间
	OpenTimeout time.Duration
	// 最大并发请求数
	MaxConcurrentRequests int
}

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateHalfOpen
	StateOpen
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	config          CircuitBreakerConfig
	state           CircuitBreakerState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	nextAttemptTime time.Time
	concurrentCount int
	mu              sync.RWMutex
}

// NewCircuitBreaker 创建新的熔断器
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}
}

// DefaultCircuitBreakerConfig 默认熔断器配置
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:      5, // 5次失败后打开熔断器
		SuccessThreshold:      3, // 3次成功后关闭熔断器
		Timeout:               30 * time.Second,
		OpenTimeout:           60 * time.Second, // 熔断器打开后60秒尝试半开
		MaxConcurrentRequests: 100,
	}
}

// Allow 检查是否允许请求
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		// 检查并发数
		if cb.concurrentCount >= cb.config.MaxConcurrentRequests {
			return false
		}
		cb.concurrentCount++
		return true

	case StateOpen:
		// 检查是否可以尝试半开
		if now.After(cb.nextAttemptTime) {
			cb.state = StateHalfOpen
			cb.successCount = 0
			cb.concurrentCount++
			return true
		}
		return false

	case StateHalfOpen:
		// 半开状态下限制并发
		if cb.concurrentCount >= 1 {
			return false
		}
		cb.concurrentCount++
		return true

	default:
		return false
	}
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.concurrentCount--
	if cb.concurrentCount < 0 {
		cb.concurrentCount = 0
	}

	switch cb.state {
	case StateClosed:
		cb.failureCount = 0

	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failureCount = 0
		}
	}
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.concurrentCount--
	if cb.concurrentCount < 0 {
		cb.concurrentCount = 0
	}

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = StateOpen
			cb.nextAttemptTime = time.Now().Add(cb.config.OpenTimeout)
		}

	case StateHalfOpen:
		cb.state = StateOpen
		cb.nextAttemptTime = time.Now().Add(cb.config.OpenTimeout)
	}
}

// GetState 获取熔断器状态
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetStats 获取熔断器统计信息
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"state":            cb.state,
		"failure_count":    cb.failureCount,
		"success_count":    cb.successCount,
		"concurrent_count": cb.concurrentCount,
		"last_failure":     cb.lastFailureTime,
		"next_attempt":     cb.nextAttemptTime,
	}
}

// TimeoutCircuitManager 超时和熔断管理器
type TimeoutCircuitManager struct {
	timeoutConfig        TimeoutConfig
	circuitBreakerConfig CircuitBreakerConfig
	circuitBreakers      sync.Map // map[string]*CircuitBreaker
	mu                   sync.RWMutex
	enableTimeout        bool
	enableCircuitBreaker bool
}

// NewTimeoutCircuitManager 创建超时和熔断管理器
func NewTimeoutCircuitManager(timeoutConfig TimeoutConfig, cbConfig CircuitBreakerConfig) *TimeoutCircuitManager {
	return &TimeoutCircuitManager{
		timeoutConfig:        timeoutConfig,
		circuitBreakerConfig: cbConfig,
		enableTimeout:        true,
		enableCircuitBreaker: true,
	}
}

// InitTimeoutCircuitManager configures the global timeout/circuit manager from app config.
// Safe to call once at startup; subsequent calls are no-ops (sync.Once).
func InitTimeoutCircuitManager(timeoutCfg TimeoutConfig, cbCfg CircuitBreakerConfig, enableTimeout, enableCircuitBreaker bool) {
	timeoutCircuitOnce.Do(func() {
		globalTimeoutCircuitManager = NewTimeoutCircuitManager(timeoutCfg, cbCfg)
		globalTimeoutCircuitManager.enableTimeout = enableTimeout
		globalTimeoutCircuitManager.enableCircuitBreaker = enableCircuitBreaker
	})
}

// DefaultTimeoutConfig 默认超时配置
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		DefaultTimeout: 30 * time.Second,
		EndpointTimeouts: map[string]time.Duration{
			// 登录接口：10秒超时
			"/api/auth/login/password": 10 * time.Second,
			"/api/auth/login/email":    10 * time.Second,

			// 文件上传：5分钟超时
			"/api/upload": 5 * time.Minute,

			// AI相关接口：60秒超时
			"/api/assistant/chat": 60 * time.Second,
			"/api/chat/send":      60 * time.Second,

			// 语音会话调试（WebSocket / WebRTC / 文本 LLM）
			"/api/lingecho/voice-session/v1/": 3 * time.Minute,

			// 语音接口：10分钟超时（支持长时间语音会话）
			"/api/voice/lingecho/v1/": 10 * time.Minute,

			// 工作流执行：10分钟超时
			"/api/workflow/execute": 10 * time.Minute,

			// 语音相关：30秒超时
			"/api/voice/training/create": 30 * time.Second,
			"/api/voice/synthesis":       30 * time.Second,
		},
		FallbackResponse: map[string]interface{}{
			"error":   "service_unavailable",
			"message": "服务暂时不可用，请稍后重试",
			"code":    503,
		},
	}
}

// getCircuitBreaker 获取熔断器
func (tcm *TimeoutCircuitManager) getCircuitBreaker(endpoint string) *CircuitBreaker {
	if cb, ok := tcm.circuitBreakers.Load(endpoint); ok {
		return cb.(*CircuitBreaker)
	}

	cb := NewCircuitBreaker(tcm.circuitBreakerConfig)
	tcm.circuitBreakers.Store(endpoint, cb)
	return cb
}

// getTimeout 获取接口超时时间
func (tcm *TimeoutCircuitManager) getTimeout(endpoint string) time.Duration {
	if timeout, exists := tcm.timeoutConfig.EndpointTimeouts[endpoint]; exists {
		return timeout
	}
	for prefix, timeout := range tcm.timeoutConfig.EndpointTimeouts {
		if strings.HasSuffix(prefix, "/") && strings.HasPrefix(endpoint, prefix) {
			return timeout
		}
	}
	return tcm.timeoutConfig.DefaultTimeout
}

// 全局超时熔断管理器
var globalTimeoutCircuitManager *TimeoutCircuitManager
var timeoutCircuitOnce sync.Once

// GetTimeoutCircuitManager 获取全局超时熔断管理器
func GetTimeoutCircuitManager() *TimeoutCircuitManager {
	timeoutCircuitOnce.Do(func() {
		globalTimeoutCircuitManager = NewTimeoutCircuitManager(
			DefaultTimeoutConfig(),
			DefaultCircuitBreakerConfig(),
		)
	})
	return globalTimeoutCircuitManager
}

func isLongLivedEndpoint(endpoint string) bool {
	if strings.Contains(endpoint, "/ws") || strings.HasSuffix(endpoint, "/ws") {
		return true
	}
	if strings.Contains(endpoint, "/incoming/stream") || strings.HasSuffix(endpoint, "/stream") {
		return true
	}
	switch endpoint {
	case "/api/voice/lingecho/v1/", "/api/voice/lingecho/v2/", "/api/voice/ws":
		return true
	}
	return strings.HasPrefix(endpoint, "/api/lingecho/voice-session/v1/ws")
}

// TimeoutMiddleware 超时中间件
func TimeoutMiddleware() gin.HandlerFunc {
	manager := GetTimeoutCircuitManager()

	return func(c *gin.Context) {
		endpoint := c.Request.URL.Path
		if isLongLivedEndpoint(endpoint) {
			c.Next()
			return
		}
		timeout := manager.getTimeout(endpoint)

		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 替换请求上下文
		c.Request = c.Request.WithContext(ctx)

		// 使用通道来检测请求是否完成
		done := make(chan struct{})
		go func() {
			defer close(done)
			c.Next()
		}()

		select {
		case <-done:
			// 请求正常完成
			return
		case <-ctx.Done():
			// 请求超时
			logger.WarnCtx(c.Request.Context(), "Request timeout",
				zap.String("endpoint", endpoint),
				zap.Duration("timeout", timeout),
				zap.String("method", c.Request.Method))

			// 返回超时错误
			if !c.Writer.Written() {
				c.JSON(http.StatusRequestTimeout, map[string]interface{}{
					"error":   "request_timeout",
					"message": fmt.Sprintf("请求超时，超过 %v", timeout),
					"timeout": timeout.String(),
				})
			}
			c.Abort()
			return
		}
	}
}

// CircuitBreakerMiddleware 熔断器中间件
func CircuitBreakerMiddleware() gin.HandlerFunc {
	manager := GetTimeoutCircuitManager()

	return func(c *gin.Context) {
		endpoint := c.Request.URL.Path
		cb := manager.getCircuitBreaker(endpoint)

		// 检查熔断器是否允许请求
		if !cb.Allow() {
			state := cb.GetState()
			logger.WarnCtx(c.Request.Context(), "Circuit breaker blocked request",
				zap.String("endpoint", endpoint),
				zap.Int("state", int(state)))

			// 返回服务不可用错误
			c.JSON(http.StatusServiceUnavailable, manager.timeoutConfig.FallbackResponse)
			c.Abort()
			return
		}

		// 记录请求开始时间
		startTime := time.Now()

		// 执行请求
		c.Next()

		// 根据响应状态记录成功或失败
		duration := time.Since(startTime)
		status := c.Writer.Status()

		if status >= 200 && status < 400 {
			// 成功响应
			cb.RecordSuccess()
			logger.DebugCtx(c.Request.Context(), "Circuit breaker recorded success",
				zap.String("endpoint", endpoint),
				zap.Int("status", status),
				zap.Duration("duration", duration))
		} else if status >= 500 {
			// 服务器错误，记录失败
			cb.RecordFailure()
			logger.WarnCtx(c.Request.Context(), "Circuit breaker recorded failure",
				zap.String("endpoint", endpoint),
				zap.Int("status", status),
				zap.Duration("duration", duration))
		}
		// 4xx错误不记录为熔断器失败，因为这通常是客户端错误
	}
}

// CombinedTimeoutCircuitMiddleware 组合超时和熔断中间件。
// Respects enableTimeout / enableCircuitBreaker set via InitTimeoutCircuitManager.
func CombinedTimeoutCircuitMiddleware() gin.HandlerFunc {
	manager := GetTimeoutCircuitManager()

	return func(c *gin.Context) {
		if !manager.enableTimeout && !manager.enableCircuitBreaker {
			c.Next()
			return
		}

		endpoint := c.Request.URL.Path

		// WebSocket / SSE 长连接跳过超时与熔断（30s 超时会中断 stream 并触发 Flush panic）。
		if isLongLivedEndpoint(endpoint) {
			c.Next()
			return
		}

		var cb *CircuitBreaker
		if manager.enableCircuitBreaker {
			cb = manager.getCircuitBreaker(endpoint)
			if !cb.Allow() {
				state := cb.GetState()
				logger.WarnCtx(c.Request.Context(), "Circuit breaker blocked request",
					zap.String("endpoint", endpoint),
					zap.Int("state", int(state)))

				c.JSON(http.StatusServiceUnavailable, manager.timeoutConfig.FallbackResponse)
				c.Abort()
				return
			}
		}

		if !manager.enableTimeout {
			c.Next()
			if cb != nil {
				status := c.Writer.Status()
				if status >= 200 && status < 400 {
					cb.RecordSuccess()
				} else if status >= 500 {
					cb.RecordFailure()
				}
			}
			return
		}

		timeout := manager.getTimeout(endpoint)
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		done := make(chan struct{})
		var requestPanic interface{}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					requestPanic = r
				}
				close(done)
			}()
			c.Next()
		}()

		select {
		case <-done:
			status := c.Writer.Status()

			if requestPanic != nil {
				logger.ErrorCtx(c.Request.Context(), "Request panic",
					zap.String("endpoint", endpoint),
					zap.Any("panic", requestPanic),
					zap.ByteString("stack", debug.Stack()))
				if cb != nil {
					cb.RecordFailure()
				}
				if !c.Writer.Written() {
					c.JSON(http.StatusInternalServerError, map[string]interface{}{
						"error":   "internal_server_error",
						"message": "服务器内部错误",
					})
				}
				return
			}

			if cb != nil {
				if status >= 200 && status < 400 {
					cb.RecordSuccess()
				} else if status >= 500 {
					cb.RecordFailure()
				}
			}
		case <-ctx.Done():
			if cb != nil {
				cb.RecordFailure()
			}
			logger.WarnCtx(c.Request.Context(), "Request timeout",
				zap.String("endpoint", endpoint),
				zap.Duration("timeout", timeout))

			if !c.Writer.Written() {
				c.JSON(http.StatusRequestTimeout, map[string]interface{}{
					"error":   "request_timeout",
					"message": fmt.Sprintf("请求超时，超过 %v", timeout),
					"timeout": timeout.String(),
				})
			}
			c.Abort()
		}
	}
}

// GetCircuitBreakerStats 获取所有熔断器统计信息
func GetCircuitBreakerStats() map[string]interface{} {
	manager := GetTimeoutCircuitManager()
	stats := make(map[string]interface{})

	manager.circuitBreakers.Range(func(key, value interface{}) bool {
		endpoint := key.(string)
		cb := value.(*CircuitBreaker)
		stats[endpoint] = cb.GetStats()
		return true
	})

	return stats
}
