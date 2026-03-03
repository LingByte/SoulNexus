package sessions

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	// TTSEchoSuppressionWindow TTS回音抑制窗口（毫秒）
	ttsEchoSuppressionWindow = 5000
	// AudioEnergyThreshold 音频能量阈值（基础值）
	// 在 TTS 播放期间，只有能量超过这个值的 15 倍才能通过
	audioEnergyThreshold = 2000
	// TTSPlayingGracePeriod TTS播放状态的宽限期（毫秒）
	ttsPlayingGracePeriod = 1000
)

// AudioManager 音频管理器 - 回声消除
type AudioManager struct {
	mu              sync.RWMutex
	logger          *zap.Logger
	ttsOutputBuffer []TTSFrame // TTS输出音频缓冲区
	sampleRate      int
	channels        int
	echoSuppression bool
}

// TTSFrame TTS音频帧
type TTSFrame struct {
	Data      []byte
	Timestamp time.Time
	Energy    int64
}

// NewAudioManager 创建音频管理器
func NewAudioManager(sampleRate, channels int, logger *zap.Logger) *AudioManager {
	return &AudioManager{
		logger:          logger,
		ttsOutputBuffer: make([]TTSFrame, 0, 100),
		sampleRate:      sampleRate,
		channels:        channels,
		echoSuppression: true,
	}
}

// ProcessInputAudio 处理输入音频（激进的 AEC 回声消除）
// 返回 (处理后的音频数据, 是否应该处理)
func (m *AudioManager) ProcessInputAudio(data []byte, ttsPlaying bool) ([]byte, bool) {
	if len(data) == 0 {
		return nil, false
	}

	m.mu.RLock()
	echoSuppression := m.echoSuppression
	m.mu.RUnlock()

	// 不再有宽限期，TTS 结束后立即恢复 ASR
	// 如果TTS不在播放，直接处理
	if !ttsPlaying || !echoSuppression {
		return data, true
	}

	// ============ 超激进的 AEC 算法：在 TTS 播放期间严格过滤 ============
	// 策略：在 TTS 播放期间，直接过滤掉所有音频
	// 只有 VAD 明确检测到用户语音时才放通（由 VAD 组件负责）

	inputEnergy := m.calculateEnergy(data)

	// 在 TTS 播放期间，直接过滤所有音频
	// 不依赖能量检测，因为 AI 语音的能量可能很高
	// 让 VAD 组件做最终决定
	m.logger.Debug("[AudioManager] TTS 播放期间，过滤所有音频（由 VAD 做最终决定）",
		zap.Int64("energy", inputEnergy))
	return nil, false
}

// RecordTTSOutput 记录TTS输出音频
func (m *AudioManager) RecordTTSOutput(data []byte) {
	if len(data) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	energy := m.calculateEnergy(data)

	frame := TTSFrame{
		Data:      make([]byte, len(data)),
		Timestamp: time.Now(),
		Energy:    energy,
	}
	copy(frame.Data, data)

	m.ttsOutputBuffer = append(m.ttsOutputBuffer, frame)

	// 限制缓冲区大小
	if len(m.ttsOutputBuffer) > 100 {
		m.ttsOutputBuffer = m.ttsOutputBuffer[len(m.ttsOutputBuffer)-100:]
	}
}

// isTTSEcho 检查输入音频是否是TTS回音（保留但不使用）
func (m *AudioManager) isTTSEcho(inputData []byte, inputEnergy int64) bool {
	// 在激进模式下，这个方法不再使用
	// 所有的过滤都由能量检测和 VAD 完成
	return false
}

// isEchoBySpectrum 使用频谱分析检测回音（保留但不使用）
func (m *AudioManager) isEchoBySpectrum(inputData []byte) bool {
	// 在激进模式下，这个方法不再使用
	// 所有的过滤都由能量检测和 VAD 完成
	return false
}

// analyzeFrequencyBands 分析音频的低频和高频能量
// 简单实现：将音频分为两部分，计算每部分的能量
func (m *AudioManager) analyzeFrequencyBands(data []byte) (int64, int64) {
	if len(data) < 4 {
		return 0, 0
	}

	sampleCount := len(data) / 2
	midPoint := sampleCount / 2

	// 计算前半部分（低频）的能量
	var lowFreqEnergy int64
	for i := 0; i < midPoint; i++ {
		sample := int16(data[i*2]) | (int16(data[i*2+1]) << 8)
		lowFreqEnergy += int64(sample) * int64(sample)
	}

	// 计算后半部分（高频）的能量
	var highFreqEnergy int64
	for i := midPoint; i < sampleCount; i++ {
		sample := int16(data[i*2]) | (int16(data[i*2+1]) << 8)
		highFreqEnergy += int64(sample) * int64(sample)
	}

	return lowFreqEnergy, highFreqEnergy
}

// calculateEnergy 计算音频能量
func (m *AudioManager) calculateEnergy(data []byte) int64 {
	if len(data) < 2 {
		return 0
	}

	var sumSquares int64
	sampleCount := len(data) / 2

	for i := 0; i < sampleCount; i++ {
		sample := int16(data[i*2]) | (int16(data[i*2+1]) << 8)
		sumSquares += int64(sample) * int64(sample)
	}

	if sampleCount == 0 {
		return 0
	}

	return sumSquares / int64(sampleCount)
}

// calculateSimilarity 计算两个音频数据的相似度
func (m *AudioManager) calculateSimilarity(data1, data2 []byte) float64 {
	if len(data1) == 0 || len(data2) == 0 {
		return 0.0
	}

	minLen := len(data1)
	if len(data2) < minLen {
		minLen = len(data2)
	}

	if minLen < 2 {
		return 0.0
	}

	compareSamples := minLen / 2
	if compareSamples > 100 {
		compareSamples = 100
	}

	var diffSum int64
	for i := 0; i < compareSamples; i++ {
		sample1 := int16(data1[i*2]) | (int16(data1[i*2+1]) << 8)
		sample2 := int16(data2[i*2]) | (int16(data2[i*2+1]) << 8)
		diff := int64(sample1) - int64(sample2)
		if diff < 0 {
			diff = -diff
		}
		diffSum += diff
	}

	maxDiff := int64(65536) * int64(compareSamples)
	if maxDiff == 0 {
		return 0.0
	}

	similarity := 1.0 - float64(diffSum)/float64(maxDiff)
	if similarity < 0 {
		return 0.0
	}
	return similarity
}

// NotifyTTSEnd 通知TTS播放结束（已不再使用）
func (m *AudioManager) NotifyTTSEnd() {
	// 不再需要记录 TTS 结束时间，因为已移除缓冲时间
}

// Clear 清空状态
func (m *AudioManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ttsOutputBuffer = m.ttsOutputBuffer[:0]
}

func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
