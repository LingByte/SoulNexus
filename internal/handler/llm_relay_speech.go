// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 对外语音 Relay：HTTP 接口（OpenAI 兼容 + LingVoice 兼容双形态）。
// 鉴权基于 LLMToken（type=asr 走 ASR；type=tts 走 TTS）。
// 实现：
//   - POST /v1/audio/transcriptions (OpenAI compatible, multipart/form-data)
//   - POST /v1/audio/speech         (OpenAI compatible, JSON, returns audio bytes)
//   - POST /v1/speech/asr/transcribe (LingVoice 风格：multipart 或 JSON，返回 JSON 文本)
//   - POST /v1/speech/tts/synthesize (LingVoice 风格：JSON，返回 base64 / url)
//
// 渠道选择走 group + sort_order；config_json 与请求参数合并后传入 pkg/recognizer / pkg/synthesizer。

package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/LingByte/SoulNexus/pkg/utils/ssrf"
	"github.com/LingByte/lingstorage-sdk-go"
	"github.com/gin-gonic/gin"
)

const (
	relaySpeechMaxAudioBytes = models.MaxRelayAudioFetchBytes
	relayASRChunkBytes       = 8 * 1024
	relayASRResultWait       = 120 * time.Second
)

// relaySSRFProtection 用于音频 URL 拉取的 SSRF 策略：禁私网、仅 80/443、3s DNS 超时。
func relaySSRFProtection() *ssrf.Protection { return ssrf.Default() }

// fetchRelayAudioBytes 拉取 URL 音频，限制大小，并做 SSRF 防护（DNS+连接两层）。
func fetchRelayAudioBytes(ctx context.Context, rawURL string) ([]byte, error) {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return nil, errors.New("audio_url 为空")
	}
	prot := relaySSRFProtection()
	if err := prot.ValidateURL(u); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	client := prot.SafeHTTPClient(30 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("拉取 audio_url 失败 HTTP %d", resp.StatusCode)
	}
	body := io.LimitReader(resp.Body, int64(relaySpeechMaxAudioBytes)+1)
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	if len(data) > relaySpeechMaxAudioBytes {
		return nil, fmt.Errorf("audio 超过大小限制 %d 字节", relaySpeechMaxAudioBytes)
	}
	return data, nil
}

