package stack

import (
	"strings"
	"testing"
)

func TestParse_SIPRequest(t *testing.T) {
	raw := strings.Join([]string{
		"INVITE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP a.example.com:6050;branch=z9hG4bK1",
		"Via: SIP/2.0/UDP b.example.com:6050;branch=z9hG4bK2",
		"Call-Id: abc123",
		"Content-Type: application/sdp",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")

	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if msg == nil || !msg.IsRequest {
		t.Fatalf("expected request")
	}
	if msg.Method != "INVITE" || msg.RequestURI != "sip:user@domain.com" {
		t.Fatalf("first line: method=%q uri=%q", msg.Method, msg.RequestURI)
	}
	if len(msg.GetHeaders("Via")) != 2 {
		t.Fatalf("Via count: got %d", len(msg.GetHeaders("Via")))
	}
	if msg.GetHeader("Call-ID") != "abc123" {
		t.Fatalf("Call-ID: got %q", msg.GetHeader("Call-ID"))
	}
}

func TestParse_SIPResponse(t *testing.T) {
	raw := "SIP/2.0 200 OK\r\nCall-ID: x\r\nCSeq: 1 INVITE\r\nContent-Length: 0\r\n\r\n"
	msg, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if msg.IsRequest || msg.StatusCode != 200 || msg.StatusText != "OK" {
		t.Fatalf("response fields: %#v", msg)
	}
}

func TestParse_InvalidStatusLine(t *testing.T) {
	_, err := Parse("SIP/2.0 abc OK\r\n\r\n")
	if err == nil {
		t.Fatal("expected error for non-numeric status")
	}
}

func TestMessage_StringRoundTrip(t *testing.T) {
	raw := strings.Join([]string{
		"OPTIONS sip:u@h SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK77",
		"Call-ID: cid",
		"CSeq: 1 OPTIONS",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := msg.String()
	msg2, err := Parse(strings.ReplaceAll(out, "\r\n", "\n"))
	if err != nil {
		t.Fatalf("re-parse: %v\nout=%q", err, out)
	}
	if msg2.Method != "OPTIONS" || msg2.GetHeader("Call-ID") != "cid" {
		t.Fatalf("round trip mismatch: %#v", msg2)
	}
}

func TestIsSignalingNoiseDatagram(t *testing.T) {
	if !IsSignalingNoiseDatagram([]byte("\r\n\r\n")) {
		t.Fatal("expected noise")
	}
	if IsSignalingNoiseDatagram([]byte("INVITE x SIP/2.0")) {
		t.Fatal("expected not noise")
	}
}

func TestParseRAck(t *testing.T) {
	rseq, cseq, method, err := ParseRAck("1 314159 INVITE")
	if err != nil || rseq != 1 || cseq != 314159 || method != "INVITE" {
		t.Fatalf("ParseRAck: rseq=%d cseq=%d method=%q err=%v", rseq, cseq, method, err)
	}
	if _, _, _, err = ParseRAck(""); err == nil {
		t.Fatal("expected error")
	}
}
