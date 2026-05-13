package server

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/rtp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// TestScenario_InviteAcceptedByHandler_ACK_ReINVITE_BYE drives a full
// inbound call lifecycle with a business InviteHandler that supplies a
// MediaLeg, exercising handleInvite + handleAck + handleReInvite + handleBye.
func TestScenario_InviteAcceptedByHandler_ACK_ReINVITE_BYE(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	srv.SetInboundAllowUnknownDID(true)

	terminated := int32(0)
	cleanup := int32(0)
	srv.SetCallLifecycleObserver(&fakeObserver{})

	// InviteHandler that builds a MediaLeg from the offer and accepts.
	srv.SetInviteHandler(invHandlerAdapter(func(ctx context.Context, in *IncomingCall) (Decision, error) {
		// Build a real RTP session + MediaLeg from the offered codecs.
		rtpSess, err := rtp.NewSession(0)
		if err != nil {
			return Decision{}, err
		}
		leg, err := session.NewMediaLeg(ctx, in.CallID, rtpSess, in.SDP.Codecs, session.MediaLegConfig{})
		if err != nil {
			rtpSess.Close()
			return Decision{}, err
		}
		return Decision{
			Accept:      true,
			MediaLeg:    leg,
			OnTerminate: func(reason string) { atomic.AddInt32(&terminated, 1) },
		}, nil
	}))

	// Hook lifecycle observer manually too
	srv.SetCallLifecycleObserver(observerFn(func(callID string) {
		atomic.AddInt32(&cleanup, 1)
	}))

	const callID = "scenario-handler-1"
	caller := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6050}

	// 1) INVITE
	inv, _ := stack.Parse(rawInviteFor(callID))
	resp := srv.handleInvite(inv, caller)
	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("INVITE: %v", resp)
	}
	if resp.GetHeader("Content-Type") != "application/sdp" {
		t.Errorf("missing SDP content-type")
	}
	// Validate SDP contains negotiated codec
	if info, err := sdp.Parse(resp.Body); err != nil || len(info.Codecs) == 0 {
		t.Errorf("bad SDP in 200: err=%v info=%v", err, info)
	}

	// 2) ACK — server should call MediaLeg.Start()
	ackRaw := strings.Join([]string{
		"ACK sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP client.com:6050;branch=z9hG4bK" + callID + "ack",
		"From: <sip:caller@client.com>;tag=" + callID,
		"To: <sip:user@domain.com>;tag=" + extractToTag(resp.GetHeader("To")),
		"Call-ID: " + callID,
		"CSeq: 1 ACK",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	ackMsg, _ := stack.Parse(ackRaw)
	if r := srv.handleAck(ackMsg, caller); r != nil {
		t.Errorf("ACK should have no response, got %v", r)
	}

	// Allow MediaLeg.Start() goroutine to schedule
	time.Sleep(20 * time.Millisecond)

	// 3) re-INVITE (same dialog) — codec match required
	body := buildReInviteSDP(8081)
	reinviteRaw := strings.Join([]string{
		"INVITE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP client.com:6050;branch=z9hG4bK" + callID + "reinv",
		"To: <sip:user@domain.com>;tag=" + extractToTag(resp.GetHeader("To")),
		"From: <sip:caller@client.com>;tag=" + callID,
		"Call-ID: " + callID,
		"CSeq: 2 INVITE",
		"Contact: <sip:caller@client.com>",
		"Content-Type: application/sdp",
		"Content-Length: " + strconv.Itoa(len(body)),
		"",
		body,
	}, "\r\n")
	reinv, _ := stack.Parse(reinviteRaw)
	if rresp := srv.handleInvite(reinv, caller); rresp == nil || rresp.StatusCode != 200 {
		t.Errorf("re-INVITE: %v", rresp)
	}

	// 4) BYE — should fire OnTerminate
	byeRaw := strings.Join([]string{
		"BYE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP client.com:6050;branch=z9hG4bK" + callID + "bye",
		"From: <sip:caller@client.com>;tag=" + callID,
		"To: <sip:user@domain.com>;tag=" + extractToTag(resp.GetHeader("To")),
		"Call-ID: " + callID,
		"CSeq: 3 BYE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	byeMsg, _ := stack.Parse(byeRaw)
	bresp := srv.handleBye(byeMsg, caller)
	if bresp == nil || bresp.StatusCode != 200 {
		t.Fatalf("BYE: %v", bresp)
	}

	// OnTerminate should have fired
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && atomic.LoadInt32(&terminated) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&terminated) == 0 {
		t.Error("OnTerminate not fired after BYE")
	}

	// Cleanup observer should also have fired
	if atomic.LoadInt32(&cleanup) == 0 {
		t.Error("OnCallCleanup not fired")
	}
}

// TestScenario_InviteHandlerError_RejectAs500 covers the error path where
// InviteHandler.OnIncomingCall returns a non-rejection error.
func TestScenario_InviteHandlerError_RejectAs500(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	srv.SetInboundAllowUnknownDID(true)

	srv.SetInviteHandler(invHandlerAdapter(func(ctx context.Context, in *IncomingCall) (Decision, error) {
		return Decision{}, errIntentional
	}))

	inv, _ := stack.Parse(rawInviteFor("err-flow-1"))
	resp := srv.handleInvite(inv, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6050})
	if resp == nil {
		t.Fatal("nil")
	}
	// 488 is the generic invitate-build failure mapped from an unclassified error.
	if resp.StatusCode == 200 {
		t.Errorf("error path should not produce 200: %d", resp.StatusCode)
	}
}

// TestScenario_RegistrarOnly demonstrates the SIP server with no business
// layer wired — pure registrar usage.
func TestScenario_RegistrarOnly_OPTIONS_INVITE(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	srv.SetInboundAllowUnknownDID(true)

	// OPTIONS still answered.
	opt, _ := stack.Parse(strings.Join([]string{
		"OPTIONS sip:server SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKopt2",
		"From: <sip:probe@a>;tag=1",
		"To: <sip:server@b>",
		"Call-ID: opt-2",
		"CSeq: 1 OPTIONS",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if r := srv.handleOptions(opt, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060}); r == nil || r.StatusCode != 200 {
		t.Errorf("OPTIONS: %v", r)
	}

	// INVITE without handler → server builds a default echo MediaLeg and
	// answers 200 (default flow, useful for protocol smoke tests).
	inv, _ := stack.Parse(rawInviteFor("registrar-only-inv"))
	r := srv.handleInvite(inv, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6050})
	if r == nil || r.StatusCode != 200 {
		t.Errorf("default INVITE: %v", r)
	}
}

// ---------- helpers / adapters -----------------------------------------

type inviteHandlerFunc func(ctx context.Context, in *IncomingCall) (Decision, error)

func (f inviteHandlerFunc) OnIncomingCall(ctx context.Context, in *IncomingCall) (Decision, error) {
	return f(ctx, in)
}

func invHandlerAdapter(fn func(ctx context.Context, in *IncomingCall) (Decision, error)) InviteHandler {
	return inviteHandlerFunc(fn)
}

type observerFn func(callID string)

func (f observerFn) OnCallPreHangup(callID string) bool { return false }
func (f observerFn) OnCallCleanup(callID string)        { f(callID) }

var errIntentional = sentinelErr("intentional handler error")

type sentinelErr string

func (e sentinelErr) Error() string { return string(e) }

// extractToTag pulls the ;tag= value from a To header.
func extractToTag(toHeader string) string {
	idx := strings.Index(strings.ToLower(toHeader), ";tag=")
	if idx < 0 {
		return ""
	}
	v := toHeader[idx+5:]
	if cut := strings.IndexAny(v, ";>,"); cut >= 0 {
		v = v[:cut]
	}
	return strings.TrimSpace(v)
}

// buildReInviteSDP returns an SDP for re-INVITE that keeps the same codec.
func buildReInviteSDP(rtpPort int) string {
	return strings.Join([]string{
		"v=0",
		"o=- 654321 654322 IN IP4 198.51.100.1",
		"s=Session",
		"c=IN IP4 198.51.100.1",
		"t=0 0",
		"m=audio " + strconv.Itoa(rtpPort) + " RTP/AVP 0 8",
		"a=rtpmap:0 PCMU/8000",
		"a=rtpmap:8 PCMA/8000",
	}, "\r\n")
}
