package synthesizer

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQiniuService(t *testing.T) {
	apiKey := os.Getenv("QINIU_TTS_API_KEY")
	baseURL := os.Getenv("QINIU_TTS_BASE_URL")
	if apiKey == "" {
		t.Skip("missing QINIU_TTS_API_KEY")
	}

	opt := NewQiniuTTSConfig(apiKey, baseURL)

	svc := NewQiniuService(opt)

	assert.Equal(t, svc.Provider(), ProviderQiniu)
	assert.Equal(t, svc.Format().SampleRate, 16000)
	assert.Equal(t, svc.Format().BitDepth, 16)
	assert.Equal(t, svc.Format().Channels, 1)

	key := svc.CacheKey("hello")
	assert.Contains(t, key, "qiniu.tts")

	ctx := context.Background()

	h := &testSynthesisHandler{}
	err := svc.Synthesize(ctx, h, "hello LingEcho")

	if err != nil {
		t.Logf("Synthesis error: %v", err)
	} else {
		assert.Greater(t, len(h.result), 0)
	}
}
