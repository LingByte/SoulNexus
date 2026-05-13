package transaction

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// ---------- helpers --------------------------------------------------------

func mustParse(t *testing.T, raw string) *stack.Message {
	t.Helper()
	m, err := stack.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return m
}

func byeRequest(t *testing.T) *stack.Message {
	return mustParse(t, strings.Join([]string{
		"BYE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKtxbye",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>;tag=2",
		"Call-ID: cid-tx-bye",
		"CSeq: 2 BYE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
}

func twoXX(t *testing.T, req *stack.Message, status int) *stack.Message {
	resp, _ := stack.Parse(strings.Join([]string{
		"SIP/2.0 200 OK",
		"Via: " + req.GetHeader("Via"),
		"From: " + req.GetHeader("From"),
		"To: " + req.GetHeader("To"),
		"Call-ID: " + req.GetHeader("Call-ID"),
		"CSeq: " + req.GetHeader("CSeq"),
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	resp.StatusCode = status
	return resp
}

// ---------- helpers.go (small string utilities) --------------------------

func TestInviteTransactionKey_StableFormat(t *testing.T) {
	k1 := InviteTransactionKey(" branch1 ", " call-1 ")
	k2 := InviteTransactionKey("branch1", "call-1")
	if k1 != k2 {
		t.Errorf("InviteTransactionKey not whitespace-stable: %q vs %q", k1, k2)
	}
	if !strings.Contains(k1, "branch1") || !strings.Contains(k1, "call-1") {
		t.Errorf("key missing components: %q", k1)
	}
}

func TestNonInviteServerKey_NilSafety(t *testing.T) {
	if NonInviteServerKey(nil) != "" {
		t.Error("nil request must yield empty key")
	}
}

func TestTopVia_AndBranch(t *testing.T) {
	if got := TopVia(nil); got != "" {
		t.Error("TopVia(nil) should be empty")
	}
	if got := TopBranch(nil); got != "" {
		t.Error("TopBranch(nil) should be empty")
	}
	if got := BranchParam(""); got != "" {
		t.Error("BranchParam(empty) should be empty")
	}
	if got := BranchParam("SIP/2.0/UDP host;param=v"); got != "" {
		t.Error("BranchParam without branch= should be empty")
	}
	if got := BranchParam(`SIP/2.0/UDP host;branch="z9hG4bKaaa"`); got != "z9hG4bKaaa" {
		t.Errorf("BranchParam quoted: %q", got)
	}
	if got := BranchParam("SIP/2.0/UDP host;branch=z9hG4bKxx;rport"); got != "z9hG4bKxx" {
		t.Errorf("BranchParam multi-param: %q", got)
	}
}

func TestIsInviteCSeq_AndIsAckCSeq_AndIsCancelCSeq_NilAndPositive(t *testing.T) {
	if IsInviteCSeq(nil) || IsAckCSeq(nil) || IsCancelCSeq(nil) {
		t.Error("nil request must return false on CSeq predicates")
	}
	inv := mustParse(t, strings.Join([]string{
		"INVITE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKi",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>",
		"Call-ID: cid",
		"CSeq: 1 invite", // case-insensitive check
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if !IsInviteCSeq(inv) {
		t.Error("INVITE CSeq must match (case-insensitive)")
	}
	if IsAckCSeq(inv) || IsCancelCSeq(inv) {
		t.Error("non-ACK/CANCEL CSeq must not match")
	}
}

// ---------- record_route.go ----------------------------------------------

func TestRouteHeadersForDialog_NilAndEmpty(t *testing.T) {
	if got := RouteHeadersForDialog(nil); got != nil {
		t.Errorf("nil → nil expected, got %v", got)
	}
	resp := mustParse(t, strings.Join([]string{
		"SIP/2.0 200 OK",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKx",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>;tag=2",
		"Call-ID: c",
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if got := RouteHeadersForDialog(resp); got != nil {
		t.Errorf("no Record-Route → nil expected, got %v", got)
	}
}

func TestRouteHeadersForDialog_ReversesOrder(t *testing.T) {
	resp := mustParse(t, strings.Join([]string{
		"SIP/2.0 200 OK",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKx",
		"Record-Route: <sip:proxy1@a;lr>",
		"Record-Route: <sip:proxy2@b;lr>",
		"Record-Route:    ", // empty entry → must be filtered
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>;tag=2",
		"Call-ID: c",
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	got := RouteHeadersForDialog(resp)
	if len(got) != 2 {
		t.Fatalf("expected 2 routes (empty filtered), got %v", got)
	}
	// Reverse order: proxy2 should come before proxy1
	if !strings.Contains(got[0], "proxy2") {
		t.Errorf("first route after reverse should contain proxy2, got %q", got[0])
	}
	if !strings.Contains(got[1], "proxy1") {
		t.Errorf("second route should contain proxy1, got %q", got[1])
	}
}

// ---------- noninvite client/server ---------------------------------------

func TestRunNonInviteClient_GuardClauses(t *testing.T) {
	mgr := NewManager()

	// nil manager
	var nilMgr *Manager
	if _, err := nilMgr.RunNonInviteClient(context.Background(), byeRequest(t), nil, nil); err == nil {
		t.Error("nil manager must error")
	}

	// nil request
	if _, err := mgr.RunNonInviteClient(context.Background(), nil, nil, func(*stack.Message, *net.UDPAddr) error { return nil }); err == nil {
		t.Error("nil request must error")
	}

	// nil send
	if _, err := mgr.RunNonInviteClient(context.Background(), byeRequest(t), nil, nil); err == nil {
		t.Error("nil send must error")
	}

	// missing Call-ID — strip it from a fresh BYE
	noCID := mustParse(t, strings.Join([]string{
		"BYE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKbye2",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>;tag=2",
		"CSeq: 2 BYE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if _, err := mgr.RunNonInviteClient(context.Background(), noCID, nil, func(*stack.Message, *net.UDPAddr) error { return nil }); err == nil {
		t.Error("missing Call-ID must error")
	}

	// invalid CSeq
	bad := mustParse(t, strings.Join([]string{
		"BYE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKbye3",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>;tag=2",
		"Call-ID: cid",
		"CSeq: 0 BYE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if _, err := mgr.RunNonInviteClient(context.Background(), bad, nil, func(*stack.Message, *net.UDPAddr) error { return nil }); err == nil {
		t.Error("invalid CSeq must error")
	}
}

func TestRunNonInviteClient_FinalResponse(t *testing.T) {
	mgr := NewManager()
	mgr.SetT1(20 * time.Millisecond)
	mgr.SetT2(80 * time.Millisecond)

	req := byeRequest(t)
	src := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5060}

	var sendCount int32
	var sendMu sync.Mutex
	send := SendFunc(func(msg *stack.Message, addr *net.UDPAddr) error {
		sendMu.Lock()
		sendCount++
		sendMu.Unlock()
		return nil
	})

	// Drive a 200 OK response asynchronously after the first send.
	go func() {
		time.Sleep(15 * time.Millisecond)
		mgr.HandleResponse(twoXX(t, req, 200), src)
	}()

	res, err := mgr.RunNonInviteClient(context.Background(), req, src, send)
	if err != nil {
		t.Fatalf("RunNonInviteClient: %v", err)
	}
	if res == nil || res.Final == nil || res.Final.StatusCode != 200 {
		t.Fatalf("expected 200 final, got %+v", res)
	}
}

func TestRunNonInviteClient_RetransmitsBeforeFinal(t *testing.T) {
	// Force Timer E to fire several times before we feed a final response so
	// retransmitLoop's exponential backoff branch (next = next*2 / cap at T2)
	// gets covered.
	mgr := NewManager()
	mgr.SetT1(8 * time.Millisecond)
	mgr.SetT2(20 * time.Millisecond)

	req := byeRequest(t)
	src := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5070}

	var (
		mu        sync.Mutex
		sendCount int
	)
	send := SendFunc(func(msg *stack.Message, addr *net.UDPAddr) error {
		mu.Lock()
		sendCount++
		mu.Unlock()
		return nil
	})

	go func() {
		// Wait long enough for ~3 retransmits
		time.Sleep(80 * time.Millisecond)
		mgr.HandleResponse(twoXX(t, req, 200), src)
	}()

	res, err := mgr.RunNonInviteClient(context.Background(), req, src, send)
	if err != nil || res == nil {
		t.Fatalf("RunNonInviteClient: %v", err)
	}
	mu.Lock()
	got := sendCount
	mu.Unlock()
	if got < 2 {
		t.Errorf("expected ≥2 sends (initial + retransmits), got %d", got)
	}
}

func TestRunNonInviteClient_FinalAfterFinalIgnored(t *testing.T) {
	mgr := NewManager()
	mgr.SetT1(20 * time.Millisecond)

	req := byeRequest(t)
	src := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5071}
	send := SendFunc(func(*stack.Message, *net.UDPAddr) error { return nil })

	go func() {
		time.Sleep(15 * time.Millisecond)
		mgr.HandleResponse(twoXX(t, req, 200), src)
		// Re-send a duplicate final to drive the "finalSeen already" branch.
		mgr.HandleResponse(twoXX(t, req, 200), src)
	}()

	if _, err := mgr.RunNonInviteClient(context.Background(), req, src, send); err != nil {
		t.Fatalf("RunNonInviteClient: %v", err)
	}
}

func TestRunNonInviteClient_ContextCancellation(t *testing.T) {
	mgr := NewManager()
	mgr.SetT1(50 * time.Millisecond)

	req := byeRequest(t)
	send := SendFunc(func(msg *stack.Message, addr *net.UDPAddr) error { return nil })

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	if _, err := mgr.RunNonInviteClient(ctx, req, nil, send); err == nil {
		t.Error("expected ctx.Err on cancellation")
	}
}

// ---------- Manager.HandleResponse for non-INVITE client tx --------------

// ---------- BeginNonInviteServer end-to-end ------------------------------

func TestBeginNonInviteServer_GuardClauses(t *testing.T) {
	mgr := NewManager()
	send := SendFunc(func(*stack.Message, *net.UDPAddr) error { return nil })
	bye := byeRequest(t)
	final := twoXX(t, bye, 200)

	var nilMgr *Manager
	if err := nilMgr.BeginNonInviteServer(context.Background(), bye, nil, final, send); err == nil {
		t.Error("nil manager must error")
	}
	if err := mgr.BeginNonInviteServer(context.Background(), nil, nil, final, send); err == nil {
		t.Error("nil request must error")
	}
	if err := mgr.BeginNonInviteServer(context.Background(), bye, nil, nil, send); err == nil {
		t.Error("nil final must error")
	}
	if err := mgr.BeginNonInviteServer(context.Background(), bye, nil, final, nil); err == nil {
		t.Error("nil send must error")
	}
	// non-final status
	bad := twoXX(t, bye, 100)
	if err := mgr.BeginNonInviteServer(context.Background(), bye, nil, bad, send); err == nil {
		t.Error("non-final status must error")
	}
	// INVITE method must be rejected
	inv := mustParse(t, strings.Join([]string{
		"INVITE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKi2",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>",
		"Call-ID: cid",
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if err := mgr.BeginNonInviteServer(context.Background(), inv, nil, twoXX(t, inv, 200), send); err == nil {
		t.Error("INVITE method must be rejected")
	}
}

func TestBeginNonInviteServer_RegistersAndAbsorbsRetransmits(t *testing.T) {
	mgr := NewManager()
	mgr.SetT1(15 * time.Millisecond) // shorten Timer J (64*T1) so test stays fast

	bye := byeRequest(t)
	final := twoXX(t, bye, 200)
	src := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5060}

	var sentCount int32
	var mu sync.Mutex
	send := SendFunc(func(msg *stack.Message, addr *net.UDPAddr) error {
		mu.Lock()
		sentCount++
		mu.Unlock()
		return nil
	})

	if err := mgr.BeginNonInviteServer(context.Background(), bye, src, final, send); err != nil {
		t.Fatalf("BeginNonInviteServer: %v", err)
	}

	// Duplicate non-INVITE request → manager retransmits stored final response.
	if !mgr.HandleNonInviteRequest(bye, src) {
		t.Error("duplicate request should be absorbed by transaction layer")
	}

	// Wait for Timer J to expire so signalStop / unregisterNonInviteTx run.
	time.Sleep(64*15*time.Millisecond + 50*time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if sentCount < 1 {
		t.Errorf("expected at least 1 final retransmit, got %d", sentCount)
	}
}

// ---------- BuildAckForInvite + AckRequestURIFor2xx ---------------------

func inviteForAck(t *testing.T) *stack.Message {
	return mustParse(t, strings.Join([]string{
		"INVITE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKack-build",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>",
		"Call-ID: cid-ack-build",
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
}

func TestBuildAckForInvite_ErrorBranches(t *testing.T) {
	if _, err := BuildAckForInvite(nil, nil, ""); err == nil {
		t.Error("nil messages must error")
	}
	inv := inviteForAck(t)
	final := twoXX(t, inv, 200)

	// non-INVITE CSeq on the request
	notInv := mustParse(t, strings.Join([]string{
		"BYE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKbye-x",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>;tag=2",
		"Call-ID: cid",
		"CSeq: 1 BYE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if _, err := BuildAckForInvite(notInv, final, ""); err == nil {
		t.Error("non-INVITE CSeq must error")
	}

	// non-final status
	bad := twoXX(t, inv, 100)
	if _, err := BuildAckForInvite(inv, bad, ""); err == nil {
		t.Error("non-final status must error")
	}

	// invalid CSeq number
	zeroCSeq := mustParse(t, strings.Join([]string{
		"INVITE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKack-z",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>",
		"Call-ID: cid",
		"CSeq: 0 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if _, err := BuildAckForInvite(zeroCSeq, twoXX(t, zeroCSeq, 200), ""); err == nil {
		t.Error("CSeq 0 must error")
	}
}

func TestBuildAckForInvite_HappyPath_FillsHeaders(t *testing.T) {
	inv := inviteForAck(t)
	final := twoXX(t, inv, 200)
	ack, err := BuildAckForInvite(inv, final, "sip:x@10.0.0.1")
	if err != nil {
		t.Fatalf("BuildAckForInvite: %v", err)
	}
	if ack.Method != stack.MethodAck {
		t.Errorf("method = %q", ack.Method)
	}
	if ack.GetHeader("Call-ID") != "cid-ack-build" {
		t.Errorf("Call-ID lost: %q", ack.GetHeader("Call-ID"))
	}
	if !strings.Contains(ack.GetHeader("CSeq"), "ACK") {
		t.Errorf("CSeq method = %q", ack.GetHeader("CSeq"))
	}
	if ack.GetHeader("Max-Forwards") != "70" {
		t.Errorf("Max-Forwards = %q", ack.GetHeader("Max-Forwards"))
	}
}

func TestAckRequestURIFor2xx_ContactPathsAndFallback(t *testing.T) {
	resp := mustParse(t, strings.Join([]string{
		"SIP/2.0 200 OK",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKr",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>;tag=2",
		"Call-ID: c",
		"CSeq: 1 INVITE",
		"Contact: <sip:user@10.0.0.5;transport=udp>",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	got := AckRequestURIFor2xx(resp, "sip:fallback@host")
	if got != "sip:user@10.0.0.5" {
		t.Errorf("Contact-extracted URI = %q", got)
	}

	// nil response → fallback
	if got := AckRequestURIFor2xx(nil, "sip:fallback@host"); got != "sip:fallback@host" {
		t.Errorf("nil resp fallback = %q", got)
	}

	// no Contact → fallback
	noContact := mustParse(t, strings.Join([]string{
		"SIP/2.0 200 OK",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKr2",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>;tag=2",
		"Call-ID: c",
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if got := AckRequestURIFor2xx(noContact, "sip:fallback@host"); got != "sip:fallback@host" {
		t.Errorf("no-contact fallback = %q", got)
	}

	// non-sip Contact → fallback
	nonSip := mustParse(t, strings.Join([]string{
		"SIP/2.0 200 OK",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKr3",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>;tag=2",
		"Call-ID: c",
		"CSeq: 1 INVITE",
		"Contact: <tel:+12025551234>",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if got := AckRequestURIFor2xx(nonSip, "sip:fb@h"); got != "sip:fb@h" {
		t.Errorf("non-sip contact fallback = %q", got)
	}
}

// ---------- RegisterPendingInviteServer / HandleCancelRequest -----------

func TestRegisterPendingInviteServer_AllErrorBranches(t *testing.T) {
	mgr := NewManager()
	var nilMgr *Manager
	if err := nilMgr.RegisterPendingInviteServer(inviteForAck(t)); err == nil {
		t.Error("nil mgr must error")
	}
	if err := mgr.RegisterPendingInviteServer(nil); err == nil {
		t.Error("nil request must error")
	}
	// missing Call-ID
	noCID := mustParse(t, strings.Join([]string{
		"INVITE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKx",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>",
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if err := mgr.RegisterPendingInviteServer(noCID); err == nil {
		t.Error("missing Call-ID must error")
	}
	// missing branch (no Via at all)
	noBranch := &stack.Message{IsRequest: true, Method: stack.MethodInvite, Headers: map[string]string{"Call-ID": "x", "CSeq": "1 INVITE"}}
	if err := mgr.RegisterPendingInviteServer(noBranch); err == nil {
		t.Error("missing branch must error")
	}
	// CSeq 0
	bad := mustParse(t, strings.Join([]string{
		"INVITE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKy",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>",
		"Call-ID: cidp",
		"CSeq: 0 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if err := mgr.RegisterPendingInviteServer(bad); err == nil {
		t.Error("CSeq 0 must error")
	}
}

func TestHandleCancelRequest_HappyPathAndNegativeBranches(t *testing.T) {
	mgr := NewManager()
	inv := inviteForAck(t)
	if err := mgr.RegisterPendingInviteServer(inv); err != nil {
		t.Fatalf("RegisterPendingInviteServer: %v", err)
	}

	// Build a CANCEL with matching Call-ID and CSeq number
	cancel := mustParse(t, strings.Join([]string{
		"CANCEL sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKack-build", // same branch
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>",
		"Call-ID: cid-ack-build",
		"CSeq: 1 CANCEL",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))

	var sent int
	send := SendFunc(func(*stack.Message, *net.UDPAddr) error { sent++; return nil })
	if !mgr.HandleCancelRequest(cancel, nil, send) {
		t.Error("CANCEL must be matched and 200 OK sent")
	}
	if sent != 1 {
		t.Errorf("send count = %d", sent)
	}

	// Negative branches
	if mgr.HandleCancelRequest(nil, nil, send) {
		t.Error("nil cancel must be false")
	}
	bye := byeRequest(t)
	if mgr.HandleCancelRequest(bye, nil, send) {
		t.Error("BYE must not match CANCEL")
	}
	if mgr.HandleCancelRequest(cancel, nil, nil) {
		t.Error("nil send must be false")
	}

	// CANCEL with no matching pending invite
	other := mustParse(t, strings.Join([]string{
		"CANCEL sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKzzz",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>",
		"Call-ID: NO-PENDING",
		"CSeq: 1 CANCEL",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	if mgr.HandleCancelRequest(other, nil, send) {
		t.Error("unmatched Call-ID must be false")
	}
}

func TestClearPendingInviteServer_NilSafe(t *testing.T) {
	var m *Manager
	m.ClearPendingInviteServer("anything") // must not panic
	mgr := NewManager()
	mgr.ClearPendingInviteServer("nonexistent") // must not panic
}

func TestManager_HandleResponse_DispatchesToNonInviteClient(t *testing.T) {
	mgr := NewManager()

	// Manually register a tx and feed a final via HandleResponse.
	tx := &nonInviteClientTx{
		key:     nonInviteClientKey("z9hG4bKfake", "cid-fake", 7),
		ctx:     context.Background(),
		stopCh:  make(chan struct{}),
		finalCh: make(chan *stack.Message, 1),
	}
	mgr.registerNonInviteClientTx(tx.key, tx)
	defer mgr.unregisterNonInviteClientTx(tx.key)

	resp := mustParse(t, strings.Join([]string{
		"SIP/2.0 200 OK",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKfake",
		"From: <sip:a@b>;tag=1",
		"To: <sip:x@y>;tag=2",
		"Call-ID: cid-fake",
		"CSeq: 7 BYE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))

	if !mgr.HandleResponse(resp, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1}) {
		t.Error("HandleResponse should dispatch a registered final response")
	}
}
