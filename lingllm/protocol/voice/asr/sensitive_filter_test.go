package asr

import (
	"context"
	"testing"
)

func TestSensitiveFilterComponentCreation(t *testing.T) {
	config := DefaultSensitiveFilterConfig()
	filter, err := NewSensitiveFilterComponent(config)

	if err != nil {
		t.Errorf("NewSensitiveFilterComponent failed: %v", err)
	}

	if filter == nil {
		t.Error("SensitiveFilterComponent should not be nil")
	}

	if filter.Name() != "sensitive_filter" {
		t.Errorf("Name = %s, want 'sensitive_filter'", filter.Name())
	}
}

func TestSensitiveFilterComponentInvalidPattern(t *testing.T) {
	config := SensitiveFilterConfig{
		Blacklist: []string{"[invalid(regex"},
	}

	_, err := NewSensitiveFilterComponent(config)
	if err == nil {
		t.Error("NewSensitiveFilterComponent with invalid pattern should return error")
	}
}

func TestSensitiveFilterComponentFilterEmoji(t *testing.T) {
	config := DefaultSensitiveFilterConfig()
	config.FilterEmoji = true
	filter, _ := NewSensitiveFilterComponent(config)

	text := "Hello 😀 World 🎉 Test"
	result, shouldContinue, err := filter.Process(context.Background(), text)

	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if !shouldContinue {
		t.Error("shouldContinue should be true")
	}

	resultText, ok := result.(string)
	if !ok {
		t.Errorf("Result type = %T, want string", result)
	}

	// Check that emoji are replaced
	if resultText == text {
		t.Error("Emoji should have been filtered")
	}

	if !contains(resultText, "*") {
		t.Error("Result should contain replacement character")
	}
}

func TestSensitiveFilterComponentNoEmojiFilter(t *testing.T) {
	config := DefaultSensitiveFilterConfig()
	config.FilterEmoji = false
	filter, _ := NewSensitiveFilterComponent(config)

	text := "Hello 😀 World"
	result, _, _ := filter.Process(context.Background(), text)

	resultText, _ := result.(string)
	if resultText != text {
		t.Error("Text should not be modified when emoji filtering is disabled")
	}
}

func TestSensitiveFilterComponentBlacklist(t *testing.T) {
	config := SensitiveFilterConfig{
		Blacklist:   []string{"password", "secret"},
		FilterEmoji: false,
	}
	filter, _ := NewSensitiveFilterComponent(config)

	text := "my password is secret123"
	result, _, _ := filter.Process(context.Background(), text)

	resultText, _ := result.(string)

	// Check that blacklisted words are replaced
	if contains(resultText, "password") {
		t.Error("'password' should have been filtered")
	}

	if contains(resultText, "secret") {
		t.Error("'secret' should have been filtered")
	}

	if !contains(resultText, "*") {
		t.Error("Result should contain replacement character")
	}
}

func TestSensitiveFilterComponentWhitelist(t *testing.T) {
	config := SensitiveFilterConfig{
		Blacklist:   []string{"\\btest\\b"}, // Word boundary to match only "test" not "test123"
		Whitelist:   []string{"test123"},
		FilterEmoji: false,
	}
	filter, _ := NewSensitiveFilterComponent(config)

	text := "test this and test123 that"
	result, _, _ := filter.Process(context.Background(), text)

	resultText, _ := result.(string)

	// "test" should be filtered but "test123" should not
	if !contains(resultText, "test123") {
		t.Error("'test123' should not have been filtered (whitelisted)")
	}
}

func TestSensitiveFilterComponentCaseSensitive(t *testing.T) {
	config := SensitiveFilterConfig{
		Blacklist:     []string{"password"},
		CaseSensitive: false,
		FilterEmoji:   false,
	}
	filter, _ := NewSensitiveFilterComponent(config)

	text := "My PASSWORD is secret"
	result, _, _ := filter.Process(context.Background(), text)

	resultText, _ := result.(string)

	// "PASSWORD" should be filtered even though pattern is lowercase
	if contains(resultText, "PASSWORD") {
		t.Error("'PASSWORD' should have been filtered (case-insensitive)")
	}
}

func TestSensitiveFilterComponentCaseSensitiveStrict(t *testing.T) {
	config := SensitiveFilterConfig{
		Blacklist:     []string{"password"},
		CaseSensitive: true,
		FilterEmoji:   false,
	}
	filter, _ := NewSensitiveFilterComponent(config)

	text := "My PASSWORD is secret"
	result, _, _ := filter.Process(context.Background(), text)

	resultText, _ := result.(string)

	// "PASSWORD" should NOT be filtered (case-sensitive)
	if !contains(resultText, "PASSWORD") {
		t.Error("'PASSWORD' should not have been filtered (case-sensitive)")
	}
}

