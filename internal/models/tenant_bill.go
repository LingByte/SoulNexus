package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrTenantBillNotFound      = errors.New("tenant bill not found")
	ErrTenantBillAlreadyLocked = errors.New("tenant bill already finalized")
	ErrTenantBillInvalidPeriod = errors.New("invalid billing period")
	ErrTenantBillNotFinalized  = errors.New("tenant bill must be finalized before marking paid")
	ErrTenantBillAlreadyPaid   = errors.New("tenant bill already paid")
)

// TenantBill is a monthly usage statement for one tenant (pricing unlimited for now).
type TenantBill struct {
	common.BaseModel
	TenantID           uint           `json:"tenantId" gorm:"index;not null;default:0"`
	BillNo             string         `json:"billNo" gorm:"size:64;uniqueIndex;not null"`
	PeriodStart        time.Time      `json:"periodStart" gorm:"index;not null"`
	PeriodEnd          time.Time      `json:"periodEnd" gorm:"index;not null"`
	Status             string         `json:"status" gorm:"size:24;index;not null;default:draft"`
	Currency           string         `json:"currency" gorm:"size:8;not null;default:CNY"`
	TotalAmount        float64        `json:"totalAmount" gorm:"type:decimal(16,4);not null;default:0"`
	CallCount          int64          `json:"callCount" gorm:"not null;default:0"`
	ConnectedCallCount int64          `json:"connectedCallCount" gorm:"not null;default:0"`
	BilledMinutes      int64          `json:"billedMinutes" gorm:"not null;default:0"`
	InboundCallCount   int64          `json:"inboundCallCount" gorm:"not null;default:0"`
	OutboundCallCount  int64          `json:"outboundCallCount" gorm:"not null;default:0"`
	AIToHumanCount     int64          `json:"aiToHumanCount" gorm:"not null;default:0"`
	AnalysisCount      int64          `json:"analysisCount" gorm:"not null;default:0"`
	UsageDetail        datatypes.JSON `json:"usageDetail,omitempty" gorm:"type:json"`
	FinalizedAt        *time.Time     `json:"finalizedAt,omitempty"`
	PaidAt             *time.Time     `json:"paidAt,omitempty"`
	GeneratedBy        string         `json:"generatedBy,omitempty" gorm:"size:128"`
}

func (TenantBill) TableName() string {
	return constants2.TENANT_BILL_TABLE_NAME
}

// TenantBillListFilter scopes bill list queries.
type TenantBillListFilter struct {
	TenantID uint
	Status   string
	Period   string // YYYY-MM
	From     *time.Time
	To       *time.Time
}

// TenantBillDirectionUsage is per-direction usage in usageDetail JSON.
type TenantBillDirectionUsage struct {
	Direction     string `json:"direction"`
	CallCount     int64  `json:"callCount"`
	BilledMinutes int64  `json:"billedMinutes"`
}

// TenantBillDailyUsage is per-day usage in usageDetail JSON.
type TenantBillDailyUsage struct {
	Day           string `json:"day"`
	CallCount     int64  `json:"callCount"`
	BilledMinutes int64  `json:"billedMinutes"`
}

// TenantBillUsageDetail is persisted on tenant_bills.usage_detail.
type TenantBillUsageDetail struct {
	Direction []TenantBillDirectionUsage `json:"direction,omitempty"`
	Daily     []TenantBillDailyUsage     `json:"daily,omitempty"`
}

type tenantBillUsageAgg struct {
	CallCount          int64
	ConnectedCallCount int64
	BilledMinutes      int64
	InboundCallCount   int64
	OutboundCallCount  int64
	AIToHumanCount     int64
}

type tenantBillDirectionAggRow struct {
	Direction        string `gorm:"column:direction"`
	CallCount        int64  `gorm:"column:call_count"`
	BilledMinutesSum int64  `gorm:"column:billed_minutes_sum"`
}

type tenantBillDailyAggRow struct {
	Day              string `gorm:"column:day"`
	CallCount        int64  `gorm:"column:call_count"`
	BilledMinutesSum int64  `gorm:"column:billed_minutes_sum"`
}

// ParseBillingPeriod parses YYYY-MM into [start, endExclusive) in business timezone.
func ParseBillingPeriod(period string) (time.Time, time.Time, error) {
	period = strings.TrimSpace(period)
	if len(period) != 7 || period[4] != '-' {
		return time.Time{}, time.Time{}, ErrTenantBillInvalidPeriod
	}
	loc := timeutil.Location()
	start, err := time.ParseInLocation("2006-01", period, loc)
	if err != nil {
		return time.Time{}, time.Time{}, ErrTenantBillInvalidPeriod
	}
	start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, loc)
	endExclusive := start.AddDate(0, 1, 0)
	return start, endExclusive, nil
}

