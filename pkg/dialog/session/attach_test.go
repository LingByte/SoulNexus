package session_test

import (
	"context"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/session"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
)

func TestResolveMode_RealtimeWhenReady(t *testing.T) {
	env := tenantcfg.VoiceEnv{
		VoiceMode:         "realtime",
		RealtimeProvider:  "qwen",
		RealtimeConfigRaw: map[string]any{"provider": "qwen", "apiKey": "k"},
	}
	if got := session.ResolveMode(env); got != engine.ModeRealtime {
		t.Errorf("ResolveMode = %q, want realtime", got)
	}
}

func TestResolveMode_DefaultsCascadedNative(t *testing.T) {
	cases := []tenantcfg.VoiceEnv{
		{},
		{VoiceMode: "pipeline"},
		{VoiceMode: "realtime"}, // not ready → cascaded-native
		{
			VoiceMode:         "REALTIME",
			RealtimeProvider:  "qwen",
			RealtimeConfigRaw: map[string]any{"provider": "qwen"}, // no apiKey
		},
	}
	for i, env := range cases {
		if got := session.ResolveMode(env); got != engine.ModeCascadedNative {
			t.Errorf("case %d: ResolveMode = %q, want cascaded-native", i, got)
		}
	}
}

func TestAttachEngine_NilPort(t *testing.T) {
	_, err := session.AttachEngine(context.Background(), session.AttachParams{})
	if err == nil || !strings.Contains(err.Error(), "nil MediaPort") {
		t.Fatalf("err = %v, want nil MediaPort", err)
	}
}

type stubPort struct {
	callID string
}

func (stubPort) InputPCM() <-chan engine.PCMFrame {
	ch := make(chan engine.PCMFrame)
	close(ch)
	return ch
}
func (stubPort) SendOutputPCM(engine.PCMFrame) error { return nil }
func (stubPort) OnBargeIn(func())                    {}
func (stubPort) Codec() engine.CodecSpec             { return engine.CodecSpec{} }
func (stubPort) SampleRate() int                     { return 16000 }
func (p stubPort) CallID() string                    { return p.callID }
func (stubPort) TenantID() string                    { return "1" }

func TestAttachEngine_InvalidModeUsesResolveModeThenFailsHooks(t *testing.T) {
	// Empty mode → ResolveMode → ModeCascadedNative → pipeline hooks missing.
	_, err := session.AttachEngine(context.Background(), session.AttachParams{
		Port: stubPort{callID: "attach-resolve"},
		Env:  tenantcfg.VoiceEnv{VoiceMode: "pipeline"},
		Mode: "",
	})
	if err == nil {
		t.Fatal("expected error when provider hooks not wired")
	}
	if !strings.Contains(err.Error(), "pipeline credentials") && !strings.Contains(err.Error(), "provider hooks") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestAttachEngine_HybridRequiresCreds(t *testing.T) {
	_, err := session.AttachEngine(context.Background(), session.AttachParams{
		Port: stubPort{callID: "hybrid-mode"},
		Mode: engine.ModeHybrid,
	})
	if err == nil || !strings.Contains(err.Error(), "hybrid requires") {
		t.Fatalf("err = %v, want hybrid requires credentials", err)
	}
}

func TestAttachEngine_InvalidModeFallsBackToResolve(t *testing.T) {
	// Invalid Mode is rewritten via ResolveMode → cascaded-native, which then fails without hooks.
	_, err := session.AttachEngine(context.Background(), session.AttachParams{
		Port: stubPort{callID: "bad-mode"},
		Mode: engine.Mode("nope"),
	})
	if err == nil {
		t.Fatal("expected error after mode fallback")
	}
}

func TestAttachEngine_RealtimeNotReady(t *testing.T) {
	_, err := session.AttachEngine(context.Background(), session.AttachParams{
		Port: stubPort{callID: "rt-not-ready"},
		Mode: engine.ModeRealtime,
		Env:  tenantcfg.VoiceEnv{VoiceMode: "realtime"},
	})
	if err == nil || !strings.Contains(err.Error(), "realtime credentials") {
		t.Fatalf("err = %v, want realtime credentials unusable", err)
	}
}
