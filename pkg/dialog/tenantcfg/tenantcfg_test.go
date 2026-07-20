// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package tenantcfg

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// resetLoaderForTest clears the package-level loader and restores it
// at test cleanup so tests can run in any order.
func resetLoaderForTest(t *testing.T) {
	t.Helper()
	prev := Loader()
	SetLoader(nil)
	t.Cleanup(func() { SetLoader(prev) })
}

func mustJSON(t *testing.T, v map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustJSON: %v", err)
	}
	return b
}

// ----- VoiceEnvFromJSON ----------------------------------------------

func TestVoiceEnvFromJSON_HappyPath_Pipeline(t *testing.T) {
	env, err := VoiceEnvFromJSON(
		mustJSON(t, map[string]any{"provider": "qcloud", "appId": "1", "secretId": "s", "secretKey": "k"}),
		mustJSON(t, map[string]any{"provider": "qcloud", "appId": "1", "secretId": "s", "secretKey": "k", "voiceType": float64(101), "speed": float64(0), "sampleRate": float64(16000)}),
		mustJSON(t, map[string]any{"provider": "openai", "apiKey": "k", "baseUrl": "https://x"}),
		nil,
		"pipeline",
	)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if env.VoiceMode != "pipeline" {
		t.Errorf("VoiceMode = %q, want pipeline", env.VoiceMode)
	}
	if env.ASRProvider != "qcloud" || env.ASRAppID != "1" {
		t.Errorf("ASR fields wrong: %+v", env)
	}
	if env.TTSVoiceType != 101 || env.TTSSampleRate != 16000 {
		t.Errorf("TTS typed fields wrong: %+v", env)
	}
	if env.LLMProvider != "openai" || env.LLMAPIKey != "k" {
		t.Errorf("LLM fields wrong: %+v", env)
	}
}

func TestVoiceEnvFromJSON_LLMInstructions(t *testing.T) {
	env, err := VoiceEnvFromJSON(
		nil, nil,
		mustJSON(t, map[string]any{
			"provider":     "openai",
			"apiKey":       "k",
			"instructions": "你是语音助手",
		}),
		nil,
		"pipeline",
	)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if env.LLMInstructions != "你是语音助手" {
		t.Errorf("LLMInstructions = %q", env.LLMInstructions)
	}

	env2, err := VoiceEnvFromJSON(
		nil, nil,
		mustJSON(t, map[string]any{
			"provider":     "openai",
			"apiKey":       "k",
			"systemPrompt": "legacy prompt",
		}),
		nil,
		"pipeline",
	)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if env2.LLMInstructions != "legacy prompt" {
		t.Errorf("LLMInstructions from systemPrompt = %q", env2.LLMInstructions)
	}
}

func TestVoiceEnvFromJSON_HappyPath_Realtime(t *testing.T) {
	env, err := VoiceEnvFromJSON(
		nil, nil, nil,
		mustJSON(t, map[string]any{"provider": "aliyun_omni", "apiKey": "k"}),
		"realtime",
	)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if env.VoiceMode != "realtime" {
		t.Errorf("VoiceMode = %q, want realtime", env.VoiceMode)
	}
	if env.RealtimeProvider != "aliyun_omni" {
		t.Errorf("RealtimeProvider = %q", env.RealtimeProvider)
	}
}

func TestVoiceEnvFromJSON_EmptyMode_InfersRealtime(t *testing.T) {
	env, err := VoiceEnvFromJSON(
		nil, nil, nil,
		mustJSON(t, map[string]any{"provider": "qwen", "apiKey": "k"}),
		"",
	)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if env.VoiceMode != "realtime" {
		t.Errorf("empty mode + realtime ready → got %q, want realtime", env.VoiceMode)
	}
}

func TestVoiceEnvFromJSON_EmptyMode_DefaultsToPipeline(t *testing.T) {
	env, err := VoiceEnvFromJSON(nil, nil, nil, nil, "")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if env.VoiceMode != "pipeline" {
		t.Errorf("empty mode + no realtime → got %q, want pipeline", env.VoiceMode)
	}
}

func TestVoiceEnvFromJSON_MalformedJSON_ReturnsError(t *testing.T) {
	_, err := VoiceEnvFromJSON([]byte("{not json"), nil, nil, nil, "pipeline")
	if err == nil {
		t.Fatal("expected error on malformed ASR JSON")
	}
	if !strings.Contains(err.Error(), "tenantcfg: parse asr") {
		t.Errorf("err = %v, want wrapped 'tenantcfg: parse asr'", err)
	}
}

