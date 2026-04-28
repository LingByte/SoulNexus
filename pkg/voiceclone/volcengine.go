package voiceclone

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/lingstorage-sdk-go"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	optSubmit              = "submit"
	VolcengineCloneCluster = "volcano_icl"
)

var defaultHeader = []byte{0x11, 0x10, 0x11, 0x00}

// VolcengineCloneConfig 火山引擎克隆配置
type VolcengineCloneConfig struct {
	AppID         string  `json:"app_id"`
	Token         string  `json:"token"`          // WebSocket认证token（必需）
	Cluster       string  `json:"cluster"`        // 集群名称，默认 "volcano_icl"
	VoiceType     string  `json:"voice_type"`     // 音色类型（训练好的音色ID）
	Encoding      string  `json:"encoding"`       // 编码格式，默认 "pcm"
	SampleRate    int     `json:"sample_rate"`    // 采样率，默认 8000
	BitDepth      int     `json:"bit_depth"`      // 位深度，默认 16
	Channels      int     `json:"channels"`       // 声道数，默认 1
	FrameDuration string  `json:"frame_duration"` // 帧时长，默认 "20ms"
	SpeedRatio    float64 `json:"speed_ratio"`    // 语速比例，默认 1.0
	TrainingTimes int     `json:"training_times"` // 训练次数，默认 1
}

// VolcengineResponse 火山引擎WebSocket响应结构
type VolcengineResponse struct {
	ProtocolVersion          int
	HeaderSize               int
	MessageType              int
	MessageTypeSpecificFlags int
	SerializationMethod      int
	MessageCompression       int
	Reserved                 int
	SequenceNumber           int
	PayloadSize              int
	Audio                    []byte
	IsLast                   bool
	ErrorCode                int
	ErrorMessage             string
	Timestamp                *VolcengineSentenceTimestamp
}

// VolcengineSentenceTimestamp 火山引擎句子时间戳（内部使用）
type VolcengineSentenceTimestamp struct {
	Words []Word `json:"words"`
}

// Word 单词时间戳
type Word struct {
	Confidence float64 `json:"confidence"`
	EndTime    int     `json:"end_time"`
	StartTime  int     `json:"start_time"`
	Word       string  `json:"word"`
}

// VolcAddition 火山引擎附加信息
type VolcAddition struct {
	Frontend string `json:"frontend"`
}

// VolcengineCloneService 火山引擎语音克隆服务
type VolcengineCloneService struct {
	config     *VolcengineCloneConfig
	httpClient *http.Client
}

