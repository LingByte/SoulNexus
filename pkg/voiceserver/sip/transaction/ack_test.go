package transaction

import (
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

func TestBuildAckForInvite_2xx(t *testing.T) {
	invRaw := strings.Join([]string{
		"INVITE sip:callee@example.com SIP/2.0",
		"Via: SIP/2.0/UDP 192.168.1.1;branch=z9hG4bKorig",
		"From: <sip:a@b>;tag=ftag",
		"To: <sip:callee@example.com>",
		"Call-ID: cid-1",
		"CSeq: 7 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	inv, err := stack.Parse(invRaw)
	if err != nil {
		t.Fatal(err)
	}
	finalRaw := strings.Join([]string{
		"SIP/2.0 200 OK",
		"Via: SIP/2.0/UDP 192.168.1.1;branch=z9hG4bKorig",
		"From: <sip:a@b>;tag=ftag",
		"To: <sip:callee@example.com>;tag=remote",
		"Call-ID: cid-1",
		"CSeq: 7 INVITE",
		"Contact: <sip:edge@gw:5060;transport=udp>",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	final, err := stack.Parse(finalRaw)
	if err != nil {
		t.Fatal(err)
	}
	uri := AckRequestURIFor2xx(final, inv.RequestURI)
	ack, err := BuildAckForInvite(inv, final, uri)
	if err != nil {
		t.Fatal(err)
	}
	if ack.Method != stack.MethodAck {
		t.Fatalf("method %q", ack.Method)
	}
	if stack.ParseCSeqNum(ack.GetHeader("CSeq")) != 7 || !strings.Contains(ack.GetHeader("CSeq"), "ACK") {
		t.Fatalf("cseq %q", ack.GetHeader("CSeq"))
	}
}