func TestVoiceEnvFromJSON_AnthropicFields(t *testing.T) {
	env, err := VoiceEnvFromJSON(
		nil, nil,
		mustJSON(t, map[string]any{"provider": "anthropic", "apiKey": "k", "baseUrl": "https://api.anthropic.com"}),
		nil,
		"pipeline",
	)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if env.LLMProvider != "anthropic" || env.LLMAPIKey != "k" {
		t.Errorf("anthropic parse: %+v", env)
	}
}

// ----- Predicates ----------------------------------------------------

func TestVoiceReady_DispatchesByMode(t *testing.T) {
	pipelineEnv := VoiceEnv{
		VoiceMode:   "pipeline",
		ASRProvider: "qcloud",
		ASRAppID:    "1", ASRSecretID: "s", ASRSecretKey: "k",
		TTSProvider: "qcloud",
		TTSAppID:    "1", TTSSecretID: "s", TTSSecretKey: "k",
		LLMProvider: "openai", LLMAPIKey: "k",
	}
	if !VoiceReady(pipelineEnv) {
		t.Error("pipeline env should be ready")
	}

	realtimeEnv := VoiceEnv{
		VoiceMode:         "realtime",
		RealtimeProvider:  "qwen",
		RealtimeConfigRaw: map[string]any{"provider": "qwen", "apiKey": "k"},
	}
	if !VoiceReady(realtimeEnv) {
		t.Error("realtime env should be ready")
	}

	if VoiceReady(VoiceEnv{}) {
		t.Error("empty env should not be ready")
	}
}

func TestPipelineUsable_RequiresAllThree(t *testing.T) {
	base := VoiceEnv{
		ASRProvider: "qcloud", ASRAppID: "1", ASRSecretID: "s", ASRSecretKey: "k",
		TTSProvider: "qcloud", TTSAppID: "1", TTSSecretID: "s", TTSSecretKey: "k",
		LLMProvider: "openai", LLMAPIKey: "k",
	}
	if !PipelineUsable(base) {
		t.Error("full pipeline env should be usable")
	}

	// Missing each field in turn fails.
	missing := base
	missing.ASRAppID = ""
	if PipelineUsable(missing) {
		t.Error("missing ASR appId should fail")
	}
	missing = base
	missing.TTSSecretKey = ""
	if PipelineUsable(missing) {
		t.Error("missing TTS secret should fail")
	}
	missing = base
	missing.LLMAPIKey = ""
	if PipelineUsable(missing) {
		t.Error("missing LLM key should fail")
	}

	// Unknown ASR provider blocks the pipeline.
	weird := base
	weird.ASRProvider = "vendor-from-mars"
	if PipelineUsable(weird) {
		t.Error("unknown ASR provider should fail")
	}
}

func TestPipelineUsable_DefaultsBlankProviderToQCloud(t *testing.T) {
	// Both ASR & TTS providers blank → both default to "qcloud", so
	// the QCloud credential check applies.
	env := VoiceEnv{
		ASRAppID: "1", ASRSecretID: "s", ASRSecretKey: "k",
		TTSAppID: "1", TTSSecretID: "s", TTSSecretKey: "k",
		LLMProvider: "openai", LLMAPIKey: "k",
	}
	if !PipelineUsable(env) {
		t.Error("blank-provider env with full QCloud creds should be usable")
	}
}

func TestRealtimeReady_RequiresProviderAndApiKey(t *testing.T) {
	if RealtimeReady(VoiceEnv{}) {
		t.Error("empty env should not be realtime-ready")
	}
	if RealtimeReady(VoiceEnv{RealtimeConfigRaw: map[string]any{"apiKey": "k"}}) {
		t.Error("realtime without provider should not be ready")
	}
	if RealtimeReady(VoiceEnv{
		RealtimeProvider:  "qwen",
		RealtimeConfigRaw: map[string]any{"provider": "qwen"},
	}) {
		t.Error("realtime without apiKey should not be ready")
	}
	ok := RealtimeReady(VoiceEnv{
		RealtimeProvider:  "qwen",
		RealtimeConfigRaw: map[string]any{"provider": "qwen", "apiKey": "k"},
	})
	if !ok {
		t.Error("realtime with provider + apiKey should be ready")
	}
}

func TestRealtimeReady_VolcDialogue(t *testing.T) {
	if RealtimeReady(VoiceEnv{
		RealtimeProvider:  "volcengine_dialogue",
		RealtimeConfigRaw: map[string]any{"provider": "volcengine_dialogue", "appId": "123"},
	}) {
		t.Error("volc without accessKey should not be ready")
	}
	if !RealtimeReady(VoiceEnv{
		RealtimeProvider: "volcengine_dialogue",
		RealtimeConfigRaw: map[string]any{
			"provider":  "volcengine_dialogue",
			"appId":     "123",
			"accessKey": "tok",
		},
	}) {
		t.Error("volc with appId + accessKey should be ready")
	}
}

