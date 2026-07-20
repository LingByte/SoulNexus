package xiaozhi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/lingllm/media"
	"github.com/LingByte/lingllm/media/encoder"
	"github.com/LingByte/lingllm/protocol/voice/dialog"
	"github.com/LingByte/lingllm/protocol/voice/gateway"
	"github.com/LingByte/lingllm/protocol/voice/transport"
	"github.com/LingByte/lingllm/realtime"
	"github.com/gorilla/websocket"
)

// MessageHandler is a custom message handler for extensibility.
type MessageHandler func(ctx context.Context, session *wsSession, raw []byte) error

type wsSession struct {
	cfg       ServerConfig
	conn      *websocket.Conn
	callID    string
	sessionID string
	deviceID  string

	inFormat, outFormat string
	inSR, outSR         int
	inFrameMs           int
	ttsWireFrameMs      int

	opusDec media.EncoderFunc
	opusEnc media.EncoderFunc

	mode string

	voiceSess *dialog.Session
	gw        *gateway.Client

	rtAgent         realtime.Agent
	rtOut           *realtimeOutPacer
	rtInSR, rtOutSR int

	listening   atomic.Bool
	ttsActive   atomic.Bool
	rtReplyBusy atomic.Bool
	turnLLMText string

	writeMu sync.Mutex
	closed  atomic.Bool

	// Custom message handlers for extensibility
	customHandlers map[string]MessageHandler

	// Audio processing optimization
	audioProcessChan chan []byte
}

func newSession(cfg ServerConfig, conn *websocket.Conn, callID, deviceID string) *wsSession {
	return &wsSession{
		cfg:            cfg,
		conn:           conn,
		callID:         callID,
		sessionID:      callID,
		deviceID:       deviceID,
		customHandlers: make(map[string]MessageHandler),
	}
}

// RegisterMessageHandler registers a custom message handler for a message type.
// This allows third-party extensions to handle custom message types.
func (s *wsSession) RegisterMessageHandler(msgType string, handler MessageHandler) {
	if s.customHandlers == nil {
		s.customHandlers = make(map[string]MessageHandler)
	}
	s.customHandlers[msgType] = handler
}

func (s *wsSession) run(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	defer s.teardown("loop-exit")

	// Initialize audio processing channel
	s.audioProcessChan = make(chan []byte, 32) // Buffer up to 32 frames
	defer close(s.audioProcessChan)

	// Start audio processing goroutine
	go s.audioProcessWorker(ctx)

	for {
		if s.closed.Load() {
			return
		}
		_ = s.conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		mt, raw, err := s.conn.ReadMessage()
		if err != nil {
			return
		}
		switch mt {
		case websocket.TextMessage:
			s.handleText(ctx, raw)
		case websocket.BinaryMessage:
			// Non-blocking send to audio processing channel
			select {
			case s.audioProcessChan <- raw:
			default:
				// Channel full, drop frame to avoid blocking
			}
		}
	}
}

func (s *wsSession) handleText(ctx context.Context, raw []byte) {
	t, err := ParseTextFrame(raw)
	if err != nil {
		return
	}

	// Check custom handlers first
	if handler, ok := s.customHandlers[t]; ok {
		if err := handler(ctx, s, raw); err != nil {
			s.writeText(MakeError(err.Error(), false))
		}
		return
	}

	// Built-in message handlers
	switch t {
	case MsgHello:
		s.handleHello(ctx, raw)
	case MsgListen:
		s.handleListen(raw)
	case MsgAbort:
		s.handleAbort()
	case MsgPing:
		s.writeText(MakePongReply(s.sessionID))
	}
}

func (s *wsSession) handleHello(ctx context.Context, raw []byte) {
	var msg HelloMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
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
	s.outFormat = ap.Format
	s.outSR = ap.SampleRate
	s.ttsWireFrameMs = ap.FrameDuration
	if s.outFormat == AudioFormatPCM && s.ttsWireFrameMs >= 40 {
		s.ttsWireFrameMs = 20
	}

	if s.inFormat == AudioFormatOpus {
		dec, err := encoder.CreateDecode(
			media.CodecConfig{
				Codec:         "opus",
				SampleRate:    s.inSR,
				Channels:      1,
				FrameDuration: fmt.Sprintf("%dms", s.inFrameMs),
			},
			media.CodecConfig{Codec: "pcm", SampleRate: s.inSR, Channels: 1},
		)
		if err != nil {
			s.writeText(MakeError("opus decoder unavailable", true))
			return
		}
		s.opusDec = dec
	}
	if s.outFormat == AudioFormatOpus {
		enc, err := encoder.CreateEncode(
			media.CodecConfig{
				Codec:         "opus",
				SampleRate:    s.outSR,
				Channels:      1,
				FrameDuration: fmt.Sprintf("%dms", s.ttsWireFrameMs),
			},
			media.CodecConfig{Codec: "pcm", SampleRate: s.outSR, Channels: 1},
		)
		if err != nil {
			s.writeText(MakeError("opus encoder unavailable", true))
			return
		}
		s.opusEnc = enc
	}

	mode := normalizeMode(msg.Mode)
	if mode == "" {
		mode = ModePipeline
	}
	s.mode = mode

	if mode == ModeRealtime {
		s.handleHelloRealtime(ctx)
		return
	}

	s.handleHelloPipeline(ctx)
}

