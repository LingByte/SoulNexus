package tts

import "errors"

var (
	// ErrTTSServiceRequired is returned when TTSService is not provided.
	ErrTTSServiceRequired = errors.New("TTS service is required")

	// ErrSendCallbackRequired is returned when SendCallback is not provided.
	ErrSendCallbackRequired = errors.New("send callback is required")

	// ErrPipelineNotStarted is returned when pipeline operations are called before Start.
	ErrPipelineNotStarted = errors.New("pipeline not started")

	// ErrInvalidDataType is returned when data type is invalid.
	ErrInvalidDataType = errors.New("invalid data type")

	// ErrEmptyText is returned when text is empty.
	ErrEmptyText = errors.New("empty text")
)
