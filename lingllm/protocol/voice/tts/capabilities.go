package tts

import "github.com/LingByte/lingllm/synthesizer"

// Capabilities describes vendor synthesis behavior for pipeline tuning.
type Capabilities struct {
	StreamingTTFB bool
	FirstMaxChars int
	FirstMinChars int
}

// DefaultCapabilities returns conservative batch-oriented defaults.
func DefaultCapabilities() Capabilities {
	c := synthesizer.DefaultTTSCapabilities()
	return Capabilities{
		StreamingTTFB: c.StreamingTTFB,
		FirstMaxChars: c.SuggestedFirstMaxRunes,
		FirstMinChars: 2,
	}
}

// CapabilitiesFrom inspects a TTSService for optional vendor hints.
func CapabilitiesFrom(svc TTSService) Capabilities {
	if svc == nil {
		return DefaultCapabilities()
	}
	if c, ok := svc.(interface{ Capabilities() Capabilities }); ok {
		return c.Capabilities()
	}
	if a, ok := svc.(*synthesisAdapter); ok && a != nil && a.engine != nil {
		if c, ok := a.engine.(synthesizer.CapableSynthesisEngine); ok {
			cap := c.Capabilities()
			return Capabilities{
				StreamingTTFB: cap.StreamingTTFB,
				FirstMaxChars: cap.SuggestedFirstMaxRunes,
				FirstMinChars: 2,
			}
		}
	}
	return DefaultCapabilities()
}
