package synthesizer

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// FishSpeechConfig FishSpeech TTS配置
type FishSpeechConfig struct {
	APIKey        string `json:"api_key" yaml:"api_key" env:"FISHSPEECH_API_KEY"`
	ReferenceID   string `json:"reference_id" yaml:"reference_id" default:"default"` // 模型ID
	SampleRate    int    `json:"sample_rate" yaml:"sample_rate" default:"24000"`
	Channels      int    `json:"channels" yaml:"channels" default:"1"`
	BitDepth      int    `json:"bit_depth" yaml:"bit_depth" default:"16"`
	Codec         string `json:"codec" yaml:"codec" default:"wav"`
	FrameDuration string `json:"frame_duration" yaml:"frame_duration" default:"20ms"`
	Timeout       int    `json:"timeout" yaml:"timeout" default:"30"`
	Latency       string `json:"latency" yaml:"latency" default:"normal"` // normal, balanced
	Version       string `json:"version" yaml:"version" default:"s1"`
}

type FishSpeechService struct {
	opt FishSpeechConfig
	mu  sync.Mutex // 保护 opt 的并发访问
}

// ClientEvent 客户端事件
type ClientEvent struct {
	Event   string                `json:"event"`
	Token   string                `json:"token,omitempty"`
	Request *FishSpeechTTSRequest `json:"request,omitempty"`
	Text    string                `json:"text,omitempty"`
}

// FishSpeechTTSRequest TTS请求配置
type FishSpeechTTSRequest struct {
	ReferenceID string `json:"reference_id,omitempty"`
	Latency     string `json:"latency,omitempty"`
	Format      string `json:"format,omitempty"`
	Version     string `json:"version,omitempty"`
}

// ServerEvent 服务器事件
type ServerEvent struct {
	Event   string `json:"event"`
	Message string `json:"message,omitempty"`
	Format  string `json:"format,omitempty"`
	Text    string `json:"text,omitempty"`
}

// NewFishSpeechConfig 创建 FishSpeech TTS 配置
func NewFishSpeechConfig(apiKey, referenceID string) FishSpeechConfig {
	opt := FishSpeechConfig{
		APIKey:        apiKey,
		ReferenceID:   referenceID,
		SampleRate:    24000,
		Channels:      1,
		BitDepth:      16,
		Codec:         "wav",
		FrameDuration: "20ms",
		Timeout:       30,
		Latency:       "normal",
		Version:       "s1",
	}

	// 从环境变量获取默认值
	if opt.APIKey == "" {
		opt.APIKey = utils.GetEnv("FISHSPEECH_API_KEY")
	}
	if opt.ReferenceID == "" {
		opt.ReferenceID = "default"
	}

	return opt
}

// NewFishSpeechService 创建 FishSpeech TTS 服务
func NewFishSpeechService(opt FishSpeechConfig) *FishSpeechService {
	return &FishSpeechService{
		opt: opt,
	}
}

func (fs *FishSpeechService) Provider() TTSProvider {
	return ProviderFishSpeech
}

func (fs *FishSpeechService) Format() media.StreamFormat {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return media.StreamFormat{
		SampleRate:    fs.opt.SampleRate,
		BitDepth:      fs.opt.BitDepth,
		Channels:      fs.opt.Channels,
		FrameDuration: utils.NormalizeFramePeriod(fs.opt.FrameDuration),
	}
}

func (fs *FishSpeechService) CacheKey(text string) string {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("fishspeech.tts-%s-%d-%s.%s", fs.opt.ReferenceID, fs.opt.SampleRate, digest, fs.opt.Codec)
}

func (fs *FishSpeechService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	fs.mu.Lock()
	opt := fs.opt
	fs.mu.Unlock()

	// 验证配置
	if opt.APIKey == "" {
		return fmt.Errorf("FISHSPEECH_API_KEY is required")
	}

	// 生成鉴权 token
	token := generateAuthToken(opt.APIKey)

	// 构建 WebSocket URL
	wsURL := fmt.Sprintf("wss://fishspeech.live/v1/tts/ws?token=%s", url.QueryEscape(token))

	// 连接到 WebSocket
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()

	// 创建 TTS 请求
	ttsRequest := &FishSpeechTTSRequest{
		ReferenceID: opt.ReferenceID,
		Latency:     opt.Latency,
		Format:      opt.Codec,
		Version:     opt.Version,
	}

	// 发送 start 事件
	err = sendStartEvent(conn, token, ttsRequest)
	if err != nil {
		return fmt.Errorf("failed to send start event: %w", err)
	}

	// 等待 ready 事件
	if err := waitForReady(conn); err != nil {
		return fmt.Errorf("failed to wait for ready: %w", err)
	}

	// 发送文本
	err = sendTextEvent(conn, text)
	if err != nil {
		return fmt.Errorf("failed to send text event: %w", err)
	}

	// 接收音频数据
	audioData, err := receiveAudioData(ctx, conn)
	if err != nil {
		return fmt.Errorf("failed to receive audio data: %w", err)
	}

	// 发送音频数据到 handler
	if len(audioData) > 0 {
		handler.OnMessage(audioData)
	}

	logrus.WithFields(logrus.Fields{
		"provider":   "fishspeech",
		"text":       text,
		"audio_size": len(audioData),
	}).Info("fishspeech tts: synthesis completed")

	return nil
}

func (fs *FishSpeechService) Close() error {
	return nil
}

// generateAuthToken 生成鉴权 token
func generateAuthToken(apiKey string) string {
	// 这里使用 API key 本身作为 token
	// 在实际应用中，可能需要更复杂的鉴权机制
	hash := sha256.Sum256([]byte(apiKey))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// sendStartEvent 发送 start 事件
func sendStartEvent(conn *websocket.Conn, token string, request *FishSpeechTTSRequest) error {
	event := ClientEvent{
		Event:   "start",
		Token:   token,
		Request: request,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

// waitForReady 等待 ready 事件
func waitForReady(conn *websocket.Conn) error {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var event ServerEvent
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}

		if event.Event == "ready" {
			return nil
		}
	}
}

// sendTextEvent 发送文本事件
func sendTextEvent(conn *websocket.Conn, text string) error {
	event := ClientEvent{
		Event: "text",
		Text:  text,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

// receiveAudioData 接收音频数据
func receiveAudioData(ctx context.Context, conn *websocket.Conn) ([]byte, error) {
	var audioData []byte

	for {
		select {
		case <-ctx.Done():
			return audioData, ctx.Err()
		default:
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return audioData, err
			}

			switch messageType {
			case websocket.TextMessage:
				// 处理文本事件
				var event ServerEvent
				if err := json.Unmarshal(message, &event); err == nil {
					if event.Event == "error" {
						return audioData, fmt.Errorf("server error: %s", event.Message)
					}
					// 其他文本事件可以在这里处理
				}
			case websocket.BinaryMessage:
				// 接收音频数据
				audioData = append(audioData, message...)

				// 检查是否完成（可以根据实际情况调整）
				if len(audioData) > 1024 {
					return audioData, nil
				}
			}
		}
	}
}
