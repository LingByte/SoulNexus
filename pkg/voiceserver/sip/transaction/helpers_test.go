package transaction

import (
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

func TestBranchParam(t *testing.T) {
	v := "SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bKabc123;rport"
	if b := BranchParam(v); b != "z9hG4bKabc123" {
		t.Fatalf("got %q", b)
	}
}

func TestNonInviteServerKey(t *testing.T) {
	raw := "OPTIONS sip:x SIP/2.0\r\nVia: SIP/2.0/UDP 1.1.1.1;branch=z9hG4bKa\r\nFrom: f\r\nTo: t\r\nCall-ID: c\r\nCSeq: 9 OPTIONS\r\nContent-Length: 0\r\n\r\n"
	m, err := stack.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if NonInviteServerKey(m) == "" {
		t.Fatal("empty key")
	}
}

func TestIsAckCSeq(t *testing.T) {
	raw := "ACK sip:a SIP/2.0\r\nCSeq: 2 ACK\r\n\r\n"
	m, err := stack.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !IsAckCSeq(m) {
		t.Fatal("expected ACK cseq")
	}
}

func TestTopBranch(t *testing.T) {
	raw := strings.Join([]string{
		"SIP/2.0 100 Trying",
		"Via: SIP/2.0/UDP 1.1.1.1;branch=z9hG4bKx",
		"Call-ID: a",
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	m, err := stack.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if TopBranch(m) != "z9hG4bKx" {
		t.Fatalf("got %q", TopBranch(m))
	}
}
