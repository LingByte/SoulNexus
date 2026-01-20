package hardware

import "time"

// 默认配置值
const (
	DefaultASRConnectionDelay = 500 * time.Millisecond // ASR连接延迟
	DefaultLLMModel           = "deepseek-v3.1"        // 默认LLM模型
)

// WebSocket消息类型（保持与hardware协议一致）
const (
	MessageTypeConnected      = "connected"
	MessageTypeError          = "error"
	MessageTypeASRResult      = "asr_result"
	MessageTypeLLMResponse    = "llm_response"
	MessageTypeTTSStart       = "tts_start"
	MessageTypeTTSEnd         = "tts_end"
	MessageTypeNewSession     = "new_session"
	MessageTypeSessionCleared = "session_cleared"
	MessageTypePing           = "ping"
	MessageTypePong           = "pong"
)

// 音频处理常量
const (
	// AudioBufferSize 音频缓冲区大小
	AudioBufferSize = 100
	// TTSEchoSuppressionWindow TTS回音抑制窗口（毫秒）
	TTSEchoSuppressionWindow = 2000
	// AudioEnergyThreshold 音频能量阈值（用于检测有效音频）
	AudioEnergyThreshold = 1000
	// MaxMessageHistory 最大消息历史数量
	MaxMessageHistory = 100
)
