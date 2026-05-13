// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package persist

// End-to-end test for the new persistence tables (call_events,
// call_media_stats, call_recording). Boots an in-memory SQLite via
// glebarez/sqlite, runs Migrate(), inserts representative rows, then
// queries them back. The point is to lock down the schema contract so
// later refactors can't silently drop columns or rename tables.

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func freshDB(t *testing.T) *gorm.DB {
	t.Helper()
	// `:memory:` is per-connection; "file::memory:?cache=shared" gives a
	// shared handle but we don't need it here — one *gorm.DB owns one
	// connection in tests.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestMigrate_CreatesAllTables(t *testing.T) {
	db := freshDB(t)
	for _, m := range Models() {
		// Migrator.HasTable returns true for every successfully migrated
		// model; absence here means the row was added to Models() but
		// the AutoMigrate failed (column conflict, etc.).
		if !db.Migrator().HasTable(m) {
			t.Errorf("missing table for %T", m)
		}
	}
}

func TestAppendCallEvent_RoundTrip(t *testing.T) {
	db := freshDB(t)
	ctx := context.Background()
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := AppendCallEvent(ctx, db, "c-1", EventKindASRFinal, EventLevelInfo,
		at, datatypes.JSON([]byte(`{"text":"hi"}`))); err != nil {
		t.Fatalf("append: %v", err)
	}
	rows, err := ListCallEventsByCall(ctx, db, "c-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("len: %d", len(rows))
	}
	got := rows[0]
	if got.Kind != EventKindASRFinal {
		t.Errorf("kind: %s", got.Kind)
	}
	if got.Level != EventLevelInfo {
		t.Errorf("level: %s", got.Level)
	}
	if !got.At.Equal(at) {
		t.Errorf("at: %v want %v", got.At, at)
	}
	if string(got.Detail) != `{"text":"hi"}` {
		t.Errorf("detail: %s", string(got.Detail))
	}
}

func TestAppendCallEvent_NilDBIsNoOp(t *testing.T) {
	if err := AppendCallEvent(context.Background(), nil, "c-1", "k", "info", time.Now(), nil); err != nil {
		t.Fatalf("nil db should be silent: %v", err)
	}
}

func TestAppendCallMediaStats_LatestReturnsMostRecent(t *testing.T) {
	db := freshDB(t)
	ctx := context.Background()
	t0 := time.Now().Add(-time.Minute).UTC()
	t1 := t0.Add(30 * time.Second)
	for _, sample := range []*CallMediaStats{
		{CallID: "c-2", Transport: "webrtc", At: t0, RTTMs: 50, JitterMs: 5, PacketsReceived: 100},
		{CallID: "c-2", Transport: "webrtc", At: t1, RTTMs: 60, JitterMs: 8, PacketsReceived: 200, Final: true},
	} {
		if err := AppendCallMediaStats(ctx, db, sample); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	got, err := LatestCallMediaStats(ctx, db, "c-2")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Final || got.RTTMs != 60 || got.PacketsReceived != 200 {
		t.Errorf("expected final/RTT=60/PR=200; got %+v", got)
	}
}

func TestAppendCallRecording_ListByCall(t *testing.T) {
	db := freshDB(t)
	ctx := context.Background()
	for _, r := range []*CallRecording{
		{CallID: "c-3", Transport: "sip", Bucket: "b", Key: "a.wav",
			Format: "wav", Layout: RecordingLayoutStereoLR,
			SampleRate: 16000, Channels: 2, Bytes: 1024, DurationMs: 500},
		{CallID: "c-3", Transport: "sip", Bucket: "b", Key: "b.wav",
			Format: "wav", Layout: RecordingLayoutStereoLR,
			SampleRate: 16000, Channels: 2, Bytes: 2048, DurationMs: 1000},
	} {
		if err := AppendCallRecording(ctx, db, r); err != nil {
			t.Fatal(err)
		}
	}
	got, err := ListCallRecordings(ctx, db, "c-3")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len: %d", len(got))
	}
	if got[0].Key != "a.wav" || got[1].Key != "b.wav" {
		t.Errorf("order: %s, %s", got[0].Key, got[1].Key)
	}
}