// ListTenantBills returns paginated bills.
func ListTenantBills(db *gorm.DB, page, size int, f TenantBillListFilter) ([]TenantBill, int64, error) {
	if db == nil {
		return nil, 0, fmt.Errorf("db is nil")
	}
	q := db.Model(&TenantBill{})
	if f.TenantID > 0 {
		q = q.Where("tenant_id = ?", f.TenantID)
	}
	if status := strings.TrimSpace(f.Status); status != "" {
		q = q.Where("status = ?", status)
	}
	if period := strings.TrimSpace(f.Period); period != "" {
		start, endExclusive, err := ParseBillingPeriod(period)
		if err != nil {
			return nil, 0, err
		}
		q = q.Where("period_start = ?", start).Where("period_end = ?", endExclusive.Add(-time.Nanosecond))
	}
	if f.From != nil {
		q = q.Where("period_start >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("period_start <= ?", *f.To)
	}
	return utils.FindPage[TenantBill](q, page, size, "period_start DESC, id DESC", utils.MaxPageSize100)
}

// GetTenantBill loads one bill by id.
func GetTenantBill(db *gorm.DB, id uint) (TenantBill, error) {
	var row TenantBill
	err := db.First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return row, ErrTenantBillNotFound
	}
	return row, err
}

// FinalizeTenantBill locks a draft bill.
func FinalizeTenantBill(db *gorm.DB, id uint, operator string) (TenantBill, error) {
	row, err := GetTenantBill(db, id)
	if err != nil {
		return row, err
	}
	if row.Status != constants.TenantBillStatusDraft {
		return row, ErrTenantBillAlreadyLocked
	}
	now := timeutil.Now()
	updates := map[string]any{
		"status":       constants.TenantBillStatusFinalized,
		"finalized_at": now,
		"update_by":    strings.TrimSpace(operator),
	}
	if err := db.Model(&row).Updates(updates).Error; err != nil {
		return row, err
	}
	row.Status = constants.TenantBillStatusFinalized
	row.FinalizedAt = &now
	return row, nil
}

// MarkTenantBillPaid sets status to paid (requires finalized).
func MarkTenantBillPaid(db *gorm.DB, id uint, operator string) (TenantBill, error) {
	row, err := GetTenantBill(db, id)
	if err != nil {
		return row, err
	}
	if row.Status == constants.TenantBillStatusPaid {
		return row, ErrTenantBillAlreadyPaid
	}
	if row.Status != constants.TenantBillStatusFinalized {
		return row, ErrTenantBillNotFinalized
	}
	now := timeutil.Now()
	updates := map[string]any{
		"status":    constants.TenantBillStatusPaid,
		"paid_at":   now,
		"update_by": strings.TrimSpace(operator),
	}
	if err := db.Model(&row).Updates(updates).Error; err != nil {
		return row, err
	}
	row.Status = constants.TenantBillStatusPaid
	row.PaidAt = &now
	return row, nil
}

func aggregateTenantBillUsage(db *gorm.DB, tenantID uint, start, endExclusive time.Time) (tenantBillUsageAgg, TenantBillUsageDetail, error) {
	var zero tenantBillUsageAgg
	var detail TenantBillUsageDetail
	if db == nil {
		return zero, detail, gorm.ErrInvalidDB
	}
	type agg struct {
		CallCount     int64
		BilledMinutes int64
	}
	var row agg
	if err := db.Model(&TenantUsageEvent{}).
		Where("tenant_id = ? AND created_at >= ? AND created_at < ?", tenantID, start, endExclusive).
		Select("COUNT(*) as call_count, COALESCE(SUM(billed_minutes), 0) as billed_minutes").
		Scan(&row).Error; err != nil {
		return zero, detail, err
	}
	usage := tenantBillUsageAgg{
		CallCount:        row.CallCount,
		ConnectedCallCount: row.CallCount,
		BilledMinutes:    row.BilledMinutes,
	}
	return usage, detail, nil
}

func countTenantAnalysesInPeriod(_ *gorm.DB, _ uint, _, _ time.Time) (int64, error) {
	return 0, nil
}

// EnsureTenantBillsUpToDate generates finalized monthly bills for all completed months through last month.
func EnsureTenantBillsUpToDate(db *gorm.DB, tenantID uint) error {
	if db == nil || tenantID == 0 {
		return nil
	}
	var tenant Tenant
	if err := db.First(&tenant, tenantID).Error; err != nil {
		return err
	}
	loc := timeutil.Location()
	now := timeutil.Now().In(loc)
	lastMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, -1, 0)
	cursor := time.Date(tenant.CreatedAt.In(loc).Year(), tenant.CreatedAt.In(loc).Month(), 1, 0, 0, 0, 0, loc)
	for !cursor.After(lastMonthStart) {
		if err := AutoGenerateTenantBillForPeriod(db, tenantID, fmt.Sprintf("%04d-%02d", cursor.In(timeutil.Location()).Year(), int(cursor.In(timeutil.Location()).Month()))); err != nil {
			return err
		}
		cursor = cursor.AddDate(0, 1, 0)
	}
	return nil
}

