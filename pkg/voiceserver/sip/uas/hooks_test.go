package uas

import (
	"net"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

func TestHandlers_AttachNilEndpoint(t *testing.T) {
	var h Handlers
	if err := h.Attach(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultOptions(t *testing.T) {
	raw := strings.Join([]string{
		"OPTIONS sip:u@127.0.0.1 SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.2;branch=z9hG4bKo",
		"From: <sip:a@b>",
		"To: <sip:a@b>",
		"Call-ID: opt-1",
		"CSeq: 1 OPTIONS",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	req, err := stack.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := defaultOptions(req, &net.UDPAddr{})
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("resp=%v err=%v", resp, err)
	}
	if resp.GetHeader("Allow") == "" || resp.GetHeader("Accept") == "" {
		t.Fatalf("headers: %+v", resp.Headers)
	}
}

func TestNewResponse_WithBody(t *testing.T) {
	raw := "INVITE sip:a@b SIP/2.0\r\nVia: x\r\nFrom: f\r\nTo: t\r\nCall-ID: 1\r\nCSeq: 1 INVITE\r\nContent-Length: 0\r\n\r\n"
	req, err := stack.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	body := "v=0\r\no=- 0 0 IN IP4 1.1.1.1\r\n"
	resp, err := NewResponse(req, 200, "OK", body, "application/sdp")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || resp.Body == "" {
		t.Fatalf("%+v", resp)
	}
}
