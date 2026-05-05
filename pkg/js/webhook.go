package js

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils/cache"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WebhookManager Webhook管理器
type WebhookManager struct {
	cache cache.Cache
}

// NewWebhookManager 创建Webhook管理器
func NewWebhookManager(cache cache.Cache) *WebhookManager {
	return &WebhookManager{
		cache: cache,
	}
}

// VerifyWebhookSignature 验证Webhook签名
func (wm *WebhookManager) VerifyWebhookSignature(c *gin.Context, secret string) error {
	if secret == "" {
		return nil // 如果没有配置密钥，跳过验证
	}

	// 获取签名
	signature := c.GetHeader("X-Webhook-Signature")
	if signature == "" {
		signature = c.GetHeader("X-Signature")
	}
	if signature == "" {
		return fmt.Errorf("missing webhook signature")
	}

	// 获取时间戳
	timestampStr := c.GetHeader("X-Timestamp")
	if timestampStr == "" {
		timestampStr = c.DefaultQuery("timestamp", "")
	}
	if timestampStr == "" {
		return fmt.Errorf("missing timestamp")
	}

	// 验证时间戳有效性（防止重放攻击，15分钟内有效）
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format")
	}

	currentTime := time.Now().Unix()
	if abs(currentTime-timestamp) > 15*60 {
		return fmt.Errorf("request expired")
	}

	// 获取nonce
	nonce := c.GetHeader("X-Nonce")
	if nonce == "" {
		nonce = c.DefaultQuery("nonce", "")
	}

	// 读取请求体
	bodyBytes, err := c.GetRawData()
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// 将请求体重新写回上下文
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	requestBody := string(bodyBytes)

	// 构建签名数据
	var signatureData strings.Builder
	signatureData.WriteString(c.Request.Method)
	signatureData.WriteString(c.Request.URL.Path)
	signatureData.WriteString(requestBody)
	signatureData.WriteString(timestampStr)
	if nonce != "" {
		signatureData.WriteString(nonce)
	}

	// 生成期望的签名
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signatureData.String()))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// 使用时间常数比较防止时序攻击
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// CheckNonceDuplicate 检查nonce是否重复（防重复请求）
func (wm *WebhookManager) CheckNonceDuplicate(ctx context.Context, nonce string, templateID string) (bool, error) {
	if nonce == "" {
		return false, nil // 如果没有nonce，跳过检查
	}

	nonceKey := fmt.Sprintf("js:webhook:nonce:%s:%s", templateID, nonce)

	// 检查nonce是否已存在
	exists := wm.cache.Exists(ctx, nonceKey)
	if exists {
		logger.Warn("重复的nonce请求",
			zap.String("templateID", templateID),
			zap.String("nonce", nonce))
		return true, fmt.Errorf("duplicate nonce: request already processed")
	}

	// 存储nonce，15分钟过期（与时间戳验证窗口一致）
	err := wm.cache.Set(ctx, nonceKey, true, 15*time.Minute)
	if err != nil {
		logger.Error("failed to cache nonce", zap.Error(err))
		return false, err
	}

	return false, nil
}

// CheckIdempotency 检查幂等性（通过请求ID）
func (wm *WebhookManager) CheckIdempotency(ctx context.Context, requestID string, templateID string) (interface{}, bool, error) {
	if requestID == "" {
		return nil, false, nil // 如果没有请求ID，跳过检查
	}

	idempotencyKey := fmt.Sprintf("js:webhook:idempotency:%s:%s", templateID, requestID)

	// 检查是否已有相同的请求ID
	result, exists := wm.cache.Get(ctx, idempotencyKey)
	if exists {
		logger.Info("幂等性请求，返回缓存结果",
			zap.String("templateID", templateID),
			zap.String("requestID", requestID))
		return result, true, nil
	}

	return nil, false, nil
}

// StoreIdempotencyResult 存储幂等性结果
func (wm *WebhookManager) StoreIdempotencyResult(ctx context.Context, requestID string, templateID string, result interface{}) error {
	if requestID == "" {
		return nil
	}

	idempotencyKey := fmt.Sprintf("js:webhook:idempotency:%s:%s", templateID, requestID)
	// 存储结果，24小时过期
	return wm.cache.Set(ctx, idempotencyKey, result, 24*time.Hour)
}

// abs 返回绝对值
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
