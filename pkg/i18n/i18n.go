package i18n

import (
	"encoding/json"
	"fmt"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Locale represents a locale identifier (e.g., "en", "zh-CN", "en-US")
type Locale string

// DefaultLocale is the default locale
const DefaultLocale Locale = "en"

// Manager handles internationalization
type Manager struct {
	mu               sync.RWMutex
	translations     map[Locale]map[string]string
	defaultLocale    Locale
	supportedLocales []Locale
	fallbackLocale   Locale
	translator       Translator
}

// Translator interface for translation services
type Translator interface {
	Translate(text, from, to string) (string, error)
}

// Config represents i18n configuration
type Config struct {
	DefaultLocale    Locale
	SupportedLocales []Locale
	FallbackLocale   Locale
	TranslationsPath string
	Translator       Translator
}

// NewManager creates a new i18n manager
func NewManager(config *Config) *Manager {
	if config == nil {
		config = &Config{
			DefaultLocale:    DefaultLocale,
			SupportedLocales: []Locale{DefaultLocale},
			FallbackLocale:   DefaultLocale,
		}
	}

	if config.DefaultLocale == "" {
		config.DefaultLocale = DefaultLocale
	}
	if config.FallbackLocale == "" {
		config.FallbackLocale = config.DefaultLocale
	}
	if len(config.SupportedLocales) == 0 {
		config.SupportedLocales = []Locale{config.DefaultLocale}
	}

	manager := &Manager{
		translations:     make(map[Locale]map[string]string),
		defaultLocale:    config.DefaultLocale,
		supportedLocales: config.SupportedLocales,
		fallbackLocale:   config.FallbackLocale,
		translator:       config.Translator,
	}

	// Load translations if path is provided
	if config.TranslationsPath != "" {
		if err := manager.LoadTranslations(config.TranslationsPath); err != nil {
			logger.Warn("failed to load translations", zap.Error(err))
		}
	}

	return manager
}

// LoadTranslations loads translation files from directory
func (m *Manager) LoadTranslations(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("translations path does not exist: %s", path)
	}

	// Initialize translations map for each locale
	for _, locale := range m.supportedLocales {
		if m.translations[locale] == nil {
			m.translations[locale] = make(map[string]string)
		}
	}

	// Walk through translation directory
	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process JSON files
		if info.IsDir() || !strings.HasSuffix(filePath, ".json") {
			return nil
		}

		// Extract locale from filename (e.g., "messages.zh-CN.json" -> "zh-CN")
		baseName := strings.TrimSuffix(info.Name(), ".json")
		parts := strings.Split(baseName, ".")
		if len(parts) < 2 {
			return nil
		}

		locale := Locale(parts[len(parts)-1])
		if !m.isSupportedLocale(locale) {
			logger.Debug("skipping unsupported locale", zap.String("locale", string(locale)))
			return nil
		}

		// Read and parse JSON file
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read translation file %s: %w", filePath, err)
		}

		var translations map[string]string
		if err := json.Unmarshal(data, &translations); err != nil {
			return fmt.Errorf("failed to parse translation file %s: %w", filePath, err)
		}

		// Merge translations
		if m.translations[locale] == nil {
			m.translations[locale] = make(map[string]string)
		}
		for k, v := range translations {
			m.translations[locale][k] = v
		}

		logger.Info("loaded translations", zap.String("locale", string(locale)), zap.Int("count", len(translations)))
		return nil
	})
}

// LoadTranslationFile loads a single translation file
func (m *Manager) LoadTranslationFile(locale Locale, filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isSupportedLocale(locale) {
		return fmt.Errorf("unsupported locale: %s", locale)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read translation file: %w", err)
	}

	var translations map[string]string
	if err := json.Unmarshal(data, &translations); err != nil {
		return fmt.Errorf("failed to parse translation file: %w", err)
	}

	if m.translations[locale] == nil {
		m.translations[locale] = make(map[string]string)
	}
	for k, v := range translations {
		m.translations[locale][k] = v
	}

	return nil
}

// SetTranslation sets a translation key-value pair
func (m *Manager) SetTranslation(locale Locale, key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.translations[locale] == nil {
		m.translations[locale] = make(map[string]string)
	}
	m.translations[locale][key] = value
}

