package task

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// StartEmailCleaner starts the email cleanup scheduled task
func StartEmailCleaner(db *gorm.DB) {
	c := cron.New()

	// Execute cleanup task at 2 AM every day
	schedule := "0 2 * * *"

	// Add scheduled task
	_, err := c.AddFunc(schedule, func() {
		if err := CleanUnreadEmails(db); err != nil {
			logger.Error("Email cleaner task failed", zap.Error(err))
		} else {
			logger.Info("Email cleaner task completed successfully")
		}
	})

	if err != nil {
		logger.Error("Failed to add email cleaner cron job", zap.Error(err))
		return
	}

	// Start the scheduled task
	c.Start()

	logger.Info("Email cleaner started", zap.String("schedule", schedule))
}

// CleanUnreadEmails cleans up emails unread for more than seven days
func CleanUnreadEmails(db *gorm.DB) error {
	// Calculate the time seven days ago
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	// Process all enabled users after removing per-user auto-clean preference.
	var userIDs []uint
	err := db.Table(constants.USER_TABLE_NAME).
		Where("status = ?", models.UserStatusActive).
		Pluck("id", &userIDs).Error

	if err != nil {
		return err
	}

	if len(userIDs) == 0 {
		logger.Info("No enabled users found, skipping email cleanup")
		return nil
	}

	totalDeleted := int64(0)
	for _, userID := range userIDs {
		// Delete notifications unread for more than seven days for this user
		result := db.Where("user_id = ? AND `read` = ? AND created_at < ?", userID, false, sevenDaysAgo).
			Delete(&models.InternalNotification{})

		if result.Error != nil {
			logger.Warn("Failed to clean emails for user", zap.Uint("userID", userID), zap.Error(result.Error))
			continue
		}

		deletedCount := result.RowsAffected
		totalDeleted += deletedCount

		if deletedCount > 0 {
			logger.Info("Cleaned unread emails for user",
				zap.Uint("userID", userID),
				zap.Int64("deletedCount", deletedCount))
		}
	}

	logger.Info("Email cleanup completed",
		zap.Int("usersProcessed", len(userIDs)),
		zap.Int64("totalDeleted", totalDeleted))

	return nil
}
