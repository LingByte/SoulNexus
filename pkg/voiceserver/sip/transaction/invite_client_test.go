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

func testInvite(t *testing.T) *stack.Message {
	t.Helper()
	raw := strings.Join([]string{
		"INVITE sip:callee@example.com SIP/2.0",
		"Via: SIP/2.0/UDP 192.168.1.10:5060;branch=z9hG4bKunit-test",
		"From: <sip:caller@local>;tag=fromtag",
		"To: <sip:callee@example.com>",
		"Call-ID: call-unit-1",
		"CSeq: 1 INVITE",
		"Content-Type: application/sdp",
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

func testResponse(inv *stack.Message, status int, reason string) *stack.Message {
	via := TopVia(inv)
	callID := inv.GetHeader("Call-ID")
	cseq := inv.GetHeader("CSeq")
	resp := &stack.Message{
		IsRequest:    false,
		Version:      "SIP/2.0",
		StatusCode:   status,
		StatusText:   reason,
		Headers:      map[string]string{},
		HeadersMulti: map[string][]string{},
	}
	resp.SetHeader("Via", via)
	resp.SetHeader("From", inv.GetHeader("From"))
	resp.SetHeader("To", inv.GetHeader("To")+";tag=remote")
	resp.SetHeader("Call-ID", callID)
	resp.SetHeader("CSeq", cseq)
	resp.SetHeader("Content-Length", "0")
	return resp
}

func TestRunInviteClient_ProvisionalThenFinal(t *testing.T) {
	mgr := NewManager()
	mgr.SetT1(25 * time.Millisecond)

	inv := testInvite(t)
	remote := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9}

	var sends atomic.Int32
	send := func(msg *stack.Message, addr *net.UDPAddr) error {
		sends.Add(1)
		return nil
	}

	var prov atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		time.Sleep(80 * time.Millisecond)
		mgr.HandleResponse(testResponse(inv, 180, "Ringing"), remote)
		time.Sleep(5 * time.Millisecond)
		mgr.HandleResponse(testResponse(inv, 200, "OK"), remote)
	}()

	res, err := mgr.RunInviteClient(ctx, inv, remote, send, func(*stack.Message) {
		prov.Add(1)
	})
	if err != nil {
		t.Fatal(err)
	}
	final := res.Final
	if final.StatusCode != 200 {
		t.Fatalf("status=%d", final.StatusCode)
	}
	if res.Remote == nil {
		t.Fatal("nil Remote")
	}
	if prov.Load() != 1 {
		t.Fatalf("provisional callbacks: %d", prov.Load())
	}
	// Initial send + at least one retransmit before 180 at 80ms with T1=25ms.
	if n := sends.Load(); n < 2 {
		t.Fatalf("expected >=2 INVITE sends, got %d", n)
	}
}

func TestRunInviteClient_FinalWithoutProvisional(t *testing.T) {
	mgr := NewManager()
	mgr.SetT1(30 * time.Millisecond)
	inv := testInvite(t)
	remote := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9}
	var sends atomic.Int32
	send := func(msg *stack.Message, addr *net.UDPAddr) error {
		sends.Add(1)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		time.Sleep(15 * time.Millisecond)
		mgr.HandleResponse(testResponse(inv, 486, "Busy Here"), remote)
	}()

	res, err := mgr.RunInviteClient(ctx, inv, remote, send, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Final.StatusCode != 486 {
		t.Fatalf("got %d", res.Final.StatusCode)
	}
	if sends.Load() < 1 {
		t.Fatalf("sends=%d", sends.Load())
	}
}
