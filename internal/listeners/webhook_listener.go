package listeners

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	webhookDB *gorm.DB
	webhookLg *zap.Logger
)

// InitWebhookListeners previously wired call lifecycle signals to tenant HTTP webhooks.
// Telephony has been removed; this is now a no-op that retains the init signature.
func InitWebhookListeners(db *gorm.DB, lg *zap.Logger) {
	if db == nil {
		return
	}
	if lg == nil {
		lg = zap.NewNop()
	}
	webhookDB = db
	webhookLg = lg
}