func (s *wsSession) handleHelloPipeline(ctx context.Context) {
	meta := dialog.StartMeta{
		From:  s.deviceID,
		To:    "xiaozhi",
		Codec: s.inFormat,
		PCMHz: s.inSR,
	}

	gwCfg := gateway.ClientConfig{
		OnASRFinal: func(text string) {
			s.writeText(MakeSTTReply(s.sessionID, text))
		},
		OnTTSStart: func(_ string, _ string) {
			if s.ttsActive.Swap(true) {
				return
			}
			s.writeText(MakeTTSStateReplyFrames(s.sessionID, "start", s.outFormat, s.ttsWireFrameMs))
		},
		OnTurn: func(t dialog.TurnEvent) {
			if t.LLMText != "" {
				s.turnLLMText += t.LLMText
			}
			if !t.MoreSpeaksQueued {
				if s.ttsActive.CompareAndSwap(true, false) {
					s.writeText(MakeTTSStateReply(s.sessionID, "stop", s.outFormat))
				}
				if full := strings.TrimSpace(s.turnLLMText); full != "" {
					s.writeText(MakeLLMReply(full))
				}
				s.turnLLMText = ""
			}
		},
		OnHangup: func(reason string) {
			s.teardown("dialog-hangup:" + reason)
		},
	}
	if s.cfg.ConfigureClient != nil {
		s.cfg.ConfigureClient(&gwCfg)
	}

	voiceSess, gw, err := transport.NewCall(ctx, transport.CallConfig{
		CallID:        s.callID,
		DialogURL:     s.cfg.DialogWSURL,
		Meta:          meta,
		Factory:       s.cfg.SessionFactory,
		OnAudioOut:    s.emitOutboundPCMFrame,
		InputCodec:    s.inFormat,
		OutputCodec:   s.outFormat,
		PCMSampleRate: s.inSR,
		Gateway:       gwCfg,
		OnHangup:      func(reason string) { s.teardown("dialog-hangup:" + reason) },
	})
	if err != nil {
		s.writeText(MakeError("voice init failed", true))
		return
	}
	if err := gw.Start(ctx); err != nil {
		s.writeText(MakeError("dialog ws unreachable", true))
		return
	}
	if err := voiceSess.Start(ctx); err != nil {
		s.writeText(MakeError("session start failed", true))
		return
	}
	s.voiceSess = voiceSess
	s.gw = gw

	s.writeText(MakeWelcomeReply(s.sessionID, AudioParams{
		Format:        s.outFormat,
		SampleRate:    s.outSR,
		Channels:      1,
		FrameDuration: s.ttsWireFrameMs,
		BitDepth:      16,
	}))
	if s.cfg.OnSessionStart != nil {
		s.cfg.OnSessionStart(ctx, s.callID, s.deviceID)
	}
}

func (s *wsSession) handleListen(raw []byte) {
	var lm ListenMessage
	if err := json.Unmarshal(raw, &lm); err != nil {
		return
	}
	switch strings.ToLower(strings.TrimSpace(lm.State)) {
	case ListenStart:
		s.listening.Store(true)
	case ListenStop:
		s.listening.Store(false)
	}
}

func (s *wsSession) handleAbort() {
	if s.rtAgent != nil {
		_ = s.rtAgent.Cancel()
	}
	if s.rtOut != nil {
		s.rtOut.interrupt(false)
	}
	if s.voiceSess != nil {
		s.voiceSess.HandleCommand(dialog.Command{Type: dialog.CmdTTSInterrupt})
	}
	if s.ttsActive.CompareAndSwap(true, false) {
		s.writeText(MakeTTSStateReply(s.sessionID, "stop", s.outFormat))
	}
	s.rtReplyBusy.Store(false)
	s.writeText(MakeAbortConfirm(s.sessionID))
}

// audioProcessWorker processes audio frames asynchronously to avoid blocking the message loop
func (s *wsSession) audioProcessWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-s.audioProcessChan:
			if !ok {
				return
			}
			s.handleAudio(ctx, frame)
		}
	}
}

func (s *wsSession) handleAudio(ctx context.Context, frame []byte) {
	if s.mode == ModeRealtime {
		s.handleAudioRealtime(ctx, frame)
		return
	}
	if s.voiceSess == nil || !s.listening.Load() {
		return
	}
	// Don't skip audio if TTS is active - buffer it for processing
	// This improves latency by not losing audio during TTS playback
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
		pcm = frame
	}
	_ = s.voiceSess.ProcessAudio(ctx, pcm)
}

func (s *wsSession) emitOutboundPCMFrame(pcm []byte) error {
	if s.closed.Load() {
		return errors.New("xiaozhi: closed")
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

func (s *wsSession) teardown(reason string) {
	if !s.closed.CompareAndSwap(false, true) {
		return
	}
	if s.rtOut != nil {
		s.rtOut.close()
		s.rtOut = nil
	}
	if s.rtAgent != nil {
		_ = s.rtAgent.Close()
		s.rtAgent = nil
	}
	if s.gw != nil {
		s.gw.Close(reason)
	}
	if s.voiceSess != nil {
		s.voiceSess.Close(reason)
	}
	if s.conn != nil {
		_ = s.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason),
			time.Now().Add(500*time.Millisecond),
		)
		_ = s.conn.Close()
	}
	if s.cfg.OnSessionEnd != nil {
		s.cfg.OnSessionEnd(context.Background(), s.callID, reason)
	}
}

func (s *wsSession) writeText(payload []byte) {
	if s.closed.Load() || len(payload) == 0 {
		return
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_ = s.conn.WriteMessage(websocket.TextMessage, payload)
}

func (s *wsSession) writeBinary(payload []byte) error {
	if s.closed.Load() || len(payload) == 0 {
		return nil
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return s.conn.WriteMessage(websocket.BinaryMessage, payload)
}
