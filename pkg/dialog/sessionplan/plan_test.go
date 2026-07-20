package sessionplan_test

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/dialog/callflags"
	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/sessionplan"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
)

func TestBuild_DefaultCascaded(t *testing.T) {
	plan := sessionplan.Build("c1", tenantcfg.VoiceEnv{})
	if plan.Mode != engine.ModeCascadedNative {
		t.Fatalf("mode = %q", plan.Mode)
	}
	if plan.Welcome != sessionplan.WelcomeAuto {
		t.Fatalf("welcome = %q", plan.Welcome)
	}
	if plan.SkipWelcome() {
		t.Fatal("default should not skip welcome")
	}
}

func TestBuild_ScriptModeSkipsWelcome(t *testing.T) {
	const id = "script-plan-1"
	callflags.MarkScriptMode(id)
	t.Cleanup(func() { callflags.ClearScriptMode(id) })

	plan := sessionplan.Build(id, tenantcfg.VoiceEnv{})
	if plan.Welcome != sessionplan.WelcomeSkip || !plan.SkipWelcome() {
		t.Fatalf("expected skip welcome, got %+v", plan)
	}
	if !plan.Flags.ScriptMode {
		t.Fatal("expected ScriptMode flag")
	}
}

func TestBuild_RealtimeWithoutReadyFallsBack(t *testing.T) {
	env := tenantcfg.VoiceEnv{
		VoiceMode:        "realtime",
		RealtimeProvider: "aliyun_omni",
		// RealtimeConfigRaw empty → RealtimeReady false → cascaded-native.
	}
	plan := sessionplan.Build("rt-1", env)
	if plan.Mode != engine.ModeCascadedNative {
		t.Fatalf("expected cascaded-native when realtime not ready, got %q", plan.Mode)
	}
}
