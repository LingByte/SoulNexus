// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package main

// WebRTC HTTP signaling listener bootstrap. The endpoint terminates 1v1
// browser-to-AI WebRTC calls — pion handles ICE/STUN/TURN, DTLS-SRTP, NACK,
// TWCC; pkg/voice/webrtc translates Opus ↔ PCM, runs ASR + TTS, and bridges
// to the dialog plane (-dialog-ws) so the same dialog application serves
// SIP, xiaozhi, and WebRTC clients without any code change.
//
// Lives in cmd so the env-var sourcing logic matches the SIP / xiaozhi
// paths (ASR_*/TTS_*/STUN/TURN read once at startup).

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
	voicertc "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/webrtc"
	pionwebrtc "github.com/pion/webrtc/v4"
	"gorm.io/gorm"
)

// webrtcDemoHTML is the browser-side test page mounted at
// `${offerPath_root}/demo`. It captures the mic, runs the offer/answer
// dance against our /webrtc/v1/offer endpoint, plays back the AI track,
// and streams pion-side stats every 2 s. Useful for end-to-end smoke
// tests without writing any frontend code.
//
//go:embed webrtc_demo.html
var webrtcDemoHTML []byte

// mountWebRTCHandlers registers the WebRTC offer/hangup/demo endpoints
// on the supplied mux. Returns true when handlers were actually mounted
// (env vars + dialog-ws present and ICE config parsed). Errors are
// logged and the function returns false; the rest of the server keeps
// running so SIP/xiaozhi traffic isn't affected by a broken WebRTC
// stack.
func mountWebRTCHandlers(mux *http.ServeMux, offerPath, dialogWS string, db *gorm.DB, record bool, recordBucket string) bool {
	if strings.TrimSpace(dialogWS) == "" {
		log.Printf("[webrtc] disabled: -dialog-ws is empty")
		return false
	}
	factory, err := newWebRTCFactory()
	if err != nil {
		log.Printf("[webrtc] disabled: %v", err)
		return false
	}
	iceServers, err := voicertc.ParseICEServers(strings.TrimSpace(os.Getenv("WEBRTC_ICE_SERVERS")))
	if err != nil {
		log.Printf("[webrtc] disabled: bad WEBRTC_ICE_SERVERS: %v", err)
		return false
	}
	publicIPs := splitCSV(os.Getenv("WEBRTC_PUBLIC_IPS"))
	srv, err := voicertc.NewServer(voicertc.ServerConfig{
		SessionFactory: factory,
		DialogWSURL:    dialogWS,
		ICEServers:     iceServers,
		Engine: voicertc.EngineConfig{
			PublicIPs:  publicIPs,
			SinglePort: parseSinglePort(os.Getenv("WEBRTC_UDP_PORT")),
		},
		AllowedOrigins:   splitCSV(os.Getenv("WEBRTC_ALLOWED_ORIGINS")),
		CallIDPrefix:     "wrtc",
		BargeInFactory:   newBargeInDetector,
		DenoiserFactory:  newDenoiserOrNil,
		ConfigureClient:  applyDialogReconnect,
		Logger:           voiceLogger("webrtc"),
		PersisterFactory: makePersisterFactory(db, "webrtc", "inbound"),
		RecorderFactory:  makeRecorderFactory(record, recordBucket, "webrtc"),
		OnSessionStart: func(_ context.Context, callID, peer string) {
			log.Printf("[webrtc] session start call=%s peer=%s", callID, peer)
		},
		OnSessionEnd: func(_ context.Context, callID, reason string) {
			log.Printf("[webrtc] session end   call=%s reason=%s", callID, reason)
		},
	})
	if err != nil {
		log.Printf("[webrtc] init failed: %v", err)
		return false
	}
	mux.HandleFunc(offerPath, srv.HandleOffer)
	rootPath := strings.TrimSuffix(offerPath, "/offer")
	mux.HandleFunc(rootPath+"/hangup", srv.HandleHangup)
	// Demo page at e.g. /webrtc/v1/demo. Serves the embedded HTML so a
	// browser can place a call without any external static-asset setup.
	demoPath := rootPath + "/demo"
	mux.HandleFunc(demoPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(webrtcDemoHTML)
	})
	log.Printf("[webrtc] mounted: offer=%s hangup=%s demo=%s (ice=%d public=%v) -> dialog=%s",
		offerPath, rootPath+"/hangup", demoPath, len(iceServers), publicIPs, dialogWS)
	return true
}

// webrtcFactory satisfies voicertc.SessionFactory using the same QCloud
// ASR / TTS plumbing the SIP and xiaozhi paths use. The shared cached
// TTS service is constructed once and reused across calls.
type webrtcFactory struct {
	asrAppID, asrSID, asrSKey, asrModel string
	asrRate                             int
	ttsService                          voicetts.Service
	ttsSR                               int
}

func newWebRTCFactory() (*webrtcFactory, error) {
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
		VoiceKey: fmt.Sprintf("qcloud-%d-%d", ttsVoice, ttsSR),
		MaxRunes: 80,
	})
	var sharedTTS voicetts.Service = cached
	if cached == nil {
		sharedTTS = voicetts.FromSynthesisService(ttsSvc)
	}
	return &webrtcFactory{
		asrAppID: asrAppID, asrSID: asrSID, asrSKey: asrSKey, asrModel: asrModel,
		asrRate: asrRate, ttsService: sharedTTS, ttsSR: ttsSR,
	}, nil
}

func (f *webrtcFactory) NewASR(_ context.Context, _ string) (recognizer.TranscribeService, int, error) {
	opt := recognizer.NewQcloudASROption(f.asrAppID, f.asrSID, f.asrSKey)
	if f.asrModel != "" {
		opt.ModelType = f.asrModel
	}
	return recognizer.NewQcloudASR(opt), f.asrRate, nil
}

func (f *webrtcFactory) TTS(_ context.Context, _ string) (voicetts.Service, int, error) {
	return f.ttsService, f.ttsSR, nil
}

// ----- helpers -----

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseSinglePort(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	var p int
	_, _ = fmt.Sscanf(s, "%d", &p)
	return p
}

// Avoid unused import when ICEServer types aren't referenced directly.
var _ = pionwebrtc.ICEServer{}
