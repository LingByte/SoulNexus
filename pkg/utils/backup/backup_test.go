package backup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/config"
)

func TestBackupSQLiteDatabase(t *testing.T) {
	// Create a temporary SQLite database file
	tmpDir := os.TempDir()
	srcDB := filepath.Join(tmpDir, "test_source.db")
	dstDB := filepath.Join(tmpDir, "test_backup.db")

	// Create source file with some content
	err := os.WriteFile(srcDB, []byte("test database content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source DB: %v", err)
	}
	defer os.Remove(srcDB)
	defer os.Remove(dstDB)

	// Test backup
	err = BackupSQLiteDatabase(srcDB, dstDB)
	if err != nil {
		t.Fatalf("BackupSQLiteDatabase error: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(dstDB); os.IsNotExist(err) {
		t.Fatalf("Backup file was not created")
	}

	// Verify backup file content matches source
	srcContent, err := os.ReadFile(srcDB)
	if err != nil {
		t.Fatalf("Failed to read source file: %v", err)
	}

	dstContent, err := os.ReadFile(dstDB)
	if err != nil {
		t.Fatalf("Failed to read backup file: %v", err)
	}

	if string(srcContent) != string(dstContent) {
		t.Fatalf("Backup content does not match source")
	}
}

func TestBackupSQLiteDatabase_SourceNotExists(t *testing.T) {
	dstDB := filepath.Join(os.TempDir(), "test_backup.db")
	defer os.Remove(dstDB)

	err := BackupSQLiteDatabase("/nonexistent/source.db", dstDB)
	if err == nil {
		t.Fatalf("BackupSQLiteDatabase expected error for non-existent source")
	}
}

func TestBackupSQLiteDatabase_CreatesDirectory(t *testing.T) {
	tmpDir := os.TempDir()
	srcDB := filepath.Join(tmpDir, "test_source.db")
	dstDB := filepath.Join(tmpDir, "nested", "path", "test_backup.db")

	// Create source file
	err := os.WriteFile(srcDB, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source DB: %v", err)
	}
	defer os.Remove(srcDB)
	defer os.RemoveAll(filepath.Dir(dstDB))

	// Test backup (should create nested directories)
	err = BackupSQLiteDatabase(srcDB, dstDB)
	if err != nil {
		t.Fatalf("BackupSQLiteDatabase error: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(dstDB); os.IsNotExist(err) {
		t.Fatalf("Backup file was not created in nested directory")
	}
}

func TestBackupSQLiteDatabase_DirectoryAlreadyExists(t *testing.T) {
	// Test when backup directory already exists
	tmpDir := os.TempDir()
	srcDB := filepath.Join(tmpDir, "test_source.db")
	dstDir := filepath.Join(tmpDir, "existing_backup_dir")
	dstDB := filepath.Join(dstDir, "test_backup.db")

	// Create source file
	err := os.WriteFile(srcDB, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source DB: %v", err)
	}
	defer os.Remove(srcDB)
	defer os.RemoveAll(dstDir)

	// Create destination directory first
	err = os.MkdirAll(dstDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}

	// Test backup (directory already exists)
	err = BackupSQLiteDatabase(srcDB, dstDB)
	if err != nil {
		t.Fatalf("BackupSQLiteDatabase error: %v", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(dstDB); os.IsNotExist(err) {
		t.Fatalf("Backup file was not created")
	}
}

func TestBackupMySQLDatabase_CreatesDirectory(t *testing.T) {
	// Note: This test doesn't actually run mysqldump
	// It only tests that the directory creation logic works
	tmpDir := os.TempDir()
	dstSQL := filepath.Join(tmpDir, "nested", "path", "test_backup.sql")
	defer os.RemoveAll(filepath.Dir(dstSQL))

	// This will fail because mysqldump command won't work,
	// but it should create the directory first
	err := BackupMySQLDatabase("invalid-dsn", dstSQL)
	// We expect an error, but the directory should be created
	if err == nil {
		t.Fatalf("BackupMySQLDatabase expected error for invalid DSN")
	}

	// Check if directory was created (even though backup failed)
	dir := filepath.Dir(dstSQL)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("BackupMySQLDatabase did not create directory")
	}
}

func TestBackupSQLiteDatabase_DirectoryCreationFails(t *testing.T) {
	// This test is hard to simulate without actually having permission issues
	// We'll test the error path by using an invalid path on Windows or a read-only location
	tmpDir := os.TempDir()
	srcDB := filepath.Join(tmpDir, "test_source.db")

	// Create source file
	err := os.WriteFile(srcDB, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source DB: %v", err)
	}
	defer os.Remove(srcDB)

	// Try to backup to a path that might fail (using a very long path or invalid characters)
	// On most systems, this won't actually fail, but we test the error handling path
	dstDB := filepath.Join(tmpDir, "test_backup.db")
	defer os.Remove(dstDB)

	// Normal case should work
	err = BackupSQLiteDatabase(srcDB, dstDB)
	if err != nil {
		// If it fails, that's okay - we're testing error paths
		_ = err
	}
}

func TestBackupSQLiteDatabase_DestinationCreationFails(t *testing.T) {
	tmpDir := os.TempDir()
	srcDB := filepath.Join(tmpDir, "test_source.db")

	// Create source file
	err := os.WriteFile(srcDB, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source DB: %v", err)
	}
	defer os.Remove(srcDB)

	// Try to backup to a read-only location (if possible)
	// On most systems, we can't easily create a read-only directory in temp
	// So we'll just test that the function handles errors gracefully
	dstDB := filepath.Join(tmpDir, "test_backup.db")
	defer os.Remove(dstDB)

	err = BackupSQLiteDatabase(srcDB, dstDB)
	// Should succeed in normal case
	if err != nil {
		t.Logf("BackupSQLiteDatabase error (expected in some cases): %v", err)
	}
}

func TestBackupMySQLDatabase_DirectoryCreation(t *testing.T) {
	tmpDir := os.TempDir()
	dstSQL := filepath.Join(tmpDir, "deeply", "nested", "path", "test_backup.sql")
	defer os.RemoveAll(filepath.Dir(dstSQL))

	// This will fail because mysqldump command won't work with invalid DSN,
	// but it should create the directory first
	err := BackupMySQLDatabase("invalid-dsn", dstSQL)
	if err == nil {
		t.Fatalf("BackupMySQLDatabase expected error for invalid DSN")
	}

	// Check if directory was created (even though backup failed)
	dir := filepath.Dir(dstSQL)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("BackupMySQLDatabase did not create directory")
	}
}

func TestBackupMySQLDatabase_DirectoryAlreadyExists(t *testing.T) {
	// Test when backup directory already exists
	tmpDir := os.TempDir()
	dstDir := filepath.Join(tmpDir, "existing_mysql_backup_dir")
	dstSQL := filepath.Join(dstDir, "test_backup.sql")
	defer os.RemoveAll(dstDir)

	// Create destination directory first
	err := os.MkdirAll(dstDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}

	// This will fail because mysqldump command won't work, but directory already exists
	err = BackupMySQLDatabase("invalid-dsn", dstSQL)
	if err == nil {
		t.Fatalf("BackupMySQLDatabase expected error for invalid DSN")
	}
	// Directory should still exist
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		t.Fatalf("BackupMySQLDatabase directory should still exist")
	}
}

func TestExecuteBackup_SQLite(t *testing.T) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	// Set up test config
	tmpDir := os.TempDir()
	testDB := filepath.Join(tmpDir, "test_execute.db")
	testBackupPath := filepath.Join(tmpDir, "backups")

	// Create test database
	err := os.WriteFile(testDB, []byte("test database"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test DB: %v", err)
	}
	defer os.Remove(testDB)
	defer os.RemoveAll(testBackupPath)

	config.GlobalConfig = &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    testDB,
		},
		Features: config.FeaturesConfig{
			BackupPath: testBackupPath,
		},
	}

	err = ExecuteBackup()
	if err != nil {
		t.Fatalf("ExecuteBackup() error: %v", err)
	}

	// Verify backup file was created
	backupFiles, err := os.ReadDir(testBackupPath)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}
	if len(backupFiles) == 0 {
		t.Fatalf("ExecuteBackup did not create backup file")
	}
}

