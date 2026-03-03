package voiceprint

import (
	"time"
)

// RegisterRequest 声纹注册请求
type RegisterRequest struct {
	SpeakerID   string                 `json:"speaker_id" validate:"required"`
	AssistantID string                 `json:"assistant_id" validate:"required"`
	AudioData   []byte                 `json:"-"`
	AudioFormat string                 `json:"audio_format,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RegisterResponse 声纹注册响应
type RegisterResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"msg"`
	SpeakerID string    `json:"speaker_id,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// IdentifyRequest 声纹识别请求
type IdentifyRequest struct {
	CandidateIDs []string `json:"candidate_ids" validate:"required,min=1"`
	AssistantID  string   `json:"assistant_id" validate:"required"`
	AudioData    []byte   `json:"-"`
	AudioFormat  string   `json:"audio_format,omitempty"`
	Threshold    float64  `json:"threshold,omitempty"`
	MaxResults   int      `json:"max_results,omitempty"`
}

// IdentifyResponse 声纹识别响应
type IdentifyResponse struct {
	SpeakerID  string    `json:"speaker_id"`
	Score      float64   `json:"score"`
	Timestamp  time.Time `json:"timestamp,omitempty"`
	Confidence string    `json:"confidence,omitempty"`
}

// DeleteRequest 声纹删除请求
type DeleteRequest struct {
	SpeakerID   string `json:"speaker_id" validate:"required"`
	AssistantID string `json:"assistant_id,omitempty"`
}

// DeleteResponse 声纹删除响应
type DeleteResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"msg"`
	SpeakerID string    `json:"speaker_id,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status           string    `json:"status"`
	TotalVoiceprints int       `json:"total_voiceprints"`
	Timestamp        time.Time `json:"timestamp,omitempty"`
}

// SpeakerInfo 说话人信息
type SpeakerInfo struct {
	SpeakerID    string                 `json:"speaker_id"`
	RegisterTime time.Time              `json:"register_time"`
	UpdateTime   time.Time              `json:"update_time"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// IdentifyResult 识别结果（扩展版本）
type IdentifyResult struct {
	SpeakerID   string        `json:"speaker_id"`
	Score       float64       `json:"score"`
	Confidence  string        `json:"confidence"`
	Threshold   float64       `json:"threshold"`
	IsMatch     bool          `json:"is_match"`
	ProcessTime time.Duration `json:"process_time"`
	Timestamp   time.Time     `json:"timestamp"`
}

// BatchRegisterRequest 批量注册请求
type BatchRegisterRequest struct {
	Speakers []RegisterRequest `json:"speakers" validate:"required,min=1"`
}

// BatchRegisterResponse 批量注册响应
type BatchRegisterResponse struct {
	Success   int                `json:"success"`
	Failed    int                `json:"failed"`
	Total     int                `json:"total"`
	Results   []RegisterResponse `json:"results"`
	Timestamp time.Time          `json:"timestamp"`
}

// BatchIdentifyRequest 批量识别请求
type BatchIdentifyRequest struct {
	Requests []IdentifyRequest `json:"requests" validate:"required,min=1"`
}

// BatchIdentifyResponse 批量识别响应
type BatchIdentifyResponse struct {
	Success   int                `json:"success"`
	Failed    int                `json:"failed"`
	Total     int                `json:"total"`
	Results   []IdentifyResponse `json:"results"`
	Timestamp time.Time          `json:"timestamp"`
}

// Statistics 统计信息
type Statistics struct {
	TotalSpeakers        int       `json:"total_speakers"`
	TotalIdentifications int       `json:"total_identifications"`
	SuccessRate          float64   `json:"success_rate"`
	AverageScore         float64   `json:"average_score"`
	LastActivity         time.Time `json:"last_activity"`
}

// GetConfidenceLevel 根据相似度分数获取置信度等级
func (r *IdentifyResult) GetConfidenceLevel() string {
	switch {
	case r.Score >= 0.8:
		return "very_high"
	case r.Score >= 0.6:
		return "high"
	case r.Score >= 0.4:
		return "medium"
	case r.Score >= 0.2:
		return "low"
	default:
		return "very_low"
	}
}

// IsHighConfidence 判断是否为高置信度
func (r *IdentifyResult) IsHighConfidence() bool {
	return r.Score >= 0.6
}

// IsValidMatch 判断是否为有效匹配
func (r *IdentifyResult) IsValidMatch() bool {
	return r.Score >= r.Threshold
}
