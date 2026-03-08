package tools

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/voiceprint"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// VoiceprintIdentifyTool 声纹识别工具
type VoiceprintIdentifyTool struct {
	db                *gorm.DB
	voiceprintService *voiceprint.Service
	logger            *zap.Logger
	audioBuffer       [][]byte
	audioBufferMutex  sync.Mutex
	assistantID       string
	identifiedSpeaker *IdentifiedSpeaker
	identifiedMutex   sync.RWMutex
	// 用于存储当前句子的音频（直接存储字节，不分割成帧）
	currentSentenceAudio []byte
	sentenceAudioMutex   sync.Mutex
	// 音频统计信息
	totalAudioBytes    int64
	minAudioBytesForID int // 最少需要多少字节才能进行识别（默认16000字节 = 0.5秒）
}

// IdentifiedSpeaker 已识别的说话人信息
type IdentifiedSpeaker struct {
	SpeakerID   string
	SpeakerName string
	Description string
	Confidence  float64
}

// NewVoiceprintIdentifyTool 创建声纹识别工具
func NewVoiceprintIdentifyTool(
	db *gorm.DB,
	voiceprintService *voiceprint.Service,
	assistantID string,
	logger *zap.Logger,
) *VoiceprintIdentifyTool {
	return &VoiceprintIdentifyTool{
		db:                   db,
		voiceprintService:    voiceprintService,
		logger:               logger,
		audioBuffer:          make([][]byte, 0),
		assistantID:          assistantID,
		minAudioBytesForID:   16000,                  // 最少0.5秒的音频（16000字节 = 0.5秒 @ 16kHz, 16-bit, mono）
		currentSentenceAudio: make([]byte, 0, 32000), // 预分配1秒音频的容量，避免频繁扩容
	}
}

// AddAudioFrame 添加音频帧到缓冲区
// 注意：这个方法会被频繁调用，每次可能只有很小的音频块
func (t *VoiceprintIdentifyTool) AddAudioFrame(data []byte) {
	if len(data) == 0 {
		return
	}

	t.sentenceAudioMutex.Lock()
	defer t.sentenceAudioMutex.Unlock()

	// 直接追加音频数据，不分割成帧
	// 因为PCM回调可能会传入不同大小的音频块
	t.currentSentenceAudio = append(t.currentSentenceAudio, data...)
}

// StartNewSentence 标记新句子开始（在 ASR 开始时调用）
func (t *VoiceprintIdentifyTool) StartNewSentence() {
	t.sentenceAudioMutex.Lock()
	defer t.sentenceAudioMutex.Unlock()
	// 重用底层数组，避免重新分配
	t.currentSentenceAudio = t.currentSentenceAudio[:0]
}

// GetSentenceAudio 获取当前句子的音频数据（在 ASR 完成时调用）
func (t *VoiceprintIdentifyTool) GetSentenceAudio() []byte {
	t.sentenceAudioMutex.Lock()
	defer t.sentenceAudioMutex.Unlock()

	if len(t.currentSentenceAudio) == 0 {
		return nil
	}

	// 直接返回累积的音频数据
	audioData := make([]byte, len(t.currentSentenceAudio))
	copy(audioData, t.currentSentenceAudio)
	return audioData
}

// GetIdentifiedSpeaker 获取已识别的说话人
func (t *VoiceprintIdentifyTool) GetIdentifiedSpeaker() *IdentifiedSpeaker {
	t.identifiedMutex.RLock()
	defer t.identifiedMutex.RUnlock()
	return t.identifiedSpeaker
}

