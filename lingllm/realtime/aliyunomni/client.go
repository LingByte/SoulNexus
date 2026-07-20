package aliyunomni

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// DashScope Qwen-Omni realtime — full-duplex multimodal Agent.
//
// Wire protocol:
//
//	wss://dashscope.aliyuncs.com/api-ws/v1/realtime?model=<model>
//
// Event family is the OpenAI-realtime-style JSON stream documented at:
//
//	https://www.alibabacloud.com/help/en/model-studio/qwen-omni-realtime
//
// Reference Python SDK: dashscope.audio.qwen_omni.OmniRealtimeConversation
//
// Why a hand-rolled client (no Go SDK):
//   - DashScope ships a Python SDK only for this endpoint.
//   - The wire format is small (~10 event types we care about) and stable.
//   - We already wrote a sibling client for Qwen-TTS realtime
//     (`synthesizer/aliyun.go`); the connection/auth/commit shape is
//     identical so the cost of a second client is low.
//
// Mapping to realtime.Agent events:
//
//	wire: session.created / session.updated         → EventSessionOpen (once)
//	wire: input_audio_buffer.speech_started         → EventUserSpeechStarted
//	wire: input_audio_buffer.speech_stopped         → EventUserSpeechEnded
//	wire: conversation.item.input_audio_transcription.completed
//	                                                → EventUserTranscript(final=true)
//	wire: response.audio_transcript.delta           → EventAssistantText(final=false)
//	wire: response.audio_transcript.done            → EventAssistantText(final=true)
//	wire: response.audio.delta (base64)             → EventAssistantAudio
//	wire: response.done                             → EventAssistantTurnEnd
//	wire: error                                     → EventError(fatal)
//	WS close                                        → EventSessionClose

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/lingllm/realtime"
	"github.com/gorilla/websocket"
)

// ProviderSlug is the canonical slug under which this implementation
// registers (see init() below). Aliases keep credential JSON forwards-
// compatible with users who copied DashScope sample code verbatim.
const (
	ProviderSlug   = "aliyun_omni"
	defaultBaseURL = "wss://dashscope.aliyuncs.com/api-ws/v1/realtime"
	defaultModel   = "qwen3.5-omni-flash-realtime-2026-03-15"
	defaultVoice   = "Cherry"
	defaultDialMs  = 10000
	defaultSendBuf = 64
)

func init() {
	realtime.Register(New, ProviderSlug, "qwen_omni", "dashscope_omni")
}

// Config is the typed shape of the credential JSON. New() mirrors it from a
// map so factory wiring stays simple.
type Config struct {
	APIKey        string
	BaseURL       string
	Model         string
	Voice         string
	Instructions  string
	DialTimeoutMs int
}

// New is the realtime.Provider entry point. cfg keys (camelCase preferred,
// snake_case accepted):
//
//	apiKey       (required) DashScope API key (sk-…)
//	model                  default qwen3.5-omni-flash-realtime-2026-03-15
//	voice                  default Cherry
//	instructions           system prompt / persona
//	baseUrl                override WS endpoint
//	dialTimeoutMs          handshake timeout, default 10000
func New(cfg map[string]any, opts realtime.Options) (realtime.Agent, error) {
	c := Config{
		APIKey:        firstString(cfg, "apiKey", "api_key"),
		BaseURL:       firstString(cfg, "baseUrl", "base_url"),
		Model:         firstString(cfg, "model"),
		Voice:         firstString(cfg, "voice"),
		Instructions:  firstString(cfg, "instructions"),
		DialTimeoutMs: firstInt(cfg, "dialTimeoutMs", "dial_timeout_ms"),
	}
	if strings.TrimSpace(c.APIKey) == "" {
		return nil, fmt.Errorf("aliyunomni: apiKey is required")
	}
	if c.BaseURL == "" {
		c.BaseURL = defaultBaseURL
	}
	if c.Model == "" {
		c.Model = defaultModel
	}
	if c.Voice == "" {
		c.Voice = defaultVoice
	}
	if c.DialTimeoutMs <= 0 {
		c.DialTimeoutMs = defaultDialMs
	}
	// Merge tenant instructions (credential) with per-call rules (SystemPrompt).
	// Previously SystemPrompt was ignored when instructions was set — tenant
	// persona must stay primary and augment rules append after it.
	instr := strings.TrimSpace(c.Instructions)
	sys := strings.TrimSpace(opts.SystemPrompt)
	switch {
	case instr != "" && sys != "":
		c.Instructions = instr + "\n\n" + sys
	case sys != "":
		c.Instructions = sys
	}
	if strings.TrimSpace(opts.Voice) != "" {
		c.Voice = opts.Voice
	}
	return &agent{cfg: c, opts: opts, sendCh: make(chan []byte, defaultSendBuf)}, nil
}

