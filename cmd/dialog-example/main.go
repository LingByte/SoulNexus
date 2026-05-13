// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// dialog-example — minimal dialog-plane WebSocket server with LLM streaming.
//
// VoiceServer dials this server (one WS per inbound SIP call) and streams
// asr.partial / asr.final / dtmf / call.* events. On every asr.final this
// example calls a configured LLM via QueryStream, segments tokens into
// utterances using pkg/voice/tts.Segmenter, and pushes one tts.speak command
// per utterance back to VoiceServer for low-latency playback.
//
// Configuration (.env or process env):
//
//	LLM_PROVIDER  - openai | alibaba | coze | ollama  (default openai)
//	LLM_APIKEY    - provider API key
//	LLM_APP_ID    - provider-specific second arg:
//	                  alibaba -> AppID
//	                  coze    -> Bot ID  (or {"botId":"...","userId":"..."} JSON)
//	                  openai  -> base URL (default https://api.openai.com/v1)
//	                  ollama  -> base URL
//	LLM_MODEL     - chat model name (default per provider)
//	LLM_SYSTEM    - system prompt (optional)
//
// Usage
//
//	go run ./cmd/dialog-example -addr 127.0.0.1:7080 -path /ws/call
//
// On voice binary (SoulNexus repo root):
//
//	go run ./cmd/voice \
//	  -sip 192.168.3.69:5060 -local-ip 192.168.3.69 \
//	  -rtp-start 31000 -rtp-end 31010 \
//	  -dialog-ws ws://127.0.0.1:7080/ws/call -record
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/llm"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/utils"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

func main() {
	// Best-effort .env load so utils.GetEnv sees LLM_* keys when running locally.
	_ = godotenv.Overload(".env")

	addr := flag.String("addr", "127.0.0.1:7080", "HTTP listen address")
	path := flag.String("path", "/ws/call", "WebSocket route")
	fallback := flag.String("fallback", "您好，我现在没法回答这个问题，请稍后再试。",
		"Spoken back to the caller when the LLM call fails")
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Usage of dialog-example:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	cfg := loadLLMConfig()
	if cfg.APIKey == "" {
		log.Printf("[warn] LLM_APIKEY is empty — LLM calls will fail. Set LLM_PROVIDER/LLM_APIKEY/LLM_APP_ID in .env or env.")
	}
	log.Printf("dialog-example LLM config: provider=%s model=%s app_id=%q",
		cfg.Provider, cfg.Model, cfg.AppID)

	mux := http.NewServeMux()
	mux.HandleFunc(*path, func(w http.ResponseWriter, r *http.Request) {
		handleCall(w, r, cfg, *fallback)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("dialog-example: connect via " + *path + "?call_id=...\n"))
	})

	log.Printf("dialog-example listening: ws://%s%s", *addr, *path)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

// ---------- LLM config ------------------------------------------------------

type llmConfig struct {
	Provider     string
	APIKey       string
	AppID        string // 2nd factory arg: alibaba=AppID, coze=BotID, openai/ollama=baseURL
	Model        string
	SystemPrompt string
}

func loadLLMConfig() llmConfig {
	cfg := llmConfig{
		Provider:     strings.TrimSpace(utils.GetEnv("LLM_PROVIDER")),
		APIKey:       strings.TrimSpace(utils.GetEnv("LLM_APIKEY")),
		AppID:        strings.TrimSpace(utils.GetEnv("LLM_APP_ID")),
		Model:        strings.TrimSpace(utils.GetEnv("LLM_MODEL")),
		SystemPrompt: strings.TrimSpace(utils.GetEnv("LLM_SYSTEM")),
	}
	if cfg.Provider == "" {
		cfg.Provider = "openai"
	}
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = "你是一个语音电话助手，请用简短、自然、口语化的中文回答用户问题，避免使用 Markdown 标记。"
	}
	if cfg.Model == "" {
		switch strings.ToLower(cfg.Provider) {
		case "alibaba":
			cfg.Model = "qwen-plus"
		case "coze":
			cfg.Model = "" // Coze model is bot-side
		case "ollama":
			cfg.Model = "qwen2.5:7b"
		default:
			cfg.Model = "gpt-4o-mini"
		}
	}
	return cfg
}

// ---------- WS handler ------------------------------------------------------

var upgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool { return true },
}

