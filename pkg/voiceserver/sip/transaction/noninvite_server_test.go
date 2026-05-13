package transaction

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

func TestNonInviteServer_Retransmit(t *testing.T) {
	mgr := NewManager()
	raw := strings.Join([]string{
		"OPTIONS sip:x SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.2;branch=z9hG4bKopt",
		"From: <sip:a@b>",
		"To: <sip:x>",
		"Call-ID: opt-c",
		"CSeq: 2 OPTIONS",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	req, err := stack.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	final := &stack.Message{
		IsRequest: false, Version: "SIP/2.0", StatusCode: 200, StatusText: "OK",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{},
	}
	final.SetHeader("Via", TopVia(req))
	final.SetHeader("From", req.GetHeader("From"))
	final.SetHeader("To", req.GetHeader("To"))
	final.SetHeader("Call-ID", req.GetHeader("Call-ID"))
	final.SetHeader("CSeq", req.GetHeader("CSeq"))
	final.SetHeader("Content-Length", "0")

	var sends atomic.Int32
	send := func(*stack.Message, *net.UDPAddr) error {
		sends.Add(1)
		return nil
	}
	if err := mgr.BeginNonInviteServer(context.Background(), req, &net.UDPAddr{}, final, send); err != nil {
		t.Fatal(err)
	}
	if !mgr.HandleNonInviteRequest(req, &net.UDPAddr{}) {
		t.Fatal("expected duplicate OPTIONS")
	}
	if sends.Load() < 1 {
		t.Fatalf("sends=%d", sends.Load())
	}
}
