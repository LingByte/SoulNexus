package uas

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
)

// ---------- helpers --------------------------------------------------------

func mustParseReq(t *testing.T, raw string) *stack.Message {
	t.Helper()
	m, err := stack.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return m
}

func reqInvite(t *testing.T) *stack.Message {
	return mustParseReq(t, strings.Join([]string{
		"INVITE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKuas-cov-1",
		"From: <sip:a@b>;tag=x",
		"To: <sip:x@y>",
		"Call-ID: cid-uas-cov",
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
}

func reqBye(t *testing.T) *stack.Message {
	return mustParseReq(t, strings.Join([]string{
		"BYE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKbye-cov",
		"From: <sip:a@b>;tag=x",
		"To: <sip:x@y>;tag=y",
		"Call-ID: cid-bye-cov",
		"CSeq: 2 BYE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
}

func reqAck(t *testing.T) *stack.Message {
	return mustParseReq(t, strings.Join([]string{
		"ACK sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKack-cov",
		"From: <sip:a@b>;tag=x",
		"To: <sip:x@y>;tag=y",
		"Call-ID: cid-uas-cov",
		"CSeq: 1 ACK",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
}

// ---------- wrap* ----------------------------------------------------------

func TestWrapInvite_NilHandler(t *testing.T) {
	if got := wrapInvite(nil); got != nil {
		t.Error("nil handler must yield nil wrapper")
	}
}

func TestWrapInvite_PropagatesResponseAndErrorTo500(t *testing.T) {
	// Returns a 200 if no error
	hOK := wrapInvite(func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
		return ErrorResponse(req, 200, "OK")
	})
	resp := hOK(reqInvite(t), nil)
	if resp == nil || resp.StatusCode != 200 {
		t.Errorf("OK handler resp = %+v", resp)
	}

	// Error path → 500
	hErr := wrapInvite(func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
		return nil, errors.New("boom")
	})
	resp2 := hErr(reqInvite(t), nil)
	if resp2 == nil || resp2.StatusCode != 500 {
		t.Errorf("error handler resp = %+v", resp2)
	}
}

func TestWrapSimple_NilHandler(t *testing.T) {
	if got := wrapSimple(nil); got != nil {
		t.Error("nil simple handler must yield nil wrapper")
	}
}

func TestWrapSimple_PropagatesResponseAndError(t *testing.T) {
	hErr := wrapSimple(func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
		return nil, errors.New("boom")
	})
	resp := hErr(reqBye(t), nil)
	if resp == nil || resp.StatusCode != 500 {
		t.Errorf("simple error path = %+v", resp)
	}

	hOK := wrapSimple(func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
		return ErrorResponse(req, 200, "OK")
	})
	resp2 := hOK(reqBye(t), nil)
	if resp2 == nil || resp2.StatusCode != 200 {
		t.Errorf("simple ok path = %+v", resp2)
	}
}

func TestWrapAck_NilHandler(t *testing.T) {
	if got := wrapAck(nil); got != nil {
		t.Error("nil ack handler must yield nil wrapper")
	}
}

func TestWrapAck_ErrorReturnsNilResponse(t *testing.T) {
	called := false
	h := wrapAck(func(req *stack.Message, addr *net.UDPAddr) error {
		called = true
		return errors.New("ack err")
	})
	if r := h(reqAck(t), nil); r != nil {
		t.Errorf("ACK error path must yield nil, got %+v", r)
	}
	if !called {
		t.Error("ack handler not invoked")
	}
	hOK := wrapAck(func(req *stack.Message, addr *net.UDPAddr) error { return nil })
	if r := hOK(reqAck(t), nil); r != nil {
		t.Errorf("ACK ok path must yield nil, got %+v", r)
	}
}

// ---------- Attach branches -----------------------------------------------

func TestHandlers_Attach_AllSimpleSlotsRegistered(t *testing.T) {
	dummy := func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
		return ErrorResponse(req, 200, "OK")
	}
	ack := func(req *stack.Message, addr *net.UDPAddr) error { return nil }
	h := Handlers{
		Invite:    dummy,
		Ack:       ack,
		Bye:       dummy,
		Cancel:    dummy,
		Options:   dummy,
		Register:  dummy,
		Info:      dummy,
		Prack:     dummy,
		Subscribe: dummy,
		Notify:    dummy,
		Publish:   dummy,
		Refer:     dummy,
		Message:   dummy,
		Update:    dummy,
	}
	ep := stack.NewEndpoint(stack.EndpointConfig{Host: "127.0.0.1", Port: 0, ReadBufSize: 2048})
	if err := h.Attach(ep); err != nil {
		t.Fatalf("Attach: %v", err)
	}
}

func TestHandlers_Attach_DefaultOptionsWhenNil(t *testing.T) {
	ep := stack.NewEndpoint(stack.EndpointConfig{Host: "127.0.0.1", Port: 0, ReadBufSize: 2048})
	if err := (Handlers{}).Attach(ep); err != nil {
		t.Fatalf("Attach minimal: %v", err)
	}
}

// ---------- defaultOptions ------------------------------------------------

func TestDefaultOptions_AllowAndAcceptHeaders(t *testing.T) {
	opt := mustParseReq(t, strings.Join([]string{
		"OPTIONS sip:u@h SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.2;branch=z9hG4bKopt-cov",
		"From: <sip:a@b>;tag=1",
		"To: <sip:a@b>",
		"Call-ID: opt-cov",
		"CSeq: 1 OPTIONS",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n"))
	resp, err := defaultOptions(opt, nil)
	if err != nil {
		t.Fatalf("defaultOptions: %v", err)
	}
	if !strings.Contains(resp.GetHeader("Allow"), "INVITE") {
		t.Errorf("Allow missing INVITE: %q", resp.GetHeader("Allow"))
	}
	if resp.GetHeader("Accept") != "application/sdp" {
		t.Errorf("Accept = %q", resp.GetHeader("Accept"))
	}
}

// ---------- ErrorResponse default-reason path -----------------------------

func TestErrorResponse_EmptyReasonDefault(t *testing.T) {
	resp, err := ErrorResponse(reqBye(t), 404, "")
	if err != nil {
		t.Fatalf("ErrorResponse: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("status = %d", resp.StatusCode)
	}
	if strings.TrimSpace(resp.StatusText) == "" {
		t.Error("default reason should fill in StatusText")
	}
}

// ---------- ChainNonInviteServerTx + ChainAckServerTx --------------------

func TestChainNonInviteServerTx_NilCases(t *testing.T) {
	mgr := transaction.NewManager()
	if got := ChainNonInviteServerTx(nil, nil); got != nil {
		t.Error("both nil → nil")
	}
	inner := func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
		return ErrorResponse(req, 200, "OK")
	}
	if got := ChainNonInviteServerTx(nil, inner); got == nil {
		t.Error("nil mgr → inner unchanged, expected non-nil")
	}
	chained := ChainNonInviteServerTx(mgr, inner)
	resp, err := chained(reqBye(t), nil)
	if err != nil {
		t.Fatalf("chained: %v", err)
	}
	if resp == nil {
		t.Error("first BYE should pass through to inner and return resp")
	}
}

func TestChainAckServerTx_AllBranches(t *testing.T) {
	mgr := transaction.NewManager()

	if got := ChainAckServerTx(nil, nil); got != nil {
		t.Error("both nil → nil expected")
	}

	called := false
	innerOnly := ChainAckServerTx(nil, func(req *stack.Message, addr *net.UDPAddr) error {
		called = true
		return nil
	})
	if err := innerOnly(reqAck(t), nil); err != nil {
		t.Fatalf("innerOnly: %v", err)
	}
	if !called {
		t.Error("inner not invoked")
	}

	mgrOnly := ChainAckServerTx(mgr, nil)
	if err := mgrOnly(reqAck(t), nil); err != nil {
		t.Fatalf("mgrOnly: %v", err)
	}

	both := ChainAckServerTx(mgr, func(req *stack.Message, addr *net.UDPAddr) error { return errors.New("inner err") })
	if err := both(reqAck(t), nil); err == nil {
		t.Error("inner error must propagate")
	}
}

// ---------- AfterResponseSentBeginServerTx + AfterResponseSentBeginInviteServer

func TestAfterResponseSentBeginServerTx_GuardClauses(t *testing.T) {
	mgr := transaction.NewManager()
	send := transaction.SendFunc(func(msg *stack.Message, addr *net.UDPAddr) error { return nil })

	hook := AfterResponseSentBeginServerTx(mgr, context.Background(), send)
	// Nil request → must not panic
	hook(nil, &stack.Message{StatusCode: 200}, nil)
	// Nil response → must not panic
	hook(reqInvite(t), nil, nil)

	// Out-of-range status → return early (still no panic)
	resp1xx := &stack.Message{StatusCode: 100}
	hook(reqInvite(t), resp1xx, nil)

	// Invite final response (200) → exercises BeginInviteServer branch
	resp200, _ := ErrorResponse(reqInvite(t), 200, "OK")
	hook(reqInvite(t), resp200, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1})

	// Non-INVITE final → exercises BeginNonInviteServer branch
	bye := reqBye(t)
	bye200, _ := ErrorResponse(bye, 200, "OK")
	hook(bye, bye200, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1})

	// nil mgr branch
	hookNil := AfterResponseSentBeginServerTx(nil, nil, nil)
	hookNil(reqInvite(t), resp200, nil)

	// AfterResponseSentBeginInviteServer is a thin alias
	alias := AfterResponseSentBeginInviteServer(mgr, nil, send)
	alias(reqInvite(t), resp200, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1})
}

// ---------- WithOnResponseSentAppended chaining --------------------------

func TestWithOnResponseSentAppended_NilFnReturnsCfg(t *testing.T) {
	cfg := stack.EndpointConfig{}
	out := WithOnResponseSentAppended(cfg, nil)
	if out.OnResponseSent != nil {
		t.Error("nil fn must not install hook")
	}
}

func TestWithOnResponseSentAppended_ChainsPrevAndNew(t *testing.T) {
	prevCalled, newCalled := 0, 0
	cfg := stack.EndpointConfig{
		OnResponseSent: func(req, resp *stack.Message, addr *net.UDPAddr) { prevCalled++ },
	}
	cfg2 := WithOnResponseSentAppended(cfg, func(req, resp *stack.Message, addr *net.UDPAddr) { newCalled++ })
	cfg2.OnResponseSent(&stack.Message{}, &stack.Message{}, &net.UDPAddr{})
	if prevCalled != 1 || newCalled != 1 {
		t.Errorf("prev=%d new=%d", prevCalled, newCalled)
	}
}

// ---------- ChainInviteServerTx nil branches ----------------------------

func TestChainInviteServerTx_NilBranches(t *testing.T) {
	if got := ChainInviteServerTx(nil, nil); got != nil {
		t.Error("nil + nil → nil")
	}
	inner := func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
		return ErrorResponse(req, 200, "OK")
	}
	if got := ChainInviteServerTx(nil, inner); got == nil {
		t.Error("nil mgr → inner unchanged, expected non-nil")
	}
}
