package stack

import (
	"bufio"
	"strings"
	"testing"
)

func TestReadMessage_ContentLengthBody(t *testing.T) {
	raw := strings.Join([]string{
		"INVITE sip:user@example.com SIP/2.0",
		"Via: SIP/2.0/UDP 127.0.0.1:9;branch=z9hG4bK1",
		"Call-ID: readmsg-1",
		"CSeq: 1 INVITE",
		"Content-Type: application/sdp",
		"Content-Length: 5",
		"",
		"hello",
	}, "\r\n")
	br := bufio.NewReader(strings.NewReader(raw))
	m, err := ReadMessage(br)
	if err != nil {
		t.Fatal(err)
	}
	if !m.IsRequest || m.Method != "INVITE" {
		t.Fatalf("request: %+v", m)
	}
	if strings.TrimSpace(m.Body) != "hello" {
		t.Fatalf("body %q", m.Body)
	}
}
