package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Aliyun DashScope Qwen-TTS realtime (WebSocket).
//
// Why a dedicated client (no Go SDK):
// ------------------------------------
// DashScope ships an official Python SDK (`dashscope.audio.qwen_tts_realtime`)
// but no Go SDK for the realtime endpoint. The wire protocol is an
// OpenAI-realtime-style JSON-event stream over a single WebSocket and is
// stable / publicly documented at:
//
//	https://www.alibabacloud.com/help/en/model-studio/qwen-tts-realtime
//
// Latency profile (measured against `qwen3-tts-flash-realtime`):
//   - First audio packet ≈ 150–300 ms after `session.finish` in server_commit
//     mode (significantly faster than qcloud HTTP TTS and competitive with
//     qcloud WS).
//   - PCM16LE @ 24 kHz mono; the upstream `WithSynthesis` player resamples to
//     the bridge sample-rate automatically.
//
// Lifecycle:
//   - Each `SynthesizeStream` / `Synthesize` call opens one WS, configures the
//     session, pushes all text in one or more `input_text_buffer.append`
//     events, sends `session.finish`, then drains audio deltas until
//     `response.done` (or `session.finished`).
//   - ctx cancellation closes the WS so the read loop unblocks immediately
//     (used by SIP barge-in).

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
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// AliyunTTSConfig is the configuration for the DashScope Qwen-TTS realtime
// synthesizer. Only APIKey is strictly required — the rest fall back to
// vendor defaults that match `qwen3-tts-flash-realtime`.
type AliyunTTSConfig struct {
	APIKey string `json:"api_key" yaml:"api_key" env:"DASHSCOPE_API_KEY"`
	// BaseURL overrides the WebSocket endpoint. When empty, defaults to the
	// Beijing region endpoint. For Singapore region set:
	//   wss://dashscope-intl.aliyuncs.com/api-ws/v1/realtime
	BaseURL string `json:"base_url" yaml:"base_url"`
	// Model defaults to "qwen3-tts-flash-realtime". The instruct variant
	// "qwen3-tts-instruct-flash-realtime" supports instruction-controlled
	// style/prosody.
	Model string `json:"model" yaml:"model" default:"qwen3-tts-flash-realtime"`
	// Voice — see vendor docs for the supported list. "Cherry" is the
	// canonical zh-CN female default.
	Voice string `json:"voice" yaml:"voice" default:"Cherry"`
	// LanguageType: Chinese | English | Auto | Japanese | Korean | ...
	LanguageType string `json:"language_type" yaml:"language_type" default:"Auto"`
	// Mode: "server_commit" (default; server auto-detects sentence
	// boundaries) or "commit" (caller drives commits via SynthesizeStream
	// boundaries). Realtime SIP traffic should stick to server_commit.
	Mode string `json:"mode" yaml:"mode" default:"server_commit"`
	// SampleRate of the returned PCM. The vendor supports 16000/22050/24000;
	// 24000 is the lowest-latency path.
	SampleRate int `json:"sample_rate" yaml:"sample_rate" default:"24000"`
	// Channels / BitDepth are fixed by the vendor (mono / 16-bit) but kept
	// here so they propagate via Format() into the upstream player.
	Channels      int    `json:"channels" yaml:"channels" default:"1"`
	BitDepth      int    `json:"bit_depth" yaml:"bit_depth" default:"16"`
	FrameDuration string `json:"frame_duration" yaml:"frame_duration" default:"20ms"`
	// DialTimeout for the initial WebSocket handshake. 0 → 10 s.
	DialTimeoutMs int `json:"dial_timeout_ms" yaml:"dial_timeout_ms"`
	// Instructions / OptimizeInstructions enable RFC-style style control.
	// Only honored by `qwen3-tts-instruct-flash-realtime`.
	Instructions         string `json:"instructions" yaml:"instructions"`
	OptimizeInstructions bool   `json:"optimize_instructions" yaml:"optimize_instructions"`
}

// NewAliyunTTSConfig builds a config populated with vendor defaults. APIKey
// falls back to the DASHSCOPE_API_KEY environment variable when blank.
func NewAliyunTTSConfig(apiKey string) AliyunTTSConfig {
	cfg := AliyunTTSConfig{
		APIKey:        apiKey,
		BaseURL:       aliyunDefaultBaseURL,
		Model:         "qwen3-tts-flash-realtime",
		Voice:         "Cherry",
		LanguageType:  "Auto",
		Mode:          "server_commit",
		SampleRate:    24000,
		Channels:      1,
		BitDepth:      16,
		FrameDuration: "20ms",
		DialTimeoutMs: 10000,
	}
	if cfg.APIKey == "" {
		cfg.APIKey = utils.GetEnv("DASHSCOPE_API_KEY")
	}
	return cfg
}

