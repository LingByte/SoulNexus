package models

import (
	"context"
	"testing"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestTryAdmitPrepaidConcurrentHeld(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Tenant{}); err != nil {
		t.Fatal(err)
	}
	tenant := Tenant{
		BaseModel:               common.BaseModel{ID: 101},
		Name:                    "t",
		Slug:                    "t",
		Status:                  constants.TenantStatusActive,
		BillingMode:             constants.TenantBillingModePrepaid,
		BillingUnlimited:        false,
		PrepaidMinutesRemaining: 1,
	}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatal(err)
	}
	billingDB = db
	billingBackend = newMemoryReserveBackend()
	ctx := context.Background()
	if err := TryAdmitCall(ctx, tenant.ID, "call-a"); err != nil {
		t.Fatalf("first admit: %v", err)
	}
	if err := TryAdmitCall(ctx, tenant.ID, "call-b"); err != ErrInsufficientBalance {
		t.Fatalf("second admit err=%v want insufficient", err)
	}
	ReleaseCall(ctx, "call-a")
	if err := TryAdmitCall(ctx, tenant.ID, "call-c"); err != nil {
		t.Fatalf("after release: %v", err)
	}
}

func TestSettlePrepaidDoesNotGoNegative(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Tenant{}, &TenantUsageEvent{}); err != nil {
		t.Fatal(err)
	}
	tenant := Tenant{
		BaseModel:               common.BaseModel{ID: 102},
		Name:                    "t",
		Slug:                    "t2",
		Status:                  constants.TenantStatusActive,
		BillingMode:             constants.TenantBillingModePrepaid,
		BillingUnlimited:        false,
		PrepaidMinutesRemaining: 2,
	}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatal(err)
	}
	billingDB = db
	billingBackend = newMemoryReserveBackend()
	ctx := context.Background()
	if err := TryAdmitCall(ctx, tenant.ID, "c1"); err != nil {
		t.Fatal(err)
	}
	if err := SettleCall(ctx, db, tenant.ID, "c1", 300); err != nil { // 5 billed minutes
		t.Fatal(err)
	}
	var row Tenant
	if err := db.First(&row, tenant.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.PrepaidMinutesRemaining != 0 {
		t.Fatalf("remaining=%d want 0 (floored)", row.PrepaidMinutesRemaining)
	}
}
