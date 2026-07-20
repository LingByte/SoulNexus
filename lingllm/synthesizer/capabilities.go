package synthesizer

// TTSCapabilities describes vendor-specific synthesis behavior for pipeline tuning.
type TTSCapabilities struct {
	// StreamingTTFB is true when the first audio chunk can arrive before synthesis completes.
	StreamingTTFB bool
	// SuggestedFirstMaxRunes is a segmenter hint for the first LLM→TTS chunk.
	SuggestedFirstMaxRunes int
}

// DefaultTTSCapabilities returns conservative defaults for batch-oriented vendors.
func DefaultTTSCapabilities() TTSCapabilities {
	return TTSCapabilities{
		StreamingTTFB:          false,
		SuggestedFirstMaxRunes: 12,
	}
}

// CapableSynthesisEngine optionally exposes vendor capabilities.
type CapableSynthesisEngine interface {
	AudioSynthesisEngine
	Capabilities() TTSCapabilities
}
