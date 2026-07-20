package models

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSaveTenantCallStatsDaily_nil(t *testing.T) {
	if err := SaveTenantCallStatsDaily(nil, &TenantCallStatsDaily{}); err != nil {
		t.Fatal(err)
	}
	if err := SaveTenantCallStatsDaily(nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestSaveTenantCallStatsDaily_duplicateUpsert(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TenantCallStatsDaily{}); err != nil {
		t.Fatal(err)
	}
	day := mustDay("2026-07-09")
	row1 := &TenantCallStatsDaily{TenantID: 42, StatDate: day, CallCount: 1}
	if err := SaveTenantCallStatsDaily(db, row1); err != nil {
		t.Fatalf("first save: %v", err)
	}
	row2 := &TenantCallStatsDaily{TenantID: 42, StatDate: day, CallCount: 9}
	if err := SaveTenantCallStatsDaily(db, row2); err != nil {
		t.Fatalf("dup save: %v", err)
	}
	got, err := GetTenantCallStatsDaily(db, 42, day)
	if err != nil {
		t.Fatal(err)
	}
	if got.CallCount != 9 {
		t.Fatalf("CallCount=%d want 9", got.CallCount)
	}
	var n int64
	db.Model(&TenantCallStatsDaily{}).Count(&n)
	if n != 1 {
		t.Fatalf("rows=%d want 1", n)
	}
}

func TestReplaceTenantCallStatsDaily_onConflict(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TenantCallStatsDaily{}); err != nil {
		t.Fatal(err)
	}
	day := mustDay("2026-07-16")
	if err := ReplaceTenantCallStatsDaily(db, &TenantCallStatsDaily{TenantID: 7, StatDate: day, CallCount: 3}); err != nil {
		t.Fatalf("first replace: %v", err)
	}
	if err := ReplaceTenantCallStatsDaily(db, &TenantCallStatsDaily{TenantID: 7, StatDate: day, CallCount: 11}); err != nil {
		t.Fatalf("second replace: %v", err)
	}
	got, err := GetTenantCallStatsDaily(db, 7, day)
	if err != nil {
		t.Fatal(err)
	}
	if got.CallCount != 11 {
		t.Fatalf("CallCount=%d want 11", got.CallCount)
	}
	var n int64
	db.Model(&TenantCallStatsDaily{}).Count(&n)
	if n != 1 {
		t.Fatalf("rows=%d want 1", n)
	}
}

func TestListTenantCallStatsDaily_empty(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TenantCallStatsDaily{}); err != nil {
		t.Fatal(err)
	}
	rows, err := ListTenantCallStatsDaily(db, 1, mustDay("2026-01-01"), mustDay("2026-01-02"))
	if err != nil || len(rows) != 0 {
		t.Fatalf("rows=%d err=%v", len(rows), err)
	}
}

func mustDay(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}
