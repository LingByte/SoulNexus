// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 语音 Relay 通用数据结构与渠道选择/参数合并辅助。
// 设计参考 LingVoice，但鉴权基于 LLMToken（type=asr/tts），渠道使用 SoulNexus 自身的 ASR/TTSChannel。
package models

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// MaxRelayAudioFetchBytes 限制 relay ASR 入站音频体积（multipart / URL 拉取 / base64 解码后）。
const MaxRelayAudioFetchBytes = 32 << 20 // 32 MiB

// SpeechASRTranscribeReq POST /v1/speech/asr/transcribe 的 JSON / form 入参。
type SpeechASRTranscribeReq struct {
	Group       string `json:"group"`
	Provider    string `json:"provider"`
	AudioBase64 string `json:"audio_base64"`
	AudioURL    string `json:"audio_url"`
	Format      string `json:"format"`
	Language    string `json:"language"`
	Extra       any    `json:"extra"`
}

// SpeechTTSSynthesizeReq POST /v1/speech/tts/synthesize 的 JSON 入参。
type SpeechTTSSynthesizeReq struct {
	Group          string                 `json:"group"`
	Text           string                 `json:"text" binding:"required"`
	Voice          string                 `json:"voice"`
	Extra          any                    `json:"extra"`
	ResponseType   string                 `json:"response_type"`
	Output         string                 `json:"output"`
	UploadBucket   string                 `json:"upload_bucket"`
	UploadKey      string                 `json:"upload_key"`
	UploadFilename string                 `json:"upload_filename"`
	AudioFormat    string                 `json:"audio_format"`
	SampleRate     int                    `json:"sample_rate"`
	TTSOptions     map[string]interface{} `json:"tts_options"`
}

// MergeASRTranscribeOptions 把请求 extra/format 合并到 provider merged 配置中（基础 map 由调用方提供）。
// 与 LingVoice 行为一致：不允许覆盖 provider；删除 model（避免 SDK 把字符串当成默认 model）。
func MergeASRTranscribeOptions(merged map[string]interface{}, body *SpeechASRTranscribeReq) {
	if body == nil || merged == nil {
		return
	}
	if body.Extra != nil {
		if m, ok := body.Extra.(map[string]interface{}); ok {
			for k, v := range m {
				k = strings.TrimSpace(k)
				if k == "" || strings.EqualFold(k, "provider") {
					continue
				}
				merged[k] = v
			}
		}
	}
	delete(merged, "model")
	f := strings.TrimSpace(body.Format)
	if f != "" {
		if _, ok := merged["format"]; !ok {
			if _, ok2 := merged["voiceFormat"]; !ok2 {
				merged["format"] = f
			}
		}
	}
}

// NormalizeRelayTTSResponseType 由 response_type / output 别名归一为 audio_base64 | url。
func NormalizeRelayTTSResponseType(responseType, output string) string {
	s := strings.TrimSpace(responseType)
	if s == "" {
		s = strings.TrimSpace(output)
	}
	switch strings.ToLower(s) {
	case "url", "audio_url":
		return "url"
	case "audio_base64", "base64", "audio_data", "data", "":
		return "audio_base64"
	default:
		return "audio_base64"
	}
}

// PickASRChannelForToken 按 token.group 选择 enabled ASR 渠道；group 覆盖优先生效。
func PickASRChannelForToken(db *gorm.DB, tok *LLMToken, groupOverride string) (*ASRChannel, error) {
	if db == nil {
		return nil, errors.New("db nil")
	}
	g := strings.TrimSpace(groupOverride)
	if g == "" && tok != nil {
		g = strings.TrimSpace(tok.Group)
	}
	if g == "" {
		g = "default"
	}
	q := db.Model(&ASRChannel{}).Where("enabled = ?", true).Where("`group` IN (?, '')", g)
	q = ScopeOrgVisible(q, TokenOrgID(tok))
	var ch ASRChannel
	if err := q.Order("sort_order DESC, id ASC").First(&ch).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

// PickTTSChannelForToken 按 token.group 选择 enabled TTS 渠道；group 覆盖优先生效。
func PickTTSChannelForToken(db *gorm.DB, tok *LLMToken, groupOverride string) (*TTSChannel, error) {
	if db == nil {
		return nil, errors.New("db nil")
	}
	g := strings.TrimSpace(groupOverride)
	if g == "" && tok != nil {
		g = strings.TrimSpace(tok.Group)
	}
	if g == "" {
		g = "default"
	}
	q := db.Model(&TTSChannel{}).Where("enabled = ?", true).Where("`group` IN (?, '')", g)
	q = ScopeOrgVisible(q, TokenOrgID(tok))
	var ch TTSChannel
	if err := q.Order("sort_order DESC, id ASC").First(&ch).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

// RelayNoSpeechChannelDetail 把"未找到渠道"错误格式化成对外可读消息。
func RelayNoSpeechChannelDetail(err error, kind, group string) string {
	if err == nil {
		return ""
	}
	g := strings.TrimSpace(group)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if g != "" {
			return fmt.Sprintf("未找到可用 %s 渠道（group=%s）", kind, g)
		}
		return fmt.Sprintf("未找到可用 %s 渠道", kind)
	}
	if g != "" {
		return fmt.Sprintf("选择 %s 渠道失败（group=%s）：%s", kind, g, err.Error())
	}
	return fmt.Sprintf("选择 %s 渠道失败：%s", kind, err.Error())
}

// ApplyTTSVoiceToMergedMap 把 HTTP voice 字段映射到 provider-specific 的 map key。
func ApplyTTSVoiceToMergedMap(provider, voice string, merged map[string]interface{}) {
	if merged == nil || strings.TrimSpace(voice) == "" {
		return
	}
	p := strings.ToLower(strings.TrimSpace(provider))
	switch p {
	case "azure":
		merged["voice"] = voice
	case "qcloud", "tencent":
		merged["voiceType"] = voice
	case "minimax":
		merged["voiceId"] = voice
	case "elevenlabs":
		merged["voiceId"] = voice
	default:
		merged["voice"] = voice
	}
}