// --- agent implementation ---------------------------------------------------

type pendingFunctionCall struct {
	CallID    string
	Name      string
	Arguments string
}

type agent struct {
	cfg  Config
	opts realtime.Options

	conn *websocket.Conn

	startOnce sync.Once
	closeOnce sync.Once
	closed    atomic.Bool
	openFired atomic.Bool

	// sendCh serializes all outbound JSON frames so PushAudio (audio frame
	// goroutine) and Cancel (control goroutine) don't race on the WS.
	sendCh chan []byte
	wg     sync.WaitGroup

	pendingFCMu sync.Mutex
	pendingFCs  []pendingFunctionCall

	rootCtx    context.Context
	rootCancel context.CancelFunc

	// startErr is reported from Start so the caller doesn't have to
	// observe the read goroutine to learn handshake failed.
	startErr atomic.Value // error
	startGo  chan struct{}
}

func (a *agent) Start(ctx context.Context) error {
	var startErr error
	a.startOnce.Do(func() {
		startErr = a.doStart(ctx)
	})
	if startErr != nil {
		return startErr
	}
	if v := a.startErr.Load(); v != nil {
		if e, _ := v.(error); e != nil {
			return e
		}
	}
	return nil
}

func (a *agent) doStart(ctx context.Context) error {
	wsURL, err := buildURL(a.cfg.BaseURL, a.cfg.Model)
	if err != nil {
		return err
	}
	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = time.Duration(a.cfg.DialTimeoutMs) * time.Millisecond
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+a.cfg.APIKey)
	headers.Set("X-DashScope-OmniRealtime", "true")

	conn, resp, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		status := -1
		if resp != nil {
			status = resp.StatusCode
			resp.Body.Close()
		}
		return fmt.Errorf("aliyunomni: dial %s (status=%d): %w", wsURL, status, err)
	}
	a.conn = conn

	// Detach from caller ctx after successful Start: realtime sessions
	// outlive the function call. Caller controls teardown via Close().
	a.rootCtx, a.rootCancel = context.WithCancel(context.Background())
	a.startGo = make(chan struct{})

	a.wg.Add(2)
	go a.writeLoop()
	go a.readLoop()

	// Send session.update first so the server knows our voice / modalities
	// before we push any audio. This event is fire-and-forget; we surface
	// success implicitly via session.updated event in readLoop.
	if err := a.sendJSON(map[string]any{
		"type":    "session.update",
		"session": a.buildSession(),
	}, false); err != nil {
		_ = conn.Close()
		return fmt.Errorf("aliyunomni: session.update: %w", err)
	}
	return nil
}

func (a *agent) buildSession() map[string]any {
	mods := a.opts.Modalities
	if len(mods) == 0 {
		mods = []string{"text", "audio"}
	}
	session := map[string]any{
		"voice":               a.cfg.Voice,
		"modalities":          mods,
		"input_audio_format":  "pcm",
		"output_audio_format": "pcm",
		"input_audio_transcription": map[string]any{
			"model": "gummy-realtime-v1",
		},
	}
	if tools := realtime.ToolsForSession(a.opts.Tools); len(tools) > 0 {
		session["tools"] = tools
	}
	if !a.opts.DisableServerVAD {
		// Vendor default is server VAD on; we set it explicitly so future
		// schema changes don't silently flip behaviour.
		session["turn_detection"] = map[string]any{
			"type":                "server_vad",
			"threshold":           0.5,
			"prefix_padding_ms":   500,
			"silence_duration_ms": 800,
			"create_response":     true,
			"interrupt_response":  true,
		}
	} else {
		session["turn_detection"] = nil
	}
	if strings.TrimSpace(a.cfg.Instructions) != "" {
		session["instructions"] = a.cfg.Instructions
	}
	// Temperature: forward only when caller explicitly set it (>0). The
	// Aliyun Qwen-Omni realtime API mirrors OpenAI's session.update
	// shape and accepts a top-level `temperature` field; the documented
	// supported range is [0.6, 1.2] with default ~0.8. We clamp here so
	// a "lower is more deterministic" preset from voice attach doesn't
	// get silently rejected by the server. If you want truly
	// deterministic behaviour, pair this with a tight system prompt —
	// at 0.6 the model is still LLM, not a state machine.
	if a.opts.Temperature > 0 {
		t := a.opts.Temperature
		if t < 0.6 {
			t = 0.6
		}
		if t > 1.2 {
			t = 1.2
		}
		session["temperature"] = t
	}
	return session
}

