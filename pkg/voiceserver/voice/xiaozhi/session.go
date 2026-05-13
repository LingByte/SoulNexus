// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package xiaozhi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/media/encoder"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/asr"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/recorder"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/vad"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// SessionFactory builds the per-call ASR + TTS services. It is invoked once
// per inbound hardware WS connection so each session has its own recognizer
// state (vendors like QCloud have stateful sessions per connection).
//
// The TTS Service is allowed to be a process-shared cached service — it has
// no per-connection state beyond what the Pipeline holds. Returning the same
// pointer across sessions is fine and recommended.
type SessionFactory interface {
	// NewASR returns a fresh recognizer + the rate it expects PCM16 in.
	NewASR(ctx context.Context, callID string) (svc recognizer.TranscribeService, sampleRate int, err error)
	// TTS returns the (possibly shared) streaming TTS service that produces
	// PCM16 at sampleRate. Channels is always 1.
	TTS(ctx context.Context, callID string) (svc tts.Service, sampleRate int, err error)
}

// ServerConfig wires the hardware adapter to the rest of the system. Only
// SessionFactory and DialogWSURL are required; the rest are optional.
type ServerConfig struct {
	// PersisterFactory, when non-nil, mints a per-call SessionPersister
	// at the start of every accepted session. Lifecycle hooks fire on
	// the returned persister: OnAccept after the welcome handshake,
	// OnASRFinal / OnTurn during the dialog, OnTerminate on teardown.
	// Typical implementations write into voiceserver.db so xiaozhi
	// calls show up in the same call log as SIP.
	PersisterFactory func(ctx context.Context, callID, fromHeader, toHeader, remoteAddr string) gateway.SessionPersister

	// RecorderFactory, when non-nil, mints a per-call recorder. The
	// session writes decoded caller PCM and AI TTS PCM into it through
	// the call lifetime; on teardown the recorder produces a stereo
	// WAV (L=device, R=AI) and the persister writes a call_recording
	// row. nil = recording disabled (no WAV, no row).
	RecorderFactory func(callID, codec string, sampleRate int) *recorder.Recorder

	// SessionFactory builds ASR / TTS for each session. Required.
	SessionFactory SessionFactory

	// DialogWSURL is the dialog-plane WebSocket the session dials out to.
	// Same protocol the SIP path uses (pkg/voice/gateway). Required.
	DialogWSURL string

	// CallIDPrefix is prepended to auto-generated call IDs ("xz-..."). Empty =
	// "xz". Useful when running multiple hardware servers behind one dialog.
	CallIDPrefix string

	// OnSessionStart is called once a hardware session is fully wired
	// (hello received, ASR / TTS up, dialog WS connected). Optional —
	// typical use is to create a persistence row.
	OnSessionStart func(ctx context.Context, callID, deviceID string)

	// OnSessionEnd is called on teardown for any reason. Optional.
	OnSessionEnd func(ctx context.Context, callID, reason string)

	// BargeInFactory, when non-nil, returns a per-call VAD detector
	// that is handed to the dialog gateway so user speech during TTS
	// playback interrupts the AI. Nil = barge-in disabled for every
	// session on this server. The factory pattern (rather than a
	// shared detector) means each session keeps its own noise-floor
	// state — two concurrent device sessions never leak noise into
	// each other. See pkg/voice/vad for defaults.
	BargeInFactory func() *vad.Detector

	// ConfigureClient is an optional hook invoked just before each
	// per-session gateway.NewClient. The cmd layer uses it to inject
	// process-wide settings (dialog-plane reconnect attempts, hold
	// prompts) without ServerConfig having to grow a field for every
	// new gateway knob. nil = no extra configuration.
	ConfigureClient func(*gateway.ClientConfig)

	// Logger optional.
	Logger *zap.Logger
}

// bargeInFromFactory is a nil-safe wrapper around a
// ServerConfig.BargeInFactory; mirrors the helper in the webrtc
// package so both transports have identical wiring at the call site.
func bargeInFromFactory(fn func() *vad.Detector) *vad.Detector {
	if fn == nil {
		return nil
	}
	return fn()
}

