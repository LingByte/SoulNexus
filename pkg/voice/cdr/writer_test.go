// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cdr

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestWriter(t *testing.T) (*Writer, string) {
	t.Helper()
	dir := t.TempDir()
	w := NewWriter(Config{
		Dir:            dir,
		BaseName:       "cdr",
		BufSize:        32,
		MaxFileBytes:   10_000,
		RotateInterval: 200 * time.Millisecond,
	})
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return w, dir
}

func waitForWritten(t *testing.T, w *Writer, want uint64, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if got, _, _ := w.Stats(); got >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	got, _, _ := w.Stats()
	t.Fatalf("expected at least %d written, got %d", want, got)
}

func TestWriter_EmitAndReadBack(t *testing.T) {
	w, dir := newTestWriter(t)
	defer w.Stop()

	rec := NewCallRecord("call-001", "websocket", time.Now())
	rec.Scenario = "outbound"
	rec.Codec = "pcmu"
	rec.Finalize(time.Now().Add(30 * time.Second))
	rec.EndStatus = "ok"
	if !w.Emit(rec) {
		t.Fatal("Emit should succeed with empty buffer")
	}
	waitForWritten(t, w, 1, 2*time.Second)

	// Read back the live file. It's still the .current.jsonl since
	// we didn't trigger rotation.
	live := filepath.Join(dir, "cdr.current.jsonl")
	f, err := os.Open(live)
	if err != nil {
		t.Fatalf("open live file: %v", err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		t.Fatal("no line written")
	}
	var got CallRecord
	if err := json.Unmarshal(scanner.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON line: %v", err)
	}
	if got.CallID != "call-001" || got.Transport != "websocket" || got.EndStatus != "ok" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if got.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("schema version not set: %d", got.SchemaVersion)
	}
	if got.DurationMs <= 0 {
		t.Errorf("duration not computed: %d", got.DurationMs)
	}
}

func TestWriter_DropsOnFullBuffer(t *testing.T) {
	dir := t.TempDir()
	// Tiny buffer + no actual file write throughput shouldn't be
	// the bottleneck on a temp dir — but we make buf = 4 so even a
	// quick burst of 1000 should overflow at some point.
	w := NewWriter(Config{
		Dir:      dir,
		BaseName: "cdr",
		BufSize:  4,
	})
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	var dropped int
	for i := 0; i < 5000; i++ {
		r := NewCallRecord("c", "websocket", time.Now())
		r.Finalize(time.Now())
		if !w.Emit(r) {
			dropped++
		}
	}
	if dropped == 0 {
		t.Log("no drops observed (drain kept up); skipping strict assertion")
		return
	}
	_, statsDrop, _ := w.Stats()
	if statsDrop == 0 {
		t.Error("Stats.dropped should reflect at least one drop")
	}
}

func TestWriter_SizeBasedRotation(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(Config{
		Dir:            dir,
		BaseName:       "cdr",
		BufSize:        128,
		MaxFileBytes:   512, // tiny so a few records triggers rotation
		RotateInterval: time.Hour,
	})
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	for i := 0; i < 20; i++ {
		r := NewCallRecord("c", "websocket", time.Now())
		r.Finalize(time.Now())
		r.HangupReason = strings.Repeat("x", 80) // pad
		w.Emit(r)
	}
	// Allow drain + rotation.
	time.Sleep(200 * time.Millisecond)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var rotated int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") && !strings.Contains(e.Name(), "current") {
			rotated++
		}
	}
	if rotated == 0 {
		t.Errorf("expected at least one rotated file in %s, got entries: %v",
			dir, dirNames(entries))
	}
}

func TestWriter_StopRotatesFinalFile(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(Config{
		Dir:            dir,
		BaseName:       "cdr",
		BufSize:        16,
		MaxFileBytes:   1 << 20,
		RotateInterval: time.Hour,
	})
	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	r := NewCallRecord("c", "websocket", time.Now())
	r.Finalize(time.Now())
	w.Emit(r)
	waitForWritten(t, w, 1, time.Second)

	w.Stop()

	// After Stop, current.jsonl should be gone — renamed to a
	// timestamped file. (Empty .current.jsonl is deleted; non-empty
	// is renamed.)
	if _, err := os.Stat(filepath.Join(dir, "cdr.current.jsonl")); err == nil {
		t.Error("Stop must rotate-away the live file")
	}
	entries, _ := os.ReadDir(dir)
	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "cdr-") && strings.HasSuffix(e.Name(), ".jsonl") {
			found = true
		}
	}
	if !found {
		t.Errorf("no timestamped CDR file after Stop; entries: %v", dirNames(entries))
	}
}

func TestNewCallRecord_PopulatesSchema(t *testing.T) {
	start := time.Now()
	r := NewCallRecord("abc", "websocket", start)
	if r.SchemaVersion != CurrentSchemaVersion {
		t.Error("SchemaVersion must be set")
	}
	if r.CallID != "abc" {
		t.Error("CallID round-trip")
	}
	if r.StartEpoch != start.Unix() {
		t.Errorf("StartEpoch=%d want %d", r.StartEpoch, start.Unix())
	}
}

func TestCallRecord_FinalizeClampsNegativeDuration(t *testing.T) {
	r := CallRecord{StartTS: time.Now()}
	r.Finalize(r.StartTS.Add(-time.Second)) // end before start
	if r.DurationMs != 0 {
		t.Errorf("negative duration must clamp to 0, got %d", r.DurationMs)
	}
}

func TestWriter_NilSafe(t *testing.T) {
	var w *Writer
	if w.Emit(CallRecord{}) {
		t.Error("nil writer Emit must return false")
	}
	w.Stop()
	if a, b, c := w.Stats(); a != 0 || b != 0 || c != 0 {
		t.Errorf("nil writer Stats must return zeros; got (%d,%d,%d)", a, b, c)
	}
}

func dirNames(entries []os.DirEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}
