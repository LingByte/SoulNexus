package models

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
)

// SIPSession records conversational sessions within a call.
//
// Think of it as "dialog/turn session" metadata, not RTP session.
// It can later be linked to ASR/TTS metrics, transcripts, summaries, etc.
type SIPSession struct {
	BaseModel

	CallID string `json:"callId" gorm:"size:128;index;not null"`

	// Conversation identity
	DialogID string `json:"dialogId" gorm:"size:128;index"` // optional: per-turn / per-dialog id

	// Text artifacts
	ASRText string `json:"asrText" gorm:"type:text"`
	LLMText string `json:"llmText" gorm:"type:text"`

	// Timing
	StartAt *time.Time `json:"startAt" gorm:"index"`
	EndAt   *time.Time `json:"endAt" gorm:"index"`

	// Providers
	ASRProvider string `json:"asrProvider" gorm:"size:64;index"`
	TTSProvider string `json:"ttsProvider" gorm:"size:64;index"`
	LLMModel    string `json:"llmModel" gorm:"size:128;index"`
}

func (SIPSession) TableName() string {
	return constants.SIP_SESSION_TABLE_NAME
}