// Server is the public entry point. Mount Handle on an HTTP route to accept
// xiaozhi-esp32 hardware connections.
type Server struct {
	cfg ServerConfig
	up  websocket.Upgrader
}

// NewServer validates cfg and returns an HTTP-ready Server.
func NewServer(cfg ServerConfig) (*Server, error) {
	if cfg.SessionFactory == nil {
		return nil, errors.New("xiaozhi: nil SessionFactory")
	}
	if strings.TrimSpace(cfg.DialogWSURL) == "" {
		return nil, errors.New("xiaozhi: empty DialogWSURL")
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.CallIDPrefix == "" {
		cfg.CallIDPrefix = "xz"
	}
	return &Server{
		cfg: cfg,
		up: websocket.Upgrader{
			// Hardware devices generally have no Origin header; accept all.
			CheckOrigin:     func(_ *http.Request) bool { return true },
			ReadBufferSize:  4 * 1024,
			WriteBufferSize: 16 * 1024,
		},
	}, nil
}

// Handle implements http.Handler. Mount on the path the firmware connects to
// (commonly "/xiaozhi/v1/" or "/ws/voice").
func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := s.up.Upgrade(w, r, nil)
	if err != nil {
		s.cfg.Logger.Warn("xiaozhi: upgrade failed", zap.Error(err))
		return
	}
	deviceID := strings.TrimSpace(r.Header.Get("Device-Id"))
	if deviceID == "" {
		// Some firmware uses lowercase / underscore variants.
		deviceID = strings.TrimSpace(r.Header.Get("device-id"))
	}
	macAddr := strings.TrimSpace(r.Header.Get("X-Mac-Address"))

	callID := fmt.Sprintf("%s-%d", s.cfg.CallIDPrefix, time.Now().UnixNano())
	sess := newSession(s.cfg, conn, callID, deviceID, macAddr)
	sess.run(r.Context())
}

// ---------- Session ---------------------------------------------------------

// session is one hardware-side WebSocket. It owns:
//
//   - the device WS conn
//   - an opus decoder (device → PCM)  / opus encoder (PCM → device)
//   - a voice.Attached holding ASR + TTS pipelines
//   - a gateway.Client dialled out to the dialog-plane WS
//
// All state is per-session; nothing is shared across connections except the
// (optional) cached TTS service supplied by SessionFactory.
type session struct {
	cfg       ServerConfig
	conn      *websocket.Conn
	callID    string
	sessionID string
	deviceID  string
	macAddr   string
	log       *zap.Logger

	// Negotiated audio profile (filled at hello-time).
	inFormat   string
	inSR       int
	inFrameMs  int
	outFormat  string
	outSR      int
	outFrameMs int

	// Codec workers. Opus dec/enc are nil for PCM mode.
	opusDec media.EncoderFunc
	opusEnc media.EncoderFunc

	// Voice pipelines.
	asrPipe *asr.Pipeline
	ttsPipe *tts.Pipeline
	att     *voice.Attached

	// Dialog-plane bridge.
	gw *gateway.Client

	// Listen-state machine. Only forward audio to ASR while listening; the
	// firmware sends frames continuously but expects the server to ignore
	// them outside listen-start/listen-stop.
	listening atomic.Bool

	// ttsActive tracks whether we are currently inside a tts:start ... tts:stop
	// envelope. Used to emit a final tts:stop on barge-in / teardown without
	// double-firing it from OnTurn.
	ttsActive atomic.Bool

	// persister is built from cfg.PersisterFactory at hello time and
	// used through the call lifecycle. nil = persistence disabled.
	persister gateway.SessionPersister

	// rec is built from cfg.RecorderFactory at hello time. nil =
	// recording disabled. Captures both directions of decoded PCM and
	// produces a stereo WAV at teardown.
	rec *recorder.Recorder

	// WS write serialization. gorilla/websocket forbids concurrent writes.
	writeMu sync.Mutex

	closed atomic.Bool
}

