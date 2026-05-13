package server

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"net"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/rtp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// IncomingCall carries the inbound INVITE for InviteHandler.OnIncomingCall.
//
// The business layer inspects these fields (typically From/To URIs or SDP
// codec list) and decides whether to accept, reject, or forward the call.
// Nothing in this struct is persisted by the SIP server; persistence is
// the CallStore's job.
type IncomingCall struct {
	CallID              string
	FromURI             string
	ToURI               string
	RemoteSignalingAddr *net.UDPAddr
	SDP                 *sdp.Info
	RawMessage          *stack.Message

	// RTPSession is the server-allocated RTP socket whose port has already
	// been (or will be) advertised in the 200 OK SDP answer. Business
	// handlers MUST pass this same session to session.NewMediaLeg — building
	// a MediaLeg on a fresh siprtp.NewSession would cause the SDP to point
	// at a socket nobody is listening on, and inbound RTP would be lost.
	RTPSession *rtp.Session
}

// Decision tells the SIP server how to answer an INVITE.
//
// Accept=false (the default) makes the server send the StatusCode/ReasonPhrase
// back as a final response (defaults to 480 Temporarily Unavailable when
// StatusCode is 0).
//
// Accept=true requires MediaLeg to be non-nil. The server sends 200 OK with
// the leg's local SDP, wires in your OnTerminate hook for BYE teardown, and
// starts the media pipeline on ACK.
type Decision struct {
	Accept       bool
	StatusCode   int
	ReasonPhrase string

	// MediaLeg is the fully-configured audio leg for this call. Required
	// when Accept=true. Build it with session.NewMediaLeg — the business
	// layer gets to decide codec preferences, RTP taps, and filters.
	MediaLeg *session.MediaLeg

	// OnTerminate is invoked once when the call is torn down (BYE, CANCEL
	// after 200 OK, or session timeout). Use it to stop ASR/TTS/LLM workers
	// tied to this call. Optional.
	OnTerminate func(reason string)
}

// InviteHandler is the primary business-layer entry point for the SIP server.
//
// Register via SIPServer.SetInviteHandler. If nil, all inbound INVITEs receive
// 480 Temporarily Unavailable — useful for a pure registrar-only deployment.
type InviteHandler interface {
	OnIncomingCall(ctx context.Context, inv *IncomingCall) (Decision, error)
}

// DTMFSink receives DTMF events from both SIP INFO bodies (application/dtmf-relay)
// and RFC 2833 telephone-event RTP payloads (routed from MediaLeg's decoder).
type DTMFSink interface {
	// OnDTMF fires for each completed DTMF keypress. digit is one of
	// "0"-"9", "*", "#", "A"-"D". end is true when the RFC 2833 end bit
	// was set (keypress released); SIP INFO deliveries always have end=true.
	OnDTMF(callID string, digit string, end bool)
}

// TransferHandler handles SIP REFER requests (call transfer / attended transfer).
//
// notify is a callback the handler calls to send NOTIFY progress updates back
// to the caller (frag is the sipfrag body, subState is the Subscription-State
// header value like "active;expires=60" or "terminated;reason=noresource").
type TransferHandler interface {
	OnRefer(ctx context.Context, callID, referTo string, notify func(frag, subState string))
}

// CallLifecycleObserver observes call teardown in a cross-cutting way.
//
// It is called for BYE, CANCEL after 200 OK, session timeout, and any other
// path where the server proactively cleans a call's state. Use this for
// metrics, audit trails, or resource cleanup that must happen regardless of
// whether the business InviteHandler is present.
//
// OnCallPreHangup lets the observer claim the hangup: return true to signal
// "I've already torn down everything for this call, don't send BYE". Useful
// for WebSeat/transfer bridges whose teardown is owned elsewhere.
type CallLifecycleObserver interface {
	OnCallPreHangup(callID string) (handled bool)
	OnCallCleanup(callID string)
}

// -------- Config --------------------------------------------------------

// Config is the full runtime configuration. Populate once at startup
// (typically in cmd/voiceserver/main.go from env/flags) and pass to New;
// SIPServer reads from the struct thereafter. No package-level env reads
// anywhere — every tunable is here.
//
// The zero Config yields a UDP-only server. Host/Port default to
// "127.0.0.1:5060"; LocalIP defaults to "127.0.0.1".
type Config struct {
	// Host is the SIP UDP listen address; empty = 0.0.0.0.
	Host string

	// Port is the SIP UDP listen port; 0 = ephemeral.
	Port int

	// LocalIP is used in SDP response c=IN IP4 <LocalIP>. Empty = "127.0.0.1".
	LocalIP string

	// OnSIPResponse is optional: receives SIP responses on the listen socket
	// (useful when the same socket is used for outbound legs).
	OnSIPResponse func(resp *stack.Message, addr *net.UDPAddr)

	// SIPTCPListen is the TCP listen address. Empty = no TCP transport.
	SIPTCPListen string

	// SIPTLSListen is the TLS listen address. Empty = no TLS transport.
	// Requires SIPTLSCertFile + SIPTLSKeyFile.
	SIPTLSListen   string
	SIPTLSCertFile string
	SIPTLSKeyFile  string

	// RTP port allocation. RTPFixedPort overrides the range when > 0.
	// Otherwise RTPPortStart..RTPPortEnd are rotated per new session.
	// If all zero, the OS picks an ephemeral port per call.
	RTPFixedPort int
	RTPPortStart int
	RTPPortEnd   int

	// INVITE gating.
	InviteAllowCIDRs []string // empty = no source CIDR filtering
	InviteRatePerSec float64  // 0 = unlimited
	InviteRateBurst  int      // 0 = default

	// INVITE 180/100rel/early-media behaviour.
	InviteRingbackMS    int  // 0 = no 180 Ringing
	InviteSend180       bool // default true
	InviteForce100rel   bool // default false
	InviteEarlyMediaSDP bool // default false

	// Digest auth (REGISTER challenge). Empty realm disables challenge.
	DigestRealm    string
	DigestUser     string
	DigestPassword string

	// RegisterStaticPassword, when set, requires REGISTER requests to carry
	// the X-SIP-Register-Password header with this exact value (constant-time
	// compared). Independent from Digest auth above. Empty disables the check.
	RegisterStaticPassword string
}
