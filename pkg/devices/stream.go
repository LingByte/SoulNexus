package devices

import (
	"context"
	"io"
	"sync"

	"github.com/gen2brain/malgo"
)

// StreamContext 流式音频操作的上下文
type StreamContext struct {
	ctx    *malgo.AllocatedContext
	config StreamConfig
	mu     sync.RWMutex
}

// NewStreamContext 创建新的流式音频上下文
// 如果 config 为 nil，则使用默认配置
func NewStreamContext(config *StreamConfig) (*StreamContext, error) {
	if config == nil {
		defaultConfig := DefaultStreamConfig()
		config = &defaultConfig
	}

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}

	return &StreamContext{
		ctx:    ctx,
		config: *config,
	}, nil
}

// Close 关闭流式音频上下文并释放资源
func (sc *StreamContext) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.ctx != nil {
		sc.ctx.Uninit()
		sc.ctx.Free()
		sc.ctx = nil
	}

	return nil
}

// GetContext 获取底层的 malgo 上下文（用于设备列表等功能）
func (sc *StreamContext) GetContext() *malgo.AllocatedContext {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.ctx
}

// Capture 将传入的样本录制到提供的 writer 中
// 该函数在默认上下文中初始化一个捕获设备，使用提供的流配置
// 录制将持续将样本写入 writer，直到 writer 返回错误或 context 信号完成
func (sc *StreamContext) Capture(ctx context.Context, w io.Writer) error {
	sc.mu.RLock()
	deviceConfig := sc.config.asDeviceConfig(malgo.Capture)
	malgoCtx := sc.ctx
	sc.mu.RUnlock()

	if malgoCtx == nil {
		return io.ErrClosedPipe
	}

	abortChan := make(chan error, 1)
	defer close(abortChan)
	aborted := false
	var abortMu sync.Mutex

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: func(outputSamples, inputSamples []byte, frameCount uint32) {
			abortMu.Lock()
			if aborted {
				abortMu.Unlock()
				return
			}
			abortMu.Unlock()

			_, err := w.Write(inputSamples)
			if err != nil {
				abortMu.Lock()
				if !aborted {
					aborted = true
					select {
					case abortChan <- err:
					default:
					}
				}
				abortMu.Unlock()
			}
		},
	}

	return sc.stream(ctx, abortChan, deviceConfig, deviceCallbacks)
}

// Playback 从 reader 流式传输样本到音频设备
// 该函数在默认上下文中初始化一个播放设备，使用提供的流配置
// 播放将持续从 reader 读取样本并播放，直到 reader 返回错误或 context 信号完成
func (sc *StreamContext) Playback(ctx context.Context, r io.Reader) error {
	sc.mu.RLock()
	deviceConfig := sc.config.asDeviceConfig(malgo.Playback)
	malgoCtx := sc.ctx
	sc.mu.RUnlock()

	if malgoCtx == nil {
		return io.ErrClosedPipe
	}

	abortChan := make(chan error, 1)
	defer close(abortChan)
	aborted := false
	var abortMu sync.Mutex

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: func(outputSamples, inputSamples []byte, frameCount uint32) {
			abortMu.Lock()
			if aborted {
				abortMu.Unlock()
				return
			}
			abortMu.Unlock()

			if frameCount == 0 {
				return
			}

			read, err := io.ReadFull(r, outputSamples)
			if read <= 0 {
				if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
					abortMu.Lock()
					if !aborted {
						aborted = true
						select {
						case abortChan <- err:
						default:
						}
					}
					abortMu.Unlock()
				}
				return
			}
		},
	}

	return sc.stream(ctx, abortChan, deviceConfig, deviceCallbacks)
}

// stream 内部流处理函数
func (sc *StreamContext) stream(ctx context.Context, abortChan chan error, deviceConfig malgo.DeviceConfig, deviceCallbacks malgo.DeviceCallbacks) error {
	sc.mu.RLock()
	malgoCtx := sc.ctx
	sc.mu.RUnlock()

	if malgoCtx == nil {
		return io.ErrClosedPipe
	}

	device, err := malgo.InitDevice(malgoCtx.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		return err
	}
	defer device.Uninit()

	err = device.Start()
	if err != nil {
		return err
	}

	ctxChan := ctx.Done()
	if ctxChan != nil {
		select {
		case <-ctxChan:
			err = ctx.Err()
		case err = <-abortChan:
		}
	} else {
		err = <-abortChan
	}

	return err
}

// CaptureToWriter 便捷函数：使用默认上下文进行录制
// 如果需要在多个操作之间共享上下文，建议使用 NewStreamContext 创建 StreamContext
func CaptureToWriter(ctx context.Context, w io.Writer, config StreamConfig) error {
	streamCtx, err := NewStreamContext(&config)
	if err != nil {
		return err
	}
	defer streamCtx.Close()

	return streamCtx.Capture(ctx, w)
}

// PlaybackFromReader 便捷函数：使用默认上下文进行播放
// 如果需要在多个操作之间共享上下文，建议使用 NewStreamContext 创建 StreamContext
func PlaybackFromReader(ctx context.Context, r io.Reader, config StreamConfig) error {
	streamCtx, err := NewStreamContext(&config)
	if err != nil {
		return err
	}
	defer streamCtx.Close()

	return streamCtx.Playback(ctx, r)
}