const aliyunDefaultBaseURL = "wss://dashscope.aliyuncs.com/api-ws/v1/realtime"

// AliyunService implements SynthesisService against DashScope Qwen-TTS realtime.
type AliyunService struct {
	mu  sync.Mutex
	opt AliyunTTSConfig
}

// NewAliyunService creates a service handle. The WebSocket is opened lazily
// per Synthesize / SynthesizeStream call (matches the QCloud WS adapter and
// keeps barge-in semantics simple).
func NewAliyunService(opt AliyunTTSConfig) *AliyunService {
	if opt.BaseURL == "" {
		opt.BaseURL = aliyunDefaultBaseURL
	}
	if opt.Model == "" {
		opt.Model = "qwen3-tts-flash-realtime"
	}
	if opt.Voice == "" {
		opt.Voice = "Cherry"
	}
	if opt.LanguageType == "" {
		opt.LanguageType = "Auto"
	}
	if opt.Mode == "" {
		opt.Mode = "server_commit"
	}
	if opt.SampleRate == 0 {
		opt.SampleRate = 24000
	}
	if opt.Channels == 0 {
		opt.Channels = 1
	}
	if opt.BitDepth == 0 {
		opt.BitDepth = 16
	}
	if opt.FrameDuration == "" {
		opt.FrameDuration = "20ms"
	}
	if opt.DialTimeoutMs <= 0 {
		opt.DialTimeoutMs = 10000
	}
	if opt.APIKey == "" {
		opt.APIKey = utils.GetEnv("DASHSCOPE_API_KEY")
	}
	return &AliyunService{opt: opt}
}

func (a *AliyunService) Provider() TTSProvider {
	return ProviderAliyun
}

func (a *AliyunService) Format() media.StreamFormat {
	a.mu.Lock()
	defer a.mu.Unlock()
	return media.StreamFormat{
		SampleRate:    a.opt.SampleRate,
		BitDepth:      a.opt.BitDepth,
		Channels:      a.opt.Channels,
		FrameDuration: utils.NormalizeFramePeriod(a.opt.FrameDuration),
	}
}

func (a *AliyunService) CacheKey(text string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("aliyun.tts-%s-%s-%s-%d-%s.pcm",
		a.opt.Model, a.opt.Voice, a.opt.LanguageType, a.opt.SampleRate, digest)
}

// Synthesize implements SynthesisService. Audio chunks (PCM16LE) are
// forwarded to handler.OnMessage as they arrive.
func (a *AliyunService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	if handler == nil {
		return fmt.Errorf("aliyun tts: nil handler")
	}
	return a.SynthesizeStream(ctx, text, func(pcm []byte) error {
		if len(pcm) == 0 {
			return nil
		}
		handler.OnMessage(pcm)
		return nil
	})
}

