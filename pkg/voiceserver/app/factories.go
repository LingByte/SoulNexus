// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/LingByte/SoulNexus/pkg/utils"
	voicetts "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
)

// XiaozhiFactory satisfies xiaozhi.SessionFactory by minting per-session
// QCloud ASR services and a process-shared cached QCloud TTS service.
//
// The struct is exported so cmd/voice and internal/handler/voice can both
// hold the value without import cycles, but its fields are kept package-
// private — callers should treat it as opaque and only invoke the
// SessionFactory methods.
type XiaozhiFactory struct {
	asrAppID string
	asrSID   string
	asrSKey  string
	asrModel string
	asrRate  int

	ttsService voicetts.Service // shared (process-cached)
	ttsSR      int
}

// NewXiaozhiFactory reads the QCloud ASR/TTS env vars and returns a
// session factory ready to feed xiaozhi.ServerConfig.SessionFactory.
// Returns an error when any required env is missing — callers typically
// log and disable the xiaozhi transport rather than aborting the process.
func NewXiaozhiFactory() (*XiaozhiFactory, error) {
	asrAppID := strings.TrimSpace(utils.GetEnv("ASR_APPID"))
	asrSID := strings.TrimSpace(utils.GetEnv("ASR_SECRET_ID"))
	asrSKey := strings.TrimSpace(utils.GetEnv("ASR_SECRET_KEY"))
	asrModel := strings.TrimSpace(utils.GetEnv("ASR_MODEL_TYPE"))
	ttsAppID := strings.TrimSpace(utils.GetEnv("TTS_APPID"))
	ttsSID := strings.TrimSpace(utils.GetEnv("TTS_SECRET_ID"))
	ttsSKey := strings.TrimSpace(utils.GetEnv("TTS_SECRET_KEY"))
	if asrAppID == "" || asrSID == "" || asrSKey == "" {
		return nil, fmt.Errorf("missing ASR_APPID/ASR_SECRET_ID/ASR_SECRET_KEY")
	}
	if ttsAppID == "" || ttsSID == "" || ttsSKey == "" {
		return nil, fmt.Errorf("missing TTS_APPID/TTS_SECRET_ID/TTS_SECRET_KEY")
	}
	asrRate := 16000
	if strings.Contains(strings.ToLower(asrModel), "8k") {
		asrRate = 8000
	}
	const ttsSR = 16000
	var ttsVoice int64 = 101007
	ttsCfg := synthesizer.NewQcloudTTSConfig(ttsAppID, ttsSID, ttsSKey, ttsVoice, "pcm", ttsSR)
	ttsSvc := synthesizer.NewQCloudService(ttsCfg)
	cached, _ := voicetts.NewCachingService(voicetts.FromSynthesisService(ttsSvc), voicetts.CacheConfig{
		VoiceKey:   fmt.Sprintf("qcloud-%d-%d", ttsVoice, ttsSR),
		MaxRunes:   80,
		ChunkBytes: 0,
	})
	var sharedTTS voicetts.Service = cached
	if cached == nil {
		sharedTTS = voicetts.FromSynthesisService(ttsSvc)
	}
	// Best-effort prewarm of common short replies (shared cache benefits
	// every device after the first call). Same gate as the SIP path so
	// VOICE_TTS_PREWARM=false (the default) means absolutely zero TTS
	// calls fire at boot, regardless of which transports are mounted.
	if cached != nil && TTSPrewarmEnabled() {
		go cached.Prewarm(context.Background(), PrewarmTexts(), func(t string, e error) {
			log.Printf("[xiaozhi][tts-cache] prewarm %q failed: %v", t, e)
		})
	}
	return &XiaozhiFactory{
		asrAppID:   asrAppID,
		asrSID:     asrSID,
		asrSKey:    asrSKey,
		asrModel:   asrModel,
		asrRate:    asrRate,
		ttsService: sharedTTS,
		ttsSR:      ttsSR,
	}, nil
}

