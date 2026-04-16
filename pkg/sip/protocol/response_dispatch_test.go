package protocol

import (
	"net"
	"sync/atomic"
	"testing"
	"time"
)

func TestServer_OnSIPResponse(t *testing.T) {
	s := NewServer("127.0.0.1", 0)
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = s.Stop() }()

	var saw atomic.Bool
	s.OnSIPResponse = func(resp *Message, _ *net.UDPAddr) {
		if resp != nil && resp.StatusCode == 200 {
			saw.Store(true)
		}
	}

	raw := "SIP/2.0 200 OK\r\n" +
		"Via: SIP/2.0/UDP 127.0.0.1:9;branch=z9hG4bKtest\r\n" +
		"From: <sip:a@b>;tag=1\r\n" +
		"To: <sip:a@b>;tag=2\r\n" +
		"Call-ID: cid-1\r\n" +
		"CSeq: 1 INVITE\r\n" +
		"Content-Length: 0\r\n\r\n"

	addr, _ := net.ResolveUDPAddr("udp", s.Conn.LocalAddr().String())
	_, _ = s.Conn.WriteToUDP([]byte(raw), addr)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if saw.Load() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("OnSIPResponse not invoked for 200 OK")
}
