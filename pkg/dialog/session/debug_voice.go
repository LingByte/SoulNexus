package session

import (
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
)

// EffectiveVoiceEnv applies debug-session overrides on top of tenant/assistant
// voice config. Browser WebSocket/WebRTC debug uses the cascaded pipeline attach
// path (ASR+LLM+TTS); multimodal realtime is not supported there.
func EffectiveVoiceEnv(sess *Session, env tenantcfg.VoiceEnv) tenantcfg.VoiceEnv {
	if sess == nil || !IsDebugCall(sess.CallID) {
		return env
	}
	switch sess.Transport {
	case TransportWebSocket, TransportWebRTC:
		env.VoiceMode = "pipeline"
	}
	return env
}
