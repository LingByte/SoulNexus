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
	"github.com/LingByte/SoulNexus/pkg/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogConfigInfo(t *testing.T) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	// Create test config
	config.GlobalConfig = &config.Config{
		MachineID: 12345,
		Server: config.ServerConfig{
			Name:          "Test Server",
			Desc:          "Test Description",
			Logo:          "/test/logo.png",
			URL:           "https://test.example.com",
			TermsURL:      "https://test.example.com/terms",
			Mode:          "test",
			Addr:          ":8080",
			APIPrefix:     "/api",
			DocsPrefix:    "/docs",
			AdminPrefix:   "/admin",
			AuthPrefix:    "/auth",
			MonitorPrefix: "/metrics",
		},
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    "./test.db",
		},
		Log: logger.LogConfig{
			Level:      "info",
			Filename:   "./test.log",
			MaxSize:    100,
			MaxAge:     30,
			MaxBackups: 5,
		},
		Auth: config.AuthConfig{
			APISecretKey:     "test-secret",
			SecretExpireDays: "7",
			SessionSecret:    "session-secret",
		},
		Services: config.ServicesConfig{
			Mail: notification.MailConfig{
				Host:     "smtp.test.com",
				Username: "test@example.com",
				Password: "password",
				From:     "noreply@test.com",
				Port:     587,
			},
			Voice: config.VoiceConfig{
				Xunfei: config.XunfeiVoiceConfig{
					WSAppId:     "test-app-id",
					WSAPIKey:    "test-api-key",
					WSAPISecret: "test-api-secret",
				},
			},
		},
		Features: config.FeaturesConfig{
			LanguageEnabled: true,
			SearchEnabled:   true,
			SearchPath:      "./search",
			SearchBatchSize: 100,
			BackupEnabled:   true,
			BackupPath:      "./backups",
			BackupSchedule:  "0 2 * * *",
		},
	}

	// Capture logs by replacing the global logger
	core, recorded := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)

	// Replace the global logger in the logger package
	originalLogger := logger.Lg
	logger.Lg = testLogger
	defer func() {
		logger.Lg = originalLogger
	}()

	// Call the function
	LogConfigInfo()

	// Verify logs were written
	entries := recorded.All()
	assert.Greater(t, len(entries), 0, "Should have logged configuration info")

	// Check for specific log messages
	logMessages := make([]string, len(entries))
	for i, entry := range entries {
		logMessages[i] = entry.Message
	}

	expectedMessages := []string{
		"system config load finished",
		"global config",
		"base config",
		"api config",
		"log config",
		"mail config",
		"search config",
		"backup config",
		"xun fei config",
	}

	for _, expected := range expectedMessages {
		assert.Contains(t, logMessages, expected, "Should contain log message: %s", expected)
	}

	// Verify specific field values in logs
	var globalConfigEntry *observer.LoggedEntry
	for _, entry := range entries {
		if entry.Message == "global config" {
			globalConfigEntry = &entry
			break
		}
	}

	require.NotNil(t, globalConfigEntry, "Should have global config log entry")

	// Check fields
	fields := make(map[string]interface{})
	for _, field := range globalConfigEntry.Context {
		fields[field.Key] = field.String
	}

	assert.Equal(t, "Test Server", fields["server_name"])
	assert.Equal(t, "Test Description", fields["server_desc"])
	assert.Equal(t, "/test/logo.png", fields["server_logo"])
	assert.Equal(t, "https://test.example.com", fields["server_url"])
	assert.Equal(t, "test", fields["mode"])
}

func TestLogConfigInfo_EmptyConfig(t *testing.T) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	// Create minimal config
	config.GlobalConfig = &config.Config{
		Server:   config.ServerConfig{},
		Database: config.DatabaseConfig{},
		Log:      logger.LogConfig{},
		Auth:     config.AuthConfig{},
		Services: config.ServicesConfig{
			Mail: notification.MailConfig{},
			Voice: config.VoiceConfig{
				Xunfei: config.XunfeiVoiceConfig{},
			},
		},
		Features: config.FeaturesConfig{},
	}

	// Capture logs by replacing the global logger
	core, recorded := observer.New(zapcore.InfoLevel)
	testLogger := zap.New(core)

	// Replace the global logger in the logger package
	originalLogger := logger.Lg
	logger.Lg = testLogger
	defer func() {
		logger.Lg = originalLogger
	}()

	// Should not panic with empty config
	assert.NotPanics(t, func() {
		LogConfigInfo()
	})

	// Verify logs were still written
	entries := recorded.All()
	assert.Greater(t, len(entries), 0)
}

func TestPrintBannerFromFile(t *testing.T) {
	// Create temporary banner file
	tmpDir := t.TempDir()
	bannerPath := filepath.Join(tmpDir, "banner.txt")

	bannerContent := `
  ╔══════════════════════════════════════╗
  ║            Test Banner               ║
  ║         Welcome to LingEcho          ║
  ╚══════════════════════════════════════╝
`
	err := os.WriteFile(bannerPath, []byte(bannerContent), 0644)
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call function
	err = PrintBannerFromFile(bannerPath)
	assert.NoError(t, err)

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains banner content (without ANSI codes)
	assert.Contains(t, output, "Test Banner")
	assert.Contains(t, output, "Welcome to LingEcho")

	// Verify ANSI color codes are present
	assert.Contains(t, output, "\x1b[38;5;")
	assert.Contains(t, output, "\x1b[0m")
}

