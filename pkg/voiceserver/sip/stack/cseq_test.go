package stack

import (
	"bufio"
	"net"
	"strings"
	"sync/atomic"
	"testing"
)

func TestParseCSeqNum(t *testing.T) {
	if n := ParseCSeqNum("42 INVITE"); n != 42 {
		t.Fatalf("got %d", n)
	}
	if n := ParseCSeqNum(""); n != 0 {
		t.Fatalf("empty: got %d", n)
	}
}

func TestWithCSeqACK(t *testing.T) {
	if s := WithCSeqACK(3); s != "3 ACK" {
		t.Fatalf("got %q", s)
	}
	if s := WithCSeqACK(0); s != "1 ACK" {
		t.Fatalf("got %q", s)
	}
}

func TestParse_EmptyRejected(t *testing.T) {
	if _, err := Parse(""); err == nil {
		t.Fatal("expected error")
	}
	if _, err := Parse("   "); err == nil {
		t.Fatal("expected error")
	}
}

func TestBodyBytesLen(t *testing.T) {
	if BodyBytesLen("a\nb") != len([]byte("a\r\nb")) {
		t.Fatalf("got %d", BodyBytesLen("a\nb"))
	}
	if BodyBytesLen("") != 0 {
		t.Fatal("empty")
	}
}

func TestMessage_AddHeader(t *testing.T) {
	m := &Message{Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	m.AddHeader("Via", "one")
	m.AddHeader("Via", "two")
	if len(m.GetHeaders("via")) != 2 {
		t.Fatalf("via: %#v", m.GetHeaders("Via"))
	}
}

func TestReadMessage_WithBody(t *testing.T) {
	raw := "INVITE sip:a SIP/2.0\r\nContent-Length: 5\r\n\r\nhello"
	br := bufio.NewReader(strings.NewReader(raw))
	msg, err := ReadMessage(br)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Body != "hello" {
		t.Fatalf("body %q", msg.Body)
	}
}

func TestUDPTransport_LocalAddr(t *testing.T) {
	addr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		t.Fatal(err)
	}
	tr := NewUDPTransport(conn)
	if tr.LocalAddr() == nil {
		conn.Close()
		t.Fatal("nil addr")
	}
	if !strings.Contains(tr.String(), "UDP") {
		conn.Close()
		t.Fatal(tr.String())
	}
	_ = tr.Close()
}

func TestEndpoint_DispatchAndNotify(t *testing.T) {
	ep := NewEndpoint(EndpointConfig{Host: "127.0.0.1", Port: 0})
	if err := ep.Open(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ep.Close() }()

	ep.RegisterHandler(MethodOptions, func(msg *Message, _ *net.UDPAddr) *Message {
		return &Message{IsRequest: false, Version: "SIP/2.0", StatusCode: 200, StatusText: "OK"}
	})
	ep.SetNoRouteHandler(func(_ *Message, _ *net.UDPAddr) *Message {
		return &Message{IsRequest: false, Version: "SIP/2.0", StatusCode: 404, StatusText: "Nope"}
	})

	req, _ := Parse("OPTIONS sip:x SIP/2.0\r\nContent-Length: 0\r\n\r\n")
	if resp := ep.DispatchRequest(req, nil); resp == nil || resp.StatusCode != 200 {
		t.Fatalf("dispatch options: %#v", resp)
	}

	unk, _ := Parse("REGISTER sip:r SIP/2.0\r\nContent-Length: 0\r\n\r\n")
	if resp := ep.DispatchRequest(unk, nil); resp == nil || resp.StatusCode != 404 {
		t.Fatalf("no-route: %#v", resp)
	}

	var saw atomic.Bool
	ep2 := NewEndpoint(EndpointConfig{
		Host: "127.0.0.1",
		Port: 0,
		OnSIPResponse: func(resp *Message, _ *net.UDPAddr) {
			if resp != nil && resp.StatusCode == 180 {
				saw.Store(true)
			}
		},
	})
	ep2.InvokeOnSIPResponse(&Message{StatusCode: 180}, nil)
	if !saw.Load() {
		t.Fatal("InvokeOnSIPResponse")
	}
}
