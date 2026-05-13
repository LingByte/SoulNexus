package uas

import (
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

func reqOPTIONS() *stack.Message {
	raw := strings.Join([]string{
		"OPTIONS sip:u@h SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK1",
		"From: <sip:a@b>;tag=1",
		"To: <sip:a@b>",
		"Call-ID: cid1",
		"CSeq: 1 OPTIONS",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	m, err := stack.Parse(raw)
	if err != nil {
		panic(err)
	}
	return m
}

func TestNewResponse_Basic(t *testing.T) {
	req := reqOPTIONS()
	resp, err := NewResponse(req, 200, "OK", "line1\nline2", "application/sdp")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || resp.Body == "" {
		t.Fatalf("%+v", resp)
	}
	if resp.GetHeader("Call-ID") != "cid1" {
		t.Fatal(resp.GetHeader("Call-ID"))
	}
	if resp.GetHeader("Content-Type") != "application/sdp" {
		t.Fatal(resp.GetHeader("Content-Type"))
	}
}

func TestNewResponse_Invalid(t *testing.T) {
	if _, err := NewResponse(nil, 200, "OK", "", ""); err == nil {
		t.Fatal("nil req")
	}
	resp, _ := stack.Parse("SIP/2.0 100 Trying\r\n\r\n")
	if _, err := NewResponse(resp, 200, "OK", "", ""); err == nil {
		t.Fatal("response not request")
	}
	if _, err := NewResponse(reqOPTIONS(), 99, "x", "", ""); err == nil {
		t.Fatal("bad status")
	}
	if _, err := NewResponse(reqOPTIONS(), 700, "x", "", ""); err == nil {
		t.Fatal("bad status high")
	}
}

func TestErrorResponse_DefaultReasons(t *testing.T) {
	req := reqOPTIONS()
	for _, tc := range []struct {
		code   int
		substr string
	}{
		{400, "Bad Request"},
		{404, "Not Found"},
		{503, "Service Unavailable"},
	} {
		resp, err := ErrorResponse(req, tc.code, "")
		if err != nil || resp.StatusCode != tc.code || !strings.Contains(resp.StatusText, tc.substr) {
			t.Fatalf("%d: %+v %v", tc.code, resp, err)
		}
	}
	r, err := ErrorResponse(req, 599, "")
	if err != nil || r.StatusText != "Error" {
		t.Fatalf("%+v", r)
	}
}
