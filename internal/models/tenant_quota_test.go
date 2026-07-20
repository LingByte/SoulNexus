package models

import (
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/internal/constants"
)

func TestTenantOperationalError(t *testing.T) {
	now := time.Now()
	if err := TenantOperationalError(Tenant{Status: constants.TenantStatusSuspended}, now); err != ErrTenantQuotaSuspended {
		t.Fatalf("suspended err=%v", err)
	}
	expired := now.Add(-time.Hour)
	if err := TenantOperationalError(Tenant{LicenseExpiresAt: &expired}, now); err != ErrTenantLicenseExpired {
		t.Fatalf("license err=%v", err)
	}
	if err := TenantOperationalError(Tenant{Status: constants.TenantStatusActive}, now); err != nil {
		t.Fatal(err)
	}
}

func TestSumTenantBilledMinutesBetween_nil(t *testing.T) {
	n, err := SumTenantBilledMinutesBetween(nil, 1, time.Now(), time.Now())
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
