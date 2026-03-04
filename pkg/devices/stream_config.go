package devices

import "github.com/gen2brain/malgo"

// StreamConfig 描述音频流的参数
// 默认值将使用默认设备的默认值
type StreamConfig struct {
	// Format 音频格式，如果为 FormatUnknown 则使用设备默认值
	Format malgo.FormatType
	// Channels 声道数，如果为 0 则使用设备默认值
	Channels int
	// SampleRate 采样率，如果为 0 则使用设备默认值
	SampleRate int
	// AlsaNoMMap ALSA NoMMap 设置，默认 1
	AlsaNoMMap uint32
}

// DefaultStreamConfig 返回默认的流配置
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		Format:     malgo.FormatS16,
		Channels:   1,
		SampleRate: 44100,
		AlsaNoMMap: 1,
	}
}

// asDeviceConfig 将 StreamConfig 转换为 malgo.DeviceConfig
func (config StreamConfig) asDeviceConfig(deviceType malgo.DeviceType) malgo.DeviceConfig {
	deviceConfig := malgo.DefaultDeviceConfig(deviceType)

	if config.Format != malgo.FormatUnknown {
		deviceConfig.Capture.Format = config.Format
		deviceConfig.Playback.Format = config.Format
	}

	if config.Channels != 0 {
		deviceConfig.Capture.Channels = uint32(config.Channels)
		deviceConfig.Playback.Channels = uint32(config.Channels)
	}

	if config.SampleRate != 0 {
		deviceConfig.SampleRate = uint32(config.SampleRate)
	}

	if config.AlsaNoMMap != 0 {
		deviceConfig.Alsa.NoMMap = config.AlsaNoMMap
	}

	return deviceConfig
}
