package tasks

import (
	"context"

	"gorm.io/gorm"
)

// DeliverAIReport is a no-op after telephony AI reports were removed.
func DeliverAIReport(ctx context.Context, db *gorm.DB, tenantID uint, row any, snap any, period, reportType string, doPush bool) {
	_ = ctx
	_ = db
	_ = tenantID
	_ = row
	_ = snap
	_ = period
	_ = reportType
	_ = doPush
}