func newSession(cfg ServerConfig, conn *websocket.Conn, callID, deviceID, macAddr string) *session {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &session{
		cfg:       cfg,
		conn:      conn,
		callID:    callID,
		sessionID: callID,
		deviceID:  deviceID,
		macAddr:   macAddr,
		log: logger.With(
			zap.String("call_id", callID),
			zap.String("device_id", deviceID),
		),
	}
}

func (s *session) run(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	defer s.teardown("loop-exit")

	// We don't send `connected` until the hello handshake completes —
	// xiaozhi-esp32 firmware blocks on its own hello reply, not a generic
	// ack.

	for {
		if s.closed.Load() {
			return
		}
		_ = s.conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		mt, raw, err := s.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
				websocket.CloseNoStatusReceived) {
				s.log.Warn("xiaozhi: read end", zap.Error(err))
			}
			return
		}
		switch mt {
		case websocket.TextMessage:
			s.handleText(ctx, raw)
		case websocket.BinaryMessage:
			s.handleAudio(ctx, raw)
		case websocket.PingMessage:
			// gorilla replies with pong automatically.
		default:
			// Ignore other frame types.
		}
	}
}

// handleText dispatches a JSON control frame.
func (s *session) handleText(ctx context.Context, raw []byte) {
	t, err := ParseTextFrame(raw)
	if err != nil {
		s.log.Warn("xiaozhi: bad text frame", zap.Error(err))
		return
	}
	switch t {
	case MsgHello:
		s.handleHello(ctx, raw)
	case MsgListen:
		s.handleListen(raw)
	case MsgAbort:
		s.handleAbort()
	case MsgPing:
		s.writeText(MakePongReply(s.sessionID))
	default:
		s.log.Debug("xiaozhi: unknown message type", zap.String("type", t))
	}
}

