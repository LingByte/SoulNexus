package webhook

import (
	"testing"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDeliverAttemptBlocksSSRF(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:webhook_ssrf?mode=memory&cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.TenantWebhook{}, &models.TenantWebhookDelivery{}); err != nil {
		t.Fatal(err)
	}
	hook := models.TenantWebhook{TenantID: 1, Name: "t", URL: "http://127.0.0.1/hook", Enabled: true, Events: []byte(`["call.ended"]`)}
	if err := db.Create(&hook).Error; err != nil {
		t.Fatal(err)
	}
	if err := utils.ValidateURLForSSRF(hook.URL); err == nil {
		t.Fatal("expected SSRF error for loopback URL")
	}
	deliverAttempt(db, nil, hook, "call.ended", "c1", []byte(`{}`), 1, 3, 0)
	var n int64
	_ = db.Model(&models.TenantWebhookDelivery{}).Where("status = ?", "dlq").Count(&n).Error
	if n != 1 {
		t.Fatalf("want 1 dlq row, got %d", n)
	}
}
