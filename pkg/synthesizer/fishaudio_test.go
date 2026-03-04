package synthesizer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFishAudioConfig(t *testing.T) {
	config := NewFishAudioConfig("test-api-key", "test-model-id")

	assert.Equal(t, config.APIKey, "test-api-key")
	assert.Equal(t, config.ReferenceID, "test-model-id")
	assert.Equal(t, config.Model, "s1")
	assert.Equal(t, config.Format, "mp3")
	assert.Equal(t, config.SampleRate, 44100)
	assert.Equal(t, config.Temperature, 0.7)
	assert.Equal(t, config.TopP, 0.7)
	assert.Equal(t, config.Latency, "normal")
}

func TestFishAudioService(t *testing.T) {
	config := NewFishAudioConfig("test-api-key", "test-model-id")
	svc := NewFishAudioService(config)

	assert.Equal(t, svc.Provider(), ProviderFishAudio)
	assert.Equal(t, svc.Format().SampleRate, 44100)
	assert.Equal(t, svc.Format().BitDepth, 16)
	assert.Equal(t, svc.Format().Channels, 1)
}

func TestFishAudioCacheKey(t *testing.T) {
	config := NewFishAudioConfig("test-api-key", "test-model-id")
	svc := NewFishAudioService(config)

	cacheKey := svc.CacheKey("Hello, world!")
	assert.NotEmpty(t, cacheKey)
	assert.Contains(t, cacheKey, "fishaudio.tts")
	assert.Contains(t, cacheKey, "test-model-id")
}
