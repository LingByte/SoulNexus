package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMyMemoryTranslator_Translate(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"responseData": {
				"translatedText": "Hello",
				"match": 0.95
			},
			"quotaFinished": false,
			"mtLangSupported": true,
			"responseDetails": "",
			"responseStatus": 200
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	translator := NewMyMemoryTranslator("")
	translator.baseURL = server.URL

	result, err := translator.Translate("你好", "zh-CN", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", result)
	}
}

func TestMyMemoryTranslator_Translate_SameLanguage(t *testing.T) {
	translator := NewMyMemoryTranslator("")

	result, err := translator.Translate("Hello", "en", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", result)
	}
}

func TestMyMemoryTranslator_Translate_EmptyText(t *testing.T) {
	translator := NewMyMemoryTranslator("")

	_, err := translator.Translate("", "en", "zh-CN")
	if err == nil {
		t.Error("expected error for empty text")
	}
}

func TestMyMemoryTranslator_Translate_QuotaFinished(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{
			"responseData": {
				"translatedText": "",
				"match": 0
			},
			"quotaFinished": true,
			"mtLangSupported": true,
			"responseDetails": "",
			"responseStatus": 200
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	translator := NewMyMemoryTranslator("")
	translator.baseURL = server.URL

	_, err := translator.Translate("Hello", "en", "zh-CN")
	if err == nil {
		t.Error("expected error for quota finished")
	}
}

func TestMyMemoryTranslator_TranslateBatch(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		response := `{
			"responseData": {
				"translatedText": "Hello",
				"match": 0.95
			},
			"quotaFinished": false,
			"mtLangSupported": true,
			"responseDetails": "",
			"responseStatus": 200
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	translator := NewMyMemoryTranslator("")
	translator.baseURL = server.URL

	texts := []string{"你好", "世界"}
	results, err := translator.TranslateBatch(texts, "zh-CN", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

func TestNormalizeLangCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"en", "en"},
		{"EN", "en"},
		{"zh", "zh-CN"},
		{"zh-CN", "zh-CN"},
		{"zh-cn", "zh-CN"},
		{"zh-TW", "zh-TW"},
		{"en-US", "en"},
		{"fr-FR", "fr"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := normalizeLangCode(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeLangCode(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestGetSupportedLanguages(t *testing.T) {
	languages := GetSupportedLanguages()
	if len(languages) == 0 {
		t.Error("expected non-empty language list")
	}

	// Check for common languages
	hasEn := false
	hasZh := false
	for _, lang := range languages {
		if lang == "en" {
			hasEn = true
		}
		if lang == "zh-CN" {
			hasZh = true
		}
	}

	if !hasEn {
		t.Error("expected English to be in supported languages")
	}
	if !hasZh {
		t.Error("expected Chinese to be in supported languages")
	}
}
