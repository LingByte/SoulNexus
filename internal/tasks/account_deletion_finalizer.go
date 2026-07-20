package tasks

import (
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var accountDeletionCron *cron.Cron

func StartAccountDeletionFinalizer(db *gorm.DB) {
	c := cron.New()
	accountDeletionCron = c
	schedule := "15 */6 * * *"
	_, err := c.AddFunc(schedule, func() {
		if err := FinalizeExpiredAccountDeletions(db); err != nil {
			logger.Error("account deletion finalizer failed", zap.Error(err))
		}
	})
	if err != nil {
		logger.Error("failed to add account deletion finalizer cron", zap.Error(err))
		return
	}
	c.Start()
	logger.Info("account deletion finalizer started", zap.String("schedule", schedule))
}

func StopAccountDeletionFinalizer() {
	if accountDeletionCron != nil {
		accountDeletionCron.Stop()
	}
}

func FinalizeExpiredAccountDeletions(db *gorm.DB) error {
	cutoff := time.Now().Add(-constants.AccountDeletionCoolingPeriod)

	var tenantUsers []models.TenantUser
	if err := db.Where("status = ? AND deletion_requested_at IS NOT NULL AND deletion_requested_at <= ?",
		constants.TenantUserStatusPendingDeletion, cutoff).
		Find(&tenantUsers).Error; err != nil {
		return err
	}
	for _, u := range tenantUsers {
		if _, err := models.SoftDeleteTenantUserByID(db, u.ID, "account-deletion-finalizer"); err != nil {
			logger.Warn("finalize tenant user deletion failed", zap.Uint("userId", u.ID), zap.Error(err))
		}
	}

	var admins []models.PlatformAdmin
	if err := db.Where("status = ? AND deletion_requested_at IS NOT NULL AND deletion_requested_at <= ?",
		constants.PlatformAdminStatusPendingDeletion, cutoff).
		Find(&admins).Error; err != nil {
		return err
	}
	for _, a := range admins {
		if err := db.Delete(&models.PlatformAdmin{}, a.ID).Error; err != nil {
			logger.Warn("finalize platform admin deletion failed", zap.Uint("adminId", a.ID), zap.Error(err))
		}
	}
	return nil
}
