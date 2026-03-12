package asr

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voice/errhandler"
	"go.uber.org/zap"
)

// Pool ASR连接池，用于控制并发连接数
type Pool struct {
	maxConcurrent int
	currentCount  int
	mu            sync.RWMutex
	waitQueue     chan *Service
	logger        *zap.Logger
}

var (
	globalPool *Pool
	poolOnce   sync.Once
)

// GetGlobalPool 获取全局ASR连接池
func GetGlobalPool(maxConcurrent int, logger *zap.Logger) *Pool {
	poolOnce.Do(func() {
		globalPool = &Pool{
			maxConcurrent: maxConcurrent,
			waitQueue:     make(chan *Service, 100), // 等待队列
			logger:        logger,
		}
	})
	return globalPool
}

// Acquire 获取连接许可（非阻塞）
func (p *Pool) Acquire() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentCount < p.maxConcurrent {
		p.currentCount++
		p.logger.Debug("ASR连接池：获取连接许可",
			zap.Int("current", p.currentCount),
			zap.Int("max", p.maxConcurrent),
		)
		return true
	}

	return false
}

// Release 释放连接许可
func (p *Pool) Release() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.currentCount > 0 {
		p.currentCount--
		p.logger.Debug("ASR连接池：释放连接许可",
			zap.Int("current", p.currentCount),
			zap.Int("max", p.maxConcurrent),
		)
	}

	// 尝试从等待队列中唤醒一个服务
	select {
	case service := <-p.waitQueue:
		// 如果队列中有等待的服务，尝试让它连接
		go func() {
			// 给一点时间让当前连接完全释放
			time.Sleep(100 * time.Millisecond)
			if p.Acquire() {
				// 重新尝试连接
				if err := service.Connect(); err != nil {
					p.logger.Warn("ASR连接池：等待队列中的服务连接失败", zap.Error(err))
					p.Release()
				}
			} else {
				// 如果还是无法获取，放回队列
				select {
				case p.waitQueue <- service:
				default:
					p.logger.Warn("ASR连接池：等待队列已满")
				}
			}
		}()
	default:
		// 没有等待的服务
	}
}

// TryConnectWithPool 尝试连接，如果连接池已满则等待
func (p *Pool) TryConnectWithPool(service *Service, ctx context.Context) error {
	// 尝试获取连接许可
	if p.Acquire() {
		// 成功获取，直接连接
		return service.Connect()
	}

	// 连接池已满，等待
	p.logger.Warn("ASR连接池已满，等待连接许可",
		zap.Int("current", p.currentCount),
		zap.Int("max", p.maxConcurrent),
	)

	// 将服务放入等待队列
	select {
	case p.waitQueue <- service:
		// 等待一段时间后重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			// 再次尝试
			if p.Acquire() {
				return service.Connect()
			}
			// 如果还是无法获取，返回错误
			return errhandler.NewRecoverableError("ASR", "连接池已满，请稍后重试", nil)
		}
	default:
		// 等待队列已满
		return errhandler.NewRecoverableError("ASR", "连接池和等待队列已满，请稍后重试", nil)
	}
}

// GetCurrentCount 获取当前连接数
func (p *Pool) GetCurrentCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentCount
}

// GetMaxConcurrent 获取最大并发数
func (p *Pool) GetMaxConcurrent() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.maxConcurrent
}
