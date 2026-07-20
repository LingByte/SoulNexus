package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"gorm.io/gorm"
)

var (
	ErrTenantQuotaExceeded  = errors.New("tenant quota exceeded")
	ErrTenantQuotaSuspended = errors.New("tenant quota suspended")
	ErrTenantLicenseExpired = errors.New("tenant license expired")
	ErrTenantUserLimit      = errors.New("tenant user limit reached")
)

// TenantQuotaSnapshot is the live quota consumption view for dashboards.
type TenantQuotaSnapshot struct {
	MaxConcurrentCalls int        `json:"maxConcurrentCalls"`
	DailyMinuteLimit   int64      `json:"dailyMinuteLimit"`
	MonthlyMinuteLimit int64      `json:"monthlyMinuteLimit"`
	DailyMinutesUsed   int64      `json:"dailyMinutesUsed"`
	MonthlyMinutesUsed int64      `json:"monthlyMinutesUsed"`
	HeldMinutes        int64      `json:"heldMinutes"`
	ActiveReservations int64      `json:"activeReservations"`
	QuotaSuspended     bool       `json:"quotaSuspended"`
	LicenseExpiresAt   *time.Time `json:"licenseExpiresAt,omitempty"`
	LicenseValid       bool       `json:"licenseValid"`
}

// TenantOperationalError returns nil when the tenant may place or receive calls.
func TenantOperationalError(t Tenant, now time.Time) error {
	if t.Status == constants.TenantStatusSuspended || t.QuotaSuspended {
		return ErrTenantQuotaSuspended
	}
	if t.LicenseExpiresAt != nil && !t.LicenseExpiresAt.After(now) {
		return ErrTenantLicenseExpired
	}
	return nil
}

// SumTenantBilledMinutesBetween aggregates metered minutes from usage events.
func SumTenantBilledMinutesBetween(db *gorm.DB, tenantID uint, start, endExclusive time.Time) (int64, error) {
	if db == nil || tenantID == 0 {
		return 0, nil
	}
	var total int64
	err := db.Model(&TenantUsageEvent{}).
		Where("tenant_id = ? AND created_at >= ? AND created_at < ?", tenantID, start, endExclusive).
		Select("COALESCE(SUM(billed_minutes), 0)").
		Scan(&total).Error
	return total, err
}

// TenantDailyMinutesUsed returns billed minutes consumed on the given local calendar day.
func TenantDailyMinutesUsed(db *gorm.DB, tenantID uint, day time.Time) (int64, error) {
	loc := day.Location()
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)
	return SumTenantBilledMinutesBetween(db, tenantID, start, end)
}

// TenantMonthlyMinutesUsed returns billed minutes consumed in the calendar month containing day.
func TenantMonthlyMinutesUsed(db *gorm.DB, tenantID uint, day time.Time) (int64, error) {
	loc := day.Location()
	start := time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 1, 0)
	return SumTenantBilledMinutesBetween(db, tenantID, start, end)
}

// CheckTenantMinuteQuotas verifies daily/monthly limits (0 = unlimited).
func CheckTenantMinuteQuotas(db *gorm.DB, t Tenant, now time.Time) error {
	if t.BillingUnlimited {
		return nil
	}
	if t.DailyMinuteLimit > 0 {
		used, err := TenantDailyMinutesUsed(db, t.ID, now)
		if err != nil {
			return err
		}
		if used >= t.DailyMinuteLimit {
			return fmt.Errorf("%w: daily minute limit", utils.ErrQuotaExceeded)
		}
	}
	if t.MonthlyMinuteLimit > 0 {
		used, err := TenantMonthlyMinutesUsed(db, t.ID, now)
		if err != nil {
			return err
		}
		if used >= t.MonthlyMinuteLimit {
			return fmt.Errorf("%w: monthly minute limit", utils.ErrQuotaExceeded)
		}
	}
	return nil
}

