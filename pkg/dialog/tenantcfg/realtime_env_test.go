package tenantcfg

import "testing"

func TestRealtimeOutputSampleRate_Default(t *testing.T) {
	if got := RealtimeOutputSampleRate(VoiceEnv{}); got != 24000 {
		t.Errorf("default output rate = %d, want 24000", got)
	}
}

func TestRealtimeTemperature_AliyunFloor(t *testing.T) {
	env := VoiceEnv{RealtimeProvider: "aliyun_omni"}
	if got := RealtimeTemperature(env); got != 0.6 {
		t.Errorf("aliyun floor = %v, want 0.6", got)
	}
}

func TestRealtimeInstructions(t *testing.T) {
	env := VoiceEnv{RealtimeConfigRaw: map[string]any{"instructions": "  hello  "}}
	if got := RealtimeInstructions(env); got != "hello" {
		t.Errorf("instructions = %q", got)
	}
}

func TestEffectiveRealtimeOperatorCore_PrefersAssistantPrompt(t *testing.T) {
	env := VoiceEnv{
		LLMInstructions:   "你是景区客服",
		RealtimeConfigRaw: map[string]any{"instructions": "legacy realtime"},
	}
	if got := EffectiveRealtimeOperatorCore(env); got != "你是景区客服" {
		t.Fatalf("got %q", got)
	}
	env.LLMInstructions = ""
	if got := EffectiveRealtimeOperatorCore(env); got != "legacy realtime" {
		t.Fatalf("fallback got %q", got)
	}
}

func TestRealtimeVoice(t *testing.T) {
	env := VoiceEnv{RealtimeConfigRaw: map[string]any{"voice": "Cherry"}}
	if got := RealtimeVoice(env); got != "Cherry" {
		t.Errorf("voice = %q", got)
	}
}