// PushAudio base64-encodes pcm and queues an input_audio_buffer.append
// event. Frames must be PCM16LE @ Options.InputSampleRate (default 16 kHz)
// mono. voice attach typically pushes 20 ms (640 bytes) chunks.
func (a *agent) PushAudio(pcm []byte) error {
	if a.closed.Load() {
		return realtime.ErrAgentClosed
	}
	if len(pcm) == 0 {
		return nil
	}
	return a.sendJSON(map[string]any{
		"type":  "input_audio_buffer.append",
		"audio": base64.StdEncoding.EncodeToString(pcm),
	}, true)
}

func (a *agent) CommitInputAudio() error {
	if a.closed.Load() {
		return realtime.ErrAgentClosed
	}
	return a.sendJSON(map[string]any{"type": "input_audio_buffer.commit"}, false)
}

// Cancel sends response.cancel so the model stops its current audio reply.
// voice attach calls this on barge-in to free server resources; the local
// outbound buffer must also be flushed by the media layer (we do it on
// EventUserSpeechStarted, but Cancel is the explicit mechanism).
func (a *agent) Cancel() error {
	if a.closed.Load() {
		return realtime.ErrAgentClosed
	}
	return a.sendJSON(map[string]any{"type": "response.cancel"}, false)
}

// EnableServerVAD switches the omni session to server-side turn detection.
// Used after post-handoff welcome TTS so caller speech triggers replies without
// manual input_audio_buffer.commit.
func (a *agent) EnableServerVAD() error {
	if a.closed.Load() || a.conn == nil {
		return realtime.ErrAgentClosed
	}
	a.opts.DisableServerVAD = false
	return a.sendJSON(map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"turn_detection": map[string]any{
				"type":                "server_vad",
				"threshold":           0.5,
				"prefix_padding_ms":   500,
				"silence_duration_ms": 800,
			},
		},
	}, false)
}

// UpdateInstructions sends session.update with new instructions mid-call.
func (a *agent) UpdateInstructions(instructions string) error {
	instructions = strings.TrimSpace(instructions)
	if instructions == "" {
		return nil
	}
	a.cfg.Instructions = instructions
	if a.closed.Load() || a.conn == nil {
		return realtime.ErrAgentClosed
	}
	return a.sendJSON(map[string]any{
		"type": "session.update",
		"session": map[string]any{
			"instructions": instructions,
		},
	}, false)
}

func (a *agent) Close() error {
	a.closeOnce.Do(func() {
		a.closed.Store(true)
		if a.rootCancel != nil {
			a.rootCancel()
		}
		// Try a polite close before tearing the socket. Don't block on it;
		// the read loop will see the close frame or a read error and
		// unwind either way.
		if a.conn != nil {
			_ = a.conn.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "client close"),
				time.Now().Add(200*time.Millisecond))
			_ = a.conn.Close()
		}
		// Best-effort: drain any goroutines so callers can rely on Close
		// being a complete teardown.
		a.wg.Wait()
	})
	return nil
}

// sendJSON queues a single JSON event onto the writer goroutine. Drops on
// closed channel after Close() — callers see ErrAgentClosed via the
// closed flag, never a panic.
func (a *agent) sendJSON(event map[string]any, nonBlocking bool) error {
	if a.closed.Load() {
		return realtime.ErrAgentClosed
	}
	if event["event_id"] == nil {
		event["event_id"] = fmt.Sprintf("event_%d", time.Now().UnixNano())
	}
	buf, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if nonBlocking {
		select {
		case a.sendCh <- buf:
			return nil
		case <-a.rootCtx.Done():
			return realtime.ErrAgentClosed
		default:
			// WS writer backed up — drop this frame so voice media EventBus
			// workers are not stalled (mobile / jitter bursts).
			return nil
		}
	}
	select {
	case a.sendCh <- buf:
		return nil
	case <-a.rootCtx.Done():
		return realtime.ErrAgentClosed
	}
}

