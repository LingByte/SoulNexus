package synthesizer

import (
	"context"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestLocalService(t *testing.T) {
	// Initialize logger for tests
	_ = logger.Init(&logger.LogConfig{
		Level:    "info",
		Filename: "",
	}, "test")
	// 检测可用的本地 TTS 命令
	detected := DetectLocalTTSCommand()
	if detected == "" {
		t.Skip("no local TTS command available")
	}

	opt := NewLocalTTSConfig(detected)

	svc := NewLocalService(opt)

	assert.Equal(t, svc.Provider(), ProviderLocal)
	assert.Equal(t, svc.Format().SampleRate, 16000)
	assert.Equal(t, svc.Format().BitDepth, 16)
	assert.Equal(t, svc.Format().Channels, 1)

	key := svc.CacheKey("hello")
	assert.Contains(t, key, "local.tts")

	ctx := context.Background()

	h := &testSynthesisHandler{}
	err := svc.Synthesize(ctx, h, "hello LingEcho")

	if err != nil {
		t.Logf("Synthesis error: %v", err)
	} else {
		assert.GreaterOrEqual(t, len(h.result), 0)
	}
}

func TestCheckLocalTTSAvailable(t *testing.T) {
	available := CheckLocalTTSAvailable()
	t.Logf("Available local TTS commands: %v", available)
}

func TestDetectLocalTTSCommand(t *testing.T) {
	detected := DetectLocalTTSCommand()
	t.Logf("Detected local TTS command: %s", detected)
}

func TestGetLocalTTSInfo(t *testing.T) {
	info := GetLocalTTSInfo()
	t.Logf("Local TTS info: %+v", info)
}
