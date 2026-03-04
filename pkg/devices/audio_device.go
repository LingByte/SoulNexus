package devices

import (
	"fmt"
	"sync"

	"github.com/gen2brain/malgo"
)

// AudioDevice 音频设备工具类，支持录制和播放
type AudioDevice struct {
	ctx              *malgo.AllocatedContext
	captureDevice    *malgo.Device
	playbackDevice   *malgo.Device
	config           *DeviceConfig
	capturedSamples  []byte
	playbackPosition uint32
	mu               sync.RWMutex
	isRecording      bool
	isPlaying        bool
}

// DeviceConfig 设备配置
type DeviceConfig struct {
	// 采样率，默认 44100
	SampleRate uint32
	// 声道数，默认 1（单声道）
	Channels uint32
	// 音频格式，默认 FormatS16
	Format malgo.FormatType
	// ALSA NoMMap 设置，默认 1
	AlsaNoMMap uint32
	// 日志回调函数，可选
	LogCallback func(message string)
}

// DefaultDeviceConfig 返回默认设备配置
func DefaultDeviceConfig() *DeviceConfig {
	return &DeviceConfig{
		SampleRate: 44100,
		Channels:   1,
		Format:     malgo.FormatS16,
		AlsaNoMMap: 1,
		LogCallback: func(message string) {
			// 默认不输出日志
		},
	}
}

// NewAudioDevice 创建新的音频设备实例
func NewAudioDevice(config *DeviceConfig) (*AudioDevice, error) {
	if config == nil {
		config = DefaultDeviceConfig()
	}

	// 设置默认值
	if config.SampleRate == 0 {
		config.SampleRate = 44100
	}
	if config.Channels == 0 {
		config.Channels = 1
	}
	if config.Format == 0 {
		config.Format = malgo.FormatS16
	}
	if config.AlsaNoMMap == 0 {
		config.AlsaNoMMap = 1
	}
	if config.LogCallback == nil {
		config.LogCallback = func(message string) {}
	}

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, config.LogCallback)
	if err != nil {
		return nil, fmt.Errorf("初始化音频上下文失败: %w", err)
	}

	return &AudioDevice{
		ctx:             ctx,
		config:          config,
		capturedSamples: make([]byte, 0),
	}, nil
}

// StartRecording 开始录制音频
func (ad *AudioDevice) StartRecording() error {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	if ad.isRecording {
		return fmt.Errorf("已经在录制中")
	}

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = ad.config.Format
	deviceConfig.Capture.Channels = ad.config.Channels
	deviceConfig.SampleRate = ad.config.SampleRate
	deviceConfig.Alsa.NoMMap = ad.config.AlsaNoMMap

	// 清空之前的录制数据
	ad.capturedSamples = make([]byte, 0)

	onRecvFrames := func(pSample2, pSample []byte, framecount uint32) {
		ad.mu.Lock()
		defer ad.mu.Unlock()
		ad.capturedSamples = append(ad.capturedSamples, pSample...)
	}

	captureCallbacks := malgo.DeviceCallbacks{
		Data: onRecvFrames,
	}

	device, err := malgo.InitDevice(ad.ctx.Context, deviceConfig, captureCallbacks)
	if err != nil {
		return fmt.Errorf("初始化录制设备失败: %w", err)
	}

	err = device.Start()
	if err != nil {
		device.Uninit()
		return fmt.Errorf("启动录制设备失败: %w", err)
	}

	ad.captureDevice = device
	ad.isRecording = true
	return nil
}

// StopRecording 停止录制
func (ad *AudioDevice) StopRecording() error {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	if !ad.isRecording || ad.captureDevice == nil {
		return fmt.Errorf("当前未在录制")
	}

	ad.captureDevice.Uninit()
	ad.captureDevice = nil
	ad.isRecording = false
	return nil
}

// StartPlayback 开始播放录制的音频
func (ad *AudioDevice) StartPlayback() error {
	return ad.StartPlaybackFromData(ad.capturedSamples)
}

