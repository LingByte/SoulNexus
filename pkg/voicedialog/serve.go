// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package voicedialog implements the dialog-plane WebSocket server consumed by
// cmd/voice when it dials SoulNexus (or compatible) /ws/call. It speaks the
// pkg/voiceserver/voice/gateway event protocol: ASR/DTMF events in,
// tts.speak / tts.interrupt commands out, with LLM streaming and utterance
// segmentation mirroring cmd/dialog-example.
//
// It lives under pkg/ so HTTP handlers import a stable path without Gin types.
package voicedialog

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/gateway"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"
	"github.com/gorilla/websocket"
)

// Config holds per-call LLM settings (typically from UserCredential + Agent).
type Config struct {
	Provider     string
	APIKey       string
	APIURL       string // OpenAI base URL, Alibaba AppID, Coze bot JSON or bot id, Ollama base URL
	Model        string
	SystemPrompt string
	MaxTokens    int     // 0 = unlimited (omit in LLM request)
	Temperature  float32 // 0 = caller should apply default (e.g. 0.7)
}

// Serve blocks until the WebSocket read loop ends or ctx is cancelled.
// The caller must perform HTTP auth and WebSocket upgrade; Serve does not close conn.
func Serve(ctx context.Context, conn *websocket.Conn, callID string, cfg Config, fallback string) {
	logger.Info(fmt.Sprintf("[call=%s] dialog-plane WS from %s", callID, conn.RemoteAddr()))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	provider, err := llm.NewLLMProvider(ctx, cfg.Provider, cfg.APIKey, cfg.APIURL, cfg.SystemPrompt)
	if err != nil {
		logger.Info(fmt.Sprintf("[call=%s] llm init: %v", callID, err))
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

	for {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			logger.Info(fmt.Sprintf("[call=%s] read end: %v", callID, err))
			return
		}
		var ev gateway.Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			logger.Info(fmt.Sprintf("[call=%s] bad event: %v raw=%s", callID, err, string(raw)))
			continue
		}
		c.dispatch(ctx, ev)
	}
}

type callConn struct {
	callID   string
	conn     *websocket.Conn
	cfg      Config
	fallback string
	provider llm.LLMHandler

	writeM sync.Mutex

	turnSeq    atomic.Int64
	llmBusy    atomic.Bool
	lastFinal  string
	lastFinalM sync.Mutex
}

