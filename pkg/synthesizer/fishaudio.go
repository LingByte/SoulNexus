package synthesizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/sirupsen/logrus"
)

// FishAudioConfig Fish Audio TTS 配置
type FishAudioConfig struct {
	APIKey      string  `json:"api_key" yaml:"api_key" env:"FISHAUDIO_API_KEY"`
	ReferenceID string  `json:"reference_id" yaml:"reference_id" default:""` // 模型ID
	Model       string  `json:"model" yaml:"model" default:"s1"`             // 模型版本: s1, speech-1.6, speech-1.5
	SampleRate  int     `json:"sample_rate" yaml:"sample_rate" default:"44100"`
	Channels    int     `json:"channels" yaml:"channels" default:"1"`
	BitDepth    int     `json:"bit_depth" yaml:"bit_depth" default:"16"`
	Format      string  `json:"format" yaml:"format" default:"mp3"` // wav, pcm, mp3, opus
	Temperature float64 `json:"temperature" yaml:"temperature" default:"0.7"`
	TopP        float64 `json:"top_p" yaml:"top_p" default:"0.7"`
	Latency     string  `json:"latency" yaml:"latency" default:"normal"` // low, normal, balanced
	ChunkLength int     `json:"chunk_length" yaml:"chunk_length" default:"300"`
	Normalize   bool    `json:"normalize" yaml:"normalize" default:"true"`
	MPEGBitrate int     `json:"mp3_bitrate" yaml:"mp3_bitrate" default:"128"` // 64, 128, 192
	Timeout     int     `json:"timeout" yaml:"timeout" default:"30"`
}

type FishAudioService struct {
	opt FishAudioConfig
	mu  sync.Mutex
}

// FishAudioRequest Fish Audio TTS 请求
type FishAudioRequest struct {
	Text                      string  `json:"text"`
	Model                     string  `json:"model"`
	ReferenceID               string  `json:"reference_id,omitempty"`
	Temperature               float64 `json:"temperature,omitempty"`
	TopP                      float64 `json:"top_p,omitempty"`
	Format                    string  `json:"format,omitempty"`
	SampleRate                *int    `json:"sample_rate,omitempty"`
	ChunkLength               int     `json:"chunk_length,omitempty"`
	Normalize                 bool    `json:"normalize,omitempty"`
	Latency                   string  `json:"latency,omitempty"`
	MaxNewTokens              int     `json:"max_new_tokens,omitempty"`
	RepetitionPenalty         float64 `json:"repetition_penalty,omitempty"`
	MinChunkLength            int     `json:"min_chunk_length,omitempty"`
	ConditionOnPreviousChunks bool    `json:"condition_on_previous_chunks,omitempty"`
	EarlyStopThreshold        float64 `json:"early_stop_threshold,omitempty"`
}

// NewFishAudioConfig 创建 Fish Audio TTS 配置
func NewFishAudioConfig(apiKey, referenceID string) FishAudioConfig {
	opt := FishAudioConfig{
		APIKey:      apiKey,
		ReferenceID: referenceID,
		Model:       "s1",
		SampleRate:  44100,
		Channels:    1,
		BitDepth:    16,
		Format:      "mp3",
		Temperature: 0.7,
		TopP:        0.7,
		Latency:     "normal",
		ChunkLength: 300,
		Normalize:   true,
		MPEGBitrate: 128,
		Timeout:     30,
	}

	// 从环境变量获取默认值
	if opt.APIKey == "" {
		opt.APIKey = utils.GetEnv("FISHAUDIO_API_KEY")
	}

	return opt
}

// NewFishAudioService 创建 Fish Audio TTS 服务
func NewFishAudioService(opt FishAudioConfig) *FishAudioService {
	return &FishAudioService{
		opt: opt,
	}
}

func (fa *FishAudioService) Provider() TTSProvider {
	return ProviderFishAudio
}

func (fa *FishAudioService) Format() media.StreamFormat {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	sampleRate := fa.opt.SampleRate
	if fa.opt.Format == "opus" {
		sampleRate = 48000
	}

	return media.StreamFormat{
		SampleRate:    sampleRate,
		BitDepth:      fa.opt.BitDepth,
		Channels:      fa.opt.Channels,
		FrameDuration: 20,
	}
}