// loadRelayASRAudio 解析 multipart 或 JSON 入参，返回 (body, audio, source, err)。
// source ∈ {"upload","url","base64"}。multipart 字段：audio(file) / audio_url / audio_base64 / group / provider / format / language / extra(JSON)。
func loadRelayASRAudio(c *gin.Context) (models.SpeechASRTranscribeReq, []byte, string, error) {
	ct := strings.ToLower(strings.TrimSpace(c.ContentType()))
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := c.Request.ParseMultipartForm(int64(relaySpeechMaxAudioBytes)); err != nil {
			return models.SpeechASRTranscribeReq{}, nil, "", fmt.Errorf("解析 multipart 失败: %w", err)
		}
		var body models.SpeechASRTranscribeReq
		body.Group = strings.TrimSpace(c.PostForm("group"))
		body.Provider = strings.TrimSpace(c.PostForm("provider"))
		body.Format = strings.TrimSpace(c.PostForm("format"))
		body.Language = strings.TrimSpace(c.PostForm("language"))
		// OpenAI 兼容字段：model（忽略）/ language / response_format / prompt 等暂不展开。
		if ex := strings.TrimSpace(c.PostForm("extra")); ex != "" {
			var raw any
			if err := json.Unmarshal([]byte(ex), &raw); err == nil {
				body.Extra = raw
			}
		}
		file, ferr := c.FormFile("audio")
		if file == nil || ferr != nil {
			file, ferr = c.FormFile("file") // OpenAI 兼容
		}
		url := strings.TrimSpace(c.PostForm("audio_url"))
		b64 := strings.TrimSpace(c.PostForm("audio_base64"))
		n := 0
		if ferr == nil && file != nil {
			n++
		}
		if url != "" {
			n++
		}
		if b64 != "" {
			n++
		}
		if n > 1 {
			return body, nil, "", errors.New("音频来源请只选一种：表单文件 audio/file、audio_url 或 audio_base64")
		}
		if ferr == nil && file != nil {
			f, err := file.Open()
			if err != nil {
				return body, nil, "", err
			}
			defer f.Close()
			data, err := io.ReadAll(io.LimitReader(f, int64(relaySpeechMaxAudioBytes)+1))
			if err != nil {
				return body, nil, "", err
			}
			if len(data) > relaySpeechMaxAudioBytes {
				return body, nil, "", fmt.Errorf("audio 超过大小限制 %d 字节", relaySpeechMaxAudioBytes)
			}
			return body, data, "upload", nil
		}
		if url != "" {
			body.AudioURL = url
			data, err := fetchRelayAudioBytes(c.Request.Context(), url)
			if err != nil {
				return body, nil, "url", err
			}
			return body, data, "url", nil
		}
		if b64 != "" {
			raw, decErr := base64.StdEncoding.DecodeString(b64)
			if decErr != nil {
				return body, nil, "", fmt.Errorf("audio_base64 解码失败: %w", decErr)
			}
			if len(raw) > relaySpeechMaxAudioBytes {
				return body, nil, "", fmt.Errorf("audio 超过大小限制 %d 字节", relaySpeechMaxAudioBytes)
			}
			body.AudioBase64 = b64
			return body, raw, "base64", nil
		}
		return body, nil, "", errors.New("请上传表单文件字段 audio/file，或提供 audio_url / audio_base64")
	}

	// JSON
	var body models.SpeechASRTranscribeReq
	if err := c.ShouldBindJSON(&body); err != nil {
		return body, nil, "", err
	}
	b64 := strings.TrimSpace(body.AudioBase64)
	u := strings.TrimSpace(body.AudioURL)
	if b64 != "" && u != "" {
		return body, nil, "", errors.New("audio_base64 与 audio_url 请二选一")
	}
	if b64 == "" && u == "" {
		return body, nil, "", errors.New("请提供 audio_base64 或 audio_url 之一")
	}
	if b64 != "" {
		raw, decErr := base64.StdEncoding.DecodeString(b64)
		if decErr != nil {
			return body, nil, "", fmt.Errorf("audio_base64 解码失败: %w", decErr)
		}
		if len(raw) > relaySpeechMaxAudioBytes {
			return body, nil, "", fmt.Errorf("audio 超过大小限制 %d 字节", relaySpeechMaxAudioBytes)
		}
		return body, raw, "base64", nil
	}
	data, err := fetchRelayAudioBytes(c.Request.Context(), u)
	if err != nil {
		return body, nil, "url", err
	}
	return body, data, "url", nil
}

// relayASRSendPace 按"略慢于 16kHz mono PCM 实时"估算节流。
func relayASRSendPace(n int) time.Duration {
	if n <= 0 {
		return 0
	}
	const bytesPerSec = 32000.0 // 16kHz * 2 bytes
	sec := float64(n) / bytesPerSec
	d := time.Duration(sec * float64(time.Second))
	if d < 8*time.Millisecond {
		d = 8 * time.Millisecond
	}
	return d
}

