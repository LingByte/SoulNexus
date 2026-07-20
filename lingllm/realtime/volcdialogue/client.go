package volcdialogue

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Volcengine / 豆包端到端实时语音大模型 (Realtime Dialogue API).
//
//	wss://openspeech.bytedance.com/api/v3/realtime/dialogue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/lingllm/realtime"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	ProviderSlug      = "volcengine_dialogue"
	defaultWSURL      = "wss://openspeech.bytedance.com/api/v3/realtime/dialogue"
	defaultResourceID = "volc.speech.dialog"
	defaultAppKey     = "PlgvMymc7f3tQnJ6"
	defaultModelO     = "1.2.1.1" // O2.0
	defaultModelSC    = "2.2.0.0" // SC2.0
	defaultSpeaker    = "zh_female_vv_jupiter_bigtts"
	defaultDialMs     = 15000
	defaultSendBuf    = 32
)

func init() {
	realtime.Register(New, ProviderSlug, "volc_realtime", "doubao_realtime", "volcengine_realtime")
}

// Config is the tenant credential JSON for this provider.
type Config struct {
	AppID             string
	AccessKey         string
	AppKey            string
	ResourceID        string
	BaseURL           string
	Model             string // 1.2.1.1 (O) or 2.2.0.0 (SC)
	Speaker           string
	BotName           string
	SystemRole        string
	SpeakingStyle     string
	CharacterManifest string
	DialTimeoutMs     int
}

// New is the realtime.Provider entry point.
//
//	cfg keys: appId, accessKey (or access_token/token), appKey, resourceId,
//	model, speaker/voice, botName, systemRole, speakingStyle,
//	characterManifest, baseUrl, dialTimeoutMs
func New(cfg map[string]any, opts realtime.Options) (realtime.Agent, error) {
	c := Config{
		AppID:             firstString(cfg, "appId", "app_id"),
		AccessKey:         firstString(cfg, "accessKey", "access_key", "access_token", "token"),
		AppKey:            firstString(cfg, "appKey", "app_key"),
		ResourceID:        firstString(cfg, "resourceId", "resource_id"),
		BaseURL:           firstString(cfg, "baseUrl", "base_url"),
		Model:             firstString(cfg, "model"),
		Speaker:           firstString(cfg, "speaker", "voice"),
		BotName:           firstString(cfg, "botName", "bot_name"),
		SystemRole:        firstString(cfg, "systemRole", "system_role", "instructions"),
		SpeakingStyle:     firstString(cfg, "speakingStyle", "speaking_style"),
		CharacterManifest: firstString(cfg, "characterManifest", "character_manifest"),
		DialTimeoutMs:     firstInt(cfg, "dialTimeoutMs", "dial_timeout_ms"),
	}
	if c.AppID == "" || c.AccessKey == "" {
		return nil, fmt.Errorf("volcdialogue: appId and accessKey are required")
	}
	if c.AppKey == "" {
		c.AppKey = defaultAppKey
	}
	if c.ResourceID == "" {
		c.ResourceID = defaultResourceID
	}
	if c.BaseURL == "" {
		c.BaseURL = defaultWSURL
	}
	if c.Model == "" {
		c.Model = defaultModelO
	}
	if c.Speaker == "" {
		c.Speaker = defaultSpeaker
	}
	if strings.TrimSpace(opts.Voice) != "" {
		c.Speaker = strings.TrimSpace(opts.Voice)
	}
	if c.DialTimeoutMs <= 0 {
		c.DialTimeoutMs = defaultDialMs
	}

	sys := strings.TrimSpace(c.SystemRole)
	if sys == "" {
		sys = strings.TrimSpace(opts.SystemPrompt)
	} else if sp := strings.TrimSpace(opts.SystemPrompt); sp != "" {
		sys = sys + "\n\n" + sp
	}
	c.SystemRole = sys

	return &agent{
		cfg:       c,
		opts:      opts,
		sessionID: uuid.New().String(),
		sendCh:    make(chan []byte, defaultSendBuf),
	}, nil
}

