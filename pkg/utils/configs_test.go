// configs_test.go
package utils

import (
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	glog "gorm.io/gorm/logger"
)

func setupConfigTestDB() *gorm.DB {
	// 自定义一个“静音 + 忽略 RecordNotFound”的 logger
	silentLogger := glog.New(
		log.New(io.Discard, "", log.LstdFlags), // 丢弃输出
		glog.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  glog.Silent, // 或 glog.Error
			IgnoreRecordNotFoundError: true,        // 关键：忽略 not found
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: silentLogger,
	})
	if err != nil {
		panic("failed to connect database")
	}
	if err := db.AutoMigrate(&Config{}); err != nil {
		panic(err)
	}
	return db
}

func TestConfigStruct(t *testing.T) {
	db := setupConfigTestDB()

	// Test creating a config
	config := Config{
		Key:      "TEST_KEY",
		Desc:     "Test Description",
		Autoload: true,
		Public:   true,
		Format:   "text",
		Value:    "test_value",
	}

	result := db.Create(&config)
	assert.NoError(t, result.Error)
	assert.NotZero(t, config.ID)
	assert.NotZero(t, config.CreatedAt)
	assert.NotZero(t, config.UpdatedAt)
}

func TestSetValue(t *testing.T) {
	db := setupConfigTestDB()

	// Test setting a new value
	SetValue(db, "test_key", "test_value", "text", true, true)

	// Verify the value was set
	var config Config
	result := db.Where("key", "TEST_KEY").First(&config)
	assert.NoError(t, result.Error)
	assert.Equal(t, "TEST_KEY", config.Key)
	assert.Equal(t, "test_value", config.Value)
	assert.Equal(t, "text", config.Format)
	assert.True(t, config.Autoload)
	assert.True(t, config.Public)

	// Test updating existing value
	SetValue(db, "test_key", "updated_value", "text", false, false)

	var updatedConfig Config
	result = db.Where("key", "TEST_KEY").First(&updatedConfig)
	assert.NoError(t, result.Error)
	assert.Equal(t, "updated_value", updatedConfig.Value)
	assert.False(t, updatedConfig.Autoload)
	assert.False(t, updatedConfig.Public)
}

func TestGetValue(t *testing.T) {
	db := setupConfigTestDB()

	// Test getting non-existent value
	value := GetValue(db, "non_existent_key")
	assert.Equal(t, "", value)

	// Set a value first
	SetValue(db, "get_test_key", "get_test_value", "text", true, true)

	// Test getting existing value
	value = GetValue(db, "get_test_key")
	assert.Equal(t, "get_test_value", value)
}

func TestGetIntValue(t *testing.T) {
	db := setupConfigTestDB()

	// Test with non-existent key
	value := GetIntValue(db, "non_existent_int_key", 42)
	assert.Equal(t, 42, value)

	// Set an integer value
	SetValue(db, "int_test_key", "123", "int", true, true)

	// Test getting integer value
	value = GetIntValue(db, "int_test_key", 0)
	assert.Equal(t, 123, value)

	// Set invalid integer value
	SetValue(db, "invalid_int_key", "not_a_number", "text", true, true)

	// Test with invalid integer value
	value = GetIntValue(db, "invalid_int_key", 999)
	assert.Equal(t, 999, value)
}

func TestGetBoolValue(t *testing.T) {
	db := setupConfigTestDB()

	// Test with non-existent key
	value := GetBoolValue(db, "non_existent_bool_key")
	assert.False(t, value)

	// Set a boolean value
	SetValue(db, "bool_test_key", "true", "bool", true, true)

	// Test getting boolean value
	value = GetBoolValue(db, "bool_test_key")
	assert.True(t, value)

	// Set false boolean value
	SetValue(db, "bool_false_key", "false", "bool", true, true)

	// Test getting false boolean value
	value = GetBoolValue(db, "bool_false_key")
	assert.False(t, value)
}

