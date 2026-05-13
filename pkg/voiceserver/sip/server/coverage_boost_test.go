package server

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// ---------- prependProxyVia -------------------------------------------------

func TestPrependProxyVia_DefaultsAndDecrement(t *testing.T) {
	m := &stack.Message{IsRequest: true, Method: "INVITE", RequestURI: "sip:x@y"}
	m.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bKold;rport")
	m.SetHeader("Max-Forwards", "70")

	prependProxyVia(m, "", 0) // default host/port

	vias := m.GetHeaders("via")
	if len(vias) != 2 {
		t.Fatalf("vias=%d want 2", len(vias))
	}
	if !strings.HasPrefix(vias[0], "SIP/2.0/UDP 127.0.0.1:6050;branch=z9hG4bK") {
		t.Errorf("top via malformed: %q", vias[0])
	}
	if mf := m.GetHeader("Max-Forwards"); mf != "69" {
		t.Errorf("Max-Forwards=%q want 69", mf)
	}
}

func TestPrependProxyVia_NilSafe(t *testing.T) {
	prependProxyVia(nil, "127.0.0.1", 5060) // must not panic
}

func TestPrependProxyVia_ExplicitHostPort(t *testing.T) {
	m := &stack.Message{IsRequest: true, Method: "INVITE", RequestURI: "sip:x@y"}
	m.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bKold")
	prependProxyVia(m, "proxy.test", 5070)
	if v := m.GetHeader("Via"); !strings.Contains(v, "proxy.test:5070") {
		t.Errorf("via=%q", v)
	}
}

// ---------- pending INVITE snap --------------------------------------------

func TestPendingInviteSnap_RoundTrip(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	// take on empty / missing returns nil
	if snap := s.takePendingInviteSnap(""); snap != nil {
		t.Error("empty callID should return nil")
	}
	if snap := s.takePendingInviteSnap("missing"); snap != nil {
		t.Error("missing callID should return nil")
	}
	// clearPendingInviteSnap safe on empty
	s.clearPendingInviteSnap("")
	s.clearPendingInviteSnap("missing")

	// nil server paths
	var nilSrv *SIPServer
	if snap := nilSrv.takePendingInviteSnap("c"); snap != nil {
		t.Error("nil server should return nil")
	}
	nilSrv.clearPendingInviteSnap("c")
}

// ---------- buildUASBye / buildReferNotify ---------------------------------

