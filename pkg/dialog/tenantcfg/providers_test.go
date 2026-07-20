package tenantcfg

import "testing"

func TestVoiceProvidersFromTenantAliyun(t *testing.T) {
	out := VoiceProvidersFromTenant(
		"pipeline",
		nil,
		MustJSONMap(`{"provider":"aliyun","apiKey":"sk-x"}`),
		nil,
	)
	if out.TtsProvider != "aliyun" {
		t.Fatalf("tts=%q", out.TtsProvider)
	}
	if out.VoiceMode != "pipeline" {
		t.Fatalf("mode=%q", out.VoiceMode)
	}
}

func TestProviderFromJSON(t *testing.T) {
	if got := ProviderFromJSON(MustJSONMap(`{"provider":"aliyun_omni"}`)); got != "aliyun_omni" {
		t.Fatalf("got=%q", got)
	}
}
