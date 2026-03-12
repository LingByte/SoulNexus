package asr

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/voice/errhandler"
	"github.com/LingByte/SoulNexus/pkg/voice/reconnect"
	"go.uber.org/zap"
)

// Service ASR服务实现
type Service struct {
	ctx          context.Context
	cancel       context.CancelFunc
	credential   *models.UserCredential
	language     string
	transcriber  recognizer.TranscribeService
	errorHandler *errhandler.Handler
	reconnectMgr *reconnect.Manager
	logger       *zap.Logger
	mu           sync.RWMutex
	connected    bool
	pool         *Pool // 连接池引用
	onResult     func(text string, isLast bool, duration time.Duration, uuid string)
	onError      func(err error)
}

// NewService 创建ASR服务
func NewService(
	ctx context.Context,
	credential *models.UserCredential,
	language string,
	transcriber recognizer.TranscribeService,
	errorHandler *errhandler.Handler,
	logger *zap.Logger,
) *Service {
	ctx, cancel := context.WithCancel(ctx)

	service := &Service{
		ctx:          ctx,
		cancel:       cancel,
		credential:   credential,
		language:     language,
		transcriber:  transcriber,
		errorHandler: errorHandler,
		logger:       logger,
	}

	// 创建重连管理器
	strategy := reconnect.NewExponentialBackoffStrategy()
	// 设置限流错误检查器
	strategy.SetRateLimitChecker(errorHandler.IsRateLimitError)
	reconnectMgr := reconnect.NewManager(ctx, logger, strategy)
	reconnectMgr.SetReconnectCallback(service.reconnect)
	reconnectMgr.SetDisconnectCallback(service.onDisconnect)
	service.reconnectMgr = reconnectMgr

	return service
}

// SetPool 设置连接池
func (s *Service) SetPool(pool *Pool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pool = pool
}

// SetCallbacks 设置回调函数
func (s *Service) SetCallbacks(
	onResult func(text string, isLast bool, duration time.Duration, uuid string),
	onError func(err error),
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onResult = onResult
	s.onError = onError
}

// Connect 建立连接
func (s *Service) Connect() error {
	s.mu.Lock()
	if s.connected {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	// 初始化ASR服务（不在这里获取连接池许可，在receiveLoop中获取）
	s.transcriber.Init(
		func(text string, isLast bool, duration time.Duration, uuid string) {
			if s.onResult != nil {
				s.onResult(text, isLast, duration, uuid)
			}
		},
		func(err error, isFatal bool) {
			if s.onError != nil {
				s.onError(err)
			}
			if err != nil {
				classified := s.errorHandler.Classify(err, "ASR")
				if classified.Type == errhandler.ErrorTypeFatal {
					s.mu.Lock()
					s.connected = false
					if s.pool != nil {
						s.pool.Release()
					}
					s.mu.Unlock()
				} else if classified.Type == errhandler.ErrorTypeTransient {
					// 临时错误，通知重连管理器
					s.reconnectMgr.NotifyDisconnect(err)
				}
			}
		},
	)

	// 启动连接和接收循环（在receiveLoop中获取连接池许可）
	go s.receiveLoop()

	return nil
}

// Disconnect 断开连接
func (s *Service) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil
	}

	s.cancel()
	if s.transcriber != nil {
		s.transcriber.StopConn()
	}

	s.connected = false

	// 释放连接池许可
	if s.pool != nil {
		s.pool.Release()
	}

	return nil
}

// SendAudio 发送音频数据
func (s *Service) SendAudio(data []byte) error {
	s.mu.RLock()
	connected := s.connected
	transcriber := s.transcriber
	s.mu.RUnlock()

	if !connected || transcriber == nil {
		return errhandler.NewTransientError("ASR", "服务未连接", nil)
	}

	if err := transcriber.SendAudioBytes(data); err != nil {
		// 检查是否是连接问题
		if !transcriber.Activity() {
			s.mu.Lock()
			s.connected = false
			s.mu.Unlock()
			s.reconnectMgr.NotifyDisconnect(err)
		}
		return errhandler.NewTransientError("ASR", "发送音频失败", err)
	}

	return nil
}

// IsConnected 检查是否已连接
func (s *Service) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// Activity 检查服务是否活跃
func (s *Service) Activity() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.connected || s.transcriber == nil {
		return false
	}
	return s.transcriber.Activity()
}