func primeUASDialog(t *testing.T, s *SIPServer, callID string) *net.UDPAddr {
	t.Helper()
	inv := &stack.Message{IsRequest: true, Method: "INVITE", RequestURI: "sip:callee@1.2.3.4",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	inv.SetHeader("From", "<sip:caller@example.com>;tag=abc")
	inv.SetHeader("Contact", "<sip:callee@1.2.3.4:5060>")
	inv.SetHeader("CSeq", "1 INVITE")
	remote := &net.UDPAddr{IP: net.ParseIP("1.2.3.4"), Port: 5060}
	s.rememberUASDialog(callID, remote, inv, "<sip:server@local>;tag=xyz")
	return remote
}

func TestBuildUASBye_Happy(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	primeUASDialog(t, s, "cid-bye")

	msg, dst, err := s.buildUASBye("cid-bye")
	if err != nil {
		t.Fatalf("buildUASBye err: %v", err)
	}
	if msg == nil || msg.Method != stack.MethodBye {
		t.Fatalf("bad msg: %+v", msg)
	}
	if dst == nil || dst.Port != 5060 {
		t.Errorf("dst=%v", dst)
	}
	if !strings.Contains(msg.GetHeader("CSeq"), "BYE") {
		t.Errorf("CSeq=%q", msg.GetHeader("CSeq"))
	}
	if msg.GetHeader("Call-ID") != "cid-bye" {
		t.Errorf("Call-ID=%q", msg.GetHeader("Call-ID"))
	}
}

func TestBuildUASBye_Errors(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	if _, _, err := s.buildUASBye(""); err == nil {
		t.Error("empty callID should error")
	}
	if _, _, err := s.buildUASBye("nope"); err == nil {
		t.Error("missing dialog should error")
	}
	var nilS *SIPServer
	if _, _, err := nilS.buildUASBye("x"); err == nil {
		t.Error("nil server should error")
	}
}

func TestBuildReferNotify_Happy(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	primeUASDialog(t, s, "cid-ref")

	msg, dst, err := s.buildReferNotify("cid-ref", "SIP/2.0 200 OK", "")
	if err != nil {
		t.Fatalf("buildReferNotify err: %v", err)
	}
	if msg.Method != stack.MethodNotify {
		t.Errorf("method=%q", msg.Method)
	}
	if !strings.Contains(msg.GetHeader("Subscription-State"), "active") {
		t.Errorf("default sub state missing, got %q", msg.GetHeader("Subscription-State"))
	}
	if !strings.Contains(msg.GetHeader("Content-Type"), "sipfrag") {
		t.Errorf("content-type=%q", msg.GetHeader("Content-Type"))
	}
	if dst == nil {
		t.Error("dst nil")
	}

	// custom subscription state
	msg2, _, err := s.buildReferNotify("cid-ref", "SIP/2.0 487 Terminated", "terminated;reason=noresource")
	if err != nil {
		t.Fatalf("custom: %v", err)
	}
	if msg2.GetHeader("Subscription-State") != "terminated;reason=noresource" {
		t.Errorf("sub state=%q", msg2.GetHeader("Subscription-State"))
	}
}

func TestBuildReferNotify_Errors(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	if _, _, err := s.buildReferNotify("", "frag", ""); err == nil {
		t.Error("empty callID should error")
	}
	if _, _, err := s.buildReferNotify("none", "frag", ""); err == nil {
		t.Error("missing dialog should error")
	}
}

// ---------- resendInviteProgress -------------------------------------------

func TestResendInviteProgress_NilGuards(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	// All variations must be nil-safe (ep is nil when not started, addr nil)
	s.resendInviteProgress(nil, nil)
	s.resendInviteProgress(&inviteFlightState{}, nil)
	s.resendInviteProgress(&inviteFlightState{}, &net.UDPAddr{})

	var nilS *SIPServer
	nilS.resendInviteProgress(&inviteFlightState{}, &net.UDPAddr{})
}

// ---------- abortInviteFlight ----------------------------------------------

func TestAbortInviteFlight_NilGuards(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	s.abortInviteFlight(nil, nil, nil, "") // nil flight
	var nilS *SIPServer
	nilS.abortInviteFlight(&inviteFlightState{callID: "c"}, nil, nil, "") // nil server

	// flight set but nil ep+addr: should still take stopCallSessionLocked branch
	fl := &inviteFlightState{callID: "c-abort"}
	s.abortInviteFlight(fl, nil, nil, "")
}

// ---------- stopCallSessionLocked ------------------------------------------

func TestStopCallSessionLocked_NilSafe(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	s.stopCallSessionLocked("")   // empty → no-op
	s.stopCallSessionLocked("no") // missing → no-op

	var nilS *SIPServer
	nilS.stopCallSessionLocked("c") // nil server → no-op
}

// ---------- buildNotifyForSubscription -------------------------------------

func TestBuildNotifyForSubscription(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	// nil guards
	if s.buildNotifyForSubscription(nil, 1, "b", "") != nil {
		t.Error("nil sub must yield nil")
	}
	if s.buildNotifyForSubscription(&presenceSub{}, 1, "b", "") != nil {
		t.Error("sub with nil req must yield nil")
	}

	req := &stack.Message{IsRequest: true, Method: "SUBSCRIBE", RequestURI: "sip:bob@example.com",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	req.SetHeader("From", "<sip:alice@example.com>;tag=1")
	req.SetHeader("To", "<sip:bob@example.com>")
	req.SetHeader("Call-ID", "sub-1")

	sub := &presenceSub{req: req, expiresSec: 0} // default 3600
	n := s.buildNotifyForSubscription(sub, 2, "<presence/>", "application/pidf+xml")
	if n == nil || n.Method != stack.MethodNotify {
		t.Fatalf("bad notify: %+v", n)
	}
	if !strings.Contains(n.GetHeader("Subscription-State"), "expires=3600") {
		t.Errorf("default expires missing: %q", n.GetHeader("Subscription-State"))
	}
	if n.GetHeader("Call-ID") != "sub-1" {
		t.Errorf("call-id=%q", n.GetHeader("Call-ID"))
	}

	// with explicit expires + empty body + empty content-type branches
	sub.expiresSec = 60
	req.RequestURI = "" // exercise default RequestURI branch
	n2 := s.buildNotifyForSubscription(sub, 3, "", "")
	if n2.RequestURI != "sip:presence" {
		t.Errorf("default RequestURI=%q", n2.RequestURI)
	}
	if n2.Body != "" || n2.GetHeader("Content-Length") != "0" {
		t.Errorf("empty body expected, body=%q clen=%q", n2.Body, n2.GetHeader("Content-Length"))
	}
}

// ---------- TCP listener smoke ---------------------------------------------

func TestListenTCP_StartsAndStops(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1", SIPTCPListen: "127.0.0.1:0"})
	defer s.Stop()

	s.ensureSigCtx()
	// startSigTransportListeners requires port > 0; use listenTCP directly on ephemeral.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listenTCP(s.sigCtx, addr) // will error on listen (port reused quickly) → returns immediately
	}()

	// Separately, test listener loop shutdown via sigCancel (using a fresh port).
	s2 := New(Config{LocalIP: "127.0.0.1"})
	defer s2.Stop()
	s2.ensureSigCtx()

	lnOK, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen2: %v", err)
	}
	addr2 := lnOK.Addr().String()
	_ = lnOK.Close()

	loopDone := make(chan struct{})
	go func() {
		defer close(loopDone)
		s2.listenTCP(s2.sigCtx, addr2)
	}()
	time.Sleep(50 * time.Millisecond)
	s2.sigCancel()
	select {
	case <-loopDone:
	case <-time.After(3 * time.Second):
		t.Error("listenTCP did not exit after cancel")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		// first listenTCP might block on accept loop; cancel root
		s.sigCancel()
		<-done
	}
}

// ---------- dispatchSignalingRequestTCP guards -----------------------------

func TestDispatchSignalingRequestTCP_NilGuards(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	// send nil → early return
	s.dispatchSignalingRequestTCP(&stack.Message{}, nil, nil)

	// ep nil → early return
	called := false
	s.dispatchSignalingRequestTCP(nil, nil, func(m *stack.Message) error {
		called = true
		return nil
	})
	if called {
		t.Error("send should not be called on nil req")
	}
}

// ---------- clearPendingInviteSnap nil map path ----------------------------

func TestClearPendingInviteSnap_NilMap(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	// fresh server, pendingInv is nil; clearing should not panic
	s.clearPendingInviteSnap("x")
}
