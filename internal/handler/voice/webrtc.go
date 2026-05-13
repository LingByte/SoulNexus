// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voice

import (
	"context"
	"log"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/app"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	voicertc "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/webrtc"
)

// mountWebRTC registers the WebRTC offer/hangup endpoints. Returns true
// when handlers were actually mounted (env vars + dialog-ws present and
// ICE config parsed). Errors are logged and the function returns false
// — the rest of the listener keeps running so SIP/xiaozhi traffic isn't
// affected by a broken WebRTC stack.
//
// pion handles ICE/STUN/TURN, DTLS-SRTP, NACK, TWCC; pkg/voiceserver/
// voice/webrtc translates Opus ↔ PCM, runs ASR + TTS, and bridges to the
// dialog plane (VOICE_DIALOG_WS) so the same dialog application serves
// SIP, xiaozhi, and WebRTC clients without any code change.
func (h *Handlers) mountWebRTC(r gin.IRoutes) bool {
	dialogWS := strings.TrimSpace(h.cfg.DialogWS)
	if dialogWS == "" {
		log.Printf("[webrtc] disabled: VOICE_DIALOG_WS is empty")
		return false
	}
	factory, err := app.NewWebRTCFactory()
	if err != nil {
		log.Printf("[webrtc] disabled: %v", err)
		return false
	}
	iceServers, err := voicertc.ParseICEServers(strings.TrimSpace(utils.GetEnv("WEBRTC_ICE_SERVERS")))
	if err != nil {
		log.Printf("[webrtc] disabled: bad WEBRTC_ICE_SERVERS: %v", err)
		return false
	}
	publicIPs := app.SplitCSV(utils.GetEnv("WEBRTC_PUBLIC_IPS"))
	srv, err := voicertc.NewServer(voicertc.ServerConfig{
		SessionFactory: factory,
		DialogWSURL:    dialogWS,
		ICEServers:     iceServers,
		Engine: voicertc.EngineConfig{
			PublicIPs:  publicIPs,
			SinglePort: app.ParseSinglePort(utils.GetEnv("WEBRTC_UDP_PORT")),
		},
		AllowedOrigins:   app.SplitCSV(utils.GetEnv("WEBRTC_ALLOWED_ORIGINS")),
		CallIDPrefix:     "wrtc",
		BargeInFactory:   app.NewBargeInDetector,
		DenoiserFactory:  app.NewDenoiserOrNil,
		ConfigureClient:  app.ApplyDialogReconnect,
		Logger:           app.VoiceLogger("webrtc"),
		PersisterFactory: app.MakePersisterFactory(h.cfg.DB, "webrtc", "inbound"),
		RecorderFactory:  app.MakeRecorderFactory(h.cfg.Record, "webrtc"),
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
	offerPath := h.cfg.WebRTCOfferPath
	r.Any(offerPath, gin.WrapF(srv.HandleOffer))
	rootPath := strings.TrimSuffix(offerPath, "/offer")
	r.Any(rootPath+"/hangup", gin.WrapF(srv.HandleHangup))
	log.Printf("[webrtc] mounted: offer=%s hangup=%s (ice=%d public=%v) -> dialog=%s",
		offerPath, rootPath+"/hangup", len(iceServers), publicIPs, gateway.RedactDialogDialURL(dialogWS))
	return true
}
