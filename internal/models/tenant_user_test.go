package models

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestGetAuthenticatedTenantUser_guards(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TenantUser{}); err != nil {
		t.Fatal(err)
	}
	_, err = GetAuthenticatedTenantUser(db, 0, 1)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("zero user err=%v", err)
	}
}

func TestCheckTenantUserPhoneExists_empty(t *testing.T) {
	ok, err := CheckTenantUserPhoneExists(nil, "", 0)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestRecordSuccessfulLoginCity_noop(t *testing.T) {
	RecordSuccessfulLoginCity(nil, 0, "Shanghai")
}

func TestCountTenantUsers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&TenantUser{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&TenantUser{TenantID: 1, Email: "u@example.com", Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	n, err := CountTenantUsers(db, 1)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}
