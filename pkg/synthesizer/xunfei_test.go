package synthesizer

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXunfeiService(t *testing.T) {
	appID := os.Getenv("XUNFEI_APP_ID")
	apiKey := os.Getenv("XUNFEI_API_KEY")
	apiSecret := os.Getenv("XUNFEI_API_SECRET")
	if appID == "" || apiKey == "" || apiSecret == "" {
		t.Skip("missing XUNFEI credentials")
	}

	opt := NewXunfeiTTSConfig(appID, apiKey, apiSecret)

	svc := NewXunfeiService(opt)

	assert.Equal(t, svc.Provider(), ProviderXunfei)
	assert.Equal(t, svc.Format().SampleRate, 24000)
	assert.Equal(t, svc.Format().BitDepth, 16)
	assert.Equal(t, svc.Format().Channels, 1)

	key := svc.CacheKey("hello")
	assert.Contains(t, key, "xunfei.tts")

	ctx := context.Background()

	h := &testSynthesisHandler{}
	err := svc.Synthesize(ctx, h, "hello LingEcho")

	if err != nil {
		t.Logf("Synthesis error: %v", err)
	} else {
		assert.Greater(t, len(h.result), 0)
	}
}
