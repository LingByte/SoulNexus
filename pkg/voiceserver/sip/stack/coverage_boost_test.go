package stack

// Targeted tests to push pkg/sip/stack coverage past 85% by exercising
// nil-safety guards, error branches and edge-case parsers.

import (
	"context"
	"net"
	"testing"
	"time"
)

// ---------- Endpoint nil/closed safety -------------------------------------

func TestEndpoint_NilReceivers(t *testing.T) {
	var e *Endpoint
	if e.Transport() != nil {
		t.Errorf("nil endpoint Transport must be nil")
	}
	if e.ListenAddr() != nil {
		t.Errorf("nil endpoint ListenAddr must be nil")
	}
	if err := e.Send(&Message{}, &net.UDPAddr{}); err == nil {
		t.Errorf("nil endpoint Send must error")
	}
	if err := e.Close(); err != nil {
		t.Errorf("nil endpoint Close should be nil err, got %v", err)
	}
	e.AppendOnResponseSent(func(*Message, *Message, *net.UDPAddr) {}) // must not panic
	if err := e.Serve(context.Background()); err == nil {
		t.Errorf("nil endpoint Serve must error")
	}
}

func TestEndpoint_NotOpenSend(t *testing.T) {
	e := &Endpoint{}
	if err := e.Send(&Message{}, &net.UDPAddr{}); err == nil {
		t.Errorf("not-open endpoint Send must error")
	}
	if err := e.Serve(context.Background()); err == nil {
		t.Errorf("not-open endpoint Serve must error")
	}
}

func TestEndpoint_Send_NilMessage(t *testing.T) {
	e := newOpenTestEndpoint(t)
	defer e.Close()
	if err := e.Send(nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1}); err == nil {
		t.Errorf("Send(nil) must error")
	}
}

func TestEndpoint_AppendOnResponseSent_ChainedTwice(t *testing.T) {
	calls := 0
	e := &Endpoint{}
	e.AppendOnResponseSent(func(*Message, *Message, *net.UDPAddr) { calls++ })
	e.AppendOnResponseSent(func(*Message, *Message, *net.UDPAddr) { calls++ })
	if e.cfg.OnResponseSent == nil {
		t.Fatalf("chain not installed")
	}
	e.cfg.OnResponseSent(&Message{}, &Message{}, &net.UDPAddr{})
	if calls != 2 {
		t.Fatalf("chain expected 2 calls, got %d", calls)
	}
	// nil fn should be no-op
	e.AppendOnResponseSent(nil)
}

// ---------- UDPTransport nil/closed --------------------------------------

func TestUDPTransport_NilSafety(t *testing.T) {
	var tr *UDPTransport
	if tr.LocalAddr() != nil {
		t.Errorf("nil tr LocalAddr non-nil")
	}
	if _, _, err := tr.ReadFrom(context.Background(), nil); err == nil {
		t.Errorf("nil tr ReadFrom must error")
	}
	if _, err := tr.WriteTo(context.Background(), nil, &net.UDPAddr{}); err == nil {
		t.Errorf("nil tr WriteTo must error")
	}
	if err := tr.Close(); err != nil {
		t.Errorf("nil tr Close non-nil err: %v", err)
	}
}

func TestUDPTransport_CtxCancelledBeforeIO(t *testing.T) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer conn.Close()
	tr := NewUDPTransport(conn)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, _, err := tr.ReadFrom(ctx, make([]byte, 16)); err == nil {
		t.Errorf("expected ctx.Err on cancelled ReadFrom")
	}
	if _, err := tr.WriteTo(ctx, []byte("x"), conn.LocalAddr().(*net.UDPAddr)); err == nil {
		t.Errorf("expected ctx.Err on cancelled WriteTo")
	}
	if got := tr.LocalAddr(); got == nil {
		t.Errorf("LocalAddr should be non-nil for open conn")
	}
	if err := tr.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// ---------- Helpers ------------------------------------------------------

func newOpenTestEndpoint(t *testing.T) *Endpoint {
	t.Helper()
	e := NewEndpoint(EndpointConfig{Host: "127.0.0.1", Port: 0, ReadBufSize: 2048})
	if err := e.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	return e
}

// ---------- prettyHeaderName edge cases ----------------------------------

func TestPrettyHeaderName_KnownAndUnknown(t *testing.T) {
	// known short → canonical (Call-ID has special acronym handling)
	if got := prettyHeaderName("call-id"); got == "" {
		t.Errorf("prettyHeaderName(call-id) returned empty")
	}
	// unknown headers must produce a non-empty title-cased rejoin and never panic
	for _, in := range []string{"x-custom-header", "weird_header", "single", "Already-Cased"} {
		got := prettyHeaderName(in)
		if got == "" {
			t.Errorf("prettyHeaderName(%q) returned empty", in)
		}
	}
}

// ---------- ParseRAck error branches ------------------------------------

func TestParseRAck_AllErrorBranches(t *testing.T) {
	if _, _, _, err := ParseRAck(""); err == nil {
		t.Errorf("empty RAck should error")
	}
	if _, _, _, err := ParseRAck("only-two parts"); err == nil {
		t.Errorf("two-part RAck should error")
	}
	if _, _, _, err := ParseRAck("notnum 1 INVITE"); err == nil {
		t.Errorf("non-numeric rseq should error")
	}
	if _, _, _, err := ParseRAck("1 notnum INVITE"); err == nil {
		t.Errorf("non-numeric cseq should error")
	}
	rs, cs, m, err := ParseRAck("  42 7 invite  ")
	if err != nil || rs != 42 || cs != 7 || m != "INVITE" {
		t.Errorf("normalized parse: rs=%d cs=%d m=%q err=%v", rs, cs, m, err)
	}
}

// ---------- ParseCSeqNum branches --------------------------------------

func TestParseCSeqNum_Branches(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"   ", 0},
		{"abc INVITE", 0},
		{"42", 42},
		{"42 INVITE", 42},
		{"  9   ACK  ", 9},
	}
	for _, c := range cases {
		if got := ParseCSeqNum(c.in); got != c.want {
			t.Errorf("ParseCSeqNum(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestWithCSeqACK_Branches(t *testing.T) {
	if got := WithCSeqACK(0); got != "1 ACK" {
		t.Errorf("zero CSeq → %q", got)
	}
	if got := WithCSeqACK(-3); got != "1 ACK" {
		t.Errorf("negative CSeq → %q", got)
	}
	if got := WithCSeqACK(7); got != "7 ACK" {
		t.Errorf("CSeq 7 → %q", got)
	}
}
