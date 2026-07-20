// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package tenantcfg

import (
	"encoding/json"
	"fmt"
	"strings"
)

// VoiceEnv holds tenant voice pipeline settings parsed from the
// tenant's per-row JSON columns (asr, tts, llm, realtime + voice_mode).
//
// Field layout is the same as the legacy conversation.VoiceEnv
// (which now aliases this type) — see that comment block for per-field
// semantics. We keep typed fields for QCloud-specific TTS tweaks and
// historical legacy callers; new fields should go on TTSConfigRaw /
// RealtimeConfigRaw and be read provider-side.
type VoiceEnv struct {
	LLMProvider string
	LLMBaseURL  string
	LLMAppID    string
	LLMAPIKey        string
	LLMModel         string
	LLMInstructions  string // pipeline / voicedialog system prompt (llm_config.instructions)

	ASRProvider  string
	ASRAppID     string
	ASRSecretID  string
	ASRSecretKey string
	ASRModelType string

	TTSProvider   string
	TTSAppID      string
	TTSSecretID   string
	TTSSecretKey  string
	TTSVoiceType  int64
	TTSSpeed      int64
	TTSSampleRate int
	// TTSConfigRaw is the full tenant TTS JSON (provider + per-vendor
	// fields), passed through to synthesizer.NewStreamingFromCredential.
	TTSConfigRaw map[string]any

	// VoiceMode selects which voice attach path runs:
	//   "pipeline" — legacy 3-layer ASR → LLM → TTS
	//   "realtime" — single full-duplex multimodal model
	// Empty defaults to "pipeline" (post-parsing).
	VoiceMode string
	// RealtimeProvider is a denormalised copy of
	// RealtimeConfigRaw["provider"] for cheap readiness checks.
	RealtimeProvider string
	// RealtimeConfigRaw is the full tenant realtime JSON.
	RealtimeConfigRaw map[string]any
	// ASRConfigRaw is the full tenant/assistant ASR JSON for factory routing.
	ASRConfigRaw map[string]any

	// Assistant overlay (from assistants / assistant_versions).
	TenantID               uint
	AssistantWelcome       string
	KnowledgeNamespace     string
	KnowledgeNamespaceID   uint
	KnowledgeCollection    string
	AssistantID            uint
	AssistantVersionID     uint
	// NLU binding (assistant.nlu_model_id → ready tenant profile paths).
	NluModelID         uint
	NluModelPath       string
	NluTokenizerPath   string
	NluIntentsPath     string
	NluPrototypesPath  string
	NluMinConfidence   float64
	VadConfigRaw           map[string]any
	AgentConfigRaw         map[string]any
	HotWordsRaw            []byte
	InterruptionConfigRaw  map[string]any
	AudioTrackConfigRaw    map[string]any
	AudioProcessConfigRaw  map[string]any
	QueryRewriterRaw       map[string]any
	McpServersRaw          []byte
	VoiceDialogWsURL       string

	// LLMRatePer1kTokens is tenant LLM cost (CNY per 1k tokens) from llm_config JSON.
	LLMRatePer1kTokens float64
}

