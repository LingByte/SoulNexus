package voiceprintconfig

import (
	"context"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingllm/voiceprint"
)

func TestResolveEnabledDisabledByDefault(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv(envProvider, "")

	if _, _, ok := ResolveEnabled(); ok {
		t.Fatal("expected disabled when VOICEPRINT_PROVIDER unset")
	}
}

func TestResolveEnabledHTTP(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv(envProvider, "http")
	t.Setenv("VOICEPRINT_API_KEY", "test-key")

	slug, provider, ok := ResolveEnabled()
	if !ok || slug != "http" || provider != voiceprint.ProviderHTTP {
		t.Fatalf("got slug=%q provider=%q ok=%v", slug, provider, ok)
	}
}

func TestResolveEnabledHTTPRequiresAPIKey(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv(envProvider, "http")
	t.Setenv("VOICEPRINT_API_KEY", "")

	if _, _, ok := ResolveEnabled(); ok {
		t.Fatal("expected disabled without VOICEPRINT_API_KEY")
	}
}

func TestDefaultHTTPBaseURLFromAddr(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv("VOICEPRINT_BASE_URL", "")
	t.Setenv("VOICEPRINT_SERVICE_URL", "")

	if got := defaultHTTPBaseURL(); got != "http://127.0.0.1:8005" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveEnabledXunfei(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv(envProvider, "xunfei")
	t.Setenv("XUNFEI_APP_ID", "app123")
	t.Setenv("XUNFEI_API_KEY", "key123")
	t.Setenv("XUNFEI_WS_API_SECRET", "secret123")

	slug, provider, ok := ResolveEnabled()
	if !ok || slug != "xunfei" || provider != voiceprint.ProviderXunfei {
		t.Fatalf("got slug=%q provider=%q ok=%v", slug, provider, ok)
	}
}

func TestResolveEnabledVolcengine(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv(envProvider, "volcengine")
	t.Setenv("VOLCENGINE_ACCESS_KEY", "ak")
	t.Setenv("VOLCENGINE_SECRET_KEY", "sk")

	slug, provider, ok := ResolveEnabled()
	if !ok || slug != "volcengine" || provider != voiceprint.ProviderVolcengine {
		t.Fatalf("got slug=%q provider=%q ok=%v", slug, provider, ok)
	}
}

func TestTenantGroupID(t *testing.T) {
	if got := TenantGroupID(42); got != "lingecho-tenant-42" {
		t.Fatalf("got %q", got)
	}
}

func TestRunSelfTestWithoutProbe(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv(envProvider, "http")
	t.Setenv("VOICEPRINT_API_KEY", "k")

	report := RunSelfTest(nil, false)
	if !report.OK || !report.Enabled || report.Provider != "http" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestRunSelfTestDisabled(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv(envProvider, "")

	report := RunSelfTest(nil, false)
	if report.OK || report.Enabled {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestConfigSummaryVolcengineSupportsIdentify(t *testing.T) {
	out := ConfigSummary(voiceprint.ProviderVolcengine)
	if out["supportsIdentify"] != true {
		t.Fatalf("volcengine should support identify: %+v", out)
	}
}

func TestNewBridgeRequiresEnable(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv(envProvider, "none")

	if _, err := NewBridge(); err == nil {
		t.Fatal("expected error when disabled")
	}
}

func TestNewBridgeHTTP(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv(envProvider, "http")
	t.Setenv("VOICEPRINT_API_KEY", "k")

	bridge, err := NewBridge()
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()
	if !bridge.SupportsIdentify() {
		t.Fatal("http bridge should support identify")
	}
}

func TestEnsureTenantGroupVolcengineSkipsCreateGroup(t *testing.T) {
	t.Cleanup(utils.PurgeEnvCacheForTest)
	t.Setenv(envProvider, "volcengine")
	t.Setenv("VOLCENGINE_ACCESS_KEY", "ak")
	t.Setenv("VOLCENGINE_SECRET_KEY", "sk")

	bridge, err := NewBridge()
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()

	if err := bridge.EnsureTenantGroup(context.Background(), 1, "demo"); err != nil {
		t.Fatalf("volcengine should not require tenant group: %v", err)
	}
}