func TestRealtimeReady_ReadsProviderFromRaw(t *testing.T) {
	// RealtimeProvider field empty but raw map has provider — should
	// still be considered ready.
	ok := RealtimeReady(VoiceEnv{
		RealtimeConfigRaw: map[string]any{"provider": "qwen", "apiKey": "k"},
	})
	if !ok {
		t.Error("realtime with provider only in raw map should be ready")
	}
}

func TestLLMReady_PerProvider(t *testing.T) {
	cases := []struct {
		name     string
		env      VoiceEnv
		expected bool
	}{
		{"anthropic: key", VoiceEnv{LLMProvider: "anthropic", LLMAPIKey: "k"}, true},
		{"anthropic: empty", VoiceEnv{LLMProvider: "anthropic"}, false},
		{"unsupported: alibaba", VoiceEnv{LLMProvider: "alibaba", LLMAPIKey: "k", LLMAppID: "a"}, false},
		{"unsupported: coze", VoiceEnv{LLMProvider: "coze", LLMAPIKey: "k", LLMBaseURL: "https://x"}, false},
		{"ollama: baseurl", VoiceEnv{LLMProvider: "ollama", LLMBaseURL: "http://localhost:11434"}, true},
		{"ollama: empty", VoiceEnv{LLMProvider: "ollama"}, false},
		{"openai default: key", VoiceEnv{LLMProvider: "openai", LLMAPIKey: "k"}, true},
		{"openai default: empty", VoiceEnv{LLMProvider: "openai"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := llmReady(tc.env); got != tc.expected {
				t.Errorf("llmReady = %v, want %v", got, tc.expected)
			}
		})
	}
}

// ----- Loader / Resolve ----------------------------------------------

func TestResolve_NoLoaderInstalled(t *testing.T) {
	resetLoaderForTest(t)
	env, ok, err := Resolve(context.Background(), 42, "")
	if ok || err != nil || env.VoiceMode != "" {
		t.Errorf("no loader → got (%+v, %v, %v); want (zero, false, nil)", env, ok, err)
	}
}

func TestResolve_TenantIDZero(t *testing.T) {
	resetLoaderForTest(t)
	SetLoader(func(context.Context, uint, string) (VoiceConfigBundle, bool) {
		t.Fatal("loader should not be called for tenantID=0")
		return VoiceConfigBundle{}, false
	})
	_, ok, err := Resolve(context.Background(), 0, "")
	if ok || err != nil {
		t.Errorf("tenantID=0 → got (ok=%v, err=%v); want (false, nil)", ok, err)
	}
}

func TestResolve_LoaderReturnsNotOK(t *testing.T) {
	resetLoaderForTest(t)
	SetLoader(func(context.Context, uint, string) (VoiceConfigBundle, bool) {
		return VoiceConfigBundle{}, false
	})
	_, ok, err := Resolve(context.Background(), 99, "")
	if ok || err != nil {
		t.Errorf("loader-not-ok → got (ok=%v, err=%v)", ok, err)
	}
}

func TestResolve_LoaderHappy(t *testing.T) {
	resetLoaderForTest(t)
	SetLoader(func(_ context.Context, tid uint, _ string) (VoiceConfigBundle, bool) {
		if tid != 7 {
			t.Errorf("loader received tid=%d, want 7", tid)
		}
		return VoiceConfigBundle{
			Llm:       mustJSON(t, map[string]any{"provider": "openai", "apiKey": "k"}),
			VoiceMode: "pipeline",
		}, true
	})
	env, ok, err := Resolve(context.Background(), 7, "")
	if !ok || err != nil {
		t.Fatalf("Resolve failed: ok=%v err=%v", ok, err)
	}
	if env.LLMAPIKey != "k" {
		t.Errorf("env not populated: %+v", env)
	}
}

func TestResolve_LoaderHappyWithJSONError(t *testing.T) {
	resetLoaderForTest(t)
	SetLoader(func(context.Context, uint, string) (VoiceConfigBundle, bool) {
		return VoiceConfigBundle{Asr: []byte("{not json"), VoiceMode: "pipeline"}, true
	})
	_, ok, err := Resolve(context.Background(), 7, "")
	if ok {
		t.Error("malformed JSON should not be ok")
	}
	if err == nil {
		t.Fatal("expected JSON parse error")
	}
	// Must wrap a json.SyntaxError-class error somewhere down the chain.
	var syn *json.SyntaxError
	if !errors.As(err, &syn) {
		t.Errorf("err = %v, want wraps *json.SyntaxError", err)
	}
}