// relayASRRecognizeOnce 走 pkg/recognizer 工厂的"一次性识别"流程：连接 → 推送 → SendEnd → 等待 isLast。
func relayASRRecognizeOnce(ctx context.Context, provider string, merged map[string]interface{}, audio []byte, language string) (string, error) {
	if len(audio) == 0 {
		return "", errors.New("audio is empty")
	}
	prov := strings.ToLower(strings.TrimSpace(provider))
	tc, err := recognizer.NewTranscriberConfigFromMap(prov, merged, language)
	if err != nil {
		return "", err
	}
	svc, err := recognizer.GetGlobalFactory().CreateTranscriber(tc)
	if err != nil {
		return "", err
	}
	defer func() { _ = svc.StopConn() }()

	done := make(chan string, 1)
	errFatal := make(chan error, 1)
	var lastMu sync.Mutex
	var lastPartial string

	svc.Init(func(t string, isLast bool, _ time.Duration, _ string) {
		lastMu.Lock()
		if strings.TrimSpace(t) != "" {
			lastPartial = t
		}
		lastMu.Unlock()
		if isLast {
			select {
			case done <- strings.TrimSpace(t):
			default:
			}
		}
	}, func(e error, fatal bool) {
		if fatal && e != nil {
			select {
			case errFatal <- e:
			default:
			}
		}
	})

	if err := svc.ConnAndReceive("relay"); err != nil {
		return "", err
	}

	for i := 0; i < len(audio); i += relayASRChunkBytes {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		end := i + relayASRChunkBytes
		if end > len(audio) {
			end = len(audio)
		}
		sent := end - i
		if err := svc.SendAudioBytes(audio[i:end]); err != nil {
			return "", err
		}
		if end < len(audio) {
			delay := relayASRSendPace(sent)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	if err := svc.SendEnd(); err != nil {
		return "", err
	}

	waitCtx, cancel := context.WithTimeout(ctx, relayASRResultWait)
	defer cancel()
	select {
	case t := <-done:
		if t != "" {
			return t, nil
		}
		lastMu.Lock()
		lp := strings.TrimSpace(lastPartial)
		lastMu.Unlock()
		return lp, nil
	case e := <-errFatal:
		if e != nil {
			return "", e
		}
		return "", errors.New("ASR failed")
	case <-waitCtx.Done():
		return "", fmt.Errorf("ASR 等待结果超时或已取消: %w", waitCtx.Err())
	}
}

// buildASRMergedConfig 将渠道 config_json + 请求 extra/format 合并到 provider merged map。
func buildASRMergedConfig(ch *models.ASRChannel, body *models.SpeechASRTranscribeReq) (string, map[string]interface{}, error) {
	provider := strings.TrimSpace(ch.Provider)
	if p := strings.TrimSpace(body.Provider); p != "" {
		provider = p
	}
	provKey := strings.ToLower(strings.TrimSpace(provider))
	merged := map[string]interface{}{"provider": provKey}
	if cj := strings.TrimSpace(ch.ConfigJSON); cj != "" {
		var extra map[string]interface{}
		if err := json.Unmarshal([]byte(cj), &extra); err != nil {
			return provider, nil, fmt.Errorf("渠道 configJson 无效: %w", err)
		}
		for k, v := range extra {
			merged[k] = v
		}
	}
	models.MergeASRTranscribeOptions(merged, body)
	merged["provider"] = provKey
	return provider, merged, nil
}

// buildTTSMergedConfig 将渠道 config_json + 请求 voice/audio_format/sample_rate/tts_options 合并。
func buildTTSMergedConfig(ch *models.TTSChannel, body *models.SpeechTTSSynthesizeReq) (map[string]interface{}, error) {
	provKey := strings.ToLower(strings.TrimSpace(ch.Provider))
	merged := map[string]interface{}{"provider": provKey}
	if cj := strings.TrimSpace(ch.ConfigJSON); cj != "" {
		var extra map[string]interface{}
		if err := json.Unmarshal([]byte(cj), &extra); err != nil {
			return nil, fmt.Errorf("渠道 configJson 无效: %w", err)
		}
		for k, v := range extra {
			merged[k] = v
		}
	}
	if v := strings.TrimSpace(body.Voice); v != "" {
		models.ApplyTTSVoiceToMergedMap(ch.Provider, v, merged)
	}
	if af := strings.TrimSpace(body.AudioFormat); af != "" {
		merged["format"] = af
	}
	if body.SampleRate > 0 {
		merged["sampleRate"] = body.SampleRate
	}
	if body.TTSOptions != nil {
		for k, v := range body.TTSOptions {
			k = strings.TrimSpace(k)
			if k == "" || strings.EqualFold(k, "provider") {
				continue
			}
			merged[k] = v
		}
	}
	if body.Extra != nil {
		if m, ok := body.Extra.(map[string]interface{}); ok {
			for k, v := range m {
				k = strings.TrimSpace(k)
				if k == "" || strings.EqualFold(k, "provider") {
					continue
				}
				merged[k] = v
			}
		}
	}
	return merged, nil
}

// ttsBufferHandler 累积 TTS 音频帧用于"非流式"返回。
type ttsRelayBufferHandler struct {
	mu  sync.Mutex
	buf []byte
}

func (t *ttsRelayBufferHandler) OnMessage(data []byte) {
	t.mu.Lock()
	t.buf = append(t.buf, data...)
	t.mu.Unlock()
}
func (t *ttsRelayBufferHandler) OnTimestamp(_ synthesizer.SentenceTimestamp) {}

func (t *ttsRelayBufferHandler) Bytes() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]byte, len(t.buf))
	copy(out, t.buf)
	return out
}

