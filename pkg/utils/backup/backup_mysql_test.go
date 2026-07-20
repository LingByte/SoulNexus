package backup

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBackupMySQLToStoreEmptySQLite(t *testing.T) {
	// Use SQLite via GORM to exercise early failure paths without MySQL.
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE t1 (id INTEGER PRIMARY KEY, v TEXT)`).Error; err != nil {
		t.Fatal(err)
	}
	t.Setenv("BACKUP_PATH", t.TempDir())
	store := &memStore{}
	err = backupMySQLToStore(db, store)
	// SQLite does not have INFORMATION_SCHEMA.TABLES the same way; expect query error or partial dump.
	if err != nil {
		if !strings.Contains(err.Error(), "query") && !strings.Contains(err.Error(), "TABLE") {
			t.Fatalf("unexpected: %v", err)
		}
		return
	}
	if len(store.files) == 0 {
		t.Fatal("expected backup file on success path")
	}
}

func TestBackupMySQLToStoreNilUnderlying(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	err := backupMySQLToStore(db, &memStore{})
	if err == nil {
		t.Fatal("expected error with closed sql.DB")
	}
}
