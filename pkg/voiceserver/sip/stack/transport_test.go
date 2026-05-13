package stack

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestUDPTransport_ReadWriteRoundTrip(t *testing.T) {
	a, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	tx := NewUDPTransport(a)
	rx := NewUDPTransport(b)
	peerB := b.LocalAddr().(*net.UDPAddr)
	payload := []byte("ping")
	if _, err := tx.WriteTo(context.Background(), payload, peerB); err != nil {
		t.Fatal(err)
	}

	_ = b.SetReadDeadline(time.Now().Add(time.Second))
	buf := make([]byte, 64)
	n, from, err := rx.ReadFrom(context.Background(), buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "ping" {
		t.Fatalf("payload %q", buf[:n])
	}
	if from == nil {
		t.Fatal("nil from")
	}
}

func TestUDPTransport_ReadFromCancelledContext(t *testing.T) {
	a, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	tr := NewUDPTransport(a)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = tr.ReadFrom(ctx, make([]byte, 64))
	if err != context.Canceled {
		t.Fatalf("got %v", err)
	}
}

func TestParse_InvalidRequestLine(t *testing.T) {
	if _, err := Parse("FOO\r\n\r\n"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := Parse("INVITE\r\n\r\n"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParse_InvalidResponseLine(t *testing.T) {
	if _, err := Parse("SIP/2.0 xyz OK\r\n\r\n"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := Parse("SIP/2.0\r\n\r\n"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParse_ResponseMissingParts(t *testing.T) {
	msg, err := Parse("SIP/2.0 404\r\nContent-Length: 0\r\n\r\n")
	if err != nil || msg.StatusCode != 404 {
		t.Fatalf("%v %#v", err, msg)
	}
}

func TestMessageNilHeaders(t *testing.T) {
	var m *Message
	if m.GetHeader("x") != "" || len(m.GetHeaders("x")) != 0 {
		t.Fatal()
	}
	m = &Message{}
	m.SetHeader("k", "v")
	if m.GetHeader("k") != "v" {
		t.Fatal()
	}
}

func TestPrettyHeaderNameUnknown(t *testing.T) {
	raw := strings.Join([]string{
		"OPTIONS sip:x SIP/2.0",
		"x-custom-header: abc",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := msg.String()
	if !strings.Contains(out, "X-Custom-Header") {
		t.Fatalf("%s", out)
	}
}

func TestEndpointNilOps(t *testing.T) {
	var ep *Endpoint
	ep.RegisterHandler("INVITE", nil)
	ep.SetNoRouteHandler(nil)
	if ep.DispatchRequest(&Message{Method: "INVITE"}, nil) != nil {
		t.Fatal()
	}
	ep.InvokeOnSIPResponse(&Message{}, nil)
	if err := ep.Open(); err == nil {
		t.Fatal("expected error")
	}
	if ep.Transport() != nil {
		t.Fatal()
	}
	if ep.ListenAddr() != nil {
		t.Fatal()
	}
	if err := ep.Send(nil, nil); err == nil {
		t.Fatal()
	}
	if err := ep.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ep.Serve(nil); err == nil {
		t.Fatal()
	}
	ep.AppendOnResponseSent(nil)
}
