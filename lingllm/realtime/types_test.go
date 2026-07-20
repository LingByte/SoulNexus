package realtime

import (
	"testing"
)

func TestEventType(t *testing.T) {
	tests := []struct {
		name      string
		eventType EventType
		expected  string
	}{
		{"SessionOpen", EventSessionOpen, "session.open"},
		{"SessionClose", EventSessionClose, "session.close"},
		{"UserTranscript", EventUserTranscript, "user.transcript"},
		{"UserSpeechStarted", EventUserSpeechStarted, "user.speech.started"},
		{"UserSpeechEnded", EventUserSpeechEnded, "user.speech.ended"},
		{"AssistantText", EventAssistantText, "assistant.text"},
		{"AssistantAudio", EventAssistantAudio, "assistant.audio"},
		{"AssistantTurnEnd", EventAssistantTurnEnd, "assistant.turn.end"},
		{"Error", EventError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.eventType) != tt.expected {
				t.Errorf("EventType = %s, want %s", tt.eventType, tt.expected)
			}
		})
	}
}

func TestEventStruct(t *testing.T) {
	event := Event{
		Type:    EventUserTranscript,
		Text:    "hello world",
		Final:   true,
		AudioPC: []byte{0x00, 0x01, 0x02},
		Vendor:  "test_vendor",
	}

	if event.Type != EventUserTranscript {
		t.Errorf("Type = %s, want %s", event.Type, EventUserTranscript)
	}

	if event.Text != "hello world" {
		t.Errorf("Text = %s, want 'hello world'", event.Text)
	}

	if !event.Final {
		t.Error("Final should be true")
	}

	if len(event.AudioPC) != 3 {
		t.Errorf("AudioPC length = %d, want 3", len(event.AudioPC))
	}

	if event.Vendor != "test_vendor" {
		t.Errorf("Vendor = %s, want 'test_vendor'", event.Vendor)
	}
}

func TestOptionsDefaults(t *testing.T) {
	opts := Options{
		SystemPrompt:     "You are helpful",
		Voice:            "alloy",
		InputSampleRate:  0,
		OutputSampleRate: 0,
		OnEvent: func(e Event) {
			// no-op
		},
	}

	if opts.SystemPrompt != "You are helpful" {
		t.Errorf("SystemPrompt = %s, want 'You are helpful'", opts.SystemPrompt)
	}

	if opts.Voice != "alloy" {
		t.Errorf("Voice = %s, want 'alloy'", opts.Voice)
	}

	if opts.OnEvent == nil {
		t.Error("OnEvent should not be nil")
	}
}

func TestRegisterAndLookup(t *testing.T) {
	// Create a test provider
	testProvider := func(cfg map[string]any, opts Options) (Agent, error) {
		return nil, nil
	}

	// Register with multiple slugs
	Register(testProvider, "test_provider", "test_alias", "another_alias")

	// Test lookup with different slugs
	tests := []struct {
		slug string
		want bool
	}{
		{"test_provider", true},
		{"test_alias", true},
		{"another_alias", true},
		{"TEST_PROVIDER", true}, // Should be case-insensitive
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.slug, func(t *testing.T) {
			got := Lookup(tt.slug) != nil
			if got != tt.want {
				t.Errorf("Lookup(%s) found = %v, want %v", tt.slug, got, tt.want)
			}
		})
	}
}

func TestRegisteredProviders(t *testing.T) {
	// Register test providers
	testProvider := func(cfg map[string]any, opts Options) (Agent, error) {
		return nil, nil
	}
	Register(testProvider, "test_provider_1", "test_provider_2")

	providers := RegisteredProviders()
	if len(providers) == 0 {
		t.Error("RegisteredProviders should return at least one provider")
	}

	// Check that test providers are registered
	providerMap := make(map[string]bool)
	for _, p := range providers {
		providerMap[p] = true
	}

	if !providerMap["test_provider_1"] {
		t.Error("Expected provider test_provider_1 not found in registered providers")
	}

	if !providerMap["test_provider_2"] {
		t.Error("Expected provider test_provider_2 not found in registered providers")
	}
}

func TestErrUnknownProvider(t *testing.T) {
	err := &ErrUnknownProvider{Provider: "unknown"}
	errMsg := err.Error()

	if errMsg == "" {
		t.Error("Error message should not be empty")
	}

	if !contains(errMsg, "unknown") {
		t.Errorf("Error message should contain 'unknown', got: %s", errMsg)
	}

	if !contains(errMsg, "registered") {
		t.Errorf("Error message should contain 'registered', got: %s", errMsg)
	}
}

func TestErrAgentClosed(t *testing.T) {
	if ErrAgentClosed == nil {
		t.Error("ErrAgentClosed should not be nil")
	}

	if ErrAgentClosed.Error() != "realtime: agent already closed" {
		t.Errorf("ErrAgentClosed message = %s, want 'realtime: agent already closed'", ErrAgentClosed.Error())
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