// handleHello completes the audio negotiation and brings up ASR / TTS / the
// dialog-plane WS. It is the only handshake gate — until hello succeeds, all
// subsequent frames are dropped.
func (s *session) handleHello(ctx context.Context, raw []byte) {
	var msg HelloMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		s.log.Warn("xiaozhi: hello parse", zap.Error(err))
		s.writeText(MakeError("bad hello", true))
		return
	}
	ap := DefaultHelloAudio()
	if msg.AudioParams != nil {
		ap = *msg.AudioParams
	}
	MergeHelloAudio(&ap)
	s.inFormat = ap.Format
	s.inSR = ap.SampleRate
	s.inFrameMs = ap.FrameDuration
	// Echo back the same profile for outbound TTS so the firmware can decode
	// without re-negotiation.
	s.outFormat = ap.Format
	s.outSR = ap.SampleRate
	s.outFrameMs = ap.FrameDuration

	// Build codecs.
	if s.inFormat == AudioFormatOpus {
		dec, err := encoder.CreateDecode(
			media.CodecConfig{Codec: "opus", SampleRate: s.inSR, Channels: 1, FrameDuration: fmt.Sprintf("%dms", s.inFrameMs)},
			media.CodecConfig{Codec: "pcm", SampleRate: s.inSR, Channels: 1},
		)
		if err != nil {
			s.log.Error("xiaozhi: opus decoder", zap.Error(err))
			s.writeText(MakeError("opus decoder unavailable", true))
			return
		}
		s.opusDec = dec
	}
	if s.outFormat == AudioFormatOpus {
		enc, err := encoder.CreateEncode(
			media.CodecConfig{Codec: "opus", SampleRate: s.outSR, Channels: 1, FrameDuration: fmt.Sprintf("%dms", s.outFrameMs)},
			media.CodecConfig{Codec: "pcm", SampleRate: s.outSR, Channels: 1},
		)
		if err != nil {
			s.log.Error("xiaozhi: opus encoder", zap.Error(err))
			s.writeText(MakeError("opus encoder unavailable", true))
			return
		}
		s.opusEnc = enc
	}

	// Build ASR + TTS pipelines.
	asrSvc, asrSR, err := s.cfg.SessionFactory.NewASR(ctx, s.callID)
	if err != nil || asrSvc == nil {
		s.log.Error("xiaozhi: asr factory", zap.Error(err))
		s.writeText(MakeError("asr unavailable", true))
		return
	}
	asrPipe, err := asr.New(asr.Options{
		ASR:             asrSvc,
		InputSampleRate: s.inSR,
		SampleRate:      asrSR,
		Channels:        1,
		DialogID:        s.callID,
		Logger:          s.log,
	})
	if err != nil {
		s.log.Error("xiaozhi: asr pipeline", zap.Error(err))
		s.writeText(MakeError("asr init failed", true))
		return
	}
	s.asrPipe = asrPipe

	ttsSvc, ttsSR, err := s.cfg.SessionFactory.TTS(ctx, s.callID)
	if err != nil || ttsSvc == nil {
		s.log.Error("xiaozhi: tts factory", zap.Error(err))
		s.writeText(MakeError("tts unavailable", true))
		return
	}
	ttsPipe, err := tts.New(tts.Config{
		Service:          ttsSvc,
		InputSampleRate:  ttsSR,
		OutputSampleRate: s.outSR,
		Channels:         1,
		FrameDuration:    time.Duration(s.outFrameMs) * time.Millisecond,
		PaceRealtime:     true,
		Sink:             s.ttsSink,
		Logger:           s.log,
	})
	if err != nil {
		s.log.Error("xiaozhi: tts pipeline", zap.Error(err))
		s.writeText(MakeError("tts init failed", true))
		return
	}
	ttsPipe.Start(ctx)
	s.ttsPipe = ttsPipe

	// Wrap into a voice.Attached so we can reuse gateway.Client unchanged.
	s.att = &voice.Attached{ASR: s.asrPipe, TTS: s.ttsPipe}

	// Dial dialog plane.
	gwCfg := gateway.ClientConfig{
		URL:      s.cfg.DialogWSURL,
		Attached: s.att,
		CallID:   s.callID,
		BargeIn:  bargeInFromFactory(s.cfg.BargeInFactory),
		OnHangup: func(reason string) {
			s.log.Info("xiaozhi: dialog hangup", zap.String("reason", reason))
			// Record the dialog-side hangup as its own event so the
			// timeline distinguishes "the dialog plane asked us to end
			// the call" from "the device dropped the WS". The
			// terminating call.terminated event fires from teardown.
			if s.persister != nil {
				detail, _ := json.Marshal(map[string]any{"reason": reason})
				s.persister.OnEvent(context.Background(), "dialog.hangup", "info", detail)
			}
			s.teardown("dialog-hangup:" + reason)
		},
		// Push every final ASR transcript to the device so the firmware UI
		// (or browser overlay) can render it. xiaozhi-esp32 firmware
		// expects {"type":"stt","text":...}; missing this leaves the
		// caption area blank even when LLM/TTS work fine.
		OnASRFinal: func(text string) {
			s.writeText(MakeSTTReply(s.sessionID, text))
			if s.persister != nil {
				s.persister.OnASRFinal(context.Background(), text)
			}
		},
		// Wrap each speak with the xiaozhi tts:start / tts:stop envelope.
		// The firmware uses tts:start to unmute its speaker and tts:stop
		// to flush its decoder; raw binary frames without the envelope are
		// dropped on most builds.
		OnTTSStart: func(_ string, _ string) {
			s.ttsActive.Store(true)
			s.writeText(MakeTTSStateReply(s.sessionID, "start", s.outFormat))
		},
		OnTurn: func(t gateway.TurnEvent) {
			if s.ttsActive.CompareAndSwap(true, false) {
				s.writeText(MakeTTSStateReply(s.sessionID, "stop", s.outFormat))
			}
			if s.persister != nil {
				s.persister.OnTurn(context.Background(), t)
			}
			s.log.Info("xiaozhi: turn",
				zap.String("utter", t.UtteranceID),
				zap.Int("dur_ms", t.DurationMs),
				zap.Bool("ok", t.OK))
		},
		Logger: s.log,
	}
	if s.cfg.ConfigureClient != nil {
		s.cfg.ConfigureClient(&gwCfg)
	}
	gw, err := gateway.NewClient(gwCfg)
	if err != nil {
		s.log.Error("xiaozhi: gateway client", zap.Error(err))
		s.writeText(MakeError("dialog ws init failed", true))
		return
	}
	if err := gw.Start(ctx, gateway.StartMeta{
		From:  s.deviceID,
		To:    "xiaozhi",
		Codec: s.inFormat,
		PCMHz: s.inSR,
	}); err != nil {
		s.log.Error("xiaozhi: gateway dial", zap.Error(err))
		s.writeText(MakeError("dialog ws unreachable", true))
		return
	}
	s.gw = gw

	// Send welcome reply only after everything is ready, so the firmware
	// doesn't immediately start streaming audio into a half-built pipeline.
	s.writeText(MakeWelcomeReply(s.sessionID, AudioParams{
		Format:        s.outFormat,
		SampleRate:    s.outSR,
		Channels:      1,
		FrameDuration: s.outFrameMs,
		BitDepth:      16,
	}))
	if s.cfg.OnSessionStart != nil {
		s.cfg.OnSessionStart(ctx, s.callID, s.deviceID)
	}
	// Build the per-call recorder before the persister so we can stamp
	// recording metadata into the persister's call_recording row at
	// teardown time.
	if s.cfg.RecorderFactory != nil {
		s.rec = s.cfg.RecorderFactory(s.callID, s.inFormat, s.inSR)
	}
	if s.cfg.PersisterFactory != nil {
		remoteAddr := ""
		if s.conn != nil && s.conn.RemoteAddr() != nil {
			remoteAddr = s.conn.RemoteAddr().String()
		}
		fromHdr := s.deviceID
		if fromHdr == "" {
			fromHdr = remoteAddr
		}
		s.persister = s.cfg.PersisterFactory(ctx, s.callID, fromHdr, "xiaozhi", remoteAddr)
		if s.persister != nil {
			// Stamp the negotiated codec / sample rate on the call row.
			s.persister.OnAccept(ctx, s.inFormat, s.inSR, remoteAddr)
			// Record the hello payload as a structured event so the
			// per-call timeline shows what the device asked for.
			detail, _ := json.Marshal(map[string]any{
				"in_format":      s.inFormat,
				"in_sample_rate": s.inSR,
				"out_format":     s.outFormat,
				"frame_ms":       s.inFrameMs,
				"device_id":      s.deviceID,
			})
			s.persister.OnEvent(ctx, "xiaozhi.hello", "info", detail)
		}
	}
	log.Printf("[xiaozhi] call=%s device=%s connected: in=%s/%dHz out=%s/%dHz dialog=%s",
		s.callID, s.deviceID, s.inFormat, s.inSR, s.outFormat, s.outSR, gateway.RedactDialogDialURL(s.cfg.DialogWSURL))
}

