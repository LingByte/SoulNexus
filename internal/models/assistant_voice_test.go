package models

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/utils/common"
)

func TestBundleFromAssistant_usesTenantVoiceMode(t *testing.T) {
	tenant := Tenant{VoiceMode: "realtime"}
	ast := Assistant{
		BaseModel: common.BaseModel{ID: 1},
		Enabled:   true,
		Prompt:    "hello",
	}
	fallback := tenantcfg.VoiceConfigBundle{VoiceMode: tenant.VoiceMode}
	bundle, ok := bundleFromAssistant(nil, tenant, ast, fallback)
	if !ok {
		t.Fatal("expected ok")
	}
	if bundle.VoiceMode != "realtime" {
		t.Fatalf("VoiceMode = %q, want realtime from tenant", bundle.VoiceMode)
	}
}

func TestBundleFromAssistant_defaultPipelineWhenTenantEmpty(t *testing.T) {
	tenant := Tenant{}
	ast := Assistant{
		BaseModel: common.BaseModel{ID: 1},
		Enabled:   true,
		Prompt:    "hello",
	}
	fallback := tenantcfg.VoiceConfigBundle{VoiceMode: tenant.VoiceMode}
	bundle, ok := bundleFromAssistant(nil, tenant, ast, fallback)
	if !ok {
		t.Fatal("expected ok")
	}
	if bundle.VoiceMode != "pipeline" {
		t.Fatalf("VoiceMode = %q, want pipeline default", bundle.VoiceMode)
	}
}

func TestApplyAssistantSelectedTimbre_overridesCredentialDefaultVoice(t *testing.T) {
	// Simulates user-key overlay replacing TTS JSON (which wipes assistant voice).
	bundle := tenantcfg.VoiceConfigBundle{
		Tts: tenantcfg.MustJSONMap(`{"provider":"aliyun","voice":"longxiaochun"}`),
	}
	out := applyAssistantSelectedTimbre(nil, 0, bundle, "longwan", "")
	got := voiceIDFromLegJSON(out.Tts)
	if got != "longwan" {
		t.Fatalf("tts voice = %q, want longwan (assistant selection after credential overlay)", got)
	}
}