func TestPrintBannerFromFile_FileNotFound(t *testing.T) {
	err := PrintBannerFromFile("/nonexistent/banner.txt")
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestPrintBannerFromFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	bannerPath := filepath.Join(tmpDir, "empty.txt")

	err := os.WriteFile(bannerPath, []byte(""), 0644)
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = PrintBannerFromFile(bannerPath)
	assert.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Should have at least one line (empty line)
	assert.Contains(t, output, "\x1b[0m")
}

func TestPrintBannerFromFile_SingleLine(t *testing.T) {
	tmpDir := t.TempDir()
	bannerPath := filepath.Join(tmpDir, "single.txt")

	err := os.WriteFile(bannerPath, []byte("Single Line Banner"), 0644)
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = PrintBannerFromFile(bannerPath)
	assert.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "Single Line Banner")
}

func TestPrintBannerFromFile_MultipleLines(t *testing.T) {
	tmpDir := t.TempDir()
	bannerPath := filepath.Join(tmpDir, "multi.txt")

	bannerContent := strings.Join([]string{
		"Line 1",
		"Line 2",
		"Line 3",
		"Line 4",
		"Line 5",
		"Line 6",
		"Line 7", // More than 6 lines to test color cycling
	}, "\n")

	err := os.WriteFile(bannerPath, []byte(bannerContent), 0644)
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = PrintBannerFromFile(bannerPath)
	assert.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify all lines are present
	for i := 1; i <= 7; i++ {
		assert.Contains(t, output, "Line "+string(rune('0'+i)))
	}

	// Verify different colors are used (color cycling)
	assert.Contains(t, output, "\x1b[38;5;165m") // First color
	assert.Contains(t, output, "\x1b[38;5;189m") // Second color
}

func TestPrintBannerFromFile_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	bannerPath := filepath.Join(tmpDir, "large.txt")

	// Create a large banner with many lines
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "Banner line "+string(rune('0'+i%10)))
	}
	bannerContent := strings.Join(lines, "\n")

	err := os.WriteFile(bannerPath, []byte(bannerContent), 0644)
	require.NoError(t, err)

	// Should handle large files without issues
	err = PrintBannerFromFile(bannerPath)
	assert.NoError(t, err)
}

func TestPrintBannerFromFile_PermissionDenied(t *testing.T) {
	tmpDir := t.TempDir()
	bannerPath := filepath.Join(tmpDir, "noperm.txt")

	err := os.WriteFile(bannerPath, []byte("test"), 0644)
	require.NoError(t, err)

	// Remove read permission
	err = os.Chmod(bannerPath, 0000)
	require.NoError(t, err)

	defer os.Chmod(bannerPath, 0644) // Restore for cleanup

	err = PrintBannerFromFile(bannerPath)
	assert.Error(t, err)
}

// Benchmark tests
func BenchmarkLogConfigInfo(b *testing.B) {
	// Save original config
	originalConfig := config.GlobalConfig
	defer func() {
		config.GlobalConfig = originalConfig
	}()

	// Create test config
	config.GlobalConfig = &config.Config{
		MachineID: 12345,
		Server: config.ServerConfig{
			Name: "Benchmark Server",
			Desc: "Benchmark Description",
			Mode: "benchmark",
			Addr: ":8080",
		},
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			DSN:    "./bench.db",
		},
		Log: logger.LogConfig{
			Level:    "info",
			Filename: "./bench.log",
		},
		Auth:     config.AuthConfig{},
		Services: config.ServicesConfig{},
		Features: config.FeaturesConfig{},
	}

	// Use a no-op logger for benchmarking
	logger := zap.NewNop()
	originalLogger := zap.L()
	zap.ReplaceGlobals(logger)
	defer zap.ReplaceGlobals(originalLogger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LogConfigInfo()
	}
}

func BenchmarkPrintBannerFromFile(b *testing.B) {
	tmpDir := b.TempDir()
	bannerPath := filepath.Join(tmpDir, "bench_banner.txt")

	bannerContent := strings.Repeat("Benchmark Banner Line\n", 10)
	err := os.WriteFile(bannerPath, []byte(bannerContent), 0644)
	if err != nil {
		b.Fatal(err)
	}

	// Redirect stdout to discard output during benchmark
	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := PrintBannerFromFile(bannerPath)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test color cycling specifically
func TestPrintBannerFromFile_ColorCycling(t *testing.T) {
	tmpDir := t.TempDir()
	bannerPath := filepath.Join(tmpDir, "colors.txt")

	// Create exactly 12 lines to test color cycling (6 colors * 2)
	lines := make([]string, 12)
	for i := 0; i < 12; i++ {
		lines[i] = "Color test line " + string(rune('A'+i))
	}
	bannerContent := strings.Join(lines, "\n")

	err := os.WriteFile(bannerPath, []byte(bannerContent), 0644)
	require.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = PrintBannerFromFile(bannerPath)
	assert.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify that colors cycle (first and seventh line should have same color)
	lines = strings.Split(output, "\n")
	if len(lines) >= 12 {
		// Extract color codes from first and seventh lines
		firstLineColor := extractColorCode(lines[0])
		seventhLineColor := extractColorCode(lines[6])

		assert.Equal(t, firstLineColor, seventhLineColor, "Colors should cycle every 6 lines")
	}
}

// Helper function to extract color code from a line
func extractColorCode(line string) string {
	start := strings.Index(line, "\x1b[38;5;")
	if start == -1 {
		return ""
	}
	end := strings.Index(line[start:], "m")
	if end == -1 {
		return ""
	}
	return line[start : start+end+1]
}