// NewVolcengineCloneService 创建火山引擎克隆服务
func NewVolcengineCloneService(config VolcengineCloneConfig) *VolcengineCloneService {
	if config.Cluster == "" {
		config.Cluster = VolcengineCloneCluster
	}
	if config.Encoding == "" {
		config.Encoding = "pcm"
	}
	if config.SampleRate == 0 {
		config.SampleRate = 8000
	}
	if config.BitDepth == 0 {
		config.BitDepth = 16
	}
	if config.Channels == 0 {
		config.Channels = 1
	}
	if config.FrameDuration == "" {
		config.FrameDuration = "20ms"
	}
	if config.SpeedRatio == 0 {
		config.SpeedRatio = 1.0
	}
	if config.TrainingTimes == 0 {
		config.TrainingTimes = 1
	}

	return &VolcengineCloneService{
		config: &config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Provider 返回服务提供商
func (s *VolcengineCloneService) Provider() Provider {
	return ProviderVolcengine
}

// GetTrainingTexts 获取训练文本（火山引擎暂不支持，返回错误）
func (s *VolcengineCloneService) GetTrainingTexts(ctx context.Context, textID int64) (*TrainingText, error) {
	return nil, fmt.Errorf("volcengine does not support GetTrainingTexts API")
}

// CreateTask 创建训练任务
// 注意：火山引擎的训练需要先在控制台创建 speaker_id，然后通过 SubmitAudio 上传音频
// 这里返回一个占位任务ID，实际训练通过 SubmitAudio 完成
func (s *VolcengineCloneService) CreateTask(ctx context.Context, req *CreateTaskRequest) (*CreateTaskResponse, error) {
	// 火山引擎的训练流程：
	// 1. 在控制台创建 speaker_id（或使用已有的）
	// 2. 通过 SubmitAudio 上传音频进行训练
	// 3. 通过 QueryTaskStatus 查询训练状态
	//
	// 为了兼容接口，这里返回一个基于 speaker_id 的任务ID
	// 实际使用时，speaker_id 应该从控制台获取或作为参数传入
	return nil, fmt.Errorf("volcengine training requires speaker_id from console, please use SubmitAudio directly with speaker_id")
}

// SubmitAudio 提交音频文件进行训练
// speaker_id 需要从控制台获取，或通过 TaskID 参数传入（格式：speaker_id:xxx）
func (s *VolcengineCloneService) SubmitAudio(ctx context.Context, req *SubmitAudioRequest) error {
	if s.config.Token == "" {
		return fmt.Errorf("token is required for training")
	}

	// 从 TaskID 中提取 speaker_id，格式：speaker_id:xxx 或直接是 speaker_id
	speakerID := req.TaskID
	if strings.HasPrefix(speakerID, "speaker_id:") {
		speakerID = strings.TrimPrefix(speakerID, "speaker_id:")
	}

	url := "https://openspeech.bytedance.com/api/v1/mega_tts/audio/upload"

	// 读取音频文件并编码为 base64
	audioData, err := io.ReadAll(req.AudioFile)
	if err != nil {
		return fmt.Errorf("failed to read audio file: %w", err)
	}
	audioBase64 := base64.StdEncoding.EncodeToString(audioData)

	// 检测音频格式
	// 火山引擎支持的格式：wav、mp3、ogg、m4a、aac、pcm
	// 注意：pcm 格式仅支持 24kHz 单通道
	audioFormat := "wav" // 默认格式

	// 尝试从 TaskID 或其他参数推断格式
	// 如果 TaskID 包含格式信息（如 "speaker_id:xxx:wav"），可以提取
	if strings.Contains(req.TaskID, ":") {
		parts := strings.Split(req.TaskID, ":")
		if len(parts) > 2 {
			// 格式：speaker_id:xxx:format
			format := strings.ToLower(parts[2])
			// 验证格式是否支持
			supportedFormats := map[string]bool{
				"wav": true, "mp3": true, "ogg": true,
				"m4a": true, "aac": true, "pcm": true,
			}
			if supportedFormats[format] {
				audioFormat = format
			}
		}
	}

	// 注意：如果无法从参数推断格式，默认使用 wav
	// 实际使用时，建议在调用 SubmitAudio 前根据文件扩展名判断格式
	// 并通过 TaskID 参数传入格式信息，或修改 SubmitAudioRequest 添加格式字段

	// 构建请求体
	requestBody := map[string]interface{}{
		"appid":      s.config.AppID,
		"speaker_id": speakerID,
		"audios": []map[string]interface{}{
			{
				"audio_bytes":  audioBase64,
				"audio_format": audioFormat,
			},
		},
		"source":     2, // 固定值
		"language":   0, // 0=中文，1=英文，2=日语等
		"model_type": 1, // 1=声音复刻ICL1.0效果
	}

	// 额外参数（可选）
	extraParams := map[string]interface{}{}
	extraParamsJSON, _ := json.Marshal(extraParams)
	requestBody["extra_params"] = string(extraParamsJSON)

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer;%s", s.config.Token))
	httpReq.Header.Set("Resource-Id", "volc.megatts.voiceclone")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("training failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		BaseResp struct {
			StatusCode    int    `json:"StatusCode"`
			StatusMessage string `json:"StatusMessage"`
		} `json:"BaseResp"`
		SpeakerID string `json:"speaker_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.BaseResp.StatusCode != 0 {
		// 特殊处理：已达上传次数限制（错误码 1123）
		if apiResp.BaseResp.StatusCode == 1123 {
			return fmt.Errorf("training failed: 已达上传次数限制（同一音色最多支持10次上传），错误信息: %s", apiResp.BaseResp.StatusMessage)
		}
		return fmt.Errorf("training failed: %s", apiResp.BaseResp.StatusMessage)
	}

	return nil
}

// QueryTaskStatus 查询任务状态
// taskID 应该是 speaker_id
func (s *VolcengineCloneService) QueryTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	if s.config.Token == "" {
		return nil, fmt.Errorf("token is required for querying status")
	}

	// 从 TaskID 中提取 speaker_id
	speakerID := taskID
	if strings.HasPrefix(speakerID, "speaker_id:") {
		speakerID = strings.TrimPrefix(speakerID, "speaker_id:")
	}

	url := "https://openspeech.bytedance.com/api/v1/mega_tts/status"

	requestBody := map[string]interface{}{
		"appid":      s.config.AppID,
		"speaker_id": speakerID,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer;%s", s.config.Token))
	httpReq.Header.Set("Resource-Id", "volc.megatts.voiceclone")

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query status failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		BaseResp struct {
			StatusCode    int    `json:"StatusCode"`
			StatusMessage string `json:"StatusMessage"`
		} `json:"BaseResp"`
		SpeakerID  string `json:"speaker_id"`
		Status     int    `json:"status"` // 0=NotFound, 1=Training, 2=Success, 3=Failed, 4=Active
		CreateTime int64  `json:"create_time"`
		Version    string `json:"version"`
		DemoAudio  string `json:"demo_audio"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if apiResp.BaseResp.StatusCode != 0 {
		return nil, fmt.Errorf("query status failed: %s", apiResp.BaseResp.StatusMessage)
	}

	// 转换状态
	var trainingStatus TrainingStatus
	switch apiResp.Status {
	case 0: // NotFound
		trainingStatus = TrainingStatusFailed
	case 1: // Training
		trainingStatus = TrainingStatusInProgress
	case 2, 4: // Success, Active (都可以使用)
		trainingStatus = TrainingStatusSuccess
	case 3: // Failed
		trainingStatus = TrainingStatusFailed
	default:
		trainingStatus = TrainingStatusInProgress
	}

	return &TaskStatus{
		TaskID:     speakerID,
		TaskName:   speakerID,
		Status:     trainingStatus,
		AssetID:    speakerID, // 火山引擎使用 speaker_id 作为 asset_id
		TrainVID:   apiResp.Version,
		FailedDesc: apiResp.BaseResp.StatusMessage,
		Progress:   0, // 火山引擎不返回进度
		CreatedAt:  time.Unix(apiResp.CreateTime/1000, 0),
		UpdatedAt:  time.Now(),
	}, nil
}

// parseResponse 解析火山引擎WebSocket二进制响应
func parseVolcengineResponse(message []byte) (*VolcengineResponse, error) {
	if len(message) < 4 {
		return nil, errors.New("message too short")
	}
	r := &VolcengineResponse{}
	r.ProtocolVersion = int((message[0] & 0xf0) >> 4)
	r.HeaderSize = int(message[0] & 0x0f)
	r.MessageType = int((message[1] & 0xf0) >> 4)
	r.MessageTypeSpecificFlags = int(message[1] & 0x0f)
	r.SerializationMethod = int((message[2] & 0xf0) >> 4)
	r.MessageCompression = int(message[2] & 0x0f)
	r.Reserved = int(message[3])

	headerExt := []byte{}
	if r.HeaderSize > 1 {
		if len(message) < r.HeaderSize*4 {
			return nil, errors.New("header size mismatch")
		}
		headerExt = message[4 : r.HeaderSize*4]
	}
	payload := message[r.HeaderSize*4:]

	if r.HeaderSize != 1 {
		logrus.WithFields(logrus.Fields{
			"header_extensions": headerExt,
		}).Debug("volcengine: header extensions")
	}

	switch r.MessageType {
	case 11: // audio
		if r.MessageTypeSpecificFlags == 0 {
			return r, nil
		}
		if len(payload) < 8 {
			return nil, errors.New("audio payload too short")
		}
		seq := int32(binary.BigEndian.Uint32(payload[:4]))
		size := int(binary.BigEndian.Uint32(payload[4:8]))
		r.SequenceNumber = int(seq)
		r.PayloadSize = size
		payload = payload[8:]
		if size > 0 {
			if len(payload) < size {
				return nil, errors.New("audio payload size mismatch")
			}
			r.Audio = payload[:size]
		}
		if seq < 0 {
			r.IsLast = true
		}
		return r, nil

	case 15: // error
		if len(payload) < 8 {
			return nil, errors.New("error payload too short")
		}
		code := int(binary.BigEndian.Uint32(payload[:4]))
		msgSize := int(binary.BigEndian.Uint32(payload[4:8]))
		msg := payload[8:]
		if len(msg) < msgSize {
			return nil, errors.New("error message size mismatch")
		}
		if r.MessageCompression == 1 {
			gr, err := gzip.NewReader(bytes.NewReader(msg[:msgSize]))
			if err != nil {
				return nil, err
			}
			defer func(gr *gzip.Reader) {
				if err := gr.Close(); err != nil {
					logrus.Warnf("error closing gzip reader: %v", err)
				}
			}(gr)
			unzipped, err := io.ReadAll(gr)
			if err != nil {
				return nil, err
			}
			msg = unzipped
		} else {
			msg = msg[:msgSize]
		}
		r.ErrorCode = code
		r.ErrorMessage = string(msg)
		return r, nil

	case 12: // 0xc, Frontend message (时间戳)
		if len(payload) < 4 {
			return nil, errors.New("frontend message payload too short")
		}
		msgSize := int(binary.BigEndian.Uint32(payload[:4]))
		msg := payload[4:]
		if len(msg) < msgSize {
			return nil, errors.New("frontend message size mismatch")
		}
		if r.MessageCompression == 1 {
			gr, err := gzip.NewReader(bytes.NewReader(msg[:msgSize]))
			if err != nil {
				return nil, err
			}
			defer func(gr *gzip.Reader) {
				if err := gr.Close(); err != nil {
					logrus.Warnf("error closing gzip reader: %v", err)
				}
			}(gr)
			unzipped, err := io.ReadAll(gr)
			if err != nil {
				return nil, err
			}
			msg = unzipped
		} else {
			msg = msg[:msgSize]
		}

		var frontendMsg VolcAddition
		if err := json.Unmarshal(msg, &frontendMsg); err == nil && frontendMsg.Frontend != "" {
			var sentenceTimestamp VolcengineSentenceTimestamp
			if err := json.Unmarshal([]byte(frontendMsg.Frontend), &sentenceTimestamp); err == nil {
				r.Timestamp = &sentenceTimestamp
				return r, nil
			}
		}
		return r, nil

	default:
		logrus.WithFields(logrus.Fields{
			"message_type": r.MessageType,
		}).Warn("volcengine: undefined message type")
		return nil, fmt.Errorf("unknown message type: %d", r.MessageType)
	}
}

// Synthesize 使用训练好的音色合成语音
func (s *VolcengineCloneService) Synthesize(ctx context.Context, req *SynthesizeRequest) (*SynthesizeResponse, error) {
	return s.synthesizeWithWebSocket(ctx, req)
}

// SynthesizeStream 流式合成语音
func (s *VolcengineCloneService) SynthesizeStream(ctx context.Context, req *SynthesizeRequest, handler SynthesisHandler) error {
	return s.synthesizeStreamWithWebSocket(ctx, req, handler)
}

// synthesizeWithWebSocket 使用WebSocket合成语音
func (s *VolcengineCloneService) synthesizeWithWebSocket(ctx context.Context, req *SynthesizeRequest) (*SynthesizeResponse, error) {
	if s.config.Token == "" {
		return nil, fmt.Errorf("token is required for WebSocket API")
	}
	if s.config.AppID == "" {
		return nil, fmt.Errorf("app_id is required for WebSocket API")
	}
	if req.AssetID == "" {
		return nil, fmt.Errorf("asset_id (voice_type) is required for synthesis")
	}

	// 构建请求参数
	input := s.buildWebSocketRequestParams(req.Text, req.AssetID)

	// 添加调试日志
	logrus.WithFields(logrus.Fields{
		"app_id":     s.config.AppID,
		"voice_type": req.AssetID,
		"cluster":    s.config.Cluster,
	}).Debug("volcengine: synthesizing with WebSocket")

	// 压缩请求
	compressed := gzipCompress(input)

	// 构建二进制消息
	payloadSize := len(compressed)
	payloadArr := make([]byte, 4)
	binary.BigEndian.PutUint32(payloadArr, uint32(payloadSize))

	clientRequest := make([]byte, len(defaultHeader))
	copy(clientRequest, defaultHeader)
	clientRequest = append(clientRequest, payloadArr...)
	clientRequest = append(clientRequest, compressed...)

	// 连接到WebSocket
	wsURL := "wss://openspeech.bytedance.com/api/v1/tts/ws_binary"
	headers := http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer;%s", s.config.Token)},
		"Resource-Id":   []string{"volc.megatts.voiceclone"}, // 声音复刻资源ID
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()

	// 发送请求
	if err := conn.WriteMessage(websocket.BinaryMessage, clientRequest); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// 接收响应
	var allAudioData []byte
	start := time.Now()
	var ttfb int64

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return nil, fmt.Errorf("WebSocket error: %w", err)
			}
			break
		}

		resp, err := parseVolcengineResponse(message)
		if err != nil {
			logrus.WithError(err).Error("failed to parse response")
			continue
		}

		// 处理错误消息
		if resp.MessageType == 15 {
			return nil, fmt.Errorf("volcengine error: code=%d, msg=%s", resp.ErrorCode, resp.ErrorMessage)
		}

		// 记录TTFB
		if len(resp.Audio) > 0 && ttfb == 0 {
			ttfb = time.Since(start).Milliseconds()
			logrus.WithFields(logrus.Fields{
				"duration": ttfb,
			}).Info("volcengine_clone: ttfb done")
		}

		// 收集音频数据
		if len(resp.Audio) > 0 {
			allAudioData = append(allAudioData, resp.Audio...)
		}

		// 检查是否完成
		if resp.IsLast {
			break
		}
	}

	if len(allAudioData) == 0 {
		return nil, fmt.Errorf("no audio data received")
	}

	return &SynthesizeResponse{
		AudioData:  allAudioData,
		Format:     "pcm",
		SampleRate: s.config.SampleRate,
	}, nil
}

