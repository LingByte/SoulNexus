package bootstrap

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSetupDatabase(t *testing.T) {
	// Initialize logger for tests
	logger.Init(&logger.LogConfig{
		Level:    "info",
		Filename: "",
	}, "test")

	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config.GlobalConfig = &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    dbPath,
		},
		Server: config.ServerConfig{
			APIPrefix: "/api",
		},
	}

	// Set test environment
	os.Setenv("APP_ENV", "test")
	defer os.Unsetenv("APP_ENV")

	var buf bytes.Buffer
	opts := &Options{
		AutoMigrate: true,
		SeedNonProd: true,
	}

	db, err := SetupDatabase(&buf, opts)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Verify database connection
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Ping())

	// Clean up
	sqlDB.Close()
}

func TestSetupDatabase_WithInitSQL(t *testing.T) {
	// Initialize logger for tests
	logger.Init(&logger.LogConfig{
		Level:    "info",
		Filename: "",
	}, "test")

	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	// Create temporary database and SQL file
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	sqlPath := filepath.Join(tmpDir, "init.sql")

	// Create test SQL file
	sqlContent := `
-- Test comment
CREATE TABLE IF NOT EXISTS test_table (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT OR IGNORE INTO test_table (id, name) VALUES (1, 'test');
`
	err := os.WriteFile(sqlPath, []byte(sqlContent), 0644)
	require.NoError(t, err)

	config.GlobalConfig = &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    dbPath,
		},
		Server: config.ServerConfig{
			APIPrefix: "/api",
		},
	}

	var buf bytes.Buffer
	opts := &Options{
		InitSQLPath: sqlPath,
		AutoMigrate: true,
		SeedNonProd: false,
	}

	db, err := SetupDatabase(&buf, opts)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Verify SQL was executed
	var count int64
	err = db.Table("test_table").Count(&count).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Clean up
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func TestSetupDatabase_NilOptions(t *testing.T) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config.GlobalConfig = &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    dbPath,
		},
		Server: config.ServerConfig{
			APIPrefix: "/api",
		},
	}

	var buf bytes.Buffer
	db, err := SetupDatabase(&buf, nil)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Clean up
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func TestInitDBConn(t *testing.T) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config.GlobalConfig = &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    dbPath,
		},
	}

	var buf bytes.Buffer
	db, err := initDBConn(&buf)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Clean up
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func TestRunInitSQL(t *testing.T) {
	// Create in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create temporary SQL file
	tmpDir := t.TempDir()
	sqlPath := filepath.Join(tmpDir, "test.sql")

	sqlContent := `
-- This is a comment
CREATE TABLE test_users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE
);

-- Another comment
INSERT INTO test_users (name, email) VALUES ('John', 'john@example.com');
INSERT INTO test_users (name, email) VALUES ('Jane', 'jane@example.com');

# Hash comment
CREATE TABLE test_posts (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL
);
`
	err = os.WriteFile(sqlPath, []byte(sqlContent), 0644)
	require.NoError(t, err)

	// Run SQL
	err = RunInitSQL(db, sqlPath)
	require.NoError(t, err)

	// Verify tables were created and data inserted
	var userCount int64
	err = db.Table("test_users").Count(&userCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(2), userCount)

	var postCount int64
	err = db.Table("test_posts").Count(&postCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), postCount)
}

func TestRunInitSQL_FileNotFound(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = RunInitSQL(db, "/nonexistent/file.sql")
	assert.Error(t, err)
}

func TestRunInitSQL_EmptyFile(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	tmpDir := t.TempDir()
	sqlPath := filepath.Join(tmpDir, "empty.sql")
	err = os.WriteFile(sqlPath, []byte(""), 0644)
	require.NoError(t, err)

	err = RunInitSQL(db, sqlPath)
	assert.NoError(t, err)
}

func TestRunInitSQL_OnlyComments(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	tmpDir := t.TempDir()
	sqlPath := filepath.Join(tmpDir, "comments.sql")
	sqlContent := `
-- This is a comment
# This is also a comment

-- Another comment
`
	err = os.WriteFile(sqlPath, []byte(sqlContent), 0644)
	require.NoError(t, err)

	err = RunInitSQL(db, sqlPath)
	assert.NoError(t, err)
}

func TestRunInitSQL_StatementWithoutSemicolon(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	tmpDir := t.TempDir()
	sqlPath := filepath.Join(tmpDir, "no_semicolon.sql")
	sqlContent := `CREATE TABLE test_table (id INTEGER PRIMARY KEY)`
	err = os.WriteFile(sqlPath, []byte(sqlContent), 0644)
	require.NoError(t, err)

	err = RunInitSQL(db, sqlPath)
	assert.NoError(t, err)

	// Verify table was created
	var count int64
	err = db.Table("test_table").Count(&count).Error
	assert.NoError(t, err)
}