// handleListen flips the input gate: while !listening the inbound audio
// frames are decoded but dropped before reaching ASR. This mirrors the
// firmware's expectation that the server only "hears" between start/stop.
func (s *session) handleListen(raw []byte) {
	var lm ListenMessage
	if err := json.Unmarshal(raw, &lm); err != nil {
		return
	}
	state := strings.ToLower(strings.TrimSpace(lm.State))
	switch state {
	case ListenStart:
		s.listening.Store(true)
	case ListenStop:
		s.listening.Store(false)
		// Flush whatever the recognizer has buffered so the user gets a
		// final transcript even without a sentence-end on the wire.
		if s.asrPipe != nil {
			_ = s.asrPipe.Flush()
		}
	}
	if s.persister != nil && state != "" {
		detail, _ := json.Marshal(map[string]any{"state": state, "mode": lm.Mode})
		s.persister.OnEvent(context.Background(), "xiaozhi.listen", "info", detail)
	}
}

// handleAbort interrupts ongoing TTS playback (barge-in initiated by the
// firmware, e.g. from a touch button or wake-word detection on-device).
// We mirror the SoulNexus reference: cancel the in-flight Speak, drain the
// dialog-side queue (handled by gateway.Client on the next CmdTTSInterrupt
// path — here we already cancel locally), then send tts:stop so the device
// flushes its audio buffer, finally an abort:confirmed acknowledgement.
func (s *session) handleAbort() {
	if s.ttsPipe != nil {
		s.ttsPipe.Interrupt()
	}
	if s.ttsActive.CompareAndSwap(true, false) {
		s.writeText(MakeTTSStateReply(s.sessionID, "stop", s.outFormat))
	}
	s.writeText(MakeAbortConfirm(s.sessionID))
	if s.persister != nil {
		s.persister.OnEvent(context.Background(), "xiaozhi.abort", "info", nil)
	}
}

