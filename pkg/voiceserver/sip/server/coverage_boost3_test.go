package server

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// ---------- runOneTCPConn via real TCP -------------------------------------

func TestRunOneTCPConn_DispatchesRequestAndWritesResponse(t *testing.T) {
	s := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1", SIPTCPListen: "127.0.0.1:0"})
	if err := s.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer s.Stop()

	// Spin up our own listener we fully control (deterministic address).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		s.runOneTCPConn(s.sigCtx, conn)
	}()

	// Dial and send OPTIONS — handleOptions answers 200.
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	opt := strings.Join([]string{
		"OPTIONS sip:server@127.0.0.1 SIP/2.0",
		"Via: SIP/2.0/TCP 127.0.0.1:1234;branch=z9hG4bK-tcp-opt;rport",
		"Max-Forwards: 70",
		"From: <sip:ping@example.com>;tag=p",
		"To: <sip:server@127.0.0.1>",
		"Call-ID: tcp-opt-1",
		"CSeq: 1 OPTIONS",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	if _, err := conn.Write([]byte(opt)); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	body := string(buf[:n])
	if !strings.HasPrefix(body, "SIP/2.0 200") {
		t.Errorf("expected 200 OK response, got: %q", body[:30])
	}

	// Closing the client side should make runOneTCPConn return.
	conn.Close()
	select {
	case <-serverDone:
	case <-time.After(3 * time.Second):
		t.Fatal("runOneTCPConn did not exit after client close")
	}
}

// ---------- handleCancel: 481 when no matching pending INVITE --------------

func TestHandleCancel_No481WhenUnknown(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	cancel := &stack.Message{IsRequest: true, Method: stack.MethodCancel, RequestURI: "sip:target@127.0.0.1",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	cancel.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1:5060;branch=z9hG4bK-cx-1")
	cancel.SetHeader("From", "<sip:a@example.com>;tag=1")
	cancel.SetHeader("To", "<sip:target@127.0.0.1>")
	cancel.SetHeader("Call-ID", "never-seen")
	cancel.SetHeader("CSeq", "1 CANCEL")

	resp := s.handleCancel(cancel, &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5060})
	if resp == nil {
		t.Fatal("handleCancel should return 481 when unknown")
	}
	if resp.StatusCode != 481 {
		t.Errorf("status=%d want 481", resp.StatusCode)
	}
}

func TestHandleCancel_WrongMethodNil(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	if resp := s.handleCancel(nil, nil); resp != nil {
		t.Error("nil msg should return nil")
	}
	ok := &stack.Message{IsRequest: false, Method: ""}
	if resp := s.handleCancel(ok, nil); resp != nil {
		t.Error("non-request should return nil")
	}
	wrong := &stack.Message{IsRequest: true, Method: "INVITE"}
	if resp := s.handleCancel(wrong, nil); resp != nil {
		t.Error("wrong method should return nil")
	}
}

// ---------- listenTCP with bad address logs and returns ---------------------

func TestListenTCP_BadAddressReturnsEarly(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	s.ensureSigCtx()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listenTCP(s.sigCtx, "definitely:not:an:addr")
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listenTCP with bad addr should return immediately")
	}
}

// ---------- listenTLS with missing cert returns early -----------------------

func TestListenTLS_BadCertReturnsEarly(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	s.ensureSigCtx()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listenTLS(s.sigCtx, "127.0.0.1:0", "/no/such/cert.pem", "/no/such/key.pem")
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listenTLS with bad cert should return immediately")
	}
}

// ---------- startSigTransportListeners gating ------------------------------

func TestStartSigTransportListeners_NoConfigNoop(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	// sigCtx is nil → early return branch
	s.startSigTransportListeners()

	// With sigCtx but empty TCPListen/TLSListen → both branches skipped
	s.ensureSigCtx()
	s.startSigTransportListeners()
}