func TestCheckValue(t *testing.T) {
	db := setupConfigTestDB()

	// Test checking and creating a new value
	CheckValue(db, "check_test_key", "default_value", "text", true, true)

	// Verify the value was created
	value := GetValue(db, "check_test_key")
	assert.Equal(t, "default_value", value)

	// Try checking the same key again (should not update)
	CheckValue(db, "check_test_key", "another_value", "text", false, false)

	// Should still have the original value
	value = GetValue(db, "check_test_key")
	assert.Equal(t, "default_value", value)
}

func TestLoadAutoloads(t *testing.T) {
	db := setupConfigTestDB()

	// Create some configs, some with autoload=true
	SetValue(db, "autoload_true_key", "autoload_value", "text", true, false)
	SetValue(db, "autoload_false_key", "no_autoload_value", "text", false, true)

	// Clear cache to ensure we're testing the load functionality
	configValueCache.Purge()

	// Load autoload configs
	LoadAutoloads(db)

	// Check that autoload config is in cache
	value := GetValue(db, "autoload_true_key")
	assert.Equal(t, "autoload_value", value)

	// Check that non-autoload config is not in cache (would need to hit DB)
	configValueCache.Remove("AUTOLOAD_FALSE_KEY")
	value = GetValue(db, "autoload_false_key")
	assert.Equal(t, "no_autoload_value", value)
}

func TestLoadPublicConfigs(t *testing.T) {
	db := setupConfigTestDB()

	// Create some configs, some with public=true
	SetValue(db, "public_true_key", "public_value", "text", false, true)
	SetValue(db, "public_false_key", "private_value", "text", true, false)

	// Load public configs
	configs := LoadPublicConfigs(db)

	// Check that we got the public config
	found := false
	for _, config := range configs {
		if config.Key == "PUBLIC_TRUE_KEY" {
			found = true
			assert.Equal(t, "public_value", config.Value)
			break
		}
	}
	assert.True(t, found)

	// Check that public config is now in cache
	value := GetValue(db, "public_true_key")
	assert.Equal(t, "public_value", value)
}

func TestGetEnv(t *testing.T) {
	// 保存原始状态并在测试结束后恢复
	originalNonExistent := os.Getenv("NON_EXISTENT_ENV_KEY")
	originalTest := os.Getenv("TEST_ENV_KEY")
	defer func() {
		if originalNonExistent == "" {
			os.Unsetenv("NON_EXISTENT_ENV_KEY")
		} else {
			os.Setenv("NON_EXISTENT_ENV_KEY", originalNonExistent)
		}
		if originalTest == "" {
			os.Unsetenv("TEST_ENV_KEY")
		} else {
			os.Setenv("TEST_ENV_KEY", originalTest)
		}
	}()

	// 确保环境变量不存在
	os.Unsetenv("NON_EXISTENT_ENV_KEY")

	// Test getting non-existent env
	value := GetEnv("NON_EXISTENT_ENV_KEY")
	assert.Equal(t, "", value)

	// Set an environment variable
	os.Setenv("TEST_ENV_KEY", "test_env_value")

	// Test getting existing env
	value = GetEnv("TEST_ENV_KEY")
	assert.Equal(t, "test_env_value", value)
}

