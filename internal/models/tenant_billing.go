package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	constants2 "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"github.com/LingByte/SoulNexus/pkg/utils/timeutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TenantUsageEvent records one metered call (idempotent by call_id).
type TenantUsageEvent struct {
	common.BaseModel
	TenantID        uint    `json:"tenantId" gorm:"index;not null"`
	CallID          string  `json:"callId" gorm:"size:128;uniqueIndex;not null"`
	DurationSec     int     `json:"durationSec" gorm:"not null;default:0"`
	BilledMinutes   int64   `json:"billedMinutes" gorm:"not null;default:0"`
	MinutesDeducted int64   `json:"minutesDeducted" gorm:"not null;default:0"`
	AmountCharged   float64 `json:"amountCharged" gorm:"type:decimal(16,4);not null;default:0"`
}

func (TenantUsageEvent) TableName() string {
	return constants2.TENANT_USAGE_EVENT_TABLE_NAME
}

// TenantBillingAccount is the JSON view of tenant billing state.
type TenantBillingAccount struct {
	TenantID                uint    `json:"tenantId"`
	BillingMode             string  `json:"billingMode"`
	BillingUnlimited        bool    `json:"billingUnlimited"`
	PrepaidMinutesRemaining int64   `json:"prepaidMinutesRemaining"`
	RemainingMinutesDisplay string  `json:"remainingMinutesDisplay"`
	BillingRatePerMinute    float64 `json:"billingRatePerMinute"`
	BillingCurrency         string  `json:"billingCurrency"`
	MeteredBilledMinutes    int64   `json:"meteredBilledMinutes"`
	MeteredCallCount        int64   `json:"meteredCallCount"`
}

// TenantAllowsInboundCall reports whether billing policy permits starting a call (quick check without reservation).
func TenantAllowsInboundCall(t Tenant) bool {
	if t.BillingUnlimited {
		return true
	}
	if tenantBillingMode(t.BillingMode) != constants.TenantBillingModePrepaid {
		return true
	}
	return t.PrepaidMinutesRemaining > 0
}

func tenantBillingMode(mode string) string {
	return strings.TrimSpace(strings.ToLower(mode))
}

// TenantBillingAccountFrom builds API payload for consoles.
func TenantBillingAccountFrom(t Tenant) TenantBillingAccount {
	currency := strings.TrimSpace(t.BillingCurrency)
	if currency == "" {
		currency = constants.TenantBillCurrencyCNY
	}
	mode := tenantBillingMode(t.BillingMode)
	if mode == "" {
		mode = constants.TenantBillingModePrepaid
	}
	display := constants.TenantBillingBalanceUnlimitedLabel
	if !t.BillingUnlimited {
		switch mode {
		case constants.TenantBillingModePrepaid:
			display = fmt.Sprintf("%d", t.PrepaidMinutesRemaining)
		case constants.TenantBillingModePostpaid:
			display = constants.TenantBillingModePostpaid
		}
	}
	return TenantBillingAccount{
		TenantID:                t.ID,
		BillingMode:             mode,
		BillingUnlimited:        t.BillingUnlimited,
		PrepaidMinutesRemaining: t.PrepaidMinutesRemaining,
		RemainingMinutesDisplay: display,
		BillingRatePerMinute:    t.BillingRatePerMinute,
		BillingCurrency:         currency,
		MeteredBilledMinutes:    t.MeteredBilledMinutes,
		MeteredCallCount:        t.MeteredCallCount,
	}
}

// RecordTenantCallUsage meters one ended call and applies prepaid minute deduction when configured.
func RecordTenantCallUsage(db *gorm.DB, tenantID uint, callID string, durationSec int) error {
	if db == nil || tenantID == 0 || strings.TrimSpace(callID) == "" {
		return nil
	}
	billedMinutes := billedMinutesFromDurationSec(durationSec)
	if billedMinutes < 0 {
		billedMinutes = 0
	}

	return db.Transaction(func(tx *gorm.DB) error {
		ev := TenantUsageEvent{
			TenantID:      tenantID,
			CallID:        strings.TrimSpace(callID),
			DurationSec:   durationSec,
			BilledMinutes: billedMinutes,
		}
		ev.SetCreateInfo("system")
		res := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "call_id"}}, DoNothing: true}).Create(&ev)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}

		var tenant Tenant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&tenant, tenantID).Error; err != nil {
			return err
		}

		minutesDeducted := int64(0)
		amountCharged := float64(0)
		mode := tenantBillingMode(tenant.BillingMode)
		if mode == "" {
			mode = constants.TenantBillingModePrepaid
		}

		if !tenant.BillingUnlimited && billedMinutes > 0 {
			switch mode {
			case constants.TenantBillingModePrepaid:
				if tenant.PrepaidMinutesRemaining <= 0 {
					minutesDeducted = 0
				} else if billedMinutes >= tenant.PrepaidMinutesRemaining {
					minutesDeducted = tenant.PrepaidMinutesRemaining
				} else {
					minutesDeducted = billedMinutes
				}
			case constants.TenantBillingModePostpaid:
				if tenant.BillingRatePerMinute > 0 {
					amountCharged = float64(billedMinutes) * tenant.BillingRatePerMinute
				}
			}
		}

		updates := map[string]any{
			"metered_billed_minutes": gorm.Expr("metered_billed_minutes + ?", billedMinutes),
			"metered_call_count":     gorm.Expr("metered_call_count + 1"),
			"updated_at":             timeutil.Now(),
		}
		if minutesDeducted > 0 {
			updates["prepaid_minutes_remaining"] = gorm.Expr("prepaid_minutes_remaining - ?", minutesDeducted)
		}
		if err := tx.Model(&tenant).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Model(&ev).Updates(map[string]any{
			"minutes_deducted": minutesDeducted,
			"amount_charged":   amountCharged,
		}).Error
	})
}

