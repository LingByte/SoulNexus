package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/carlmjohnson/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	SsmlSpeak = "<speak>"

	VolcengineCloneCluster = "volcano_icl"
	VolcengineLLMCluster   = "volcano_tts"
	optSubmit              = "submit"
	optQuery               = "query"
)

var defaultHeader = []byte{0x11, 0x10, 0x11, 0x00}

// VolcengineTTSServResponse 火山引擎TTS响应结构
type VolcengineTTSServResponse struct {
	ReqID     string       `json:"reqid"`
	Code      int          `json:"code"`
	Message   string       `json:"message"`
	Operation string       `json:"operation"`
	Sequence  int          `json:"sequence"`
	Data      string       `json:"data"`
	Addition  VolcAddition `json:"addition"`
}

// VolcAddition 火山引擎附加信息
type VolcAddition struct {
	Frontend string `json:"frontend"`
}

// VolcengineTTSOption 火山引擎标准TTS配置
// 支持的常用音色类型（VoiceType）：
// - BV700_streaming: 默认音色
// - BV700_V2_streaming: V2版本
// - BV213_streaming: 广西老表（男声）
// - BV025_streaming: 甜美台妹（女声）
// 更多音色类型请参考火山引擎官方文档
type VolcengineTTSOption struct {
	AppID         string  `json:"appID"`         // 应用ID
	AccessToken   string  `json:"accessToken"`   // 访问令牌
	Cluster       string  `json:"cluster"`       // 集群名称，如 volcano_tts
	VoiceType     string  `json:"voiceType"`     // 音色类型，如 BV700_streaming
	Language      string  `json:"language"`      // 语言代码，如 zh、en
	Rate          int     `json:"rate"`          // 采样率，默认 8000
	Encoding      string  `json:"encoding"`      // 编码格式，默认 pcm
	SpeedRatio    float32 `json:"speedRatio"`    // 语速比例，默认 1.0
	VolumeRatio   float32 `json:"volumeRatio"`   // 音量比例，默认 1.0
	PitchRatio    float32 `json:"pitchRatio"`    // 音调比例，默认 1.0
	Channels      int     `json:"channels"`      // 声道数，默认 1
	BitDepth      int     `json:"bitDepth"`      // 位深度，默认 16
	FrameDuration string  `json:"frameDuration"` // 帧时长，默认 20ms
	TextType      string  `json:"textType"`      // 文本类型，plain 或 ssml
	Ssml          bool    `json:"ssml"`          // 是否使用 SSML
}

// VolcengineService 火山引擎标准TTS服务
type VolcengineService struct {
	opt VolcengineTTSOption
	mu  sync.Mutex
}

// NewVolcengineTTSOption 创建火山引擎TTS配置
func NewVolcengineTTSOption(appID, accessToken, cluster string) VolcengineTTSOption {
	return VolcengineTTSOption{
		AppID:         appID,
		AccessToken:   accessToken,
		Cluster:       cluster,
		VoiceType:     "BV700_streaming",
		Language:      "",
		Rate:          16000, // 火山引擎标准采样率 16000Hz
		Encoding:      "pcm",
		SpeedRatio:    1.0,
		VolumeRatio:   1.0,
		PitchRatio:    1.0,
		Channels:      1,
		BitDepth:      16,
		FrameDuration: "20ms",
		TextType:      "plain",
		Ssml:          false,
	}
}

// NewVolcengineService 创建火山引擎TTS服务
func NewVolcengineService(opt VolcengineTTSOption) *VolcengineService {
	return &VolcengineService{
		opt: opt,
	}
}

func (v *VolcengineService) Provider() TTSProvider {
	return ProviderVolcengine
}

func (v *VolcengineService) Format() media.StreamFormat {
	v.mu.Lock()
	defer v.mu.Unlock()
	return media.StreamFormat{
		SampleRate:    v.opt.Rate,
		BitDepth:      v.opt.BitDepth,
		Channels:      v.opt.Channels,
		FrameDuration: utils.NormalizeFramePeriod(v.opt.FrameDuration),
	}
}

func (v *VolcengineService) CacheKey(text string) string {
	v.mu.Lock()
	defer v.mu.Unlock()
	digest := media.MediaCache().BuildKey(text)
	speedRatio := int(v.opt.SpeedRatio * 100)
	return fmt.Sprintf("volcengine.tts-%s-%s-%d-%d-%s.pcm", v.opt.VoiceType, v.opt.Encoding, v.opt.Rate, speedRatio, digest)
}

