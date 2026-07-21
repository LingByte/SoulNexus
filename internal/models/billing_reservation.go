package models

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AdmissionReserveMinutes is prepaid/postpaid admission hold (1 billed minute slot per active call).
const AdmissionReserveMinutes int64 = 1

const reservationTTL = 6 * time.Hour

var (
	ErrInsufficientBalance   = errors.New("insufficient prepaid minutes")
	ErrReservationNotFound   = errors.New("billing reservation not found")
	ErrTenantOperational     = errors.New("tenant not operational")
	ErrTenantConcurrentLimit = errors.New("tenant concurrent call limit reached")

	billingDB      *gorm.DB
	billingBackend reserveBackend
	billingInit    sync.Once
)

// InitBillingReservations constructs the process-wide call admission backend (Redis when REDIS_URL set, else in-memory).
func InitBillingReservations(db *gorm.DB) {
	billingInit.Do(func() {
		billingDB = db
		addr := strings.TrimSpace(utils.GetEnv("REDIS_URL"))
		var backend reserveBackend
		if addr != "" {
			if rdb, err := newRedisReserveBackend(addr); err == nil {
				backend = rdb
				if logger.Lg != nil {
					logger.Lg.Info("billing reservations using redis", zap.String("redis_url", addr))
				}
			} else if logger.Lg != nil {
				logger.Lg.Warn("billing redis unavailable, using in-memory reservations", zap.Error(err))
			}
		}
		if backend == nil {
			backend = newMemoryReserveBackend()
			if addr == "" && logger.Lg != nil {
				logger.Lg.Warn("REDIS_URL unset; billing reservations use in-memory store (single process only)")
			}
		}
		billingBackend = backend
	})
	if billingBackend == nil {
		billingBackend = newMemoryReserveBackend()
	}
}

func billingReserveBackend() reserveBackend {
	if billingBackend == nil {
		return newMemoryReserveBackend()
	}
	return billingBackend
}

// TryAdmitCall reserves capacity before accepting an inbound INVITE or outbound dial.
func TryAdmitCall(ctx context.Context, tenantID uint, callID string) error {
	if tenantID == 0 {
		return nil
	}
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return nil
	}
	db := billingDB
	if db == nil {
		return nil
	}
	t, err := GetActiveTenantByID(db, tenantID)
	if err != nil {
		return err
	}
	if err := TenantOperationalError(t, time.Now()); err != nil {
		return fmt.Errorf("%w: %v", ErrTenantOperational, err)
	}
	if err := CheckTenantMinuteQuotas(db, t, time.Now()); err != nil {
		return err
	}
	backend := billingReserveBackend()
	if t.BillingUnlimited {
		return admitWithConcurrentCap(ctx, db, backend, t, callID, nil)
	}
	mode := strings.TrimSpace(strings.ToLower(t.BillingMode))
	if mode == "" {
		mode = constants.TenantBillingModePrepaid
	}
	switch mode {
	case constants.TenantBillingModePrepaid:
		return tryAdmitPrepaid(ctx, db, backend, t, callID)
	case constants.TenantBillingModePostpaid:
		return tryAdmitPostpaid(ctx, db, backend, tenantID, callID)
	default:
		return admitWithConcurrentCap(ctx, db, backend, t, callID, nil)
	}
}

func admitWithConcurrentCap(ctx context.Context, db *gorm.DB, backend reserveBackend, t Tenant, callID string, afterHold func() error) error {
	unlock, err := backend.lockTenant(ctx, t.ID)
	if err != nil {
		return err
	}
	defer unlock()
	if t.MaxConcurrentCalls > 0 {
		held, err := backend.heldMinutes(ctx, t.ID)
		if err != nil {
			return err
		}
		if held >= int64(t.MaxConcurrentCalls) {
			return ErrTenantConcurrentLimit
		}
	}
	if afterHold != nil {
		return afterHold()
	}
	return backend.putReservation(ctx, t.ID, callID, AdmissionReserveMinutes)
}

func tryAdmitPrepaid(ctx context.Context, db *gorm.DB, backend reserveBackend, t Tenant, callID string) error {
	return admitWithConcurrentCap(ctx, db, backend, t, callID, func() error {
		if db != nil {
			fresh, err := GetActiveTenantByID(db, t.ID)
			if err != nil {
				return err
			}
			t = fresh
		}
		if t.PrepaidMinutesRemaining <= 0 {
			return ErrInsufficientBalance
		}
		held, err := backend.heldMinutes(ctx, t.ID)
		if err != nil {
			return err
		}
		if t.PrepaidMinutesRemaining-held < AdmissionReserveMinutes {
			return ErrInsufficientBalance
		}
		return backend.putReservation(ctx, t.ID, callID, AdmissionReserveMinutes)
	})
}

