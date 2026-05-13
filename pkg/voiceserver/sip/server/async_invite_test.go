package server

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Tests targeting runInviteAsync and handlePrack (RFC 3262) paths.

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// uacSink starts a UDP listener and returns received datagrams on a channel.
// It handles packets until the test closes the listener.
type uacSink struct {
	addr *net.UDPAddr
	pc   net.PacketConn
	msgs chan string
}

func newUACSink(t *testing.T) *uacSink {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("uac listen: %v", err)
	}
	u := &uacSink{
		addr: pc.LocalAddr().(*net.UDPAddr),
		pc:   pc,
		msgs: make(chan string, 32),
	}
	go func() {
		buf := make([]byte, 16*1024)
		for {
			n, _, err := u.pc.ReadFrom(buf)
			if err != nil {
				return
			}
			s := string(buf[:n])
			select {
			case u.msgs <- s:
			default:
			}
		}
	}()
	return u
}

func (u *uacSink) close() { _ = u.pc.Close() }

func (u *uacSink) waitFor(t *testing.T, timeout time.Duration, match func(string) bool) (string, bool) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case m := <-u.msgs:
			if match(m) {
				return m, true
			}
		case <-deadline:
			return "", false
		}
	}
}

func rawInviteForAsync(callID, uacAddr string) string {
	return strings.Join([]string{
		"INVITE sip:target@127.0.0.1 SIP/2.0",
		"Via: SIP/2.0/UDP " + uacAddr + ";branch=z9hG4bK-" + callID + ";rport",
		"Max-Forwards: 70",
		"From: <sip:a@example.com>;tag=a",
		"To: <sip:target@127.0.0.1>",
		"Call-ID: " + callID,
		"CSeq: 1 INVITE",
		"Contact: <sip:a@" + uacAddr + ">",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
}

// ---------- Guard clauses --------------------------------------------------

func TestRunInviteAsync_NilGuards(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	// Should return immediately on every nil combination.
	s.runInviteAsync(nil, nil, nil, nil, false, "", "c")
	s.runInviteAsync(&stack.Message{}, nil, nil, nil, false, "", "c")
	s.runInviteAsync(&stack.Message{}, &net.UDPAddr{}, nil, nil, false, "", "c")
	s.runInviteAsync(&stack.Message{}, &net.UDPAddr{}, &inviteFlightState{}, nil, false, "", "c")
	var nilS *SIPServer
	nilS.runInviteAsync(&stack.Message{}, &net.UDPAddr{}, &inviteFlightState{}, &stack.Message{}, false, "", "c")
}

// ---------- Plain path: Send180=true, not reliable, no ringback -------------

func TestRunInviteAsync_PlainPath_Sends180And200(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1", InviteSend180: true})
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	uac := newUACSink(t)
	defer uac.close()

	req, err := stack.Parse(rawInviteForAsync("async-plain-1", uac.addr.String()))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	resp200 := srv.makeResponse(req, 200, "OK", "v=0\r\n", ";tag=srv")
	resp200.SetHeader("Contact", "<sip:server@127.0.0.1>")
	resp200.SetHeader("Allow", "INVITE, ACK, BYE")

	flight := &inviteFlightState{
		flightKey: inviteFlightKey(req),
		callID:    "async-plain-1",
		prackDone: make(chan struct{}, 1),
	}
	srv.inviteByCall.Store("async-plain-1", flight)

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.runInviteAsync(req, uac.addr, flight, resp200, false, "", "async-plain-1")
	}()

	// Expect 180 Ringing
	_, ok := uac.waitFor(t, 2*time.Second, func(m string) bool { return strings.Contains(m, "SIP/2.0 180") })
	if !ok {
		t.Error("did not receive 180 Ringing")
	}
	// Expect 200 OK
	_, ok = uac.waitFor(t, 2*time.Second, func(m string) bool { return strings.Contains(m, "SIP/2.0 200") })
	if !ok {
		t.Error("did not receive 200 OK")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("runInviteAsync did not return")
	}

	if !flight.completed {
		t.Error("flight should be marked completed")
	}
}

// ---------- Ringback path: small ringback delay, then 200 ------------------

func TestRunInviteAsync_RingbackDelay(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1", InviteRingbackMS: 50})
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	uac := newUACSink(t)
	defer uac.close()

	req, _ := stack.Parse(rawInviteForAsync("async-ring-1", uac.addr.String()))
	resp200 := srv.makeResponse(req, 200, "OK", "", ";tag=srv")
	resp200.SetHeader("Contact", "<sip:server@127.0.0.1>")

	flight := &inviteFlightState{flightKey: inviteFlightKey(req), callID: "async-ring-1", prackDone: make(chan struct{}, 1)}
	srv.inviteByCall.Store("async-ring-1", flight)

	start := time.Now()
	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.runInviteAsync(req, uac.addr, flight, resp200, false, "", "async-ring-1")
	}()
	<-done
	elapsed := time.Since(start)
	if elapsed < 40*time.Millisecond {
		t.Errorf("ringback delay not applied: %v", elapsed)
	}
	if _, ok := uac.waitFor(t, 2*time.Second, func(m string) bool { return strings.Contains(m, "SIP/2.0 200") }); !ok {
		t.Error("200 OK not received")
	}
}

