package hardware

import (
	"context"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/llm"
	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/synthesizer"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SessionConfig 会话配置
type SessionConfig struct {
	Conn         *websocket.Conn
	Credential   *models.UserCredential
	AssistantID  int
	Language     string
	Speaker      string
	Temperature  float64
	MaxTokens    int
	SystemPrompt string
	KnowledgeKey string
	LLMModel     string
	DB           *gorm.DB
	Logger       *zap.Logger
	Context      context.Context
	// VAD 配置
	EnableVAD            bool    // 是否启用VAD
	VADThreshold         float64 // VAD阈值
	VADConsecutiveFrames int     // 需要连续超过阈值的帧数
}

// SessionInterface 语音会话接口
type SessionInterface interface {
	// Start 启动会话
	Start() error

	// Stop 停止会话
	Stop() error

	// HandleAudio 处理音频数据
	HandleAudio(data []byte) error

	// HandleText 处理文本消息
	HandleText(data []byte) error

	// IsActive 检查会话是否活跃
	IsActive() bool
}

// ASRService ASR服务接口
type ASRService interface {
	// Connect 建立连接
	Connect() error

	// Disconnect 断开连接
	Disconnect() error

	// SendAudio 发送音频数据
	SendAudio(data []byte) error

	// IsConnected 检查是否已连接
	IsConnected() bool

	// Activity 检查服务是否活跃
	Activity() bool
}

// TTSService TTS服务接口
type TTSService interface {
	// Synthesize 合成语音
	Synthesize(ctx context.Context, text string) (<-chan []byte, error)

	// Close 关闭服务
	Close() error
}

// LLMService LLM服务接口
type LLMService interface {
	// Query 查询
	Query(ctx context.Context, text string) (string, error)

	// Close 关闭服务
	Close() error
}

// MessageWriter 消息写入器接口
type MessageWriter interface {
	// SendASRResult 发送ASR识别结果
	SendASRResult(text string) error

	// SendTTSAudio 发送TTS音频数据
	SendTTSAudio(data []byte) error

	// SendError 发送错误消息
	SendError(message string, fatal bool) error

	// SendConnected 发送连接成功消息
	SendConnected() error

	// SendLLMResponse 发送LLM响应
	SendLLMResponse(text string) error

	// SendTTSStart 发送TTS开始消息
	SendTTSStart(format media.StreamFormat) error

	// SendTTSEnd 发送TTS结束消息
	SendTTSEnd() error

	// SendWelcome 发送Welcome消息（xiaozhi协议）
	SendWelcome(audioFormat string, sampleRate, channels int, features map[string]interface{}) (string, error)

	// Close 关闭写入器
	Close() error
}

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	// HandleError 处理错误
	HandleError(err error, service string) error

	// IsFatal 判断是否是致命错误
	IsFatal(err error) bool
}

// StateManager 状态管理器接口
type StateManager interface {
	// SetProcessing 设置处理状态
	SetProcessing(processing bool)

	// IsProcessing 检查是否正在处理
	IsProcessing() bool

	// SetTTSPlaying 设置TTS播放状态
	SetTTSPlaying(playing bool)

	// IsTTSPlaying 检查TTS是否正在播放
	IsTTSPlaying() bool

	// SetFatalError 设置致命错误状态
	SetFatalError(fatal bool)

	// IsFatalError 检查是否有致命错误
	IsFatalError() bool

	// UpdateASRText 更新ASR文本并返回增量文本
	UpdateASRText(text string, isLast bool) string

	// Clear 清空状态
	Clear()
}

// AudioManager 音频管理器接口（解决TTS冲突）
type AudioManager interface {
	// ProcessInputAudio 处理输入音频（智能过滤TTS回音）
	ProcessInputAudio(data []byte, ttsPlaying bool) ([]byte, bool)

	// RecordTTSOutput 记录TTS输出音频（用于回声消除）
	RecordTTSOutput(data []byte)

	// Clear 清空状态
	Clear()
}

// ServiceFactory 服务工厂接口
type ServiceFactory interface {
	// CreateASR 创建ASR服务
	CreateASR(credential *models.UserCredential, language string, sampleRate, channels int) (recognizer.TranscribeService, error)

	// CreateTTS 创建TTS服务
	CreateTTS(credential *models.UserCredential, speaker string, sampleRate, channels int) (synthesizer.SynthesisService, error)

	// CreateLLM 创建LLM服务
	CreateLLM(ctx context.Context, credential *models.UserCredential, systemPrompt string) (llm.LLMProvider, error)
}
