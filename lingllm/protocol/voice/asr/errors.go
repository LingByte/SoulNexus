package asr

import "errors"

// Common errors
var (
	ErrEmptyInputStages = errors.New("asr: input stages cannot be empty")
	ErrNilData          = errors.New("asr: data is nil")
	ErrInvalidDataType  = errors.New("asr: invalid data type")
	ErrPipelineClosed   = errors.New("asr: pipeline already closed")
	ErrNilEngine        = errors.New("asr: nil engine")
)
