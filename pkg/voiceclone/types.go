package voiceclone

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"io"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
)

// Provider 语音克隆服务提供商
type Provider string

const (
	ProviderXunfei     Provider = "xunfei"     // 讯飞星火
	ProviderVolcengine Provider = "volcengine" // 火山引擎
)

// TrainingStatus 训练状态
type TrainingStatus int

const (
	TrainingStatusQueued     TrainingStatus = 2  // 排队中
	TrainingStatusInProgress TrainingStatus = -1 // 训练中
	TrainingStatusSuccess    TrainingStatus = 1  // 成功
	TrainingStatusFailed     TrainingStatus = 0  // 失败
)

// TrainingText 训练文本
type TrainingText struct {
	TextID   int64         `json:"text_id"`
	TextName string        `json:"text_name"`
	Segments []TextSegment `json:"segments"`
}

// TextSegment 文本段落
type TextSegment struct {
	SegID   interface{} `json:"seg_id"` // 可能是字符串或数字
	SegText string      `json:"seg_text"`
}

// CreateTaskRequest 创建训练任务请求
type CreateTaskRequest struct {
	TaskName string `json:"task_name"` // 任务名称
	Sex      int    `json:"sex"`       // 性别 1:男 2:女
	AgeGroup int    `json:"age_group"` // 年龄段 1:儿童 2:青年 3:中年 4:中老年
	Language string `json:"language"`  // 语言代码，如 zh, en
}

// CreateTaskResponse 创建训练任务响应
type CreateTaskResponse struct {
	TaskID string `json:"task_id"` // 任务ID
}

// SubmitAudioRequest 提交音频请求
type SubmitAudioRequest struct {
	TaskID    string    `json:"task_id"`     // 任务ID
	TextID    int64     `json:"text_id"`     // 训练文本ID
	TextSegID int64     `json:"text_seg_id"` // 文本段落ID
	AudioFile io.Reader `json:"-"`           // 音频文件
	Language  string    `json:"language"`    // 语言代码
}

// TaskStatus 任务状态
type TaskStatus struct {
	TaskID     string         `json:"task_id"`     // 任务ID
	TaskName   string         `json:"task_name"`   // 任务名称
	Status     TrainingStatus `json:"status"`      // 训练状态
	AssetID    string         `json:"asset_id"`    // 音色ID（训练成功后返回）
	TrainVID   string         `json:"train_vid"`   // 音库ID
	FailedDesc string         `json:"failed_desc"` // 失败原因
	Progress   float64        `json:"progress"`    // 训练进度 0-100
	CreatedAt  time.Time      `json:"created_at"`  // 创建时间
	UpdatedAt  time.Time      `json:"updated_at"`  // 更新时间
}

// SynthesizeRequest 合成请求
type SynthesizeRequest struct {
	AssetID  string `json:"asset_id"` // 音色ID
	Text     string `json:"text"`     // 要合成的文本
	Language string `json:"language"` // 语言代码
}

// SynthesizeResponse 合成响应
type SynthesizeResponse struct {
	AudioData  []byte  `json:"audio_data"`  // 音频数据
	Format     string  `json:"format"`      // 音频格式，如 pcm, wav, mp3
	SampleRate int     `json:"sample_rate"` // 采样率
	Duration   float64 `json:"duration"`    // 音频时长（秒）
}

// SynthesisHandler 流式合成处理器接口（与 synthesis 包兼容）
type SynthesisHandler interface {
	OnMessage([]byte)
	OnTimestamp(timestamp SentenceTimestamp)
}

// SentenceTimestamp 句子时间戳（兼容 synthesis 包）
type SentenceTimestamp struct {
	StartTime int64 `json:"start_time"` // 开始时间（毫秒）
	EndTime   int64 `json:"end_time"`   // 结束时间（毫秒）
}

// VoiceCloneService 语音克隆服务接口
type VoiceCloneService interface {
	// Provider 返回服务提供商名称
	Provider() Provider

	// GetTrainingTexts 获取训练文本列表
	GetTrainingTexts(ctx context.Context, textID int64) (*TrainingText, error)

	// CreateTask 创建训练任务
	CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error)

	// SubmitAudio 提交音频文件
	SubmitAudio(ctx context.Context, req *SubmitAudioRequest) error

	// QueryTaskStatus 查询任务状态
	QueryTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error)

	// Synthesize 使用训练好的音色合成语音（批量模式，返回完整音频）
	Synthesize(ctx context.Context, req *SynthesizeRequest) (*SynthesizeResponse, error)

	// SynthesizeStream 使用训练好的音色流式合成语音（流式模式，通过 handler 回调）
	SynthesizeStream(ctx context.Context, req *SynthesizeRequest, handler SynthesisHandler) error

	// SynthesizeToStorage 合成并保存到存储
	SynthesizeToStorage(ctx context.Context, req *SynthesizeRequest, storageKey string) (string, error)
}

