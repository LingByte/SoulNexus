package task

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/pkg/alert"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// StartQuotaAlertChecker starts the quota alert checking scheduled task
func StartQuotaAlertChecker(db *gorm.DB) {
	// Create alert trigger service
	triggerService := alert.NewTriggerService(db)

	// Create quota checker
	checker := alert.NewQuotaChecker(db, triggerService)

	// Execute a check immediately at startup
	logger.Info("Executing quota alert check at startup")
	checker.CheckAllQuotaAlerts()

	// Use cron for scheduled execution
	c := cron.New()

	// Execute quota check every 5 minutes
	schedule := "*/5 * * * *"

	// Add scheduled task
	_, err := c.AddFunc(schedule, func() {
		logger.Info("Starting quota alert check execution")
		checker.CheckAllQuotaAlerts()
		logger.Info("Quota alert check completed")
	})

	if err != nil {
		logger.Error("Failed to add quota alert checker cron job", zap.Error(err))
		return
	}

	// Start the scheduled task
	c.Start()

	logger.Info("Quota alert checker started", zap.String("schedule", schedule))
}
