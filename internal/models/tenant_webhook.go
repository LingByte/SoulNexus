package models

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TenantWebhook is an HTTP callback endpoint for call lifecycle events.
type TenantWebhook struct {
	common.BaseModel

	TenantID uint           `json:"tenantId" gorm:"index;not null"`
	Name     string         `json:"name" gorm:"size:128;not null"`
	URL      string         `json:"url" gorm:"size:512;not null"`
	Secret   string         `json:"secret,omitempty" gorm:"size:128"`
	Events   datatypes.JSON `json:"events" gorm:"type:json;not null"`
	Enabled  bool           `json:"enabled" gorm:"index;not null;default:true"`
}

func (TenantWebhook) TableName() string {
	return constants.TENANT_WEBHOOK_TABLE_NAME
}

// TenantWebhookDelivery records one delivery attempt (audit / retry visibility).
type TenantWebhookDelivery struct {
	common.BaseModel

	TenantID    uint       `json:"tenantId" gorm:"index;not null"`
	WebhookID   uint       `json:"webhookId" gorm:"index;not null"`
	Event       string     `json:"event" gorm:"size:64;index;not null"`
	CallID      string     `json:"callId" gorm:"size:128;index"`
	Status      string     `json:"status" gorm:"size:24;index;not null"` // success|failed|pending|dlq
	HTTPCode    int        `json:"httpCode" gorm:"index"`
	Error       string     `json:"error,omitempty" gorm:"type:text"`
	Payload     string     `json:"payload,omitempty" gorm:"type:text"`
	Attempt     int        `json:"attempt" gorm:"not null;default:1"`
	MaxAttempts int        `json:"maxAttempts" gorm:"not null;default:5"`
	NextRetryAt *time.Time `json:"nextRetryAt,omitempty" gorm:"index"`
}

func (TenantWebhookDelivery) TableName() string {
	return constants.TENANT_WEBHOOK_DELIVERY_TABLE_NAME
}

func ListTenantWebhooksPage(db *gorm.DB, tenantID uint, page, size int) ([]TenantWebhook, int64, error) {
	q := db.Model(&TenantWebhook{}).Where("tenant_id = ?", tenantID)
	return utils.FindPage[TenantWebhook](q, page, size, "id DESC", utils.DefaultMaxPageSize)
}
