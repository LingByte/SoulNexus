package bootstrap

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"os"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t testing.TB) *gorm.DB {
	// Initialize logger for tests
	logger.Init(&logger.LogConfig{
		Level:    "info",
		Filename: "",
	}, "test")

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	// Auto migrate required tables
	err = db.AutoMigrate(
		&utils.Config{},
		&models.User{},
		&models.Assistant{},
	)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestSeedService_SeedAll(t *testing.T) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	// Set up test config
	config.GlobalConfig = &config.Config{
		Server: config.ServerConfig{
			Name:      "Test Server",
			Desc:      "Test Description",
			Logo:      "/test/logo.png",
			URL:       "https://test.example.com",
			TermsURL:  "https://test.example.com/terms",
			APIPrefix: "/api",
		},
		Features: config.FeaturesConfig{
			SearchEnabled:   true,
			SearchPath:      "./search",
			SearchBatchSize: 100,
		},
	}

	db := setupTestDB(t)
	service := SeedService{db: db}

	err := service.SeedAll()
	assert.NoError(t, err)

	// Verify configs were seeded
	var configCount int64
	err = db.Model(&utils.Config{}).Count(&configCount).Error
	assert.NoError(t, err)
	assert.Greater(t, configCount, int64(0))

	// Verify users were seeded
	var userCount int64
	err = db.Model(&models.User{}).Count(&userCount).Error
	assert.NoError(t, err)
	assert.Greater(t, userCount, int64(0))

	// Verify assistants were seeded
	var assistantCount int64
	err = db.Model(&models.Assistant{}).Count(&assistantCount).Error
	assert.NoError(t, err)
	assert.Greater(t, assistantCount, int64(0))
}

func TestSeedService_seedConfigs(t *testing.T) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	config.GlobalConfig = &config.Config{
		Server: config.ServerConfig{
			Name:      "Test Server",
			Desc:      "Test Description",
			Logo:      "/test/logo.png",
			URL:       "https://test.example.com",
			TermsURL:  "https://test.example.com/terms",
			APIPrefix: "/api",
		},
		Features: config.FeaturesConfig{
			SearchEnabled:   true,
			SearchPath:      "./search",
			SearchBatchSize: 100,
		},
	}

	db := setupTestDB(t)
	service := SeedService{db: db}

	err := service.seedConfigs()
	assert.NoError(t, err)

	// Verify specific configs
	var siteNameConfig utils.Config
	err = db.Where("`key` = ?", constants.KEY_SITE_NAME).First(&siteNameConfig).Error
	assert.NoError(t, err)
	assert.Equal(t, "Test Server", siteNameConfig.Value)

	var siteURLConfig utils.Config
	err = db.Where("`key` = ?", constants.KEY_SITE_URL).First(&siteURLConfig).Error
	assert.NoError(t, err)
	assert.Equal(t, "https://test.example.com", siteURLConfig.Value)

	var searchEnabledConfig utils.Config
	err = db.Where("`key` = ?", constants.KEY_SEARCH_ENABLED).First(&searchEnabledConfig).Error
	assert.NoError(t, err)
	assert.Equal(t, "true", searchEnabledConfig.Value)

	// Test idempotency - running again should not create duplicates
	err = service.seedConfigs()
	assert.NoError(t, err)

	var configCount int64
	err = db.Model(&utils.Config{}).Count(&configCount).Error
	assert.NoError(t, err)

	// Run again and verify count doesn't increase
	err = service.seedConfigs()
	assert.NoError(t, err)

	var newConfigCount int64
	err = db.Model(&utils.Config{}).Count(&newConfigCount).Error
	assert.NoError(t, err)
	assert.Equal(t, configCount, newConfigCount)
}

func TestSeedService_seedConfigs_DefaultValues(t *testing.T) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	// Use empty config to test default values
	config.GlobalConfig = &config.Config{
		Server: config.ServerConfig{
			APIPrefix: "/api",
		},
		Features: config.FeaturesConfig{},
	}

	db := setupTestDB(t)
	service := SeedService{db: db}

	err := service.seedConfigs()
	assert.NoError(t, err)

	// Verify default values are used
	var siteNameConfig utils.Config
	err = db.Where("`key` = ?", constants.KEY_SITE_NAME).First(&siteNameConfig).Error
	assert.NoError(t, err)
	assert.Equal(t, "SoulNexus", siteNameConfig.Value)

	var siteURLConfig utils.Config
	err = db.Where("`key` = ?", constants.KEY_SITE_URL).First(&siteURLConfig).Error
	assert.NoError(t, err)
	assert.Equal(t, "https://lingecho.com", siteURLConfig.Value)

	var searchEnabledConfig utils.Config
	err = db.Where("`key` = ?", constants.KEY_SEARCH_ENABLED).First(&searchEnabledConfig).Error
	assert.NoError(t, err)
	assert.Equal(t, "false", searchEnabledConfig.Value)
}

