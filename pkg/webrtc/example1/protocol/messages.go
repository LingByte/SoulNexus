package protocol

import (
	"encoding/json"
	"time"
)

// MessageType 消息类型
type MessageType string

const (
	// 连接管理
	TypeInit       MessageType = "init"
	TypeConnected  MessageType = "connected"
	TypeDisconnect MessageType = "disconnect"

	// WebRTC信令
	TypeOffer        MessageType = "offer"
	TypeAnswer       MessageType = "answer"
	TypeICECandidate MessageType = "ice_candidate"

	// 文本消息
	TypeTextMessage  MessageType = "text_message"
	TypeTextResponse MessageType = "text_response"

	// 语音识别
	TypeASRStart   MessageType = "asr_start"
	TypeASRResult  MessageType = "asr_result"
	TypeASRInterim MessageType = "asr_interim"
	TypeASRStop    MessageType = "asr_stop"

	// 语音合成
	TypeTTSRequest  MessageType = "tts_request"
	TypeTTSStart    MessageType = "tts_start"
	TypeTTSComplete MessageType = "tts_complete"

	// 控制消息
	TypeReady MessageType = "ready"
	TypePing  MessageType = "ping"
	TypePong  MessageType = "pong"
	TypeError MessageType = "error"
)

// Message 基础消息结构
type Message struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorData  `json:"error,omitempty"`
}

// ErrorData 错误数据
type ErrorData struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// NewMessage 创建新消息
func NewMessage(msgType MessageType, sessionID string, data interface{}) *Message {
	return &Message{
		Type:      msgType,
		SessionID: sessionID,
		Timestamp: time.Now().UnixMilli(),
		Data:      data,
	}
}

// NewErrorMessage 创建错误消息
func NewErrorMessage(sessionID string, code, message string, details interface{}) *Message {
	return &Message{
		Type:      TypeError,
		SessionID: sessionID,
		Timestamp: time.Now().UnixMilli(),
		Error: &ErrorData{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// ToJSON 转换为JSON
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON 从JSON解析
func (m *Message) FromJSON(data []byte) error {
	return json.Unmarshal(data, m)
}

// InitData 初始化消息数据
type InitData struct {
	ServerVersion    string   `json:"server_version"`
	SupportedCodecs  []string `json:"supported_codecs"`
	MaxAudioDuration int      `json:"max_audio_duration"` // 毫秒
}

// ConnectedData 连接确认数据
type ConnectedData struct {
	ConnectionState string `json:"connection_state"`
	AudioCodec      string `json:"audio_codec"`
	SampleRate      int    `json:"sample_rate"`
}

// DisconnectData 断开连接数据
type DisconnectData struct {
	Reason string `json:"reason"` // user_requested, timeout, error
}

// WebRTCData WebRTC信令数据
type WebRTCData struct {
	SDP        string   `json:"sdp"`
	Candidates []string `json:"candidates"`
}

// ICECandidateData ICE候选者数据
type ICECandidateData struct {
	Candidate     string `json:"candidate"`
	SDPMid        string `json:"sdp_mid,omitempty"`
	SDPMLineIndex int    `json:"sdp_mline_index,omitempty"`
}

// TextMessageData 文本消息数据
type TextMessageData struct {
	MessageID string `json:"message_id"`
	Text      string `json:"text"`
	Language  string `json:"language"`
}

// TextResponseData 文本回复数据
type TextResponseData struct {
	MessageID  string  `json:"message_id"`
	ResponseID string  `json:"response_id"`
	Text       string  `json:"text"`
	Language   string  `json:"language"`
	Confidence float64 `json:"confidence"`
}

// ASRStartData 开始语音识别数据
type ASRStartData struct {
	Language   string `json:"language"`
	Format     string `json:"format"`
	SampleRate int    `json:"sample_rate"`
}

// ASRResultData 语音识别结果数据
type ASRResultData struct {
	ResultID   string  `json:"result_id"`
	Text       string  `json:"text"`
	IsFinal    bool    `json:"is_final"`
	Confidence float64 `json:"confidence"`
	Language   string  `json:"language"`
	StartTime  int64   `json:"start_time"` // 毫秒
	EndTime    int64   `json:"end_time"`   // 毫秒
}

// ASRStopData 停止语音识别数据
type ASRStopData struct{}

// TTSRequestData TTS请求数据
type TTSRequestData struct {
	RequestID string  `json:"request_id"`
	Text      string  `json:"text"`
	Voice     string  `json:"voice"`
	Speed     float64 `json:"speed"`
	Pitch     float64 `json:"pitch"`
}

// TTSStartData TTS开始数据
type TTSStartData struct {
	RequestID   string `json:"request_id"`
	AudioFormat string `json:"audio_format"`
	SampleRate  int    `json:"sample_rate"`
}

// TTSCompleteData TTS完成数据
type TTSCompleteData struct {
	RequestID  string `json:"request_id"`
	DurationMs int    `json:"duration_ms"`
}

// ReadyData 准备就绪数据
type ReadyData struct {
	Direction string `json:"direction"` // send, receive, both
}

// PongData 心跳响应数据
type PongData struct {
	ServerTime int64 `json:"server_time"`
}

// ErrorCode 错误代码
type ErrorCode string

const (
	ErrConnectionFailed  ErrorCode = "ERR_CONNECTION_FAILED"
	ErrInvalidMessage    ErrorCode = "ERR_INVALID_MESSAGE"
	ErrSessionNotFound   ErrorCode = "ERR_SESSION_NOT_FOUND"
	ErrWebRTCFailed      ErrorCode = "ERR_WEBRTC_FAILED"
	ErrAudioEncodeFailed ErrorCode = "ERR_AUDIO_ENCODE_FAILED"
	ErrASRFailed         ErrorCode = "ERR_ASR_FAILED"
	ErrTTSFailed         ErrorCode = "ERR_TTS_FAILED"
	ErrTimeout           ErrorCode = "ERR_TIMEOUT"
	ErrRateLimit         ErrorCode = "ERR_RATE_LIMIT"
)
