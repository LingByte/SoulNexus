package models

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestListTenantUsageEventsPage_empty(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TenantUsageEvent{}); err != nil {
		t.Fatal(err)
	}
	rows, total, err := ListTenantUsageEventsPage(db, 1, 10, TenantUsageEventListFilter{})
	if err != nil || total != 0 || len(rows) != 0 {
		t.Fatalf("rows=%d total=%d err=%v", len(rows), total, err)
	}
}

func TestListTenantUsageEventsPage_filter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TenantUsageEvent{}); err != nil {
		t.Fatal(err)
	}
	evt := TenantUsageEvent{TenantID: 1, CallID: "c1", DurationSec: 60, BilledMinutes: 1}
	if err := db.Create(&evt).Error; err != nil {
		t.Fatal(err)
	}
	rows, total, err := ListTenantUsageEventsPage(db, 1, 10, TenantUsageEventListFilter{TenantID: 1})
	if err != nil || total != 1 || len(rows) != 1 {
		t.Fatalf("rows=%d total=%d err=%v", len(rows), total, err)
	}
	_ = time.Now()
}