// Identify 执行声纹识别
func (t *VoiceprintIdentifyTool) Identify(ctx context.Context) (*IdentifiedSpeaker, error) {
	// 获取当前句子的音频
	audioData := t.GetSentenceAudio()
	if len(audioData) == 0 {
		return nil, fmt.Errorf("no audio data to identify")
	}

	// 检查音频长度是否足够
	if len(audioData) < t.minAudioBytesForID {
		return nil, fmt.Errorf("audio too short: %d bytes (minimum %d bytes required)", len(audioData), t.minAudioBytesForID)
	}

	// 将 PCM 数据转换为 WAV 格式
	wavData := t.pcmToWAV(audioData, 16000, 1, 16)

	// 获取该助手下的所有声纹
	var voiceprints []models.Voiceprint
	if err := t.db.Where("assistant_id = ?", t.assistantID).Find(&voiceprints).Error; err != nil {
		return nil, fmt.Errorf("failed to get voiceprints: %w", err)
	}

	if len(voiceprints) == 0 {
		return nil, fmt.Errorf("no voiceprints found for assistant %s", t.assistantID)
	}

	// 构建候选 ID 列表
	candidateIDs := make([]string, len(voiceprints))
	voiceprintMap := make(map[string]*models.Voiceprint)
	for i, vp := range voiceprints {
		candidateIDs[i] = vp.SpeakerID
		voiceprintMap[vp.SpeakerID] = &vp
	}

	t.logger.Info("[VoiceprintIdentify] 开始声纹识别",
		zap.Int("candidates", len(candidateIDs)),
		zap.Int("wav_size", len(wavData)))

	// 调用声纹识别服务
	result, err := t.voiceprintService.IdentifyVoiceprint(ctx, &voiceprint.IdentifyRequest{
		CandidateIDs: candidateIDs,
		AssistantID:  t.assistantID,
		AudioData:    wavData,
	})
	if err != nil {
		return nil, fmt.Errorf("voiceprint identification failed: %w", err)
	}

	// 记录识别结果用于调试
	t.logger.Info("[VoiceprintIdentify] 识别结果",
		zap.Float64("score", result.Score),
		zap.Bool("is_match", result.IsMatch),
		zap.String("speaker_id", result.SpeakerID),
		zap.String("confidence", result.Confidence))

	// 检查是否匹配
	if !result.IsMatch {
		// 即使不匹配，如果分数足够高（> 0.3），也返回识别结果
		// 这样可以让 LLM 知道识别到了谁，即使置信度不够高
		if result.Score > 0.3 {
			t.logger.Warn("[VoiceprintIdentify] 识别分数低于阈值，但仍返回结果",
				zap.Float64("score", result.Score),
				zap.Float64("threshold", result.Threshold))
			// 继续处理，不返回错误
		} else {
			return nil, fmt.Errorf("no matching voiceprint found (score: %.2f, threshold: %.2f)", result.Score, result.Threshold)
		}
	}

	// 获取对应的声纹记录
	vp := voiceprintMap[result.SpeakerID]
	if vp == nil {
		return nil, fmt.Errorf("voiceprint record not found for speaker %s", result.SpeakerID)
	}

	// 构建识别结果
	identified := &IdentifiedSpeaker{
		SpeakerID:   result.SpeakerID,
		SpeakerName: vp.SpeakerName,
		Description: vp.Description,
		Confidence:  result.Score,
	}

	// 保存识别结果
	t.identifiedMutex.Lock()
	t.identifiedSpeaker = identified
	t.identifiedMutex.Unlock()

	t.logger.Info("[VoiceprintIdentify] 识别成功",
		zap.String("speaker_id", identified.SpeakerID),
		zap.String("speaker_name", identified.SpeakerName),
		zap.Float64("confidence", identified.Confidence))

	return identified, nil
}

// ClearIdentification 清除识别结果
func (t *VoiceprintIdentifyTool) ClearIdentification() {
	t.identifiedMutex.Lock()
	defer t.identifiedMutex.Unlock()
	t.identifiedSpeaker = nil
}

// GetGreetingMessage 获取问候消息
func (t *VoiceprintIdentifyTool) GetGreetingMessage() string {
	t.identifiedMutex.RLock()
	defer t.identifiedMutex.RUnlock()

	if t.identifiedSpeaker == nil {
		return ""
	}

	// 构建问候消息
	greeting := fmt.Sprintf("听您的声音，您是%s吗？", t.identifiedSpeaker.SpeakerName)
	return greeting
}

// GetUserDescription 获取用户描述（用于 LLM System Prompt）
func (t *VoiceprintIdentifyTool) GetUserDescription() string {
	t.identifiedMutex.RLock()
	defer t.identifiedMutex.RUnlock()

	if t.identifiedSpeaker == nil || t.identifiedSpeaker.Description == "" {
		return ""
	}

	return fmt.Sprintf("关于用户 %s 的信息：%s", t.identifiedSpeaker.SpeakerName, t.identifiedSpeaker.Description)
}

