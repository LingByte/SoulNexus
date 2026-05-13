package transaction

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

func uasInvite(t *testing.T) *stack.Message {
	t.Helper()
	raw := strings.Join([]string{
		"INVITE sip:callee@example.com SIP/2.0",
		"Via: SIP/2.0/UDP 192.168.1.10:5060;branch=z9hG4bKuas1",
		"From: <sip:caller@local>;tag=fromtag",
		"To: <sip:callee@example.com>",
		"Call-ID: uas-call-1",
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

func uasFinal(inv *stack.Message, status int, reason string) *stack.Message {
	resp := &stack.Message{
		IsRequest:    false,
		Version:      "SIP/2.0",
		StatusCode:   status,
		StatusText:   reason,
		Headers:      map[string]string{},
		HeadersMulti: map[string][]string{},
	}
	resp.SetHeader("Via", TopVia(inv))
	resp.SetHeader("From", inv.GetHeader("From"))
	resp.SetHeader("To", inv.GetHeader("To")+";tag=rem")
	resp.SetHeader("Call-ID", inv.GetHeader("Call-ID"))
	resp.SetHeader("CSeq", inv.GetHeader("CSeq"))
	resp.SetHeader("Content-Length", "0")
	return resp
}

func TestBeginInviteServer_DuplicateInviteRetransmitsFinal(t *testing.T) {
	mgr := NewManager()
	inv := uasInvite(t)
	remote := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5061}
	var sends atomic.Int32
	send := func(msg *stack.Message, addr *net.UDPAddr) error {
		sends.Add(1)
		return nil
	}
	final := uasFinal(inv, 486, "Busy Here")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := mgr.BeginInviteServer(ctx, inv, remote, final, send); err != nil {
		t.Fatal(err)
	}
	if !mgr.HandleInviteRequest(inv, remote) {
		t.Fatal("expected duplicate INVITE to hit server tx")
	}
	if sends.Load() < 1 {
		t.Fatalf("sends=%d", sends.Load())
	}
}

func TestBeginInviteServer_2xxTimerGAndAck(t *testing.T) {
	mgr := NewManager()
	mgr.SetT1(20 * time.Millisecond)
	mgr.SetT2(80 * time.Millisecond)

	inv := uasInvite(t)
	remote := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5062}
	var sends atomic.Int32
	send := func(msg *stack.Message, addr *net.UDPAddr) error {
		sends.Add(1)
		return nil
	}
	final := uasFinal(inv, 200, "OK")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := mgr.BeginInviteServer(ctx, inv, remote, final, send); err != nil {
		t.Fatal(err)
	}
	time.Sleep(75 * time.Millisecond)
	if sends.Load() < 2 {
		t.Fatalf("expected Timer G retransmits, sends=%d", sends.Load())
	}

	ackRaw := strings.Join([]string{
		"ACK sip:callee@example.com SIP/2.0",
		"Via: SIP/2.0/UDP 192.168.1.10:5060;branch=z9hG4bKuas1",
		"From: <sip:caller@local>;tag=fromtag",
		"To: <sip:callee@example.com>;tag=rem",
		"Call-ID: uas-call-1",
		"CSeq: 1 ACK",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	ack, err := stack.Parse(ackRaw)
	if err != nil {
		t.Fatal(err)
	}
	if !mgr.HandleAck(ack, remote) {
		t.Fatal("expected ACK to match server tx")
	}
	time.Sleep(60 * time.Millisecond)
	if mgr.HandleInviteRequest(inv, remote) {
		t.Fatalf("expected no server tx after ACK (duplicate INVITE should miss)")
	}
}
