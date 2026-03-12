package js

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/pkg/cache"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// QuotaManager 资源配额管理器
type QuotaManager struct {
	cache cache.Cache
}

// NewQuotaManager 创建配额管理器
func NewQuotaManager(cache cache.Cache) *QuotaManager {
	return &QuotaManager{
		cache: cache,
	}
}

// ExecutionMetrics 执行指标
type ExecutionMetrics struct {
	TemplateID    string
	ExecutionTime time.Duration
	MemoryUsedMB  int
	APICallsCount int
	StartTime     time.Time
}

// CheckQuota 检查配额是否超限
func (qm *QuotaManager) CheckQuota(ctx context.Context, templateID string, maxExecutionTime, maxMemoryMB, maxAPICalls int) (bool, error) {
	// 获取当前执行指标
	metricsKey := fmt.Sprintf("js:template:metrics:%s", templateID)
	metricsObj, exists := qm.cache.Get(ctx, metricsKey)
	if !exists {
		return true, nil
	}

	metrics, ok := metricsObj.(ExecutionMetrics)
	if !ok {
		return true, nil
	}

	// 检查执行时间
	if metrics.ExecutionTime > time.Duration(maxExecutionTime)*time.Millisecond {
		logger.Warn("执行时间超限",
			zap.String("templateID", templateID),
			zap.Duration("executionTime", metrics.ExecutionTime),
			zap.Int("maxExecutionTime", maxExecutionTime))
		return false, fmt.Errorf("执行时间超限: %v > %dms", metrics.ExecutionTime, maxExecutionTime)
	}

	// 检查内存使用
	if metrics.MemoryUsedMB > maxMemoryMB {
		logger.Warn("内存使用超限",
			zap.String("templateID", templateID),
			zap.Int("memoryUsedMB", metrics.MemoryUsedMB),
			zap.Int("maxMemoryMB", maxMemoryMB))
		return false, fmt.Errorf("内存使用超限: %dMB > %dMB", metrics.MemoryUsedMB, maxMemoryMB)
	}

	// 检查API调用次数
	if metrics.APICallsCount > maxAPICalls {
		logger.Warn("API调用次数超限",
			zap.String("templateID", templateID),
			zap.Int("apiCallsCount", metrics.APICallsCount),
			zap.Int("maxAPICalls", maxAPICalls))
		return false, fmt.Errorf("API调用次数超限: %d > %d", metrics.APICallsCount, maxAPICalls)
	}

	return true, nil
}

// RecordExecution 记录执行指标
func (qm *QuotaManager) RecordExecution(ctx context.Context, templateID string, metrics ExecutionMetrics) error {
	metricsKey := fmt.Sprintf("js:template:metrics:%s", templateID)
	// 记录执行指标，过期时间5分钟
	return qm.cache.Set(ctx, metricsKey, metrics, 5*time.Minute)
}

// IncrementAPICall 增加API调用计数
func (qm *QuotaManager) IncrementAPICall(ctx context.Context, templateID string) error {
	metricsKey := fmt.Sprintf("js:template:metrics:%s", templateID)
	metricsObj, exists := qm.cache.Get(ctx, metricsKey)

	var metrics ExecutionMetrics
	if exists {
		if m, ok := metricsObj.(ExecutionMetrics); ok {
			metrics = m
		}
	}

	metrics.TemplateID = templateID
	metrics.APICallsCount++
	if metrics.StartTime.IsZero() {
		metrics.StartTime = time.Now()
	}

	return qm.cache.Set(ctx, metricsKey, metrics, 5*time.Minute)
}