func (c *callConn) dispatch(ctx context.Context, ev gateway.Event) {
	switch ev.Type {
	case gateway.EvCallStarted:
		logger.Info(fmt.Sprintf("[call=%s] started: from=%s to=%s codec=%s pcm_hz=%d",
			c.callID, ev.From, ev.To, ev.Codec, ev.PCMHz))

	case gateway.EvCallEnded:
		logger.Info(fmt.Sprintf("[call=%s] ended: %s", c.callID, ev.Reason))

	case gateway.EvASRPartial:
		logger.Info(fmt.Sprintf("[call=%s] asr-partial: %s", c.callID, ev.Text))

	case gateway.EvASRFinal:
		text := strings.TrimSpace(ev.Text)
		if text == "" {
			return
		}
		if isNoiseOnlyASRFinal(text) {
			logger.Info(fmt.Sprintf("[call=%s] asr-final ignored (noise-only): %q", c.callID, text))
			return
		}
		c.lastFinalM.Lock()
		if text == c.lastFinal {
			c.lastFinalM.Unlock()
			return
		}
		c.lastFinal = text
		c.lastFinalM.Unlock()

		logger.Info(fmt.Sprintf("[call=%s] asr-final  : %s", c.callID, text))
		go c.runTurn(ctx, text)

	case gateway.EvTTSInterrupt:
		logger.Info(fmt.Sprintf("[call=%s] tts-interrupt (playback cut by voiceserver)", c.callID))

	case gateway.EvASRError:
		logger.Info(fmt.Sprintf("[call=%s] asr-error fatal=%v: %s", c.callID, ev.Fatal, ev.Message))

	case gateway.EvDTMF:
		logger.Info(fmt.Sprintf("[call=%s] dtmf %s end=%v", c.callID, ev.Digit, ev.End))

	case gateway.EvTTSStarted:
		logger.Info(fmt.Sprintf("[call=%s] tts-started utt=%s", c.callID, ev.UtteranceID))

	case gateway.EvTTSEnded:
		logger.Info(fmt.Sprintf("[call=%s] tts-ended  utt=%s ok=%v", c.callID, ev.UtteranceID, ev.OK))

	default:
		logger.Info(fmt.Sprintf("[call=%s] unknown event type=%q", c.callID, ev.Type))
	}
}

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

	seg := tts.NewSegmenter(tts.SegmenterConfig{
		// Balance: pure sentence-only + huge MaxRunes waits too long for the
		// first tts.speak and ships multi-sentence blobs (slow TTF, long
		// monologue). Defaults (MinRunes 12 / MaxRunes 40) over-split on commas.
		// Here: clause flush only after a substantial clause, hard-cap length,
		// and modest idle so streaming still chunks without "语义碎屑".
		OmitClauseFlushes: false,
		SentenceEnders:    []string{"。", "！", "？", ".", "!", "?", "\n"},
		// Chinese models often use … / …… instead of commas; treat as soft
		// boundaries so we do not ship one 8s tts.speak before a period.
		ClauseMarkers: []string{"，", "、", "；", "：", "—", "……", "…"},
		MinRunes:      12,
		MaxRunes:      32,
		IdleFlush:     200 * time.Millisecond,
	}, func(text string, _ bool) {
		speak(text)
	})
	defer seg.Reset()

	var sv streamView
	startedTTS := false

	temp := c.cfg.Temperature
	if temp <= 0 {
		temp = 0.7
	}
	opts := &llm.QueryOptions{
		Model:       c.cfg.Model,
		Temperature: temp,
		MaxTokens:   c.cfg.MaxTokens,
	}

	_, err := c.provider.QueryStream(userText, opts, func(piece string, isComplete bool) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if msg, ok := tryExtractMessage(piece); ok {
			piece = msg
		}
		if delta := sv.observe(piece); delta != "" {
			if !startedTTS {
				startedTTS = true
				firstByteMs = int(time.Since(t0).Milliseconds())
				logger.Info(fmt.Sprintf("[call=%s] llm first-byte=%dms", c.callID, firstByteMs))
			}
			seg.Push(delta)
		}
		if isComplete {
			seg.Complete()
		}
		return nil
	})
	if err != nil {
		logger.Info(fmt.Sprintf("[call=%s] llm stream failed: %v (fullSeen=%q)",
			c.callID, err, ellipsize(sv.seen, 200)))
		seg.Complete()
		if !startedTTS && c.fallback != "" {
			speak(c.fallback)
		}
		return
	}
	seg.Complete()
	if !startedTTS {
		logger.Info(fmt.Sprintf("[call=%s] llm produced no speakable text; using fallback", c.callID))
		if c.fallback != "" {
			speak(c.fallback)
		}
	}
}

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

// DefaultModelForProvider picks a chat model when the agent leaves llmModel empty.
func DefaultModelForProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "alibaba":
		return "qwen-plus"
	case "coze":
		return ""
	case "ollama":
		return "qwen2.5:7b"
	default:
		return "gpt-4o-mini"
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

func (c *callConn) sendCommand(cmd gateway.Command) {
	cmd.CallID = c.callID
	data, err := json.Marshal(cmd)
	if err != nil {
		logger.Info(fmt.Sprintf("[call=%s] marshal cmd: %v", c.callID, err))
		return
	}
	c.writeM.Lock()
	defer c.writeM.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		logger.Info(fmt.Sprintf("[call=%s] write cmd %s: %v", c.callID, cmd.Type, err))
	}
}
