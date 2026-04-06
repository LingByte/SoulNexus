package models

import (
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
)

// ACD pool route types (single table for SIP accounts and Web/WebRTC seats).
const (
	ACDPoolRouteTypeSIP = "sip"
	ACDPoolRouteTypeWeb = "web"
)

// SipSource applies when RouteType is SIP (empty for web rows).
const (
	ACDSipSourceInternal = "internal" // TargetValue = registered sip_users.username
	ACDSipSourceTrunk    = "trunk"    // TargetValue = external dial string (PSTN / carrier)
)

// WorkState is real-time seat eligibility for this row (agent UI, admin, or SIP/Web gateway updates it).
// cmd/sip blind transfer (DTMF) picks only rows with WorkState == ACDWorkStateAvailable (plus weight>0).
// Use ringing/busy/acw/break to take a row out of the next-offer set without deleting it.
const (
	ACDWorkStateOffline   = "offline"   // not signed in / not bound
	ACDWorkStateAvailable = "available" // idle, eligible for next offer
	ACDWorkStateRinging   = "ringing"   // offer in progress; do not assign another call
	ACDWorkStateBusy      = "busy"      // media connected / on call
	ACDWorkStateACW       = "acw"       // after-call work (wrap-up)
	ACDWorkStateBreak     = "break"     // rest / pause
)

// ACDPoolTarget is one row in the transfer routing table (acd_pool_targets) when cmd/sip uses a database.
// Selection: highest Weight first, then lowest id; only Weight>0 and WorkState==available.
// SIP rows: internal TargetValue = sip_users.username; trunk = dial string + trunk host fields.
// Web rows: TargetValue usually empty; WebSeat handoff when this row wins over SIP rows by Weight.
type ACDPoolTarget struct {
	BaseModel

	Name string `json:"name" gorm:"size:128"` // optional admin label

	// RouteType is ACDPoolRouteTypeSIP or ACDPoolRouteTypeWeb.
	RouteType string `json:"routeType" gorm:"size:16;not null;index"`

	// TargetValue: sip internal → sip_users.username; sip trunk → dial digits / URI; web → usually empty.
	TargetValue string `json:"targetValue" gorm:"size:256"`

	// SipSource: internal | trunk when RouteType is SIP; empty when web.
	SipSource string `json:"sipSource" gorm:"size:16;not null;default:'';index"`

	// Sip trunk only: next SIP hop for INVITE (Request-URI host:port + optional signaling override).
	// Internal / web rows leave these empty / port 0.
	SipTrunkHost          string `json:"sipTrunkHost" gorm:"size:128"`
	SipTrunkPort          int    `json:"sipTrunkPort" gorm:"not null;default:0"`
	SipTrunkSignalingAddr string `json:"sipTrunkSignalingAddr" gorm:"size:160"` // host:port, optional; default host:port from above

	// SIP only: optional outbound From user / display (like SIP_CALLER_ID); empty → cmd/sip uses global .env default.
	SipCallerID          string `json:"sipCallerId" gorm:"size:64"`
	SipCallerDisplayName string `json:"sipCallerDisplayName" gorm:"size:128"`

	// Weight: higher = higher priority when selecting; 0 = disabled (not eligible).
	Weight int `json:"weight" gorm:"not null;default:0;index"`

	// WorkState: see ACDWorkState*; default offline until sign-in or integration sets available.
	WorkState string `json:"workState" gorm:"size:24;not null;default:offline;index"`

	// WorkStateAt: optional last transition (ring timeouts, metrics).
	WorkStateAt *time.Time `json:"workStateAt"`
}

func (ACDPoolTarget) TableName() string {
	return constants.ACD_POOL_TARGET_TABLE_NAME
}
