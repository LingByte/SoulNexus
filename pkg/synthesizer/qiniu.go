package synthesizer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/sirupsen/logrus"
)

// QiniuTTSConfig 七牛云TTS配置
type QiniuTTSConfig struct {
	APIKey        string `json:"api_key" yaml:"api_key" env:"QINIU_TTS_API_KEY"`
	BaseURL       string `json:"base_url" yaml:"base_url" env:"QINIU_TTS_BASE_URL"`
	VoiceType     string `json:"voice_type" yaml:"voice_type" default:"female_cn_001"`
	SampleRate    int    `json:"sample_rate" yaml:"sample_rate" default:"16000"`
	Channels      int    `json:"channels" yaml:"channels" default:"1"`
	BitDepth      int    `json:"bit_depth" yaml:"bit_depth" default:"16"`
	Codec         string `json:"codec" yaml:"codec" default:"pcm"`
	FrameDuration string `json:"frame_duration" yaml:"frame_duration" default:"20ms"`
	Timeout       int    `json:"timeout" yaml:"timeout" default:"30"`
	Retries       int    `json:"retries" yaml:"retries" default:"0"`
}

type QiniuService struct {
	opt QiniuTTSConfig
	mu  sync.Mutex // 保护 opt 的并发访问
}

// NewQiniuTTSConfig 创建七牛云TTS配置
func NewQiniuTTSConfig(apiKey, baseURL string) QiniuTTSConfig {
	opt := QiniuTTSConfig{
		APIKey:        apiKey,
		BaseURL:       baseURL,
		VoiceType:     "qiniu_zh_female_tmjxxy",
		Codec:         "pcm",
		SampleRate:    30000,
		Channels:      1,
		BitDepth:      16,
		FrameDuration: "20ms",
		Timeout:       30,
		Retries:       0,
	}

	// 从环境变量获取默认值
	if opt.APIKey == "" {
		opt.APIKey = utils.GetEnv("QINIU_TTS_API_KEY")
	}
	if opt.BaseURL == "" {
		opt.BaseURL = utils.GetEnv("QINIU_TTS_BASE_URL")
	}
	if opt.BaseURL == "" {
		opt.BaseURL = "https://openai.qiniu.com/v1"
	}

	// 解析超时时间
	if timeout := utils.GetIntEnv("QINIU_TTS_TIMEOUT"); timeout > 0 {
		opt.Timeout = int(timeout)
	}

	// 解析重试次数
	if retries := utils.GetIntEnv("QINIU_TTS_RETRIES"); retries > 0 {
		opt.Retries = int(retries)
	}

	return opt
}

// NewQiniuService 创建七牛云TTS服务
func NewQiniuService(opt QiniuTTSConfig) *QiniuService {
	return &QiniuService{
		opt: opt,
	}
}

func (qs *QiniuService) Provider() TTSProvider {
	return ProviderQiniu
}

func (qs *QiniuService) Format() media.StreamFormat {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	return media.StreamFormat{
		SampleRate:    qs.opt.SampleRate,
		BitDepth:      qs.opt.BitDepth,
		Channels:      qs.opt.Channels,
		FrameDuration: utils.NormalizeFramePeriod(qs.opt.FrameDuration),
	}
}

func (qs *QiniuService) CacheKey(text string) string {
	qs.mu.Lock()
	defer qs.mu.Unlock()
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("qiniu.tts-%s-%d-%s.%s", qs.opt.VoiceType, qs.opt.SampleRate, digest, qs.opt.Codec)
}

