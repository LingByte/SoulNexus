package i18n

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

func init() {
	// Initialize logger for tests
	if logger.Lg == nil {
		logger.Lg = zap.NewNop() // Use no-op logger for tests
	}
}

func TestNewManager(t *testing.T) {
	manager := NewManager(nil)
	if manager == nil {
		t.Fatal("expected non-nil manager")
	}
	if manager.defaultLocale != DefaultLocale {
		t.Errorf("expected default locale %s, got %s", DefaultLocale, manager.defaultLocale)
	}
}

func TestNewManager_WithConfig(t *testing.T) {
	config := &Config{
		DefaultLocale:    "zh-CN",
		SupportedLocales: []Locale{"en", "zh-CN", "zh-TW"},
		FallbackLocale:   "en",
	}
	manager := NewManager(config)
	if manager.defaultLocale != "zh-CN" {
		t.Errorf("expected default locale zh-CN, got %s", manager.defaultLocale)
	}
	if len(manager.supportedLocales) != 3 {
		t.Errorf("expected 3 supported locales, got %d", len(manager.supportedLocales))
	}
}

func TestManager_SetTranslation(t *testing.T) {
	manager := NewManager(nil)
	manager.SetTranslation("en", "test.key", "Test Value")

	value := manager.GetTranslation("en", "test.key")
	if value != "Test Value" {
		t.Errorf("expected 'Test Value', got '%s'", value)
	}
}

func TestManager_GetTranslation_Fallback(t *testing.T) {
	manager := NewManager(&Config{
		DefaultLocale:    "en",
		SupportedLocales: []Locale{"en", "zh-CN"},
		FallbackLocale:   "en",
	})

	// Set translation only in fallback locale
	manager.SetTranslation("en", "test.key", "Fallback Value")

	// Try to get from unsupported locale
	value := manager.GetTranslation("zh-CN", "test.key")
	if value != "Fallback Value" {
		t.Errorf("expected fallback value, got '%s'", value)
	}
}

func TestManager_GetTranslation_NotFound(t *testing.T) {
	manager := NewManager(nil)
	value := manager.GetTranslation("en", "nonexistent.key")
	if value != "nonexistent.key" {
		t.Errorf("expected key to be returned, got '%s'", value)
	}
}

func TestManager_T_WithArgs(t *testing.T) {
	manager := NewManager(nil)
	manager.SetTranslation("en", "hello", "Hello, %s!")

	result := manager.T("en", "hello", "World")
	if result != "Hello, World!" {
		t.Errorf("expected 'Hello, World!', got '%s'", result)
	}
}

func TestManager_DetectLocale(t *testing.T) {
	manager := NewManager(&Config{
		DefaultLocale:    "en",
		SupportedLocales: []Locale{"en", "zh-CN", "zh-TW", "fr"},
		FallbackLocale:   "en",
	})

	tests := []struct {
		input    string
		expected Locale
	}{
		{"en", "en"},
		{"en-US", "en"},
		{"zh-CN", "zh-CN"},
		{"zh-TW", "zh-TW"},
		{"zh_CN", "zh-CN"},
		{"fr-FR", "fr"},
		{"unsupported", "en"}, // Should fallback to default
		{"", "en"},
	}

	for _, tt := range tests {
		result := manager.DetectLocale(tt.input)
		if result != tt.expected {
			t.Errorf("DetectLocale(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestManager_ParseAcceptLanguage(t *testing.T) {
	manager := NewManager(&Config{
		DefaultLocale:    "en",
		SupportedLocales: []Locale{"en", "zh-CN", "zh-TW", "fr"},
		FallbackLocale:   "en",
	})

	tests := []struct {
		input    string
		expected Locale
	}{
		{"en-US,en;q=0.9", "en"},
		{"zh-CN,zh;q=0.9,en;q=0.8", "zh-CN"},
		{"fr-FR,fr;q=0.9,en;q=0.8", "fr"},
		{"", "en"},
		{"unsupported,en;q=0.9", "en"},
	}

	for _, tt := range tests {
		result := manager.ParseAcceptLanguage(tt.input)
		if result != tt.expected {
			t.Errorf("ParseAcceptLanguage(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestManager_LoadTranslationFile(t *testing.T) {
	// Create temporary translation file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.json")

	content := `{"test.key": "Test Value", "test.key2": "Test Value 2"}`
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	manager := NewManager(&Config{
		DefaultLocale:    "en",
		SupportedLocales: []Locale{"en"},
		FallbackLocale:   "en",
	})

	if err := manager.LoadTranslationFile("en", filePath); err != nil {
		t.Fatalf("failed to load translation file: %v", err)
	}

	if manager.GetTranslation("en", "test.key") != "Test Value" {
		t.Error("translation not loaded correctly")
	}
}

func TestManager_LoadTranslations(t *testing.T) {
	// Create temporary translations directory
	tmpDir := t.TempDir()

	// Create translation files
	enFile := filepath.Join(tmpDir, "messages.en.json")
	zhFile := filepath.Join(tmpDir, "messages.zh-CN.json")

	os.WriteFile(enFile, []byte(`{"welcome": "Welcome"}`), 0644)
	os.WriteFile(zhFile, []byte(`{"welcome": "欢迎"}`), 0644)

	manager := NewManager(&Config{
		DefaultLocale:    "en",
		SupportedLocales: []Locale{"en", "zh-CN"},
		FallbackLocale:   "en",
		TranslationsPath: tmpDir,
	})

	if manager.GetTranslation("en", "welcome") != "Welcome" {
		t.Error("English translation not loaded")
	}
	if manager.GetTranslation("zh-CN", "welcome") != "欢迎" {
		t.Error("Chinese translation not loaded")
	}
}

func TestManager_GetTranslations(t *testing.T) {
	manager := NewManager(nil)
	manager.SetTranslation("en", "key1", "value1")
	manager.SetTranslation("en", "key2", "value2")

	translations := manager.GetTranslations("en")
	if len(translations) != 2 {
		t.Errorf("expected 2 translations, got %d", len(translations))
	}
	if translations["key1"] != "value1" {
		t.Error("translation value mismatch")
	}
}

func TestManager_IsSupportedLocale(t *testing.T) {
	manager := NewManager(&Config{
		SupportedLocales: []Locale{"en", "zh-CN"},
	})

	if !manager.IsSupportedLocale("en") {
		t.Error("en should be supported")
	}
	if !manager.IsSupportedLocale("zh-CN") {
		t.Error("zh-CN should be supported")
	}
	if manager.IsSupportedLocale("fr") {
		t.Error("fr should not be supported")
	}
}
