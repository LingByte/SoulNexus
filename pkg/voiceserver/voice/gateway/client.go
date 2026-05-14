package gateway

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/metrics"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/vad"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ClientConfig configures a per-call dialog-plane WebSocket client.
type ClientConfig struct {
	// URL is the dialog-side WebSocket endpoint (ws:// or wss://). Required.
	// The call_id is appended as a query parameter automatically. Optional
	// auth keys (apiKey, apiSecret, agentId) are often supplied out-of-band
	// (e.g. cmd/voice merges DIALOG_* env vars before dial).
	URL string

	// Attached is a live voice.Attached produced by voice.Attach. Required.
	// The client wires its ASR callbacks and drives its TTS pipeline.
	Attached *voice.Attached

	// CallID for this connection. Required. Echoed into every Event / Command.
	CallID string

	// HandshakeHeaders is added to the WS upgrade (Authorization, tenant, etc.).
	HandshakeHeaders http.Header

	// DialTimeout is the WS upgrade timeout (default 5s).
	DialTimeout time.Duration

	// OnHangup fires when the dialog app issues a hangup command. The caller
	// should then tear down the SIP dialog (send BYE / observer return true).
	OnHangup func(reason string)

	// OnASRFinal fires (after the event has already been forwarded over the
	// WS) for every final transcript. Used by the persistence layer to
	// remember the most recent user utterance so it can be paired with the
	// next assistant TTS into a dialog turn row. Optional.
	OnASRFinal func(text string)

	// OnTTSStart fires just before each queued tts.speak begins playback —
	// after tts.started has been emitted to the dialog plane but before any
	// audio frames hit the media transport. Transports that wrap audio in a
	// framing message (xiaozhi-esp32 tts:start, RTSP markers, …) use this to
	// inject the wrapper. Optional.
	OnTTSStart func(utteranceID, text string)

	// OnTurn fires after each tts.speak completes. It carries enough
	// context to persist a full dialog turn:
	//
	//   - UtteranceID — matches tts.started / tts.ended events
	//   - LLMText     — the text that was actually spoken
	//   - Meta        — optional dialog-side metadata (LLM model, latency)
	//   - DurationMs  — wall-clock time the Speak call took
	//   - OK          — false on synthesis or playback error
	//
	// Persisters typically combine this with the most recent OnASRFinal to
	// build a (user, assistant) pair. Optional.
	OnTurn func(t TurnEvent)

	// BargeIn, when non-nil, enables user-interrupts-AI barge-in.
	// The Client will wire this detector into the ASR pipeline with
	// "is TTS playing" = Attached.TTS.IsPlaying and will fire
	// TTS.Interrupt() plus a tts.interrupt event to the dialog plane
	// whenever the detector trips. Passing a zero-value detector
	// (vad.NewDetector()) gets sensible defaults; callers who want
	// different thresholds should tune the detector before Start.
	// nil = barge-in disabled.
	BargeIn *vad.Detector

	// ReconnectAttempts is how many times the Client will try to
	// re-establish the dialog WebSocket after the initial connection
	// drops mid-call. 0 = no reconnect, immediately propagate hangup
	// (legacy behavior). Each attempt waits ReconnectInitialBackoff
	// × 2^(attempt-1) before redialing, capped at 30s.
	ReconnectAttempts int

	// ReconnectInitialBackoff is the wait before the first redial.
	// Subsequent attempts double this (exponential backoff). 0 → 1s.
	ReconnectInitialBackoff time.Duration

	// HoldTextFirst / HoldTextRetry / HoldTextGiveUp are spoken to
	// the user via TTS during reconnect: HoldTextFirst once when the
	// WS first drops, HoldTextRetry between subsequent attempts, and
	// HoldTextGiveUp once when all attempts have failed (right before
	// the final hangup). Empty strings skip that prompt — useful when
	// the dialog plane is local and reconnects are sub-second.
	HoldTextFirst  string
	HoldTextRetry  string
	HoldTextGiveUp string

	// ASRSentenceFilter, when non-nil, intercepts every recogniser
	// callback before it becomes an asr.partial / asr.final event.
	// Use cases:
	//   - Suppress half-sentence partials so the dialog plane only
	//     sees complete sentences (cuts LLM thrash on chatty ASR).
	//   - Drop near-duplicate hypotheses caused by recogniser jitter
	//     (extra spaces, swapped punctuation).
	//
	// Lifetime: one filter per call. The Client never reuses or
	// resets it across calls — supply a fresh asr.NewSentenceFilter
	// from your factory. Pass nil to keep the legacy "every partial
	// flows through unchanged" behaviour.
	//
	// Trade-off: enabling this adds 1–2 partials of latency on each
	// turn (we wait for a sentence terminator) but typically halves
	// the number of asr.partial events the LLM has to digest.
	ASRSentenceFilter ASRSentenceFilter

	// Logger is optional.
	Logger *zap.Logger
}

