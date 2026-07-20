package models

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestBuildAssistantStatsPayload_nil(t *testing.T) {
	if got := buildAssistantStatsPayload(nil); got.TotalCalls != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestAvgSec(t *testing.T) {
	if avgSec(100, 0) != 0 {
		t.Fatal("zero count")
	}
	if avgSec(100, 4) != 25 {
		t.Fatal("average")
	}
}

func TestCollectTenantDailyStats_nilDB(t *testing.T) {
	if err := CollectTenantDailyStats(context.Background(), nil, 1, time.Now()); err != gorm.ErrInvalidDB {
		t.Fatalf("err=%v", err)
	}
}