// StartPlaybackFromData 从指定的音频数据开始播放
func (ad *AudioDevice) StartPlaybackFromData(audioData []byte) error {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	if ad.isPlaying {
		return fmt.Errorf("正在播放中")
	}

	if len(audioData) == 0 {
		return fmt.Errorf("音频数据为空")
	}

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	deviceConfig.Playback.Format = ad.config.Format
	deviceConfig.Playback.Channels = ad.config.Channels
	deviceConfig.SampleRate = ad.config.SampleRate
	deviceConfig.Alsa.NoMMap = ad.config.AlsaNoMMap

	// 重置播放位置
	ad.playbackPosition = 0

	sizeInBytes := uint32(malgo.SampleSizeInBytes(deviceConfig.Playback.Format))
	playbackData := make([]byte, len(audioData))
	copy(playbackData, audioData)

	onSendFrames := func(pSample, nil []byte, framecount uint32) {
		ad.mu.Lock()
		defer ad.mu.Unlock()

		samplesToRead := framecount * ad.config.Channels * sizeInBytes
		remaining := uint32(len(playbackData)) - ad.playbackPosition

		if samplesToRead > remaining {
			samplesToRead = remaining
		}

		if samplesToRead > 0 {
			copy(pSample, playbackData[ad.playbackPosition:ad.playbackPosition+samplesToRead])
			ad.playbackPosition += samplesToRead
		}

		// 如果播放完毕，填充静音
		if samplesToRead < framecount*ad.config.Channels*sizeInBytes {
			for i := samplesToRead; i < framecount*ad.config.Channels*sizeInBytes; i++ {
				pSample[i] = 0
			}
		}

		// 如果播放完毕，停止设备
		if ad.playbackPosition >= uint32(len(playbackData)) {
			ad.playbackPosition = 0
		}
	}

	playbackCallbacks := malgo.DeviceCallbacks{
		Data: onSendFrames,
	}

	device, err := malgo.InitDevice(ad.ctx.Context, deviceConfig, playbackCallbacks)
	if err != nil {
		return fmt.Errorf("初始化播放设备失败: %w", err)
	}

	err = device.Start()
	if err != nil {
		device.Uninit()
		return fmt.Errorf("启动播放设备失败: %w", err)
	}

	ad.playbackDevice = device
	ad.isPlaying = true
	return nil
}

// StopPlayback 停止播放
func (ad *AudioDevice) StopPlayback() error {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	if !ad.isPlaying || ad.playbackDevice == nil {
		return fmt.Errorf("当前未在播放")
	}

	ad.playbackDevice.Uninit()
	ad.playbackDevice = nil
	ad.isPlaying = false
	ad.playbackPosition = 0
	return nil
}

