package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/stretchr/testify/assert"
)

type testSynthesisHandler struct {
	result []byte
}

func (h *testSynthesisHandler) OnMessage(buf []byte) {
	h.result = append(h.result, buf...)
}

func (h *testSynthesisHandler) OnTimestamp(timestamp SentenceTimestamp) {

}

func TestQCloudSerivce(t *testing.T) {
	// Initialize logger for tests
	_ = logger.Init(&logger.LogConfig{
		Level:    "info",
		Filename: "",
	}, "test")
	appID := utils.GetEnv("QCLOUD_APP_ID")
	secretID := utils.GetEnv("QCLOUD_SECRET_ID")
	secretKey := utils.GetEnv("QCLOUD_SECRET")
	if appID == "" {
		t.Skip("missing parameters")
	}

	opt := NewQcloudTTSConfig(appID, secretID, secretKey, 1005, "pcm", 16000)

	svc := NewQCloudService(opt)

	assert.Equal(t, svc.Provider(), ProviderTencent)
	assert.Equal(t, svc.Format().SampleRate, 16000)
	assert.Equal(t, svc.Format().BitDepth, 16)
	assert.Equal(t, svc.Format().Channels, 1)

	key := svc.CacheKey("hello")
	assert.Equal(t, key, "qcloud.tts-1005-1-16000-5d41402abc4b2a76b9719d911017c592.pcm")
	ctx := context.Background()

	h := &testSynthesisHandler{}
	svc.Synthesize(ctx, h, "hello lingecho")
	// 保存为OPUS文件（可直接播放）
	err := ioutil.WriteFile("output.opus", h.result, 0644)
	if err != nil {
		t.Fatal("保存音频文件失败:", err)
	}

	t.Log("音频文件已保存为 output.opus")
}