func handleCall(w http.ResponseWriter, r *http.Request, cfg llmConfig, fallback string) {
	callID := r.URL.Query().Get("call_id")
	if callID == "" {
		http.Error(w, "missing ?call_id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[call=%s] upgrade: %v", callID, err)
		return
	}
	log.Printf("[call=%s] dialog WS upgraded from %s", callID, r.RemoteAddr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Per-call provider so each call has independent conversation history.
	provider, err := llm.NewLLMProvider(ctx, cfg.Provider, cfg.APIKey, cfg.AppID, cfg.SystemPrompt)
	if err != nil {
		log.Printf("[call=%s] llm init: %v", callID, err)
		_ = conn.Close()
		return
	}
	defer provider.Hangup()

	c := &callConn{
		callID:   callID,
		conn:     conn,
		cfg:      cfg,
		fallback: fallback,
		provider: provider,
	}

	defer func() {
		_ = conn.Close()
		log.Printf("[call=%s] dialog WS closed", callID)
	}()

	for {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[call=%s] read end: %v", callID, err)
			return
		}
		var ev gateway.Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			log.Printf("[call=%s] bad event: %v raw=%s", callID, err, string(raw))
			continue
		}
		c.dispatch(ctx, ev)
	}
}

// ---------- per-call state --------------------------------------------------

type callConn struct {
	callID   string
	conn     *websocket.Conn
	cfg      llmConfig
	fallback string
	provider llm.LLMProvider

	writeM sync.Mutex

	turnSeq    atomic.Int64 // monotonic turn ID for utterance prefixing
	llmBusy    atomic.Bool  // a turn is currently producing tokens
	lastFinal  string       // last emitted ASR final (dedupe)
	lastFinalM sync.Mutex
}

func (c *callConn) dispatch(ctx context.Context, ev gateway.Event) {
	switch ev.Type {
	case gateway.EvCallStarted:
		log.Printf("[call=%s] started: from=%s to=%s codec=%s pcm_hz=%d",
			c.callID, ev.From, ev.To, ev.Codec, ev.PCMHz)

	case gateway.EvCallEnded:
		log.Printf("[call=%s] ended: %s", c.callID, ev.Reason)

	case gateway.EvASRPartial:
		log.Printf("[call=%s] asr-partial: %s", c.callID, ev.Text)

	case gateway.EvASRFinal:
		text := strings.TrimSpace(ev.Text)
		if text == "" {
			return
		}
		c.lastFinalM.Lock()
		if text == c.lastFinal {
			c.lastFinalM.Unlock()
			return
		}
		c.lastFinal = text
		c.lastFinalM.Unlock()

		log.Printf("[call=%s] asr-final  : %s", c.callID, text)
		go c.runTurn(ctx, text)

	case gateway.EvASRError:
		log.Printf("[call=%s] asr-error fatal=%v: %s", c.callID, ev.Fatal, ev.Message)

	case gateway.EvDTMF:
		log.Printf("[call=%s] dtmf %s end=%v", c.callID, ev.Digit, ev.End)

	case gateway.EvTTSStarted:
		log.Printf("[call=%s] tts-started utt=%s", c.callID, ev.UtteranceID)

	case gateway.EvTTSEnded:
		log.Printf("[call=%s] tts-ended  utt=%s ok=%v", c.callID, ev.UtteranceID, ev.OK)

	default:
		log.Printf("[call=%s] unknown event type=%q", c.callID, ev.Type)
	}
}

