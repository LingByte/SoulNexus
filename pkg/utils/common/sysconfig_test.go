// configs_test.go
// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package common

import (
	"io"
	"log"
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