type agent struct {
	cfg       Config
	opts      realtime.Options
	sessionID string

	conn *websocket.Conn

	startOnce sync.Once
	closeOnce sync.Once
	closed    atomic.Bool
	openFired atomic.Bool

	sendCh chan []byte
	wg     sync.WaitGroup

	rootCtx    context.Context
	rootCancel context.CancelFunc
}

func (a *agent) Start(ctx context.Context) error {
	var err error
	a.startOnce.Do(func() { err = a.doStart(ctx) })
	return err
}

func (a *agent) doStart(ctx context.Context) error {
	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = time.Duration(a.cfg.DialTimeoutMs) * time.Millisecond

	headers := http.Header{}
	headers.Set("X-Api-App-ID", a.cfg.AppID)
	headers.Set("X-Api-Access-Key", a.cfg.AccessKey)
	headers.Set("X-Api-Resource-Id", a.cfg.ResourceID)
	headers.Set("X-Api-App-Key", a.cfg.AppKey)
	headers.Set("X-Api-Connect-Id", uuid.New().String())

	conn, resp, err := dialer.DialContext(ctx, a.cfg.BaseURL, headers)
	if err != nil {
		status := -1
		if resp != nil {
			status = resp.StatusCode
			resp.Body.Close()
		}
		return fmt.Errorf("volcdialogue: dial (status=%d): %w", status, err)
	}
	a.conn = conn

	if err := a.handshake(); err != nil {
		_ = conn.Close()
		return err
	}

	a.rootCtx, a.rootCancel = context.WithCancel(context.Background())
	a.wg.Add(2)
	go a.writeLoop()
	go a.readLoop()
	return nil
}

func (a *agent) handshake() error {
	// StartConnection
	frame, err := marshalJSONEvent(eventStartConnection, "", map[string]any{})
	if err != nil {
		return err
	}
	if err := a.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return fmt.Errorf("volcdialogue: send StartConnection: %w", err)
	}
	f, err := a.readOneHandshake()
	if err != nil {
		return fmt.Errorf("volcdialogue: StartConnection: %w", err)
	}
	if f.event != eventConnectionStarted {
		return fmt.Errorf("volcdialogue: expected ConnectionStarted(50), got event %d: %s", f.event, string(f.payload))
	}

	// StartSession
	payload := a.buildStartSession()
	frame, err = marshalJSONEvent(eventStartSession, a.sessionID, payload)
	if err != nil {
		return err
	}
	if err := a.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		return fmt.Errorf("volcdialogue: send StartSession: %w", err)
	}
	f, err = a.readOneHandshake()
	if err != nil {
		return fmt.Errorf("volcdialogue: StartSession: %w", err)
	}
	if f.event != eventSessionStarted {
		if f.event == eventSessionFailed {
			return fmt.Errorf("volcdialogue: session failed: %s", string(f.payload))
		}
		return fmt.Errorf("volcdialogue: expected SessionStarted(150), got event %d: %s", f.event, string(f.payload))
	}

	a.fireOnce(realtime.Event{Type: realtime.EventSessionOpen, Vendor: ProviderSlug})
	return nil
}

func (a *agent) readOneHandshake() (*frame, error) {
	_, data, err := a.conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	return parseFrame(data)
}