// ASRSentenceFilter is the minimal contract the gateway consumes from
// pkg/voice/asr.SentenceFilter. Defining it here (rather than
// importing the concrete type) avoids a hard package dependency:
// callers that don't need filtering pay zero cost in build graph.
type ASRSentenceFilter interface {
	Update(text string, isFinal bool) string
	Reset()
}

// TurnEvent is delivered to ClientConfig.OnTurn after each Speak call.
type TurnEvent struct {
	UtteranceID string
	LLMText     string
	Meta        *CommandMeta
	// DurationMs is the wall-clock time the underlying TTS Speak call
	// took (synthesis + paced playback).
	DurationMs int
	// TTSFirstByteMs is the time from invoking Speak() to the first
	// PCM frame actually leaving the TTS pipeline. Measures the TTS
	// engine's cold-start / time-to-first-byte. 0 if the Speak failed
	// before producing any audio.
	TTSFirstByteMs int
	// E2EFirstByteMs is the user-perceived end-to-end response latency:
	// time from the most recent ASR final to the first PCM frame
	// shipped on this Speak. Set ONLY on the first Speak that follows a
	// given ASR final — subsequent intra-turn Speaks (sentence-segmented
	// LLM streaming) leave it 0 because they are not what the human
	// "heard the AI start to reply" against. 0 also when no ASR final
	// preceded this Speak (e.g. unprompted greeting).
	E2EFirstByteMs int
	// MoreSpeaksQueued is true when another tts.speak is already waiting on
	// the gateway worker queue when this Speak returns. xiaozhi uses it to
	// avoid tts:stop/tts:start between adjacent chunks so playback stays continuous.
	MoreSpeaksQueued bool
	OK               bool
}

// Client streams ASR events / DTMF to a dialog app via WebSocket and executes
// TTS / hangup commands received back. It holds exactly one WS per call.
//
// TTS commands are serialized through a single-worker queue: a dialog app may
// fire many tts.speak commands rapidly (e.g. one per LLM-segmented utterance),
// but TTS.Speak is itself a long-running stateful call (the pipeline paces
// frames at realtime onto the MediaLeg). Running them in parallel would
// produce overlapping audio on the wire. The worker drains the queue strictly
// in order, emitting tts.started / tts.ended events around each Speak.
type Client struct {
	cfg    ClientConfig
	log    *zap.Logger
	conn   *websocket.Conn
	writeM sync.Mutex

	started atomic.Bool
	closed  atomic.Bool

	runCtx    context.Context
	runCancel context.CancelFunc

	ttsQueue chan ttsJob
	ttsWg    sync.WaitGroup

	// asrFinalAt holds the wall-clock at which the most recent
	// ASR-final transcript was forwarded to the dialog plane. The
	// next ttsWorker iteration consumes (and clears) it to compute
	// E2EFirstByteMs — the user-perceived "stopped talking → AI
	// starts talking" latency. Pointer-typed so a single atomic load
	// can distinguish "no final pending" (nil) from "final pending"
	// (non-nil) without an extra flag.
	asrFinalAt atomic.Pointer[time.Time]

	// transport is "sip" / "xiaozhi" / "webrtc" — captured from
	// StartMeta.To so metrics carry a transport label consistent with
	// the persister and /metrics output. Atomic string swap; read-only
	// after Start() returns, so we could have stored it plain but the
	// atomic pattern matches the other lifecycle fields.
	transport atomic.Pointer[string]
}

