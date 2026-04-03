package protocol

import (
	"strings"
	"testing"
)

func TestParse_SIPRequest(t *testing.T) {
	raw := strings.Join([]string{
		"INVITE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP a.example.com:5060;branch=z9hG4bK1",
		"Via: SIP/2.0/UDP b.example.com:5060;branch=z9hG4bK2",
		"Call-Id: abc123",
		"Content-Type: application/sdp",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")

	msg, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if msg == nil {
		t.Fatalf("expected non-nil message")
	}
	if !msg.IsRequest {
		t.Fatalf("expected request message")
	}
	if msg.Method != "INVITE" {
		t.Fatalf("method mismatch: got=%s", msg.Method)
	}
	if msg.RequestURI != "sip:user@domain.com" {
		t.Fatalf("uri mismatch: got=%s", msg.RequestURI)
	}
	if msg.GetHeader("Via") == "" {
		t.Fatalf("expected Via header")
	}
	if len(msg.GetHeaders("Via")) != 2 {
		t.Fatalf("expected 2 Via values, got=%d", len(msg.GetHeaders("Via")))
	}
	// Header key case-insensitive
	if msg.GetHeader("Call-ID") != "abc123" {
		t.Fatalf("Call-ID mismatch: got=%q", msg.GetHeader("Call-ID"))
	}
}

func TestParse_Invalid(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestParseSDP_Basic(t *testing.T) {
	sdp := strings.Join([]string{
		"v=0",
		"o=- 123456 123456 IN IP4 192.168.1.100",
		"s=Session",
		"c=IN IP4 192.168.1.100",
		"t=0 0",
		"m=audio 49170 RTP/AVP 0",
		"a=rtpmap:0 PCMU/8000",
	}, "\r\n")

	info, err := ParseSDP(sdp)
	if err != nil {
		t.Fatalf("ParseSDP failed: %v", err)
	}
	if info.IP != "192.168.1.100" {
		t.Fatalf("ip mismatch: got=%s", info.IP)
	}
	if info.Port != 49170 {
		t.Fatalf("port mismatch: got=%d", info.Port)
	}
	if len(info.Codecs) != 1 {
		t.Fatalf("expected 1 codec, got=%d", len(info.Codecs))
	}
	if info.Codecs[0].PayloadType != 0 {
		t.Fatalf("payload type mismatch: got=%d", info.Codecs[0].PayloadType)
	}
	if info.Codecs[0].Name != "pcmu" {
		t.Fatalf("codec name mismatch: got=%s", info.Codecs[0].Name)
	}
	if info.Codecs[0].ClockRate != 8000 {
		t.Fatalf("clock mismatch: got=%d", info.Codecs[0].ClockRate)
	}
}

func TestGenerateSDP_RoundTrip_CanParse(t *testing.T) {
	codecs := []SDPCodec{
		{PayloadType: 0, Name: "pcmu", ClockRate: 8000},
		{PayloadType: 8, Name: "pcma", ClockRate: 8000},
	}
	body := GenerateSDP("127.0.0.1", 5004, codecs)
	info, err := ParseSDP(body)
	if err != nil {
		t.Fatalf("ParseSDP failed: %v", err)
	}
	if info.IP != "127.0.0.1" {
		t.Fatalf("ip mismatch: got=%s", info.IP)
	}
	if info.Port != 5004 {
		t.Fatalf("port mismatch: got=%d", info.Port)
	}
	if len(info.Codecs) == 0 {
		t.Fatalf("expected codecs")
	}
}

