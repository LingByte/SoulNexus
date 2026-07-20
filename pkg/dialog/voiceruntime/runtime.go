package voiceruntime

import (
	"context"
	"encoding/json"
	"strings"

	dialogaudio "github.com/LingByte/SoulNexus/pkg/dialog/audio"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"go.uber.org/zap"
)

// InterruptionPolicy controls barge-in behavior for voice sessions.
type InterruptionPolicy struct {
	VADBargeInEnabled             bool
	UnInterruptableAfterPlayStart int // ms grace after TTS start
}

// ProcessConfig describes uplink audio processing (VAD / denoise).
type ProcessConfig struct {
	VadType string
}

// Runtime applies optional uplink preprocessing for native voice attach.
type Runtime struct {
	AudioProc ProcessConfig
	EffectMix *dialogaudio.EffectMixer
}

// Prepare builds a runtime for the given tenant voice env (voice sessions).
func Prepare(ctx context.Context, env tenantcfg.VoiceEnv, sampleRate int, lg *zap.Logger) *Runtime {
	rt := &Runtime{AudioProc: ProcessConfigFromEnv(env)}
	if mix, err := dialogaudio.LoadEffectMixerFromConfig(ctx, env.AudioTrackConfigRaw, sampleRate, lg); err == nil {
		rt.EffectMix = mix
	}
	return rt
}

// ProcessUplinkPCM may transform uplink PCM; default is pass-through.
func (r *Runtime) ProcessUplinkPCM(pcm []byte) []byte {
	return pcm
}

// UserSpeechLikely reports whether uplink PCM likely contains user speech.
func (r *Runtime) UserSpeechLikely(_ []byte) bool {
	return true
}

// ParseInterruption reads interruption JSON from tenant config.
func ParseInterruption(raw string, defaultVADBargeIn bool) InterruptionPolicy {
	p := InterruptionPolicy{VADBargeInEnabled: defaultVADBargeIn}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return p
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return p
	}
	if v, ok := m["vadBargeInEnabled"].(bool); ok {
		p.VADBargeInEnabled = v
	}
	if v, ok := m["uninterruptableAfterPlayStartMs"].(float64); ok && v > 0 {
		p.UnInterruptableAfterPlayStart = int(v)
	}
	return p
}

// InterruptionFromEnv derives interruption policy from VoiceEnv.
func InterruptionFromEnv(env tenantcfg.VoiceEnv) InterruptionPolicy {
	p := InterruptionPolicy{VADBargeInEnabled: true}
	if len(env.InterruptionConfigRaw) == 0 {
		return p
	}
	if v, ok := env.InterruptionConfigRaw["vadBargeInEnabled"].(bool); ok {
		p.VADBargeInEnabled = v
	}
	if v, ok := env.InterruptionConfigRaw["uninterruptableAfterPlayStartMs"].(float64); ok && v > 0 {
		p.UnInterruptableAfterPlayStart = int(v)
	}
	return p
}

// ProcessConfigFromEnv derives audio processing settings from VoiceEnv.
func ProcessConfigFromEnv(env tenantcfg.VoiceEnv) ProcessConfig {
	cfg := ProcessConfig{VadType: "energy"}
	if len(env.AudioProcessConfigRaw) == 0 {
		return cfg
	}
	if v, ok := env.AudioProcessConfigRaw["vadType"].(string); ok && strings.TrimSpace(v) != "" {
		cfg.VadType = strings.TrimSpace(v)
	}
	return cfg
}