// handleAudio decodes one inbound binary frame and feeds the PCM into ASR.
// Frames received before hello (no decoder, no pipeline) or while !listening
// are silently dropped. Captured PCM is also pushed into the recorder if
// one was built; we record only while listening so background noise
// between turns is not captured.
func (s *session) handleAudio(ctx context.Context, frame []byte) {
	if s.asrPipe == nil || !s.listening.Load() {
		return
	}
	var pcm []byte
	if s.opusDec != nil {
		out, err := s.opusDec(&media.AudioPacket{Payload: frame})
		if err != nil || len(out) == 0 {
			return
		}
		ap, _ := out[0].(*media.AudioPacket)
		if ap == nil || len(ap.Payload) == 0 {
			return
		}
		pcm = ap.Payload
	} else {
		// PCM mode: device sends raw PCM16LE. Accept as-is.
		pcm = frame
	}
	if err := s.asrPipe.ProcessPCM(ctx, pcm); err != nil {
		s.log.Debug("xiaozhi: asr feed", zap.Error(err))
	}
	if s.rec != nil {
		s.rec.WriteCaller(pcm)
	}
}

// ttsSink is the sink the TTS Pipeline calls for each paced PCM16 frame.
// We encode to opus (when negotiated) and write a binary WS frame. The
// Pipeline already paces realtime, so we don't need extra flow control.
//
// The PCM is also pushed into the recorder before encoding so the AI
// channel of the stereo recording is captured at the bridge sample rate
// (no opus round-trip artefacts in the WAV).
func (s *session) ttsSink(pcm []byte) error {
	if s.closed.Load() {
		return errors.New("xiaozhi: session closed")
	}
	if s.rec != nil {
		s.rec.WriteAI(pcm)
	}
	var payload []byte
	if s.opusEnc != nil {
		out, err := s.opusEnc(&media.AudioPacket{Payload: pcm})
		if err != nil || len(out) == 0 {
			return err
		}
		ap, _ := out[0].(*media.AudioPacket)
		if ap == nil || len(ap.Payload) == 0 {
			return nil
		}
		payload = ap.Payload
	} else {
		payload = pcm
	}
	return s.writeBinary(payload)
}

// teardown closes everything in reverse build order. Idempotent.
func (s *session) teardown(reason string) {
	if !s.closed.CompareAndSwap(false, true) {
		return
	}
	if s.gw != nil {
		s.gw.Close(reason)
	}
	if s.att != nil {
		s.att.Close()
	}
	if s.conn != nil {
		_ = s.conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason),
			time.Now().Add(500*time.Millisecond))
		_ = s.conn.Close()
	}
	// Flush the recorder BEFORE OnTerminate so the call_recording row
	// shares the call's lifetime metadata. Persister sees OnRecording
	// then OnTerminate in that order — matches what the SIP path does.
	if s.rec != nil {
		if info, ok := s.rec.Flush(context.Background()); ok && s.persister != nil {
			s.persister.OnRecording(context.Background(), info)
		}
	}
	if s.persister != nil {
		s.persister.OnTerminate(context.Background(), reason)
	}
	if s.cfg.OnSessionEnd != nil {
		s.cfg.OnSessionEnd(context.Background(), s.callID, reason)
	}
	log.Printf("[xiaozhi] call=%s closed: %s", s.callID, reason)
}

// writeText / writeBinary serialise WS writes (gorilla forbids concurrent
// writes on the same conn).
func (s *session) writeText(payload []byte) {
	if s.closed.Load() || len(payload) == 0 {
		return
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := s.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		s.log.Debug("xiaozhi: write text", zap.Error(err))
	}
}

func (s *session) writeBinary(payload []byte) error {
	if s.closed.Load() || len(payload) == 0 {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return s.conn.WriteMessage(websocket.BinaryMessage, payload)
}
