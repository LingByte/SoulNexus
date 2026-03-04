package voice

import (
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
)

// VADDetector 语音活动检测器
type VADDetector struct {
	mu                      sync.RWMutex
	enabled                 bool
	threshold               float64 // RMS 阈值（绝对值）
	adaptiveThreshold       float64 // 自适应阈值（相对值，相对于环境噪音）
	consecutiveFramesNeeded int     // 需要连续超过阈值的帧数
	frameCounter            int     // 当前连续帧计数
	logger                  *zap.Logger
	lastLogTime             time.Time // 上次日志时间（用于限流）
	// 自适应阈值相关
	noiseLevel      float64   // 环境噪音水平（滑动平均）
	noiseSamples    []float64 // 最近的噪音样本（用于计算滑动平均）
	maxNoiseSamples int       // 最大噪音样本数
}

// NewVADDetector 创建新的 VAD 检测器
func NewVADDetector() *VADDetector {
	return &VADDetector{
		enabled:                 true,
		threshold:               500.0, // 绝对值阈值（作为后备）
		adaptiveThreshold:       0,     // 自适应阈值（0表示未初始化）
		consecutiveFramesNeeded: 1,     // 改为1帧，更敏感
		frameCounter:            0,
		logger:                  nil,
		lastLogTime:             time.Now(),
		noiseLevel:              0,
		noiseSamples:            make([]float64, 0),
		maxNoiseSamples:         20, // 保留最近20个样本用于计算平均噪音
	}
}

// SetLogger 设置日志记录器
func (v *VADDetector) SetLogger(logger *zap.Logger) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.logger = logger
}

// CheckBargeIn 检查是否应该中断 TTS（barge-in 检测）
// 返回 true 如果检测到用户说话
func (v *VADDetector) CheckBargeIn(pcmData []byte, ttsPlaying bool) bool {
	if len(pcmData) < 2 {
		return false
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	// 只在 TTS 播放时检测
	if !v.enabled || !ttsPlaying {
		v.frameCounter = 0
		return false
	}

	// 计算音频能量 (RMS)
	rms := calculateRMS(pcmData)

	// 更新噪音水平（使用滑动平均，只记录低能量的样本作为噪音）
	if rms < 200 { // 只将低能量样本作为噪音
		v.noiseSamples = append(v.noiseSamples, rms)
		if len(v.noiseSamples) > v.maxNoiseSamples {
			v.noiseSamples = v.noiseSamples[1:]
		}
		// 计算平均噪音
		var sum float64
		for _, sample := range v.noiseSamples {
			sum += sample
		}
		if len(v.noiseSamples) > 0 {
			v.noiseLevel = sum / float64(len(v.noiseSamples))
			// 自适应阈值 = 噪音水平的 3 倍，但至少为 50，最多为绝对值阈值
			v.adaptiveThreshold = v.noiseLevel * 3.0
			if v.adaptiveThreshold < 50 {
				v.adaptiveThreshold = 50
			}
			if v.adaptiveThreshold > v.threshold {
				v.adaptiveThreshold = v.threshold
			}
		}
	}

	// 使用自适应阈值（如果已初始化）或绝对值阈值
	effectiveThreshold := v.threshold
	if v.adaptiveThreshold > 0 {
		effectiveThreshold = v.adaptiveThreshold
	}

	// 限流日志：每秒最多记录一次
	now := time.Now()
	shouldLog := v.logger != nil && now.Sub(v.lastLogTime) >= time.Second

	// 检查能量是否超过阈值
	if rms > effectiveThreshold {
		v.frameCounter++
		if shouldLog {
			v.lastLogTime = now
			v.logger.Debug("VAD检测：音频能量超过阈值",
				zap.Float64("rms", rms),
				zap.Float64("effectiveThreshold", effectiveThreshold),
				zap.Float64("absoluteThreshold", v.threshold),
				zap.Float64("adaptiveThreshold", v.adaptiveThreshold),
				zap.Float64("noiseLevel", v.noiseLevel),
				zap.Int("frameCounter", v.frameCounter),
				zap.Int("framesNeeded", v.consecutiveFramesNeeded),
			)
		}
		// 达到连续帧数要求，触发 barge-in
		if v.frameCounter >= v.consecutiveFramesNeeded {
			if v.logger != nil {
				v.logger.Info("VAD触发barge-in",
					zap.Float64("rms", rms),
					zap.Float64("effectiveThreshold", effectiveThreshold),
					zap.Float64("adaptiveThreshold", v.adaptiveThreshold),
					zap.Float64("noiseLevel", v.noiseLevel),
					zap.Int("frames", v.frameCounter),
				)
			}
			v.frameCounter = 0 // 重置计数器
			return true
		}
	} else {
		// 能量低于阈值，重置计数器
		if v.frameCounter > 0 && shouldLog {
			v.lastLogTime = now
			v.logger.Debug("VAD检测：音频能量低于阈值，重置计数器",
				zap.Float64("rms", rms),
				zap.Float64("effectiveThreshold", effectiveThreshold),
				zap.Float64("adaptiveThreshold", v.adaptiveThreshold),
				zap.Float64("noiseLevel", v.noiseLevel),
				zap.Int("previousFrameCounter", v.frameCounter),
			)
		}
		v.frameCounter = 0
	}

	return false
}

// SetEnabled 设置 VAD 是否启用
func (v *VADDetector) SetEnabled(enabled bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.enabled = enabled
	if !enabled {
		v.frameCounter = 0
	}
}

// SetThreshold 设置 RMS 阈值
func (v *VADDetector) SetThreshold(threshold float64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.threshold = threshold
}

// SetConsecutiveFrames 设置需要连续超过阈值的帧数
func (v *VADDetector) SetConsecutiveFrames(frames int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.consecutiveFramesNeeded = frames
}

// calculateRMS 计算 16-bit PCM 音频数据的 RMS (Root Mean Square)
// 返回值范围：0 到 32768（对于16-bit PCM）
// 正常语音的RMS通常在 500-5000 之间，静音通常在 0-100 之间
func calculateRMS(pcmData []byte) float64 {
	if len(pcmData) < 2 {
		return 0
	}

	var sumSquares float64
	sampleCount := len(pcmData) / 2

	if sampleCount == 0 {
		return 0
	}

	for i := 0; i < len(pcmData)-1; i += 2 {
		// 转换为 int16 (little-endian)
		sample := int16(pcmData[i]) | int16(pcmData[i+1])<<8
		// 使用绝对值
		absSample := math.Abs(float64(sample))
		sumSquares += absSample * absSample
	}

	return math.Sqrt(sumSquares / float64(sampleCount))
}