// receiveLoop 接收循环
func (s *Service) receiveLoop() {
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("ASR接收循环退出")
			// 退出时释放连接池许可
			s.mu.Lock()
			if s.pool != nil && s.connected {
				s.pool.Release()
			}
			s.connected = false
			s.mu.Unlock()
			return
		default:
		}

		// 如果有连接池，先获取连接许可
		if s.pool != nil {
			if !s.pool.Acquire() {
				s.logger.Warn("ASR连接池已满，等待后重试")
				// 连接池已满，等待一段时间后重试
				select {
				case <-s.ctx.Done():
					return
				case <-time.After(2 * time.Second):
					continue
				}
			}
		}

		// 尝试连接
		err := s.transcriber.ConnAndReceive("")
		if err != nil {
			// 连接失败，立即释放连接池许可
			s.mu.Lock()
			s.connected = false
			if s.pool != nil {
				s.pool.Release()
			}
			s.mu.Unlock()

			classified := s.errorHandler.Classify(err, "ASR")

			// 检查是否是并发超限错误
			if s.errorHandler.IsRateLimitError(err) {
				s.logger.Warn("ASR并发超限，等待后重试",
					zap.Error(err),
					zap.String("action", "rate_limit_detected"),
				)

				// 并发超限错误，等待更长时间（30秒）再重连
				// 不通知重连管理器，直接在这里等待
				select {
				case <-s.ctx.Done():
					return
				case <-time.After(30 * time.Second): // 等待30秒
					// 继续循环，尝试重新获取连接池许可并连接
					s.logger.Info("ASR并发超限等待结束，尝试重新连接")
					continue
				}
			}

			if classified.Type == errhandler.ErrorTypeFatal {
				s.logger.Error("ASR连接致命错误", zap.Error(err))
				if s.onError != nil {
					s.onError(classified)
				}
				return
			}

			// 其他错误，通知重连管理器（但需要先等待一段时间）
			s.logger.Warn("ASR连接失败，等待后重连", zap.Error(err))
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(3 * time.Second): // 等待3秒后再通知重连
				s.reconnectMgr.NotifyDisconnect(err)
				// 等待重连管理器处理
				select {
				case <-s.ctx.Done():
					return
				case <-time.After(2 * time.Second):
					// 继续循环
				}
			}
		} else {
			// 连接成功
			s.mu.Lock()
			s.connected = true
			s.mu.Unlock()
			s.reconnectMgr.Reset()
			s.logger.Info("ASR连接成功")

			// ConnAndReceive 对于某些提供商（如腾讯云）会立即返回
			// 我们需要通过 Activity() 来检查连接是否真的活跃
			// 保持连接，等待音频数据或连接断开
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-s.ctx.Done():
					s.mu.Lock()
					s.connected = false
					if s.pool != nil {
						s.pool.Release()
					}
					s.mu.Unlock()
					return
				case <-ticker.C:
					// 定期检查连接是否还活跃
					if !s.transcriber.Activity() {
						s.logger.Info("ASR连接已断开（Activity检查）")
						s.mu.Lock()
						s.connected = false
						if s.pool != nil {
							s.pool.Release()
						}
						s.mu.Unlock()

						// 等待一段时间后再尝试重连
						select {
						case <-s.ctx.Done():
							return
						case <-time.After(2 * time.Second):
							// 继续外层循环，尝试重新连接
							goto reconnect
						}
					}
				}
			}

		reconnect:
			// 继续循环，尝试重新连接
		}
	}
}

// reconnect 重连
func (s *Service) reconnect() error {
	s.mu.Lock()
	if s.connected {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	// 重新初始化
	s.transcriber.Init(
		func(text string, isLast bool, duration time.Duration, uuid string) {
			if s.onResult != nil {
				s.onResult(text, isLast, duration, uuid)
			}
		},
		func(err error, isFatal bool) {
			if s.onError != nil {
				s.onError(err)
			}
		},
	)

	// 启动新的接收循环（receiveLoop会自己获取连接池许可）
	go s.receiveLoop()

	return nil
}

// onDisconnect 断开连接回调
func (s *Service) onDisconnect(err error) {
	s.logger.Warn("ASR连接断开", zap.Error(err))
}
