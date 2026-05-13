// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voice

import (
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/xiaozhi"
	"gorm.io/gorm"
)

// Config wires the voice HTTP listener: which transports to mount,
// where they listen, and the credentials/secrets they consult. cmd/voice
// builds this from internal/config.VoiceServerConfig and hands it to
// NewHandlers.
//
// Zero-value semantics:
//   - EnableXiaozhi/EnableWebRTC/EnableSFU = false → that transport is
//     skipped. The HTTP listener still starts as long as ≥1 mount
//     succeeds (or only /healthz + /metrics + /media respond).
//   - SoulnexusHardwarePath empty → the legacy ESP32 binding wrapper
//     mount is not registered.
//   - DB nil → persistence is disabled across all transports.
type Config struct {
	// Shared
	DialogWS     string   // VOICE_DIALOG_WS, with DIALOG_* env merged in
	DB           *gorm.DB // optional; nil disables call_recording / call_events
	Record       bool     // VOICE_RECORD
	RecordBucket string   // VOICE_RECORD_BUCKET

	// WebSocket-Hardware (xiaozhi protocol + optional SoulNexus binding wrapper)
	EnableXiaozhi                  bool
	XiaozhiPath                    string
	SoulnexusHardwarePath          string
	SoulnexusHardwareBindingURL    string
	SoulnexusHardwareBindingSecret string

	// WebRTC (1v1 browser ↔ AI)
	EnableWebRTC    bool
	WebRTCOfferPath string

	// WS-Web (SFU multi-party)
	EnableSFU           bool
	SFUPath             string
	SFUSecret           string
	SFUAllowAnon        bool
	SFUMaxParticipants  int
	SFUMaxRooms         int
	SFUAllowedOrigins   string // CSV; "*" allows any
	SFUTokenAdminSecret string // required to call /token outside anon mode
	SFURecord           bool
	SFURecordBucket     string
	SFUWebhookURL       string
}

// Handlers groups the voice transport HTTP handlers. Construct via
// NewHandlers and call Register(r) once after the gin engine is set up.
//
// State:
//   - cfg snapshots the Config supplied at construction time.
//   - xiaozhiSrv is set after Register if EnableXiaozhi is true and ASR/
//     TTS env vars are present. Soulnexus hardware mount reuses this
//     server (binding wrapper merges ?payload= and delegates).
type Handlers struct {
	cfg Config

	xiaozhiSrv *xiaozhi.Server // populated by mountXiaozhi
}

// NewHandlers returns a Handlers ready to register routes. It does not
// open any network listener or external connection on its own.
func NewHandlers(cfg Config) *Handlers {
	return &Handlers{cfg: cfg}
}

// XiaozhiServer returns the xiaozhi.Server constructed during Register,
// or nil if the xiaozhi transport was disabled or failed to mount.
// cmd/voice consults this for cross-transport diagnostics.
func (h *Handlers) XiaozhiServer() *xiaozhi.Server { return h.xiaozhiSrv }
