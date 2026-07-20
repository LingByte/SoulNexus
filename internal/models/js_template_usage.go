package models

import (
	"strings"
	"time"

	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

const (
	JSTemplateUsageLoad         = "load"          // embed.js fetched
	JSTemplateUsageSessionStart = "session_start" // voice-session created from widget
)

// JSTemplateUsageLog records JS template (mini-program / APP / H5) load and session opens.
type JSTemplateUsageLog struct {
	ID         uint      `json:"id,string" gorm:"primaryKey;autoIncrement:false"`
	TenantID   uint      `json:"tenantId,string" gorm:"index;not null"`
	JsSourceID string    `json:"jsSourceId" gorm:"column:js_source_id;size:64;index;not null"`
	Event      string    `json:"event" gorm:"size:32;index;not null"` // load | session_start
	SessionID  string    `json:"sessionId,omitempty" gorm:"column:session_id;size:128;index"`
	CredentialID uint    `json:"credentialId,string,omitempty" gorm:"column:credential_id;index;default:0"`
	UserID     uint      `json:"userId,string,omitempty" gorm:"column:user_id;index;default:0"`
	CreatedAt  time.Time `json:"createdAt" gorm:"index"`
}

func (JSTemplateUsageLog) TableName() string {
	return constants2.JS_TEMPLATE_USAGE_TABLE_NAME
}

// RecordJSTemplateUsage appends one usage row (best-effort; never blocks callers on failure).
func RecordJSTemplateUsage(db *gorm.DB, tenantID uint, jsSourceID, event, sessionID string, credentialID, userID uint) error {
	if db == nil {
		return nil
	}
	jsSourceID = strings.TrimSpace(jsSourceID)
	event = strings.TrimSpace(event)
	if jsSourceID == "" || event == "" {
		return nil
	}
	// Resolve tenant from template when caller only has jsSourceID (public embed load).
	if tenantID == 0 {
		if row, err := GetActiveJSTemplateByJsSourceID(db, jsSourceID); err == nil {
			tenantID = row.TenantID
		}
	}
	if tenantID == 0 {
		return nil
	}
	entry := JSTemplateUsageLog{
		ID:           utils.NextSnowflakeUint(),
		TenantID:     tenantID,
		JsSourceID:   jsSourceID,
		Event:        event,
		SessionID:    strings.TrimSpace(sessionID),
		CredentialID: credentialID,
		UserID:       userID,
		CreatedAt:    time.Now().UTC(),
	}
	return db.Create(&entry).Error
}

// ListJSTemplateUsagePage lists usage for a tenant, optional jsSourceID filter.
func ListJSTemplateUsagePage(db *gorm.DB, tenantID uint, jsSourceID string, page, size int) ([]JSTemplateUsageLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	q := db.Model(&JSTemplateUsageLog{}).Where("tenant_id = ?", tenantID)
	if id := strings.TrimSpace(jsSourceID); id != "" {
		q = q.Where("js_source_id = ?", id)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []JSTemplateUsageLog
	err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}
