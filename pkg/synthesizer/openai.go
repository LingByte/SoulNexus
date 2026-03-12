package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/sirupsen/logrus"
)

// OpenAIConfig OpenAI TTS配置
type OpenAIConfig struct {
	APIKey        string  `json:"api_key" yaml:"api_key" env:"OPENAI_API_KEY"`
	Model         string  `json:"model" yaml:"model" default:"tts-1"`
	Voice         string  `json:"voice" yaml:"voice" default:"alloy"`
	Speed         float64 `json:"speed" yaml:"speed" default:"1.0"`
	SampleRate    int     `json:"sample_rate" yaml:"sample_rate" default:"24000"`
	Channels      int     `json:"channels" yaml:"channels" default:"1"`
	BitDepth      int     `json:"bit_depth" yaml:"bit_depth" default:"16"`
	Codec         string  `json:"codec" yaml:"codec" default:"mp3"`
	FrameDuration string  `json:"frame_duration" yaml:"frame_duration" default:"20ms"`
	Timeout       int     `json:"timeout" yaml:"timeout" default:"30"`
	BaseURL       string  `json:"base_url" yaml:"base_url" default:"https://api.openai.com"`
}

type OpenAIService struct {
	opt    OpenAIConfig
	mu     sync.Mutex // 保护 opt 的并发访问
	client *http.Client
}

// OpenAIRequest OpenAI API 请求
type OpenAIRequest struct {
	Model          string  `json:"model"`
	Input          string  `json:"input"`
	Voice          string  `json:"voice"`
	ResponseFormat string  `json:"response_format,omitempty"`
	Speed          float64 `json:"speed,omitempty"`
}

const (
	openAITTSURL = "/v1/audio/speech"
)

// NewOpenAIConfig 创建 OpenAI TTS 配置
func NewOpenAIConfig(apiKey string) OpenAIConfig {
	opt := OpenAIConfig{
		APIKey:        apiKey,
		Model:         "tts-1",
		Voice:         "alloy",
		Speed:         1.0,
		SampleRate:    24000,
		Channels:      1,
		BitDepth:      16,
		Codec:         "mp3",
		FrameDuration: "20ms",
		Timeout:       30,
		BaseURL:       "https://api.openai.com",
	}

	// 从环境变量获取默认值
	if opt.APIKey == "" {
		opt.APIKey = utils.GetEnv("OPENAI_API_KEY")
	}

	return opt
}

// NewOpenAIService 创建 OpenAI TTS 服务
func NewOpenAIService(opt OpenAIConfig) *OpenAIService {
	return &OpenAIService{
		opt:    opt,
		client: &http.Client{},
	}
}

func (os *OpenAIService) Provider() TTSProvider {
	return ProviderOpenAI
}

func (os *OpenAIService) Format() media.StreamFormat {
	os.mu.Lock()
	defer os.mu.Unlock()
	return media.StreamFormat{
		SampleRate:    os.opt.SampleRate,
		BitDepth:      os.opt.BitDepth,
		Channels:      os.opt.Channels,
		FrameDuration: utils.NormalizeFramePeriod(os.opt.FrameDuration),
	}
}

func (os *OpenAIService) CacheKey(text string) string {
	os.mu.Lock()
	defer os.mu.Unlock()
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("openai.tts-%s-%s-%d-%s.%s", os.opt.Model, os.opt.Voice, os.opt.SampleRate, digest, os.opt.Codec)
}

func (os *OpenAIService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	os.mu.Lock()
	opt := os.opt
	os.mu.Unlock()

	// 验证配置
	if opt.APIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	// 构建请求
	req := OpenAIRequest{
		Model:          opt.Model,
		Input:          text,
		Voice:          opt.Voice,
		ResponseFormat: opt.Codec,
		Speed:          opt.Speed,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	url := opt.BaseURL + openAITTSURL
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+opt.APIKey)

	// 发送请求
	resp, err := os.client.Do(httpReq)
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
		"provider":   "openai",
		"text":       text,
		"audio_size": len(audioData),
		"model":      opt.Model,
		"voice":      opt.Voice,
	}).Info("openai tts: synthesis completed")

	return nil
}

func (os *OpenAIService) Close() error {
	if os.client != nil {
		os.client.CloseIdleConnections()
	}
	return nil
}
