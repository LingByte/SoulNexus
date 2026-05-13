// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package main

// xiaozhi WebSocket adapter mount. Registers a handler on the shared
// HTTP mux owned by main.startHTTPListener — the same listener that
// also serves the WebRTC offer endpoint and demo page. ESP32 hardware
// (audio_params.format=opus) and browser web clients (format=pcm) both
// land on this single URL; the handshake distinguishes them.
//
// Lives in the voiceserver command rather than the library so the
// env-var sourcing logic matches the SIP path (ASR_*/TTS_* read once at
// startup) — pkg/voice/xiaozhi stays free of process-wide config.

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	voicetts "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/xiaozhi"
	"gorm.io/gorm"
)

// xiaozhiDemoHTML is the browser-side test page mounted at
// `/xiaozhi/demo`. It captures the mic at 16 kHz, streams PCM16 LE
// frames over the xiaozhi WS protocol to the same endpoint ESP32
// firmware uses, and plays back the AI's TTS PCM via Web Audio. The
// fully-self-contained page exercises the same dialog plane as the
// ESP32 hardware path, which is the cheapest end-to-end smoke test.
//
//go:embed xiaozhi_demo.html
var xiaozhiDemoHTML []byte

// mountXiaozhiHandlers registers the xiaozhi WS handler on the supplied
// mux. Returns true when the handler was actually mounted (i.e. ASR/TTS
// env vars present and -dialog-ws non-empty). Errors are logged and
// the function returns false — the rest of the server keeps running so
// SIP traffic isn't affected by a misconfigured xiaozhi stack.
func mountXiaozhiHandlers(mux *http.ServeMux, path, dialogWS string, db *gorm.DB, record bool, recordBucket string) bool {
	if strings.TrimSpace(dialogWS) == "" {
		log.Printf("[xiaozhi] disabled: -dialog-ws is empty")
		return false
	}
	factory, err := newXiaozhiFactory()
	if err != nil {
		log.Printf("[xiaozhi] disabled: %v", err)
		return false
	}
	srv, err := xiaozhi.NewServer(xiaozhi.ServerConfig{
		SessionFactory:   factory,
		DialogWSURL:      dialogWS,
		CallIDPrefix:     "xz",
		BargeInFactory:   newBargeInDetector,
		ConfigureClient:  applyDialogReconnect,
		Logger:           voiceLogger("xiaozhi"),
		PersisterFactory: makePersisterFactory(db, "xiaozhi", "inbound"),
		RecorderFactory:  makeRecorderFactory(record, recordBucket, "xiaozhi"),
		OnSessionStart: func(_ context.Context, callID, deviceID string) {
			log.Printf("[xiaozhi] session start call=%s device=%s", callID, deviceID)
		},
		OnSessionEnd: func(_ context.Context, callID, reason string) {
			log.Printf("[xiaozhi] session end   call=%s reason=%s", callID, reason)
		},
	})
	if err != nil {
		log.Printf("[xiaozhi] init failed: %v", err)
		return false
	}
	mux.HandleFunc(path, srv.Handle)
	// Browser demo at /xiaozhi/demo — does NOT collide with the WS
	// subtree because path has trailing slash (/xiaozhi/v1/) and the
	// demo is at a different prefix. The page uses ws://<host>/xiaozhi/v1/
	// by default but is configurable.
	mux.HandleFunc("/xiaozhi/demo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(xiaozhiDemoHTML)
	})
	log.Printf("[xiaozhi] mounted: ws://<http>%s demo=/xiaozhi/demo -> dialog=%s (esp32+web)", path, dialogWS)
	return true
}

// xiaozhiFactory satisfies xiaozhi.SessionFactory by minting per-session
// QCloud ASR services and a process-shared cached QCloud TTS service.
type xiaozhiFactory struct {
	asrAppID string
	asrSID   string
	asrSKey  string
	asrModel string
	asrRate  int

	ttsService voicetts.Service // shared (process-cached)
	ttsSR      int
}

func newXiaozhiFactory() (*xiaozhiFactory, error) {
	asrAppID := strings.TrimSpace(os.Getenv("ASR_APPID"))
	asrSID := strings.TrimSpace(os.Getenv("ASR_SECRET_ID"))
	asrSKey := strings.TrimSpace(os.Getenv("ASR_SECRET_KEY"))
	asrModel := strings.TrimSpace(os.Getenv("ASR_MODEL_TYPE"))
	ttsAppID := strings.TrimSpace(os.Getenv("TTS_APPID"))
	ttsSID := strings.TrimSpace(os.Getenv("TTS_SECRET_ID"))
	ttsSKey := strings.TrimSpace(os.Getenv("TTS_SECRET_KEY"))
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
	// `-tts-prewarm=false` (the default) means absolutely zero TTS calls
	// fire at boot, regardless of which transports are mounted.
	if cached != nil && ttsPrewarmEnabled() {
		go cached.Prewarm(context.Background(), prewarmTexts(), func(t string, e error) {
			log.Printf("[xiaozhi][tts-cache] prewarm %q failed: %v", t, e)
		})
	}
	return &xiaozhiFactory{
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
func (f *xiaozhiFactory) NewASR(_ context.Context, _ string) (recognizer.TranscribeService, int, error) {
	opt := recognizer.NewQcloudASROption(f.asrAppID, f.asrSID, f.asrSKey)
	if f.asrModel != "" {
		opt.ModelType = f.asrModel
	}
	return recognizer.NewQcloudASR(opt), f.asrRate, nil
}

// TTS returns the shared cached service.
func (f *xiaozhiFactory) TTS(_ context.Context, _ string) (voicetts.Service, int, error) {
	return f.ttsService, f.ttsSR, nil
}