// runTurn drives one ASR-final → LLM stream → segmented TTS turn.
//
// Latency strategy (key design decision)
//
// We use streaming LLM (`QueryStream`) and pipe each delta through a sentence
// segmenter. As soon as the segmenter sees a sentence boundary (`。！？\n`)
// we fire one `tts.speak` for that sentence — without waiting for the LLM to
// finish. The downstream gateway client serialises tts.speak through a single
// worker, so playback order is preserved while *synthesis* of sentence N+1
// happens in parallel with *playback* of sentence N. For multi-sentence
// replies this overlaps LLM tail latency with TTS playback and saves several
// seconds of perceived response time.
//
// Provider stream-shape compatibility
//
// LLM providers disagree on what each callback chunk means:
//
//   - OpenAI / Ollama / Coze: each `piece` is a delta (new token text).
//   - Alibaba Bailian app:    each `piece` is the cumulative `message` field
//     parsed from a JSON-wrapped response. The same prefix is re-emitted on
//     every SSE event until the JSON closes.
//
// To stay vendor-agnostic, `streamView` below converts both shapes into a
// canonical "growing string" view and emits true deltas to the segmenter.
// JSON unwrapping for Bailian already happens in the alibaba_provider; we
// also defensively unwrap any `{"message":"..."}` envelope here.
//
// Barge-in: a new asr.final while a turn is in flight pre-empts the prior
// turn (TTS interrupt + LLM cancel + drained pending utterances).
func (c *callConn) runTurn(ctx context.Context, userText string) {
	if c.llmBusy.Load() {
		c.sendCommand(gateway.Command{Type: gateway.CmdTTSInterrupt})
		c.provider.Interrupt()
	}
	c.llmBusy.Store(true)
	defer c.llmBusy.Store(false)

	turn := c.turnSeq.Add(1)
	utterIdx := 0
	t0 := time.Now()
	var firstByteMs int

	speak := func(text string) {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		utterIdx++
		meta := &gateway.CommandMeta{
			LLMModel:   c.cfg.Model,
			LLMFirstMs: firstByteMs,
			LLMWallMs:  int(time.Since(t0).Milliseconds()),
			UserText:   userText,
		}
		c.sendCommand(gateway.Command{
			Type:        gateway.CmdTTSSpeak,
			Text:        text,
			UtteranceID: fmt.Sprintf("t%d-u%d", turn, utterIdx),
			Meta:        meta,
		})
	}

	seg := tts.NewSegmenter(tts.SegmenterConfig{}, func(text string, _ bool) {
		speak(text)
	})
	defer seg.Reset()

	var sv streamView
	startedTTS := false

	_, err := c.provider.QueryStream(userText, llm.QueryOptions{
		Model:  c.cfg.Model,
		Stream: true,
	}, func(piece string, isComplete bool) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Allow JSON-wrapped piece payloads — the alibaba provider already
		// extracts `message` once parseable, but defensive unwrap protects
		// against future providers that pass JSON directly.
		if msg, ok := tryExtractMessage(piece); ok {
			piece = msg
		}
		if delta := sv.observe(piece); delta != "" {
			if !startedTTS {
				startedTTS = true
				firstByteMs = int(time.Since(t0).Milliseconds())
				log.Printf("[call=%s] llm first-byte=%dms", c.callID, firstByteMs)
			}
			seg.Push(delta)
		}
		if isComplete {
			seg.Complete()
		}
		return nil
	})
	if err != nil {
		log.Printf("[call=%s] llm stream failed: %v (fullSeen=%q)",
			c.callID, err, ellipsize(sv.seen, 200))
		// Drain anything buffered in the segmenter before fallback so a
		// partial reply still gets played. Then speak the fallback line.
		seg.Complete()
		if !startedTTS && c.fallback != "" {
			speak(c.fallback)
		}
		return
	}
	// Final flush in case the LLM ended without a sentence terminator.
	seg.Complete()
	if !startedTTS {
		// No content at all — speak fallback so the call doesn't die silent.
		log.Printf("[call=%s] llm produced no speakable text; using fallback", c.callID)
		if c.fallback != "" {
			speak(c.fallback)
		}
	}
}

// streamView normalizes a streaming LLM source (delta or cumulative) into a
// canonical "growing string" perspective and emits per-call deltas.
//
// On each `observe(piece)`:
//
//   - if `piece` extends what we have seen (cumulative producer like
//     Alibaba), the new suffix is the delta;
//   - otherwise (delta producer like OpenAI), we append the piece to seen
//     and treat the whole piece as the delta.
//
// Duplicate cumulative emissions (same text re-sent) collapse to "" so we
// never push duplicates into the segmenter.
type streamView struct {
	seen string
}

func (sv *streamView) observe(piece string) string {
	if piece == "" {
		return ""
	}
	if strings.HasPrefix(piece, sv.seen) {
		if len(piece) == len(sv.seen) {
			return ""
		}
		delta := piece[len(sv.seen):]
		sv.seen = piece
		return delta
	}
	sv.seen += piece
	return piece
}

// tryExtractMessage pulls the `message` field out of a JSON object embedded
// in raw. Returns (msg, true) only if raw appears to contain a *closed*
// JSON object with a non-empty message — partial fragments fail the parse
// and fall through, which is what we want during streaming.
func tryExtractMessage(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", false
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return "", false
	}
	var p struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(s[start:end+1]), &p); err != nil {
		return "", false
	}
	msg := strings.TrimSpace(p.Message)
	if msg == "" {
		return "", false
	}
	return msg, true
}

// ellipsize truncates s to max runes, appending … if it was clipped. Used
// only for log preview formatting, never for the actual TTS payload.
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

// sendCommand serializes one Command and writes it to the WS. CallID is
// stamped automatically. Errors are logged and dropped (peer is gone).
func (c *callConn) sendCommand(cmd gateway.Command) {
	cmd.CallID = c.callID
	data, err := json.Marshal(cmd)
	if err != nil {
		log.Printf("[call=%s] marshal cmd: %v", c.callID, err)
		return
	}
	c.writeM.Lock()
	defer c.writeM.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("[call=%s] write cmd %s: %v", c.callID, cmd.Type, err)
	}
}
