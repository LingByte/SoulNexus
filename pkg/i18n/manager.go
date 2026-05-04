package i18n

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// Manager holds flattened translation strings keyed by locale (filename stem, e.g. en, zh-CN).
type Manager struct {
	translations map[string]map[string]string
}

// NewManager creates an empty translation manager. Config is reserved for future use.
func NewManager(_ interface{}) *Manager {
	return &Manager{
		translations: make(map[string]map[string]string),
	}
}

// LoadTranslations walks dir for *.yaml / *.yml files and merges nested keys into flat "a.b.c" form.
func (m *Manager) LoadTranslations(dir string) {
	if m == nil || dir == "" {
		return
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return
	}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}
		stem := strings.TrimSuffix(info.Name(), ext)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var root map[string]interface{}
		if err := yaml.Unmarshal(data, &root); err != nil || root == nil {
			return nil
		}
		flat := make(map[string]string)
		flatten("", root, flat)
		if m.translations[stem] == nil {
			m.translations[stem] = make(map[string]string)
		}
		for k, v := range flat {
			m.translations[stem][k] = v
		}
		return nil
	})
}

func flatten(prefix string, in map[string]interface{}, out map[string]string) {
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch t := v.(type) {
		case map[string]interface{}:
			flatten(key, t, out)
		case string:
			out[key] = t
		default:
			// Ignore non-string leaves (numbers, bools) for validation copy.
		}
	}
}

// GetTranslation returns the message for key in locale, falling back to en then the key itself.
func (m *Manager) GetTranslation(locale Locale, key string) string {
	if m == nil || key == "" {
		return key
	}
	loc := normalizeLocale(string(locale))
	candidates := []string{loc}
	if strings.HasPrefix(strings.ToLower(loc), "zh") && loc != "zh-CN" {
		candidates = append(candidates, "zh-CN")
	}
	if loc != "en" {
		candidates = append(candidates, "en")
	}
	for _, l := range candidates {
		dict := m.translations[l]
		if dict == nil {
			continue
		}
		if msg, ok := dict[key]; ok && msg != "" {
			return msg
		}
	}
	return key
}

// Middleware parses Accept-Language and stores a normalized locale on the Gin context.
func Middleware(_ *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(ginLocaleKey, normalizeLocale(c.GetHeader("Accept-Language")))
		c.Next()
	}
}

// GetLocaleFromGin returns the locale set by Middleware, or derives it from Accept-Language.
func GetLocaleFromGin(c *gin.Context) Locale {
	if c == nil {
		return Locale("en")
	}
	if v, ok := c.Get(ginLocaleKey); ok {
		if s, ok := v.(string); ok && s != "" {
			return Locale(normalizeLocale(s))
		}
	}
	return Locale(normalizeLocale(c.GetHeader("Accept-Language")))
}
