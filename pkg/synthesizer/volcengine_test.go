package synthesizer

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVolcengineService(t *testing.T) {
	// 使用环境变量或直接配置
	appID := os.Getenv("VOLCE_APP_ID")
	accessToken := os.Getenv("VOLCE_TOKEN")
	cluster := os.Getenv("VOLCE_CLUSTER")
	voiceType := os.Getenv("VOLCE_VOICE_TYPE")
	if appID == "" || accessToken == "" || cluster == "" || voiceType == "" {
		t.Skip("missing VOLCE_APP_ID or VOLCE_TOKEN or VOLCE_CLUSTER or VOLCE_VOICE_TYPE")
	}

	opt := NewVolcengineTTSOption(appID, accessToken, cluster)
	opt.VoiceType = voiceType

	svc := NewVolcengineService(opt)

	assert.Equal(t, svc.Provider(), ProviderVolcengine)
	assert.Equal(t, svc.Format().SampleRate, 8000)
	assert.Equal(t, svc.Format().BitDepth, 16)
	assert.Equal(t, svc.Format().Channels, 1)

	key := svc.CacheKey("hello")
	assert.Contains(t, key, "volcengine.tts")

	ctx := context.Background()

	h := &testSynthesisHandler{}
	err := svc.Synthesize(ctx, h, "你好，这是火山引擎TTS测试")

	if err != nil {
		t.Logf("Synthesis error: %v", err)
		t.Fatal(err)
	} else {
		assert.Greater(t, len(h.result), 0, "should get audio data")
		t.Logf("Successfully synthesized audio, size: %d bytes", len(h.result))
	}
}

func TestVolcengineServiceWithFromCredential(t *testing.T) {
	// 测试从配置创建服务
	appID := os.Getenv("VOLCE_APP_ID")
	accessToken := os.Getenv("VOLCE_TOKEN")
	cluster := os.Getenv("VOLCE_CLUSTER")
	if appID == "" || accessToken == "" || cluster == "" {
		t.Skip("missing VOLCE_APP_ID or VOLCE_TOKEN")
	}

	// 模拟从 UserCredential 的 TtsConfig 创建服务
	config := TTSCredentialConfig{
		"provider":    "volcengine",
		"appId":       appID,
		"accessToken": accessToken,
		"cluster":     cluster,
		"voiceType":   "BV700_streaming",
		"rate":        8000,
		"encoding":    "pcm",
		"speedRatio":  1.0,
	}

	svc, err := NewSynthesisServiceFromCredential(config)
	if err != nil {
		t.Fatalf("Failed to create service from credential: %v", err)
	}

	assert.NotNil(t, svc)
	assert.Equal(t, svc.Provider(), ProviderVolcengine)

	ctx := context.Background()
	h := &testSynthesisHandler{}
	err = svc.Synthesize(ctx, h, "测试从配置创建服务")

	if err != nil {
		t.Logf("Synthesis error: %v", err)
		t.Fatal(err)
	} else {
		assert.Greater(t, len(h.result), 0, "should get audio data")
		t.Logf("Successfully synthesized audio from credential, size: %d bytes", len(h.result))
	}
}
