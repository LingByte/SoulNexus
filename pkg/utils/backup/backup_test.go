package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestLoadManifestEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	m, err := loadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Records) != 0 {
		t.Fatalf("expected empty records, got %d", len(m.Records))
	}
}

func TestManifestAddAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	m, err := loadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	rec := BackupRecord{
		Key:      "backup/sys_backup_20260101_120000.db",
		FileName: "sys_backup_20260101_120000.db",
		Size:     42,
		Date:     time.Now().UTC().Format(time.RFC3339),
		URL:      "http://example/backup.db",
	}
	if err := m.addRecord(rec); err != nil {
		t.Fatal(err)
	}
	m2, err := loadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(m2.Records) != 1 || m2.Records[0].Key != rec.Key || m2.Records[0].Size != 42 {
		t.Fatalf("reload %#v", m2.Records)
	}
}

func TestManifestRemoveRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	m := &BackupManifest{path: path, Records: []BackupRecord{
		{Key: "a", FileName: "a"},
		{Key: "b", FileName: "b"},
		{Key: "c", FileName: "c"},
	}}
	if err := m.save(); err != nil {
		t.Fatal(err)
	}
	if err := m.removeRecords(map[int]bool{1: true}); err != nil {
		t.Fatal(err)
	}
	if len(m.Records) != 2 || m.Records[0].Key != "a" || m.Records[1].Key != "c" {
		t.Fatalf("after remove %#v", m.Records)
	}
}

func TestLoadManifestInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadManifest(path); err == nil {
		t.Fatal("expected unmarshal error")
	}
}

func TestBackupKeyFormat(t *testing.T) {
	key := backupKey("sql")
	if !strings.HasPrefix(key, defaultBackupPrefix+"/sys_backup_") {
		t.Fatalf("key %q", key)
	}
	if !strings.HasSuffix(key, ".sql") {
		t.Fatalf("key %q", key)
	}
}

func TestExecuteBackupValidation(t *testing.T) {
	if err := ExecuteBackup(nil, nil, DatabaseConfig{}); err == nil {
		t.Fatal("nil db should error")
	}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	err = ExecuteBackup(db, &memStore{}, DatabaseConfig{Driver: "unknown"})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("got %v", err)
	}
}

func TestManifestSaveEmptyPath(t *testing.T) {
	m := &BackupManifest{Records: []BackupRecord{}}
	if err := m.save(); err == nil {
		t.Fatal("expected empty path error")
	}
}

func TestBackupRecordJSONRoundTrip(t *testing.T) {
	rec := BackupRecord{
		Key:      "backup/k",
		FileName: "k",
		Size:     100,
		Date:     "2026-01-01T00:00:00Z",
		URL:      "http://x",
	}
	b, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	var out BackupRecord
	if err := json.Unmarshal(b, &out); err != nil || out.Key != rec.Key {
		t.Fatalf("round trip %#v err=%v", out, err)
	}
}

func TestExecuteBackupSQLiteDriver(t *testing.T) {
	t.Setenv(constants.ENV_BACKUP_PATH, t.TempDir())
	// nil store should fail after driver dispatch would need db - test unsupported only above
	err := ExecuteBackup(nil, nil, DatabaseConfig{Driver: constants.DBDriverSQLite, DSN: "file:missing.db"})
	if err == nil {
		t.Fatal("expected error with nil store")
	}
}
