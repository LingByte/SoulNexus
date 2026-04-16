package handlers

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/response"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/LingByte/lingstorage-sdk-go"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	gonanoid "github.com/matoous/go-nanoid"
)

// registerOpenAPIRoutes 注册对外开放 API 路由，使用 apiKey + apiSecret 鉴权
func (h *Handlers) registerOpenAPIRoutes(r *gin.RouterGroup) {
	open := r.Group("/open")
	open.Use(models.AuthApiRequired)
	{
		open.GET("/me", h.openGetMe)
		open.POST("/tts", h.openTTS)
		open.POST("/asr", h.openASR)
		open.GET("/ws/asr", h.openWSASR)
		open.GET("/ws/tts", h.openWSTTS)
	}
}

// openGetMe 返回当前鉴权用户的基本信息
func (h *Handlers) openGetMe(c *gin.Context) {
	user := models.CurrentUser(c)
	if user == nil {
		response.AbortWithStatus(c, http.StatusUnauthorized)
		return
	}
	response.Success(c, "ok", gin.H{
		"id":       user.ID,
		"username": user.DisplayName,
		"email":    user.Email,
	})
}

type openWSInitMessage struct {
	Type  string `json:"type"`
	Codec string `json:"codec"`
}

type wsChunkHandler struct {
	onMessage func([]byte) error
}

func (h *wsChunkHandler) OnMessage(data []byte) {
	if len(data) == 0 || h.onMessage == nil {
		return
	}
	_ = h.onMessage(data)
}

func (h *wsChunkHandler) OnTimestamp(_ synthesizer.SentenceTimestamp) {}

func readWSInit(conn *websocket.Conn) (*openWSInitMessage, error) {
	msgType, payload, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	if msgType != websocket.TextMessage {
		return nil, fmt.Errorf("first message must be text init")
	}
	var initMsg openWSInitMessage
	if err := json.Unmarshal(payload, &initMsg); err != nil {
		return nil, fmt.Errorf("invalid init json: %w", err)
	}
	if initMsg.Type != "init" {
		return nil, fmt.Errorf("first message type must be init")
	}
	if initMsg.Codec == "" {
		initMsg.Codec = "pcm"
	}
	return &initMsg, nil
}

