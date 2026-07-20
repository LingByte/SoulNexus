// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package tasks

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification/webhook"
	"github.com/LingByte/SoulNexus/pkg/utils/redislock"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// StartWebhookRetryWorker polls pending webhook deliveries for exponential backoff retries.
// With REDIS_ADDR, uses a short-lived lock so only one replica processes a tick.
func StartWebhookRetryWorker(db *gorm.DB) {
	if db == nil {
		return
	}
	if addr := strings.TrimSpace(os.Getenv("REDIS_ADDR")); addr != "" {
		if err := redislock.InitFromEnv(); err != nil && logger.Lg != nil {
			logger.Lg.Warn("webhook retry redis lock unavailable; continuing without distributed lock", zap.Error(err))
		}
	}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			unlock, ok, err := redislock.TryLock(ctx, "soulnexus:webhook_retry", 8*time.Second)
			cancel()
			if err != nil && logger.Lg != nil {
				logger.Lg.Warn("webhook retry lock error", zap.Error(err))
				continue
			}
			if !ok {
				continue
			}
			n := webhook.ProcessPendingRetries(db, logger.Lg, 32)
			unlock()
			if n > 0 && logger.Lg != nil {
				logger.Lg.Info("webhook retries processed", zap.Int("count", n))
			}
		}
	}()
}
