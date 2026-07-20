package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestListBillingPlansPage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&BillingPlan{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&BillingPlan{Name: "Basic", Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	rows, total, err := ListBillingPlansPage(db, 1, 10, "Basic")
	if err != nil || total != 1 || len(rows) != 1 {
		t.Fatalf("rows=%d total=%d err=%v", len(rows), total, err)
	}
}
