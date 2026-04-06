package models

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/datatypes"
)

// SIP call end_status (set on BYE).
const (
	SIPCallEndUnknown = ""

	SIPCallEndCompletedRemote = "completed_remote"
	SIPCallEndCompletedLocal  = "completed_local"

	SIPCallEndAfterTransferRemote = "after_transfer_remote"
	SIPCallEndAfterTransferLocal  = "after_transfer_local"
)

// SIPCallDialogTurn is one ASR→LLM exchange stored in SIPCall.Turns (JSON array).
type SIPCallDialogTurn struct {
	ASRText     string    `json:"asrText"`
	LLMText     string    `json:"llmText"`
	ASRProvider string    `json:"asrProvider,omitempty"`
	TTSProvider string    `json:"ttsProvider,omitempty"`
	LLMModel    string    `json:"llmModel,omitempty"`
	At          time.Time `json:"at"`
}

// SIPCall records one SIP call lifecycle (INVITE -> ACK -> BYE) and optional AI dialog turns in JSON.
type SIPCall struct {
	BaseModel

	CallID string `json:"callId" gorm:"size:128;uniqueIndex;not null"`

	FromHeader string `json:"fromHeader" gorm:"type:text"`
	ToHeader   string `json:"toHeader" gorm:"type:text"`
	CSeqInvite string `json:"cseqInvite" gorm:"size:64"`

	RemoteAddr string `json:"remoteAddr" gorm:"size:128;index"` // ip:port of SIP signaling peer

	Direction string `json:"direction" gorm:"size:16;index"` // inbound | outbound

	// RTP endpoints
	RemoteRTPAddr string `json:"remoteRtpAddr" gorm:"size:128;index"` // ip:port from SDP
	LocalRTPAddr  string `json:"localRtpAddr" gorm:"size:128;index"`  // ip:port allocated locally

	// Negotiated codec
	PayloadType uint8  `json:"payloadType" gorm:"index"`
	Codec       string `json:"codec" gorm:"size:32;index"` // pcmu/pcma/opus/g722...
	ClockRate   int    `json:"clockRate"`

	// State
	State         string     `json:"state" gorm:"size:32;index"` // init, ringing, established, ended, failed
	InviteAt      *time.Time `json:"inviteAt" gorm:"index"`
	AckAt         *time.Time `json:"ackAt" gorm:"index"`
	ByeAt         *time.Time `json:"byeAt" gorm:"index"`
	EndedAt       *time.Time `json:"endedAt" gorm:"index"`
	FailureReason string     `json:"failureReason" gorm:"type:text"`

	RecordingURL string `json:"recordingUrl" gorm:"size:1024"`
	DurationSec  int    `json:"durationSec" gorm:"default:0"`

	// EndStatus is one of SIPCallEnd* constants.
	EndStatus string `json:"endStatus" gorm:"size:64;index"`

	// AI dialog (same row as call)
	Turns       datatypes.JSON `json:"turns" gorm:"type:json"`
	TurnCount   int            `json:"turnCount" gorm:"default:0"`
	FirstTurnAt *time.Time     `json:"firstTurnAt"`
	LastTurnAt  *time.Time     `json:"lastTurnAt"`

	HadSIPTransfer bool `json:"hadSipTransfer" gorm:"column:had_sip_transfer;default:0"`
	HadWebSeat     bool `json:"hadWebSeat" gorm:"column:had_web_seat;default:0"`
}

func (SIPCall) TableName() string {
	return constants.SIP_CALL_TABLE_NAME
}
