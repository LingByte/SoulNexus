package models

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRecordLoginHistory_andList(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&LoginHistory{}); err != nil {
		t.Fatal(err)
	}
	if err := RecordLoginHistory(db, LoginHistoryInput{
		PrincipalType: LoginHistoryPrincipalTenantUser,
		PrincipalID:   1,
		TenantID:      10,
		Email:         "a@example.com",
		ClientIP:      "127.0.0.1",
		LoginMethod:   "password",
		Success:       true,
	}); err != nil {
		t.Fatal(err)
	}
	list, total, err := ListLoginHistoryPage(db, LoginHistoryPrincipalTenantUser, 1, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("total=%d len=%d", total, len(list))
	}
}

func TestRecordLoginHistory_skipsInvalid(t *testing.T) {
	if err := RecordLoginHistory(nil, LoginHistoryInput{}); err != nil {
		t.Fatal(err)
	}
}