// SynthesizeStream synthesizes `text` and pushes PCM16LE @ Format().SampleRate
// chunks into `callback` as they arrive. The signature mirrors
// QCloudService.SynthesizeStream so it slots into the SIP voicedialog adapter.
func (a *AliyunService) SynthesizeStream(ctx context.Context, text string, callback func(pcm []byte) error) error {
	if a == nil {
		return fmt.Errorf("aliyun tts: nil service")
	}
	if callback == nil {
		return fmt.Errorf("aliyun tts: nil callback")
	}
	a.mu.Lock()
	opt := a.opt
	a.mu.Unlock()

	if strings.TrimSpace(opt.APIKey) == "" {
		return fmt.Errorf("aliyun tts: DASHSCOPE_API_KEY is required")
	}
	if strings.TrimSpace(text) == "" {
		return nil
	}

	wsURL, err := buildAliyunWSURL(opt.BaseURL, opt.Model)
	if err != nil {
		return fmt.Errorf("aliyun tts: %w", err)
	}

	dialer := websocket.DefaultDialer
	if opt.DialTimeoutMs > 0 {
		d := *websocket.DefaultDialer
		d.HandshakeTimeout = time.Duration(opt.DialTimeoutMs) * time.Millisecond
		dialer = &d
	}
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+opt.APIKey)

	conn, resp, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		status := -1
		if resp != nil {
			status = resp.StatusCode
			resp.Body.Close()
		}
		return fmt.Errorf("aliyun tts: dial %s (status=%d): %w", wsURL, status, err)
	}
	defer conn.Close()

	// Watcher: ctx cancellation → close conn so ReadMessage returns.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	if err := sendAliyunSessionUpdate(conn, opt); err != nil {
		return fmt.Errorf("aliyun tts: session.update: %w", err)
	}
	if err := sendAliyunEvent(conn, map[string]any{
		"type": "input_text_buffer.append",
		"text": text,
	}); err != nil {
		return fmt.Errorf("aliyun tts: append text: %w", err)
	}
	// In commit mode the client must trigger synthesis explicitly. In
	// server_commit mode the server begins streaming as soon as a sentence
	// boundary is detected; session.finish below flushes any tail text.
	if strings.EqualFold(opt.Mode, "commit") {
		if err := sendAliyunEvent(conn, map[string]any{
			"type": "input_text_buffer.commit",
		}); err != nil {
			return fmt.Errorf("aliyun tts: commit: %w", err)
		}
	}
	if err := sendAliyunEvent(conn, map[string]any{"type": "session.finish"}); err != nil {
		return fmt.Errorf("aliyun tts: session.finish: %w", err)
	}

	var cbErr error
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Normal close after response.done / session.finished is fine.
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return cbErr
			}
			return fmt.Errorf("aliyun tts: read: %w", err)
		}
		evt := aliyunEvent{}
		if err := json.Unmarshal(raw, &evt); err != nil {
			// Unexpected non-JSON frame — log and keep reading; the vendor
			// can in theory add binary frames in the future.
			logrus.WithError(err).WithField("vendor", "aliyun_tts").
				Warn("aliyun tts: non-json frame")
			continue
		}
		switch evt.Type {
		case "response.audio.delta":
			if evt.Delta == "" || cbErr != nil {
				continue
			}
			pcm, decErr := base64.StdEncoding.DecodeString(evt.Delta)
			if decErr != nil {
				logrus.WithError(decErr).Warn("aliyun tts: bad base64 delta")
				continue
			}
			if err := callback(pcm); err != nil {
				// Surface the first callback error but keep the WS open
				// so we can read response.done cleanly.
				cbErr = err
			}
		case "response.done", "session.finished":
			// Drain done event then return; the server typically follows
			// session.finished with a normal close.
			if evt.Type == "session.finished" {
				return cbErr
			}
		case "error":
			msg := "unknown error"
			if evt.Error != nil {
				if evt.Error.Message != "" {
					msg = evt.Error.Message
				} else if evt.Error.Code != "" {
					msg = evt.Error.Code
				}
			}
			return fmt.Errorf("aliyun tts: server error: %s", msg)
		}
	}
}

// Close is a no-op — connections are scoped to each Synthesize call.
func (a *AliyunService) Close() error { return nil }

// --- protocol helpers --------------------------------------------------------

func buildAliyunWSURL(base, model string) (string, error) {
	if base == "" {
		base = aliyunDefaultBaseURL
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse base_url: %w", err)
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return "", fmt.Errorf("base_url must be ws:// or wss://, got %q", u.Scheme)
	}
	q := u.Query()
	if q.Get("model") == "" && model != "" {
		q.Set("model", model)
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func sendAliyunSessionUpdate(conn *websocket.Conn, opt AliyunTTSConfig) error {
	session := map[string]any{
		"mode":            opt.Mode,
		"voice":           opt.Voice,
		"language_type":   opt.LanguageType,
		"response_format": "pcm",
		"sample_rate":     opt.SampleRate,
	}
	if strings.TrimSpace(opt.Instructions) != "" {
		session["instructions"] = opt.Instructions
		session["optimize_instructions"] = opt.OptimizeInstructions
	}
	return sendAliyunEvent(conn, map[string]any{
		"type":    "session.update",
		"session": session,
	})
}

func sendAliyunEvent(conn *websocket.Conn, event map[string]any) error {
	// Vendor expects each event to carry a client-side event_id; we mint a
	// millisecond-resolution one which is enough to disambiguate logs.
	event["event_id"] = fmt.Sprintf("event_%d", time.Now().UnixMilli())
	buf, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, buf)
}

type aliyunEvent struct {
	Type    string           `json:"type"`
	EventID string           `json:"event_id,omitempty"`
	Delta   string           `json:"delta,omitempty"`
	Error   *aliyunErrorBody `json:"error,omitempty"`
}

type aliyunErrorBody struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Compile-time guards.
var (
	_ SynthesisService = (*AliyunService)(nil)
	_ interface {
		SynthesizeStream(context.Context, string, func([]byte) error) error
	} = (*AliyunService)(nil)
	_ = errors.New
)