// recordSpeechUsage 写一条审计记录；任何错误都仅 logrus 记录，不影响主流程。
func (h *Handlers) recordSpeechUsage(c *gin.Context, kind models.SpeechUsageKind, tok *models.LLMToken, ch interface {
	GetID() uint
	GetProvider() string
	GetGroup() string
}, started time.Time, status int, success bool, errMsg string, audioBytes int64, textChars int, model string, reqSnap, respSnap any) {
	row := models.SpeechUsage{
		Kind:       string(kind),
		Status:     status,
		Success:    success,
		ErrorMsg:   truncRune(errMsg, 2000),
		AudioBytes: audioBytes,
		TextChars:  textChars,
		DurationMs: time.Since(started).Milliseconds(),
		ClientIP:   c.ClientIP(),
		UserAgent:  truncRune(c.Request.UserAgent(), 240),
		Model:      truncRune(model, 120),
	}
	if tok != nil {
		row.UserID = tok.UserID
		row.TokenID = tok.ID
		row.GroupKey = tok.Group
		row.GroupID = tok.OrgID
	}
	if ch != nil {
		row.ChannelID = ch.GetID()
		row.Provider = ch.GetProvider()
		if row.GroupKey == "" {
			row.GroupKey = ch.GetGroup()
		}
	}
	if reqSnap != nil {
		if b, err := json.Marshal(reqSnap); err == nil {
			row.ReqSnap = truncRune(string(b), 4000)
		}
	}
	if respSnap != nil {
		if b, err := json.Marshal(respSnap); err == nil {
			row.RespSnap = truncRune(string(b), 4000)
		}
	}
	_ = models.CreateSpeechUsage(h.db, &row)
}

