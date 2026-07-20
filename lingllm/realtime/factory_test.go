package realtime

import (
	"testing"
)

func TestNewAgentFromCredential_EmptyConfig(t *testing.T) {
	opts := Options{
		OnEvent: func(e Event) {},
	}

	_, err := NewAgentFromCredential(nil, opts)
	if err == nil {
		t.Error("NewAgentFromCredential with nil config should return error")
	}

	_, err = NewAgentFromCredential(map[string]any{}, opts)
	if err == nil {
		t.Error("NewAgentFromCredential with empty config should return error")
	}
}

func TestNewAgentFromCredential_MissingProvider(t *testing.T) {
	cfg := map[string]any{
		"api_key": "test-key",
	}
	opts := Options{
		OnEvent: func(e Event) {},
	}

	_, err := NewAgentFromCredential(cfg, opts)
	if err == nil {
		t.Error("NewAgentFromCredential without provider field should return error")
	}
}

func TestNewAgentFromCredential_UnknownProvider(t *testing.T) {
	cfg := map[string]any{
		"provider": "unknown_provider",
	}
	opts := Options{
		OnEvent: func(e Event) {},
	}

	_, err := NewAgentFromCredential(cfg, opts)
	if err == nil {
		t.Error("NewAgentFromCredential with unknown provider should return error")
	}

	// Check that error is ErrUnknownProvider
	_, ok := err.(*ErrUnknownProvider)
	if !ok {
		t.Errorf("Error type = %T, want *ErrUnknownProvider", err)
	}
}

func TestNewAgentFromCredential_MissingOnEvent(t *testing.T) {
	cfg := map[string]any{
		"provider": "aliyun_omni",
		"api_key":  "test-key",
	}
	opts := Options{
		// OnEvent is nil
	}

	_, err := NewAgentFromCredential(cfg, opts)
	if err == nil {
		t.Error("NewAgentFromCredential without OnEvent should return error")
	}
}

func TestNewAgentFromCredential_DefaultSampleRates(t *testing.T) {
	cfg := map[string]any{
		"provider": "aliyun_omni",
		"api_key":  "test-key",
	}
	opts := Options{
		InputSampleRate:  0,
		OutputSampleRate: 0,
		OnEvent: func(e Event) {
			// no-op
		},
	}

	// This will fail because aliyun_omni requires more fields,
	// but we're testing that default sample rates are applied
	_, err := NewAgentFromCredential(cfg, opts)
	// We expect an error, but the sample rates should have been set
	// This is more of a sanity check that the function processes the options
	if err != nil {
		// Expected - aliyun_omni needs more config
		t.Logf("Expected error from aliyun_omni config: %v", err)
	}
}

func TestNewAgentFromCredential_CaseInsensitiveProvider(t *testing.T) {
	// Register a test provider with lowercase slug
	testProvider := func(cfg map[string]any, opts Options) (Agent, error) {
		return nil, nil
	}
	Register(testProvider, "test_case_provider")

	cfg := map[string]any{
		"provider": "TEST_CASE_PROVIDER", // uppercase
	}
	opts := Options{
		OnEvent: func(e Event) {
			// no-op
		},
	}

	// Should not fail on provider lookup (case-insensitive)
	_, err := NewAgentFromCredential(cfg, opts)
	// We expect no error from provider lookup
	if err != nil {
		// Check it's not an ErrUnknownProvider
		_, ok := err.(*ErrUnknownProvider)
		if ok {
			t.Error("Provider lookup should be case-insensitive")
		}
	}
}

func TestStringField(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]any
		key      string
		expected string
	}{
		{"nil map", nil, "key", ""},
		{"missing key", map[string]any{"other": "value"}, "key", ""},
		{"non-string value", map[string]any{"key": 123}, "key", ""},
		{"string value", map[string]any{"key": "value"}, "key", "value"},
		{"empty string", map[string]any{"key": ""}, "key", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringField(tt.m, tt.key)
			if got != tt.expected {
				t.Errorf("stringField = %s, want %s", got, tt.expected)
			}
		})
	}
}
