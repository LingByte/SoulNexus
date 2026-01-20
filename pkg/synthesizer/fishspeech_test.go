package synthesizer

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFishSpeechService(t *testing.T) {
	apiKey := os.Getenv("FISHSPEECH_API_KEY")
	referenceID := os.Getenv("FISHSPEECH_REFERENCE_ID")
	if apiKey == "" {
		t.Skip("missing FISHSPEECH_API_KEY")
	}

	if referenceID == "" {
		referenceID = "default"
	}

	opt := NewFishSpeechConfig(apiKey, referenceID)

	svc := NewFishSpeechService(opt)

	assert.Equal(t, svc.Provider(), ProviderFishSpeech)
	assert.Equal(t, svc.Format().SampleRate, 24000)
	assert.Equal(t, svc.Format().BitDepth, 16)
	assert.Equal(t, svc.Format().Channels, 1)

	key := svc.CacheKey("hello")
	assert.Contains(t, key, "fishspeech.tts")

	ctx := context.Background()

	h := &testSynthesisHandler{}
	err := svc.Synthesize(ctx, h, "hello LingEcho")

	if err != nil {
		t.Logf("Synthesis error: %v", err)
	} else {
		assert.Greater(t, len(h.result), 0)
	}
}
