package reconnect

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Strategy 重连策略
type Strategy interface {
	// NextDelay 获取下一次重连延迟
	NextDelay(attempt int, err error) time.Duration

	// ShouldRetry 判断是否应该重试
	ShouldRetry(attempt int, err error) bool
}

// ExponentialBackoffStrategy 指数退避策略
type ExponentialBackoffStrategy struct {
	InitialDelay     time.Duration
	MaxDelay         time.Duration
	MaxAttempts      int
	Multiplier       float64
	RateLimitDelay   time.Duration // 限流错误的特殊延迟
	IsRateLimitError func(error) bool
}

// NewExponentialBackoffStrategy 创建指数退避策略
func NewExponentialBackoffStrategy() *ExponentialBackoffStrategy {
	return &ExponentialBackoffStrategy{
		InitialDelay:   1 * time.Second,
		MaxDelay:       30 * time.Second,
		MaxAttempts:    10,
		Multiplier:     2.0,
		RateLimitDelay: 10 * time.Second, // 并发超限时使用更长的延迟
	}
}

// SetRateLimitChecker 设置限流错误检查函数
func (s *ExponentialBackoffStrategy) SetRateLimitChecker(checker func(error) bool) {
	s.IsRateLimitError = checker
}

// NextDelay 获取下一次重连延迟
func (s *ExponentialBackoffStrategy) NextDelay(attempt int, err error) time.Duration {
	// 如果是限流错误，使用更长的延迟
	if s.IsRateLimitError != nil && err != nil && s.IsRateLimitError(err) {
		// 限流错误使用固定延迟 + 指数退避
		baseDelay := s.RateLimitDelay
		backoffDelay := time.Duration(float64(s.InitialDelay) * pow(s.Multiplier, float64(attempt)))
		delay := baseDelay + backoffDelay
		if delay > s.MaxDelay*2 { // 限流错误允许更长的延迟
			delay = s.MaxDelay * 2
		}
		return delay
	}

	// 普通错误使用标准指数退避
	delay := time.Duration(float64(s.InitialDelay) * pow(s.Multiplier, float64(attempt)))
	if delay > s.MaxDelay {
		delay = s.MaxDelay
	}
	return delay
}

// ShouldRetry 判断是否应该重试
func (s *ExponentialBackoffStrategy) ShouldRetry(attempt int, err error) bool {
	return attempt < s.MaxAttempts
}

// pow 计算幂
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// Manager 重连管理器
type Manager struct {
	ctx          context.Context
	cancel       context.CancelFunc
	logger       *zap.Logger
	strategy     Strategy
	mu           sync.RWMutex
	attempt      int
	reconnecting bool
	lastError    error // 记录最后一次错误，用于判断是否需要特殊处理
	onReconnect  func() error
	onDisconnect func(error)
}

// NewManager 创建重连管理器
func NewManager(ctx context.Context, logger *zap.Logger, strategy Strategy) *Manager {
	ctx, cancel := context.WithCancel(ctx)
	return &Manager{
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger,
		strategy: strategy,
		attempt:  0,
	}
}

// SetReconnectCallback 设置重连回调
func (m *Manager) SetReconnectCallback(fn func() error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onReconnect = fn
}

// SetDisconnectCallback 设置断开连接回调
func (m *Manager) SetDisconnectCallback(fn func(error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onDisconnect = fn
}

// Start 启动重连管理器
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.reconnecting {
		return nil // 已经在重连中
	}

	m.reconnecting = true
	m.attempt = 0

	go m.reconnectLoop()
	return nil
}

// Stop 停止重连管理器
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cancel()
	m.reconnecting = false
	m.attempt = 0
}

// NotifyDisconnect 通知连接断开
func (m *Manager) NotifyDisconnect(err error) {
	m.mu.Lock()
	shouldStart := !m.reconnecting
	if shouldStart {
		m.lastError = err // 记录错误
	}
	m.mu.Unlock()

	if m.onDisconnect != nil {
		m.onDisconnect(err)
	}

	if shouldStart {
		m.Start()
	}
}

// IsReconnecting 检查是否正在重连
func (m *Manager) IsReconnecting() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.reconnecting
}

// reconnectLoop 重连循环
func (m *Manager) reconnectLoop() {
	for {
		select {
		case <-m.ctx.Done():
			m.mu.Lock()
			m.reconnecting = false
			m.mu.Unlock()
			return
		default:
		}

		m.mu.Lock()
		attempt := m.attempt
		onReconnect := m.onReconnect
		m.mu.Unlock()

		if onReconnect == nil {
			m.logger.Warn("重连回调未设置")
			m.mu.Lock()
			m.reconnecting = false
			m.mu.Unlock()
			return
		}

		// 尝试重连
		err := onReconnect()
		if err == nil {
			// 重连成功
			m.logger.Info("重连成功",
				zap.Int("attempt", attempt),
			)
			m.mu.Lock()
			m.reconnecting = false
			m.attempt = 0
			m.mu.Unlock()
			return
		}

		// 重连失败
		m.mu.Lock()
		m.attempt++
		nextAttempt := m.attempt
		shouldRetry := m.strategy.ShouldRetry(nextAttempt, err)
		m.mu.Unlock()

		if !shouldRetry {
			m.logger.Error("重连失败，达到最大重试次数",
				zap.Int("attempt", nextAttempt),
				zap.Error(err),
			)
			m.mu.Lock()
			m.reconnecting = false
			m.mu.Unlock()
			return
		}

		delay := m.strategy.NextDelay(nextAttempt, err)
		m.logger.Warn("重连失败，等待后重试",
			zap.Int("attempt", nextAttempt),
			zap.Duration("delay", delay),
			zap.Error(err),
		)

		select {
		case <-m.ctx.Done():
			m.mu.Lock()
			m.reconnecting = false
			m.mu.Unlock()
			return
		case <-time.After(delay):
			// 继续重连
		}
	}
}

// Reset 重置重连状态
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.attempt = 0
	m.reconnecting = false
}