func (a *agent) writeLoop() {
	defer a.wg.Done()
	for {
		select {
		case <-a.rootCtx.Done():
			return
		case buf, ok := <-a.sendCh:
			if !ok {
				return
			}
			if err := a.conn.WriteMessage(websocket.TextMessage, buf); err != nil {
				// Surface as fatal; the read loop will also observe the
				// broken WS and clean up.
				a.emit(realtime.Event{
					Type:   realtime.EventError,
					Vendor: ProviderSlug,
					Err:    fmt.Errorf("aliyunomni: write: %w", err),
					Fatal:  true,
				})
				return
			}
		}
	}
}

func (a *agent) readLoop() {
	defer a.wg.Done()
	defer a.emit(realtime.Event{Type: realtime.EventSessionClose, Vendor: ProviderSlug})
	for {
		_, raw, err := a.conn.ReadMessage()
		if err != nil {
			if a.closed.Load() {
				return
			}
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			a.emit(realtime.Event{
				Type:   realtime.EventError,
				Vendor: ProviderSlug,
				Err:    fmt.Errorf("aliyunomni: read: %w", err),
				Fatal:  true,
			})
			return
		}
		a.dispatch(raw)
	}
}

func (a *agent) dispatch(raw []byte) {
	var head wireHead
	if err := json.Unmarshal(raw, &head); err != nil {
		// Non-JSON or unknown shape — log via error event but keep
		// session alive. voice attach treats non-fatal errors as warnings.
		a.emit(realtime.Event{
			Type:   realtime.EventError,
			Vendor: ProviderSlug,
			Err:    fmt.Errorf("aliyunomni: bad json: %w", err),
			Raw:    raw,
		})
		return
	}
	switch head.Type {
	case "session.created", "session.updated":
		// Fire EventSessionOpen exactly once. Vendor sends both events
		// during a normal handshake; we de-dupe so voice attach gets a
		// single edge-trigger.
		if a.openFired.CompareAndSwap(false, true) {
			a.emit(realtime.Event{Type: realtime.EventSessionOpen, Vendor: ProviderSlug, Raw: raw})
		}

	case "input_audio_buffer.speech_started":
		a.emit(realtime.Event{Type: realtime.EventUserSpeechStarted, Vendor: ProviderSlug})

	case "input_audio_buffer.speech_stopped":
		a.emit(realtime.Event{Type: realtime.EventUserSpeechEnded, Vendor: ProviderSlug})

	case "conversation.item.input_audio_transcription.completed":
		var msg wireTranscript
		_ = json.Unmarshal(raw, &msg)
		if msg.Transcript != "" {
			a.emit(realtime.Event{
				Type:   realtime.EventUserTranscript,
				Vendor: ProviderSlug,
				Text:   msg.Transcript,
				Final:  true,
			})
		}

	case "response.audio_transcript.delta":
		var msg wireDelta
		_ = json.Unmarshal(raw, &msg)
		if msg.Delta != "" {
			a.emit(realtime.Event{
				Type:   realtime.EventAssistantText,
				Vendor: ProviderSlug,
				Text:   msg.Delta,
				Final:  false,
			})
		}

	case "response.audio_transcript.done":
		var msg wireTranscript
		_ = json.Unmarshal(raw, &msg)
		text := msg.Transcript
		if text == "" {
			// Some payload variants put final text in `text` instead of
			// `transcript`; fall back so voice keyword detection sees it.
			var alt wireDelta
			_ = json.Unmarshal(raw, &alt)
			text = alt.Delta
		}
		a.emit(realtime.Event{
			Type:   realtime.EventAssistantText,
			Vendor: ProviderSlug,
			Text:   text,
			Final:  true,
		})

	case "response.audio.delta", "response.output_audio.delta":
		var msg wireDelta
		_ = json.Unmarshal(raw, &msg)
		if msg.Delta == "" {
			return
		}
		pcm, err := base64.StdEncoding.DecodeString(msg.Delta)
		if err != nil {
			a.emit(realtime.Event{
				Type:   realtime.EventError,
				Vendor: ProviderSlug,
				Err:    fmt.Errorf("aliyunomni: bad base64 audio: %w", err),
			})
			return
		}
		a.emit(realtime.Event{
			Type:    realtime.EventAssistantAudio,
			Vendor:  ProviderSlug,
			AudioPC: pcm,
		})

	case "response.function_call_arguments.done":
		var msg wireFunctionCallDone
		_ = json.Unmarshal(raw, &msg)
		if msg.Name != "" && msg.CallID != "" {
			a.pendingFCMu.Lock()
			a.pendingFCs = append(a.pendingFCs, pendingFunctionCall{
				CallID:    msg.CallID,
				Name:      msg.Name,
				Arguments: msg.Arguments,
			})
			a.pendingFCMu.Unlock()
		} else {
			a.emit(realtime.Event{
				Type:   realtime.EventError,
				Vendor: ProviderSlug,
				Err:    fmt.Errorf("aliyunomni: function_call_arguments.done missing name/call_id"),
				Raw:    raw,
			})
		}

	case "response.done":
		a.finishResponseTurn(raw)

	case "error":
		var msg wireError
		_ = json.Unmarshal(raw, &msg)
		text := "unknown"
		if msg.Error != nil {
			if msg.Error.Message != "" {
				text = msg.Error.Message
			} else if msg.Error.Code != "" {
				text = msg.Error.Code
			}
		}
		// Benign: we may call response.cancel after the turn already ended
		// (forced TTS confirm path + late assistant.final from Omni).
		fatal := true
		lower := strings.ToLower(text)
		if strings.Contains(lower, "none active response") || strings.Contains(lower, "no active response") {
			fatal = false
		}
		a.emit(realtime.Event{
			Type:   realtime.EventError,
			Vendor: ProviderSlug,
			Err:    fmt.Errorf("aliyunomni: server error: %s", text),
			Fatal:  fatal,
			Raw:    raw,
		})
	default:
		// Silently ignore events we don't translate. Forwarding raw to
		// debug logs is the media layer's job.
	}
}

