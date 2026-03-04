package constants

const (
	MessageTypeHello  = "hello"
	MessageTypeListen = "listen"
	MessageTypeAbort  = "abort"
	MessageTypePing   = "ping"
)

const (
	COMPONENT_OPUS_DECODE = "opus_decode"
	COMPONENT_FILTER_WORD = "filter_word"
	COMPONENT_VAD         = "vad"
	COMPONENT_INPUT       = "input"
)

const (
	DefaultTemperature          = 0.7
	DefaultLLMModel             = "deepseek-v3.1" // 默认LLM模型
	DefaultVADThreshold         = 0.8             // VAD 阈值（SileroVAD 概率值 0-1，0.8 = 80% 置信度）
	DefaultEnabledVAD           = false           // 禁用 VAD（Barge-in），因为 AEC 还不够好，容易误触发
	DefaultVADConsecutiveFrames = 30              // 提升到 30（600ms，需要 600ms 持续语音才能打断）
	DefaultMaxLLMToken          = 60              // 减少到 60，强制 AI 只说一两句话
)

const HARDWARE_SESSION_PREFIX = "hardware_session_"
const HARDWARE_WRITER_PREFIX = "hardware_writer_"

// LLMSystemPromptTemplate LLM Hardware Prompt Template
const LLMSystemPromptTemplate = "你是一个简洁的语音助手。助手信息：%s\n\n核心规则：\n1. 回复简洁并且不超过%d个token\n2. 纯文本，无Markdown、emoji、特殊符号\n3. 口语化，适合语音播报\n4. 再见/拜拜时调用goodbye工具，切换声音时调用switch_speaker工具\n5. 当用户问'我是谁'、'你认识我吗'、'我叫什么'等身份相关问题时，立即调用voiceprint_identify工具的'identify'操作进行声纹识别"
