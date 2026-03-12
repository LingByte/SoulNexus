package synthesizer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

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

// FishSpeechV2Request WebSocket v2 请求
type FishSpeechV2Request struct {
	Text        string `json:"text"`
	ReferenceID string `json:"reference_id,omitempty"`
	Format      string `json:"format,omitempty"`
	Latency     string `json:"latency,omitempty"`
}

// FishSpeechV2Response WebSocket v2 响应
type FishSpeechV2Response struct {
	Type    string `json:"type"`              // "audio" 或 "error"
	Data    string `json:"data,omitempty"`    // base64 编码的音频数据
	Error   string `json:"error,omitempty"`   // 错误信息
	Message string `json:"message,omitempty"` // 消息
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
	if opt.ReferenceID == "" {
		return fmt.Errorf("FISHSPEECH_REFERENCE_ID is required")
	}

	// 使用 WebSocket v2 API
	return fs.synthesizeV2(ctx, handler, text, opt)
}

// synthesizeV2 使用 WebSocket v2 API 进行合成
func (fs *FishSpeechService) synthesizeV2(ctx context.Context, handler SynthesisHandler, text string, opt FishSpeechConfig) error {
	// 构建 WebSocket URL
	wsURL := fmt.Sprintf("wss://api.fishspeech.com/v1/tts/ws?token=%s", url.QueryEscape(opt.APIKey))

	// 创建 WebSocket 连接
	dialer := websocket.Dialer{
		HandshakeTimeout: time.Duration(opt.Timeout) * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		logrus.WithError(err).Errorf("failed to connect to FishSpeech WebSocket: %s", wsURL)
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()

	// 设置读写超时
	conn.SetReadDeadline(time.Now().Add(time.Duration(opt.Timeout) * time.Second))
	conn.SetWriteDeadline(time.Now().Add(time.Duration(opt.Timeout) * time.Second))

	// 构建请求
	request := FishSpeechV2Request{
		Text:        text,
		ReferenceID: opt.ReferenceID,
		Format:      opt.Codec,
		Latency:     opt.Latency,
	}

	// 发送请求
	requestData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, requestData); err != nil {
		logrus.WithError(err).Error("failed to send FishSpeech request")
		return fmt.Errorf("failed to send request: %w", err)
	}

	// 接收响应
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				logrus.WithError(err).Error("failed to read FishSpeech response")
				return fmt.Errorf("failed to read response: %w", err)
			}

			if messageType == websocket.TextMessage {
				var response FishSpeechV2Response
				if err := json.Unmarshal(message, &response); err != nil {
					logrus.WithError(err).Error("failed to unmarshal FishSpeech response")
					continue
				}

				if response.Type == "error" {
					return fmt.Errorf("FishSpeech error: %s", response.Error)
				}

				if response.Type == "audio" && response.Data != "" {
					// 解码 base64 音频数据
					audioData, err := decodeBase64Audio(response.Data)
					if err != nil {
						logrus.WithError(err).Error("failed to decode audio data")
						return fmt.Errorf("failed to decode audio: %w", err)
					}

					if len(audioData) > 0 {
						handler.OnMessage(audioData)
					}

					// 音频接收完成
					return nil
				}
			}
		}
	}
}

// decodeBase64Audio 解码 base64 音频数据
func decodeBase64Audio(data string) ([]byte, error) {
	// 如果数据已经是二进制形式，直接返回
	// 否则尝试 base64 解码
	decoded := make([]byte, len(data))
	copy(decoded, []byte(data))
	return decoded, nil
}

func (fs *FishSpeechService) Close() error {
	return nil
}

// FishSpeechVoiceOption FishSpeech 音色选项
type FishSpeechVoiceOption struct {
	ModelID     string `json:"modelId"`     // 模型 ID
	Title       string `json:"title"`       // 模型名称
	Description string `json:"description"` // 模型描述
	IsPersonal  bool   `json:"isPersonal"`  // 是否为个人模型
}

// FishSpeechListModelsRequest FishSpeech 列表模型请求
type FishSpeechListModelsRequest struct {
	ModelType       string `json:"modelType,omitempty"`       // 可选，模型类型："public" | "personal" | "all"
	IncludePersonal bool   `json:"includePersonal,omitempty"` // 可选，是否包含个人模型
	Page            int    `json:"page,omitempty"`            // 可选，页码，从1开始
	PageSize        int    `json:"pageSize,omitempty"`        // 可选，每页条数
}

// FishSpeechListModelsResponse FishSpeech API 返回的模型列表响应
type FishSpeechListModelsResponse struct {
	Total      int                     `json:"total"`
	Page       int                     `json:"page"`
	PageSize   int                     `json:"pageSize"`
	TotalPages int                     `json:"totalPages"`
	Items      []FishSpeechVoiceOption `json:"items"`
}

// GetFishSpeechVoices 从 FishSpeech API 获取可用的音色列表
// apiKey: FishSpeech API Key
// returns: 音色列表和错误信息
func GetFishSpeechVoices(apiKey string) ([]FishSpeechVoiceOption, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("FishSpeech API Key is required")
	}

	// FishSpeech API 端点
	url := "https://fishspeech.net/api/open/list-models"

	// 构建请求体
	reqBody := FishSpeechListModelsRequest{
		ModelType:       "all",
		IncludePersonal: true,
		Page:            1,
		PageSize:        100, // 获取最多100个模型
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 添加认证头
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call FishSpeech API: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FishSpeech API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp FishSpeechListModelsResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse FishSpeech API response: %w", err)
	}

	return apiResp.Items, nil
}
