package synthesizer

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestElevenLabsService(t *testing.T) {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	voiceID := os.Getenv("ELEVENLABS_VOICE_ID")
	if apiKey == "" {
		t.Skip("missing ELEVENLABS_API_KEY")
	}

	if voiceID == "" {
		voiceID = "21m00Tcm4TlvDq8ikWAM" // 默认 Rachel 音色
	}

	opt := NewElevenLabsConfig(apiKey, voiceID)

	svc := NewElevenLabsService(opt)

	assert.Equal(t, svc.Provider(), ProviderElevenLabs)
	assert.Equal(t, svc.Format().SampleRate, 44100)
	assert.Equal(t, svc.Format().BitDepth, 16)
	assert.Equal(t, svc.Format().Channels, 1)

	key := svc.CacheKey("hello")
	assert.Contains(t, key, "elevenlabs.tts")

	ctx := context.Background()

	h := &testSynthesisHandler{}
	err := svc.Synthesize(ctx, h, "Hello, this is a test from ElevenLabs TTS")

	if err != nil {
		t.Logf("Synthesis error: %v", err)
	} else {
		assert.Greater(t, len(h.result), 0)
	}
}
