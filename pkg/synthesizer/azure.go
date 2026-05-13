package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	_ "encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/sirupsen/logrus"
)

// AzureConfig Azure TTS配置
type AzureConfig struct {
	SubscriptionKey string `json:"subscription_key" yaml:"subscription_key" env:"AZURE_SUBSCRIPTION_KEY"`
	Region          string `json:"region" yaml:"region" env:"AZURE_REGION"`
	Voice           string `json:"voice" yaml:"voice" default:"zh-CN-XiaoxiaoNeural"`
	Language        string `json:"language" yaml:"language"` // 语言代码，用于 SSML 的 xml:lang
	SampleRate      int    `json:"sample_rate" yaml:"sample_rate" default:"22050"`
	Channels        int    `json:"channels" yaml:"channels" default:"1"`
	BitDepth        int    `json:"bit_depth" yaml:"bit_depth" default:"16"`
	Codec           string `json:"codec" yaml:"codec" default:"audio-24khz-48kbitrate-mono-mp3"`
	FrameDuration   string `json:"frame_duration" yaml:"frame_duration" default:"20ms"`
	Timeout         int    `json:"timeout" yaml:"timeout" default:"30"`
	BaseURL         string `json:"base_url" yaml:"base_url"`
}

type AzureService struct {
	opt    AzureConfig
	mu     sync.Mutex // 保护 opt 的并发访问
	client *http.Client
}

// AzureRequest Azure TTS API 请求
type AzureRequest struct {
	Text string `json:"text"`
}

const (
	azureTTSURLTemplate = "https://%s.tts.speech.microsoft.com/cognitiveservices/v1"
)

// NewAzureConfig 创建 Azure TTS 配置
func NewAzureConfig(subscriptionKey, region string) AzureConfig {
	opt := AzureConfig{
		SubscriptionKey: subscriptionKey,
		Region:          region,
		Voice:           "zh-CN-XiaoxiaoNeural",
		SampleRate:      22050,
		Channels:        1,
		BitDepth:        16,
		Codec:           "audio-24khz-48kbitrate-mono-mp3",
		FrameDuration:   "20ms",
		Timeout:         30,
	}

	// 从环境变量获取默认值
	if opt.SubscriptionKey == "" {
		opt.SubscriptionKey = utils.GetEnv("AZURE_SUBSCRIPTION_KEY")
	}
	if opt.Region == "" {
		opt.Region = utils.GetEnv("AZURE_REGION")
	}

	return opt
}

// NewAzureService 创建 Azure TTS 服务
func NewAzureService(opt AzureConfig) *AzureService {
	return &AzureService{
		opt:    opt,
		client: &http.Client{},
	}
}

func (as *AzureService) Provider() TTSProvider {
	return ProviderAzure
}

func (as *AzureService) Format() media.StreamFormat {
	as.mu.Lock()
	defer as.mu.Unlock()
	return media.StreamFormat{
		SampleRate:    as.opt.SampleRate,
		BitDepth:      as.opt.BitDepth,
		Channels:      as.opt.Channels,
		FrameDuration: utils.NormalizeFramePeriod(as.opt.FrameDuration),
	}
}

func (as *AzureService) CacheKey(text string) string {
	as.mu.Lock()
	defer as.mu.Unlock()
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("azure.tts-%s-%s-%d-%s.%s", as.opt.Voice, as.opt.Region, as.opt.SampleRate, digest, "mp3")
}

