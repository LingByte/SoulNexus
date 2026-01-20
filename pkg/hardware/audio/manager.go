package audio

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	// TTSEchoSuppressionWindow TTS回音抑制窗口（毫秒）
	ttsEchoSuppressionWindow = 2000
	// AudioEnergyThreshold 音频能量阈值（用于检测有效音频）
	audioEnergyThreshold = 1000
)

// Manager 音频管理器 - 解决TTS冲突问题
// 通过智能过滤TTS回音，实现真正的双向流
type Manager struct {
	mu              sync.RWMutex
	logger          *zap.Logger
	ttsOutputBuffer []TTSFrame // TTS输出音频缓冲区（用于回声消除）
	ttsOutputIndex  int        // 当前TTS输出索引
	maxTTSSamples   int        // 最大TTS样本数
	sampleRate      int        // 采样率
	channels        int        // 声道数
	echoSuppression bool       // 是否启用回音抑制
}

// TTSFrame TTS音频帧
type TTSFrame struct {
	Data      []byte
	Timestamp time.Time
	Energy    int64 // 音频能量（用于快速匹配）
}

// NewManager 创建音频管理器
func NewManager(sampleRate, channels int, logger *zap.Logger) *Manager {
	// 计算最大TTS样本数（基于回音抑制窗口）
	// 假设16-bit PCM，每样本2字节
	samplesPerMs := sampleRate * channels / 1000
	maxSamples := samplesPerMs * ttsEchoSuppressionWindow / 2 // 除以2因为16-bit

	return &Manager{
		logger:          logger,
		ttsOutputBuffer: make([]TTSFrame, 0, 100),
		sampleRate:      sampleRate,
		channels:        channels,
		maxTTSSamples:   maxSamples,
		echoSuppression: true, // 默认启用回音抑制
	}
}

// ProcessInputAudio 处理输入音频（智能过滤TTS回音）
// 返回 (处理后的音频数据, 是否应该处理)
func (m *Manager) ProcessInputAudio(data []byte, ttsPlaying bool) ([]byte, bool) {
	if len(data) == 0 {
		return nil, false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// 如果TTS不在播放，直接处理
	if !ttsPlaying || !m.echoSuppression {
		return data, true
	}

	// 计算输入音频的能量
	inputEnergy := m.calculateEnergy(data)

	// 如果能量太低，可能是静音或无效音频
	if inputEnergy < audioEnergyThreshold {
		m.logger.Debug("输入音频能量过低，忽略",
			zap.Int64("energy", inputEnergy),
		)
		return nil, false
	}

	// 检查是否是TTS回音
	if m.isTTSEcho(data, inputEnergy) {
		m.logger.Debug("检测到TTS回音，过滤",
			zap.Int64("energy", inputEnergy),
		)
		return nil, false
	}

	// 不是回音，可以处理
	return data, true
}

// RecordTTSOutput 记录TTS输出音频（用于回声消除）
func (m *Manager) RecordTTSOutput(data []byte) {
	if len(data) == 0 {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 计算能量
	energy := m.calculateEnergy(data)

	// 添加到缓冲区
	frame := TTSFrame{
		Data:      make([]byte, len(data)),
		Timestamp: time.Now(),
		Energy:    energy,
	}
	copy(frame.Data, data)

	m.ttsOutputBuffer = append(m.ttsOutputBuffer, frame)

	// 限制缓冲区大小（保留最近的回音抑制窗口内的数据）
	windowSize := m.sampleRate * m.channels * ttsEchoSuppressionWindow / 1000
	totalSamples := 0
	for i := len(m.ttsOutputBuffer) - 1; i >= 0; i-- {
		totalSamples += len(m.ttsOutputBuffer[i].Data) / 2 // 16-bit = 2 bytes per sample
		if totalSamples > windowSize {
			// 删除超出窗口的数据
			m.ttsOutputBuffer = m.ttsOutputBuffer[i+1:]
			break
		}
	}

	// 限制最大帧数
	if len(m.ttsOutputBuffer) > 100 {
		m.ttsOutputBuffer = m.ttsOutputBuffer[len(m.ttsOutputBuffer)-100:]
	}
}

// isTTSEcho 检查输入音频是否是TTS回音
func (m *Manager) isTTSEcho(inputData []byte, inputEnergy int64) bool {
	if len(m.ttsOutputBuffer) == 0 {
		return false
	}

	// 只检查最近几帧（减少计算量）
	checkFrames := 10
	if len(m.ttsOutputBuffer) < checkFrames {
		checkFrames = len(m.ttsOutputBuffer)
	}

	startIdx := len(m.ttsOutputBuffer) - checkFrames

	// 快速能量匹配
	for i := startIdx; i < len(m.ttsOutputBuffer); i++ {
		frame := m.ttsOutputBuffer[i]

		// 如果能量差异太大，不可能是回音
		energyDiff := abs64(inputEnergy - frame.Energy)
		if energyDiff > inputEnergy/2 {
			continue
		}

		// 检查时间窗口（回音通常在TTS输出后200-2000ms内）
		timeDiff := time.Since(frame.Timestamp)
		if timeDiff < 200*time.Millisecond || timeDiff > 2*time.Second {
			continue
		}

		// 如果数据长度相似，进行更详细的比较
		if abs(len(inputData)-len(frame.Data)) < len(frame.Data)/4 {
			// 计算相似度（简化版：比较前几个样本）
			similarity := m.calculateSimilarity(inputData, frame.Data)
			if similarity > 0.7 {
				return true
			}
		}
	}

	return false
}

// calculateEnergy 计算音频能量
func (m *Manager) calculateEnergy(data []byte) int64 {
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
func (m *Manager) calculateSimilarity(data1, data2 []byte) float64 {
	if len(data1) == 0 || len(data2) == 0 {
		return 0.0
	}

	// 取较小的长度进行比较
	minLen := len(data1)
	if len(data2) < minLen {
		minLen = len(data2)
	}

	if minLen < 2 {
		return 0.0
	}

	// 比较前N个样本（简化版）
	compareSamples := minLen / 2
	if compareSamples > 100 {
		compareSamples = 100 // 最多比较100个样本
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

	// 归一化相似度（0-1）
	maxDiff := int64(65536) * int64(compareSamples) // 16-bit最大值
	if maxDiff == 0 {
		return 0.0
	}

	similarity := 1.0 - float64(diffSum)/float64(maxDiff)
	if similarity < 0 {
		return 0.0
	}
	return similarity
}

// Clear 清空状态
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ttsOutputBuffer = m.ttsOutputBuffer[:0]
	m.ttsOutputIndex = 0
}

// SetEchoSuppression 设置回音抑制开关
func (m *Manager) SetEchoSuppression(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.echoSuppression = enabled
}

// abs 返回绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// abs64 返回绝对值
func abs64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// min 返回最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
