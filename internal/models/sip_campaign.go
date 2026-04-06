package models

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/datatypes"
)

const (
	SIPCampaignStatusDraft   = "draft"
	SIPCampaignStatusRunning = "running"
	SIPCampaignStatusPaused  = "paused"
	SIPCampaignStatusDone    = "done"

	SIPCampaignContactReady      = "ready"
	SIPCampaignContactDialing    = "dialing"
	SIPCampaignContactAnswered   = "answered"
	SIPCampaignContactFailed     = "failed"
	SIPCampaignContactRetrying   = "retrying"
	SIPCampaignContactExhausted  = "exhausted"
	SIPCampaignContactSuppressed = "suppressed"
)

// SIPCampaign stores one outbound campaign configuration.
type SIPCampaign struct {
	BaseModel

	Name string `json:"name" gorm:"size:128;index;not null"`

	Status          string `json:"status" gorm:"size:24;index;not null;default:draft"`
	Scenario        string `json:"scenario" gorm:"size:24;index;not null;default:campaign"`
	MediaProfile    string `json:"mediaProfile" gorm:"size:24;index;not null;default:ai_voice"`
	ScriptID        string `json:"scriptId" gorm:"size:128;index"`
	ScriptVersion   string `json:"scriptVersion" gorm:"size:64"`
	ScriptSpec      datatypes.JSON `json:"scriptSpec" gorm:"type:json"`
	SystemPrompt    string `json:"systemPrompt" gorm:"type:text"`
	OpeningMessage  string `json:"openingMessage" gorm:"type:text"`
	ClosingMessage  string `json:"closingMessage" gorm:"type:text"`
	RetrySchedule   string `json:"retrySchedule" gorm:"size:256;default:5m,30m,2h"`
	MaxAttempts     int    `json:"maxAttempts" gorm:"default:3"`
	MaxCallDuration int    `json:"maxCallDuration" gorm:"default:180"` // seconds
	OutboundHost    string `json:"outboundHost" gorm:"size:128"`
	OutboundPort    int    `json:"outboundPort" gorm:"default:5060"`
	SignalingAddr   string `json:"signalingAddr" gorm:"size:128"`
	RequestURIFmt   string `json:"requestUriFmt" gorm:"size:256"` // e.g. sip:%s@gw.local:5060

	TaskConcurrency   int       `json:"taskConcurrency" gorm:"default:5"`
	GlobalConcurrency int       `json:"globalConcurrency" gorm:"default:20"`
	AllowedTimeStart  string    `json:"allowedTimeStart" gorm:"size:8;default:09:00"`
	AllowedTimeEnd    string    `json:"allowedTimeEnd" gorm:"size:8;default:21:00"`
	Timezone          string    `json:"timezone" gorm:"size:64;default:Asia/Shanghai"`
	Metadata          datatypes.JSON `json:"metadata" gorm:"type:json"`

	StartedAt *time.Time `json:"startedAt" gorm:"index"`
	EndedAt   *time.Time `json:"endedAt" gorm:"index"`
}

func (SIPCampaign) TableName() string {
	return constants.SIP_CAMPAIGN_TABLE_NAME
}

// SIPCampaignContact is one callee in a campaign queue.
type SIPCampaignContact struct {
	BaseModel

	CampaignID uint   `json:"campaignId" gorm:"index;not null"`
	Phone      string `json:"phone" gorm:"size:64;index;not null"`
	RequestURI string `json:"requestUri" gorm:"size:256"`
	Display    string `json:"display" gorm:"size:128"`
	CallerUser string `json:"callerUser" gorm:"size:128"`
	CallerName string `json:"callerName" gorm:"size:128"`

	Status        string     `json:"status" gorm:"size:24;index;not null;default:ready"`
	Priority      int        `json:"priority" gorm:"index;default:0"`
	AttemptCount  int        `json:"attemptCount" gorm:"default:0"`
	MaxAttempts   int        `json:"maxAttempts" gorm:"default:3"`
	NextRunAt     *time.Time `json:"nextRunAt" gorm:"index"`
	LastDialAt    *time.Time `json:"lastDialAt" gorm:"index"`
	LastCallID    string     `json:"lastCallId" gorm:"size:128;index"`
	CorrelationID string     `json:"correlationId" gorm:"size:128;index"`
	FailureReason string     `json:"failureReason" gorm:"type:text"`
	Variables     datatypes.JSON `json:"variables" gorm:"type:json"`
}

func (SIPCampaignContact) TableName() string {
	return constants.SIP_CAMPAIGN_CONTACT_TABLE_NAME
}

// SIPCallAttempt keeps attempt-level dial lifecycle and retry decisions.
type SIPCallAttempt struct {
	BaseModel

	CampaignID uint `json:"campaignId" gorm:"index;not null"`
	ContactID  uint `json:"contactId" gorm:"index;not null"`

	AttemptNo     int       `json:"attemptNo" gorm:"index;not null"`
	CallID        string    `json:"callId" gorm:"size:128;index"`
	CorrelationID string    `json:"correlationId" gorm:"size:128;index"`
	State         string    `json:"state" gorm:"size:24;index;not null;default:created"` // created|dialing|answered|failed|retry_pending
	SIPStatusCode int       `json:"sipStatusCode" gorm:"index"`
	FailureReason string    `json:"failureReason" gorm:"type:text"`
	DialedAt      *time.Time `json:"dialedAt" gorm:"index"`
	AnsweredAt    *time.Time `json:"answeredAt" gorm:"index"`
	EndedAt       *time.Time `json:"endedAt" gorm:"index"`
	NextRetryAt   *time.Time `json:"nextRetryAt" gorm:"index"`
}

func (SIPCallAttempt) TableName() string {
	return constants.SIP_CALL_ATTEMPT_TABLE_NAME
}

// SIPScriptRun keeps per-call script step traces for audit.
type SIPScriptRun struct {
	BaseModel

	CampaignID    uint           `json:"campaignId" gorm:"index;not null"`
	ContactID     uint           `json:"contactId" gorm:"index"`
	AttemptID     uint           `json:"attemptId" gorm:"index"`
	CallID        string         `json:"callId" gorm:"size:128;index"`
	CorrelationID string         `json:"correlationId" gorm:"size:128;index"`
	ScriptID      string         `json:"scriptId" gorm:"size:128;index"`
	ScriptVersion string         `json:"scriptVersion" gorm:"size:64"`
	StepID        string         `json:"stepId" gorm:"size:128;index"`
	StepType      string         `json:"stepType" gorm:"size:32"`
	Result        string         `json:"result" gorm:"size:32"` // ok|timeout|retry|skipped|failed
	InputText     string         `json:"inputText" gorm:"type:text"`
	OutputText    string         `json:"outputText" gorm:"type:text"`
	Variables     datatypes.JSON `json:"variables" gorm:"type:json"`
}

func (SIPScriptRun) TableName() string {
	return constants.SIP_SCRIPT_RUN_TABLE_NAME
}

