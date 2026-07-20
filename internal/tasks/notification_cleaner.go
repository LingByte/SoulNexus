package tasks

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/notification/inbox"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var notificationCleanerCron *cron.Cron

func StartNotificationCleaner(db *gorm.DB) {
	c := cron.New()
	notificationCleanerCron = c
	schedule := "0 2 * * *"
	_, err := c.AddFunc(schedule, func() {
		if err := CleanOldUnreadNotifications(db); err != nil {
			logger.Error("notification cleaner task failed", zap.Error(err))
		}
	})
	if err != nil {
		logger.Error("failed to add notification cleaner cron job", zap.Error(err))
		return
	}
	c.Start()
	logger.Info("notification cleaner started", zap.String("schedule", schedule))
}

func StopNotificationCleaner() {
	if notificationCleanerCron != nil {
		notificationCleanerCron.Stop()
	}
}

func CleanOldUnreadNotifications(db *gorm.DB) error {
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	n, err := inbox.NewService(db).CleanOldUnread(sevenDaysAgo)
	if err != nil {
		return err
	}
	if n > 0 {
		logger.Info("inbox cleanup completed", zap.Int64("totalDeleted", n))
	}
	return nil
}