// finishResponseTurn executes pending function calls (if any), posts outputs
// back to DashScope, and signals assistant turn end to voice attach.
func (a *agent) finishResponseTurn(raw []byte) {
	a.pendingFCMu.Lock()
	pending := append([]pendingFunctionCall(nil), a.pendingFCs...)
	a.pendingFCs = nil
	a.pendingFCMu.Unlock()

	if len(pending) > 0 && a.opts.ToolHandler != nil {
		for _, fc := range pending {
			var args map[string]any
			if fc.Arguments != "" {
				_ = json.Unmarshal([]byte(fc.Arguments), &args)
			}
			if args == nil {
				args = map[string]any{}
			}
			output := a.opts.ToolHandler(fc.Name, args)
			_ = a.sendJSON(map[string]any{
				"type": "conversation.item.create",
				"item": map[string]any{
					"type":    "function_call_output",
					"call_id": fc.CallID,
					"output":  output,
				},
			}, false)
		}
		_ = a.sendJSON(map[string]any{"type": "response.create"}, false)
	}
	ev := realtime.Event{Type: realtime.EventAssistantTurnEnd, Vendor: ProviderSlug, Raw: raw}
	if len(pending) > 0 {
		names := make([]string, 0, len(pending))
		for _, fc := range pending {
			names = append(names, fc.Name)
		}
		ev.Text = strings.Join(names, ",")
	}
	a.emit(ev)
}

func (a *agent) emit(ev realtime.Event) {
	if a.opts.OnEvent == nil {
		return
	}
	// Don't deliver events after Close to avoid races with caller-side
	// state machines tearing down.
	if a.closed.Load() && ev.Type != realtime.EventSessionClose {
		return
	}
	a.opts.OnEvent(ev)
}

// --- protocol helpers --------------------------------------------------------

func buildURL(base, model string) (string, error) {
	if base == "" {
		base = defaultBaseURL
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("aliyunomni: parse baseUrl: %w", err)
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return "", fmt.Errorf("aliyunomni: baseUrl must be ws:// or wss://, got %q", u.Scheme)
	}
	q := u.Query()
	if q.Get("model") == "" && model != "" {
		q.Set("model", model)
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return ""
}

func firstInt(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			switch t := v.(type) {
			case int:
				return t
			case int64:
				return int(t)
			case float64:
				return int(t)
			}
		}
	}
	return 0
}

// --- wire types -------------------------------------------------------------

type wireHead struct {
	Type string `json:"type"`
}

type wireDelta struct {
	Delta string `json:"delta"`
}

type wireTranscript struct {
	Transcript string `json:"transcript"`
}

type wireFunctionCallDone struct {
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type wireError struct {
	Error *struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// Compile-time guards.
var (
	_ realtime.Agent    = (*agent)(nil)
	_ realtime.Provider = New
	_                   = errors.New
)
