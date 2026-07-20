package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestTenantUserHasRoleNameMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TenantRole{}, &TenantUserRole{}); err != nil {
		t.Fatal(err)
	}
	ok, err := TenantUserHasRoleName(db, 1, "admin")
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
