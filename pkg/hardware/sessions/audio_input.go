package sessions

import (
	"context"
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/hardware/constants"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
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

	// 使用 AudioManager 处理输入音频（智能回声过滤）
	if s.pipeline != nil && s.pipeline.audioManager != nil {
		ttsPlaying := s.pipeline.IsTTSPlaying()
		processedData, shouldProcess := s.pipeline.audioManager.ProcessInputAudio(pcmData, ttsPlaying)

		if !shouldProcess {
			// 音频被过滤（回声或低能量），发送静音帧保持连接
			silentFrame := make([]byte, len(pcmData))
			err := s.asr.SendAudioBytes(silentFrame)
			if err != nil {
				s.logger.Error(fmt.Sprintf("[ASRInput] 发送静音帧失败, %v", err))
				if s.pipeline != nil {
					go s.pipeline.TriggerReconnect()
				}
				return nil, false, err
			}
			return nil, true, nil
		}

		// 使用处理后的数据
		pcmData = processedData
	}

	// 发送音频数据
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
