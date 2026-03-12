package sessions

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/hardware/constants"
	"go.uber.org/zap"

	"github.com/LingByte/SoulNexus/pkg/vad"
)

// VADComponent VAD 管道组件（使用远程 SileroVAD 服务）
// 用于在 TTS 播放期间检测用户语音（Barge-in）
type VADComponent struct {
	enabled     bool
	logger      *zap.Logger
	mu          sync.RWMutex
	lastLogTime time.Time

	// VAD SDK 相关
	sessionManager *vad.SessionManager
	sessionID      string
	threshold      float64 // VAD 阈值（0-1）

	// Barge-in 相关
	bargeInCallback   func()      // 检测到 barge-in 时的回调
	getTTSPlaying     func() bool // 获取 TTS 播放状态的回调
	frameCounter      int         // 连续检测到语音的帧数
	consecutiveFrames int         // 需要连续检测到语音的帧数
}

// NewVADComponent 创建 VAD 管道组件（使用远程 VAD 服务）
//   - enabled: 是否启用 VAD
//   - logger: 日志记录器
//   - vadServiceURL: VAD 服务 URL（例如 http://localhost:7073）
//   - sessionID: 会话 ID（用于区分不同用户）
//   - threshold: VAD 阈值（0-1，例如 0.95）
//   - *VADComponent: VAD 组件实例
//   - error: 错误信息
func NewVADComponent(
	enabled bool,
	logger *zap.Logger,
	vadServiceURL string,
	sessionID string,
	threshold float64,
) (*VADComponent, error) {
	if logger == nil {
		logger = zap.NewNop()
	}
	client := vad.NewClient(vadServiceURL, logger)
	if err := client.HealthCheck(); err != nil {
		logger.Error("[VAD] 服务健康检查失败", zap.Error(err))
		return nil, fmt.Errorf("VAD service health check failed: %w", err)
	}
	component := &VADComponent{
		enabled:           enabled,
		logger:            logger,
		lastLogTime:       time.Now(),
		sessionManager:    vad.NewSessionManager(client, logger),
		sessionID:         sessionID,
		threshold:         threshold,
		frameCounter:      0,
		consecutiveFrames: 5, // 改为 5 帧（100ms），减少误触发
	}

	logger.Info("[VAD] 组件已初始化",
		zap.Bool("enabled", enabled),
		zap.String("service_url", vadServiceURL),
		zap.String("session_id", sessionID),
		zap.Float64("threshold", threshold))

	return component, nil
}

// SetBargeInCallback 设置 barge-in 回调
// 当检测到用户语音时调用此回调
func (v *VADComponent) SetBargeInCallback(callback func()) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.bargeInCallback = callback
}

// SetTTSPlayingCallback 设置获取 TTS 播放状态的回调
// 用于判断是否在 TTS 播放期间
func (v *VADComponent) SetTTSPlayingCallback(callback func() bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.getTTSPlaying = callback
}

// Name 返回组件名称
func (v *VADComponent) Name() string {
	return constants.COMPONENT_VAD
}

// Process 处理音频数据
// 返回：处理后的数据，是否继续处理，错误
func (v *VADComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	pcmData, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("invalid data type for VAD")
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	// 如果 VAD 未启用，直接通过
	if !v.enabled {
		return pcmData, true, nil
	}

	// 检查是否在 TTS 播放期间
	isTTSPlaying := false
	if v.getTTSPlaying != nil {
		isTTSPlaying = v.getTTSPlaying()
	}

	// 只在 TTS 播放期间才进行 VAD 检测（用于 Barge-in）
	if !isTTSPlaying {
		v.frameCounter = 0
		return pcmData, true, nil
	}

	// 调用远程 VAD 服务
	result, err := v.sessionManager.ProcessAudio(v.sessionID, pcmData, "pcm", v.threshold)
	if err != nil {
		v.logger.Error("[VAD] 检测失败", zap.Error(err))
		// 错误时直接通过
		return pcmData, true, nil
	}

	// 限流日志：每 3 秒最多记录一次
	now := time.Now()
	shouldLog := now.Sub(v.lastLogTime) >= 3*time.Second

	// 检查是否有语音
	if result.HaveVoice {
		v.frameCounter++
		if shouldLog {
			v.lastLogTime = now
			v.logger.Debug("[VAD] 检测到语音",
				zap.Float64("speech_prob", result.SpeechProb),
				zap.Int("frameCounter", v.frameCounter),
				zap.Bool("voice_stop", result.VoiceStop),
			)
		}

		// 达到连续帧数要求，触发 barge-in
		if v.frameCounter >= v.consecutiveFrames {
			if v.bargeInCallback != nil {
				v.logger.Info("[VAD] 检测到用户语音，触发 Barge-in",
					zap.Float64("speech_prob", result.SpeechProb),
					zap.Int("frames", v.frameCounter))
				v.bargeInCallback()
				v.frameCounter = 0
			}
		}
	} else {
		// 无语音，重置计数器
		if v.frameCounter > 0 && shouldLog {
			v.lastLogTime = now
			v.logger.Debug("[VAD] 未检测到语音，重置计数器",
				zap.Float64("speech_prob", result.SpeechProb),
				zap.Int("previousFrameCounter", v.frameCounter),
			)
		}
		v.frameCounter = 0
	}

	return pcmData, true, nil
}

// SetEnabled 设置 VAD 是否启用
func (v *VADComponent) SetEnabled(enabled bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.enabled = enabled
	if !enabled {
		v.frameCounter = 0
	}
	v.logger.Info("[VAD] 状态已更新", zap.Bool("enabled", enabled))
}

// SetConsecutiveFrames 设置需要连续检测到语音的帧数
func (v *VADComponent) SetConsecutiveFrames(frames int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if frames <= 0 {
		frames = 1
	}
	v.consecutiveFrames = frames
	v.logger.Info("[VAD] 连续帧数已更新", zap.Int("frames", frames))
}

// Reset 重置 VAD 状态
func (v *VADComponent) Reset() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.frameCounter = 0
	v.logger.Debug("[VAD] 状态已重置")
}

// ResetSession 重置会话
func (v *VADComponent) ResetSession() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.frameCounter = 0
	return v.sessionManager.ResetSession(v.sessionID)
}

// Close 关闭 VAD 组件，释放资源
func (v *VADComponent) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.sessionManager != nil {
		if err := v.sessionManager.Close(); err != nil {
			v.logger.Error("[VAD] 关闭会话管理器失败", zap.Error(err))
			return err
		}
	}

	v.logger.Info("[VAD] 组件已关闭")
	return nil
}

// GetSessionInfo 获取会话信息（用于调试）
func (v *VADComponent) GetSessionInfo() map[string]interface{} {
	v.mu.RLock()
	defer v.mu.RUnlock()

	session := v.sessionManager.GetSession(v.sessionID)
	if session == nil {
		return map[string]interface{}{
			"session_id": v.sessionID,
			"status":     "not_found",
		}
	}

	return map[string]interface{}{
		"session_id":         v.sessionID,
		"have_voice":         session.HaveVoice,
		"last_speech_prob":   session.LastSpeechProb,
		"frame_counter":      v.frameCounter,
		"consecutive_frames": v.consecutiveFrames,
	}
}
