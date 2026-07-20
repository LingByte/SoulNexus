package websocket

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/session"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/SoulNexus/pkg/dialog/transport/pcm"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(*http.Request) bool { return true },
}

// Serve upgrades HTTP to WebSocket and attaches the dialog engine.
// Wire dialect follows xiaozhi-esp32 (hello / listen / abort / tts / stt) while
// keeping legacy debug frames (transcript.*, hangup, status) for the console UI.
func Serve(w http.ResponseWriter, r *http.Request, lg engine.Logger) {
	mgr := session.Default()
	if mgr == nil {
		http.Error(w, "voice session manager not initialized", http.StatusServiceUnavailable)
		return
	}
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		http.Error(w, "session_id required", http.StatusBadRequest)
		return
	}
	sess, ok := mgr.Get(sessionID)
	if !ok || sess == nil {
		http.Error(w, "unknown session", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	runCtx, runCancel := context.WithCancel(context.Background())
	defer func() {
		runCancel()
		_ = conn.Close()
		sess.Close()
	}()

	var writeMu sync.Mutex
	writeJSON := func(payload []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(websocket.TextMessage, payload)
	}
	writeBinary := func(payload []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteMessage(websocket.BinaryMessage, payload)
	}

	env, envOK, _ := tenantcfg.Resolve(context.Background(), sess.TenantID, sess.CallID)
	if envOK {
		env = session.EffectiveVoiceEnv(sess, env)
	}
	mode := "pipeline"
	if envOK {
		mode = string(session.ResolveMode(env))
	}
	sess.BindTurnNotify(writeJSON, mode, "websocket")

	ww := WireWriter{WriteJSON: writeJSON, WriteBinary: writeBinary}
	ttsEnv := newTTSEnvelope(sess.ID, sess.SampleRate, ww)
	port := pcm.NewPort(pcm.Config{
		SessionID:  sess.ID,
		TenantID:   strconv.FormatUint(uint64(sess.TenantID), 10),
		SampleRate: sess.SampleRate,
	})
	port.OutputFn = func(f engine.PCMFrame) error {
		if len(f.Data) == 0 {
			return nil
		}
		return ttsEnv.WritePCM(f.Data)
	}

	if err := sess.BindPort(port); err != nil {
		_ = writeJSON(errorFrame(err.Error()))
		return
	}
	hello, _ := EncodeHello(sess.ID, sess.SampleRate)
	if err := writeJSON(hello); err != nil {
		return
	}

	if envOK && mode != string(engine.ModeRealtime) {
		session.StartAssistantWelcomePrewarm(sess.CallID, env, sess.SampleRate)
		welcome, welcomeErr := session.PlayAssistantWelcome(runCtx, sess.CallID, env, sess.SampleRate, func(pcmData []byte) error {
			return ttsEnv.WritePCM(pcmData)
		}, true)
		if welcome != "" {
			if b, err := EncodeTranscript(sess.ID, TypeTranscriptAssistant, welcome, true); err == nil {
				_ = writeJSON(b)
			}
			if b, err := EncodeLLMResponse(welcome); err == nil {
				_ = writeJSON(b)
			}
		}
		if welcomeErr != nil && runCtx.Err() == nil {
			_ = writeJSON(errorFrame("welcome TTS: " + welcomeErr.Error()))
		}
	}

	if err := sess.StartEngine(runCtx, lg); err != nil {
		_ = writeJSON(errorFrame(err.Error()))
		return
	}
	if envOK {
		if b, err := EncodeStatus(sess.ID, "ready", "engine attached ("+mode+")"); err == nil {
			_ = writeJSON(b)
		}
	}

	// Default listening=true keeps console debug UI working without an explicit listen.
	// Xiaozhi clients still send listen start/stop to gate the mic.
	var listening atomic.Bool
	listening.Store(true)

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		switch msgType {
		case websocket.BinaryMessage:
			if len(data) == 0 || !listening.Load() {
				continue
			}
			if err := PushInputFromBinary(port, data); err != nil {
				_ = writeJSON(errorFrame(err.Error()))
			}
		case websocket.TextMessage:
			fr, err := DecodeFrame(data)
			if err != nil {
				continue
			}
			switch fr.Type {
			case TypeHello:
				// Client hello (xiaozhi): already welcomed; optional renegotiation ignored for PCM MVP.
			case TypeListen:
				switch fr.State {
				case ListenStart, "":
					listening.Store(true)
				case ListenStop:
					listening.Store(false)
				}
			case TypeAbort:
				listening.Store(true)
				ttsEnv.ForceStop()
				port.TriggerBargeIn()
				if b, err := EncodeAbortConfirm(sess.ID); err == nil {
					_ = writeJSON(b)
				}
			case TypePing:
				if b, err := EncodePong(sess.ID); err == nil {
					_ = writeJSON(b)
				}
			case TypeHangup:
				return
			case TypePCM:
				if !listening.Load() {
					continue
				}
				if err := PushInputFromWire(port, fr); err != nil {
					_ = writeJSON(errorFrame(err.Error()))
				}
			}
		}
	}
}