// openTTS TTS 一句话合成，返回音频 URL
func (h *Handlers) openTTS(c *gin.Context) {
	var req struct {
		Text string `json:"text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, "参数错误", err.Error())
		return
	}

	cred := h.getCredentialFromCtx(c)
	if cred == nil || len(cred.TtsConfig) == 0 {
		response.Fail(c, "未配置 TTS", "请在凭证中配置 ttsConfig")
		return
	}

	svc, err := synthesizer.NewSynthesisServiceFromCredential(synthesizer.TTSCredentialConfig(cred.TtsConfig))
	if err != nil {
		response.Fail(c, "初始化 TTS 失败", err.Error())
		return
	}
	defer svc.Close()

	buf := &synthesizer.SynthesisBuffer{}
	if err := svc.Synthesize(c.Request.Context(), buf, req.Text); err != nil {
		response.Fail(c, "TTS 合成失败", err.Error())
		return
	}

	format := svc.Format()
	wavData := buildWAV(buf.Data, format.SampleRate, format.Channels, format.BitDepth)

	key := fmt.Sprintf("open/tts/%d_%d.wav", cred.ID, time.Now().UnixMilli())
	result, err := config.GlobalStore.UploadBytes(&lingstorage.UploadBytesRequest{
		Bucket:   config.GlobalConfig.Services.Storage.Bucket,
		Data:     wavData,
		Filename: key,
		Key:      key,
	})
	if err != nil {
		response.Fail(c, "上传音频失败", err.Error())
		return
	}

	sampleRate := format.SampleRate
	if sampleRate == 0 {
		sampleRate = 8000
	}
	channels := format.Channels
	if channels == 0 {
		channels = 1
	}
	bitDepth := format.BitDepth
	if bitDepth == 0 {
		bitDepth = 16
	}
	duration := float64(len(buf.Data)) / float64(sampleRate*channels*(bitDepth/8))
	response.Success(c, "ok", gin.H{
		"url":      result.URL,
		"duration": duration,
	})
}

// openASR 一句话识别，上传 WAV 文件返回识别文字
func (h *Handlers) openASR(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.Fail(c, "获取文件失败", err.Error())
		return
	}
	defer file.Close()

	audioData, err := io.ReadAll(file)
	if err != nil {
		response.Fail(c, "读取文件失败", err.Error())
		return
	}
	cred := h.getCredentialFromCtx(c)
	if cred == nil || len(cred.AsrConfig) == 0 {
		response.Fail(c, "未配置 ASR", "请在凭证中配置 asrConfig")
		return
	}

	appID := cred.GetASRConfigString("appId")
	secretID := cred.GetASRConfigString("secretId")
	secretKey := cred.GetASRConfigString("secret")
	if appID == "" || secretID == "" || secretKey == "" {
		response.Fail(c, "ASR 配置不完整", "缺少 appId/secretId/secret")
		return
	}

	opt := recognizer.NewQcloudASROption(appID, secretID, secretKey)
	asr := recognizer.NewQcloudASR(opt)

	var (
		resultText string
		resultErr  error
		wg         sync.WaitGroup
	)
	wg.Add(1)
	asr.Init(
		func(text string, final bool, _ time.Duration, _ string) {
			if final {
				resultText = text
				wg.Done()
			}
		},
		func(err error, _ bool) {
			resultErr = err
			wg.Done()
		},
	)

	dialogID, _ := gonanoid.Nanoid()
	if err := asr.ConnAndReceive(dialogID); err != nil {
		response.Fail(c, "ASR 连接失败", err.Error())
		return
	}

	// 跳过 WAV 头（44 字节）
	pcm := audioData
	if len(audioData) > 44 {
		pcm = audioData[44:]
	}

	// 分块发送，每块 3200 字节（16k * 2bytes * 0.1s）
	chunkSize := 3200
	for i := 0; i < len(pcm); i += chunkSize {
		end := i + chunkSize
		if end > len(pcm) {
			end = len(pcm)
		}
		if err := asr.SendAudioBytes(pcm[i:end]); err != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = asr.SendEnd()

	wg.Wait()

	if resultErr != nil {
		response.Fail(c, "ASR 识别失败", resultErr.Error())
		return
	}

	// 计算时长（假设 16k 单声道 16bit）
	sampleRate := 16000
	duration := float64(len(pcm)) / float64(sampleRate*2)

	response.Success(c, "ok", gin.H{
		"text":     resultText,
		"duration": duration,
	})
}

// openWSASR WebSocket 流式 ASR
// 首帧必须发送 {"type":"init","codec":"pcm"}，然后发送二进制 PCM16 音频。
func (h *Handlers) openWSASR(c *gin.Context) {
	cred := h.getCredentialFromCtx(c)
	if cred == nil || len(cred.AsrConfig) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 500, "msg": "未配置 ASR", "data": nil})
		return
	}
	conn, err := voiceUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	initMsg, err := readWSInit(conn)
	if err != nil {
		_ = conn.WriteJSON(gin.H{"type": "error", "message": err.Error()})
		return
	}
	if initMsg.Codec != "pcm" {
		_ = conn.WriteJSON(gin.H{"type": "error", "message": "asr websocket only supports codec=pcm"})
		return
	}
	_ = conn.WriteJSON(gin.H{"type": "ready", "codec": "pcm"})

	asrProvider := cred.GetASRProvider()
	asrCfg := map[string]interface{}{
		"provider": asrProvider,
		"language": "zh",
	}
	for k, v := range cred.AsrConfig {
		asrCfg[k] = v
	}
	cfg, err := recognizer.NewTranscriberConfigFromMap(asrProvider, asrCfg, "zh")
	if err != nil {
		_ = conn.WriteJSON(gin.H{"type": "error", "message": err.Error()})
		return
	}
	asrService, err := recognizer.GetGlobalFactory().CreateTranscriber(cfg)
	if err != nil {
		_ = conn.WriteJSON(gin.H{"type": "error", "message": err.Error()})
		return
	}

	var writeMu sync.Mutex
	asrService.Init(
		func(text string, final bool, duration time.Duration, uuid string) {
			writeMu.Lock()
			defer writeMu.Unlock()
			_ = conn.WriteJSON(gin.H{
				"type":     "result",
				"text":     text,
				"final":    final,
				"duration": duration.Milliseconds(),
				"uuid":     uuid,
			})
		},
		func(err error, fatal bool) {
			writeMu.Lock()
			defer writeMu.Unlock()
			_ = conn.WriteJSON(gin.H{
				"type":    "error",
				"fatal":   fatal,
				"message": err.Error(),
			})
		},
	)

	dialogID, _ := gonanoid.Nanoid()
	if err := asrService.ConnAndReceive(dialogID); err != nil {
		_ = conn.WriteJSON(gin.H{"type": "error", "message": err.Error()})
		return
	}

	for {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			_ = asrService.SendEnd()
			return
		}
		switch msgType {
		case websocket.BinaryMessage:
			if len(payload) > 0 {
				_ = asrService.SendAudioBytes(payload)
			}
		case websocket.TextMessage:
			var ctrl struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(payload, &ctrl)
			if ctrl.Type == "end" {
				_ = asrService.SendEnd()
			} else if ctrl.Type == "ping" {
				writeMu.Lock()
				_ = conn.WriteJSON(gin.H{"type": "pong"})
				writeMu.Unlock()
			}
		}
	}
}

// openWSTTS WebSocket 流式 TTS
// 首帧必须发送 {"type":"init","codec":"pcm"}，然后发送 {"text":"..."}。
func (h *Handlers) openWSTTS(c *gin.Context) {
	cred := h.getCredentialFromCtx(c)
	if cred == nil || len(cred.TtsConfig) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 500, "msg": "未配置 TTS", "data": nil})
		return
	}
	conn, err := voiceUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	initMsg, err := readWSInit(conn)
	if err != nil {
		_ = conn.WriteJSON(gin.H{"type": "error", "message": err.Error()})
		return
	}
	if initMsg.Codec != "pcm" {
		_ = conn.WriteJSON(gin.H{"type": "error", "message": "tts websocket only supports codec=pcm"})
		return
	}
	_ = conn.WriteJSON(gin.H{"type": "ready", "codec": "pcm"})

	svc, err := synthesizer.NewSynthesisServiceFromCredential(synthesizer.TTSCredentialConfig(cred.TtsConfig))
	if err != nil {
		_ = conn.WriteJSON(gin.H{"type": "error", "message": err.Error()})
		return
	}
	defer svc.Close()

	var writeMu sync.Mutex
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var req struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(payload, &req); err != nil {
			writeMu.Lock()
			_ = conn.WriteJSON(gin.H{"type": "error", "message": "invalid json"})
			writeMu.Unlock()
			continue
		}
		if req.Type == "ping" {
			writeMu.Lock()
			_ = conn.WriteJSON(gin.H{"type": "pong"})
			writeMu.Unlock()
			continue
		}
		if req.Text == "" {
			continue
		}
		writeMu.Lock()
		_ = conn.WriteJSON(gin.H{"type": "start", "codec": "pcm"})
		writeMu.Unlock()
		err = svc.Synthesize(c.Request.Context(), &wsChunkHandler{
			onMessage: func(chunk []byte) error {
				writeMu.Lock()
				defer writeMu.Unlock()
				return conn.WriteMessage(websocket.BinaryMessage, chunk)
			},
		}, req.Text)
		if err != nil {
			writeMu.Lock()
			_ = conn.WriteJSON(gin.H{"type": "error", "message": err.Error()})
			writeMu.Unlock()
			continue
		}
		writeMu.Lock()
		_ = conn.WriteJSON(gin.H{"type": "end"})
		writeMu.Unlock()
	}
}

// buildWAV 将 PCM 数据加上 WAV 头
func buildWAV(pcm []byte, sampleRate, channels, bitDepth int) []byte {
	if sampleRate == 0 {
		sampleRate = 8000
	}
	if channels == 0 {
		channels = 1
	}
	if bitDepth == 0 {
		bitDepth = 16
	}
	header := make([]byte, 44)
	dataSize := len(pcm)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+dataSize))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1)
	binary.LittleEndian.PutUint16(header[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(sampleRate*channels*bitDepth/8))
	binary.LittleEndian.PutUint16(header[32:34], uint16(channels*bitDepth/8))
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitDepth))
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize))
	return append(header, pcm...)
}

// getCredentialFromCtx 从请求头或 query 参数中取当前鉴权凭证
func (h *Handlers) getCredentialFromCtx(c *gin.Context) *models.UserCredential {
	apiKey := c.GetHeader("X-API-KEY")
	apiSecret := c.GetHeader("X-API-SECRET")
	if apiKey == "" {
		apiKey = c.Query("apiKey")
		apiSecret = c.Query("apiSecret")
	}
	if apiKey == "" || apiSecret == "" {
		return nil
	}
	cred, _ := models.GetUserCredentialByApiSecretAndApiKey(h.db, apiKey, apiSecret)
	return cred
}

// getCredentialIDFromCtx 获取当前凭证 ID（用于生成存储 key）
func (h *Handlers) getCredentialIDFromCtx(c *gin.Context) uint {
	if cred := h.getCredentialFromCtx(c); cred != nil {
		return cred.ID
	}
	return 0
}