// SuspendTenantQuota marks the tenant as quota-suspended (blocks new calls).
func SuspendTenantQuota(db *gorm.DB, tenantID uint, updateBy string) error {
	if db == nil || tenantID == 0 {
		return nil
	}
	return db.Model(&Tenant{}).Where("id = ?", tenantID).Updates(map[string]any{
		"quota_suspended": true,
		"updated_at":      timeutil.Now(),
		"update_by":       updateBy,
	}).Error
}

// EnforceTenantUserLimit returns ErrTenantUserLimit when max members would be exceeded.
func EnforceTenantUserLimit(db *gorm.DB, tenantID uint) error {
	t, err := GetActiveTenantByID(db, tenantID)
	if err != nil {
		return err
	}
	if t.MaxUserCount <= 0 {
		return nil
	}
	n, err := CountTenantUsers(db, tenantID)
	if err != nil {
		return err
	}
	if int(n) >= t.MaxUserCount {
		return ErrTenantUserLimit
	}
	return nil
}

// BuildTenantQuotaSnapshot assembles quota usage for API responses.
func BuildTenantQuotaSnapshot(db *gorm.DB, t Tenant, heldMinutes, activeReservations int64) (TenantQuotaSnapshot, error) {
	now := timeutil.Now()
	daily, err := TenantDailyMinutesUsed(db, t.ID, now)
	if err != nil {
		return TenantQuotaSnapshot{}, err
	}
	monthly, err := TenantMonthlyMinutesUsed(db, t.ID, now)
	if err != nil {
		return TenantQuotaSnapshot{}, err
	}
	licenseValid := t.LicenseExpiresAt == nil || t.LicenseExpiresAt.After(now)
	return TenantQuotaSnapshot{
		MaxConcurrentCalls: t.MaxConcurrentCalls,
		DailyMinuteLimit:   t.DailyMinuteLimit,
		MonthlyMinuteLimit: t.MonthlyMinuteLimit,
		DailyMinutesUsed:   daily,
		MonthlyMinutesUsed: monthly,
		HeldMinutes:        heldMinutes,
		ActiveReservations: activeReservations,
		QuotaSuspended:     t.QuotaSuspended,
		LicenseExpiresAt:   t.LicenseExpiresAt,
		LicenseValid:       licenseValid,
	}, nil
}

// TenantQuotaPatch is the platform quota settings payload.
type TenantQuotaPatch struct {
	MaxConcurrentCalls *int
	DailyMinuteLimit   *int64
	MonthlyMinuteLimit *int64
	LicenseExpiresAt   *time.Time
	QuotaSuspended     *bool
	MaxUserCount       *int
}

// PatchTenantQuotas updates platform-managed quota fields.
func PatchTenantQuotas(db *gorm.DB, tenantID uint, patch TenantQuotaPatch, updateBy string) error {
	upd := map[string]any{
		"updated_at": timeutil.Now(),
		"update_by":  updateBy,
	}
	if patch.MaxConcurrentCalls != nil {
		upd["max_concurrent_calls"] = *patch.MaxConcurrentCalls
	}
	if patch.DailyMinuteLimit != nil {
		upd["daily_minute_limit"] = *patch.DailyMinuteLimit
	}
	if patch.MonthlyMinuteLimit != nil {
		upd["monthly_minute_limit"] = *patch.MonthlyMinuteLimit
	}
	if patch.LicenseExpiresAt != nil {
		upd["license_expires_at"] = patch.LicenseExpiresAt
	}
	if patch.QuotaSuspended != nil {
		upd["quota_suspended"] = *patch.QuotaSuspended
	}
	if patch.MaxUserCount != nil {
		upd["max_user_count"] = *patch.MaxUserCount
	}
	if len(upd) <= 2 {
		return nil
	}
	return db.Model(&Tenant{}).Where("id = ?", tenantID).Updates(upd).Error
}
