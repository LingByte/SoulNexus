package models

import (
	"testing"

	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestTenantAllowsInboundCall(t *testing.T) {
	if !TenantAllowsInboundCall(Tenant{BillingUnlimited: true}) {
		t.Fatal("unlimited should allow")
	}
	if TenantAllowsInboundCall(Tenant{BillingMode: constants.TenantBillingModePrepaid, PrepaidMinutesRemaining: 0}) {
		t.Fatal("prepaid zero should block")
	}
}

func TestTenantBillingAccountFrom(t *testing.T) {
	acct := TenantBillingAccountFrom(Tenant{
		BillingMode:             constants.TenantBillingModePrepaid,
		PrepaidMinutesRemaining: 42,
	})
	if acct.BillingMode != constants.TenantBillingModePrepaid || acct.PrepaidMinutesRemaining != 42 {
		t.Fatalf("%+v", acct)
	}
}

func TestPatchTenantBilling_invalidMode(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Tenant{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&Tenant{Name: "t", Slug: "t", Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	bad := "invalid"
	if err := PatchTenantBilling(db, 1, TenantBillingPatch{BillingMode: &bad}, "test"); err == nil {
		t.Fatal("expected invalid billing mode error")
	}
}