// StartDuplex 启动双工模式（同时录制和播放）
// playbackData 是要播放的音频数据，如果为 nil 则只播放静音
func (ad *AudioDevice) StartDuplex(playbackData []byte) error {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	if ad.isRecording || ad.isPlaying {
		return fmt.Errorf("设备正在使用中")
	}

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Duplex)
	deviceConfig.Capture.Format = ad.config.Format
	deviceConfig.Capture.Channels = ad.config.Channels
	deviceConfig.Playback.Format = ad.config.Format
	deviceConfig.Playback.Channels = ad.config.Channels
	deviceConfig.SampleRate = ad.config.SampleRate
	deviceConfig.Alsa.NoMMap = ad.config.AlsaNoMMap

	// 清空之前的录制数据
	ad.capturedSamples = make([]byte, 0)
	ad.playbackPosition = 0

	sizeInBytes := uint32(malgo.SampleSizeInBytes(deviceConfig.Capture.Format))
	playbackBuf := make([]byte, len(playbackData))
	if playbackData != nil {
		copy(playbackBuf, playbackData)
	}

	onDuplexFrames := func(pOutputSample, pInputSample []byte, framecount uint32) {
		ad.mu.Lock()
		defer ad.mu.Unlock()

		// 录制部分
		ad.capturedSamples = append(ad.capturedSamples, pInputSample...)

		// 播放部分
		if playbackData != nil && len(playbackBuf) > 0 {
			samplesToRead := framecount * ad.config.Channels * sizeInBytes
			remaining := uint32(len(playbackBuf)) - ad.playbackPosition

			if samplesToRead > remaining {
				samplesToRead = remaining
			}

			if samplesToRead > 0 {
				copy(pOutputSample, playbackBuf[ad.playbackPosition:ad.playbackPosition+samplesToRead])
				ad.playbackPosition += samplesToRead
			}

			// 如果播放完毕，填充静音
			if samplesToRead < framecount*ad.config.Channels*sizeInBytes {
				for i := samplesToRead; i < framecount*ad.config.Channels*sizeInBytes; i++ {
					pOutputSample[i] = 0
				}
			}

			// 如果播放完毕，重置位置（循环播放）
			if ad.playbackPosition >= uint32(len(playbackBuf)) {
				ad.playbackPosition = 0
			}
		} else {
			// 只播放静音
			for i := range pOutputSample {
				pOutputSample[i] = 0
			}
		}
	}

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: onDuplexFrames,
	}

	device, err := malgo.InitDevice(ad.ctx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		return fmt.Errorf("初始化双工设备失败: %w", err)
	}

	err = device.Start()
	if err != nil {
		device.Uninit()
		return fmt.Errorf("启动双工设备失败: %w", err)
	}

	ad.captureDevice = device
	ad.playbackDevice = device
	ad.isRecording = true
	ad.isPlaying = true
	return nil
}

// StopDuplex 停止双工模式
func (ad *AudioDevice) StopDuplex() error {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	if !ad.isRecording && !ad.isPlaying {
		return fmt.Errorf("当前未在双工模式")
	}

	if ad.captureDevice != nil {
		ad.captureDevice.Uninit()
		ad.captureDevice = nil
	}
	if ad.playbackDevice != nil && ad.playbackDevice != ad.captureDevice {
		ad.playbackDevice.Uninit()
		ad.playbackDevice = nil
	}

	ad.isRecording = false
	ad.isPlaying = false
	ad.playbackPosition = 0
	return nil
}

// GetCapturedData 获取录制的音频数据
func (ad *AudioDevice) GetCapturedData() []byte {
	ad.mu.RLock()
	defer ad.mu.RUnlock()

	data := make([]byte, len(ad.capturedSamples))
	copy(data, ad.capturedSamples)
	return data
}

// ClearCapturedData 清空录制的音频数据
func (ad *AudioDevice) ClearCapturedData() {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	ad.capturedSamples = make([]byte, 0)
	ad.playbackPosition = 0
}

// IsRecording 检查是否正在录制
func (ad *AudioDevice) IsRecording() bool {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	return ad.isRecording
}

// IsPlaying 检查是否正在播放
func (ad *AudioDevice) IsPlaying() bool {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	return ad.isPlaying
}

// GetCapturedDataSize 获取录制的音频数据大小（字节数）
func (ad *AudioDevice) GetCapturedDataSize() uint32 {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	return uint32(len(ad.capturedSamples))
}

// Close 关闭设备并释放资源
func (ad *AudioDevice) Close() error {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	var err error

	if ad.captureDevice != nil {
		ad.captureDevice.Uninit()
		ad.captureDevice = nil
	}

	if ad.playbackDevice != nil && ad.playbackDevice != ad.captureDevice {
		ad.playbackDevice.Uninit()
		ad.playbackDevice = nil
	}

	if ad.ctx != nil {
		ad.ctx.Uninit()
		ad.ctx.Free()
		ad.ctx = nil
	}

	ad.isRecording = false
	ad.isPlaying = false
	ad.capturedSamples = nil
	ad.playbackPosition = 0

	return err
}