func TestLookupEnv(t *testing.T) {
	// 清理可能影响测试的环境变量
	os.Unsetenv("TEST_ENV_KEY_FROM_FILE")
	defer os.Unsetenv("TEST_ENV_KEY_FROM_FILE")

	// Create a temporary .env file for testing
	envContent := `
# This is a comment
TEST_ENV_KEY_FROM_FILE=test_value_from_file
ANOTHER_KEY=another_value
INVALID_LINE
`
	err := os.WriteFile(".env", []byte(envContent), 0644)
	assert.NoError(t, err)
	defer os.Remove(".env")

	// Test reading from .env file
	value, found := LookupEnv("TEST_ENV_KEY_FROM_FILE")
	assert.True(t, found)
	assert.Equal(t, "test_value_from_file", value)

	// Test with environment variable (should take precedence)
	os.Setenv("TEST_ENV_KEY_FROM_FILE", "test_value_from_env")
	defer os.Unsetenv("TEST_ENV_KEY_FROM_FILE")

	value, found = LookupEnv("TEST_ENV_KEY_FROM_FILE")
	assert.True(t, found)
	assert.Equal(t, "test_value_from_env", value, "环境变量应该优先于.env文件中的值")

	// Test non-existent key
	os.Unsetenv("NON_EXISTENT_KEY")
	defer os.Unsetenv("NON_EXISTENT_KEY")

	value, found = LookupEnv("NON_EXISTENT_KEY")
	assert.False(t, found)
	assert.Equal(t, "", value)
}

func TestGetBoolEnv(t *testing.T) {
	// 保存原始状态并在测试结束后恢复
	originalNonExistent := os.Getenv("NON_EXISTENT_BOOL_KEY")
	originalTrue := os.Getenv("BOOL_TEST_KEY")
	originalFalse := os.Getenv("BOOL_FALSE_TEST_KEY")
	defer func() {
		if originalNonExistent == "" {
			os.Unsetenv("NON_EXISTENT_BOOL_KEY")
		} else {
			os.Setenv("NON_EXISTENT_BOOL_KEY", originalNonExistent)
		}
		if originalTrue == "" {
			os.Unsetenv("BOOL_TEST_KEY")
		} else {
			os.Setenv("BOOL_TEST_KEY", originalTrue)
		}
		if originalFalse == "" {
			os.Unsetenv("BOOL_FALSE_TEST_KEY")
		} else {
			os.Setenv("BOOL_FALSE_TEST_KEY", originalFalse)
		}
	}()

	// Test with non-existent key
	value := GetBoolEnv("NON_EXISTENT_BOOL_KEY")
	assert.False(t, value)

	// Set a boolean environment variable
	os.Setenv("BOOL_TEST_KEY", "true")

	value = GetBoolEnv("BOOL_TEST_KEY")
	assert.True(t, value)

	// Test with false value
	os.Setenv("BOOL_FALSE_TEST_KEY", "false")

	value = GetBoolEnv("BOOL_FALSE_TEST_KEY")
	assert.False(t, value)
}

func TestGetIntEnv(t *testing.T) {
	// 保存原始状态并在测试结束后恢复
	originalNonExistent := os.Getenv("NON_EXISTENT_INT_KEY")
	originalValid := os.Getenv("INT_TEST_KEY")
	originalInvalid := os.Getenv("INVALID_INT_TEST_KEY")
	defer func() {
		if originalNonExistent == "" {
			os.Unsetenv("NON_EXISTENT_INT_KEY")
		} else {
			os.Setenv("NON_EXISTENT_INT_KEY", originalNonExistent)
		}
		if originalValid == "" {
			os.Unsetenv("INT_TEST_KEY")
		} else {
			os.Setenv("INT_TEST_KEY", originalValid)
		}
		if originalInvalid == "" {
			os.Unsetenv("INVALID_INT_TEST_KEY")
		} else {
			os.Setenv("INVALID_INT_TEST_KEY", originalInvalid)
		}
	}()

	// Test with non-existent key
	value := GetIntEnv("NON_EXISTENT_INT_KEY")
	assert.Equal(t, int64(0), value)

	// Set an integer environment variable
	os.Setenv("INT_TEST_KEY", "12345")

	value = GetIntEnv("INT_TEST_KEY")
	assert.Equal(t, int64(12345), value)

	// Test with invalid integer
	os.Setenv("INVALID_INT_TEST_KEY", "not_a_number")

	value = GetIntEnv("INVALID_INT_TEST_KEY")
	assert.Equal(t, int64(0), value)
}