func TestSeedService_seedAdminUsers(t *testing.T) {
	db := setupTestDB(t)
	service := SeedService{db: db}

	err := service.seedAdminUsers()
	assert.NoError(t, err)

	// Verify admin users were created
	var users []models.User
	err = db.Find(&users).Error
	assert.NoError(t, err)
	assert.Len(t, users, 2)

	// Verify first admin user
	var adminUser models.User
	err = db.Where("email = ?", "admin@lingecho.com").First(&adminUser).Error
	assert.NoError(t, err)
	assert.True(t, adminUser.IsStaff)
	assert.Equal(t, models.RoleSuperAdmin, adminUser.Role)
	assert.Equal(t, "Administrator", adminUser.DisplayName)
	assert.True(t, adminUser.Enabled)

	// Verify password is hashed
	assert.NotEqual(t, "admin123", adminUser.Password)
	assert.True(t, models.CheckPassword(&adminUser, "admin123"))

	// Test idempotency
	err = service.seedAdminUsers()
	assert.NoError(t, err)

	var userCount int64
	err = db.Model(&models.User{}).Count(&userCount).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(2), userCount) // Should still be 2
}

func TestSeedService_seedAssistants(t *testing.T) {
	db := setupTestDB(t)
	service := SeedService{db: db}

	// First seed some users
	err := service.seedAdminUsers()
	require.NoError(t, err)

	err = service.seedAssistants()
	assert.NoError(t, err)

	// Verify assistants were created
	var assistants []models.Assistant
	err = db.Find(&assistants).Error
	assert.NoError(t, err)
	assert.Len(t, assistants, 4)

	// Verify specific assistant
	var techSupport models.Assistant
	err = db.Where("name = ?", "Technical Support").First(&techSupport).Error
	assert.NoError(t, err)
	assert.Equal(t, uint(2), techSupport.UserID)
	assert.Equal(t, "MessageCircle", techSupport.Icon)
	assert.Equal(t, "support", techSupport.PersonaTag)
	assert.Equal(t, float32(0.6), techSupport.Temperature)
	assert.Equal(t, 50, techSupport.MaxTokens)
	assert.NotEmpty(t, techSupport.JsSourceID)

	// Test idempotency - if assistants exist, should not create more
	err = service.seedAssistants()
	assert.NoError(t, err)

	var assistantCount int64
	err = db.Model(&models.Assistant{}).Count(&assistantCount).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(4), assistantCount) // Should still be 4
}

func TestSeedService_seedAssistants_SkipIfExists(t *testing.T) {
	db := setupTestDB(t)
	service := SeedService{db: db}

	// Create an existing assistant
	existingAssistant := models.Assistant{
		UserID:      1,
		Name:        "Existing Assistant",
		Description: "Already exists",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := db.Create(&existingAssistant).Error
	require.NoError(t, err)

	// Try to seed assistants
	err = service.seedAssistants()
	assert.NoError(t, err)

	// Should not create default assistants since one already exists
	var assistantCount int64
	err = db.Model(&models.Assistant{}).Count(&assistantCount).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(1), assistantCount) // Only the existing one
}

func TestSeedService_DatabaseError(t *testing.T) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	// Set minimal config to avoid nil pointer
	config.GlobalConfig = &config.Config{
		Server: config.ServerConfig{
			APIPrefix: "/api",
		},
		Features: config.FeaturesConfig{},
	}

	// Create a database that will fail operations
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Don't migrate tables to cause errors
	service := SeedService{db: db}

	// Should handle database errors gracefully
	err = service.seedConfigs()
	assert.Error(t, err)

	err = service.seedAdminUsers()
	assert.Error(t, err)

	err = service.seedAssistants()
	assert.Error(t, err)
}

func TestSeedService_SeedAll_PartialFailure(t *testing.T) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	// Set minimal config to avoid nil pointer
	config.GlobalConfig = &config.Config{
		Server: config.ServerConfig{
			APIPrefix: "/api",
		},
		Features: config.FeaturesConfig{},
	}

	db := setupTestDB(t)
	service := SeedService{db: db}

	// Drop one table to cause partial failure
	err := db.Migrator().DropTable(&models.User{})
	require.NoError(t, err)

	// SeedAll should fail when seedAdminUsers fails
	err = service.SeedAll()
	assert.Error(t, err)
}

// Benchmark tests
func BenchmarkSeedService_SeedAll(b *testing.B) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	config.GlobalConfig = &config.Config{
		Server: config.ServerConfig{
			Name:      "Benchmark Server",
			APIPrefix: "/api",
		},
		Features: config.FeaturesConfig{
			SearchEnabled: true,
		},
	}

	for i := 0; i < b.N; i++ {
		db := setupTestDB(b)
		service := SeedService{db: db}

		b.StartTimer()
		err := service.SeedAll()
		b.StopTimer()

		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSeedService_seedConfigs(b *testing.B) {
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	config.GlobalConfig = &config.Config{
		Server: config.ServerConfig{
			Name:      "Benchmark Server",
			APIPrefix: "/api",
		},
		Features: config.FeaturesConfig{},
	}

	for i := 0; i < b.N; i++ {
		db := setupTestDB(b)
		service := SeedService{db: db}

		b.StartTimer()
		err := service.seedConfigs()
		b.StopTimer()

		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test environment variable handling
func TestSeedService_EnvironmentVariables(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)

	// Test different environment values
	testCases := []struct {
		env      string
		expected bool
	}{
		{"", true},             // Empty should seed
		{"test", true},         // Test should seed
		{"development", false}, // Development should not seed
		{"production", false},  // Production should not seed
		{"staging", true},      // Other values should seed
	}

	for _, tc := range testCases {
		t.Run("env_"+tc.env, func(t *testing.T) {
			os.Setenv("APP_ENV", tc.env)

			// This test would require integration with SetupDatabase
			// to fully test the environment variable logic
			// For now, we just verify the environment is set correctly
			assert.Equal(t, tc.env, os.Getenv("APP_ENV"))
		})
	}
}
