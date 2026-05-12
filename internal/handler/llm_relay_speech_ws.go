// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 对外语音 Relay：WebSocket 流式接口。
//
// ASR 流式  GET /v1/speech/asr/stream?token=...&group=...&provider=...&format=...&language=...
//   客户端 → 服务端：
//     - Binary frame：PCM/编码音频字节（建议 16kHz mono PCM，按厂商要求）
//     - Text frame   ：JSON 控制消息，可选：
//         {"type":"end"}              结束发送，等待 final
//         {"type":"close"}            主动关闭
//   服务端 → 客户端：均为 Text JSON：
//     {"type":"ready","provider":"...","channel_id":1,"group":"default"}
//     {"type":"partial","text":"..."}
//     {"type":"final","text":"...","duration_ms":1234}
//     {"type":"error","message":"..."}
//
// TTS 流式  GET /v1/speech/tts/stream?token=...&group=...&voice=...&audio_format=...
//   客户端 → 服务端：
//     - Text JSON：{"type":"speak","text":"..."} 触发合成
//     - Text JSON：{"type":"close"}              主动关闭
//   服务端 → 客户端：
//     - Text JSON：{"type":"started","format":{...},"provider":"...","channel_id":1}
//     - Binary frame：原始音频字节（chunk）
//     - Text JSON：{"type":"finished","audio_bytes":12345}
//     - Text JSON：{"type":"error","message":"..."}
//
// 鉴权：和 HTTP 一致，优先 Authorization: Bearer <key>，其次 ?token=<key>。
//
// 注意：浏览器原生 WebSocket 无法设置自定义 header，因此提供 ?token= 查询参数兜底。

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var relaySpeechUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 1024,
	WriteBufferSize: 1024 * 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// relayResolveTokenForWS 兼容浏览器无法发自定义 header：从 Authorization 或 ?token= 取 key。
func relayResolveTokenForWS(c *gin.Context) string {
	if k := extractRelayAPIKey(c); k != "" {
		return k
	}
	if k := strings.TrimSpace(c.Query("token")); k != "" {
		return k
	}
	if k := strings.TrimSpace(c.Query("api_key")); k != "" {
		return k
	}
	return ""
}

// relayAuthWS 校验 Token + type；失败时直接 400/401（升级前），或返回 nil。
func (h *Handlers) relayAuthWS(c *gin.Context, want models.LLMTokenType) *models.LLMToken {
	key := relayResolveTokenForWS(c)
	if key == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"type": "auth_error", "message": "missing API key"}})
		return nil
	}
	tok, err := models.LookupActiveLLMToken(h.db, key)
	if err != nil || tok == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"type": "auth_error", "message": "invalid token"}})
		return nil
	}
	got := tok.Type
	if got == "" {
		got = models.LLMTokenTypeLLM
	}
	if got != want {
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{
			"type":    "token_type_mismatch",
			"message": "this token type (" + string(got) + ") cannot access " + string(want) + " endpoints",
		}})
		return nil
	}
	return tok
}

// wsSendJSON 写一条 JSON Text 帧（带写锁）。
func wsSendJSON(conn *websocket.Conn, mu *sync.Mutex, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(20 * time.Second))
	return conn.WriteMessage(websocket.TextMessage, b)
}

func wsSendBinary(conn *websocket.Conn, mu *sync.Mutex, data []byte) error {
	mu.Lock()
	defer mu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(20 * time.Second))
	return conn.WriteMessage(websocket.BinaryMessage, data)
}

// =================== ASR 流式 ===================