func (fa *FishAudioService) CacheKey(text string) string {
	fa.mu.Lock()
	defer fa.mu.Unlock()
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("fishaudio.tts-%s-%s-%d-%s.%s", fa.opt.Model, fa.opt.ReferenceID, fa.opt.SampleRate, digest, fa.opt.Format)
}

func (fa *FishAudioService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	fa.mu.Lock()
	opt := fa.opt
	fa.mu.Unlock()

	// 验证配置
	if opt.APIKey == "" {
		return fmt.Errorf("FISHAUDIO_API_KEY is required")
	}

	// 调用 Fish Audio API
	return fa.synthesizeWithAPI(ctx, handler, text, opt)
}

// synthesizeWithAPI 使用 Fish Audio REST API 进行合成
func (fa *FishAudioService) synthesizeWithAPI(ctx context.Context, handler SynthesisHandler, text string, opt FishAudioConfig) error {
	// 构建请求
	sampleRate := opt.SampleRate
	if opt.Format == "opus" {
		sampleRate = 48000
	}

	request := FishAudioRequest{
		Text:                      text,
		Model:                     opt.Model,
		ReferenceID:               opt.ReferenceID,
		Temperature:               opt.Temperature,
		TopP:                      opt.TopP,
		Format:                    opt.Format,
		SampleRate:                &sampleRate,
		ChunkLength:               opt.ChunkLength,
		Normalize:                 opt.Normalize,
		Latency:                   opt.Latency,
		MaxNewTokens:              1024,
		RepetitionPenalty:         1.2,
		MinChunkLength:            50,
		ConditionOnPreviousChunks: true,
		EarlyStopThreshold:        1.0,
	}

	// 序列化请求
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.fish.audio/v1/tts", bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", opt.APIKey))
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	client := &http.Client{
		Timeout: time.Duration(opt.Timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		logrus.WithError(err).Error("failed to call Fish Audio API")
		return fmt.Errorf("failed to call Fish Audio API: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logrus.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"body":        string(body),
		}).Error("Fish Audio API error")
		return fmt.Errorf("Fish Audio API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 读取音频数据
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if len(audioData) == 0 {
		return fmt.Errorf("empty audio data from Fish Audio API")
	}

	// 发送音频数据到 handler
	handler.OnMessage(audioData)

	logrus.WithFields(logrus.Fields{
		"provider":   "fishaudio",
		"model":      opt.Model,
		"text":       text,
		"audio_size": len(audioData),
		"format":     opt.Format,
	}).Info("fishaudio tts: synthesis completed")

	return nil
}

func (fa *FishAudioService) Close() error {
	return nil
}

// FishAudioVoiceOption Fish Audio 音色选项
type FishAudioVoiceOption struct {
	ID          string   `json:"_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	State       string   `json:"state"`
	CoverImage  string   `json:"cover_image"`
	Languages   []string `json:"languages"`
	Author      struct {
		ID       string `json:"_id"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	} `json:"author"`
}

// FishAudioListModelsResponse Fish Audio API 返回的模型列表响应
type FishAudioListModelsResponse struct {
	Total int                    `json:"total"`
	Items []FishAudioVoiceOption `json:"items"`
}

// GetFishAudioVoices 从 Fish Audio API 获取可用的音色列表
// apiKey: Fish Audio API Key
// returns: 音色列表和错误信息
func GetFishAudioVoices(apiKey string) ([]FishAudioVoiceOption, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Fish Audio API Key is required")
	}

	// Fish Audio API 端点 - 获取 TTS 模型列表
	url := "https://api.fish.audio/model?type=tts&page_size=100&page_number=1"

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 添加认证头
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")

	// 执行请求
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Fish Audio API: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Fish Audio API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp FishAudioListModelsResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse Fish Audio API response: %w", err)
	}

	// 过滤出状态为 "trained" 的模型
	var trainedModels []FishAudioVoiceOption
	for _, model := range apiResp.Items {
		if model.State == "trained" {
			trainedModels = append(trainedModels, model)
		}
	}

	return trainedModels, nil
}
