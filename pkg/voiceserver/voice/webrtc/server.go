// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package webrtc

// server.go: HTTP signaling for the WebRTC adapter. The browser POSTs an
// SDP offer; we reply with an SDP answer carrying every gathered ICE
// candidate. No WebSocket, no trickle ICE — one round-trip and the call is
// up. This is the simplest possible signaling that still feels production-
// quality (CORS handled, JSON validated, errors typed).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	voicegateway "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/recorder"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/vad"
	pionwebrtc "github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

// ServerConfig wires the adapter to ASR/TTS and the dialog plane.
type ServerConfig struct {
	// PersisterFactory, when non-nil, mints a per-call SessionPersister
	// at the start of every accepted call. Lifecycle hooks fire on the
	// returned persister: OnAccept after ICE / DTLS comes up,
	// OnASRFinal / OnTurn during the dialog, OnTerminate on teardown.
	PersisterFactory func(ctx context.Context, callID, fromHeader, toHeader, remoteAddr string) voicegateway.SessionPersister

	// RecorderFactory, when non-nil, mints a per-call recorder. The
	// session writes decoded caller PCM and AI TTS PCM into it through
	// the call lifetime; on teardown the recorder produces a stereo
	// WAV (L=browser/peer, R=AI) and the persister writes a
	// call_recording row. nil = recording disabled.
	RecorderFactory func(callID, codec string, sampleRate int) *recorder.Recorder

	// SessionFactory mints a per-call recognizer + (shared) TTS service.
	// Required.
	SessionFactory SessionFactory

	// DialogWSURL is the dialog-plane WebSocket endpoint reused from the
	// SIP / xiaozhi paths (pkg/voice/gateway). Required.
	DialogWSURL string

	// Engine configures the pion API. STUN/TURN servers, NAT 1:1, UDP
	// mux. Sensible defaults are applied when zero-valued (single
	// public STUN server on stun.l.google.com:19302).
	Engine EngineConfig

	// ICEServers populated into every PeerConnection. Convenience wrapper
	// for Engine.ICEServers — set whichever you prefer.
	ICEServers []pionwebrtc.ICEServer

	// CallIDPrefix prepended to auto-generated call IDs ("wrtc"). Empty
	// → "wrtc".
	CallIDPrefix string

	// AllowedOrigins, if non-empty, restricts the CORS Origin header to
	// the listed values. Empty list = reflect any Origin (dev mode).
	AllowedOrigins []string

	// OnSessionStart / OnSessionEnd are persistence hooks parallel to
	// the xiaozhi adapter. Optional.
	OnSessionStart func(ctx context.Context, callID, peerInfo string)
	OnSessionEnd   func(ctx context.Context, callID, reason string)

	// BargeInFactory, when non-nil, returns a per-call VAD detector
	// that is handed to the dialog gateway so user speech during TTS
	// playback interrupts the AI. Nil = barge-in disabled for every
	// call on this server. The factory pattern (rather than a shared
	// detector) means each call gets its own noise-floor state — two
	// concurrent WebRTC calls on the same server never leak noise
	// samples into each other.
	BargeInFactory func() *vad.Detector

	// DenoiserFactory, when non-nil, returns a per-call noise
	// suppressor (typically rnnoise.New()) wired into the ASR feed
	// path. Same per-call lifetime / no-cross-talk reasoning as
	// BargeInFactory. The session calls Close() on the denoiser when
	// the call ends, so factories that allocate native state (rnnoise
	// CGO) clean up correctly. Nil = denoising disabled.
	//
	// Caveat: rnnoise expects 48 kHz audio. The WebRTC bridge
	// collapses Opus to 16 kHz mono before ASR sees it; the model
	// still produces output at 16 kHz but with reduced accuracy
	// (frames are 30 ms instead of the trained 10 ms). For best
	// quality the dialog plane could deploy server-side rnnoise on
	// the 48 kHz Opus PCM directly — that's a future hook.
	DenoiserFactory func() Denoiser

	// ConfigureClient is an optional hook invoked just before each
	// per-call gateway.NewClient. The cmd layer uses it to inject
	// process-wide settings (dialog-plane reconnect attempts, hold
	// prompts) without ServerConfig having to grow a field for every
	// new gateway knob. nil = no extra configuration.
	ConfigureClient func(*voicegateway.ClientConfig)

	// Logger optional.
	Logger *zap.Logger
}

// Denoiser is the minimal interface this package consumes from a
// noise-suppression backend. It mirrors asr.Denoiser plus a Close
// hook so the session can release native resources at teardown.
// Defined here (rather than imported from pkg/audio/rnnoise) so this
// package never has to know which DSP library produced the
// implementation — operators can plug in WebRTC AEC3, custom DSP,
// etc. as long as the type satisfies this two-method contract.
type Denoiser interface {
	Process(pcm []byte) []byte
	Close()
}

// bargeInFromFactory is a small nil-safe wrapper around a
// ServerConfig.BargeInFactory. Keeping it as a tiny helper rather than
// inlining the nil-check at every call site means the "barge-in
// disabled" path is a single AST node everywhere, so turning the
// feature off at config time is obviously zero-cost in code review.
func bargeInFromFactory(fn func() *vad.Detector) *vad.Detector {
	if fn == nil {
		return nil
	}
	return fn()
}