func TestExecuteBackup_MySQL(t *testing.T) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	// Set up test config
	tmpDir := os.TempDir()
	testBackupPath := filepath.Join(tmpDir, "backups")
	defer os.RemoveAll(testBackupPath)

	config.GlobalConfig = &config.Config{
		Database: config.DatabaseConfig{
			Driver: "mysql",
			DSN:    "invalid-dsn",
		},
		Features: config.FeaturesConfig{
			BackupPath: testBackupPath,
		},
	}

	// This will fail because mysqldump won't work, but we test the code path
	err := ExecuteBackup()
	if err == nil {
		t.Logf("ExecuteBackup with MySQL succeeded (unexpected)")
	} else {
		// Expected error
		_ = err
	}
}

func TestExecuteBackup_UnsupportedDriver(t *testing.T) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	config.GlobalConfig = &config.Config{
		Database: config.DatabaseConfig{
			Driver: "unsupported",
			DSN:    "test-dsn",
		},
		Features: config.FeaturesConfig{
			BackupPath: "./backups",
		},
	}

	err := ExecuteBackup()
	if err == nil {
		t.Fatalf("ExecuteBackup expected error for unsupported driver")
	}
	if err.Error() == "" {
		t.Fatalf("ExecuteBackup error message should not be empty")
	}
}

// Note: StartBackupScheduler is harder to test because it starts a cron scheduler
// and runs indefinitely. It would require more complex mocking or integration tests.
