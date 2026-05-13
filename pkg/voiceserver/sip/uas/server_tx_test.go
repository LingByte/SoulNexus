package uas

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
)

func TestChainInviteServerTx(t *testing.T) {
	mgr := transaction.NewManager()
	var innerCalls atomic.Int32
	inner := func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
		innerCalls.Add(1)
		return uasFinalForTx(t, req, 100, "Trying"), nil
	}
	chained := ChainInviteServerTx(mgr, inner)
	inv := uasInviteForTx(t)
	remote := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1}
	final := uasFinalForTx(t, inv, 486, "Busy")
	_ = mgr.BeginInviteServer(context.Background(), inv, remote, final, func(*stack.Message, *net.UDPAddr) error { return nil })
	if _, err := chained(inv, remote); err != nil {
		t.Fatal(err)
	}
	if innerCalls.Load() != 0 {
		t.Fatalf("inner should not run on retransmit, got %d", innerCalls.Load())
	}
}

func uasInviteForTx(t *testing.T) *stack.Message {
	t.Helper()
	raw := strings.Join([]string{
		"INVITE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKchain",
		"From: <sip:a@b>",
		"To: <sip:x@y>",
		"Call-ID: chain-1",
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	m, err := stack.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func uasFinalForTx(t *testing.T, inv *stack.Message, status int, reason string) *stack.Message {
	t.Helper()
	r := &stack.Message{
		IsRequest:    false,
		Version:      "SIP/2.0",
		StatusCode:   status,
		StatusText:   reason,
		Headers:      map[string]string{},
		HeadersMulti: map[string][]string{},
	}
	r.SetHeader("Via", transaction.TopVia(inv))
	r.SetHeader("From", inv.GetHeader("From"))
	r.SetHeader("To", inv.GetHeader("To"))
	r.SetHeader("Call-ID", inv.GetHeader("Call-ID"))
	r.SetHeader("CSeq", inv.GetHeader("CSeq"))
	r.SetHeader("Content-Length", "0")
	return r
}

func TestWithOnResponseSentAppended(t *testing.T) {
	var a, b atomic.Int32
	cfg := stack.EndpointConfig{}
	cfg = WithOnResponseSentAppended(cfg, func(*stack.Message, *stack.Message, *net.UDPAddr) { a.Add(1) })
	cfg = WithOnResponseSentAppended(cfg, func(*stack.Message, *stack.Message, *net.UDPAddr) { b.Add(1) })
	cfg.OnResponseSent(nil, nil, nil)
	if a.Load() != 1 || b.Load() != 1 {
		t.Fatalf("a=%d b=%d", a.Load(), b.Load())
	}
}

func TestChainInviteServerTx_NewInviteHitsInner(t *testing.T) {
	mgr := transaction.NewManager()
	var innerCalls atomic.Int32
	inner := func(req *stack.Message, addr *net.UDPAddr) (*stack.Message, error) {
		innerCalls.Add(1)
		return uasFinalForTx(t, req, 100, "Trying"), nil
	}
	chained := ChainInviteServerTx(mgr, inner)
	inv := uasInviteForTx(t)
	if _, err := chained(inv, &net.UDPAddr{}); err != nil {
		t.Fatal(err)
	}
	if innerCalls.Load() != 1 {
		t.Fatalf("inner calls=%d", innerCalls.Load())
	}
}
