package task

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ProcessDueAccountDeletions 执行已到期的永久注销，返回处理条数。
func ProcessDueAccountDeletions(db *gorm.DB) int {
	if db == nil {
		return 0
	}
	users, err := models.ListUsersDueForAccountDeletion(db, time.Now())
	if err != nil {
		logger.Error("list due account deletions failed", zap.Error(err))
		return 0
	}
	n := 0
	for i := range users {
		u := users[i]
		if err := models.FinalizeAccountDeletion(db, u.ID, "account_deletion_cron"); err != nil {
			logger.Warn("finalize account deletion failed",
				zap.Uint("userID", u.ID),
				zap.Error(err))
			continue
		}
		n++
	}
	return n
}

// StartAccountDeletionScheduler 定时扫描冷静期已结束的账号并执行注销。
func StartAccountDeletionScheduler(db *gorm.DB) {
	c := cron.New()
	_, err := c.AddFunc("*/10 * * * *", func() {
		if k := ProcessDueAccountDeletions(db); k > 0 {
			logger.Info("account deletion cron completed", zap.Int("finalized", k))
		}
	})
	if err != nil {
		logger.Error("account deletion cron register failed", zap.Error(err))
		return
	}
	c.Start()
	logger.Info("account deletion scheduler started", zap.String("schedule", "*/10 * * * *"))
}