// handleRelaySpeechASRStream GET /v1/speech/asr/stream
func (h *Handlers) handleRelaySpeechASRStream(c *gin.Context) {
	tok := h.relayAuthWS(c, models.LLMTokenTypeASR)
	if tok == nil {
		return
	}
	// 选择渠道（升级前完成；失败可直接 503）。
	body := models.SpeechASRTranscribeReq{
		Group:    strings.TrimSpace(c.Query("group")),
		Provider: strings.TrimSpace(c.Query("provider")),
		Format:   strings.TrimSpace(c.Query("format")),
		Language: strings.TrimSpace(c.Query("language")),
	}
	ch, err := models.PickASRChannelForToken(h.db, tok, body.Group)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
			"type":    "no_asr_channel",
			"message": models.RelayNoSpeechChannelDetail(err, "ASR", body.Group),
		}})
		return
	}
	provider, merged, mErr := buildASRMergedConfig(ch, &body)
	if mErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "channel_config", "message": mErr.Error()}})
		return
	}

	conn, err := relaySpeechUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var writeMu sync.Mutex
	tc, err := recognizer.NewTranscriberConfigFromMap(strings.ToLower(provider), merged, body.Language)
	if err != nil {
		_ = wsSendJSON(conn, &writeMu, gin.H{"type": "error", "message": err.Error()})
		return
	}
	svc, err := recognizer.GetGlobalFactory().CreateTranscriber(tc)
	if err != nil {
		_ = wsSendJSON(conn, &writeMu, gin.H{"type": "error", "message": err.Error()})
		return
	}
	defer func() { _ = svc.StopConn() }()

	startedAt := time.Now()
	finalCh := make(chan struct{}, 1)
	svc.Init(func(t string, isLast bool, _ time.Duration, _ string) {
		txt := strings.TrimSpace(t)
		if txt == "" && !isLast {
			return
		}
		if isLast {
			_ = wsSendJSON(conn, &writeMu, gin.H{
				"type":        "final",
				"text":        txt,
				"duration_ms": time.Since(startedAt).Milliseconds(),
			})
			select {
			case finalCh <- struct{}{}:
			default:
			}
			return
		}
		_ = wsSendJSON(conn, &writeMu, gin.H{"type": "partial", "text": txt})
	}, func(e error, fatal bool) {
		if e == nil {
			return
		}
		_ = wsSendJSON(conn, &writeMu, gin.H{"type": "error", "fatal": fatal, "message": e.Error()})
	})

	if err := svc.ConnAndReceive("relay-stream"); err != nil {
		_ = wsSendJSON(conn, &writeMu, gin.H{"type": "error", "message": err.Error()})
		return
	}
	_ = wsSendJSON(conn, &writeMu, gin.H{
		"type":       "ready",
		"provider":   provider,
		"channel_id": ch.ID,
		"group":      ch.Group,
	})

	// 读循环：处理音频帧 / 控制消息。
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	conn.SetReadLimit(8 * 1024 * 1024)
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
		return nil
	})

	for {
		msgType, data, rerr := conn.ReadMessage()
		if rerr != nil {
			break
		}
		switch msgType {
		case websocket.BinaryMessage:
			if err := svc.SendAudioBytes(data); err != nil {
				_ = wsSendJSON(conn, &writeMu, gin.H{"type": "error", "message": err.Error()})
				cancel()
				return
			}
		case websocket.TextMessage:
			var ctrl struct {
				Type string `json:"type"`
			}
			if jerr := json.Unmarshal(data, &ctrl); jerr != nil {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(ctrl.Type)) {
			case "end":
				if err := svc.SendEnd(); err != nil {
					_ = wsSendJSON(conn, &writeMu, gin.H{"type": "error", "message": err.Error()})
					return
				}
				// 等待 final 或客户端主动关闭。
				select {
				case <-finalCh:
				case <-time.After(relayASRResultWait):
					_ = wsSendJSON(conn, &writeMu, gin.H{"type": "error", "message": "ASR 等待 final 超时"})
				case <-ctx.Done():
				}
				return
			case "close":
				return
			}
		}
	}
}

// =================== TTS 流式 ===================

// streamingTTSHandler 把 svc.Synthesize 的 OnMessage 数据按帧推到 WS。
type streamingTTSHandler struct {
	conn *websocket.Conn
	mu   *sync.Mutex
	bytes int64
	err   atomicErr
}

type atomicErr struct {
	mu  sync.Mutex
	err error
}

