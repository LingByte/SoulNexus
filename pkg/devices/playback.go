package devices

import (
	"fmt"
	"sync"

	"github.com/gen2brain/malgo"
)

// StreamAudioPlayer 用于流式播放音频数据的播放器
type StreamAudioPlayer struct {
	ctx         *malgo.AllocatedContext
	device      *malgo.Device
	channels    uint32
	sampleRate  uint32
	audioBuffer chan []byte
	format      malgo.FormatType
	// 内部缓冲区，用于平滑数据流
	internalBuffer []byte
	mu             sync.RWMutex
}

// NewStreamAudioPlayer 创建流式音频播放器
// channels: 声道数（1=单声道, 2=立体声）
// sampleRate: 采样率（如 8000, 16000, 48000）
// format: 音频格式（malgo.FormatS16 表示 16-bit signed integer）
func NewStreamAudioPlayer(channels uint32, sampleRate uint32, format malgo.FormatType) (*StreamAudioPlayer, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		fmt.Printf("<%v>\n", message)
	})
	if err != nil {
		return nil, err
	}

	// 增大缓冲区以减少音频不连续（约4秒的缓冲）
	// 对于8kHz，20ms帧，4秒 = 200帧
	bufferSize := 200
	player := &StreamAudioPlayer{
		ctx:            ctx,
		channels:       channels,
		sampleRate:     sampleRate,
		format:         format,
		audioBuffer:    make(chan []byte, bufferSize),
		internalBuffer: make([]byte, 0, 8192), // 预分配内部缓冲区
	}

	return player, nil
}

// Play 开始播放音频流
func (p *StreamAudioPlayer) Play() error {
	deviceConfig := malgo.DefaultDeviceConfig(malgo.Playback)
	deviceConfig.Playback.Format = p.format
	deviceConfig.Playback.Channels = p.channels
	deviceConfig.SampleRate = p.sampleRate
	deviceConfig.Alsa.NoMMap = 1

	// 计算每帧的字节数
	bytesPerSample := 2 // FormatS16 = 2 bytes per sample
	bytesPerFrame := bytesPerSample * int(p.channels)

	// 数据回调函数 - 改进版本，使用内部缓冲区平滑数据流
	onSamples := func(pOutputSample, pInputSamples []byte, framecount uint32) {
		bytesNeeded := int(framecount) * bytesPerFrame

		p.mu.Lock()
		defer p.mu.Unlock()

		// 从channel填充内部缓冲区
	bufferLoop:
		for len(p.internalBuffer) < bytesNeeded {
			select {
			case data := <-p.audioBuffer:
				if len(data) > 0 {
					p.internalBuffer = append(p.internalBuffer, data...)
				}
			default:
				// 没有更多数据，跳出 for 循环
				break bufferLoop
			}
		}

		// 从内部缓冲区复制数据到输出
		if len(p.internalBuffer) >= bytesNeeded {
			copy(pOutputSample, p.internalBuffer[:bytesNeeded])
			// 移除已使用的数据
			p.internalBuffer = p.internalBuffer[bytesNeeded:]
		} else {
			// 数据不足，复制现有数据并填充静音
			copied := copy(pOutputSample, p.internalBuffer)
			p.internalBuffer = p.internalBuffer[:0]
			// 平滑填充静音（淡出）
			if copied > 0 {
				// 对最后几个样本进行淡出处理
				fadeSamples := copied / 2
				if fadeSamples > 64 {
					fadeSamples = 64 // 最多淡出64字节（32个样本）
				}
				if fadeSamples > 0 {
					for i := copied - fadeSamples; i < copied; i += 2 {
						if i+1 < copied {
							sample := int16(pOutputSample[i]) | int16(pOutputSample[i+1])<<8
							fadeFactor := float64(copied-i) / float64(fadeSamples)
							sample = int16(float64(sample) * fadeFactor)
							pOutputSample[i] = byte(sample)
							pOutputSample[i+1] = byte(sample >> 8)
						}
					}
				}
			}
			// 填充剩余部分为静音
			for i := copied; i < bytesNeeded; i++ {
				pOutputSample[i] = 0
			}
		}
	}

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: onSamples,
	}

	var err error
	p.device, err = malgo.InitDevice(p.ctx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		return err
	}

	err = p.device.Start()
	if err != nil {
		return err
	}

	return nil
}

// Write 写入音频数据到播放缓冲区
func (p *StreamAudioPlayer) Write(data []byte) error {
	select {
	case p.audioBuffer <- data:
		return nil
	default:
		return fmt.Errorf("音频缓冲区已满")
	}
}

// ClearBuffer 清空播放缓冲区，用于防止音频重复/回声
func (p *StreamAudioPlayer) ClearBuffer() {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 清空内部缓冲区
	p.internalBuffer = p.internalBuffer[:0]

	// 清空channel中的数据
	for {
		select {
		case <-p.audioBuffer:
			// 丢弃数据
		default:
			return
		}
	}
}

// Close 关闭流式播放器
func (p *StreamAudioPlayer) Close() {
	if p.device != nil {
		p.device.Uninit()
	}
	if p.ctx != nil {
		p.ctx.Uninit()
		p.ctx.Free()
	}
	close(p.audioBuffer)
}

func main() {
	//if len(os.Args) < 2 {
	//	fmt.Println("请提供音频文件路径")
	//	os.Exit(1)
	//}
	//
	//player, err := NewAudioPlayer(os.Args[1])
	//if err != nil {
	//	fmt.Println("创建播放器失败:", err)
	//	os.Exit(1)
	//}
	//defer player.Close()
	//
	//err = player.Play()
	//if err != nil {
	//	fmt.Println("播放失败:", err)
	//	os.Exit(1)
	//}
	//
	//fmt.Println("正在播放，按回车键退出...")
	//fmt.Scanln()
	//
	//// 方式1: 从内存数据播放，自动检测格式
	//audioData := []byte{...} // 你的音频数据
	//player, err := NewAudioPlayerFromData(audioData, "")
	//if err != nil {
	//	// 处理错误
	//}
	//defer player.Close()
	//player.Play()
	//
	//// 方式2: 从内存数据播放，指定格式
	//player, err := NewAudioPlayerFromData(audioData, "wav")
	//
	//// 方式3: 从 io.Reader 播放
	//reader := bytes.NewReader(audioData)
	//player, err := NewAudioPlayerFromReader(reader, "mp3")
}
