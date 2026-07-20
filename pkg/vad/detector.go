package vad

import (
	llmvad "github.com/LingByte/lingllm/vad"
	"go.uber.org/zap"
)

// Detector wraps lingllm/vad energy detection for voice barge-in.
type Detector struct {
	*llmvad.EnergyDetector
}

// AssistantConfig maps assistant vad_config JSON to RMS detector knobs.
type AssistantConfig = llmvad.AssistantConfig

// NewDetector builds a barge-in detector with voice-aligned defaults.
func NewDetector() *Detector {
	return &Detector{EnergyDetector: llmvad.NewEnergyDetector()}
}

// ParseAssistantConfig decodes assistant vad_config JSON.
func ParseAssistantConfig(raw map[string]any) AssistantConfig {
	return llmvad.ParseAssistantConfig(raw)
}

// ParseAssistantConfigBytes decodes raw JSON bytes.
func ParseAssistantConfigBytes(raw []byte) AssistantConfig {
	return llmvad.ParseAssistantConfigBytes(raw)
}

// SetLogger attaches an optional zap logger.
func (d *Detector) SetLogger(logger *zap.Logger) {
	if d == nil || d.EnergyDetector == nil {
		return
	}
	if logger == nil {
		d.EnergyDetector.SetLogFunc(nil)
		return
	}
	d.EnergyDetector.SetLogFunc(func(msg string) {
		logger.Info(msg)
	})
}

// CalculateRMS computes RMS for PCM16LE frames.
func CalculateRMS(pcmData []byte) float64 {
	return llmvad.CalculateRMS(pcmData)
}

func calculateRMS(pcmData []byte) float64 {
	return CalculateRMS(pcmData)
}
