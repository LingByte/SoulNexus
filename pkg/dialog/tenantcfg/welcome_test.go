package tenantcfg

import "testing"

func TestResolvedAssistantWelcome(t *testing.T) {
	t.Run("top level", func(t *testing.T) {
		got := ResolvedAssistantWelcome(VoiceEnv{AssistantWelcome: "  hello  "})
		if got != "hello" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("realtime config fallback", func(t *testing.T) {
		got := ResolvedAssistantWelcome(VoiceEnv{
			RealtimeConfigRaw: map[string]any{"welcome": "from realtime"},
		})
		if got != "from realtime" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("top level wins", func(t *testing.T) {
		got := ResolvedAssistantWelcome(VoiceEnv{
			AssistantWelcome:  "primary",
			RealtimeConfigRaw: map[string]any{"welcome": "fallback"},
		})
		if got != "primary" {
			t.Fatalf("got %q", got)
		}
	})
}
