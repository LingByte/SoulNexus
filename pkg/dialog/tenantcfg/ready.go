// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package tenantcfg

import "strings"

// VoiceReady reports whether env has the minimum fields for the
// configured voice attach path. Dispatches by VoiceMode:
//
//   - "realtime": only RealtimeConfigRaw is consulted (multimodal
//     collapses ASR/TTS/LLM into one provider).
//   - "pipeline" (default): legacy ASR + TTS + LLM readiness check.
func VoiceReady(env VoiceEnv) bool {
	if strings.EqualFold(env.VoiceMode, "realtime") {
		return RealtimeReady(env)
	}
	return PipelineUsable(env)
}

// PipelineUsable is the cascaded-mode credential gate. Returns true
// when ASR / TTS / LLM all have at least the minimum field set for
// their resolved provider. Mirrors the legacy
// conversation.pipelineCredsUsable.
//
// We intentionally avoid calling provider factories here (they'd
// open network handles); cheap field presence is enough as a gate —
// the factory will surface a typed error later if config is malformed.
func PipelineUsable(env VoiceEnv) bool {
	asp := strings.ToLower(strings.TrimSpace(env.ASRProvider))
	if asp == "" {
		asp = "qcloud"
	}
	if !asrCredentialsReady(asp, env) {
		return false
	}
	tsp := strings.ToLower(strings.TrimSpace(env.TTSProvider))
	if tsp == "" {
		tsp = "qcloud"
	}
	if !ttsCredentialsReady(tsp, env) {
		return false
	}
	return llmReady(env)
}

// asrCredentialsReady checks minimum ASR credential fields per provider.
func asrCredentialsReady(provider string, env VoiceEnv) bool {
	raw := env.ASRConfigRaw
	switch provider {
	case "qcloud", "tencent":
		if env.ASRAppID != "" && env.ASRSecretID != "" && env.ASRSecretKey != "" {
			return true
		}
		if raw == nil {
			return false
		}
		appID := strFromMap(raw, "appId", "app_id")
		secretID := strFromMap(raw, "secretId", "secret_id")
		secretKey := strFromMap(raw, "secretKey", "secret_key")
		return appID != "" && secretID != "" && secretKey != ""
	case "volcengine", "volcllmasr":
		if raw == nil {
			return false
		}
		appID := strFromMap(raw, "appId", "app_id")
		token := strFromMap(raw, "token", "accessToken", "access_token", "apiKey", "api_key")
		return appID != "" && token != ""
	case "aliyun", "google", "qiniu", "funasr", "funasr_realtime", "gladia", "whisper", "openai", "deepgram", "voiceapi":
		if raw == nil {
			return false
		}
		for _, k := range []string{"apiKey", "api_key"} {
			if v, ok := raw[k]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return true
				}
			}
		}
		return false
	case "xfyun_mul":
		if raw == nil {
			return false
		}
		return strFromMap(raw, "appId", "app_id") != "" &&
			strFromMap(raw, "apiKey", "api_key") != "" &&
			strFromMap(raw, "apiSecret", "api_secret") != ""
	case "aws":
		if raw == nil {
			return false
		}
		hasKey := strFromMap(raw, "accessKeyId", "access_key_id", "apiKey", "api_key") != ""
		hasRegion := strFromMap(raw, "region") != ""
		return hasKey && hasRegion
	case "baidu":
		if raw == nil {
			return false
		}
		return strFromMap(raw, "apiKey", "api_key") != "" && strFromMap(raw, "secretKey", "secret_key") != ""
	case "local":
		if raw == nil {
			return false
		}
		return strFromMap(raw, "endpoint") != ""
	default:
		return raw != nil
	}
}

// RealtimeReady is the realtime-mode credential gate. Requires a
// provider string + at least one apiKey field on RealtimeConfigRaw.
// Provider-specific validation lives in pkg/realtime; this rules out
// the "tenant left the form blank" case so the call leg can fall
// back to config_error.wav rather than ringing into a broken session.
func RealtimeReady(env VoiceEnv) bool {
	raw := env.RealtimeConfigRaw
	if len(raw) == 0 {
		return false
	}
	provider := strings.ToLower(strings.TrimSpace(env.RealtimeProvider))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(strFromMap(raw, "provider")))
	}
	if provider == "" {
		return false
	}
	switch provider {
	case "volcengine_dialogue", "volc_realtime", "doubao_realtime", "volcengine_realtime":
		appID := strFromMap(raw, "appId", "app_id")
		accessKey := strFromMap(raw, "accessKey", "access_key", "access_token", "token")
		return appID != "" && accessKey != ""
	default:
		for _, k := range []string{"apiKey", "api_key"} {
			if v, ok := raw[k]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return true
				}
			}
		}
		return false
	}
}