func truncRune(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// channelIDProvider 让 ASR/TTS Channel 共用一个抽象。
type channelIDProvider interface {
	GetID() uint
	GetProvider() string
	GetGroup() string
}

// speechUsageBuilder 在 handler 主体内逐步填充字段，defer 时落库。
type speechUsageBuilder struct {
	kind       models.SpeechUsageKind
	tok        *models.LLMToken
	ch         channelIDProvider
	status     int
	success    bool
	errMsg     string
	audioBytes int64
	textChars  int
	model      string
	reqSnap    any
	respSnap   any
}

// finish 由 defer 调用；自动写一行 SpeechUsage。
func (h *Handlers) finishSpeechUsage(c *gin.Context, started time.Time, b *speechUsageBuilder) {
	if b == nil || b.kind == "" {
		return
	}
	h.recordSpeechUsage(c, b.kind, b.tok, b.ch, started,
		b.status, b.success, b.errMsg, b.audioBytes, b.textChars, b.model, b.reqSnap, b.respSnap)
}

// =================== HTTP Handlers ===================

// handleRelaySpeechASRTranscribe POST /v1/speech/asr/transcribe
//
// 鉴权：Bearer LLMToken，type=asr。
// 入参（任选其一）：multipart 表单 file/audio + group/provider/format/language；或 JSON {audio_base64|audio_url, ...}。
// 返回：{ text, provider, channel_id, group, audio_source, audio_bytes, message? }
func (h *Handlers) handleRelaySpeechASRTranscribe(c *gin.Context) {
	started := time.Now()
	u := &speechUsageBuilder{kind: models.SpeechUsageKindASR}
	defer h.finishSpeechUsage(c, started, u)

	tok, ok := relayTokenFromRequest(h.db, c)
	if !ok {
		u.status = http.StatusUnauthorized
		u.errMsg = "unauthenticated"
		return
	}
	u.tok = tok
	if !requireTokenType(c, tok, models.LLMTokenTypeASR) {
		u.status = http.StatusForbidden
		u.errMsg = "token type mismatch"
		return
	}
	body, audio, src, err := loadRelayASRAudio(c)
	u.audioBytes = int64(len(audio))
	u.reqSnap = gin.H{"group": body.Group, "provider": body.Provider, "format": body.Format, "language": body.Language, "audio_source": src}
	if err != nil {
		u.status, u.errMsg = http.StatusBadRequest, err.Error()
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "bad_request", "message": err.Error()}})
		return
	}
	if len(audio) == 0 {
		u.status, u.errMsg = http.StatusBadRequest, "empty audio"
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "bad_request", "message": "音频内容为空"}})
		return
	}
	ch, perr := models.PickASRChannelForToken(h.db, tok, body.Group)
	if perr != nil {
		u.status, u.errMsg = http.StatusServiceUnavailable, perr.Error()
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
			"type":    "no_asr_channel",
			"message": models.RelayNoSpeechChannelDetail(perr, "ASR", body.Group),
		}})
		return
	}
	u.ch = *ch
	provider, merged, mErr := buildASRMergedConfig(ch, &body)
	u.model = provider
	if mErr != nil {
		u.status, u.errMsg = http.StatusInternalServerError, mErr.Error()
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "channel_config", "message": mErr.Error()}})
		return
	}
	text, recErr := relayASRRecognizeOnce(c.Request.Context(), strings.ToLower(provider), merged, audio, body.Language)
	if recErr != nil {
		u.status, u.errMsg = http.StatusBadGateway, recErr.Error()
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "asr_failed", "message": recErr.Error()}})
		return
	}
	out := gin.H{
		"text":         text,
		"provider":     provider,
		"channel_id":   ch.ID,
		"group":        ch.Group,
		"audio_source": src,
		"audio_bytes":  len(audio),
	}
	if strings.TrimSpace(text) == "" {
		out["message"] = "未识别到有效文本（请检查音频格式与渠道 config_json 中 model/format 是否匹配）"
	}
	u.status, u.success, u.textChars = http.StatusOK, true, utf8.RuneCountInString(text)
	u.respSnap = gin.H{"text_len": u.textChars, "audio_bytes": u.audioBytes}
	response.Success(c, "transcribed", out)
}

