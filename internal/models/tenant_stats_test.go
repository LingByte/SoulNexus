package models

import (
	"testing"

	"gorm.io/gorm"
)

func TestEncodeDecodeStatsPayload(t *testing.T) {
	in := CalloutStatsPayload{TotalAttempts: 5, CallerNumber: "138"}
	raw := EncodeStatsPayload(in)
	var out CalloutStatsPayload
	DecodeStatsPayload(raw, &out)
	if out.TotalAttempts != 5 || out.CallerNumber != "138" {
		t.Fatalf("%+v", out)
	}
}

func TestSaveTenantStats_nilGuards(t *testing.T) {
	if err := SaveTenantStats(nil, &TenantStats{}); err != gorm.ErrInvalidDB {
		t.Fatalf("nil db err=%v", err)
	}
	if err := SaveTenantStats(nil, nil); err != gorm.ErrInvalidDB {
		t.Fatalf("nil row err=%v", err)
	}
}

func TestTenantStats_TableName(t *testing.T) {
	if (TenantStats{}).TableName() != "tenant_stats" {
		t.Fatal("table name")
	}
}
