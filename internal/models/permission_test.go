package models

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestReplaceTenantRolePermissions_invalidID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Permission{}, &TenantRolePermission{}); err != nil {
		t.Fatal(err)
	}
	err = ReplaceTenantRolePermissions(db, 1, []uint{9999}, "op")
	if !errors.Is(err, ErrInvalidOrgReference) {
		t.Fatalf("err=%v", err)
	}
}

func TestSyncPermissionCatalog(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Permission{}); err != nil {
		t.Fatal(err)
	}
	if err := SyncPermissionCatalog(db); err != nil {
		t.Fatal(err)
	}
	var n int64
	if err := db.Model(&Permission{}).Count(&n).Error; err != nil || n == 0 {
		t.Fatalf("count=%d err=%v", n, err)
	}
}
