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

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/utils"
	"github.com/sirupsen/logrus"
)

// ElevenLabsConfig ElevenLabs TTS配置
type ElevenLabsConfig struct {
	APIKey        string `json:"api_key" yaml:"api_key" env:"ELEVENLABS_API_KEY"`
	VoiceID       string `json:"voice_id" yaml:"voice_id" default:"21m00Tcm4TlvDq8ikWAM"` // 默认 Rachel 音色
	ModelID       string `json:"model_id" yaml:"model_id" default:"eleven_monolingual_v1"`
	LanguageCode  string `json:"language_code" yaml:"language_code"` // 语言代码，如 en, zh, ja 等
	SampleRate    int    `json:"sample_rate" yaml:"sample_rate" default:"44100"`
	Channels      int    `json:"channels" yaml:"channels" default:"1"`
	BitDepth      int    `json:"bit_depth" yaml:"bit_depth" default:"16"`
	Codec         string `json:"codec" yaml:"codec" default:"mp3"`
	FrameDuration string `json:"frame_duration" yaml:"frame_duration" default:"20ms"`
	Timeout       int    `json:"timeout" yaml:"timeout" default:"30"`
	// 语音设置
	Stability       float64 `json:"stability" yaml:"stability" default:"0.5"`                // 0.0-1.0
	SimilarityBoost float64 `json:"similarity_boost" yaml:"similarity_boost" default:"0.75"` // 0.0-1.0
	Style           float64 `json:"style" yaml:"style" default:"0.0"`                        // 0.0-1.0
	UseSpeakerBoost bool    `json:"use_speaker_boost" yaml:"use_speaker_boost" default:"true"`
}

type ElevenLabsService struct {
	opt    ElevenLabsConfig
	mu     sync.Mutex // 保护 opt 的并发访问
	client *http.Client
}

// ElevenLabsRequest ElevenLabs API 请求
type ElevenLabsRequest struct {
	Text          string                   `json:"text"`
	ModelID       string                   `json:"model_id,omitempty"`
	VoiceSettings *ElevenLabsVoiceSettings `json:"voice_settings,omitempty"`
	LanguageCode  string                   `json:"language_code,omitempty"`
}

// ElevenLabsVoiceSettings 音色设置
type ElevenLabsVoiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
	Style           float64 `json:"style"`
	UseSpeakerBoost bool    `json:"use_speaker_boost"`
}

const (
	elevenlabsBaseURL = "https://api.elevenlabs.io/v1"
	elevenlabsTTSURL  = elevenlabsBaseURL + "/text-to-speech/%s"
)

// NewElevenLabsConfig 创建 ElevenLabs TTS 配置
func NewElevenLabsConfig(apiKey, voiceID string) ElevenLabsConfig {
	opt := ElevenLabsConfig{
		APIKey:          apiKey,
		VoiceID:         voiceID,
		ModelID:         "eleven_monolingual_v1",
		SampleRate:      44100,
		Channels:        1,
		BitDepth:        16,
		Codec:           "mp3",
		FrameDuration:   "20ms",
		Timeout:         30,
		Stability:       0.5,
		SimilarityBoost: 0.75,
		Style:           0.0,
		UseSpeakerBoost: true,
	}

	// 从环境变量获取默认值
	if opt.APIKey == "" {
		opt.APIKey = utils.GetEnv("ELEVENLABS_API_KEY")
	}
	if opt.VoiceID == "" {
		opt.VoiceID = "21m00Tcm4TlvDq8ikWAM" // 默认 Rachel 音色
	}

	return opt
}

// NewElevenLabsService 创建 ElevenLabs TTS 服务
func NewElevenLabsService(opt ElevenLabsConfig) *ElevenLabsService {
	return &ElevenLabsService{
		opt:    opt,
		client: &http.Client{},
	}
}

func (es *ElevenLabsService) Provider() TTSProvider {
	return ProviderElevenLabs
}

func (es *ElevenLabsService) Format() media.StreamFormat {
	es.mu.Lock()
	defer es.mu.Unlock()
	return media.StreamFormat{
		SampleRate:    es.opt.SampleRate,
		BitDepth:      es.opt.BitDepth,
		Channels:      es.opt.Channels,
		FrameDuration: media.NormalizeFramePeriod(es.opt.FrameDuration),
	}
}

func (es *ElevenLabsService) CacheKey(text string) string {
	es.mu.Lock()
	defer es.mu.Unlock()
	digest := media.MediaCache().BuildKey(text)
	return fmt.Sprintf("elevenlabs.tts-%s-%s-%d-%s.%s", es.opt.VoiceID, es.opt.ModelID, es.opt.SampleRate, digest, es.opt.Codec)
}

func (es *ElevenLabsService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	es.mu.Lock()
	opt := es.opt
	es.mu.Unlock()

	// 验证配置
	if opt.APIKey == "" {
		return fmt.Errorf("ELEVENLABS_API_KEY is required")
	}

	// 构建请求
	req := ElevenLabsRequest{
		Text:    text,
		ModelID: opt.ModelID,
		VoiceSettings: &ElevenLabsVoiceSettings{
			Stability:       opt.Stability,
			SimilarityBoost: opt.SimilarityBoost,
			Style:           opt.Style,
			UseSpeakerBoost: opt.UseSpeakerBoost,
		},
	}
	// 如果配置了语言代码，添加到请求中
	if opt.LanguageCode != "" {
		req.LanguageCode = opt.LanguageCode
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	url := fmt.Sprintf(elevenlabsTTSURL, opt.VoiceID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("xi-api-key", opt.APIKey)

	// 发送请求
	resp, err := es.client.Do(httpReq)
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
		"provider":   "elevenlabs",
		"text":       text,
		"audio_size": len(audioData),
		"voice_id":   opt.VoiceID,
	}).Info("elevenlabs tts: synthesis completed")

	return nil
}

func (es *ElevenLabsService) Close() error {
	if es.client != nil {
		es.client.CloseIdleConnections()
	}
	return nil
}