func (a *agent) buildStartSession() startSessionPayload {
	outRate := a.opts.OutputSampleRate
	if outRate <= 0 {
		outRate = 24000
	}
	dialogExtra := map[string]any{
		"strict_audit": false,
		"input_mod":    "audio",
		"model":        a.cfg.Model,
	}
	// SC2.0 uses character_manifest; O/O2 uses bot_name + system_role.
	if strings.TrimSpace(a.cfg.CharacterManifest) != "" {
		return startSessionPayload{
			ASR: asrPayload{Format: "pcm", Rate: 16000, Bits: 16, Channel: 1},
			TTS: ttsPayload{
				Speaker: a.cfg.Speaker,
				AudioConfig: audioConfig{
					Channel: 1, Format: "pcm_s16le", SampleRate: outRate,
				},
			},
			Dialog: dialogPayload{
				CharacterManifest: a.cfg.CharacterManifest,
				Extra:             dialogExtra,
			},
		}
	}
	botName := a.cfg.BotName
	if botName == "" {
		botName = "豆包"
	}
	style := a.cfg.SpeakingStyle
	if style == "" {
		style = "专业、简洁、友好"
	}
	return startSessionPayload{
		ASR: asrPayload{
			Format: "pcm", Rate: 16000, Bits: 16, Channel: 1,
			Extra: map[string]any{"enable_itn_convert": true},
		},
		TTS: ttsPayload{
			Speaker: a.cfg.Speaker,
			AudioConfig: audioConfig{
				Channel: 1, Format: "pcm_s16le", SampleRate: outRate,
			},
		},
		Dialog: dialogPayload{
			BotName:       botName,
			SystemRole:    a.cfg.SystemRole,
			SpeakingStyle: style,
			Extra:         dialogExtra,
		},
	}
}

func (a *agent) PushAudio(pcm []byte) error {
	if a.closed.Load() {
		return realtime.ErrAgentClosed
	}
	if len(pcm) == 0 {
		return nil
	}
	frame, err := marshalAudioTask(a.sessionID, pcm)
	if err != nil {
		return err
	}
	return a.enqueue(frame, true)
}

func (a *agent) CommitInputAudio() error { return nil }

func (a *agent) Cancel() error {
	// Server VAD handles turn boundaries; local media layer drops in-flight audio.
	return nil
}

func (a *agent) UpdateInstructions(instructions string) error {
	instructions = strings.TrimSpace(instructions)
	if instructions == "" {
		return nil
	}
	a.cfg.SystemRole = instructions
	if a.closed.Load() || a.conn == nil {
		return realtime.ErrAgentClosed
	}
	payload := map[string]any{
		"dialog": map[string]any{
			"system_role": instructions,
		},
	}
	frame, err := marshalJSONEvent(201, a.sessionID, payload) // UpdateConfig
	if err != nil {
		return err
	}
	return a.enqueue(frame, false)
}

func (a *agent) Close() error {
	a.closeOnce.Do(func() {
		a.closed.Store(true)
		if a.rootCancel != nil {
			a.rootCancel()
		}
		if a.conn != nil {
			if frame, err := marshalJSONEvent(eventFinishSession, a.sessionID, map[string]any{}); err == nil {
				_ = a.conn.WriteMessage(websocket.BinaryMessage, frame)
			}
			if frame, err := marshalJSONEvent(eventFinishConnection, "", map[string]any{}); err == nil {
				_ = a.conn.WriteMessage(websocket.BinaryMessage, frame)
			}
			_ = a.conn.Close()
		}
		a.wg.Wait()
		if !a.openFired.Load() {
			return
		}
		a.emit(realtime.Event{Type: realtime.EventSessionClose, Vendor: ProviderSlug})
	})
	return nil
}

func (a *agent) enqueue(frame []byte, nonBlocking bool) error {
	if a.closed.Load() {
		return realtime.ErrAgentClosed
	}
	if nonBlocking {
		select {
		case a.sendCh <- frame:
			return nil
		case <-a.rootCtx.Done():
			return realtime.ErrAgentClosed
		default:
			return nil
		}
	}
	select {
	case a.sendCh <- frame:
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
		case frame, ok := <-a.sendCh:
			if !ok {
				return
			}
			if a.conn == nil {
				continue
			}
			if err := a.conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				a.emit(realtime.Event{
					Type:   realtime.EventError,
					Err:    fmt.Errorf("volcdialogue: write: %w", err),
					Fatal:  true,
					Vendor: ProviderSlug,
				})
				return
			}
		}
	}
}

