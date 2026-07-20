package backup

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestBuildPostgresCreateTableSQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE demo (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	// buildPostgresCreateTable uses PostgreSQL information_schema; on SQLite it should fail gracefully.
	_, err = buildPostgresCreateTable(db, "main", "demo")
	if err == nil {
		t.Fatal("expected error on sqlite for postgres schema query")
	}
	if !strings.Contains(err.Error(), "query") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestBackupPostgresToStoreNilDB(t *testing.T) {
	err := backupPostgresToStore(nil, &memStore{})
	if err == nil || !strings.Contains(err.Error(), "sql.DB") {
		t.Fatalf("got %v", err)
	}
}
