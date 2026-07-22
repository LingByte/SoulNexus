package tenantcfg

import (
	"encoding/json"
	"strings"
)

// ProviderFromJSON reads the top-level provider slug from a tenant AI config blob.
func ProviderFromJSON(raw []byte) string {
	m := parseJSONMapBytes(raw)
	if len(m) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(strFromMap(m, "provider")))
}

// TenantVoiceProviders holds non-secret voice routing slugs for UI.
type TenantVoiceProviders struct {
	VoiceMode         string   `json:"voiceMode"`
	TtsProvider       string   `json:"ttsProvider"`
	RealtimeProvider  string   `json:"realtimeProvider"`
	TtsVoiceIDs       []string `json:"ttsVoiceIds,omitempty"`
	RealtimeVoiceIDs  []string `json:"realtimeVoiceIds,omitempty"`
}

// VoiceProvidersFromTenant builds UI-facing provider slugs from tenant columns.
func VoiceProvidersFromTenant(voiceMode string, asrRaw, ttsRaw, rtRaw []byte) TenantVoiceProviders {
	vm := strings.TrimSpace(voiceMode)
	if vm == "" {
		vm = "pipeline"
	}
	if strings.EqualFold(vm, "realtime") {
		vm = "realtime"
	} else {
		vm = "pipeline"
	}
	out := TenantVoiceProviders{
		VoiceMode:        vm,
		TtsProvider:      ProviderFromJSON(ttsRaw),
		RealtimeProvider: ProviderFromJSON(rtRaw),
	}
	if out.TtsProvider == "" {
		out.TtsProvider = "qcloud"
	}
	if out.RealtimeProvider == "" {
		out.RealtimeProvider = "aliyun_omni"
	}
	_ = asrRaw
	return out
}

// MustJSONMap is a test helper.
func MustJSONMap(s string) []byte {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	b, _ := json.Marshal(m)
	return b
}