func TestLoadEnvs(t *testing.T) {
	type TestConfig struct {
		StringValue string `env:"STRING_TEST_KEY"`
		IntValue    int    `env:"INT_TEST_KEY"`
		BoolValue   bool   `env:"BOOL_TEST_KEY"`
		Ignored     string `env:"-"`              // Should be ignored
		Unset       string `env:"UNSET_TEST_KEY"` // Not set in env
	}

	// 清理可能影响测试的环境变量
	os.Unsetenv("STRING_TEST_KEY")
	os.Unsetenv("INT_TEST_KEY")
	os.Unsetenv("BOOL_TEST_KEY")
	defer func() {
		os.Unsetenv("STRING_TEST_KEY")
		os.Unsetenv("INT_TEST_KEY")
		os.Unsetenv("BOOL_TEST_KEY")
	}()

	// Set environment variables
	os.Setenv("STRING_TEST_KEY", "test_string")
	os.Setenv("INT_TEST_KEY", "42")
	os.Setenv("BOOL_TEST_KEY", "true")
	defer func() {
		os.Unsetenv("STRING_TEST_KEY")
		os.Unsetenv("INT_TEST_KEY")
		os.Unsetenv("BOOL_TEST_KEY")
	}()

	// Create config instance and load envs
	config := &TestConfig{}
	LoadEnvs(config)

	// Check values were loaded correctly
	assert.Equal(t, "test_string", config.StringValue)
	assert.Equal(t, 42, config.IntValue)
	assert.True(t, config.BoolValue)
	assert.Equal(t, "", config.Ignored) // Should be empty as it's ignored
	assert.Equal(t, "", config.Unset)   // Should be empty as env var is not set
}

func TestLoadEnv(t *testing.T) {
	// 保存原始状态并在测试结束后恢复
	originalTestKey := os.Getenv("TEST_ENV_FILE_KEY")
	originalAnotherKey := os.Getenv("ANOTHER_ENV_KEY")
	defer func() {
		if originalTestKey == "" {
			os.Unsetenv("TEST_ENV_FILE_KEY")
		} else {
			os.Setenv("TEST_ENV_FILE_KEY", originalTestKey)
		}
		if originalAnotherKey == "" {
			os.Unsetenv("ANOTHER_ENV_KEY")
		} else {
			os.Setenv("ANOTHER_ENV_KEY", originalAnotherKey)
		}
	}()

	// Create a temporary .env.test file for testing
	envContent := `
# Test environment file
TEST_ENV_FILE_KEY=test_value_from_env_file
ANOTHER_ENV_KEY=another_value
`
	envFile := ".env.test"
	err := os.WriteFile(envFile, []byte(envContent), 0644)
	assert.NoError(t, err)
	defer os.Remove(envFile)

	// Load the environment file
	err = LoadEnv("test")
	assert.NoError(t, err)

	// Check that environment variables were set
	value := os.Getenv("TEST_ENV_FILE_KEY")
	assert.Equal(t, "test_value_from_env_file", value)

	value = os.Getenv("ANOTHER_ENV_KEY")
	assert.Equal(t, "another_value", value)

	// Test loading non-existent env file
	err = LoadEnv("nonexistent")
	assert.Error(t, err)
}

func TestCacheExpiration(t *testing.T) {
	// Create a cache with short expiration for testing
	shortCache := NewExpiredLRUCache[string, string](10, 10*time.Millisecond)

	// Add an item
	shortCache.Add("test_key", "test_value")

	// Verify it exists
	value, found := shortCache.Get("test_key")
	assert.True(t, found)
	assert.Equal(t, "test_value", value)

	// Wait for expiration
	time.Sleep(20 * time.Millisecond)

	// Verify it's expired
	value, found = shortCache.Get("test_key")
	assert.False(t, found)
	assert.Equal(t, "", value)
}