// ---------- Reliable path: PRACK arrives → fast path -----------------------

func TestRunInviteAsync_Reliable_PRACKUnblocks(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1", InviteForce100rel: true})
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	uac := newUACSink(t)
	defer uac.close()

	req, _ := stack.Parse(rawInviteForAsync("async-rel-1", uac.addr.String()))
	resp200 := srv.makeResponse(req, 200, "OK", "", ";tag=srv")
	resp200.SetHeader("Contact", "<sip:server@127.0.0.1>")
	resp200.SetHeader("Allow", "INVITE, ACK, BYE")

	flight := &inviteFlightState{
		flightKey:  inviteFlightKey(req),
		callID:     "async-rel-1",
		prackDone:  make(chan struct{}, 1),
		inviteCSeq: 1,
	}
	srv.inviteByCall.Store("async-rel-1", flight)

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.runInviteAsync(req, uac.addr, flight, resp200, true, "v=0\r\n", "async-rel-1")
	}()

	// Expect 183 Session Progress carrying RSeq=1 + Require:100rel
	raw183, ok := uac.waitFor(t, 2*time.Second, func(m string) bool { return strings.Contains(m, "SIP/2.0 183") })
	if !ok {
		t.Fatal("did not receive 183")
	}
	if !strings.Contains(strings.ToLower(raw183), "rseq: 1") {
		t.Errorf("183 missing RSeq: %q", raw183)
	}
	if !strings.Contains(raw183, "Require: 100rel") {
		t.Errorf("183 missing Require: %q", raw183)
	}

	// Simulate PRACK: push to prackDone directly.
	flight.prackDone <- struct{}{}

	// Expect 200 OK afterwards.
	if _, ok := uac.waitFor(t, 2*time.Second, func(m string) bool { return strings.Contains(m, "SIP/2.0 200") }); !ok {
		t.Error("200 OK not received after PRACK")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("runInviteAsync did not return")
	}
}

// ---------- Reliable path: sigCtx cancelled → badExit / abort --------------

func TestRunInviteAsync_Reliable_SigCtxCancel(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1", InviteForce100rel: true})
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	// We will Stop() manually to cancel sigCtx; no defer Stop.

	uac := newUACSink(t)
	defer uac.close()

	req, _ := stack.Parse(rawInviteForAsync("async-cancel-1", uac.addr.String()))
	resp200 := srv.makeResponse(req, 200, "OK", "", ";tag=srv")
	resp200.SetHeader("Contact", "<sip:server@127.0.0.1>")

	flight := &inviteFlightState{
		flightKey:  inviteFlightKey(req),
		callID:     "async-cancel-1",
		prackDone:  make(chan struct{}, 1),
		inviteCSeq: 1,
	}
	srv.inviteByCall.Store("async-cancel-1", flight)

	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.runInviteAsync(req, uac.addr, flight, resp200, true, "", "async-cancel-1")
	}()

	// Wait for 183 before cancelling (otherwise the reliable branch has not
	// started the select).
	if _, ok := uac.waitFor(t, 2*time.Second, func(m string) bool { return strings.Contains(m, "SIP/2.0 183") }); !ok {
		srv.Stop()
		t.Fatal("183 not received before cancel")
	}

	// Cancel the signalling context → the <-s.sigCtx.Done() branch fires.
	srv.Stop()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("runInviteAsync did not return after sigCtx cancel")
	}

	if flight.completed {
		t.Error("flight should not be completed on abort")
	}
}

