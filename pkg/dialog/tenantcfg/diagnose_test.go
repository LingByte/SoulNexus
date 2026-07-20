package tenantcfg

import "testing"

func TestVoiceReadinessReason_RealtimeMissingKey(t *testing.T) {
	env := VoiceEnv{
		VoiceMode:         "realtime",
		RealtimeProvider:  "aliyun_omni",
		RealtimeConfigRaw: map[string]any{"provider": "aliyun_omni"},
	}
	if VoiceReady(env) {
		t.Fatal("expected not ready")
	}
	r := VoiceReadinessReason(env)
	if r == "" || r == "realtime voice config incomplete" {
		t.Fatalf("expected specific reason, got %q", r)
	}
}

func TestLLMReadinessReason_OpenAIMissingKey(t *testing.T) {
	env := VoiceEnv{LLMProvider: "openai"}
	r := LLMReadinessReason(env)
	if r == "" {
		t.Fatal("expected reason")
	}
}