// ttsJob is one queued TTS utterance. meta carries optional dialog-side
// metadata (LLM model, latency) that is surfaced via ClientConfig.OnTurn
// after the Speak completes.
type ttsJob struct {
	text  string
	utter string
	meta  *CommandMeta
}

// NewClient validates cfg. The WebSocket is opened by Start.
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("voice/gateway: empty URL")
	}
	if cfg.Attached == nil {
		return nil, fmt.Errorf("voice/gateway: nil Attached")
	}
	if cfg.CallID == "" {
		return nil, fmt.Errorf("voice/gateway: empty CallID")
	}
	if cfg.Attached.ASR == nil && cfg.Attached.TTS == nil {
		return nil, fmt.Errorf("voice/gateway: Attached has neither ASR nor TTS")
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	c := &Client{cfg: cfg, log: cfg.Logger}
	if c.log == nil {
		c.log = zap.NewNop()
	}
	return c, nil
}

// Start dials the dialog WS, sends call.started, wires ASR/DTMF → events, and
// starts the command-read loop. It returns after the dial completes; the read
// loop runs in background until Close / peer-close / fatal error.
func (c *Client) Start(ctx context.Context, meta StartMeta) error {
	if c == nil {
		return fmt.Errorf("voice/gateway: nil client")
	}
	// Stash the transport label before we use it in metrics below.
	// Empty = "unknown" in Prom label space, so we fall back to that
	// rather than drop the series entirely.
	t := strings.TrimSpace(meta.To)
	if t == "" {
		t = "unknown"
	}
	c.transport.Store(&t)
	if !c.started.CompareAndSwap(false, true) {
		return fmt.Errorf("voice/gateway: already started")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	dialURL, err := appendCallIDQuery(c.cfg.URL, c.cfg.CallID)
	if err != nil {
		return err
	}

	dctx, cancel := context.WithTimeout(ctx, c.cfg.DialTimeout)
	defer cancel()
	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.DialContext(dctx, dialURL, c.cfg.HandshakeHeaders)
	if err != nil {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		return fmt.Errorf("dial dialog ws %s (http=%d): %w", RedactDialogDialURL(dialURL), code, err)
	}
	c.conn = conn

	c.runCtx, c.runCancel = context.WithCancel(context.Background())

	// Single worker for TTS so back-to-back Speak commands play sequentially.
	// Buffer is generous enough that LLM-segmented turns don't backpressure
	// the WS read loop; new commands queue and play in arrival order.
	if c.cfg.Attached.TTS != nil {
		c.ttsQueue = make(chan ttsJob, 64)
		c.ttsWg.Add(1)
		go c.ttsWorker()
		logger.Info(fmt.Sprintf("[gw] call=%s tts worker started (serial speak queue cap=64)",
			c.cfg.CallID))
	}

	// Wire ASR → events.
	if c.cfg.Attached.ASR != nil {
		c.cfg.Attached.ASR.SetTextCallback(func(text string, isFinal bool) {
			text = strings.TrimSpace(text)
			if text == "" {
				return
			}
			// Optional: route through SentenceFilter. The filter
			// returns "" to suppress (mid-sentence partial / dup
			// hypothesis), or a delta we should forward as the
			// event text. Final transcripts always pass through
			// even when the filter would otherwise return "" —
			// dropping a final would silently strand a turn.
			if f := c.cfg.ASRSentenceFilter; f != nil {
				delta := f.Update(text, isFinal)
				if delta == "" && !isFinal {
					return
				}
				if delta != "" {
					text = delta
				}
				// else: isFinal=true with empty delta → forward
				// the recogniser's text unchanged (a final must
				// fire even when its tail equals what we last
				// emitted, so the dialog plane hears "this turn
				// is closed").
			}
			t := EvASRPartial
			if isFinal {
				t = EvASRFinal
			}
			_ = c.sendEvent(Event{Type: t, CallID: c.cfg.CallID, Text: text})
			// After forwarding, fire the persistence hook so observers can
			// remember the last user utterance for the next-turn pairing.
			// We never fail the WS send for a slow observer — guard with a
			// goroutine? No: callers are expected to do their own deferred
			// work; we keep it synchronous and small.
			if isFinal {
				// Stamp the wall-clock at ASR finalization so the next
				// ttsWorker iteration can compute E2EFirstByteMs. Stamping
				// here (rather than in the user's OnASRFinal callback)
				// gets the timestamp closest to the moment the dialog
				// plane sees the final, regardless of how the user wires
				// downstream observers.
				now := time.Now()
				c.asrFinalAt.Store(&now)
				if c.cfg.OnASRFinal != nil {
					c.cfg.OnASRFinal(text)
				}
			}
		})
		c.cfg.Attached.ASR.SetErrorCallback(func(err error, fatal bool) {
			if err == nil {
				return
			}
			_ = c.sendEvent(Event{
				Type: EvASRError, CallID: c.cfg.CallID,
				Message: err.Error(), Fatal: fatal,
			})
			metrics.ASRError(c.transportLabel())
		})

		// Barge-in: when configured + TTS is currently playing, feed
		// every PCM frame through the VAD. On fire we drain the TTS
		// queue (so subsequent LLM-segmented utterances don't just
		// resume), interrupt the current Speak, and tell the dialog
		// plane via an explicit event so the LLM can adjust (e.g.
		// re-prompt, treat the remainder as unheard). A single
		// barge-in must only fire once — the Detector's internal
		// counter reset gives us that, but we additionally guard the
		// queue drain so a false re-trigger during the same Speak is
		// a no-op at the TTS layer.
		if c.cfg.BargeIn != nil && c.cfg.Attached.TTS != nil {
			det := c.cfg.BargeIn
			c.cfg.Attached.ASR.SetBargeInDetector(
				det,
				c.cfg.Attached.TTS.IsPlaying,
				func() {
					c.drainTTSQueue()
					c.cfg.Attached.TTS.Interrupt()
					_ = c.sendEvent(Event{
						Type: EvTTSInterrupt, CallID: c.cfg.CallID,
					})
					metrics.BargeIn(c.transportLabel())
					logger.Info(fmt.Sprintf("[gw] call=%s barge-in: tts interrupted by user voice",
						c.cfg.CallID))
				},
			)
		}
	}

	// Announce the call.
	_ = c.sendEvent(Event{
		Type: EvCallStarted, CallID: c.cfg.CallID,
		From: meta.From, To: meta.To, Codec: meta.Codec, PCMHz: meta.PCMHz,
	})

	go c.readLoop()
	c.log.Info("voice/gateway connected",
		zap.String("url", RedactDialogDialURL(dialURL)), zap.String("call_id", c.cfg.CallID))
	return nil
}

// StartMeta is passed into Start so call.started carries the SIP-level metadata.
type StartMeta struct {
	From  string
	To    string
	Codec string
	PCMHz int
}

// PushDTMF forwards a DTMF digit up to the dialog app. Safe to call any time
// after Start.
func (c *Client) PushDTMF(digit string, end bool) {
	if c == nil || !c.started.Load() {
		return
	}
	_ = c.sendEvent(Event{Type: EvDTMF, CallID: c.cfg.CallID, Digit: digit, End: end})
}

// ForwardTransferRequest emits a transfer.request event up to the
// dialog app, carrying the Refer-To URI parsed from an inbound SIP
// REFER. The dialog app decides what to do (typically: prompt the
// LLM, then either send a hangup command to release the leg or just
// acknowledge so the carrier completes the transfer itself). Safe to
// call any time after Start. No-op when the client hasn't started or
// is already closed.
func (c *Client) ForwardTransferRequest(target string) {
	if c == nil || !c.started.Load() || c.closed.Load() {
		return
	}
	_ = c.sendEvent(Event{
		Type:   EvTransferRequest,
		CallID: c.cfg.CallID,
		Target: target,
	})
}

// Close sends call.ended (if not already sent), closes the WS, and stops the
// read loop. Idempotent.
func (c *Client) Close(reason string) {
	if c == nil {
		return
	}
	if c.closed.Swap(true) {
		return
	}
	_ = c.sendEvent(Event{Type: EvCallEnded, CallID: c.cfg.CallID, Reason: reason})
	if c.runCancel != nil {
		c.runCancel()
	}
	// Drain the TTS queue (worker exits when ttsQueue is closed and runCtx
	// is done). Interrupt any in-flight Speak so the worker unblocks fast.
	if c.cfg.Attached.TTS != nil {
		c.cfg.Attached.TTS.Interrupt()
	}
	if c.ttsQueue != nil {
		close(c.ttsQueue)
		c.ttsWg.Wait()
		c.ttsQueue = nil
	}
	if c.conn != nil {
		_ = c.conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason),
			time.Now().Add(500*time.Millisecond))
		_ = c.conn.Close()
	}
}

