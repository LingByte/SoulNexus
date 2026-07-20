package models

import (
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestToOperationLogMineView(t *testing.T) {
	now := time.Now()
	row := OperationLog{
		BaseModel: common.BaseModel{ID: 1, CreatedAt: now},
		TenantID:  2, Operator: "op", Action: "create", Resource: "trunk",
		ResourceID: 9, DetailJSON: "{}", ClientIP: "127.0.0.1",
	}
	v := ToOperationLogMineView(row)
	if v.ID != 1 || v.Action != "create" || v.Resource != "trunk" {
		t.Fatalf("%+v", v)
	}
}

func TestWriteOperationLogNilDB(t *testing.T) {
	WriteOperationLog(nil, OperationLogInput{Action: "test"})
}

func TestWriteOperationLogFromEvent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&OperationLog{}); err != nil {
		t.Fatal(err)
	}
	WriteOperationLogFromEvent(db, OperationLogEvent{Action: "unit", Operator: "tester"})
	var n int64
	if err := db.Model(&OperationLog{}).Count(&n).Error; err != nil || n != 1 {
		t.Fatalf("count=%d err=%v", n, err)
	}
}

func TestEmitOperationLogNilDB(t *testing.T) {
	EmitOperationLog(nil, OperationLogEvent{Action: "noop"})
}
