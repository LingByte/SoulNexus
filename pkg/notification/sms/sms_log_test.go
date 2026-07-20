package sms

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openTestSMSDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&SMSLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestCreateSMSLog(t *testing.T) {
	t.Parallel()
	db := openTestSMSDB(t)
	row, err := CreateSMSLog(db, 42, "yunpian", "primary", "+8613800138000", "T1", "hello", "msg-1", SmsStatusAccepted, `{"ok":true}`, "127.0.0.1")
	if err != nil {
		t.Fatalf("CreateSMSLog: %v", err)
	}
	if row.ID == 0 || row.UserID != 42 || row.Status != SmsStatusAccepted || row.MessageID != "msg-1" {
		t.Errorf("row = %+v", row)
	}
	if row.SentAt.IsZero() {
		t.Error("SentAt should be set")
	}
}

func TestCreateFailedSMSLog(t *testing.T) {
	t.Parallel()
	db := openTestSMSDB(t)
	row, err := CreateFailedSMSLog(db, 1, "multi", "ch1", "+861", "tpl", "body", "all failed", "", "10.0.0.1")
	if err != nil {
		t.Fatalf("CreateFailedSMSLog: %v", err)
	}
	if row.Status != SmsStatusFailed || row.ErrorMsg != "all failed" {
		t.Errorf("row = %+v", row)
	}
	if !row.SentAt.IsZero() {
		t.Error("failed log should have zero SentAt")
	}
}

func TestSMSLogTableName(t *testing.T) {
	t.Parallel()
	if (SMSLog{}).TableName() != "sms_logs" {
		t.Error("unexpected table name")
	}
}