// Server is the public HTTP entry point.
type Server struct {
	cfg ServerConfig
	api *pionwebrtc.API

	mu       sync.Mutex
	sessions map[string]*session
}

// NewServer validates cfg, builds the shared pion API, and returns an
// HTTP-mountable server. The returned *Server is goroutine-safe.
func NewServer(cfg ServerConfig) (*Server, error) {
	if cfg.SessionFactory == nil {
		return nil, errors.New("webrtc: nil SessionFactory")
	}
	if strings.TrimSpace(cfg.DialogWSURL) == "" {
		return nil, errors.New("webrtc: empty DialogWSURL")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.CallIDPrefix == "" {
		cfg.CallIDPrefix = "wrtc"
	}
	// Merge the convenience ICEServers field into Engine.
	if len(cfg.ICEServers) > 0 {
		cfg.Engine.ICEServers = append(cfg.Engine.ICEServers, cfg.ICEServers...)
	}
	if len(cfg.Engine.ICEServers) == 0 {
		cfg.Engine.ICEServers = []pionwebrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		}
	}
	api, err := BuildAPI(cfg.Engine)
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:      cfg,
		api:      api,
		sessions: make(map[string]*session),
	}, nil
}

// HandleOffer is the offer-answer signaling endpoint. Mount it on the
// path the client posts to, e.g. /webrtc/v1/offer.
//
//	POST /webrtc/v1/offer
//	  Content-Type: application/json
//	  Body: {"sdp":"...", "type":"offer"}
//
//	200 OK
//	  Content-Type: application/json
//	  Body: {"sdp":"...", "type":"answer", "call_id":"wrtc-..."}
func (s *Server) HandleOffer(w http.ResponseWriter, r *http.Request) {
	s.applyCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	var offer SDPMessage
	if err := json.Unmarshal(body, &offer); err != nil {
		http.Error(w, "json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(offer.SDP) == "" || offer.Type != "offer" {
		http.Error(w, "expected {sdp,type:offer}", http.StatusBadRequest)
		return
	}

	// Per-request context with a generous timeout: ICE gathering can
	// take ~1 s on first connect, plus the offer-side SDP set.
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	sess, answer, err := newSession(ctx, s.cfg, s.api, offer)
	if err != nil {
		s.cfg.Logger.Warn("webrtc: handshake failed", zap.Error(err))
		http.Error(w, "handshake: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Capture peer info now (UA, addr) so we can log it in OnSessionStart.
	sess.clientMeta = clientInfo(r)

	s.mu.Lock()
	s.sessions[sess.callID] = sess
	s.mu.Unlock()
	go s.cleanupOnClose(sess)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(answer); err != nil {
		s.cfg.Logger.Warn("webrtc: write answer", zap.Error(err))
	}
}

// HandleHangup terminates a call by ID. Optional endpoint — the dialog
// app or the client can also tear down naturally (BYE / page close →
// ICE failed → teardown).
//
//	POST /webrtc/v1/hangup?call_id=wrtc-...
func (s *Server) HandleHangup(w http.ResponseWriter, r *http.Request) {
	s.applyCORS(w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := strings.TrimSpace(r.URL.Query().Get("call_id"))
	if id == "" {
		http.Error(w, "missing call_id", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	sess := s.sessions[id]
	delete(s.sessions, id)
	s.mu.Unlock()
	if sess == nil {
		http.Error(w, "no such call", http.StatusNotFound)
		return
	}
	sess.teardown("client-hangup")
	w.WriteHeader(http.StatusNoContent)
}

// cleanupOnClose blocks until the PeerConnection state machine reports
// Closed/Failed (which fires teardown which closes session.done), then
// deletes the registry entry so it doesn't leak across long uptimes.
func (s *Server) cleanupOnClose(sess *session) {
	<-sess.done
	s.mu.Lock()
	delete(s.sessions, sess.callID)
	s.mu.Unlock()
}

// applyCORS sets the headers needed for browser-to-server signaling.
// Allowed-Origin handling is intentionally permissive in dev (empty list
// reflects any Origin) and strict in production (only the listed values).
func (s *Server) applyCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}
	allow := false
	if len(s.cfg.AllowedOrigins) == 0 {
		allow = true
	} else {
		for _, a := range s.cfg.AllowedOrigins {
			if a == origin || a == "*" {
				allow = true
				break
			}
		}
	}
	if !allow {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Vary", "Origin")
}

// clientInfo extracts a small printable summary of the requester for log
// / persistence purposes. Falls through to RemoteAddr when behind no
// proxy.
func clientInfo(r *http.Request) string {
	xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	ua := strings.TrimSpace(r.Header.Get("User-Agent"))
	addr := r.RemoteAddr
	if xff != "" {
		addr = xff
	}
	if ua == "" {
		return addr
	}
	return fmt.Sprintf("%s ua=%q", addr, ua)
}
