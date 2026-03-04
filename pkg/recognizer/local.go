package recognizer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// LocalASRProvider 本地ASR提供商类型
type LocalASRProvider string

const (
	LocalASRProviderWhisperCpp LocalASRProvider = "whisper_cpp"
	LocalASRProviderLocal      LocalASRProvider = "local_cmd"
)

// LocalASRConfig 本地ASR配置
type LocalASRConfig struct {
	Provider     LocalASRProvider `json:"provider"`     // ASR提供商
	ModelPath    string           `json:"modelPath"`    // 模型文件路径
	Language     string           `json:"language"`     // 语言代码
	SampleRate   int              `json:"sampleRate"`   // 采样率
	Channels     int              `json:"channels"`     // 声道数
	BitDepth     int              `json:"bitDepth"`     // 位深度
	BufferSize   int              `json:"bufferSize"`   // 缓冲区大小（毫秒）
	EnableVAD    bool             `json:"enableVAD"`    // 是否启用VAD
	VADThreshold float32          `json:"vadThreshold"` // VAD阈值
	Command      string           `json:"command"`      // 外部命令（用于 local_cmd 模式）
}

// NewLocalASRConfig 创建默认本地ASR配置
func NewLocalASRConfig(provider LocalASRProvider, modelPath string) *LocalASRConfig {
	return &LocalASRConfig{
		Provider:     provider,
		ModelPath:    modelPath,
		Language:     "zh-CN",
		SampleRate:   16000,
		Channels:     1,
		BitDepth:     16,
		BufferSize:   100, // 100ms
		EnableVAD:    true,
		VADThreshold: 0.5,
		Command:      "whisper", // 默认命令
	}
}

// LocalASRService 本地ASR服务
type LocalASRService struct {
	config      *LocalASRConfig
	mu          sync.RWMutex
	connected   bool
	ctx         context.Context
	cancel      context.CancelFunc
	audioBuffer []byte
	bufferSize  int

	// 回调函数
	resultCallback TranscribeResult
	errorCallback  ProcessError

	logger *logrus.Logger
}

// NewLocalASRService 创建本地ASR服务
func NewLocalASRService(config *LocalASRConfig) (*LocalASRService, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	if config.ModelPath == "" && config.Provider != LocalASRProviderLocal {
		return nil, fmt.Errorf("模型路径不能为空")
	}

	// 计算缓冲区大小（字节）
	bufferSize := config.SampleRate * config.BitDepth / 8 * config.Channels * config.BufferSize / 1000

	service := &LocalASRService{
		config:      config,
		audioBuffer: make([]byte, 0, bufferSize),
		bufferSize:  bufferSize,
		logger:      logrus.New(),
	}

	return service, nil
}

// Init 初始化服务
func (s *LocalASRService) Init(tr TranscribeResult, er ProcessError) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.resultCallback = tr
	s.errorCallback = er
}

// Vendor 返回供应商名称
func (s *LocalASRService) Vendor() string {
	return fmt.Sprintf("local-%s", s.config.Provider)
}

// ConnAndReceive 连接并接收
func (s *LocalASRService) ConnAndReceive(dialogId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.connected {
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.connected = true

	s.logger.WithField("dialogId", dialogId).Info("本地ASR连接成功")
	return nil
}

// Activity 检查活动状态
func (s *LocalASRService) Activity() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}

// RestartClient 重启客户端
func (s *LocalASRService) RestartClient() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cancel != nil {
		s.cancel()
	}

	s.connected = false
	s.audioBuffer = s.audioBuffer[:0]

	s.logger.Info("本地ASR客户端已重启")
}

// SendAudioBytes 发送音频数据
func (s *LocalASRService) SendAudioBytes(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return fmt.Errorf("ASR服务未连接")
	}

	if len(data) == 0 {
		return nil
	}

	// 添加到缓冲区
	s.audioBuffer = append(s.audioBuffer, data...)

	// 如果缓冲区达到阈值，进行识别
	if len(s.audioBuffer) >= s.bufferSize {
		return s.processAudioBuffer(false)
	}

	return nil
}

// SendEnd 发送结束信号
func (s *LocalASRService) SendEnd() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil
	}

	// 处理剩余的音频数据
	if len(s.audioBuffer) > 0 {
		return s.processAudioBuffer(true)
	}

	return nil
}

// processAudioBuffer 处理音频缓冲区
func (s *LocalASRService) processAudioBuffer(isLast bool) error {
	if len(s.audioBuffer) == 0 {
		return nil
	}

	// 复制缓冲区数据
	audioData := make([]byte, len(s.audioBuffer))
	copy(audioData, s.audioBuffer)
	s.audioBuffer = s.audioBuffer[:0]

	// 在goroutine中进行异步识别
	go func() {
		startTime := time.Now()

		var text string
		var err error

		// 模拟识别过程（实际实现中可以调用外部命令或库）
		switch s.config.Provider {
		case LocalASRProviderWhisperCpp:
			text, err = s.processWithWhisperCpp(audioData)
		case LocalASRProviderLocal:
			text, err = s.processWithLocalCommand(audioData)
		default:
			err = fmt.Errorf("不支持的ASR提供商: %s", s.config.Provider)
		}

		duration := time.Since(startTime)

		if err != nil {
			s.logger.WithError(err).WithField("duration", duration).Error("本地ASR识别失败")
			if s.errorCallback != nil {
				s.errorCallback(err, false)
			}
			return
		}

		// 调用结果回调
		if s.resultCallback != nil && text != "" {
			s.resultCallback(text, isLast, duration, "")
		}
	}()

	return nil
}

// processWithWhisperCpp 使用 whisper.cpp 处理音频
func (s *LocalASRService) processWithWhisperCpp(audioData []byte) (string, error) {
	// 这里可以集成 whisper.cpp 的 Go 绑定
	// 目前返回模拟结果
	s.logger.Debug("使用 whisper.cpp 处理音频", "size", len(audioData))

	// 模拟处理延迟
	time.Sleep(50 * time.Millisecond)

	// 返回模拟识别结果
	return "这是一个模拟的识别结果", nil
}

// processWithLocalCommand 使用本地命令处理音频
func (s *LocalASRService) processWithLocalCommand(audioData []byte) (string, error) {
	// 这里可以调用外部命令行工具
	s.logger.Debug("使用本地命令处理音频", "command", s.config.Command, "size", len(audioData))

	// 模拟处理延迟
	time.Sleep(100 * time.Millisecond)

	// 返回模拟识别结果
	return "本地命令识别结果", nil
}

// StopConn 停止连接
func (s *LocalASRService) StopConn() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil
	}

	if s.cancel != nil {
		s.cancel()
	}

	s.connected = false
	s.audioBuffer = s.audioBuffer[:0]

	s.logger.Info("本地ASR连接已停止")
	return nil
}

// Close 关闭服务
func (s *LocalASRService) Close() error {
	s.StopConn()
	s.logger.Info("本地ASR服务已关闭")
	return nil
}