// NewASR creates a per-session QCloud recognizer.
func (f *XiaozhiFactory) NewASR(_ context.Context, _ string) (recognizer.TranscribeService, int, error) {
	opt := recognizer.NewQcloudASROption(f.asrAppID, f.asrSID, f.asrSKey)
	if f.asrModel != "" {
		opt.ModelType = f.asrModel
	}
	return recognizer.NewQcloudASR(opt), f.asrRate, nil
}

// TTS returns the shared cached service.
func (f *XiaozhiFactory) TTS(_ context.Context, _ string) (voicetts.Service, int, error) {
	return f.ttsService, f.ttsSR, nil
}

// WebRTCFactory satisfies voicertc.SessionFactory using the same QCloud
// ASR / TTS plumbing the SIP and xiaozhi paths use. The shared cached
// TTS service is constructed once and reused across calls.
type WebRTCFactory struct {
	asrAppID, asrSID, asrSKey, asrModel string
	asrRate                             int
	ttsService                          voicetts.Service
	ttsSR                               int
}

// NewWebRTCFactory mirrors NewXiaozhiFactory but without the prewarm side
// effect — WebRTC sessions are short-lived browser calls so a process-
// wide prewarm only adds startup latency.
func NewWebRTCFactory() (*WebRTCFactory, error) {
	asrAppID := strings.TrimSpace(utils.GetEnv("ASR_APPID"))
	asrSID := strings.TrimSpace(utils.GetEnv("ASR_SECRET_ID"))
	asrSKey := strings.TrimSpace(utils.GetEnv("ASR_SECRET_KEY"))
	asrModel := strings.TrimSpace(utils.GetEnv("ASR_MODEL_TYPE"))
	ttsAppID := strings.TrimSpace(utils.GetEnv("TTS_APPID"))
	ttsSID := strings.TrimSpace(utils.GetEnv("TTS_SECRET_ID"))
	ttsSKey := strings.TrimSpace(utils.GetEnv("TTS_SECRET_KEY"))
	if asrAppID == "" || asrSID == "" || asrSKey == "" {
		return nil, fmt.Errorf("missing ASR_APPID/ASR_SECRET_ID/ASR_SECRET_KEY")
	}
	if ttsAppID == "" || ttsSID == "" || ttsSKey == "" {
		return nil, fmt.Errorf("missing TTS_APPID/TTS_SECRET_ID/TTS_SECRET_KEY")
	}
	asrRate := 16000
	if strings.Contains(strings.ToLower(asrModel), "8k") {
		asrRate = 8000
	}
	const ttsSR = 16000
	var ttsVoice int64 = 101007
	ttsCfg := synthesizer.NewQcloudTTSConfig(ttsAppID, ttsSID, ttsSKey, ttsVoice, "pcm", ttsSR)
	ttsSvc := synthesizer.NewQCloudService(ttsCfg)
	cached, _ := voicetts.NewCachingService(voicetts.FromSynthesisService(ttsSvc), voicetts.CacheConfig{
		VoiceKey: fmt.Sprintf("qcloud-%d-%d", ttsVoice, ttsSR),
		MaxRunes: 80,
	})
	var sharedTTS voicetts.Service = cached
	if cached == nil {
		sharedTTS = voicetts.FromSynthesisService(ttsSvc)
	}
	return &WebRTCFactory{
		asrAppID: asrAppID, asrSID: asrSID, asrSKey: asrSKey, asrModel: asrModel,
		asrRate: asrRate, ttsService: sharedTTS, ttsSR: ttsSR,
	}, nil
}

// NewASR creates a per-session QCloud recognizer.
func (f *WebRTCFactory) NewASR(_ context.Context, _ string) (recognizer.TranscribeService, int, error) {
	opt := recognizer.NewQcloudASROption(f.asrAppID, f.asrSID, f.asrSKey)
	if f.asrModel != "" {
		opt.ModelType = f.asrModel
	}
	return recognizer.NewQcloudASR(opt), f.asrRate, nil
}

// TTS returns the shared cached service.
func (f *WebRTCFactory) TTS(_ context.Context, _ string) (voicetts.Service, int, error) {
	return f.ttsService, f.ttsSR, nil
}
