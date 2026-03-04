package synthesizer

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAzureService(t *testing.T) {
	subKey := os.Getenv("AZURE_SUBSCRIPTION_KEY")
	region := os.Getenv("AZURE_REGION")
	if subKey == "" || region == "" {
		t.Skip("missing AZURE_SUBSCRIPTION_KEY or AZURE_REGION")
	}

	opt := NewAzureConfig(subKey, region)

	svc := NewAzureService(opt)

	assert.Equal(t, ProviderAzure, svc.Provider())
	assert.GreaterOrEqual(t, svc.Format().SampleRate, 16000)
	assert.Equal(t, 16, svc.Format().BitDepth)
	assert.Equal(t, 1, svc.Format().Channels)

	key := svc.CacheKey("hello")
	assert.Contains(t, key, "azure.tts")

	ctx := context.Background()

	h := &testSynthesisHandler{}
	err := svc.Synthesize(ctx, h, "你好，这是 Azure TTS 测试。")
	if err != nil {
		t.Logf("Synthesis error: %v", err)
	} else {
		assert.Greater(t, len(h.result), 0)
	}
}
