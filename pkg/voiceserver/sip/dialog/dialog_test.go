package dialog

import (
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
)

func TestTagFromHeader(t *testing.T) {
	if g := TagFromHeader(`<sip:a@b>;tag=abc7`); g != "abc7" {
		t.Fatalf("got %q", g)
	}
}

func TestNewUASFromINVITE_AndMatchACK(t *testing.T) {
	raw := strings.Join([]string{
		"INVITE sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKd1",
		"From: <sip:a@b>;tag=rem1",
		"To: <sip:x@y>",
		"Call-ID: cid-d",
		"CSeq: 3 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	inv, err := stack.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewUASFromINVITE(inv)
	if err != nil {
		t.Fatal(err)
	}
	if d.InviteTransactionKey() != transaction.InviteTransactionKey(transaction.TopBranch(inv), "cid-d") {
		t.Fatal("tx key mismatch")
	}
	d.SetLocalTag("loc1")
	ackRaw := strings.Join([]string{
		"ACK sip:x@y SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKd1",
		"From: <sip:a@b>;tag=rem1",
		"To: <sip:x@y>;tag=loc1",
		"Call-ID: cid-d",
		"CSeq: 3 ACK",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	ack, err := stack.Parse(ackRaw)
	if err != nil {
		t.Fatal(err)
	}
	if !d.MatchACK(ack) {
		t.Fatal("expected ACK match")
	}
}