// handleRelayOpenAIAudioTranscriptions POST /v1/audio/transcriptions
//
// OpenAI Whisper 兼容（仅 multipart）。返回 { text } 或 { text, segments: [] }。
func (h *Handlers) handleRelayOpenAIAudioTranscriptions(c *gin.Context) {
	started := time.Now()
	u := &speechUsageBuilder{kind: models.SpeechUsageKindASR}
	defer h.finishSpeechUsage(c, started, u)

	tok, ok := relayTokenFromRequest(h.db, c)
	if !ok {
		u.status, u.errMsg = http.StatusUnauthorized, "unauthenticated"
		return
	}
	u.tok = tok
	if !requireTokenType(c, tok, models.LLMTokenTypeASR) {
		u.status, u.errMsg = http.StatusForbidden, "token type mismatch"
		return
	}
	body, audio, _, err := loadRelayASRAudio(c)
	u.audioBytes = int64(len(audio))
	if err != nil {
		u.status, u.errMsg = http.StatusBadRequest, err.Error()
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return
	}
	if len(audio) == 0 {
		u.status, u.errMsg = http.StatusBadRequest, "empty audio"
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "audio is empty"}})
		return
	}
	ch, perr := models.PickASRChannelForToken(h.db, tok, body.Group)
	if perr != nil {
		u.status, u.errMsg = http.StatusServiceUnavailable, perr.Error()
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
			"type":    "no_asr_channel",
			"message": models.RelayNoSpeechChannelDetail(perr, "ASR", body.Group),
		}})
		return
	}
	u.ch = *ch
	provider, merged, mErr := buildASRMergedConfig(ch, &body)
	u.model = provider
	if mErr != nil {
		u.status, u.errMsg = http.StatusInternalServerError, mErr.Error()
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "channel_config", "message": mErr.Error()}})
		return
	}
	text, recErr := relayASRRecognizeOnce(c.Request.Context(), strings.ToLower(provider), merged, audio, body.Language)
	if recErr != nil {
		u.status, u.errMsg = http.StatusBadGateway, recErr.Error()
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "asr_failed", "message": recErr.Error()}})
		return
	}
	u.status, u.success, u.textChars = http.StatusOK, true, utf8.RuneCountInString(text)
	c.JSON(http.StatusOK, gin.H{
		"text":     text,
		"language": body.Language,
	})
}