// synthesizeStreamWithWebSocket 使用WebSocket流式合成语音
func (s *VolcengineCloneService) synthesizeStreamWithWebSocket(ctx context.Context, req *SynthesizeRequest, handler SynthesisHandler) error {
	if s.config.Token == "" {
		return fmt.Errorf("token is required for WebSocket API")
	}
	if s.config.AppID == "" {
		return fmt.Errorf("app_id is required for WebSocket API")
	}
	if req.AssetID == "" {
		return fmt.Errorf("asset_id (voice_type) is required for synthesis")
	}

	// 构建请求参数
	input := s.buildWebSocketRequestParams(req.Text, req.AssetID)

	// 添加调试日志
	logrus.WithFields(logrus.Fields{
		"app_id":     s.config.AppID,
		"voice_type": req.AssetID,
		"cluster":    s.config.Cluster,
	}).Debug("volcengine: synthesizing with WebSocket (stream mode)")

	// 压缩请求
	compressed := gzipCompress(input)

	// 构建二进制消息
	payloadSize := len(compressed)
	payloadArr := make([]byte, 4)
	binary.BigEndian.PutUint32(payloadArr, uint32(payloadSize))

	clientRequest := make([]byte, len(defaultHeader))
	copy(clientRequest, defaultHeader)
	clientRequest = append(clientRequest, payloadArr...)
	clientRequest = append(clientRequest, compressed...)

	// 连接到WebSocket
	wsURL := "wss://openspeech.bytedance.com/api/v1/tts/ws_binary"
	headers := http.Header{
		"Authorization": []string{fmt.Sprintf("Bearer;%s", s.config.Token)},
		"Resource-Id":   []string{"volc.megatts.voiceclone"}, // 声音复刻资源ID
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}
	defer conn.Close()

	// 发送请求
	if err := conn.WriteMessage(websocket.BinaryMessage, clientRequest); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// 流式接收响应
	start := time.Now()
	var ttfb int64
	firstAudio := true

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return fmt.Errorf("WebSocket error: %w", err)
			}
			break
		}

		resp, err := parseVolcengineResponse(message)
		if err != nil {
			logrus.WithError(err).Error("failed to parse response")
			continue
		}

		// 处理错误消息
		if resp.MessageType == 15 {
			return fmt.Errorf("volcengine error: code=%d, msg=%s", resp.ErrorCode, resp.ErrorMessage)
		}

		// 记录TTFB并发送首帧
		if len(resp.Audio) > 0 {
			if firstAudio {
				firstAudio = false
				ttfb = time.Since(start).Milliseconds()
				logrus.WithFields(logrus.Fields{
					"duration": ttfb,
				}).Info("volcengine_clone: ttfb done (stream mode)")
			}

			// 流式回调音频数据
			if handler != nil {
				handler.OnMessage(resp.Audio)
			}

			// 如果有时间戳信息，也回调
			if resp.Timestamp != nil && handler != nil && len(resp.Timestamp.Words) > 0 {
				// 计算句子的开始和结束时间
				startTime := int64(resp.Timestamp.Words[0].StartTime)
				endTime := int64(resp.Timestamp.Words[len(resp.Timestamp.Words)-1].EndTime)
				handler.OnTimestamp(SentenceTimestamp{
					StartTime: startTime,
					EndTime:   endTime,
				})
			}
		}

		// 检查是否完成
		if resp.IsLast {
			break
		}
	}

	return nil
}

