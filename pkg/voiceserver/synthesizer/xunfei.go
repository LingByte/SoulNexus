package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/utils"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// XunfeiTTSConfig 讯飞TTS配置
type XunfeiTTSConfig struct {
	AppID         string `json:"app_id" yaml:"app_id" env:"XUNFEI_APP_ID"`
	APIKey        string `json:"api_key" yaml:"api_key" env:"XUNFEI_API_KEY"`
	APISecret     string `json:"api_secret" yaml:"api_secret" env:"XUNFEI_API_SECRET"`
	SampleRate    int    `json:"sample_rate" yaml:"sample_rate" default:"24000"`
	Channels      int    `json:"channels" yaml:"channels" default:"1"`
	BitDepth      int    `json:"bit_depth" yaml:"bit_depth" default:"16"`
	Codec         string `json:"codec" yaml:"codec" default:"raw"`
	FrameDuration string `json:"frame_duration" yaml:"frame_duration" default:"20ms"`
	Timeout       int    `json:"timeout" yaml:"timeout" default:"30"`
}

type XunfeiService struct {
	opt XunfeiTTSConfig
	mu  sync.Mutex // 保护 opt 的并发访问
}

// WSRequest WebSocket请求结构
type WSRequest struct {
	Header    WSHeader    `json:"header"`
	Parameter WSParameter `json:"parameter"`
	Payload   WSPayload   `json:"payload"`
}

// WSHeader WebSocket请求头
type WSHeader struct {
	AppID  string `json:"app_id"`
	Status int    `json:"status"`
	ResID  string `json:"res_id"`
}

// WSAudio WebSocket音频参数
type WSAudio struct {
	Encoding   string `json:"encoding"`
	SampleRate int    `json:"sample_rate"`
}

// WSTTS WebSocket TTS参数
type WSTTS struct {
	Vcn      string  `json:"vcn"`
	Volume   int     `json:"volume"`
	Rhy      int     `json:"rhy"`
	Pybuffer int     `json:"pybuffer"`
	Speed    int     `json:"speed"`
	Pitch    int     `json:"pitch"`
	Bgs      int     `json:"bgs"`
	Reg      int     `json:"reg"`
	Rdn      int     `json:"rdn"`
	Audio    WSAudio `json:"audio"`
}

// WSParameter WebSocket参数
type WSParameter struct {
	TTS WSTTS `json:"tts"`
}

// WSPayload WebSocket载荷
type WSPayload struct {
	Text struct {
		Encoding string `json:"encoding"`
		Compress string `json:"compress"`
		Format   string `json:"format"`
		Status   int    `json:"status"`
		Seq      int    `json:"seq"`
		Text     string `json:"text"`
	} `json:"text"`
}

// NewXunfeiTTSConfig 创建讯飞TTS配置
func NewXunfeiTTSConfig(appID, apiKey, apiSecret string) XunfeiTTSConfig {
	opt := XunfeiTTSConfig{
		AppID:         appID,
		APIKey:        apiKey,
		APISecret:     apiSecret,
		SampleRate:    24000,
		Channels:      1,
		BitDepth:      16,
		Codec:         "raw",
		FrameDuration: "20ms",
		Timeout:       30,
	}

	// 从环境变量获取默认值
	if opt.AppID == "" {
		opt.AppID = utils.GetEnv("XUNFEI_APP_ID")
	}
	if opt.APIKey == "" {
		opt.APIKey = utils.GetEnv("XUNFEI_API_KEY")
	}
	if opt.APISecret == "" {
		opt.APISecret = utils.GetEnv("XUNFEI_API_SECRET")
	}

	return opt
}

// NewXunfeiService 创建讯飞TTS服务
func NewXunfeiService(opt XunfeiTTSConfig) *XunfeiService {
	return &XunfeiService{
		opt: opt,
	}
}

func (xs *XunfeiService) Provider() TTSProvider {
	return ProviderXunfei
}

func (xs *XunfeiService) Format() media.StreamFormat {
	xs.mu.Lock()
	defer xs.mu.Unlock()
	return media.StreamFormat{
		SampleRate:    xs.opt.SampleRate,
		BitDepth:      xs.opt.BitDepth,
		Channels:      xs.opt.Channels,
		FrameDuration: media.NormalizeFramePeriod(xs.opt.FrameDuration),
	}
}

func (xs *XunfeiService) CacheKey(text string) string {
	xs.mu.Lock()
	defer xs.mu.Unlock()
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("xunfei.tts-%s-%d-%s.%s", "xunfei_default", xs.opt.SampleRate, digest, xs.opt.Codec)
}

