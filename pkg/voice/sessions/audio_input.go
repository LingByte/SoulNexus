package sessions

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/voice/constants"
	"go.uber.org/zap"
)

// ASRInputComponent ASR 输入阶段（发送到 ASR）
type ASRInputComponent struct {
	asr      recognizer.TranscribeService
	logger   *zap.Logger
	metrics  *PipelineMetrics
	pipeline *ASRPipeline // 引用父pipeline以检查TTS状态
}

func (s *ASRInputComponent) Name() string {
	return constants.COMPONENT_INPUT
}

func (s *ASRInputComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	pcmData, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("invalid data type")
	}

	// 记录 PCM 数据大小（用于计算音频时长）
	if s.metrics != nil {
		s.metrics.mu.Lock()
		s.metrics.TotalAudioBytes += len(pcmData)
		s.metrics.mu.Unlock()
	}

	// 发送音频数据给 ASR
	// 注意：此时音频可能已经被 EchoFilterComponent 替换为静音帧（如果 TTS 正在播放）
	err := s.asr.SendAudioBytes(pcmData)
	if err != nil {
		s.logger.Error(fmt.Sprintf("[ASRInput] 发送失败, %v", err))
		if s.pipeline != nil {
			go s.pipeline.TriggerReconnect()
		}
		return nil, false, err
	}
	return nil, true, nil
}