// VoiceEnvFromJSON builds a VoiceEnv from tenant JSON blobs.
//
// voiceMode is the value of the tenant's voice_mode column. Empty
// is treated specially:
//
//  1. Explicit "realtime" / "pipeline" → trust it.
//  2. Empty AND realtime tab has credentials → infer "realtime"
//     (migration-resilient: column may not exist on legacy rows).
//  3. Empty AND realtime tab empty → "pipeline" (legacy default).
//
// Returns an error only when one of the JSON inputs is structurally
// malformed (parse failure). Missing fields / empty objects are
// handled silently — credential readiness is the predicates' job.
func VoiceEnvFromJSON(asrRaw, ttsRaw, llmRaw, realtimeRaw []byte, voiceMode string) (VoiceEnv, error) {
	var out VoiceEnv
	am, err := parseObj(asrRaw)
	if err != nil {
		return out, fmt.Errorf("tenantcfg: parse asr: %w", err)
	}
	tm, err := parseObj(ttsRaw)
	if err != nil {
		return out, fmt.Errorf("tenantcfg: parse tts: %w", err)
	}
	lm, err := parseObj(llmRaw)
	if err != nil {
		return out, fmt.Errorf("tenantcfg: parse llm: %w", err)
	}
	rm, err := parseObj(realtimeRaw)
	if err != nil {
		return out, fmt.Errorf("tenantcfg: parse realtime: %w", err)
	}

	out.VoiceMode = strings.ToLower(strings.TrimSpace(voiceMode))
	if len(rm) > 0 {
		raw := make(map[string]any, len(rm))
		for k, v := range rm {
			raw[k] = v
		}
		out.RealtimeProvider = strings.ToLower(strings.TrimSpace(strFromMap(rm, "provider")))
		if _, ok := raw["provider"]; !ok && out.RealtimeProvider != "" {
			raw["provider"] = out.RealtimeProvider
		}
		out.RealtimeConfigRaw = raw
	}
	if out.VoiceMode == "" {
		if RealtimeReady(out) {
			out.VoiceMode = "realtime"
		} else {
			out.VoiceMode = "pipeline"
		}
	}

	out.ASRProvider = strings.ToLower(strings.TrimSpace(strFromMap(am, "provider")))
	out.ASRAppID = strFromMap(am, "appId", "app_id")
	out.ASRSecretID = strFromMap(am, "secretId", "secret_id")
	out.ASRSecretKey = strFromMap(am, "secretKey", "secret_key", "secret")
	out.ASRModelType = strFromMap(am, "modelType", "model_type")
	if len(am) > 0 {
		raw := make(map[string]any, len(am))
		for k, v := range am {
			raw[k] = v
		}
		if _, ok := raw["provider"]; !ok && out.ASRProvider != "" {
			raw["provider"] = out.ASRProvider
		}
		out.ASRConfigRaw = raw
	}

	out.TTSProvider = strings.ToLower(strings.TrimSpace(strFromMap(tm, "provider")))
	out.TTSAppID = strFromMap(tm, "appId", "app_id")
	out.TTSSecretID = strFromMap(tm, "secretId", "secret_id")
	out.TTSSecretKey = strFromMap(tm, "secretKey", "secret_key", "secret")
	if v := int64FromMap(tm, "voiceType", "voice_type"); v != 0 {
		out.TTSVoiceType = v
	}
	if v := int64FromMap(tm, "speed"); v != 0 || strFromMap(tm, "speed") == "0" {
		out.TTSSpeed = int64FromMap(tm, "speed")
	}
	out.TTSSampleRate = intFromMap(tm, "sampleRate", "sample_rate")
	if len(tm) > 0 {
		raw := make(map[string]any, len(tm))
		for k, v := range tm {
			raw[k] = v
		}
		if _, ok := raw["provider"]; !ok && out.TTSProvider != "" {
			raw["provider"] = out.TTSProvider
		}
		out.TTSConfigRaw = raw
	}

	out.LLMProvider = strings.ToLower(strings.TrimSpace(strFromMap(lm, "provider")))
	out.LLMAPIKey = strFromMap(lm, "apiKey", "api_key")
	out.LLMBaseURL = strFromMap(lm, "baseUrl", "base_url")
	out.LLMAppID = strFromMap(lm, "appId", "app_id")
	out.LLMModel = strFromMap(lm, "model")
	out.LLMInstructions = strFromMap(lm, "instructions", "systemPrompt", "system_prompt")
	out.LLMRatePer1kTokens = floatFromMap(lm, "ratePer1kTokens", "rate_per_1k_tokens", "llmRatePer1kTokens")


	return out, nil
}



// ----- internal JSON helpers ----------------------------------------

func strFromMap(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case string:
				s := strings.TrimSpace(t)
				if s != "" {
					return s
				}
			case float64:
				return strings.TrimSpace(fmt.Sprint(int64(t)))
			}
		}
	}
	return ""
}

func intFromMap(m map[string]any, keys ...string) int {
	s := strFromMap(m, keys...)
	if s == "" {
		return 0
	}
	var n int
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

func int64FromMap(m map[string]any, keys ...string) int64 {
	s := strFromMap(m, keys...)
	if s == "" {
		if v, ok := m[keys[0]]; ok {
			if f, ok := v.(float64); ok {
				return int64(f)
			}
		}
		return 0
	}
	var n int64
	_, _ = fmt.Sscanf(s, "%d", &n)
	return n
}

func floatFromMap(m map[string]any, keys ...string) float64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case float64:
				return t
			case int:
				return float64(t)
			case int64:
				return float64(t)
			case string:
				s := strings.TrimSpace(t)
				if s == "" {
					continue
				}
				var f float64
				if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

func parseObj(raw []byte) (map[string]any, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	if m == nil {
		return map[string]any{}, nil
	}
	return m, nil
}

func snakeCaseAlt(camel string) string {
	var b strings.Builder
	for i, r := range camel {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
			b.WriteRune(r + 32)
			continue
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + 32)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
