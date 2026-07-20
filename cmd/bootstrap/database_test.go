package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/internal/config"
	pkgconst "github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ===== Options =====

func TestOptions_Defaults(t *testing.T) {
	opts := &Options{}
	if opts.AutoMigrate != false {
		t.Error("AutoMigrate should default to false")
	}
	if opts.MigrateSQL != false {
		t.Error("MigrateSQL should default to false")
	}
	if opts.SeedNonProd != false {
		t.Error("SeedNonProd should default to false")
	}
	if opts.InitSQLPath != "" {
		t.Error("InitSQLPath should default to empty")
	}
}

func TestOptions_WithValues(t *testing.T) {
	opts := &Options{
		InitSQLPath: "test.sql",
		AutoMigrate: true,
		MigrateSQL:  true,
		SeedNonProd: true,
	}
	if opts.InitSQLPath != "test.sql" {
		t.Errorf("InitSQLPath = %q, want %q", opts.InitSQLPath, "test.sql")
	}
	if !opts.AutoMigrate {
		t.Error("AutoMigrate should be true")
	}
	if !opts.MigrateSQL {
		t.Error("MigrateSQL should be true")
	}
	if !opts.SeedNonProd {
		t.Error("SeedNonProd should be true")
	}
}

// ===== ResolveInitSQLPath =====

func TestResolveInitSQLPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"spaces only", "  ", ""},
		{"valid path", "scripts/sql/init.sql", "scripts/sql/init.sql"},
		{"trimmed", "  scripts/sql/init.sql  ", "scripts/sql/init.sql"},
		{"tabs", "\tscripts/sql/init.sql\t", "scripts/sql/init.sql"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveInitSQLPath(tt.input)
			if got != tt.want {
				t.Errorf("ResolveInitSQLPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ===== RunInitSQLFromPath =====

func TestRunInitSQLFromPath_Empty(t *testing.T) {
	err := RunInitSQLFromPath(nil, "")
	if err != nil {
		t.Fatalf("empty path should return nil, got: %v", err)
	}
}

func TestRunInitSQLFromPath_WhitespaceOnly(t *testing.T) {
	err := RunInitSQLFromPath(nil, "   ")
	if err != nil {
		t.Fatalf("whitespace path should return nil, got: %v", err)
	}
}

func TestRunInitSQLFromPath_NonExistent(t *testing.T) {
	err := RunInitSQLFromPath(nil, "/nonexistent/path/for/testing")
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestRunInitSQLFromPath_Directory(t *testing.T) {
	dir := t.TempDir()
	// Write two SQL files and one non-SQL file
	os.WriteFile(filepath.Join(dir, "01.sql"), []byte("SELECT 1;"), 0644)
	os.WriteFile(filepath.Join(dir, "02.sql"), []byte("SELECT 2;"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not sql"), 0644)

	// Count SQL files in directory
	entries, _ := os.ReadDir(dir)
	sqlCount := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), pkgconst.SQLFileExtension) {
			sqlCount++
		}
	}
	if sqlCount != 2 {
		t.Errorf("expected 2 SQL files, got %d", sqlCount)
	}
}

func TestRunInitSQLFromPath_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	err := RunInitSQLFromPath(nil, dir)
	if err != nil {
		t.Fatalf("empty directory should return nil, got: %v", err)
	}
}

func TestRunInitSQLFromPath_DirectoryWithNonSQLOnly(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not sql"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("markdown"), 0644)

	err := RunInitSQLFromPath(nil, dir)
	if err != nil {
		t.Fatalf("directory with no .sql files should return nil, got: %v", err)
	}
}

// ===== RunInitSQL =====

func TestRunInitSQL_FileNotFound(t *testing.T) {
	err := RunInitSQL(nil, "/nonexistent/file.sql")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestRunInitSQL_BasicStatements(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMockDB(db)

	dir := t.TempDir()
	sqlFile := filepath.Join(dir, "test.sql")
	content := "CREATE TABLE test (id INT);\nINSERT INTO test VALUES (1);\n"
	os.WriteFile(sqlFile, []byte(content), 0644)

	mock.ExpectExec("CREATE TABLE test \\(id INT\\)").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO test VALUES \\(1\\)").WillReturnResult(sqlmock.NewResult(1, 1))

	err = RunInitSQL(db, sqlFile)
	if err != nil {
		t.Fatalf("RunInitSQL failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunInitSQL_WithComments(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMockDB(db)

	dir := t.TempDir()
	sqlFile := filepath.Join(dir, "test.sql")
	content := "-- This is a comment\nCREATE TABLE test (id INT);\n# Hash comment\nINSERT INTO test VALUES (1);\n"
	os.WriteFile(sqlFile, []byte(content), 0644)

	mock.ExpectExec("CREATE TABLE test \\(id INT\\)").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO test VALUES \\(1\\)").WillReturnResult(sqlmock.NewResult(1, 1))

	err = RunInitSQL(db, sqlFile)
	if err != nil {
		t.Fatalf("RunInitSQL with comments failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunInitSQL_EmptyFile(t *testing.T) {
	db, _, err := setupMockDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMockDB(db)

	dir := t.TempDir()
	sqlFile := filepath.Join(dir, "empty.sql")
	os.WriteFile(sqlFile, []byte(""), 0644)

	if err := RunInitSQL(db, sqlFile); err != nil {
		t.Fatalf("empty SQL file should not error: %v", err)
	}
}

func TestRunInitSQL_CommentOnlyFile(t *testing.T) {
	db, _, err := setupMockDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMockDB(db)

	dir := t.TempDir()
	sqlFile := filepath.Join(dir, "comments.sql")
	content := "-- Nothing here\n# Just comments\n"
	os.WriteFile(sqlFile, []byte(content), 0644)

	if err := RunInitSQL(db, sqlFile); err != nil {
		t.Fatalf("comment-only file should not error: %v", err)
	}
}

func TestRunInitSQL_TrailingContentWithoutSemicolon(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMockDB(db)

	dir := t.TempDir()
	sqlFile := filepath.Join(dir, "trailing.sql")
	content := "SELECT 1"
	os.WriteFile(sqlFile, []byte(content), 0644)

	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))

	err = RunInitSQL(db, sqlFile)
	if err != nil {
		t.Fatalf("trailing content without semicolon failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunInitSQL_MultiLineStatement(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMockDB(db)

	dir := t.TempDir()
	sqlFile := filepath.Join(dir, "multiline.sql")
	content := "CREATE TABLE test (\n  id INT,\n  name TEXT\n);\n"
	os.WriteFile(sqlFile, []byte(content), 0644)

	mock.ExpectExec("CREATE TABLE test \\(\n  id INT,\n  name TEXT\n\\)").WillReturnResult(sqlmock.NewResult(0, 0))

	err = RunInitSQL(db, sqlFile)
	if err != nil {
		t.Fatalf("multiline statement failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunInitSQL_MultipleStatements(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMockDB(db)

	dir := t.TempDir()
	sqlFile := filepath.Join(dir, "multi.sql")
	content := "SELECT 1;\nSELECT 2;\nSELECT 3;\n"
	os.WriteFile(sqlFile, []byte(content), 0644)

	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SELECT 2").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("SELECT 3").WillReturnResult(sqlmock.NewResult(0, 0))

	err = RunInitSQL(db, sqlFile)
	if err != nil {
		t.Fatalf("multiple statements failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunInitSQL_ExecutionError(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMockDB(db)

	dir := t.TempDir()
	sqlFile := filepath.Join(dir, "error.sql")
	content := "SELECT 1;\nINVALID SQL;\n"
	os.WriteFile(sqlFile, []byte(content), 0644)

	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INVALID SQL").WillReturnError(
		&mockError{msg: "syntax error"},
	)

	err = RunInitSQL(db, sqlFile)
	if err == nil {
		t.Error("should return error when SQL execution fails")
	}
}

// ===== RunMigrations =====

func TestRunMigrations_NilDB(t *testing.T) {
	err := RunMigrations(nil)
	if err == nil {
		t.Error("RunMigrations with nil db should return error")
	}
}

func TestRunMigrations_NoErrorOnValidDB(t *testing.T) {
	// Use in-memory SQLite
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory db: %v", err)
	}

	err = RunMigrations(db)
	if err != nil {
		t.Fatalf("RunMigrations failed on in-memory db: %v", err)
	}
}

func TestRunVersionedSQLMigrations_FromDisk(t *testing.T) {
	if config.GlobalConfig == nil {
		config.GlobalConfig = &config.Config{}
	}
	config.GlobalConfig.Database.Driver = "sqlite"

	dir := t.TempDir()
	migration := "-- +goose Up\n" +
		"CREATE TABLE IF NOT EXISTS lingecho_schema_meta (k TEXT PRIMARY KEY);\n" +
		"-- +goose Down\n" +
		"DROP TABLE lingecho_schema_meta;\n"
	if err := os.WriteFile(filepath.Join(dir, "00001_meta.sql"), []byte(migration), 0o644); err != nil {
		t.Fatalf("write migration: %v", err)
	}
	t.Setenv(migrationsDirEnv, dir)

	db, err := gorm.Open(sqlite.Open("file:goose_mig?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := RunVersionedSQLMigrations(db); err != nil {
		t.Fatalf("goose up: %v", err)
	}
	// Idempotent: re-running is a no-op via goose_db_version.
	if err := RunVersionedSQLMigrations(db); err != nil {
		t.Fatalf("goose up second: %v", err)
	}
	var n int64
	if err := db.Raw("SELECT COUNT(*) FROM lingecho_schema_meta").Scan(&n).Error; err != nil {
		t.Fatalf("meta table missing: %v", err)
	}
}

// With no migrations dir (the default now that baseline SQL lives in GORM models),
// goose must be a graceful no-op rather than an error.
func TestRunVersionedSQLMigrations_NoDirIsNoop(t *testing.T) {
	if config.GlobalConfig == nil {
		config.GlobalConfig = &config.Config{}
	}
	config.GlobalConfig.Database.Driver = "sqlite"
	t.Setenv(migrationsDirEnv, filepath.Join(t.TempDir(), "does-not-exist"))

	db, err := gorm.Open(sqlite.Open("file:goose_noop?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := RunVersionedSQLMigrations(db); err != nil {
		t.Fatalf("empty migrations dir should be a no-op, got: %v", err)
	}
}

// ===== runPostMigrateSQL =====

func TestRunPostMigrateSQL_NonexistentDir(t *testing.T) {
	err := runPostMigrateSQL(nil, "/nonexistent/dir/path")
	if err != nil {
		t.Fatalf("non-existent migrations dir should return nil (skipped), got: %v", err)
	}
}

func TestRunPostMigrateSQL_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	err := runPostMigrateSQL(nil, dir)
	if err != nil {
		t.Fatalf("empty migrations dir should return nil, got: %v", err)
	}
}

func TestRunPostMigrateSQL_WithSQLFiles(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMockDB(db)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "01_index.sql"), []byte("CREATE INDEX idx_test ON test(id);\n"), 0644)
	os.WriteFile(filepath.Join(dir, "02_data.sql"), []byte("INSERT INTO test VALUES(1);\n"), 0644)
	// Non-SQL file should be skipped
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("readme"), 0644)

	mock.ExpectExec("CREATE INDEX idx_test ON test\\(id\\)").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("INSERT INTO test VALUES\\(1\\)").WillReturnResult(sqlmock.NewResult(1, 1))

	err = runPostMigrateSQL(db, dir)
	if err != nil {
		t.Fatalf("runPostMigrateSQL with SQL files failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunPostMigrateSQL_WithSubDirectories(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMockDB(db)

	dir := t.TempDir()
	// Create a subdirectory - should be ignored
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "01.sql"), []byte("SELECT 1;\n"), 0644)

	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))

	err = runPostMigrateSQL(db, dir)
	if err != nil {
		t.Fatalf("runPostMigrateSQL with subdirectory failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestRunPostMigrateSQL_AllNonSQL(t *testing.T) {
	db, _, err := setupMockDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMockDB(db)

	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("readme"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("notes"), 0644)

	err = runPostMigrateSQL(db, dir)
	if err != nil {
		t.Fatalf("runPostMigrateSQL with no SQL files should succeed: %v", err)
	}
}

// ===== RunInitSQLFromPath single file =====

func TestRunInitSQLFromPath_SingleFile(t *testing.T) {
	db, mock, err := setupMockDB(t)
	if err != nil {
		t.Fatal(err)
	}
	defer closeMockDB(db)

	dir := t.TempDir()
	sqlFile := filepath.Join(dir, "init.sql")
	os.WriteFile(sqlFile, []byte("SELECT 1;\n"), 0644)

	mock.ExpectExec("SELECT 1").WillReturnResult(sqlmock.NewResult(0, 0))

	err = RunInitSQLFromPath(db, sqlFile)
	if err != nil {
		t.Fatalf("RunInitSQLFromPath single file failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// ===== helpers =====

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, error) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, err
	}
	// GORM sqlite driver queries version on init
	mock.ExpectQuery(`select sqlite_version\(\)`).WillReturnRows(
		sqlmock.NewRows([]string{"version"}).AddRow("3.36.0"),
	)
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		sqlDB.Close()
		return nil, nil, err
	}
	return db, mock, nil
}

func closeMockDB(db *gorm.DB) {
	if db == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	_ = sqlDB.Close()
}

// mockError implements the error interface for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
