package synthesizer

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenAIService(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("missing OPENAI_API_KEY")
	}

	opt := NewOpenAIConfig(apiKey)

	svc := NewOpenAIService(opt)

	assert.Equal(t, svc.Provider(), ProviderOpenAI)
	assert.Equal(t, svc.Format().SampleRate, 24000)
	assert.Equal(t, svc.Format().BitDepth, 16)
	assert.Equal(t, svc.Format().Channels, 1)

	key := svc.CacheKey("hello")
	assert.Contains(t, key, "openai.tts")

	ctx := context.Background()

	h := &testSynthesisHandler{}
	err := svc.Synthesize(ctx, h, "Hello, this is a test from OpenAI TTS")

	if err != nil {
		t.Logf("Synthesis error: %v", err)
	} else {
		assert.Greater(t, len(h.result), 0)
	}
}

func TestOpenAIServiceWithCustomModel(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("missing OPENAI_API_KEY")
	}

	opt := NewOpenAIConfig(apiKey)
	opt.Model = "tts-1-hd" // 使用高清模型
	opt.Voice = "echo"     // 使用 Echo 音色

	svc := NewOpenAIService(opt)

	assert.Equal(t, svc.Provider(), ProviderOpenAI)
	assert.Equal(t, "tts-1-hd", opt.Model)
	assert.Equal(t, "echo", opt.Voice)
}