// ---------- handlePrack: happy + guard paths -------------------------------

func TestHandlePrack_GuardsAndSuccess(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	// Wrong method → nil
	if resp := srv.handlePrack(&stack.Message{IsRequest: true, Method: "INVITE"}, nil); resp != nil {
		t.Error("wrong method should return nil")
	}
	// Nil msg / non-request → nil
	if resp := srv.handlePrack(nil, nil); resp != nil {
		t.Error("nil msg → nil")
	}

	// Missing Call-ID → 400
	bad := &stack.Message{IsRequest: true, Method: stack.MethodPrack,
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	bad.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK-x")
	bad.SetHeader("CSeq", "1 PRACK")
	if resp := srv.handlePrack(bad, &net.UDPAddr{IP: net.ParseIP("10.0.0.1")}); resp == nil || resp.StatusCode != 400 {
		t.Errorf("missing Call-ID → 400, got %+v", resp)
	}

	// Missing RAck → 400
	bad2 := &stack.Message{IsRequest: true, Method: stack.MethodPrack,
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	bad2.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK-y")
	bad2.SetHeader("Call-ID", "pr-1")
	bad2.SetHeader("CSeq", "2 PRACK")
	if resp := srv.handlePrack(bad2, &net.UDPAddr{IP: net.ParseIP("10.0.0.1")}); resp == nil || resp.StatusCode != 400 {
		t.Errorf("missing RAck → 400, got %+v", resp)
	}

	// Call-ID unknown → 481
	good := &stack.Message{IsRequest: true, Method: stack.MethodPrack,
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	good.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK-z")
	good.SetHeader("Call-ID", "pr-unknown")
	good.SetHeader("CSeq", "2 PRACK")
	good.SetHeader("RAck", "1 1 INVITE")
	if resp := srv.handlePrack(good, &net.UDPAddr{IP: net.ParseIP("10.0.0.1")}); resp == nil || resp.StatusCode != 481 {
		t.Errorf("unknown call → 481, got %+v", resp)
	}

	// Register a flight; then PRACK with wrong RSeq → 481
	fl := &inviteFlightState{callID: "pr-known", inviteCSeq: 1, awaitRSeq: 1,
		prackDone: make(chan struct{}, 1)}
	srv.inviteByCall.Store("pr-known", fl)
	wrong := *good
	wrong.Headers = map[string]string{}
	wrong.HeadersMulti = map[string][]string{}
	wrong.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK-wrong-rseq")
	wrong.SetHeader("Call-ID", "pr-known")
	wrong.SetHeader("CSeq", "2 PRACK")
	wrong.SetHeader("RAck", "99 1 INVITE")
	if resp := srv.handlePrack(&wrong, &net.UDPAddr{IP: net.ParseIP("10.0.0.1")}); resp == nil || resp.StatusCode != 481 {
		t.Errorf("rseq mismatch → 481, got %+v", resp)
	}

	// Matching RAck → 200 + prackDone signalled.
	match := *good
	match.Headers = map[string]string{}
	match.HeadersMulti = map[string][]string{}
	match.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK-match")
	match.SetHeader("Call-ID", "pr-known")
	match.SetHeader("CSeq", "2 PRACK")
	match.SetHeader("RAck", "1 1 INVITE")
	resp := srv.handlePrack(&match, &net.UDPAddr{IP: net.ParseIP("10.0.0.1")})
	if resp == nil || resp.StatusCode != 200 {
		t.Fatalf("PRACK ok → 200, got %+v", resp)
	}
	select {
	case <-fl.prackDone:
	default:
		t.Error("prackDone not signalled")
	}

	// Wrong method check on RAck (method must be INVITE)
	badMethod := *good
	badMethod.Headers = map[string]string{}
	badMethod.HeadersMulti = map[string][]string{}
	badMethod.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK-bm")
	badMethod.SetHeader("Call-ID", "pr-bad-method")
	badMethod.SetHeader("CSeq", "2 PRACK")
	badMethod.SetHeader("RAck", "1 1 BYE")
	if resp := srv.handlePrack(&badMethod, &net.UDPAddr{IP: net.ParseIP("10.0.0.1")}); resp == nil || resp.StatusCode != 400 {
		t.Errorf("RAck method=BYE → 400, got %+v", resp)
	}
}

// ---------- inviteAsyncEnd cleanup -----------------------------------------

func TestInviteAsyncEnd_RemovesFromMaps(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()

	fl := &inviteFlightState{flightKey: "fk-1", callID: "call-1", prackDone: make(chan struct{}, 1)}
	srv.inviteByCall.Store("call-1", fl)
	srv.inviteFlights.Store("fk-1", fl)

	srv.inviteAsyncEnd("call-1")

	if _, ok := srv.inviteByCall.Load("call-1"); ok {
		t.Error("inviteByCall still has entry")
	}
	if _, ok := srv.inviteFlights.Load("fk-1"); ok {
		t.Error("inviteFlights still has entry")
	}

	// Guard paths: empty callID / nil / missing entry.
	srv.inviteAsyncEnd("")
	srv.inviteAsyncEnd("missing")
	var nilS *SIPServer
	nilS.inviteAsyncEnd("x")
}

// ---------- resendInviteProgress: completed branch -------------------------

func TestResendInviteProgress_CompletedReplays200(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	uac := newUACSink(t)
	defer uac.close()

	ok200 := strings.Join([]string{
		"SIP/2.0 200 OK",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK-r",
		"From: <sip:a@example.com>;tag=a",
		"To: <sip:b@example.com>;tag=s",
		"Call-ID: resend-1",
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	fl := &inviteFlightState{lastOK200Raw: ok200, completed: true}
	srv.resendInviteProgress(fl, uac.addr)

	if _, ok := uac.waitFor(t, 1*time.Second, func(m string) bool { return strings.HasPrefix(m, "SIP/2.0 200") }); !ok {
		t.Error("expected cached 200 to be resent")
	}

	// Pending (not completed, only provisional) branch
	prov := strings.Replace(ok200, "SIP/2.0 200 OK", "SIP/2.0 183 Session Progress", 1)
	fl2 := &inviteFlightState{lastProvRaw: prov, completed: false}
	srv.resendInviteProgress(fl2, uac.addr)
	if _, ok := uac.waitFor(t, 1*time.Second, func(m string) bool { return strings.HasPrefix(m, "SIP/2.0 183") }); !ok {
		t.Error("expected cached 183 to be resent")
	}
}

// ---------- abortInviteFlight with real ep ---------------------------------

func TestAbortInviteFlight_Sends504(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	uac := newUACSink(t)
	defer uac.close()

	req, _ := stack.Parse(rawInviteForAsync("abort-1", uac.addr.String()))
	fl := &inviteFlightState{callID: "abort-1", prackDone: make(chan struct{}, 1)}
	srv.inviteByCall.Store("abort-1", fl)

	srv.abortInviteFlight(fl, req, uac.addr, ";tag=srv")

	if _, ok := uac.waitFor(t, 1*time.Second, func(m string) bool { return strings.Contains(m, "SIP/2.0 504") }); !ok {
		t.Error("504 not received")
	}
	if _, ok := srv.inviteByCall.Load("abort-1"); ok {
		t.Error("inviteByCall entry not cleaned up")
	}
}

// ---------- parse path used by handleInvite for non-async path --------------
// Sanity — the async needs-decision; toggling InviteNeedsAsync pulls it in.

func TestInviteNeedsAsync(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	if srv.inviteNeedsAsync(nil) {
		t.Error("nil req should not need async")
	}
	req := &stack.Message{IsRequest: true, Method: "INVITE",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	if srv.inviteNeedsAsync(req) {
		t.Error("plain INVITE without 100rel/ringback should not need async")
	}
	req.SetHeader("Require", "100rel")
	if !srv.inviteNeedsAsync(req) {
		t.Error("INVITE with Require:100rel should need async")
	}

	// Force via config
	srv2 := New(Config{LocalIP: "127.0.0.1", InviteForce100rel: true})
	defer srv2.Stop()
	plain := &stack.Message{IsRequest: true, Method: "INVITE",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	if !srv2.inviteNeedsAsync(plain) {
		t.Error("Force100rel should require async")
	}
}

var _ = context.Background // keep import stable if Context ever used
var _ = atomic.LoadInt32    // avoid unused helper
