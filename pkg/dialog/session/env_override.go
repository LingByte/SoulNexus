package session

import "github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"

func cloneVoiceEnvWithRealtimeTemperature(env tenantcfg.VoiceEnv, temp float64) tenantcfg.VoiceEnv {
	if temp <= 0 {
		return env
	}
	raw := env.RealtimeConfigRaw
	if raw == nil {
		raw = map[string]any{}
	} else {
		cloned := make(map[string]any, len(raw)+1)
		for k, v := range raw {
			cloned[k] = v
		}
		raw = cloned
	}
	raw["temperature"] = temp
	env.RealtimeConfigRaw = raw
	return env
}
