package encoder

import (
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/hraban/opus"
)

// createOPUSDecode creates OPUS decoder
// OPUS standard sample rate is 48000Hz, but also supports 8000, 12000, 16000, 24000, 48000
func createOPUSDecode(src, pcm media.CodecConfig) media.EncoderFunc {
	// Use configured sample rate, if not set use OPUS standard sample rate 48000Hz
	sourceSampleRate := src.SampleRate
	if sourceSampleRate == 0 {
		sourceSampleRate = 48000 // OPUS standard sample rate
	}

	// Determine number of channels
	channels := src.Channels
	if channels == 0 {
		channels = 1 // Default mono
	}

	// 创建 OPUS 解码器
	// OPUS 支持的采样率: 8000, 12000, 16000, 24000, 48000
	decoder, err := opus.NewDecoder(sourceSampleRate, channels)
	if err != nil {
		panic(fmt.Errorf("failed to create opus decoder: %w", err))
	}

	// 创建重采样器
	res := media.DefaultResampler(sourceSampleRate, pcm.SampleRate)

	// 从 FrameDuration 解析帧时长（例如 "20ms", "60ms"）
	frameDurationMs := 20 // 默认 20ms
	if src.FrameDuration != "" {
		// 解析 "20ms", "40ms", "60ms" 等格式
		var ms int
		if _, err := fmt.Sscanf(src.FrameDuration, "%dms", &ms); err == nil && ms > 0 {
			frameDurationMs = ms
		}
	}

	// 计算每帧的样本数
	frameSize := sourceSampleRate * frameDurationMs / 1000

	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}

		// 解码 OPUS 数据为 PCM (int16)
		// 创建输出缓冲区
		pcmBuffer := make([]int16, frameSize*channels)
		n, err := decoder.Decode(audioPacket.Payload, pcmBuffer)
		if err != nil {
			return nil, fmt.Errorf("opus decode error: %w", err)
		}

		// 转换 int16 为 []byte
		decodedData := make([]byte, n*channels*2)
		for i := 0; i < n*channels; i++ {
			decodedData[i*2] = byte(pcmBuffer[i])
			decodedData[i*2+1] = byte(pcmBuffer[i] >> 8)
		}

		// 重采样到目标采样率
		if _, err = res.Write(decodedData); err != nil {
			return nil, err
		}

		data := res.Samples()
		if data == nil {
			return nil, nil
		}

		audioPacket.Payload = data
		return []media.MediaPacket{audioPacket}, nil
	}
}

// createOPUSEncode 创建 OPUS 编码器
func createOPUSEncode(src, pcm media.CodecConfig) media.EncoderFunc {
	// 使用配置的目标采样率，如果未设置则使用 OPUS 标准采样率 48000Hz
	targetSampleRate := src.SampleRate
	if targetSampleRate == 0 {
		targetSampleRate = 48000 // OPUS 标准采样率
	}

	// 验证采样率是否为 OPUS 支持的值
	validRates := []int{8000, 12000, 16000, 24000, 48000}
	isValid := false
	for _, rate := range validRates {
		if targetSampleRate == rate {
			isValid = true
			break
		}
	}
	if !isValid {
		// 如果不是有效采样率，使用最接近的有效值
		if targetSampleRate < 10000 {
			targetSampleRate = 8000
		} else if targetSampleRate < 14000 {
			targetSampleRate = 12000
		} else if targetSampleRate < 20000 {
			targetSampleRate = 16000
		} else if targetSampleRate < 36000 {
			targetSampleRate = 24000
		} else {
			targetSampleRate = 48000
		}
	}

	// 确定声道数
	channels := src.Channels
	if channels == 0 {
		channels = 1 // 默认单声道
	}

	// 创建 OPUS 编码器
	// 使用 AppAudio 模式以获得更好的音质（而不是 AppVoIP）
	encoder, err := opus.NewEncoder(targetSampleRate, channels, opus.AppAudio)
	if err != nil {
		panic(fmt.Errorf("failed to create opus encoder: %w", err))
	}

	// 设置复杂度为 10（最高质量，0-10）
	// 更高的复杂度会提高音质但增加 CPU 使用
	if err := encoder.SetComplexity(10); err != nil {
		panic(fmt.Errorf("failed to set opus complexity: %w", err))
	}

	// 创建重采样器
	res := media.DefaultResampler(pcm.SampleRate, targetSampleRate)

	// 从 FrameDuration 解析帧时长（例如 "20ms", "60ms"）
	frameDurationMs := 20 // 默认 20ms
	if src.FrameDuration != "" {
		// 解析 "20ms", "40ms", "60ms" 等格式
		var ms int
		if _, err := fmt.Sscanf(src.FrameDuration, "%dms", &ms); err == nil && ms > 0 {
			frameDurationMs = ms
		}
	}

	// 计算每帧的样本数
	frameSize := targetSampleRate * frameDurationMs / 1000

	return func(packet media.MediaPacket) ([]media.MediaPacket, error) {
		audioPacket, ok := packet.(*media.AudioPacket)
		if !ok {
			return []media.MediaPacket{packet}, nil
		}

		// 重采样到 OPUS 目标采样率
		if _, err := res.Write(audioPacket.Payload); err != nil {
			return nil, err
		}

		data := res.Samples()
		if data == nil {
			return nil, nil
		}

		// 转换 []byte 为 []int16
		pcmSamples := make([]int16, len(data)/2)
		for i := 0; i < len(pcmSamples); i++ {
			pcmSamples[i] = int16(data[i*2]) | int16(data[i*2+1])<<8
		}

		// 计算每帧需要的样本数
		samplesPerFrame := frameSize * channels
		totalSamples := len(pcmSamples)

		// 如果数据不足一帧，直接返回空包
		if totalSamples < samplesPerFrame {
			return []media.MediaPacket{}, nil
		}

		// 只编码第一帧（调用者负责分帧）
		opusBuffer := make([]byte, 4000) // 足够大的缓冲区
		n, err := encoder.Encode(pcmSamples[:samplesPerFrame], opusBuffer)
		if err != nil {
			return nil, fmt.Errorf("opus encode error: %w", err)
		}

		audioPacket.Payload = opusBuffer[:n]
		return []media.MediaPacket{audioPacket}, nil
	}
}