func tryAdmitPostpaid(ctx context.Context, db *gorm.DB, backend reserveBackend, tenantID uint, callID string) error {
	t, err := GetActiveTenantByID(db, tenantID)
	if err != nil {
		return err
	}
	return admitWithConcurrentCap(ctx, db, backend, t, callID, func() error {
		return backend.putReservation(ctx, tenantID, callID, AdmissionReserveMinutes)
	})
}

// ReleaseCall drops an admission hold when the call never completed billing settlement.
func ReleaseCall(ctx context.Context, callID string) {
	backend := billingReserveBackend()
	if backend == nil {
		return
	}
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	_ = backend.dropReservation(ctx, callID)
}

// SettleCall records usage and releases the admission hold.
func SettleCall(ctx context.Context, db *gorm.DB, tenantID uint, callID string, durationSec int) error {
	if db == nil {
		db = billingDB
	}
	if err := RecordTenantCallUsage(db, tenantID, callID, durationSec); err != nil {
		return err
	}
	_ = RecordAIPoolTransitUsage(db, tenantID, callID, durationSec)
	if backend := billingReserveBackend(); backend != nil {
		_ = backend.dropReservation(ctx, callID)
	}
	maybeAutoSuspendAfterUsage(db, tenantID)
	return nil
}

func maybeAutoSuspendAfterUsage(db *gorm.DB, tenantID uint) {
	if db == nil || tenantID == 0 {
		return
	}
	t, err := GetActiveTenantByID(db, tenantID)
	if err != nil || t.BillingUnlimited || t.QuotaSuspended {
		return
	}
	now := time.Now()
	if t.PrepaidMinutesRemaining <= 0 {
		mode := strings.TrimSpace(strings.ToLower(t.BillingMode))
		if mode == "" {
			mode = constants.TenantBillingModePrepaid
		}
		if mode == constants.TenantBillingModePrepaid {
			_ = SuspendTenantQuota(db, tenantID, "system:prepaid_exhausted")
			return
		}
	}
	if t.DailyMinuteLimit > 0 {
		if used, err := TenantDailyMinutesUsed(db, tenantID, now); err == nil && used >= t.DailyMinuteLimit {
			_ = SuspendTenantQuota(db, tenantID, "system:daily_limit")
		}
	}
	if t.MonthlyMinuteLimit > 0 {
		if used, err := TenantMonthlyMinutesUsed(db, tenantID, now); err == nil && used >= t.MonthlyMinuteLimit {
			_ = SuspendTenantQuota(db, tenantID, "system:monthly_limit")
		}
	}
}

// SyncTenantBalance refreshes cached balance after platform billing updates.
func SyncTenantBalance(ctx context.Context, tenantID uint) {
	if billingDB == nil || tenantID == 0 {
		return
	}
	t, err := GetActiveTenantByID(billingDB, tenantID)
	if err != nil {
		return
	}
	_ = billingReserveBackend().setBalanceHint(ctx, tenantID, t.PrepaidMinutesRemaining)
}

// HeldMinutesForTenant returns current admission hold minutes (approx active calls when hold=1min).
func HeldMinutesForTenant(ctx context.Context, tenantID uint) int64 {
	if tenantID == 0 {
		return 0
	}
	held, _ := billingReserveBackend().heldMinutes(ctx, tenantID)
	return held
}

type reserveBackend interface {
	lockTenant(ctx context.Context, tenantID uint) (unlock func(), err error)
	heldMinutes(ctx context.Context, tenantID uint) (int64, error)
	putReservation(ctx context.Context, tenantID uint, callID string, minutes int64) error
	dropReservation(ctx context.Context, callID string) error
	setBalanceHint(ctx context.Context, tenantID uint, balance int64) error
}

func tenantReserveKey(tenantID uint) string {
	return strconv.FormatUint(uint64(tenantID), 10)
}

func callReserveKey(callID string) string {
	return "billing:reserve:" + strings.TrimSpace(callID)
}

func tenantHeldKey(tenantID uint) string {
	return "billing:held:" + tenantReserveKey(tenantID)
}

func tenantAdmitLockKey(tenantID uint) string {
	return "billing:admit-lock:" + tenantReserveKey(tenantID)
}

func tenantBalanceKey(tenantID uint) string {
	return "billing:balance:" + tenantReserveKey(tenantID)
}

func reservationPayload(tenantID uint, minutes int64) string {
	return fmt.Sprintf("%d:%d", tenantID, minutes)
}

func noopUnlock() {}

func parseReservationPayload(raw string) (tenantID uint, minutes int64, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(raw), ":", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	tid, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || tid == 0 {
		return 0, 0, false
	}
	m, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || m <= 0 {
		return 0, 0, false
	}
	return uint(tid), m, true
}
