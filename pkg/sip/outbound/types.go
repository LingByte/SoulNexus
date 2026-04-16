package outbound

import (
	"context"
	"time"

	sipSession "github.com/LingByte/SoulNexus/pkg/sip/session"
)

// Scenario classifies why an outbound leg exists. Extensible without changing core SIP types.
type Scenario string

const (
	// ScenarioCampaign is a proactive outbound call (manual trigger or job queue) with an optional script.
	ScenarioCampaign Scenario = "campaign"
	// ScenarioTransferAgent is the agent leg after inbound user requests human support.
	ScenarioTransferAgent Scenario = "transfer_agent"
	// ScenarioCallback is a scheduled return call (same runtime as campaign, distinct for analytics).
	ScenarioCallback Scenario = "callback"
)

// DialTarget is a minimal description of where to send INVITE.
type DialTarget struct {
	// WebSeat is true when SIP_TRANSFER_NUMBER=web (or resolver): transfer goes to a browser
	// WebRTC agent, not a SIP INVITE to user "web".
	WebSeat bool
	// RequestURI is the SIP request URI, e.g. sip:+8613800138000@carrier.example;user=phone
	RequestURI string
	// SignalingAddr is the UDP address of the next SIP hop (proxy or UAS).
	SignalingAddr string // host:port
	// Optional From/Contact user part (e.g. ACD pool sipCallerId). Empty → Dial uses Manager/env default.
	CallerUser string
	// Optional quoted display-name in From when CallerUser is set (or alone if DialRequest sets caller).
	CallerDisplayName string
}

// DialRequest is one outbound attempt.
type DialRequest struct {
	Scenario Scenario
	Target   DialTarget

	// ScriptID optional reference for campaign runner (DB/job id).
	ScriptID string

	// CorrelationID ties this leg to CRM, inbound Call-ID, etc.
	CorrelationID string

	// MediaProfile selects which media/AI hooks run after connect (see MediaProfile).
	MediaProfile MediaProfile

	// CallerUser / CallerDisplayName override Target.* and Manager defaults when CallerUser non-empty.
	CallerUser        string
	CallerDisplayName string
}

// MediaProfile selects post-connect behavior on the established CallSession.
type MediaProfile string

const (
	// MediaProfileAI attaches the same ASR→LLM→TTS pipeline as inbound (env-driven).
	MediaProfileAI MediaProfile = "ai_voice"
	// MediaProfileScript runs a scripted IVR-style flow (prompts, DTMF) — orchestration TBD.
	MediaProfileScript MediaProfile = "script"
	// MediaProfileTransferBridge hands RTP to StartTransferBridge after ACK (raw G.711 relay or PCM transcode).
	MediaProfileTransferBridge MediaProfile = "transfer_bridge"
	// MediaProfileNone only brings RTP up (testing or custom hooks via callback).
	MediaProfileNone MediaProfile = "none"
)

// EstablishedLeg is passed to script/transfer hooks after 200 OK + ACK.
type EstablishedLeg struct {
	CallID        string
	Scenario      Scenario
	CorrelationID string
	Session       *sipSession.CallSession
	CreatedAt     time.Time

	// SIP headers / signaling (for analytics / DB persistence on outbound legs).
	FromHeader          string
	ToHeader            string
	RemoteSignalingAddr string
	CSeqInvite          string
}

// MediaAttachFunc wires ASR/LLM/TTS or other processors after RTP is live.
// Typically set to conversation.AttachVoicePipeline for MediaProfileAI.
type MediaAttachFunc func(ctx context.Context, cs *sipSession.CallSession) error

const (
	DialEventInvited     = "invited"
	DialEventProvisional = "provisional"
	DialEventEstablished = "established"
	DialEventFailed      = "failed"
)

// DialEvent streams lightweight dial lifecycle transitions for queue/observability.
type DialEvent struct {
	CallID        string
	CorrelationID string
	Scenario      Scenario
	MediaProfile  MediaProfile
	State         string
	StatusCode    int
	Reason        string
	At            time.Time
}