// EnsureAllTenantsBillsUpToDate runs EnsureTenantBillsUpToDate for every active tenant.
func EnsureAllTenantsBillsUpToDate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	var ids []uint
	if err := db.Model(&Tenant{}).Where("status = ?", constants.TenantStatusActive).Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if err := EnsureTenantBillsUpToDate(db, id); err != nil {
			return err
		}
	}
	return nil
}

// AutoGenerateTenantBillForPeriod creates or refreshes a finalized bill for one calendar month.
func AutoGenerateTenantBillForPeriod(db *gorm.DB, tenantID uint, period string) error {
	start, endExclusive, err := ParseBillingPeriod(period)
	if err != nil {
		return err
	}
	loc := timeutil.Location()
	now := timeutil.Now().In(loc)
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
	if !start.Before(thisMonth) {
		return nil
	}

	var existing TenantBill
	findErr := db.Where("tenant_id = ? AND period_start = ?", tenantID, start).First(&existing).Error
	if findErr == nil && existing.Status != constants.TenantBillStatusDraft {
		return nil
	}

	bill, err := upsertTenantBillForPeriod(db, tenantID, start, endExclusive, "system")
	if err != nil {
		return err
	}
	nowTS := timeutil.Now()
	return db.Model(&bill).Updates(map[string]any{
		"status":       constants.TenantBillStatusFinalized,
		"finalized_at": nowTS,
		"generated_by": "system",
	}).Error
}

func upsertTenantBillForPeriod(db *gorm.DB, tenantID uint, start, endExclusive time.Time, operator string) (TenantBill, error) {
	if db == nil {
		return TenantBill{}, fmt.Errorf("db is nil")
	}
	endInclusive := endExclusive.Add(-time.Nanosecond)

	var tenant Tenant
	if err := db.First(&tenant, tenantID).Error; err != nil {
		return TenantBill{}, err
	}

	usage, detail, err := aggregateTenantBillUsage(db, tenantID, start, endExclusive)
	if err != nil {
		return TenantBill{}, err
	}
	analysisCount, err := countTenantAnalysesInPeriod(db, tenantID, start, endExclusive)
	if err != nil {
		return TenantBill{}, err
	}
	detailJSON, err := json.Marshal(detail)
	if err != nil {
		return TenantBill{}, err
	}

	currency := strings.TrimSpace(tenant.BillingCurrency)
	if currency == "" {
		currency = constants.TenantBillCurrencyCNY
	}

	row := TenantBill{
		TenantID:           tenantID,
		BillNo:             fmt.Sprintf("B%d-%s", tenantID, strings.ReplaceAll(fmt.Sprintf("%04d-%02d", start.In(timeutil.Location()).Year(), int(start.In(timeutil.Location()).Month())), "-", "")),
		PeriodStart:        start,
		PeriodEnd:          endInclusive,
		Status:             constants.TenantBillStatusDraft,
		Currency:           currency,
		TotalAmount:        computeBillTotalAmount(tenant, usage.BilledMinutes),
		CallCount:          usage.CallCount,
		ConnectedCallCount: usage.ConnectedCallCount,
		BilledMinutes:      usage.BilledMinutes,
		InboundCallCount:   usage.InboundCallCount,
		OutboundCallCount:  usage.OutboundCallCount,
		AIToHumanCount:     usage.AIToHumanCount,
		AnalysisCount:      analysisCount,
		UsageDetail:        datatypes.JSON(detailJSON),
		GeneratedBy:        strings.TrimSpace(operator),
	}

	var existing TenantBill
	findErr := db.Where("tenant_id = ? AND period_start = ?", tenantID, start).First(&existing).Error
	if findErr == nil && existing.ID > 0 {
		row.SetUpdateInfo(operator)
		if err := db.Model(&existing).Updates(map[string]any{
			"bill_no":              row.BillNo,
			"period_end":           row.PeriodEnd,
			"currency":             row.Currency,
			"total_amount":         row.TotalAmount,
			"call_count":           row.CallCount,
			"connected_call_count": row.ConnectedCallCount,
			"billed_minutes":       row.BilledMinutes,
			"inbound_call_count":   row.InboundCallCount,
			"outbound_call_count":  row.OutboundCallCount,
			"ai_to_human_count":    row.AIToHumanCount,
			"analysis_count":       row.AnalysisCount,
			"usage_detail":         row.UsageDetail,
			"generated_by":         row.GeneratedBy,
			"update_by":            row.UpdateBy,
		}).Error; err != nil {
			return TenantBill{}, err
		}
		return GetTenantBill(db, existing.ID)
	}

	row.SetCreateInfo(operator)
	if err := db.Create(&row).Error; err != nil {
		return TenantBill{}, err
	}
	return row, nil
}

func computeBillTotalAmount(t Tenant, billedMinutes int64) float64 {
	if t.BillingUnlimited {
		return 0
	}
	if tenantBillingMode(t.BillingMode) != constants.TenantBillingModePostpaid {
		return 0
	}
	if t.BillingRatePerMinute <= 0 || billedMinutes <= 0 {
		return 0
	}
	return float64(billedMinutes) * t.BillingRatePerMinute
}