func TestSensitiveFilterComponentCustomReplacement(t *testing.T) {
	config := SensitiveFilterConfig{
		Blacklist:   []string{"secret"},
		ReplaceWith: "#",
		FilterEmoji: false,
	}
	filter, _ := NewSensitiveFilterComponent(config)

	text := "this is secret"
	result, _, _ := filter.Process(context.Background(), text)

	resultText, _ := result.(string)

	// Check that custom replacement is used
	if !contains(resultText, "#") {
		t.Error("Result should contain custom replacement character '#'")
	}

	if contains(resultText, "*") {
		t.Error("Result should not contain default replacement character '*'")
	}
}

func TestSensitiveFilterComponentEmptyText(t *testing.T) {
	config := DefaultSensitiveFilterConfig()
	filter, _ := NewSensitiveFilterComponent(config)

	text := ""
	result, shouldContinue, err := filter.Process(context.Background(), text)

	if err != nil {
		t.Errorf("Process failed: %v", err)
	}

	if !shouldContinue {
		t.Error("shouldContinue should be true")
	}

	resultText, _ := result.(string)
	if resultText != "" {
		t.Error("Empty text should remain empty")
	}
}

func TestSensitiveFilterComponentInvalidDataType(t *testing.T) {
	config := DefaultSensitiveFilterConfig()
	filter, _ := NewSensitiveFilterComponent(config)

	_, _, err := filter.Process(context.Background(), 123)
	if err == nil {
		t.Error("Process with invalid data type should return error")
	}
}

func TestSensitiveFilterComponentSetBlacklist(t *testing.T) {
	config := DefaultSensitiveFilterConfig()
	filter, _ := NewSensitiveFilterComponent(config)

	err := filter.SetBlacklist([]string{"password", "secret"})
	if err != nil {
		t.Errorf("SetBlacklist failed: %v", err)
	}

	text := "my password is secret"
	result, _, _ := filter.Process(context.Background(), text)

	resultText, _ := result.(string)
	if contains(resultText, "password") || contains(resultText, "secret") {
		t.Error("Blacklisted words should have been filtered")
	}
}

func TestSensitiveFilterComponentSetBlacklistInvalidPattern(t *testing.T) {
	config := DefaultSensitiveFilterConfig()
	filter, _ := NewSensitiveFilterComponent(config)

	err := filter.SetBlacklist([]string{"[invalid(regex"})
	if err == nil {
		t.Error("SetBlacklist with invalid pattern should return error")
	}
}

func TestSensitiveFilterComponentSetWhitelist(t *testing.T) {
	config := SensitiveFilterConfig{
		Blacklist:   []string{"\\btest\\b"}, // Word boundary to match only "test" not "test123"
		FilterEmoji: false,
	}
	filter, _ := NewSensitiveFilterComponent(config)

	err := filter.SetWhitelist([]string{"test123"})
	if err != nil {
		t.Errorf("SetWhitelist failed: %v", err)
	}

	text := "test this and test123 that"
	result, _, _ := filter.Process(context.Background(), text)

	resultText, _ := result.(string)
	if !contains(resultText, "test123") {
		t.Error("'test123' should not have been filtered (whitelisted)")
	}
}

func TestSensitiveFilterComponentSetFilterEmoji(t *testing.T) {
	config := DefaultSensitiveFilterConfig()
	config.FilterEmoji = true
	filter, _ := NewSensitiveFilterComponent(config)

	// First, verify emoji are filtered
	text := "Hello 😀"
	result1, _, _ := filter.Process(context.Background(), text)
	resultText1, _ := result1.(string)

	if resultText1 == text {
		t.Error("Emoji should have been filtered initially")
	}

	// Disable emoji filtering
	filter.SetFilterEmoji(false)

	// Now emoji should not be filtered
	result2, _, _ := filter.Process(context.Background(), text)
	resultText2, _ := result2.(string)

	if resultText2 != text {
		t.Error("Emoji should not be filtered after SetFilterEmoji(false)")
	}
}

func TestSensitiveFilterComponentSetReplaceWith(t *testing.T) {
	config := SensitiveFilterConfig{
		Blacklist:   []string{"secret"},
		ReplaceWith: "*",
		FilterEmoji: false,
	}
	filter, _ := NewSensitiveFilterComponent(config)

	// Change replacement character
	filter.SetReplaceWith("#")

	text := "this is secret"
	result, _, _ := filter.Process(context.Background(), text)

	resultText, _ := result.(string)

	if !contains(resultText, "#") {
		t.Error("Result should contain new replacement character '#'")
	}

	if contains(resultText, "*") {
		t.Error("Result should not contain old replacement character '*'")
	}
}

func TestSensitiveFilterComponentRegexPattern(t *testing.T) {
	config := SensitiveFilterConfig{
		Blacklist:   []string{"\\d{3}-\\d{4}"}, // Phone number pattern
		FilterEmoji: false,
	}
	filter, _ := NewSensitiveFilterComponent(config)

	text := "Call me at 123-4567 or 999-8888"
	result, _, _ := filter.Process(context.Background(), text)

	resultText, _ := result.(string)

	// Phone numbers should be filtered
	if contains(resultText, "123-4567") || contains(resultText, "999-8888") {
		t.Error("Phone numbers matching regex should have been filtered")
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