// handleRelaySpeechTTSSynthesize POST /v1/speech/tts/synthesize
//
// 鉴权：Bearer LLMToken，type=tts。
// 入参：JSON { group, text(*), voice, audio_format, sample_rate, response_type=audio_base64|url, upload_*  }。
// 返回：JSON { format, provider, channel_id, group, audio_base64? | url? key bucket filename size }。
func (h *Handlers) handleRelaySpeechTTSSynthesize(c *gin.Context) {
	started := time.Now()
	u := &speechUsageBuilder{kind: models.SpeechUsageKindTTS}
	defer h.finishSpeechUsage(c, started, u)

	tok, ok := relayTokenFromRequest(h.db, c)
	if !ok {
		u.status, u.errMsg = http.StatusUnauthorized, "unauthenticated"
		return
	}
	u.tok = tok
	if !requireTokenType(c, tok, models.LLMTokenTypeTTS) {
		u.status, u.errMsg = http.StatusForbidden, "token type mismatch"
		return
	}
	var body models.SpeechTTSSynthesizeReq
	if err := c.ShouldBindJSON(&body); err != nil {
		u.status, u.errMsg = http.StatusBadRequest, err.Error()
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "bad_request", "message": err.Error()}})
		return
	}
	text := strings.TrimSpace(body.Text)
	if text == "" {
		u.status, u.errMsg = http.StatusBadRequest, "empty text"
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "bad_request", "message": "text 不能为空"}})
		return
	}
	u.textChars = utf8.RuneCountInString(text)
	u.reqSnap = gin.H{"group": body.Group, "voice": body.Voice, "audio_format": body.AudioFormat, "sample_rate": body.SampleRate, "text_chars": u.textChars, "response_type": body.ResponseType}
	outMode := models.NormalizeRelayTTSResponseType(body.ResponseType, body.Output)
	ch, perr := models.PickTTSChannelForToken(h.db, tok, body.Group)
	if perr != nil {
		u.status, u.errMsg = http.StatusServiceUnavailable, perr.Error()
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
			"type":    "no_tts_channel",
			"message": models.RelayNoSpeechChannelDetail(perr, "TTS", body.Group),
		}})
		return
	}
	u.ch = *ch
	u.model = body.Voice
	merged, mErr := buildTTSMergedConfig(ch, &body)
	if mErr != nil {
		u.status, u.errMsg = http.StatusInternalServerError, mErr.Error()
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "channel_config", "message": mErr.Error()}})
		return
	}
	svc, err := synthesizer.NewSynthesisServiceFromCredential(merged)
	if err != nil {
		u.status, u.errMsg = http.StatusBadRequest, err.Error()
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "tts_init_failed", "message": err.Error()}})
		return
	}
	handler := &ttsRelayBufferHandler{}
	if err := svc.Synthesize(c.Request.Context(), handler, text); err != nil {
		_ = svc.Close()
		u.status, u.errMsg = http.StatusBadGateway, err.Error()
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "tts_failed", "message": err.Error()}})
		return
	}
	_ = svc.Close()
	audio := handler.Bytes()
	u.audioBytes = int64(len(audio))
	if len(audio) == 0 {
		u.status, u.errMsg = http.StatusBadGateway, "empty audio"
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "tts_failed", "message": "未收到音频数据"}})
		return
	}
	textChars := u.textChars

	out := gin.H{
		"response_type": outMode,
		"provider":      ch.Provider,
		"channel_id":    ch.ID,
		"group":         ch.Group,
		"text_chars":    textChars,
	}
	fmtSpec := svc.Format()
	out["format"] = gin.H{
		"sample_rate": fmtSpec.SampleRate,
		"bit_depth":   fmtSpec.BitDepth,
		"channels":    fmtSpec.Channels,
	}
	for _, k := range []string{"codec", "encoding", "response_format"} {
		if v, ok := merged[k]; ok && v != nil && fmt.Sprint(v) != "" {
			out[k] = v
		}
	}

	if outMode == "url" {
		if config.GlobalStore == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
				"type":    "storage_unavailable",
				"message": "对象存储未初始化（LINGSTORAGE_*）",
			}})
			return
		}
		bucket := strings.TrimSpace(body.UploadBucket)
		if bucket == "" && config.GlobalConfig != nil {
			bucket = config.GlobalConfig.Services.Storage.Bucket
		}
		if bucket == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
				"type":    "storage_unavailable",
				"message": "未配置存储 bucket",
			}})
			return
		}
		fname := strings.TrimSpace(body.UploadFilename)
		if fname == "" {
			fname = fmt.Sprintf("tts-%d.bin", time.Now().UnixNano())
		}
		fname = filepath.Base(fname)
		if fname == "." || fname == "" {
			fname = fmt.Sprintf("tts-%d.bin", time.Now().UnixNano())
		}
		key := strings.TrimSpace(body.UploadKey)
		if key == "" {
			key = fmt.Sprintf("relay/tts/%s/%d-%s", strings.TrimPrefix(ch.Group, "/"), time.Now().UnixNano(), fname)
		}
		up, upErr := config.GlobalStore.UploadBytes(&lingstorage.UploadBytesRequest{
			Data:     audio,
			Filename: fname,
			Bucket:   bucket,
			Key:      key,
		})
		if upErr != nil {
			u.status, u.errMsg = http.StatusBadGateway, upErr.Error()
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "storage_upload_failed", "message": upErr.Error()}})
			return
		}
		out["url"] = up.URL
		out["key"] = up.Key
		out["bucket"] = up.Bucket
		out["filename"] = up.Filename
		out["size"] = up.Size
		u.status, u.success = http.StatusOK, true
		u.respSnap = gin.H{"response_type": outMode, "size": up.Size, "key": up.Key}
		response.Success(c, "synthesized", out)
		return
	}

	// 默认 audio_base64。
	out["audio_base64"] = base64.StdEncoding.EncodeToString(audio)
	out["audio_bytes"] = len(audio)
	u.status, u.success = http.StatusOK, true
	u.respSnap = gin.H{"response_type": outMode, "audio_bytes": len(audio)}
	response.Success(c, "synthesized", out)
}

// handleRelayOpenAIAudioSpeech POST /v1/audio/speech
//
// OpenAI 兼容：JSON 入参 { model, input, voice, response_format }，直接返回二进制音频字节。
type relayOpenAIAudioSpeechReq struct {
	Model          string   `json:"model"`
	Input          string   `json:"input"`
	Voice          string   `json:"voice"`
	ResponseFormat string   `json:"response_format"`
	Speed          *float64 `json:"speed"`
	Group          string   `json:"group"`
}

