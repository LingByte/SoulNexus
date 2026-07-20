package tts

import (
	"fmt"

	"github.com/LingByte/lingllm/synthesizer"
)

// CredentialHandle bundles a lingllm AudioSynthesisEngine with the
// streaming Service contract consumed by siptts.Pipeline.
type CredentialHandle struct {
	Engine     synthesizer.AudioSynthesisEngine
	Service    Service
	SampleRate int
}

// NewFromCredential resolves tenant TTS credentials through lingllm and
// adapts the engine to the local streaming Service interface.
func NewFromCredential(cfg synthesizer.TTSCredentialConfig) (*CredentialHandle, error) {
	engine, err := synthesizer.NewAudioSynthesisEngineFromCredential(cfg)
	if err != nil {
		return nil, err
	}
	svc := FromSynthesisEngine(engine)
	if svc == nil {
		_ = engine.Close()
		return nil, fmt.Errorf("voice/tts: failed to wrap synthesis engine")
	}
	return &CredentialHandle{
		Engine:     engine,
		Service:    svc,
		SampleRate: engine.Format().SampleRate,
	}, nil
}
