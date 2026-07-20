package tenantcfg

import (
	"strconv"
	"strings"
)

// RealtimeInputSampleRate is the caller PCM rate multimodal agents expect.
const RealtimeInputSampleRate = 16000

// RealtimeOutputSampleRate reads outputSampleRate from realtime_config.
// Defaults to 24000 when unset.
func RealtimeOutputSampleRate(env VoiceEnv) int {
	if env.RealtimeConfigRaw != nil {
		for _, k := range []string{"outputSampleRate", "output_sample_rate", "sampleRate", "sample_rate"} {
			if v, ok := env.RealtimeConfigRaw[k]; ok {
				if n := numericPositive(v); n > 0 {
					return n
				}
			}
		}
	}
	return 24000
}

// RealtimeTemperature resolves sampling temperature from tenant config.
func RealtimeTemperature(env VoiceEnv) float64 {
	if env.RealtimeConfigRaw != nil {
		if v, ok := env.RealtimeConfigRaw["temperature"]; ok {
			switch t := v.(type) {
			case float64:
				if t > 0 {
					return t
				}
			case float32:
				if t > 0 {
					return float64(t)
				}
			case int:
				if t > 0 {
					return float64(t)
				}
			case int64:
				if t > 0 {
					return float64(t)
				}
			case string:
				if s := strings.TrimSpace(t); s != "" {
					if f, err := strconv.ParseFloat(s, 64); err == nil && f > 0 {
						return f
					}
				}
			}
		}
	}
	switch strings.ToLower(strings.TrimSpace(env.RealtimeProvider)) {
	case "aliyun_omni", "aliyun-omni", "qwen_omni", "qwen-omni":
		return 0.6
	}
	return 0
}

// RealtimeInstructions returns tenant persona from realtime_config.
func RealtimeInstructions(env VoiceEnv) string {
	if env.RealtimeConfigRaw == nil {
		return ""
	}
	for _, k := range []string{"instructions", "systemPrompt", "system_prompt"} {
		if v, ok := env.RealtimeConfigRaw[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

// EffectiveRealtimeOperatorCore returns the assistant/tenant persona for
// realtime sessions. Assistant Prompt is applied to LLMInstructions by
// ApplyVoiceConfigBundle (same field cascaded PipelineSystemPrompt uses);
// RealtimeInstructions covers legacy realtime_config.instructions only.
func EffectiveRealtimeOperatorCore(env VoiceEnv) string {
	if s := strings.TrimSpace(env.LLMInstructions); s != "" {
		return s
	}
	return RealtimeInstructions(env)
}

// RealtimeVoice extracts the AI voice name. Empty → provider default.
func RealtimeVoice(env VoiceEnv) string {
	if env.RealtimeConfigRaw == nil {
		return ""
	}
	for _, k := range []string{"voice", "voiceId", "voice_id"} {
		if v, ok := env.RealtimeConfigRaw[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func numericPositive(v any) int {
	switch t := v.(type) {
	case int:
		if t > 0 {
			return t
		}
	case int64:
		if t > 0 {
			return int(t)
		}
	case float64:
		if t > 0 {
			return int(t)
		}
	}
	return 0
}
