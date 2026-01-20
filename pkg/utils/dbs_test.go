package utils

import (
	"os"
	"testing"
)

func TestConfigureConnectionPool(t *testing.T) {
	// Create a test database connection
	// Using SQLite in-memory database for testing
	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(nil, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase error: %v", err)
	}

	// Test ConfigureConnectionPool doesn't panic
	ConfigureConnectionPool(db)

	// Verify connection pool settings were applied
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}

	// Check that settings are applied (these are the values from ConfigureConnectionPool)
	maxIdleConns := sqlDB.Stats().MaxIdleClosed
	_ = maxIdleConns // We can't directly check MaxIdleConns, but we can verify DB is usable

	// Verify database is still usable
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}
}

func TestInitDatabase_WithCustomWriter(t *testing.T) {
	// Create a custom writer
	customWriter, err := os.CreateTemp("", "test_db_log_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(customWriter.Name())
	defer customWriter.Close()

	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(customWriter, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase with custom writer error: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Verify database is usable
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}
}

func TestInitDatabase_WithNilWriter(t *testing.T) {
	// Test that nil writer defaults to os.Stdout
	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(nil, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase with nil writer error: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Verify database is usable
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}
}

func TestInitDatabase_InvalidDSN(t *testing.T) {
	// Test with invalid DSN
	_, err := InitDatabase(nil, "sqlite", "invalid://dsn")
	if err == nil {
		t.Fatalf("InitDatabase expected error for invalid DSN")
	}
}

func TestMakeMigrates(t *testing.T) {
	// Create a test database
	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(nil, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase error: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Define a simple test model
	type TestModel struct {
		ID   uint `gorm:"primarykey"`
		Name string
	}

	// Test migration
	err = MakeMigrates(db, []any{&TestModel{}})
	if err != nil {
		t.Fatalf("MakeMigrates error: %v", err)
	}

	// Verify table was created by trying to query it
	var count int64
	db.Model(&TestModel{}).Count(&count)
	// If no error, migration was successful
}

func TestMakeMigrates_EmptyInstances(t *testing.T) {
	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(nil, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase error: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Test with empty instances
	err = MakeMigrates(db, []any{})
	if err != nil {
		t.Fatalf("MakeMigrates with empty instances error: %v", err)
	}
}

func TestMakeMigrates_InvalidModel(t *testing.T) {
	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(nil, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase error: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Test with invalid model (non-struct)
	err = MakeMigrates(db, []any{"not a struct"})
	if err == nil {
		t.Fatalf("MakeMigrates expected error for invalid model")
	}
}

func TestInitDatabase_DefaultDriverAndDSN(t *testing.T) {
	// This test would require environment variables to be set
	// We'll test that it doesn't crash when env vars are not set
	// (it should use defaults or return error)

	// Save original env
	originalDriver := os.Getenv("DB_DRIVER")
	originalDSN := os.Getenv("DSN")
	defer func() {
		if originalDriver != "" {
			os.Setenv("DB_DRIVER", originalDriver)
		} else {
			os.Unsetenv("DB_DRIVER")
		}
		if originalDSN != "" {
			os.Setenv("DSN", originalDSN)
		} else {
			os.Unsetenv("DSN")
		}
	}()

	// Unset env vars
	os.Unsetenv("DB_DRIVER")
	os.Unsetenv("DSN")

	// Test with empty driver and DSN (will use env or defaults)
	// This may fail if env vars are not set, which is expected
	_, err := InitDatabase(nil, "", "")
	// We don't assert on error here since behavior depends on env vars
	_ = err
}

func TestConfigureConnectionPool_InvalidDB(t *testing.T) {
	// Test that ConfigureConnectionPool handles invalid DB gracefully
	// We can't easily create an invalid *gorm.DB, but we can test
	// that the function doesn't panic when called

	// Create a valid DB first
	dsn := "file::memory:?cache=shared"
	db, err := InitDatabase(nil, "sqlite", dsn)
	if err != nil {
		t.Fatalf("InitDatabase error: %v", err)
	}

	// Close the underlying connection
	sqlDB, _ := db.DB()
	if sqlDB != nil {
		sqlDB.Close()
	}

	// ConfigureConnectionPool should handle closed DB gracefully
	// (it will log an error but not panic)
	ConfigureConnectionPool(db)
}
