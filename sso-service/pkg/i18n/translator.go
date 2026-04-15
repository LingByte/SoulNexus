package i18n

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// MyMemoryTranslator implements Translator using MyMemory Translation API
type MyMemoryTranslator struct {
	client    *http.Client
	baseURL   string
	email     string // Optional: for higher rate limits
	userAgent string
}

// MyMemoryResponse represents the API response
type MyMemoryResponse struct {
	ResponseData struct {
		TranslatedText string  `json:"translatedText"`
		Match          float64 `json:"match"`
	} `json:"responseData"`
	QuotaFinished   bool   `json:"quotaFinished"`
	MTLangSupported bool   `json:"mtLangSupported"`
	ResponseDetails string `json:"responseDetails"`
	ResponseStatus  int    `json:"responseStatus"`
}

// NewMyMemoryTranslator creates a new MyMemory translator
func NewMyMemoryTranslator(email string) *MyMemoryTranslator {
	return &MyMemoryTranslator{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL:   "https://api.mymemory.translated.net/get",
		email:     email,
		userAgent: "LingFramework/1.0",
	}
}

// Translate translates text using MyMemory API
func (t *MyMemoryTranslator) Translate(text, from, to string) (string, error) {
	if text == "" {
		return "", fmt.Errorf("text cannot be empty")
	}

	// Normalize language codes
	from = normalizeLangCode(from)
	to = normalizeLangCode(to)

	if from == to {
		return text, nil
	}

	// Build request URL
	params := url.Values{}
	params.Set("q", text)
	params.Set("langpair", fmt.Sprintf("%s|%s", from, to))
	if t.email != "" {
		params.Set("de", t.email) // Email for higher rate limits
	}

	reqURL := fmt.Sprintf("%s?%s", t.baseURL, params.Encode())

	// Create request
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", t.userAgent)

	// Execute request
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var apiResp MyMemoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors
	if apiResp.ResponseStatus != 200 {
		return "", fmt.Errorf("API error: %s (status: %d)", apiResp.ResponseDetails, apiResp.ResponseStatus)
	}

	if apiResp.QuotaFinished {
		logger.Warn("MyMemory translation quota finished")
		return "", fmt.Errorf("translation quota finished")
	}

	translatedText := apiResp.ResponseData.TranslatedText
	if translatedText == "" {
		return text, nil // Return original if translation is empty
	}

	logger.Debug("translation completed",
		zap.String("from", from),
		zap.String("to", to),
		zap.Float64("match", apiResp.ResponseData.Match))

	return translatedText, nil
}

// TranslateBatch translates multiple texts
func (t *MyMemoryTranslator) TranslateBatch(texts []string, from, to string) ([]string, error) {
	results := make([]string, len(texts))
	for i, text := range texts {
		translated, err := t.Translate(text, from, to)
		if err != nil {
			return nil, fmt.Errorf("failed to translate text at index %d: %w", i, err)
		}
		results[i] = translated
		// Add small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}
	return results, nil
}

// normalizeLangCode normalizes language code to MyMemory format
func normalizeLangCode(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))

	// Handle common language codes
	langMap := map[string]string{
		"zh":    "zh-CN",
		"zh-cn": "zh-CN",
		"zh-tw": "zh-TW",
		"zh-hk": "zh-TW",
		"en":    "en",
		"en-us": "en",
		"en-gb": "en",
		"es":    "es",
		"fr":    "fr",
		"de":    "de",
		"it":    "it",
		"pt":    "pt",
		"ru":    "ru",
		"ja":    "ja",
		"ko":    "ko",
		"ar":    "ar",
		"hi":    "hi",
		"th":    "th",
		"vi":    "vi",
		"id":    "id",
		"tr":    "tr",
		"pl":    "pl",
		"nl":    "nl",
		"sv":    "sv",
		"da":    "da",
		"fi":    "fi",
		"no":    "no",
		"cs":    "cs",
		"hu":    "hu",
		"ro":    "ro",
		"el":    "el",
		"he":    "he",
		"uk":    "uk",
		"bg":    "bg",
		"hr":    "hr",
		"sk":    "sk",
		"sl":    "sl",
		"et":    "et",
		"lv":    "lv",
		"lt":    "lt",
		"mt":    "mt",
		"ga":    "ga",
		"cy":    "cy",
	}

	if normalized, ok := langMap[lang]; ok {
		return normalized
	}

	// If contains hyphen, try to normalize
	if strings.Contains(lang, "-") {
		parts := strings.Split(lang, "-")
		if len(parts) == 2 {
			// Try language-region format
			if normalized, ok := langMap[parts[0]]; ok {
				return normalized
			}
		}
	}

	// Return as-is if not found in map
	return lang
}

// GetSupportedLanguages returns list of supported language codes
func GetSupportedLanguages() []string {
	return []string{
		"en", "zh-CN", "zh-TW", "es", "fr", "de", "it", "pt", "ru",
		"ja", "ko", "ar", "hi", "th", "vi", "id", "tr", "pl", "nl",
		"sv", "da", "fi", "no", "cs", "hu", "ro", "el", "he", "uk",
		"bg", "hr", "sk", "sl", "et", "lv", "lt", "mt", "ga", "cy",
	}
}
