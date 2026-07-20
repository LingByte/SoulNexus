package models

import (
	"errors"
	"testing"

	"github.com/LingByte/SoulNexus/internal/constants"
	apperror "github.com/LingByte/SoulNexus/pkg/errors"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPlatformAdmin_TableName(t *testing.T) {
	if (PlatformAdmin{}).TableName() == "" {
		t.Fatal("empty table name")
	}
}

func TestListPlatformAdminsPage_nilDB(t *testing.T) {
	rows, total, err := ListPlatformAdminsPage(nil, 1, 10, "")
	if err != nil || rows != nil || total != 0 {
		t.Fatalf("rows=%v total=%d err=%v", rows, total, err)
	}
}

func TestUpdatePlatformAdminGuards(t *testing.T) {
	n, err := UpdatePlatformAdminStatus(nil, 0, constants.PlatformAdminStatusActive, "op")
	if err != nil || n != 0 {
		t.Fatalf("status n=%d err=%v", n, err)
	}
	n, err = UpdatePlatformAdminProfile(nil, 0, "a@b.c", "name", "op")
	if err != nil || n != 0 {
		t.Fatalf("profile n=%d err=%v", n, err)
	}
	if err := UpdatePlatformAdminPassword(nil, 0, "hash"); err != nil {
		t.Fatal(err)
	}
}

func TestGetActivePlatformAdminByGitHubID_empty(t *testing.T) {
	_, err := GetActivePlatformAdminByGitHubID(nil, "")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestPlatformAdminCRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&PlatformAdmin{}); err != nil {
		t.Fatal(err)
	}

	adm := PlatformAdmin{
		Email:        "admin@example.com",
		PasswordHash: "hash",
		DisplayName:  "Admin",
		Status:       constants.PlatformAdminStatusActive,
	}
	if err := db.Create(&adm).Error; err != nil {
		t.Fatal(err)
	}

	got, err := GetActivePlatformAdminByEmail(db, "admin@example.com")
	if err != nil || got.ID != adm.ID {
		t.Fatalf("by email id=%d err=%v", got.ID, err)
	}

	n, err := CountPlatformAdmins(db)
	if err != nil || n != 1 {
		t.Fatalf("count=%d err=%v", n, err)
	}

	if err := EnsureNotLastActivePlatformAdmin(db, adm.ID); !errors.Is(err, apperror.ErrLastActivePlatformAdmin) {
		t.Fatalf("last admin err=%v", err)
	}

	adm2 := PlatformAdmin{
		Email:        "admin2@example.com",
		PasswordHash: "hash",
		DisplayName:  "Admin2",
		Status:       constants.PlatformAdminStatusActive,
	}
	if err := db.Create(&adm2).Error; err != nil {
		t.Fatal(err)
	}
	if err := EnsureNotLastActivePlatformAdmin(db, adm.ID); err != nil {
		t.Fatal(err)
	}

	rows, total, err := ListPlatformAdminsPage(db, 1, 10, "admin")
	if err != nil || total < 2 || len(rows) < 2 {
		t.Fatalf("page len=%d total=%d err=%v", len(rows), total, err)
	}

	affected, err := UpdatePlatformAdminProfile(db, adm.ID, "new@example.com", "New Name", "op")
	if err != nil || affected != 1 {
		t.Fatalf("profile affected=%d err=%v", affected, err)
	}
}
