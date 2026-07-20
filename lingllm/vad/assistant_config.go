package vad

import (
	"encoding/json"
	"math"
)

// AssistantConfig maps assistant vad_config JSON to RMS detector knobs.
type AssistantConfig struct {
	EnergyThreshold int
	MinSpeechFrames int
	SpeechStartMs   int
	SpeechEndMs     int
	Ratio           float64
}

// ParseAssistantConfig decodes assistant vad_config JSON.
func ParseAssistantConfig(raw map[string]any) AssistantConfig {
	var cfg AssistantConfig
	if len(raw) == 0 {
		return cfg
	}
	cfg.EnergyThreshold = intFromAny(raw["energyThreshold"])
	cfg.MinSpeechFrames = intFromAny(raw["minSpeechFrames"])
	cfg.SpeechStartMs = intFromAny(raw["speechStartMs"])
	cfg.SpeechEndMs = intFromAny(raw["speechEndMs"])
	if v, ok := raw["ratio"].(float64); ok && v > 0 {
		cfg.Ratio = v
	}
	if cfg.Ratio <= 0 {
		if v := intFromAny(raw["vadMode"]); v > 0 {
			cfg.Ratio = 1 + float64(v)*0.15
		}
	}
	if pos, ok := raw["positiveSpeechThreshold"].(float64); ok && pos > 0 && cfg.Ratio <= 0 {
		cfg.Ratio = pos
	}
	return cfg
}

// ParseAssistantConfigBytes decodes raw JSON bytes.
func ParseAssistantConfigBytes(raw []byte) AssistantConfig {
	if len(raw) == 0 {
		return AssistantConfig{}
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return AssistantConfig{}
	}
	return ParseAssistantConfig(m)
}

// ApplyAssistantConfig maps assistant JSON onto the RMS barge-in detector.
func (d *EnergyDetector) ApplyAssistantConfig(cfg AssistantConfig, sipThreshold float64, sipConsec int) {
	if d == nil {
		return
	}
	threshold := sipThreshold
	if cfg.EnergyThreshold > 0 {
		threshold = float64(cfg.EnergyThreshold)
		// Legacy assistants often saved energyThreshold≈400 which false-triggers
		// barge-in on TTS echo. Floor to voice default (or 5500) for barge-in safety.
		minBarge := sipThreshold
		if minBarge < 2500 {
			minBarge = 5500
		}
		if threshold < minBarge {
			threshold = minBarge
		}
	} else if cfg.Ratio > 0 && sipThreshold > 0 {
		threshold = sipThreshold * cfg.Ratio
	}
	if threshold > 0 {
		d.SetThreshold(threshold)
	}
	frames := sipConsec
	if cfg.MinSpeechFrames > 0 {
		frames = cfg.MinSpeechFrames
	} else if cfg.SpeechStartMs > 0 {
		frames = int(math.Max(1, math.Round(float64(cfg.SpeechStartMs)/20.0)))
	}
	if frames > 0 {
		d.SetConsecutiveFrames(frames)
	}
	_ = cfg.SpeechEndMs
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}
