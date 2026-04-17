package sessions

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/voice/constants"
	"go.uber.org/zap"
)

// EchoFilterComponent 回声过滤组件
// 在 TTS 播放期间，将音频替换为静音帧，防止 ASR 识别 AI 自己的声音
// 但仍然发送数据以保持 ASR 连接活跃
type EchoFilterComponent struct {
	logger   *zap.Logger
	pipeline *ASRPipeline // 引用父pipeline以检查TTS状态
}

// NewEchoFilterComponent 创建回声过滤组件
func NewEchoFilterComponent(logger *zap.Logger, pipeline *ASRPipeline) *EchoFilterComponent {
	return &EchoFilterComponent{
		logger:   logger,
		pipeline: pipeline,
	}
}

// Name 返回组件名称
func (e *EchoFilterComponent) Name() string {
	return constants.COMPONENT_ECHO_FILTER
}

// Process 处理音频数据
// 如果 TTS 正在播放，将音频替换为静音帧
// 否则直接通过原始音频
func (e *EchoFilterComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	pcmData, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("invalid data type for echo filter")
	}

	// 检查 TTS 是否正在播放
	if e.pipeline != nil && e.pipeline.IsTTSPlaying() {
		silentFrame := make([]byte, len(pcmData))
		return silentFrame, true, nil
	}

	// TTS 未播放，直接通过原始音频
	return pcmData, true, nil
}
