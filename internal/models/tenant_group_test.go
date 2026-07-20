package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestListTenantGroupsForTenantEmpty(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TenantGroup{}); err != nil {
		t.Fatal(err)
	}
	list, err := ListTenantGroupsForTenant(db, 1)
	if err != nil || len(list) != 0 {
		t.Fatalf("len=%d err=%v", len(list), err)
	}
}
