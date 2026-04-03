package models

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
)

// SIPCall records one SIP call lifecycle (INVITE -> ACK -> BYE).
type SIPCall struct {
	BaseModel

	CallID string `json:"callId" gorm:"size:128;uniqueIndex;not null"`

	FromHeader string `json:"fromHeader" gorm:"type:text"`
	ToHeader   string `json:"toHeader" gorm:"type:text"`
	CSeqInvite string `json:"cseqInvite" gorm:"size:64"`

	RemoteAddr string `json:"remoteAddr" gorm:"size:128;index"` // ip:port of SIP signaling peer

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
}

func (SIPCall) TableName() string {
	return constants.SIP_CALL_TABLE_NAME
}

