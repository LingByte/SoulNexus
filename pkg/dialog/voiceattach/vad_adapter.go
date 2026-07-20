package voiceattach

import (
	"context"

	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/dialog/voiceruntime"
	sipvad "github.com/LingByte/SoulNexus/pkg/vad"
	"go.uber.org/zap"
)

const (
	defaultVADBargeIn      = true
	defaultVADThreshold    = 5500.0
	defaultVADConsecFrames = 3
)

type vadAdapter struct{ *sipvad.Detector }

func (v vadAdapter) CheckBargeIn(pcm []byte, synthPlaying bool) bool {
	if v.Detector == nil {
		return false
	}
	return v.Detector.CheckBargeIn(pcm, synthPlaying)
}

type envVAD struct {
	base *sipvad.Detector
	rt   *voiceruntime.Runtime
}

func (v *envVAD) CheckBargeIn(pcm []byte, synthPlaying bool) bool {
	if v == nil || !synthPlaying {
		return false
	}
	if v.rt != nil && !v.rt.UserSpeechLikely(pcm) {
		return false
	}
	if v.base != nil {
		return v.base.CheckBargeIn(pcm, synthPlaying)
	}
	return v.rt != nil
}

func buildNativeVAD(env VoiceEnv, lg *zap.Logger) cascaded.BargeInDetector {
	policy := voiceruntime.InterruptionFromEnv(tenantcfg.VoiceEnv(env))
	if !policy.VADBargeInEnabled {
		return nil
	}
	rt := voiceruntime.Prepare(context.Background(), tenantcfg.VoiceEnv(env), 16000, lg)
	d := sipvad.NewDetector()
	d.SetLogger(lg)
	threshold := defaultVADThreshold
	consec := defaultVADConsecFrames
	if v := parseVADFloat(env.VadConfigRaw, "energyThreshold"); v > 0 {
		minBarge := threshold
		if minBarge < 2500 {
			minBarge = 5500
		}
		if v < minBarge {
			if lg != nil {
				lg.Warn("native cascaded: assistant energyThreshold too low; using default",
					zap.Float64("configured", v),
					zap.Float64("effective", minBarge))
			}
			v = minBarge
		}
		d.SetThreshold(v)
		threshold = v
	} else {
		d.SetThreshold(threshold)
	}
	if n := parseVADInt(env.VadConfigRaw, "minSpeechFrames"); n > 0 {
		d.SetConsecutiveFrames(n)
		consec = n
	} else {
		d.SetConsecutiveFrames(consec)
	}
	if lg != nil {
		lg.Info("native cascaded: VAD barge-in enabled",
			zap.Float64("threshold_effective", threshold),
			zap.Int("consecutive_frames", consec),
			zap.String("vad_type", rt.AudioProc.VadType),
		)
	}
	switch rt.AudioProc.VadType {
	case "webrtc", "silero", "tinysilero", "tiny_silero", "levad_silero",
		"tinyten", "tiny_ten", "ten", "levad_ten":
		return &envVAD{base: d, rt: rt}
	}
	return vadAdapter{d}
}

func parseVADFloat(raw map[string]any, key string) float64 {
	if len(raw) == 0 {
		return 0
	}
	switch v := raw[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0
	}
}

func parseVADInt(raw map[string]any, key string) int {
	if len(raw) == 0 {
		return 0
	}
	switch v := raw[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}