func (v *VolcengineService) Synthesize(ctx context.Context, handler SynthesisHandler, text string) error {
	v.mu.Lock()
	opt := v.opt
	v.mu.Unlock()

	ttsReq := &volcengineSpeechSynthesisListener{
		handler: handler,
	}
	if text == "" {
		handler.OnMessage(make([]byte, 0))
		return nil
	}
	dataBytes, timestamp, err := ttsReq.sendRequest(ctx, opt, text)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			logrus.WithFields(logrus.Fields{
				"text": text,
			}).Warn("volcengine tts: context canceled")
			return nil
		}
		return err
	}

	// 记录音频数据大小
	if len(dataBytes) == 0 {
		logrus.WithFields(logrus.Fields{
			"text": text,
		}).Warn("volcengine tts: received empty audio data")
	} else {
		logrus.WithFields(logrus.Fields{
			"text":      text,
			"audioSize": len(dataBytes),
		}).Info("volcengine tts: received audio data")
	}

	handler.OnMessage(dataBytes)
	handler.OnTimestamp(timestamp)
	return nil
}

func (v *VolcengineService) Close() error {
	return nil
}

type volcengineSpeechSynthesisListener struct {
	handler SynthesisHandler
}

func (v *volcengineSpeechSynthesisListener) sendRequest(ctx context.Context, opt VolcengineTTSOption, text string) ([]byte, SentenceTimestamp, error) {
	reqID := uuid.NewString()
	params := make(map[string]map[string]interface{})
	params["app"] = make(map[string]interface{})
	params["app"]["appid"] = opt.AppID
	params["app"]["token"] = "access_token" // 固定值，实际 token 在 Authorization header 中
	params["app"]["cluster"] = opt.Cluster

	params["user"] = make(map[string]interface{})
	params["user"]["uid"] = "uid"

	params["audio"] = make(map[string]interface{})
	params["audio"]["voice_type"] = opt.VoiceType
	params["audio"]["encoding"] = opt.Encoding
	params["audio"]["speed_ratio"] = opt.SpeedRatio
	params["audio"]["volume_ratio"] = opt.VolumeRatio
	params["audio"]["pitch_ratio"] = opt.PitchRatio

	// 添加采样率参数
	if opt.Rate > 0 {
		params["audio"]["rate"] = opt.Rate
	}

	params["request"] = make(map[string]interface{})
	params["request"]["reqid"] = reqID
	params["request"]["text"] = text
	if strings.HasPrefix(text, SsmlSpeak) {
		params["request"]["text_type"] = "ssml"
	} else {
		params["request"]["text_type"] = "plain"
	}
	params["request"]["operation"] = optQuery
	params["request"]["with_timestamp"] = "1"

	url := "https://openspeech.bytedance.com/api/v1/tts"

	// 打印请求参数用于调试
	paramsJSON, _ := json.Marshal(params)
	logrus.WithFields(logrus.Fields{
		"url":           url,
		"appID":         opt.AppID,
		"cluster":       opt.Cluster,
		"voiceType":     opt.VoiceType,
		"accessToken":   opt.AccessToken[:10] + "...", // 只显示前10个字符
		"requestParams": string(paramsJSON),
	}).Info("volcengine tts: sending request")

	var resp VolcengineTTSServResponse
	if err := requests.URL(url).BodyJSON(&params).
		Header("Content-Type", "application/json").
		Header("Authorization", fmt.Sprintf("Bearer;%s", opt.AccessToken)).
		ToJSON(&resp).Fetch(ctx); err != nil {
		if !strings.Contains(err.Error(), "context canceled") {
			logrus.WithFields(logrus.Fields{
				"params": params,
			}).WithError(err).Error("volcengine tts: send request failed")
		}
		return nil, SentenceTimestamp{}, err
	}

	dataBytes, err := base64.StdEncoding.DecodeString(resp.Data)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"params":      params,
			"respCode":    resp.Code,
			"respMessage": resp.Message,
		}).WithError(err).Error("volcengine tts: decode string failed")
		return nil, SentenceTimestamp{}, err
	}

	if resp.Code != 3000 {
		logrus.WithFields(logrus.Fields{
			"code":          resp.Code,
			"message":       resp.Message,
			"dataLength":    len(resp.Data),
			"decodedLength": len(dataBytes),
		}).Error("volcengine tts: api error")
		return nil, SentenceTimestamp{}, fmt.Errorf("volcengine tts error: code=%d, message=%s", resp.Code, resp.Message)
	}

	logrus.WithFields(logrus.Fields{
		"reqID":         reqID,
		"text":          text,
		"audioDataSize": len(dataBytes),
		"respCode":      resp.Code,
	}).Info("volcengine tts: synthesis success")

	var timestamp SentenceTimestamp
	if resp.Addition.Frontend != "" {
		err = json.Unmarshal([]byte(resp.Addition.Frontend), &timestamp)
		if err != nil {
			logrus.WithError(err).Error("volcengine tts: decoding timestamp failed")
		}
	}
	return dataBytes, timestamp, nil
}

// gzipCompress 压缩数据
func gzipCompress(input []byte) []byte {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, _ = w.Write(input)
	_ = w.Close()
	return b.Bytes()
}

// gzipDecompress 解压数据
func gzipDecompress(input []byte) ([]byte, error) {
	b := bytes.NewBuffer(input)
	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return out, nil
}