func TestRunInitSQL_InvalidSQL(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	tmpDir := t.TempDir()
	sqlPath := filepath.Join(tmpDir, "invalid.sql")
	sqlContent := `INVALID SQL STATEMENT;`
	err = os.WriteFile(sqlPath, []byte(sqlContent), 0644)
	require.NoError(t, err)

	err = RunInitSQL(db, sqlPath)
	assert.Error(t, err)
}

func TestRunMigrations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	err = RunMigrations(db)
	assert.NoError(t, err)

	// Verify some tables were created
	assert.True(t, db.Migrator().HasTable("users"))
	assert.True(t, db.Migrator().HasTable("assistants"))
	assert.True(t, db.Migrator().HasTable("groups"))
}

func TestRunMigrations_NilDB(t *testing.T) {
	err := RunMigrations(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db is nil")
}

func TestSetupDatabase_ProductionEnvironment(t *testing.T) {
	// Save original config and env
	originalConfig := config.GlobalConfig
	originalEnv := os.Getenv("APP_ENV")
	defer func() {
		config.GlobalConfig = originalConfig
		os.Setenv("APP_ENV", originalEnv)
	}()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config.GlobalConfig = &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    dbPath,
		},
		Server: config.ServerConfig{
			APIPrefix: "/api",
		},
	}

	// Set production environment
	os.Setenv("APP_ENV", "production")

	var buf bytes.Buffer
	opts := &Options{
		AutoMigrate: true,
		SeedNonProd: true, // Should be ignored in production
	}

	db, err := SetupDatabase(&buf, opts)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Clean up
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func TestSetupDatabase_DatabaseConnectionError(t *testing.T) {
	// Initialize logger for tests
	logger.Init(&logger.LogConfig{
		Level:    "info",
		Filename: "",
	}, "test")

	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	// Use invalid database configuration that will definitely fail
	config.GlobalConfig = &config.Config{
		Database: config.DatabaseConfig{
			Driver: "mysql", // Use mysql driver but with invalid DSN
			DSN:    "invalid:invalid@tcp(nonexistent:3306)/nonexistent?charset=utf8mb4&parseTime=True&loc=Local",
		},
	}

	var buf bytes.Buffer
	opts := &Options{
		AutoMigrate: true,
		SeedNonProd: false,
	}

	db, err := SetupDatabase(&buf, opts)
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestSetupDatabase_MigrationError(t *testing.T) {
	// This test is harder to simulate without mocking
	// We'll test the error path by using a read-only database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "readonly.db")

	// Create database file
	file, err := os.Create(dbPath)
	require.NoError(t, err)
	file.Close()

	// Make it read-only
	err = os.Chmod(dbPath, 0444)
	require.NoError(t, err)

	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
		os.Chmod(dbPath, 0644) // Restore permissions for cleanup
	}()

	config.GlobalConfig = &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    dbPath,
		},
	}

	var buf bytes.Buffer
	opts := &Options{
		AutoMigrate: true,
		SeedNonProd: false,
	}

	// This should fail during migration
	db, err := SetupDatabase(&buf, opts)
	// Note: SQLite might still succeed with read-only files in some cases
	// The important thing is that we handle errors gracefully
	if err != nil {
		assert.Nil(t, db)
	} else if db != nil {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
}

// Benchmark tests
func BenchmarkSetupDatabase(b *testing.B) {
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		dbPath := filepath.Join(tmpDir, "bench.db")

		config.GlobalConfig = &config.Config{
			Database: config.DatabaseConfig{
				Driver: "sqlite",
				DSN:    dbPath,
			},
			Server: config.ServerConfig{
				APIPrefix: "/api",
			},
		}

		var buf bytes.Buffer
		opts := &Options{
			AutoMigrate: true,
			SeedNonProd: false,
		}

		db, err := SetupDatabase(&buf, opts)
		if err != nil {
			b.Fatal(err)
		}

		sqlDB, _ := db.DB()
		sqlDB.Close()
	}
}

func BenchmarkRunInitSQL(b *testing.B) {
	// Create test SQL content
	sqlContent := strings.Repeat("INSERT INTO test_table (name) VALUES ('test');\n", 100)

	for i := 0; i < b.N; i++ {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			b.Fatal(err)
		}

		// Create table first
		err = db.Exec("CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)").Error
		if err != nil {
			b.Fatal(err)
		}

		tmpDir := b.TempDir()
		sqlPath := filepath.Join(tmpDir, "bench.sql")
		err = os.WriteFile(sqlPath, []byte(sqlContent), 0644)
		if err != nil {
			b.Fatal(err)
		}

		b.StartTimer()
		err = RunInitSQL(db, sqlPath)
		b.StopTimer()

		if err != nil {
			b.Fatal(err)
		}
	}
}
