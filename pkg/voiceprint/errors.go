package voiceprint

import (
	"fmt"
)

// 声纹识别相关错误定义
var (
	ErrServiceDisabled    = fmt.Errorf("voiceprint service is disabled")
	ErrInvalidAudioFormat = fmt.Errorf("invalid audio format, only WAV is supported")
	ErrAudioTooShort      = fmt.Errorf("audio duration is too short")
	ErrAudioTooLong       = fmt.Errorf("audio duration is too long")
	ErrSpeakerNotFound    = fmt.Errorf("speaker not found")
	ErrSpeakerExists      = fmt.Errorf("speaker already exists")
	ErrLowSimilarity      = fmt.Errorf("similarity score too low")
	ErrServiceUnavailable = fmt.Errorf("voiceprint service unavailable")
	ErrInvalidResponse    = fmt.Errorf("invalid response from voiceprint service")
	ErrTimeout            = fmt.Errorf("voiceprint service timeout")
	ErrTooManyCandidates  = fmt.Errorf("too many candidate speakers")
)

// VoiceprintError 声纹识别错误类型
type VoiceprintError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *VoiceprintError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// 错误构造函数
func ErrInvalidConfig(details string) error {
	return &VoiceprintError{
		Code:    "INVALID_CONFIG",
		Message: "Invalid voiceprint configuration",
		Details: details,
	}
}

func ErrAPIRequest(details string) error {
	return &VoiceprintError{
		Code:    "API_REQUEST_FAILED",
		Message: "Voiceprint API request failed",
		Details: details,
	}
}

func ErrRegistrationFailed(speakerID, details string) error {
	return &VoiceprintError{
		Code:    "REGISTRATION_FAILED",
		Message: fmt.Sprintf("Failed to register voiceprint for speaker: %s", speakerID),
		Details: details,
	}
}

func ErrIdentificationFailed(details string) error {
	return &VoiceprintError{
		Code:    "IDENTIFICATION_FAILED",
		Message: "Failed to identify voiceprint",
		Details: details,
	}
}

func ErrDeletionFailed(speakerID, details string) error {
	return &VoiceprintError{
		Code:    "DELETION_FAILED",
		Message: fmt.Sprintf("Failed to delete voiceprint for speaker: %s", speakerID),
		Details: details,
	}
}

// IsVoiceprintError 检查是否为声纹识别错误
func IsVoiceprintError(err error) bool {
	_, ok := err.(*VoiceprintError)
	return ok
}

// GetErrorCode 获取错误代码
func GetErrorCode(err error) string {
	if vpErr, ok := err.(*VoiceprintError); ok {
		return vpErr.Code
	}
	return "UNKNOWN_ERROR"
}