// buildWebSocketRequestParams 构建WebSocket请求参数
func (s *VolcengineCloneService) buildWebSocketRequestParams(text, voiceType string) []byte {
	reqID := uuid.NewString()
	params := make(map[string]map[string]interface{})

	params["app"] = make(map[string]interface{})
	params["app"]["appid"] = s.config.AppID
	params["app"]["token"] = s.config.Token
	params["app"]["cluster"] = s.config.Cluster

	params["user"] = make(map[string]interface{})
	params["user"]["uid"] = "uid"

	params["audio"] = make(map[string]interface{})
	params["audio"]["voice_type"] = voiceType // 使用assetID作为voice_type
	params["audio"]["encoding"] = s.config.Encoding
	params["audio"]["speed_ratio"] = s.config.SpeedRatio
	params["audio"]["rate"] = s.config.SampleRate
	params["audio"]["BitRate"] = s.config.BitDepth
	params["audio"]["volume_ratio"] = 1.0
	params["audio"]["pitch_ratio"] = 1.0

	params["request"] = make(map[string]interface{})
	params["request"]["reqid"] = reqID
	params["request"]["text"] = text
	if strings.HasPrefix(text, "<speak>") {
		params["request"]["text_type"] = "ssml"
	} else {
		params["request"]["text_type"] = "plain"
	}
	params["request"]["operation"] = optSubmit
	params["request"]["with_timestamp"] = 1

	resStr, _ := json.Marshal(params)
	return resStr
}

