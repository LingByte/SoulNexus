package transaction

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

func TestHandleCancelRequest(t *testing.T) {
	mgr := NewManager()
	inv := strings.Join([]string{
		"INVITE sip:x SIP/2.0",
		"Via: SIP/2.0/UDP 1.1.1.1;branch=z9hG4bKp",
		"From: <sip:a@b>",
		"To: <sip:x>",
		"Call-ID: cc-1",
		"CSeq: 5 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	invMsg, err := stack.Parse(inv)
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.RegisterPendingInviteServer(invMsg); err != nil {
		t.Fatal(err)
	}
	cancel := strings.Join([]string{
		"CANCEL sip:x SIP/2.0",
		"Via: SIP/2.0/UDP 1.1.1.1;branch=z9hG4bKp2",
		"From: <sip:a@b>",
		"To: <sip:x>",
		"Call-ID: cc-1",
		"CSeq: 5 CANCEL",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	cMsg, err := stack.Parse(cancel)
	if err != nil {
		t.Fatal(err)
	}
	var sends atomic.Int32
	send := func(msg *stack.Message, addr *net.UDPAddr) error {
		sends.Add(1)
		return nil
	}
	if !mgr.HandleCancelRequest(cMsg, &net.UDPAddr{}, send) {
		t.Fatal("expected cancel handled")
	}
	if sends.Load() != 1 {
		t.Fatalf("sends=%d", sends.Load())
	}
}

func TestRegisterPendingClearedByBeginInviteServer(t *testing.T) {
	mgr := NewManager()
	inv, _ := stack.Parse("INVITE sip:x SIP/2.0\r\nVia: SIP/2.0/UDP 1.1.1.1;branch=z9hG4bKz\r\nFrom: f\r\nTo: t\r\nCall-ID: x1\r\nCSeq: 1 INVITE\r\n\r\n")
	_ = mgr.RegisterPendingInviteServer(inv)
	final, _ := stack.Parse("SIP/2.0 200 OK\r\nVia: SIP/2.0/UDP 1.1.1.1;branch=z9hG4bKz\r\nFrom: f\r\nTo: t\r\nCall-ID: x1\r\nCSeq: 1 INVITE\r\nContent-Length: 0\r\n\r\n")
	send := func(*stack.Message, *net.UDPAddr) error { return nil }
	if err := mgr.BeginInviteServer(context.Background(), inv, &net.UDPAddr{}, final, send); err != nil {
		t.Fatal(err)
	}
	cancel, _ := stack.Parse("CANCEL sip:x SIP/2.0\r\nVia: x\r\nFrom: f\r\nTo: t\r\nCall-ID: x1\r\nCSeq: 1 CANCEL\r\n\r\n")
	if mgr.HandleCancelRequest(cancel, &net.UDPAddr{}, send) {
		t.Fatal("pending should be cleared after BeginInviteServer")
	}
}