// GetTranslation gets a translation for a key
func (m *Manager) GetTranslation(locale Locale, key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try requested locale
	if translations, ok := m.translations[locale]; ok {
		if value, ok := translations[key]; ok {
			return value
		}
	}

	// Try fallback locale
	if locale != m.fallbackLocale {
		if translations, ok := m.translations[m.fallbackLocale]; ok {
			if value, ok := translations[key]; ok {
				return value
			}
		}
	}

	// Return key if not found
	return key
}

// T translates a key with optional arguments
func (m *Manager) T(locale Locale, key string, args ...interface{}) string {
	translation := m.GetTranslation(locale, key)
	if len(args) > 0 {
		return fmt.Sprintf(translation, args...)
	}
	return translation
}

// Translate uses external translator service
func (m *Manager) Translate(text string, from, to string) (string, error) {
	if m.translator == nil {
		return text, fmt.Errorf("translator not configured")
	}
	return m.translator.Translate(text, from, to)
}

// DetectLocale detects locale from language string (e.g., "en-US", "zh-CN")
func (m *Manager) DetectLocale(lang string) Locale {
	if lang == "" {
		return m.defaultLocale
	}

	lang = strings.TrimSpace(lang)

	// Try exact match first (case-insensitive)
	for _, supported := range m.supportedLocales {
		if strings.EqualFold(string(supported), lang) {
			return supported
		}
	}

	// Normalize to lowercase for further processing
	langLower := strings.ToLower(lang)

	// Try language code only (e.g., "en" from "en-US")
	parts := strings.Split(langLower, "-")
	if len(parts) > 0 {
		for _, supported := range m.supportedLocales {
			supportedLower := strings.ToLower(string(supported))
			if strings.HasPrefix(supportedLower, parts[0]+"-") || supportedLower == parts[0] {
				return supported
			}
		}
	}

	// Try underscore separator (e.g., "zh_CN")
	parts = strings.Split(langLower, "_")
	if len(parts) >= 2 {
		// Try format: "zh-CN"
		candidate := Locale(strings.ToLower(parts[0]) + "-" + strings.ToUpper(parts[1]))
		if m.isSupportedLocale(candidate) {
			return candidate
		}
		// Try format: "zh-cn"
		candidate = Locale(parts[0] + "-" + parts[1])
		if m.isSupportedLocale(candidate) {
			return candidate
		}
	}
	if len(parts) > 0 {
		for _, supported := range m.supportedLocales {
			supportedLower := strings.ToLower(string(supported))
			if strings.HasPrefix(supportedLower, parts[0]+"-") || supportedLower == parts[0] {
				return supported
			}
		}
	}

	return m.defaultLocale
}

// ParseAcceptLanguage parses Accept-Language header
func (m *Manager) ParseAcceptLanguage(acceptLang string) Locale {
	if acceptLang == "" {
		return m.defaultLocale
	}

	// Parse Accept-Language header (e.g., "en-US,en;q=0.9,zh-CN;q=0.8")
	languages := strings.Split(acceptLang, ",")
	for _, lang := range languages {
		// Extract language code (remove quality value)
		parts := strings.Split(strings.TrimSpace(lang), ";")
		langCode := strings.TrimSpace(parts[0])

		locale := m.DetectLocale(langCode)
		// Return if locale is supported (even if it's the default)
		if m.isSupportedLocale(locale) {
			return locale
		}
	}

	return m.defaultLocale
}

// GetDefaultLocale returns the default locale
func (m *Manager) GetDefaultLocale() Locale {
	return m.defaultLocale
}

// GetSupportedLocales returns supported locales
func (m *Manager) GetSupportedLocales() []Locale {
	return m.supportedLocales
}

// IsSupportedLocale checks if locale is supported
func (m *Manager) IsSupportedLocale(locale Locale) bool {
	return m.isSupportedLocale(locale)
}

func (m *Manager) isSupportedLocale(locale Locale) bool {
	for _, supported := range m.supportedLocales {
		if supported == locale {
			return true
		}
	}
	return false
}

// GetTranslations returns all translations for a locale
func (m *Manager) GetTranslations(locale Locale) map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	translations := make(map[string]string)
	if ts, ok := m.translations[locale]; ok {
		for k, v := range ts {
			translations[k] = v
		}
	}
	return translations
}