// pcmToWAV 将 PCM 原始音频数据转换为 WAV 格式
// sampleRate: 采样率（如 16000）
// channels: 声道数（1 为单声道）
// bitsPerSample: 位深度（16 为 16-bit）
func (t *VoiceprintIdentifyTool) pcmToWAV(pcmData []byte, sampleRate int, channels int, bitsPerSample int) []byte {
	if len(pcmData) == 0 {
		return nil
	}

	// 计算字节率和块对齐
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	dataSize := len(pcmData)

	// 计算文件大小（不包括 RIFF 头的 8 字节）
	fileSize := 36 + dataSize // fmt chunk (24 bytes) + data chunk header (8 bytes) + audio data

	// 创建 WAV 文件
	wav := make([]byte, 44+dataSize)

	// RIFF 头 (12 bytes)
	copy(wav[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(wav[4:8], uint32(fileSize))
	copy(wav[8:12], []byte("WAVE"))

	// fmt 子块 (24 bytes)
	copy(wav[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(wav[16:20], 16)                    // fmt 块大小（固定 16）
	binary.LittleEndian.PutUint16(wav[20:22], 1)                     // 音频格式（1 = PCM）
	binary.LittleEndian.PutUint16(wav[22:24], uint16(channels))      // 声道数
	binary.LittleEndian.PutUint32(wav[24:28], uint32(sampleRate))    // 采样率
	binary.LittleEndian.PutUint32(wav[28:32], uint32(byteRate))      // 字节率
	binary.LittleEndian.PutUint16(wav[32:34], uint16(blockAlign))    // 块对齐
	binary.LittleEndian.PutUint16(wav[34:36], uint16(bitsPerSample)) // 位深度

	// data 子块 (8 bytes header + audio data)
	copy(wav[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(wav[40:44], uint32(dataSize))

	// 复制 PCM 数据
	copy(wav[44:], pcmData)

	return wav
}

// RegisterVoiceprintIdentifyTool 注册声纹识别工具给 LLM
// 这样 LLM 可以主动调用声纹识别来回答"我是谁"这类问题
func RegisterVoiceprintIdentifyTool(llmService *LLMService, voiceprintTool *VoiceprintIdentifyTool) {
	if voiceprintTool == nil || llmService == nil {
		return
	}

	// 定义工具参数
	parameters := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"action": map[string]interface{}{
				"type":        "string",
				"description": "要执行的操作: 'identify' 进行声纹识别, 'get_current' 获取当前已识别的用户信息",
				"enum":        []string{"identify", "get_current"},
			},
		},
		"required": []string{"action"},
	}

	// 注册工具
	llmService.RegisterTool(
		"voiceprint_identify",
		"声纹识别工具。用于识别说话人身份或获取当前用户信息。当用户问'我是谁'、'你认识我吗'等问题时，应该使用此工具进行声纹识别。",
		parameters,
		func(args map[string]interface{}) (string, error) {
			action, ok := args["action"].(string)
			if !ok {
				return "", fmt.Errorf("action 参数必须是字符串")
			}

			switch action {
			case "identify":
				// 执行声纹识别
				identified, err := voiceprintTool.Identify(context.Background())
				if err != nil {
					return fmt.Sprintf("声纹识别失败: %v", err), nil
				}
				return fmt.Sprintf("识别成功！您是 %s，置信度: %.2f%%。%s",
					identified.SpeakerName,
					identified.Confidence*100,
					identified.Description), nil

			case "get_current":
				// 获取当前已识别的用户
				identified := voiceprintTool.GetIdentifiedSpeaker()
				if identified == nil {
					return "当前没有已识别的用户", nil
				}
				return fmt.Sprintf("当前用户是 %s，置信度: %.2f%%。%s",
					identified.SpeakerName,
					identified.Confidence*100,
					identified.Description), nil

			default:
				return "", fmt.Errorf("未知的操作: %s", action)
			}
		},
	)

	voiceprintTool.logger.Info("[VoiceprintTool] 已注册声纹识别工具给 LLM")
}
