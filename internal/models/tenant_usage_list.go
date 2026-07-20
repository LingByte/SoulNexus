package models

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"gorm.io/gorm"
)

// TenantUsageEventListFilter scopes usage event queries.
type TenantUsageEventListFilter struct {
	TenantID uint
	From     *time.Time
	To       *time.Time
}

// ListTenantUsageEventsPage returns paginated meter rows newest first.
func ListTenantUsageEventsPage(db *gorm.DB, page, size int, f TenantUsageEventListFilter) ([]TenantUsageEvent, int64, error) {
	q := db.Model(&TenantUsageEvent{})
	if f.TenantID > 0 {
		q = q.Where("tenant_id = ?", f.TenantID)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at < ?", *f.To)
	}
	return utils.FindPage[TenantUsageEvent](q, page, size, "id DESC", utils.DefaultMaxPageSize)
}