// PatchTenantBilling updates platform-managed billing settings.
func PatchTenantBilling(db *gorm.DB, tenantID uint, patch TenantBillingPatch, updateBy string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		upd := map[string]any{
			"updated_at": timeutil.Now(),
			"update_by":  strings.TrimSpace(updateBy),
		}
		if patch.BillingMode != nil {
			mode := tenantBillingMode(*patch.BillingMode)
			if mode != constants.TenantBillingModePrepaid && mode != constants.TenantBillingModePostpaid {
				return fmt.Errorf("invalid billing mode: %s", *patch.BillingMode)
			}
			upd["billing_mode"] = mode
		}
		if patch.BillingUnlimited != nil {
			upd["billing_unlimited"] = *patch.BillingUnlimited
		}
		if patch.BillingRatePerMinute != nil {
			upd["billing_rate_per_minute"] = *patch.BillingRatePerMinute
		}
		if c := strings.TrimSpace(patch.BillingCurrency); c != "" {
			upd["billing_currency"] = c
		}
		if len(upd) > 2 {
			if err := tx.Model(&Tenant{}).Where("id = ?", tenantID).Updates(upd).Error; err != nil {
				return err
			}
		}
		if patch.RechargeMinutes != nil && *patch.RechargeMinutes != 0 {
			return tx.Model(&Tenant{}).Where("id = ?", tenantID).Updates(map[string]any{
				"prepaid_minutes_remaining": gorm.Expr("prepaid_minutes_remaining + ?", *patch.RechargeMinutes),
				"updated_at":                timeutil.Now(),
				"update_by":                 strings.TrimSpace(updateBy),
			}).Error
		}
		if patch.PrepaidMinutesRemaining != nil {
			return tx.Model(&Tenant{}).Where("id = ?", tenantID).Updates(map[string]any{
				"prepaid_minutes_remaining": *patch.PrepaidMinutesRemaining,
				"updated_at":                timeutil.Now(),
				"update_by":                 strings.TrimSpace(updateBy),
			}).Error
		}
		quota := TenantQuotaPatch{
			MaxConcurrentCalls: patch.MaxConcurrentCalls,
			DailyMinuteLimit:   patch.DailyMinuteLimit,
			MonthlyMinuteLimit: patch.MonthlyMinuteLimit,
			LicenseExpiresAt:   patch.LicenseExpiresAt,
			QuotaSuspended:     patch.QuotaSuspended,
			MaxUserCount:       patch.MaxUserCount,
		}
		return PatchTenantQuotas(tx, tenantID, quota, updateBy)
	})
}

// TenantBillingPatch is the platform billing settings payload.
type TenantBillingPatch struct {
	BillingMode             *string
	BillingUnlimited        *bool
	PrepaidMinutesRemaining *int64
	RechargeMinutes         *int64
	BillingRatePerMinute    *float64
	BillingCurrency         string
	MaxConcurrentCalls      *int
	DailyMinuteLimit        *int64
	MonthlyMinuteLimit      *int64
	LicenseExpiresAt        *time.Time
	QuotaSuspended          *bool
	MaxUserCount            *int
}

// ParseTenantBillUsageDetail unmarshals usage_detail JSON on a bill row.
func ParseTenantBillUsageDetail(raw []byte) (TenantBillUsageDetail, error) {
	var detail TenantBillUsageDetail
	if len(raw) == 0 {
		return detail, nil
	}
	err := json.Unmarshal(raw, &detail)
	return detail, err
}