func (as *AzureService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	as.mu.Lock()
	opt := as.opt
	as.mu.Unlock()

	// 验证配置
	if opt.SubscriptionKey == "" {
		return fmt.Errorf("AZURE_SUBSCRIPTION_KEY is required")
	}
	if opt.Region == "" {
		return fmt.Errorf("AZURE_REGION is required")
	}

	// 确定语言代码（用于 SSML 的 xml:lang）
	lang := opt.Language
	if lang == "" {
		// 如果没有指定语言，尝试从 voice 名称中提取（例如：zh-CN-XiaoxiaoNeural -> zh-CN）
		if parts := strings.Split(opt.Voice, "-"); len(parts) >= 2 {
			lang = parts[0] + "-" + parts[1]
		} else {
			lang = "zh-CN" // 默认值
		}
	}

	// 构建 SSML
	ssml := fmt.Sprintf(`<speak version='1.0' xml:lang='%s'>
	<voice xml:lang='%s' xml:gender='Female' name='%s'>
		%s
	</voice>
</speak>`, lang, lang, opt.Voice, text)

	// 构建 URL
	url := fmt.Sprintf(azureTTSURLTemplate, opt.Region)

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(ssml))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/ssml+xml")
	httpReq.Header.Set("X-Microsoft-OutputFormat", opt.Codec)
	httpReq.Header.Set("User-Agent", "SoulNexus")
	httpReq.Header.Set("Ocp-Apim-Subscription-Key", opt.SubscriptionKey)

	// 生成鉴权头（可选，部分区域可能需要）
	authHeader, err := generateAzureAuthHeader(opt)
	if err == nil && authHeader != "" {
		httpReq.Header.Set("Authorization", authHeader)
	}

	// 发送请求
	resp, err := as.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("TTS request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 读取音频数据
	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read audio data: %w", err)
	}

	// 发送音频数据到 handler
	if len(audioData) > 0 {
		handler.OnMessage(audioData)
	}

	logrus.WithFields(logrus.Fields{
		"provider":   "azure",
		"text":       text,
		"audio_size": len(audioData),
		"voice":      opt.Voice,
		"region":     opt.Region,
	}).Info("azure tts: synthesis completed")

	return nil
}

func (as *AzureService) Close() error {
	if as.client != nil {
		as.client.CloseIdleConnections()
	}
	return nil
}

// generateAzureAuthHeader 生成 Azure 鉴权头
func generateAzureAuthHeader(opt AzureConfig) (string, error) {
	// Azure 鉴权可以使用简单的 Subscription Key，也可以使用更复杂的 OAuth 2.0
	// 这里返回空字符串，使用 Subscription Key 即可
	return "", nil
}

// GetAzureVoices 获取可用的 Azure 音色列表（示例）
func GetAzureVoices() map[string]string {
	return map[string]string{
		// 中文音色
		"zh-CN-XiaoxiaoNeural":   "晓晓（女声，温暖友好）",
		"zh-CN-YunxiNeural":      "云希（男声，温和）",
		"zh-CN-YunjianNeural":    "云健（男声，活泼）",
		"zh-CN-XiaoyiNeural":     "晓伊（女声，温柔）",
		"zh-CN-YunyangNeural":    "云扬（男声，专业）",
		"zh-CN-XiaochenNeural":   "晓辰（女声，甜美）",
		"zh-CN-XiaohanNeural":    "晓涵（女声，柔和）",
		"zh-CN-XiaomengNeural":   "晓梦（女声，活泼）",
		"zh-CN-XiaomoNeural":     "晓墨（女声，沉稳）",
		"zh-CN-XiaoqiuNeural":    "晓秋（女声，知性）",
		"zh-CN-XiaoruiNeural":    "晓睿（女声，睿智）",
		"zh-CN-XiaoshuangNeural": "晓双（女声，俏皮）",
		"zh-CN-XiaoxuanNeural":   "晓萱（女声，清新）",
		"zh-CN-XiaoyanNeural":    "晓颜（女声，优雅）",
		"zh-CN-XiaoyouNeural":    "晓悠（女声，自然）",
		"zh-CN-XiaozhenNeural":   "晓甄（女声，专业）",
		"zh-CN-YunfengNeural":    "云枫（男声，成熟）",
		"zh-CN-YunhaoNeural":     "云皓（男声，阳光）",
		"zh-CN-YunxiaNeural":     "云霞（女声，优雅）",
		"zh-CN-YunyeNeural":      "云烨（男声，磁性）",
		"zh-CN-YunzeNeural":      "云泽（男声，沉稳）",
		"zh-CN-YunzheNeural":     "云哲（男声，睿智）",
		"zh-CN-YunzhengNeural":   "云正（男声，正式）",
		// 英文音色
		"en-US-AriaNeural":  "Aria (美国女声)",
		"en-US-JennyNeural": "Jenny (美国女声)",
		"en-US-GuyNeural":   "Guy (美国男声)",
		"en-US-TonyNeural":  "Tony (美国男声)",
	}
}