func TestSetLoader_NilClears(t *testing.T) {
	resetLoaderForTest(t)
	SetLoader(func(context.Context, uint, string) (VoiceConfigBundle, bool) {
		return VoiceConfigBundle{}, true
	})
	SetLoader(nil)
	if Loader() != nil {
		t.Error("SetLoader(nil) should clear")
	}
}

// TestTTSCredentialsReady_PerProvider exercises every credential-shape
// branch in the switch so future provider additions don't silently
// regress the cardinality gate. We don't validate provider semantics;
// the factory does that at call time.
func TestTTSCredentialsReady_PerProvider(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		raw      map[string]any
		legacy   func() VoiceEnv // for typed-field providers
		expected bool
	}{
		{"qcloud full", "qcloud", nil, func() VoiceEnv {
			return VoiceEnv{TTSAppID: "1", TTSSecretID: "s", TTSSecretKey: "k"}
		}, true},
		{"qcloud missing", "qcloud", nil, func() VoiceEnv { return VoiceEnv{TTSAppID: "1"} }, false},
		{"openai apiKey", "openai", map[string]any{"apiKey": "k"}, nil, true},
		{"openai snake api_key", "openai", map[string]any{"api_key": "k"}, nil, true},
		{"openai missing", "openai", map[string]any{}, nil, false},
		{"openai nil raw", "openai", nil, nil, false},
		{"azure full", "azure", map[string]any{"subscriptionKey": "k", "region": "eastus"}, nil, true},
		{"azure missing region", "azure", map[string]any{"apiKey": "k"}, nil, false},
		{"azure nil raw", "azure", nil, nil, false},
		{"baidu token", "baidu", map[string]any{"token": "t"}, nil, true},
		{"baidu key+secret", "baidu", map[string]any{"apiKey": "k", "secretKey": "s"}, nil, true},
		{"baidu incomplete", "baidu", map[string]any{"apiKey": "k"}, nil, false},
		{"baidu nil raw", "baidu", nil, nil, false},
		{"xunfei full", "xunfei", map[string]any{"appId": "a", "apiKey": "k", "apiSecret": "s"}, nil, true},
		{"xunfei snake_case fallback", "xunfei", map[string]any{"app_id": "a", "api_key": "k", "api_secret": "s"}, nil, true},
		{"xunfei missing field", "xunfei", map[string]any{"appId": "a", "apiKey": "k"}, nil, false},
		{"xunfei nil raw", "xunfei", nil, nil, false},
		{"aws full", "aws", map[string]any{"accessKeyId": "a", "secretAccessKey": "s"}, nil, true},
		{"aws missing secret", "aws", map[string]any{"accessKeyId": "a"}, nil, false},
		{"aws nil raw", "aws", nil, nil, false},
		{"volcengine accessToken", "volcengine", map[string]any{
			"appId": "1", "accessToken": "tok",
		}, nil, true},
		{"volcengine missing token", "volcengine", map[string]any{"appId": "1"}, nil, false},
		{"volcengine_clone full", "volcengine_clone", map[string]any{
			"appId": "1", "accessToken": "tok", "assetId": "S_abc",
		}, nil, true},
		{"volcengine_clone missing assetId tenant ready", "volcengine_clone", map[string]any{
			"appId": "1", "accessToken": "tok",
		}, nil, true},
		{"unknown with raw passes", "vendor-from-mars", map[string]any{"anything": "x"}, nil, true},
		{"unknown nil raw fails", "vendor-from-mars", nil, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var env VoiceEnv
			if tc.legacy != nil {
				env = tc.legacy()
			}
			if tc.raw != nil {
				env.TTSConfigRaw = tc.raw
			}
			if got := ttsCredentialsReady(tc.provider, env); got != tc.expected {
				t.Errorf("ttsCredentialsReady(%q) = %v, want %v", tc.provider, got, tc.expected)
			}
		})
	}
}

func TestSnakeCaseAlt(t *testing.T) {
	// Note: snakeCaseAlt prefixes every uppercase letter after position 0
	// with '_' regardless of context — it's a one-pass camelCase→snake_case
	// helper, not a general capitalisation handler. We pin its actual
	// behaviour here so future refactors can't silently break the
	// xunfei provider's snake_case key fallback.
	cases := map[string]string{
		"appId":     "app_id",
		"apiKey":    "api_key",
		"apiSecret": "api_secret",
		"":          "",
		"abc":       "abc",
	}
	for in, want := range cases {
		if got := snakeCaseAlt(in); got != want {
			t.Errorf("snakeCaseAlt(%q) = %q, want %q", in, got, want)
		}
	}
}