func (qs *QiniuService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	qs.mu.Lock()
	// 创建临时配置以避免在合成过程中被修改
	opt := qs.opt
	qs.mu.Unlock()

	// 验证 API Key
	if opt.APIKey == "" {
		return fmt.Errorf("QINIU_TTS_API_KEY is required")
	}

	// 创建 HTTP 客户端
	client := &http.Client{
		Timeout: time.Duration(opt.Timeout) * time.Second,
	}

	// 准备请求
	req := QiniuTTSRequest{
		Audio: TTSAudio{
			VoiceType:  opt.VoiceType,
			Encoding:   opt.Codec,
			SpeedRatio: 1.0,
		},
		Request: TTSRequestData{
			Text: text,
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 执行请求（带重试）
	var lastErr error
	retries := opt.Retries
	if retries < 0 {
		retries = 0
	}

	for attempt := 0; attempt <= retries; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, "POST", opt.BaseURL+"/voice/tts", bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+opt.APIKey)

		resp, err := client.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
		} else {
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				lastErr = fmt.Errorf("TTS request failed with status %d: %s", resp.StatusCode, string(body))
			} else {
				var ttsResp QiniuTTSResponse
				if err := json.NewDecoder(resp.Body).Decode(&ttsResp); err != nil {
					lastErr = fmt.Errorf("failed to decode response: %w", err)
				} else {
					// 解析音频数据
					audioData, err := base64.StdEncoding.DecodeString(ttsResp.Data)
					if err != nil {
						return fmt.Errorf("failed to decode base64 audio data: %w", err)
					}

					// 发送音频数据到 handler
					handler.OnMessage(audioData)

					// 如果有时间戳信息，发送到 handler
					if ttsResp.Addition != nil && ttsResp.Addition.Duration != "" {
						// 解析时间戳（简化版本，实际可能需要更复杂的解析）
						timestamp := parseTimestamp(ttsResp.Addition.Duration, text)
						handler.OnTimestamp(timestamp)
					}

					logrus.WithFields(logrus.Fields{
						"provider":   "qiniu",
						"text":       text,
						"audio_size": len(audioData),
					}).Info("qiniu tts: synthesis completed")

					return nil
				}
			}
		}

		// 如果不是最后一次尝试，等待后重试
		if attempt < retries {
			time.Sleep(backoffDuration(attempt))
		}
	}

	return lastErr
}

func (qs *QiniuService) Close() error {
	return nil
}

// QiniuTTSRequest 七牛云TTS请求结构
type QiniuTTSRequest struct {
	Audio   TTSAudio       `json:"audio"`
	Request TTSRequestData `json:"request"`
}

// TTSAudio TTS音频配置
type TTSAudio struct {
	VoiceType  string  `json:"voice_type"`
	Encoding   string  `json:"encoding"`
	SpeedRatio float64 `json:"speed_ratio,omitempty"`
}

// TTSRequestData TTS请求数据
type TTSRequestData struct {
	Text string `json:"text"`
}

// QiniuTTSResponse 七牛云TTS响应结构
type QiniuTTSResponse struct {
	Reqid     string       `json:"reqid"`
	Operation string       `json:"operation"`
	Sequence  int          `json:"sequence"`
	Data      string       `json:"data"`
	Addition  *TTSAddition `json:"addition,omitempty"`
}

// TTSAddition TTS附加信息
type TTSAddition struct {
	Duration string `json:"duration"`
}

// backoffDuration 计算退避时间
func backoffDuration(attempt int) time.Duration {
	base := 200 * time.Millisecond
	d := base * time.Duration(1<<uint(attempt))
	if d > 2*time.Second {
		return 2 * time.Second
	}
	return d
}

// parseTimestamp 解析时间戳信息
func parseTimestamp(duration string, text string) SentenceTimestamp {
	// 简化版本：将整个文本作为一个单词处理
	// 实际应用中可以根据七牛云的响应解析更详细的时间戳
	words := []Word{
		{
			Word:       text,
			Confidence: 1.0,
			StartTime:  0,
			EndTime:    parseDurationToMs(duration),
		},
	}

	return SentenceTimestamp{
		Words: words,
	}
}

// parseDurationToMs 将持续时间字符串转换为毫秒
func parseDurationToMs(duration string) int {
	// 尝试解析为秒数字符串
	if seconds, err := parseFloat(duration); err == nil {
		return int(seconds * 1000)
	}

	// 尝试解析为时间格式
	if d, err := time.ParseDuration(duration); err == nil {
		return int(d.Milliseconds())
	}

	return 0
}

// parseFloat 解析浮点数字符串
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	// 移除单位（如 "s", "ms" 等）
	s = regexp.MustCompile(`[a-zA-Z]+`).ReplaceAllString(s, "")
	var f float64
	var err error
	if f, err = parseDurationToFloat(s); err != nil {
		return 0, fmt.Errorf("unable to parse float: %s", s)
	}
	return f, nil
}

// parseDurationToFloat 解析时长字符串为浮点数（秒）
func parseDurationToFloat(s string) (float64, error) {
	// 尝试直接解析为浮点数
	s = strings.TrimSpace(s)
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