// LLMReady reports whether env has minimum LLM credentials for
// openai, ollama, or anthropic.
func LLMReady(e VoiceEnv) bool {
	return llmReady(e)
}

// llmReady (private) is the LLM credential check for cascaded mode.
// Supported providers: openai, ollama, anthropic.
func llmReady(e VoiceEnv) bool {
	p := strings.ToLower(strings.TrimSpace(e.LLMProvider))
	if p == "" {
		p = "openai"
	}
	switch p {
	case "ollama":
		return strings.TrimSpace(e.LLMBaseURL) != ""
	case "openai", "anthropic":
		return e.LLMAPIKey != ""
	default:
		return false
	}
}

// ttsCredentialsReady (private) is a low-cost feasibility check for
// the configured TTS provider. The exhaustive validation lives in
// synthesizer.NewSynthesisServiceFromCredential; this rules out
// obviously-empty configs.
func ttsCredentialsReady(provider string, env VoiceEnv) bool {
	switch provider {
	case "qcloud", "tencent":
		return env.TTSAppID != "" && env.TTSSecretID != "" && env.TTSSecretKey != ""
	case "volcengine", "volcengine_stream", "volcengine_llm":
		return volcengineTTSCredentialsReady(env.TTSConfigRaw, false)
	case "volcengine_clone":
		return volcengineTTSCredentialsReady(env.TTSConfigRaw, true)
	case "aliyun", "dashscope", "qwen",
		"qiniu", "openai", "elevenlabs", "fishspeech", "fishaudio",
		"google", "minimax":
		raw := env.TTSConfigRaw
		if raw == nil {
			return false
		}
		for _, k := range []string{"apiKey", "api_key"} {
			if v, ok := raw[k]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					return true
				}
			}
		}
		return false
	case "azure":
		raw := env.TTSConfigRaw
		if raw == nil {
			return false
		}
		hasKey := false
		hasRegion := false
		for _, k := range []string{"subscriptionKey", "subscription_key", "apiKey", "api_key"} {
			if v, ok := raw[k]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					hasKey = true
				}
			}
		}
		if v, ok := raw["region"]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				hasRegion = true
			}
		}
		return hasKey && hasRegion
	case "baidu":
		raw := env.TTSConfigRaw
		if raw == nil {
			return false
		}
		if v, ok := raw["token"]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return true
			}
		}
		hasKey := false
		hasSecret := false
		for _, k := range []string{"apiKey", "api_key"} {
			if v, ok := raw[k]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					hasKey = true
				}
			}
		}
		for _, k := range []string{"secretKey", "secret_key"} {
			if v, ok := raw[k]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					hasSecret = true
				}
			}
		}
		return hasKey && hasSecret
	case "xunfei":
		raw := env.TTSConfigRaw
		if raw == nil {
			return false
		}
		needed := []string{"appId", "apiKey", "apiSecret"}
		for _, k := range needed {
			v, ok := raw[k]
			if !ok {
				v, ok = raw[snakeCaseAlt(k)]
				if !ok {
					return false
				}
			}
			s, ok := v.(string)
			if !ok || strings.TrimSpace(s) == "" {
				return false
			}
		}
		return true
	case "aws":
		raw := env.TTSConfigRaw
		if raw == nil {
			return false
		}
		hasID := false
		hasSecret := false
		for _, k := range []string{"accessKeyId", "access_key_id"} {
			if v, ok := raw[k]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					hasID = true
				}
			}
		}
		for _, k := range []string{"secretAccessKey", "secret_access_key"} {
			if v, ok := raw[k]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					hasSecret = true
				}
			}
		}
		return hasID && hasSecret
	default:
		// Unknown providers: defer to the credential factory at call
		// time. Returning true on a non-nil raw lets the factory
		// produce a precise error and the call leg fall back to
		// config_error.wav playback.
		return env.TTSConfigRaw != nil
	}
}

// volcengineTTSCredentialsReady matches lingllm/synthesizer FromCredential:
// appId + accessToken (or legacy apiKey/token); clone also needs assetId.
func volcengineTTSCredentialsReady(raw map[string]any, clone bool) bool {
	if raw == nil {
		return false
	}
	appID := strFromMap(raw, "appId", "app_id")
	if appID == "" {
		return false
	}
	token := strFromMap(raw, "accessToken", "access_token", "token", "apiKey", "api_key")
	if token == "" {
		return false
	}
	if !clone {
		return true
	}
	// Clone timbre (assetId) is selected on the assistant, not tenant credentials.
	return true
}
