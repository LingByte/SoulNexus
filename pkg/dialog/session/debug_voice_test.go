package session

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
)

func TestEffectiveVoiceEnv_DebugWebSocketForcesPipeline(t *testing.T) {
	s := &Session{
		ID:        "sess-1",
		CallID:    "sess-1",
		Transport: TransportWebSocket,
	}
	RegisterDebugCall(s.CallID)
	t.Cleanup(func() { UnregisterDebugCall(s.CallID) })

	env := tenantcfg.VoiceEnv{VoiceMode: "realtime"}
	got := EffectiveVoiceEnv(s, env)
	if got.VoiceMode != "pipeline" {
		t.Fatalf("VoiceMode = %q, want pipeline", got.VoiceMode)
	}
}

func TestEffectiveVoiceEnv_NonDebugUnchanged(t *testing.T) {
	s := &Session{ID: "sess-2", CallID: "sess-2", Transport: TransportWebSocket}
	env := tenantcfg.VoiceEnv{VoiceMode: "realtime"}
	got := EffectiveVoiceEnv(s, env)
	if got.VoiceMode != "realtime" {
		t.Fatalf("VoiceMode = %q, want realtime", got.VoiceMode)
	}
}

func TestEffectiveVoiceEnv_DebugTextUnchanged(t *testing.T) {
	s := &Session{ID: "sess-3", CallID: "sess-3", Transport: TransportText}
	RegisterDebugCall(s.CallID)
	t.Cleanup(func() { UnregisterDebugCall(s.CallID) })

	env := tenantcfg.VoiceEnv{VoiceMode: "realtime"}
	got := EffectiveVoiceEnv(s, env)
	if got.VoiceMode != "realtime" {
		t.Fatalf("VoiceMode = %q, want realtime for text debug", got.VoiceMode)
	}
}