func (h *Handlers) handleRelayOpenAIAudioSpeech(c *gin.Context) {
	started := time.Now()
	u := &speechUsageBuilder{kind: models.SpeechUsageKindTTS}
	defer h.finishSpeechUsage(c, started, u)

	tok, ok := relayTokenFromRequest(h.db, c)
	if !ok {
		u.status, u.errMsg = http.StatusUnauthorized, "unauthenticated"
		return
	}
	u.tok = tok
	if !requireTokenType(c, tok, models.LLMTokenTypeTTS) {
		u.status, u.errMsg = http.StatusForbidden, "token type mismatch"
		return
	}
	var req relayOpenAIAudioSpeechReq
	if err := c.ShouldBindJSON(&req); err != nil {
		u.status, u.errMsg = http.StatusBadRequest, err.Error()
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": err.Error()}})
		return
	}
	text := strings.TrimSpace(req.Input)
	if text == "" {
		u.status, u.errMsg = http.StatusBadRequest, "empty input"
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "invalid_request_error", "message": "input is required"}})
		return
	}
	u.model = req.Voice
	u.textChars = utf8.RuneCountInString(text)
	body := models.SpeechTTSSynthesizeReq{
		Group:       req.Group,
		Text:        text,
		Voice:       req.Voice,
		AudioFormat: req.ResponseFormat,
	}
	if req.Speed != nil {
		body.TTSOptions = map[string]interface{}{"speed": *req.Speed, "speedRatio": *req.Speed}
	}
	ch, perr := models.PickTTSChannelForToken(h.db, tok, body.Group)
	if perr != nil {
		u.status, u.errMsg = http.StatusServiceUnavailable, perr.Error()
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
			"type":    "no_tts_channel",
			"message": models.RelayNoSpeechChannelDetail(perr, "TTS", body.Group),
		}})
		return
	}
	u.ch = *ch
	merged, mErr := buildTTSMergedConfig(ch, &body)
	if mErr != nil {
		u.status, u.errMsg = http.StatusInternalServerError, mErr.Error()
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"type": "channel_config", "message": mErr.Error()}})
		return
	}
	svc, err := synthesizer.NewSynthesisServiceFromCredential(merged)
	if err != nil {
		u.status, u.errMsg = http.StatusBadRequest, err.Error()
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"type": "tts_init_failed", "message": err.Error()}})
		return
	}
	handler := &ttsRelayBufferHandler{}
	if err := svc.Synthesize(c.Request.Context(), handler, text); err != nil {
		_ = svc.Close()
		u.status, u.errMsg = http.StatusBadGateway, err.Error()
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "tts_failed", "message": err.Error()}})
		return
	}
	_ = svc.Close()
	audio := handler.Bytes()
	u.audioBytes = int64(len(audio))
	if len(audio) == 0 {
		u.status, u.errMsg = http.StatusBadGateway, "empty audio"
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"type": "tts_failed", "message": "未收到音频数据"}})
		return
	}
	mime := guessAudioMime(req.ResponseFormat)
	c.Header("X-Provider", ch.Provider)
	c.Header("X-Channel-ID", fmt.Sprintf("%d", ch.ID))
	c.Header("X-Group", ch.Group)
	u.status, u.success = http.StatusOK, true
	u.respSnap = gin.H{"format": req.ResponseFormat, "audio_bytes": len(audio)}
	c.Data(http.StatusOK, mime, audio)
}

func guessAudioMime(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "mp3":
		return "audio/mpeg"
	case "wav":
		return "audio/wav"
	case "opus":
		return "audio/opus"
	case "aac":
		return "audio/aac"
	case "flac":
		return "audio/flac"
	case "pcm":
		return "audio/pcm"
	default:
		return "application/octet-stream"
	}
}