func (a *atomicErr) Set(e error) { a.mu.Lock(); a.err = e; a.mu.Unlock() }
func (a *atomicErr) Get() error  { a.mu.Lock(); defer a.mu.Unlock(); return a.err }

func (s *streamingTTSHandler) OnMessage(data []byte) {
	if len(data) == 0 {
		return
	}
	if err := wsSendBinary(s.conn, s.mu, data); err != nil {
		s.err.Set(err)
		return
	}
	s.bytes += int64(len(data))
}
func (s *streamingTTSHandler) OnTimestamp(_ synthesizer.SentenceTimestamp) {}

// handleRelaySpeechTTSStream GET /v1/speech/tts/stream
func (h *Handlers) handleRelaySpeechTTSStream(c *gin.Context) {
	tok := h.relayAuthWS(c, models.LLMTokenTypeTTS)
	if tok == nil {
		return
	}
	body := models.SpeechTTSSynthesizeReq{
		Group:       strings.TrimSpace(c.Query("group")),
		Voice:       strings.TrimSpace(c.Query("voice")),
		AudioFormat: strings.TrimSpace(c.Query("audio_format")),
	}
	ch, err := models.PickTTSChannelForToken(h.db, tok, body.Group)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
			"type":    "no_tts_channel",
			"message": models.RelayNoSpeechChannelDetail(err, "TTS", body.Group),
		}})
		return
	}
	merged, mErr := buildTTSMergedConfig(ch, &body)
	if mErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "channel_config", "message": mErr.Error()}})
		return
	}
	svc, err := synthesizer.NewSynthesisServiceFromCredential(merged)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "tts_init_failed", "message": err.Error()}})
		return
	}
	defer func() { _ = svc.Close() }()

	conn, err := relaySpeechUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var writeMu sync.Mutex
	fmtSpec := svc.Format()
	_ = wsSendJSON(conn, &writeMu, gin.H{
		"type":       "started",
		"provider":   ch.Provider,
		"channel_id": ch.ID,
		"group":      ch.Group,
		"format": gin.H{
			"sample_rate": fmtSpec.SampleRate,
			"bit_depth":   fmtSpec.BitDepth,
			"channels":    fmtSpec.Channels,
		},
	})

	conn.SetReadLimit(1 * 1024 * 1024)
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(10 * time.Minute))
		return nil
	})

	for {
		mt, data, rerr := conn.ReadMessage()
		if rerr != nil {
			return
		}
		if mt != websocket.TextMessage {
			continue
		}
		var ctrl struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if jerr := json.Unmarshal(data, &ctrl); jerr != nil {
			_ = wsSendJSON(conn, &writeMu, gin.H{"type": "error", "message": "invalid control json"})
			continue
		}
		switch strings.ToLower(strings.TrimSpace(ctrl.Type)) {
		case "speak":
			text := strings.TrimSpace(ctrl.Text)
			if text == "" {
				_ = wsSendJSON(conn, &writeMu, gin.H{"type": "error", "message": "text 不能为空"})
				continue
			}
			handler := &streamingTTSHandler{conn: conn, mu: &writeMu}
			if synErr := svc.Synthesize(c.Request.Context(), handler, text); synErr != nil {
				_ = wsSendJSON(conn, &writeMu, gin.H{"type": "error", "message": synErr.Error()})
				continue
			}
			if e := handler.err.Get(); e != nil {
				_ = wsSendJSON(conn, &writeMu, gin.H{"type": "error", "message": e.Error()})
				return
			}
			_ = wsSendJSON(conn, &writeMu, gin.H{
				"type":        "finished",
				"audio_bytes": handler.bytes,
				"text_chars":  len([]rune(text)),
			})
		case "close":
			return
		default:
			_ = wsSendJSON(conn, &writeMu, gin.H{"type": "error", "message": fmt.Sprintf("unknown control type: %s", ctrl.Type)})
		}
	}
}

// LookupActiveLLMToken 防止外部包遗漏 export；包内已有同名函数复用即可。
// 此处保留私有 stub 以避免未导出包扩散；如已有同名 helper 请删除此 stub。
var _ = errors.New