// ----- internal -----

// transportLabel returns the cached transport ("sip"/"xiaozhi"/"webrtc")
// captured at Start time, or "unknown" if Start hasn't run yet.
// Always safe to call from any goroutine.
func (c *Client) transportLabel() string {
	if c == nil {
		return "unknown"
	}
	if p := c.transport.Load(); p != nil {
		return *p
	}
	return "unknown"
}

func (c *Client) sendEvent(ev Event) error {
	if c == nil || c.conn == nil || c.closed.Load() {
		return nil
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	c.writeM.Lock()
	defer c.writeM.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// readLoop drains command messages from the dialog WS until either the
// peer closes, an IO error fires, or the run-context is cancelled. On
// IO error it consults ClientConfig.ReconnectAttempts: with attempts > 0
// it tries to re-establish the connection (speaking hold messages
// through the TTS pipeline so the user doesn't hear silence), and only
// propagates OnHangup once all attempts have been exhausted. With
// attempts == 0 the legacy "fail-fast hangup" behaviour is preserved.
func (c *Client) readLoop() {
	for {
		if !c.runOnce() {
			// runOnce returned false → terminal: either Closed, or the
			// reconnect path gave up. Surface the hangup once.
			if !c.closed.Load() {
				if c.cfg.OnHangup != nil {
					c.cfg.OnHangup("dialog ws closed")
				}
				c.Close("dialog ws closed")
			}
			return
		}
	}
}

// runOnce reads commands from c.conn until error. Returns true to
// signal "we reconnected, please loop and read from the new conn",
// false to signal "give up". Caller (readLoop) handles teardown.
func (c *Client) runOnce() bool {
	c.conn.SetReadLimit(1 << 20)
	for {
		if c.runCtx != nil && c.runCtx.Err() != nil {
			return false
		}
		_ = c.conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			c.log.Debug("voice/gateway read end",
				zap.String("call_id", c.cfg.CallID), zap.Error(err))
			// Already closed (caller-initiated) → no reconnect.
			if c.closed.Load() || (c.runCtx != nil && c.runCtx.Err() != nil) {
				return false
			}
			// Try to reconnect. Returns true on success → caller loops
			// and pulls from the freshly-rebuilt c.conn.
			return c.reconnectWithHold()
		}
		var cmd Command
		if err := json.Unmarshal(data, &cmd); err != nil {
			c.log.Warn("voice/gateway bad cmd", zap.Error(err), zap.ByteString("raw", data))
			continue
		}
		c.dispatch(cmd)
	}
}

// reconnectWithHold attempts ClientConfig.ReconnectAttempts redials with
// exponential backoff. Between attempts it speaks the configured hold
// text through the existing TTS pipeline so the caller hears something
// reassuring instead of silence. Returns true once a redial succeeds
// (c.conn is replaced and ready to read), false when all attempts have
// been exhausted (in which case it has already played HoldTextGiveUp).
//
// Concurrency: writes to c.conn through c.writeM so an outgoing event
// from another goroutine never collides with the redial swap. The TTS
// queue is drained at the start so any in-flight LLM utterances don't
// resume mid-reconnect.
func (c *Client) reconnectWithHold() bool {
	maxN := c.cfg.ReconnectAttempts
	if maxN <= 0 {
		// Fail-fast mode — preserves legacy semantics for callers that
		// haven't opted into reconnect.
		return false
	}
	c.drainTTSQueue()
	c.cfg.Attached.TTS.Interrupt()

	if c.cfg.HoldTextFirst != "" && c.cfg.Attached.TTS != nil {
		// Best-effort; ignore err — Speak failure is OK during reconnect,
		// the call is already in a degraded state.
		_ = c.cfg.Attached.TTS.Speak(c.cfg.HoldTextFirst)
	}

	backoff := c.cfg.ReconnectInitialBackoff
	if backoff <= 0 {
		backoff = 1 * time.Second
	}
	const maxBackoff = 30 * time.Second

	for attempt := 1; attempt <= maxN; attempt++ {
		// Wait before redialing — but check cancellation/close every
		// 250 ms so a parallel Close() unblocks us promptly.
		deadline := time.Now().Add(backoff)
		for time.Now().Before(deadline) {
			if c.closed.Load() || (c.runCtx != nil && c.runCtx.Err() != nil) {
				return false
			}
			time.Sleep(250 * time.Millisecond)
		}

		logger.Info(fmt.Sprintf("[gw] call=%s reconnect attempt %d/%d", c.cfg.CallID, attempt, maxN))
		metrics.DialogReconnect(c.transportLabel(), "attempt")

		if err := c.dialAndSwap(); err != nil {
			c.log.Warn("voice/gateway reconnect failed",
				zap.Int("attempt", attempt),
				zap.String("call_id", c.cfg.CallID),
				zap.Error(err))
			metrics.DialogReconnect(c.transportLabel(), "fail")
			// Speak the retry message before the next attempt.
			if attempt < maxN && c.cfg.HoldTextRetry != "" && c.cfg.Attached.TTS != nil {
				_ = c.cfg.Attached.TTS.Speak(c.cfg.HoldTextRetry)
			}
			// Exponential backoff with cap.
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}
		logger.Info(fmt.Sprintf("[gw] call=%s reconnect ok on attempt %d", c.cfg.CallID, attempt))
		metrics.DialogReconnect(c.transportLabel(), "ok")
		return true
	}

	// All attempts exhausted — say goodbye and let the caller hang up.
	if c.cfg.HoldTextGiveUp != "" && c.cfg.Attached.TTS != nil {
		_ = c.cfg.Attached.TTS.Speak(c.cfg.HoldTextGiveUp)
	}
	return false
}

// dialAndSwap performs one WS dial and, on success, atomically replaces
// the live c.conn while holding c.writeM so an outgoing event from
// another goroutine cannot race on a half-swapped pointer. The previous
// conn is closed best-effort.
func (c *Client) dialAndSwap() error {
	dialURL, err := appendCallIDQuery(c.cfg.URL, c.cfg.CallID)
	if err != nil {
		return err
	}
	timeout := c.cfg.DialTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	dctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, resp, err := websocket.DefaultDialer.DialContext(dctx, dialURL, c.cfg.HandshakeHeaders)
	if err != nil {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		return fmt.Errorf("dial dialog ws %s (http=%d): %w", RedactDialogDialURL(dialURL), code, err)
	}
	c.writeM.Lock()
	old := c.conn
	c.conn = conn
	c.writeM.Unlock()
	if old != nil {
		_ = old.Close()
	}
	return nil
}

func (c *Client) dispatch(cmd Command) {
	switch cmd.Type {
	case CmdTTSSpeak:
		if c.cfg.Attached.TTS == nil || c.ttsQueue == nil {
			return
		}
		text := strings.TrimSpace(cmd.Text)
		if text == "" {
			return
		}
		// Non-blocking enqueue: if the worker is overwhelmed (rare), drop
		// the new utterance rather than stall the WS read loop.
		select {
		case c.ttsQueue <- ttsJob{text: text, utter: cmd.UtteranceID, meta: cmd.Meta}:
		default:
			c.log.Warn("voice/gateway tts queue full, dropping",
				zap.String("call_id", c.cfg.CallID),
				zap.String("utter", cmd.UtteranceID))
			_ = c.sendEvent(Event{
				Type: EvTTSEnded, CallID: c.cfg.CallID,
				UtteranceID: cmd.UtteranceID, OK: false,
			})
		}
	case CmdTTSInterrupt:
		if c.cfg.Attached.TTS == nil {
			return
		}
		// Drain pending utterances so a barge-in really stops the AI;
		// also cancel the in-flight Speak so frames stop hitting RTP.
		c.drainTTSQueue()
		c.cfg.Attached.TTS.Interrupt()
	case CmdHangup:
		if c.cfg.OnHangup != nil {
			c.cfg.OnHangup(cmd.Reason)
		}
	default:
		c.log.Warn("voice/gateway unknown command",
			zap.String("type", string(cmd.Type)),
			zap.String("call_id", c.cfg.CallID))
	}
}

// ttsWorker drains the TTS queue strictly in order, calling Speak
// synchronously so adjacent utterances never overlap on the wire. It exits
// when the queue is closed (Close was called) and either the queue drains
// naturally or runCtx is cancelled.
func (c *Client) ttsWorker() {
	defer c.ttsWg.Done()
	for {
		select {
		case <-c.runCtx.Done():
			return
		case job, ok := <-c.ttsQueue:
			if !ok {
				return
			}
			if c.runCtx.Err() != nil {
				return
			}
			start := time.Now()
			// Consume any pending ASR-final stamp so that E2EFirstByteMs
			// is only reported on the FIRST Speak after a given final.
			// Later intra-turn Speaks (from LLM sentence segmentation)
			// get asrFinalPtr == nil, which zeros out E2EFirstByteMs —
			// the intended semantic (only the first sentence counts as
			// the "AI started talking" moment from the user's ear).
			asrFinalPtr := c.asrFinalAt.Swap(nil)

			// Arm a one-shot first-frame hook on the TTS pipeline so we
			// can stamp the wall-clock at which real audio first leaves
			// the pipeline. A pointer (not a bare time.Time) lets us
			// distinguish "never fired" (Speak error / drained) from
			// "fired at T0" downstream.
			var firstByteAt atomic.Pointer[time.Time]
			c.cfg.Attached.TTS.ArmFirstFrameHook(func() {
				ts := time.Now()
				firstByteAt.Store(&ts)
			})

			logger.Info(fmt.Sprintf("[gw] call=%s tts speak begin utter=%s text=%q",
				c.cfg.CallID, job.utter, ellipsize(job.text, 40)))
			_ = c.sendEvent(Event{
				Type: EvTTSStarted, CallID: c.cfg.CallID, UtteranceID: job.utter,
			})
			if c.cfg.OnTTSStart != nil {
				c.cfg.OnTTSStart(job.utter, job.text)
			}
			err := c.cfg.Attached.TTS.Speak(job.text)
			// Defensive disarm: if Speak returned before any frame was
			// produced (synthesis error, Interrupt, ctx cancel) the hook
			// is still pending and would otherwise fire on the NEXT
			// Speak. ArmFirstFrameHook(nil) clears it.
			c.cfg.Attached.TTS.ArmFirstFrameHook(nil)
			dur := time.Since(start)

			ttsFirstMs := 0
			e2eFirstMs := 0
			if fbPtr := firstByteAt.Load(); fbPtr != nil {
				ttsFirstMs = int(fbPtr.Sub(start).Milliseconds())
				if asrFinalPtr != nil {
					// Clamp negatives to 0: if ASR finalized after the
					// TTS start (possible if a pre-canned greeting plays
					// while user still talks), reporting a negative
					// latency would be misleading.
					if d := fbPtr.Sub(*asrFinalPtr).Milliseconds(); d > 0 {
						e2eFirstMs = int(d)
					}
				}
			}

			_ = c.sendEvent(Event{
				Type: EvTTSEnded, CallID: c.cfg.CallID,
				UtteranceID: job.utter, OK: err == nil,
			})
			logger.Info(fmt.Sprintf("[gw] call=%s tts speak end   utter=%s ok=%v dur=%s ttf=%dms e2e=%dms",
				c.cfg.CallID, job.utter, err == nil, dur.Round(time.Millisecond),
				ttsFirstMs, e2eFirstMs))
			if err != nil {
				c.log.Warn("voice/gateway tts speak",
					zap.String("utter", job.utter), zap.Error(err))
			}
			// Notify the persistence observer (if any) AFTER the WS event
			// has been emitted, so a slow observer cannot delay the dialog
			// app's view of tts.ended. Errors and OK both surface here so
			// the observer can decide whether to record a turn.
			moreQueued := c.ttsQueue != nil && len(c.ttsQueue) > 0
			if c.cfg.OnTurn != nil {
				c.cfg.OnTurn(TurnEvent{
					UtteranceID:      job.utter,
					LLMText:          job.text,
					Meta:             job.meta,
					DurationMs:       int(dur.Milliseconds()),
					TTSFirstByteMs:   ttsFirstMs,
					E2EFirstByteMs:   e2eFirstMs,
					MoreSpeaksQueued: moreQueued,
					OK:               err == nil,
				})
			}
		}
	}
}

// drainTTSQueue consumes any pending jobs without playing them, emitting a
// tts.ended(ok=false) event for each so the dialog side can reconcile state.
func (c *Client) drainTTSQueue() {
	if c.ttsQueue == nil {
		return
	}
	for {
		select {
		case job := <-c.ttsQueue:
			_ = c.sendEvent(Event{
				Type: EvTTSEnded, CallID: c.cfg.CallID,
				UtteranceID: job.utter, OK: false,
			})
		default:
			return
		}
	}
}

func ellipsize(s string, max int) string {
	if max <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// MergeDialogPayloadQuery parses raw and sets query key "payload" to the
// given JSON bytes (typically a JSON object from the WebRTC offer body).
// The dialog plane parses it (e.g. SoulNexus /ws/call overlays apiKey etc.).
func MergeDialogPayloadQuery(raw string, payload []byte) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty dialog URL")
	}
	ps := strings.TrimSpace(string(payload))
	if len(payload) == 0 || ps == "" || ps == "null" {
		return "", fmt.Errorf("empty payload")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}
	q := u.Query()
	q.Set("payload", string(payload))
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// MergeDialogQueryParams parses raw and sets query keys apiKey, apiSecret,
// and agentId when the corresponding arguments are non-empty after TrimSpace.
// Existing query parameters are preserved; empty arguments do not remove
// keys already present (e.g. values merged from env at process startup).
func MergeDialogQueryParams(raw, apiKey, apiSecret, agentID string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty dialog URL")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}
	q := u.Query()
	if v := strings.TrimSpace(apiKey); v != "" {
		q.Set("apiKey", v)
	}
	if v := strings.TrimSpace(apiSecret); v != "" {
		q.Set("apiSecret", v)
	}
	if v := strings.TrimSpace(agentID); v != "" {
		q.Set("agentId", v)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func appendCallIDQuery(raw, callID string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}
	q := u.Query()
	q.Set("call_id", callID)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// RedactDialogDialURL returns u with sensitive query parameters replaced so
// logs and wrapped errors do not echo api credentials.
func RedactDialogDialURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return "<invalid-dialog-url>"
	}
	q := parsed.Query()
	if q.Get("apiKey") != "" {
		q.Set("apiKey", "***")
	}
	if q.Get("apiSecret") != "" {
		q.Set("apiSecret", "***")
	}
	if q.Get("payload") != "" {
		q.Set("payload", "***")
	}
	parsed.RawQuery = q.Encode()
	return parsed.String()
}
