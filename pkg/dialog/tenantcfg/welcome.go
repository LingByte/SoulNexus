package tenantcfg

import "strings"

// ResolvedAssistantWelcome returns the greeting text for tenant TTS or
// proactive omni hints. Top-level assistant welcome wins; otherwise
// realtime_config / llm_config JSON welcome fields are used (same as UI).
func ResolvedAssistantWelcome(env VoiceEnv) string {
	if w := strings.TrimSpace(env.AssistantWelcome); w != "" {
		return w
	}
	for _, m := range []map[string]any{env.RealtimeConfigRaw, env.AgentConfigRaw} {
		if w := welcomeFromMap(m); w != "" {
			return w
		}
	}
	return ""
}

func welcomeFromMap(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	for _, k := range []string{"welcome", "greeting"} {
		if s := strFromMap(m, k); s != "" {
			return s
		}
	}
	return ""
}
