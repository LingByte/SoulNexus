package models

import (
	"time"

	"gorm.io/gorm"
)

type CallCostSummary struct {
	TotalCalls         int64   `json:"totalCalls" gorm:"column:total_calls"`
	TotalBilledMinutes int64   `json:"totalBilledMinutes" gorm:"column:total_billed_minutes"`
	TotalAmountCharged float64 `json:"totalAmountCharged" gorm:"column:total_amount_charged"`
}

type TenantCallCostRow struct {
	TenantID      uint    `json:"tenantId"`
	TenantName    string  `json:"tenantName"`
	CallCount     int64   `json:"callCount"`
	BilledMinutes int64   `json:"billedMinutes"`
	AmountCharged float64 `json:"amountCharged"`
}

func SummarizePlatformCallCost(db *gorm.DB, from, to *time.Time) (CallCostSummary, []TenantCallCostRow, error) {
	var summary CallCostSummary
	if db == nil {
		return summary, nil, gorm.ErrInvalidDB
	}
	q := db.Model(&TenantUsageEvent{})
	if from != nil {
		q = q.Where("created_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("created_at <= ?", *to)
	}
	if err := q.Select(`
		COUNT(*) AS total_calls,
		COALESCE(SUM(billed_minutes), 0) AS total_billed_minutes,
		COALESCE(SUM(amount_charged), 0) AS total_amount_charged
	`).Scan(&summary).Error; err != nil {
		return summary, nil, err
	}

	type row struct {
		TenantID      uint
		CallCount     int64
		BilledMinutes int64
		AmountCharged float64
	}
	var grouped []row
	gq := db.Model(&TenantUsageEvent{})
	if from != nil {
		gq = gq.Where("created_at >= ?", *from)
	}
	if to != nil {
		gq = gq.Where("created_at <= ?", *to)
	}
	if err := gq.Select(`
		tenant_id,
		COUNT(*) AS call_count,
		COALESCE(SUM(billed_minutes), 0) AS billed_minutes,
		COALESCE(SUM(amount_charged), 0) AS amount_charged
	`).Group("tenant_id").Order("amount_charged DESC").Limit(100).Scan(&grouped).Error; err != nil {
		return summary, nil, err
	}
	out := make([]TenantCallCostRow, 0, len(grouped))
	for _, g := range grouped {
		name := ""
		if t, err := GetActiveTenantByID(db, g.TenantID); err == nil {
			name = t.Name
		}
		out = append(out, TenantCallCostRow{
			TenantID:      g.TenantID,
			TenantName:    name,
			CallCount:     g.CallCount,
			BilledMinutes: g.BilledMinutes,
			AmountCharged: g.AmountCharged,
		})
	}
	return summary, out, nil
}

func billedMinutesFromDurationSec(durationSec int) int64 {
	if durationSec <= 0 {
		return 0
	}
	m := int64((durationSec + 59) / 60)
	if m < 1 {
		return 1
	}
	return m
}
