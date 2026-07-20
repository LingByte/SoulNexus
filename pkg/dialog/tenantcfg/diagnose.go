package tenantcfg

import (
	"fmt"
	"strings"
)

// VoiceReadinessReason returns a human-readable reason when VoiceReady is false.
func VoiceReadinessReason(env VoiceEnv) string {
	if VoiceReady(env) {
		return ""
	}
	mode := strings.ToLower(strings.TrimSpace(env.VoiceMode))
	if mode == "" {
		if RealtimeReady(env) {
			mode = "realtime"
		} else {
			mode = "pipeline"
		}
	}
	if strings.EqualFold(mode, "realtime") {
		return realtimeReadinessReason(env)
	}
	return pipelineReadinessReason(env)
}

// LLMReadinessReason returns a human-readable reason when LLMReady is false.
func LLMReadinessReason(env VoiceEnv) string {
	if LLMReady(env) {
		return ""
	}
	p := strings.ToLower(strings.TrimSpace(env.LLMProvider))
	if p == "" {
		p = "openai"
	}
	switch p {
	case "ollama":
		if strings.TrimSpace(env.LLMBaseURL) == "" {
			return fmt.Sprintf("LLM provider=%q missing baseUrl (text chat / debug)", p)
		}
	case "openai", "anthropic":
		if strings.TrimSpace(env.LLMAPIKey) == "" {
			return fmt.Sprintf("LLM provider=%q missing apiKey (text chat / debug)", p)
		}
	default:
		return fmt.Sprintf("LLM provider=%q not supported for text chat", p)
	}
	return "LLM config incomplete"
}

func realtimeReadinessReason(env VoiceEnv) string {
	raw := env.RealtimeConfigRaw
	if len(raw) == 0 {
		return "voiceMode=realtime but realtime_config is empty — publish assistant with Realtime 配置"
	}
	provider := strings.ToLower(strings.TrimSpace(env.RealtimeProvider))
	if provider == "" {
		provider = strings.ToLower(strings.TrimSpace(strFromMap(raw, "provider")))
	}
	if provider == "" {
		return "realtime provider not set in assistant Realtime 配置"
	}
	switch provider {
	case "volcengine_dialogue", "volc_realtime", "doubao_realtime", "volcengine_realtime":
		appID := strFromMap(raw, "appId", "app_id")
		accessKey := strFromMap(raw, "accessKey", "access_key", "access_token", "token")
		var missing []string
		if appID == "" {
			missing = append(missing, "appId")
		}
		if accessKey == "" {
			missing = append(missing, "accessKey")
		}
		if len(missing) > 0 {
			return fmt.Sprintf("realtime provider=%q missing %s", provider, strings.Join(missing, ", "))
		}
	default:
		hasKey := false
		for _, k := range []string{"apiKey", "api_key"} {
			if v, ok := raw[k]; ok {
				if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
					hasKey = true
				}
			}
		}
		if !hasKey {
			return fmt.Sprintf("realtime provider=%q missing apiKey — check assistant Realtime 配置", provider)
		}
	}
	return "realtime voice config incomplete"
}

func pipelineReadinessReason(env VoiceEnv) string {
	var missing []string
	asp := strings.ToLower(strings.TrimSpace(env.ASRProvider))
	if asp == "" {
		asp = "qcloud"
	}
	if asp != "qcloud" {
		missing = append(missing, fmt.Sprintf("ASR provider %q (only qcloud supported)", asp))
	} else {
		if env.ASRAppID == "" {
			missing = append(missing, "ASR appId")
		}
		if env.ASRSecretID == "" {
			missing = append(missing, "ASR secretId")
		}
		if env.ASRSecretKey == "" {
			missing = append(missing, "ASR secretKey")
		}
	}
	tsp := strings.ToLower(strings.TrimSpace(env.TTSProvider))
	if tsp == "" {
		tsp = "qcloud"
	}
	if !ttsCredentialsReady(tsp, env) {
		missing = append(missing, fmt.Sprintf("TTS provider %q credentials", tsp))
	}
	if !llmReady(env) {
		missing = append(missing, LLMReadinessReason(env))
	}
	if len(missing) == 0 {
		return "pipeline voice config incomplete"
	}
	return "pipeline voice config incomplete: " + strings.Join(missing, "; ")
}
