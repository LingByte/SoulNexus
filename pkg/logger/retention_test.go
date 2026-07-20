package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestPurgeExpiredLogFiles(t *testing.T) {
	dir := t.TempDir()

	oldPath := filepath.Join(dir, "app-2020-01-01.log")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().AddDate(0, 0, -10)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	newPath := filepath.Join(dir, "app-today.log")
	if err := os.WriteFile(newPath, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	removed, err := PurgeExpiredLogFiles(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 removed, got %d", removed)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatal("expected old log to be removed")
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Fatalf("expected new log to remain: %v", err)
	}
}

func TestPurgeExpiredLogFilesDisabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keep.log")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().AddDate(0, 0, -30)
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	removed, err := PurgeExpiredLogFiles(dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed when disabled, got %d", removed)
	}
}

func TestPurgeExpiredLogFiles_NegativeRetention(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "old.log")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	removed, err := PurgeExpiredLogFiles(dir, -1)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed for negative retention, got %d", removed)
	}
}

func TestPurgeExpiredLogFiles_DirNotExist(t *testing.T) {
	removed, err := PurgeExpiredLogFiles("/nonexistent/path/for/test", 7)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed for nonexistent dir, got %d", removed)
	}
}

func TestPurgeExpiredLogFiles_IsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notadir.log")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	removed, err := PurgeExpiredLogFiles(path, 7)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed when path is a file, got %d", removed)
	}
}

func TestPurgeExpiredLogFiles_SkipSubdir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	removed, err := PurgeExpiredLogFiles(dir, 1)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed (subdirs skipped), got %d", removed)
	}
}

func TestPurgeExpiredLogFiles_NilLogger(t *testing.T) {
	oldLg := Lg
	Lg = nil
	defer func() { Lg = oldLg }()

	dir := t.TempDir()
	path := filepath.Join(dir, "expired.log")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().AddDate(0, 0, -10)
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	removed, err := PurgeExpiredLogFiles(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 removed when Lg is nil, got %d", removed)
	}
}

func TestLogDirFromFilename(t *testing.T) {
	got := LogDirFromFilename("/var/log/app.log")
	if got != "/var/log" {
		t.Fatalf("expected /var/log, got %q", got)
	}
}

func TestLogDirFromFilename_Empty(t *testing.T) {
	got := LogDirFromFilename("")
	if got != "./logs" {
		t.Fatalf("expected ./logs, got %q", got)
	}
}

func TestLogDirFromFilename_NoDir(t *testing.T) {
	got := LogDirFromFilename("app.log")
	if got != "." {
		t.Fatalf("expected '.', got %q", got)
	}
}

func TestPurgeExpiredLogFiles_StatError(t *testing.T) {
	// Create a file and then remove it to simulate a stat error
	dir := t.TempDir()
	path := filepath.Join(dir, "vanishing.log")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().AddDate(0, 0, -10)
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	// Remove the file so entry.Info() fails inside the loop
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	// Should not error — stat failure is logged and skipped
	removed, err := PurgeExpiredLogFiles(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed (file vanished), got %d", removed)
	}
}

func TestPurgeExpiredLogFiles_RemoveError(t *testing.T) {
	// Create a read-only directory to cause os.Remove to fail
	if os.Getuid() == 0 {
		t.Skip("skipping on root: chmod has no effect")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "expired.log")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().AddDate(0, 0, -10)
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	// Make directory read-only to cause remove failure
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o755)

	removed, err := PurgeExpiredLogFiles(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed (remove error), got %d", removed)
	}
}

func TestPurgeExpiredLogFiles_WithLogger(t *testing.T) {
	// Ensure Lg is set so the log branching within PurgeExpiredLogFiles is hit
	oldLg := Lg
	if Lg == nil {
		Lg = zap.NewNop()
	}
	defer func() { Lg = oldLg }()

	dir := t.TempDir()
	// Create an old file that will be removed (with Lg logging)
	path := filepath.Join(dir, "logged-removal.log")
	if err := os.WriteFile(path, []byte("log"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().AddDate(0, 0, -10)
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	removed, err := PurgeExpiredLogFiles(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 removed, got %d", removed)
	}
}