// SynthesizeToStorage 合成并保存到存储
func (s *VolcengineCloneService) SynthesizeToStorage(ctx context.Context, req *SynthesizeRequest, storageKey string) (string, error) {
	resp, err := s.Synthesize(ctx, req)
	if err != nil {
		return "", err
	}

	// 将 PCM 转换为 WAV 格式（浏览器可以播放）
	wavData, err := s.convertPCMToWAV(resp.AudioData, s.config.SampleRate, s.config.Channels, s.config.BitDepth)
	if err != nil {
		return "", fmt.Errorf("failed to convert PCM to WAV: %w", err)
	}

	// 如果存储路径是 .pcm，改为 .wav
	if strings.HasSuffix(storageKey, ".pcm") {
		storageKey = strings.TrimSuffix(storageKey, ".pcm") + ".wav"
	} else if !strings.HasSuffix(storageKey, ".wav") {
		storageKey = storageKey + ".wav"
	}

	// 测试上传
	result, err := config.GlobalStore.UploadBytes(&lingstorage.UploadBytesRequest{
		Data:     wavData,
		Filename: storageKey,
		Bucket:   config.GlobalConfig.Services.Storage.Bucket,
		Key:      storageKey,
	})

	// 获取URL
	return result.URL, nil
}

// convertPCMToWAV 将 PCM 音频数据转换为 WAV 格式（添加 WAV 文件头）
func (s *VolcengineCloneService) convertPCMToWAV(pcmData []byte, sampleRate int, channels int, bitDepth int) ([]byte, error) {
	// 创建44字节的WAV头部
	header := make([]byte, 44)
	dataSize := len(pcmData)

	// RIFF header
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+dataSize)) // File size
	copy(header[8:12], "WAVE")

	// fmt chunk
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)                                     // fmt chunk size
	binary.LittleEndian.PutUint16(header[20:22], 1)                                      // Audio format (PCM)
	binary.LittleEndian.PutUint16(header[22:24], uint16(channels))                       // Number of channels
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))                     // Sample rate
	binary.LittleEndian.PutUint32(header[28:32], uint32(sampleRate*channels*bitDepth/8)) // Byte rate
	binary.LittleEndian.PutUint16(header[32:34], uint16(channels*bitDepth/8))            // Block align
	binary.LittleEndian.PutUint16(header[34:36], uint16(bitDepth))                       // Bits per sample

	// data chunk
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(dataSize)) // Data size

	// 合并头部和音频数据
	wavData := append(header, pcmData...)
	return wavData, nil
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
