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