func (xs *XunfeiService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	xs.mu.Lock()
	opt := xs.opt
	xs.mu.Unlock()

	if opt.AppID == "" {
		return fmt.Errorf("XUNFEI_APP_ID is required")
	}
	if opt.APIKey == "" {
		return fmt.Errorf("XUNFEI_API_KEY is required")
	}
	if opt.APISecret == "" {
		return fmt.Errorf("XUNFEI_API_SECRET is required")
	}

	// 生成 WebSocket 鉴权 URL
	host := "cn-huabei-1.xf-yun.com"
	path := "/v1/private/voice_clone"

	wsURL, err := generateWebSocketAuthURL(host, path, opt.APIKey, opt.APISecret)
	if err != nil {
		return fmt.Errorf("failed to generate WebSocket auth URL: %w", err)
	}

	// 连接到 WebSocket 服务
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket server: %w", err)
	}
	defer conn.Close()

	// 构建 WebSocket 请求
	wsReq := WSRequest{
		Header: WSHeader{
			AppID:  opt.AppID,
			Status: 2,
			ResID:  "xunfei_default", // 使用默认音色
		},
		Parameter: WSParameter{
			TTS: WSTTS{
				Vcn:      "x5_clone", // 使用语音克隆模型
				Volume:   8,          // 音量
				Rhy:      1,          // 节奏控制
				Pybuffer: 1,
				Speed:    50, // 语速
				Pitch:    50, // 音调
				Bgs:      0,
				Reg:      2, // 音色调节
				Rdn:      2, // 随机化
				Audio: WSAudio{
					Encoding:   "raw", // 使用 raw 编码
					SampleRate: 24000, // 采样率
				},
			},
		},
		Payload: WSPayload{
			Text: struct {
				Encoding string `json:"encoding"`
				Compress string `json:"compress"`
				Format   string `json:"format"`
				Status   int    `json:"status"`
				Seq      int    `json:"seq"`
				Text     string `json:"text"`
			}{
				Encoding: "utf8",
				Compress: "raw",
				Format:   "plain",
				Status:   2,
				Seq:      1,
				Text:     base64.StdEncoding.EncodeToString([]byte(text)),
			},
		},
	}

	// 发送请求
	message, err := json.Marshal(wsReq)
	if err != nil {
		return fmt.Errorf("failed to marshal WebSocket request: %w", err)
	}

	err = conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		return fmt.Errorf("failed to send WebSocket message: %w", err)
	}

	// 接收响应并处理音频数据
	var allAudioData []byte
	done := make(chan error, 1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				done <- ctx.Err()
				return
			default:
				_, response, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						done <- fmt.Errorf("failed to read WebSocket response: %w", err)
					}
					return
				}

				// 解析响应
				var responseData map[string]interface{}
				if err := json.Unmarshal(response, &responseData); err != nil {
					logrus.WithError(err).Error("failed to unmarshal WebSocket response")
					continue
				}

				// 提取音频数据
				if payload, exists := responseData["payload"].(map[string]interface{}); exists {
					if audioData, exists := payload["audio"].(map[string]interface{}); exists {
						// 处理音频数据
						if audioBase64, ok := audioData["audio"].(string); ok && audioBase64 != "" {
							// 解码音频数据
							decodedAudio, err := base64.StdEncoding.DecodeString(audioBase64)
							if err != nil {
								logrus.WithError(err).Error("failed to decode audio data")
								continue
							}

							// 发送音频数据到 handler
							handler.OnMessage(decodedAudio)

							// 累积音频数据
							allAudioData = append(allAudioData, decodedAudio...)
						}

						// 检查状态
						if status, ok := audioData["status"].(float64); ok {
							if status == 2 {
								// 音频合成完成
								done <- nil
								return
							}
						}
					}
				}
			}
		}
	}()

	// 等待完成或超时
	select {
	case err := <-done:
		if err != nil {
			return err
		}
		logrus.WithFields(logrus.Fields{
			"provider":   "xunfei",
			"text":       text,
			"audio_size": len(allAudioData),
		}).Info("xunfei tts: synthesis completed")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (xs *XunfeiService) Close() error {
	return nil
}

// generateWebSocketAuthURL 生成 WebSocket 鉴权 URL
func generateWebSocketAuthURL(host, path, apiKey, apiSecret string) (string, error) {
	if apiKey == "" || apiSecret == "" {
		return "", fmt.Errorf("WebSocket API credentials not configured")
	}

	// 1. 生成 date 参数 (RFC1123 格式)
	now := time.Now().UTC()
	date := now.Format(time.RFC1123)

	// 2. 构建 tmp 字符串
	tmp := fmt.Sprintf("host: %s\n", host)
	tmp += fmt.Sprintf("date: %s\n", date)
	tmp += fmt.Sprintf("GET %s HTTP/1.1", path)

	// 3. 使用 hmac-sha256 算法签名
	h := hmac.New(sha256.New, []byte(apiSecret))
	h.Write([]byte(tmp))
	tmpSha := h.Sum(nil)

	// 4. base64 编码生成 signature
	signature := base64.StdEncoding.EncodeToString(tmpSha)

	// 5. 生成 authorization_origin
	authorizationOrigin := fmt.Sprintf(`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`,
		apiKey, signature)

	// 6. base64 编码生成最终的 authorization
	authorization := base64.StdEncoding.EncodeToString([]byte(authorizationOrigin))

	// 7. 生成最终 URL
	params := url.Values{}
	params.Set("authorization", authorization)
	params.Set("date", date)
	params.Set("host", host)

	finalURL := fmt.Sprintf("wss://%s%s?%s", host, path, params.Encode())
	return finalURL, nil
}
