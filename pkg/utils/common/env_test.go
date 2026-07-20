// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package common

import (
	"os"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/utils/cache"
	"github.com/stretchr/testify/assert"
)

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

	// Explicit empty process env wins over .env (unset vs empty are distinct).
	os.Setenv("TEST_ENV_KEY_FROM_FILE", "")
	defer os.Unsetenv("TEST_ENV_KEY_FROM_FILE")
	value, found = LookupEnv("TEST_ENV_KEY_FROM_FILE")
	assert.True(t, found)
	assert.Equal(t, "", value)

	// Test non-existent key
	os.Unsetenv("NON_EXISTENT_KEY")
	defer os.Unsetenv("NON_EXISTENT_KEY")

	value, found = LookupEnv("NON_EXISTENT_KEY")
	assert.False(t, found)
	assert.Equal(t, "", value)
}

func TestLookupEnv_SearchesUpwardsAndTrimsQuotes(t *testing.T) {
	// Ensure process env doesn't mask file lookup
	os.Unsetenv("DEMO_PORT")
	defer os.Unsetenv("DEMO_PORT")

	// Create a temp dir structure: parent/.env and parent/child (cwd)
	parent := t.TempDir()
	child := parent + string(os.PathSeparator) + "child"
	err := os.MkdirAll(child, 0o755)
	assert.NoError(t, err)

	envContent := `
DEMO_PORT="50400"
SOME_OTHER_KEY=abc
`
	err = os.WriteFile(parent+string(os.PathSeparator)+".env", []byte(envContent), 0o644)
	assert.NoError(t, err)

	// Switch working dir to child, so LookupEnv must walk upwards.
	origWD, err := os.Getwd()
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(origWD) }()
	err = os.Chdir(child)
	assert.NoError(t, err)

	// Purge any cached value from previous tests
	if envCache != nil {
		envCache.Purge()
	}

	v, ok := LookupEnv("DEMO_PORT")
	assert.True(t, ok)
	assert.Equal(t, "50400", v)
}

func TestLookupEnv_SupportsExportAndInlineComment(t *testing.T) {
	os.Unsetenv("DEMO_PORT")
	defer os.Unsetenv("DEMO_PORT")

	parent := t.TempDir()
	child := parent + string(os.PathSeparator) + "child"
	err := os.MkdirAll(child, 0o755)
	assert.NoError(t, err)

	envContent := `
export DEMO_PORT=50400 # comment here
`
	err = os.WriteFile(parent+string(os.PathSeparator)+".env", []byte(envContent), 0o644)
	assert.NoError(t, err)

	origWD, err := os.Getwd()
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(origWD) }()
	err = os.Chdir(child)
	assert.NoError(t, err)

	if envCache != nil {
		envCache.Purge()
	}

	v, ok := LookupEnv("DEMO_PORT")
	assert.True(t, ok)
	assert.Equal(t, "50400", v)
}

func TestLookupEnv_DoesNotTruncateHashInsideValue(t *testing.T) {
	os.Unsetenv("DSN")
	defer os.Unsetenv("DSN")

	parent := t.TempDir()
	child := parent + string(os.PathSeparator) + "child"
	err := os.MkdirAll(child, 0o755)
	assert.NoError(t, err)

	// Mimic real-world DSN where password may contain '#'
	// (e.g. "ct288...##@tcp(host:port)/db?...").
	envContent := "DSN=root:pass##word@tcp(127.0.0.1:3306)/testdb?charset=utf8mb4\n"
	err = os.WriteFile(parent+string(os.PathSeparator)+".env", []byte(envContent), 0o644)
	assert.NoError(t, err)

	origWD, err := os.Getwd()
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(origWD) }()
	err = os.Chdir(child)
	assert.NoError(t, err)

	if envCache != nil {
		envCache.Purge()
	}

	v, ok := LookupEnv("DSN")
	assert.True(t, ok)
	assert.Equal(t, "root:pass##word@tcp(127.0.0.1:3306)/testdb?charset=utf8mb4", v)
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

func TestEnvFloatAndPositiveInt(t *testing.T) {
	t.Setenv("TEST_POS_INT", "42")
	n, ok := PositiveIntEnv("TEST_POS_INT")
	if !ok || n != 42 {
		t.Fatalf("PositiveIntEnv: ok=%v n=%d", ok, n)
	}
	t.Setenv("TEST_POS_INT", "0")
	if _, ok := PositiveIntEnv("TEST_POS_INT"); ok {
		t.Fatal("zero should not be positive")
	}
	t.Setenv("TEST_ENV_FLOAT", "0.75")
	f, ok := EnvFloat("TEST_ENV_FLOAT")
	if !ok || f != 0.75 {
		t.Fatalf("EnvFloat: ok=%v f=%v", ok, f)
	}
}

func TestCacheExpiration(t *testing.T) {
	// Create a cache with short expiration for testing
	shortCache := cache.NewExpiredLRUCache[string, string](10, 10*time.Millisecond)

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