// VoiceCloneSynthesisService 实现 synthesizer.SynthesisService 接口的适配器
// 用于在硬件通话中使用克隆音色
type VoiceCloneSynthesisService struct {
	cloneService VoiceCloneService
	assetID      string // 音色ID
}

// NewVoiceCloneSynthesisService 创建适配器
func NewVoiceCloneSynthesisService(cloneService VoiceCloneService, assetID string) *VoiceCloneSynthesisService {
	return &VoiceCloneSynthesisService{
		cloneService: cloneService,
		assetID:      assetID,
	}
}

// Provider 返回提供商名称
func (v *VoiceCloneSynthesisService) Provider() synthesizer.TTSProvider {
	provider := v.cloneService.Provider()
	switch provider {
	case ProviderXunfei:
		return synthesizer.ProviderXunfei
	case ProviderVolcengine:
		return synthesizer.ProviderVolcengine
	default:
		return synthesizer.TTSProvider(provider)
	}
}

// Format 返回音频格式
func (v *VoiceCloneSynthesisService) Format() media.StreamFormat {
	// 硬件通话采样率统一为 16000Hz
	return media.StreamFormat{
		SampleRate:    16000,
		BitDepth:      16,
		Channels:      1,
		FrameDuration: 0,
	}
}

// CacheKey 返回缓存键
func (v *VoiceCloneSynthesisService) CacheKey(text string) string {
	return "voiceclone_" + v.assetID + "_" + text
}

// Synthesize 实现 synthesizer.SynthesisService 接口
func (v *VoiceCloneSynthesisService) Synthesize(ctx context.Context, handler synthesizer.SynthesisHandler, text string) error {
	// 创建适配器处理器，用于处理采样率转换
	adapterHandler := &synthesisHandlerAdapter{
		handler:          handler,
		sourceSampleRate: 24000, // 克隆音色返回 24kHz
		targetSampleRate: 16000, // 硬件通话需要 16kHz
	}

	// 调用克隆服务的流式合成
	req := &SynthesizeRequest{
		AssetID:  v.assetID,
		Text:     text,
		Language: "zh", // 默认中文
	}

	return v.cloneService.SynthesizeStream(ctx, req, adapterHandler)
}

// Close 关闭服务
func (v *VoiceCloneSynthesisService) Close() error {
	return nil
}

// synthesisHandlerAdapter 适配器，将 voiceclone.SynthesisHandler 转换为 synthesizer.SynthesisHandler
// 同时处理采样率转换（从 24kHz 到 16kHz）
type synthesisHandlerAdapter struct {
	handler          synthesizer.SynthesisHandler
	sourceSampleRate int
	targetSampleRate int
	resampler        media.SampleRateConverter
}

// OnMessage 处理音频消息，进行采样率转换
func (a *synthesisHandlerAdapter) OnMessage(data []byte) {
	if a.handler == nil || len(data) == 0 {
		return
	}

	// 如果采样率相同，直接传递
	if a.sourceSampleRate == a.targetSampleRate {
		a.handler.OnMessage(data)
		return
	}

	// 使用流式重采样器
	if a.resampler == nil {
		a.resampler = media.DefaultResampler(a.sourceSampleRate, a.targetSampleRate)
	}

	// 写入数据到重采样器
	_, err := a.resampler.Write(data)
	if err != nil {
		// 如果重采样失败，直接传递原始数据
		a.handler.OnMessage(data)
		return
	}

	// 获取重采样后的数据
	resampled := a.resampler.Samples()
	if len(resampled) > 0 {
		a.handler.OnMessage(resampled)
	}
}

// OnTimestamp 处理时间戳
func (a *synthesisHandlerAdapter) OnTimestamp(timestamp SentenceTimestamp) {
	if a.handler != nil {
		// 转换时间戳格式
		synthTimestamp := synthesizer.SentenceTimestamp{
			Words: []synthesizer.Word{},
		}
		a.handler.OnTimestamp(synthTimestamp)
	}
}

// Config 语音克隆配置
type Config struct {
	Provider Provider               `json:"provider"` // 服务提供商
	Options  map[string]interface{} `json:"options"`  // 提供商特定配置
}