func (a *agent) readLoop() {
	defer a.wg.Done()
	defer func() {
		if a.rootCancel != nil {
			a.rootCancel()
		}
	}()
	for {
		if a.closed.Load() {
			return
		}
		_, data, err := a.conn.ReadMessage()
		if err != nil {
			if !a.closed.Load() {
				a.emit(realtime.Event{
					Type:   realtime.EventError,
					Err:    fmt.Errorf("volcdialogue: read: %w", err),
					Fatal:  true,
					Vendor: ProviderSlug,
				})
			}
			return
		}
		f, err := parseFrame(data)
		if err != nil {
			a.emit(realtime.Event{
				Type:   realtime.EventError,
				Err:    err,
				Fatal:  false,
				Vendor: ProviderSlug,
			})
			continue
		}
		a.dispatch(f)
	}
}

func (a *agent) dispatch(f *frame) {
	if f.msgType == msgTypeError {
		a.emit(realtime.Event{
			Type:   realtime.EventError,
			Err:    fmt.Errorf("volcdialogue: server error %d: %s", f.errorCode, string(f.payload)),
			Fatal:  true,
			Vendor: ProviderSlug,
		})
		return
	}

	if f.isAudioServer() {
		a.emit(realtime.Event{
			Type:    realtime.EventAssistantAudio,
			AudioPC: append([]byte(nil), f.payload...),
			Vendor:  ProviderSlug,
		})
		return
	}

	if f.msgType != msgTypeFullServer {
		return
	}

	switch f.event {
	case eventASRStarted:
		a.emit(realtime.Event{Type: realtime.EventUserSpeechStarted, Vendor: ProviderSlug})

	case eventASRResponse:
		var p asrResponsePayload
		if json.Unmarshal(f.payload, &p) == nil && len(p.Results) > 0 {
			r := p.Results[0]
			if strings.TrimSpace(r.Text) != "" {
				a.emit(realtime.Event{
					Type:   realtime.EventUserTranscript,
					Text:   r.Text,
					Final:  !r.IsInterim,
					Vendor: ProviderSlug,
				})
			}
		}

	case eventASREnded:
		a.emit(realtime.Event{Type: realtime.EventUserSpeechEnded, Vendor: ProviderSlug})

	case eventChatResponse:
		var p chatResponsePayload
		if json.Unmarshal(f.payload, &p) == nil && p.Content != "" {
			a.emit(realtime.Event{
				Type:   realtime.EventAssistantText,
				Text:   p.Content,
				Final:  false,
				Vendor: ProviderSlug,
			})
		}

	case eventChatEnded:
		a.emit(realtime.Event{
			Type:   realtime.EventAssistantText,
			Text:   "",
			Final:  true,
			Vendor: ProviderSlug,
		})

	case eventTTSEnded:
		a.emit(realtime.Event{Type: realtime.EventAssistantTurnEnd, Vendor: ProviderSlug})

	case eventSessionFailed, eventDialogCommonError:
		var de dialogErrorPayload
		msg := string(f.payload)
		if json.Unmarshal(f.payload, &de) == nil && de.Message != "" {
			msg = de.Message
		}
		a.emit(realtime.Event{
			Type:   realtime.EventError,
			Err:    fmt.Errorf("volcdialogue: event %d: %s", f.event, msg),
			Fatal:  true,
			Vendor: ProviderSlug,
		})

	case eventConnectionStarted, eventSessionStarted:
		// Handled during handshake.
	}
}

func (a *agent) fireOnce(ev realtime.Event) {
	if a.openFired.CompareAndSwap(false, true) {
		ev.Vendor = ProviderSlug
		a.opts.OnEvent(ev)
	}
}

func (a *agent) emit(ev realtime.Event) {
	if a.opts.OnEvent == nil {
		return
	}
	if ev.Vendor == "" {
		ev.Vendor = ProviderSlug
	}
	a.opts.OnEvent(ev)
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
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
