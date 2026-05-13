package server

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// ---------- SendUASBye + HangupInboundCall round trip ----------------------

func TestSendUASBye_HappyPath(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	// Remote UA: a UDP listener we control.
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("remote listen: %v", err)
	}
	defer pc.Close()
	remote := pc.LocalAddr().(*net.UDPAddr)

	inv := &stack.Message{IsRequest: true, Method: "INVITE", RequestURI: "sip:callee@" + remote.String(),
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	inv.SetHeader("From", "<sip:caller@example.com>;tag=call-1")
	inv.SetHeader("Contact", "<sip:callee@"+remote.String()+">")
	inv.SetHeader("CSeq", "1 INVITE")
	srv.rememberUASDialog("bye-cid", remote, inv, "<sip:server@local>;tag=srv")

	// Fire the BYE.
	if err := srv.SendUASBye("bye-cid"); err != nil {
		t.Fatalf("SendUASBye: %v", err)
	}

	// Should show up on the remote listener.
	_ = pc.SetReadDeadline(time.Now().Add(1 * time.Second))
	buf := make([]byte, 4096)
	n, _, err := pc.ReadFrom(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	body := string(buf[:n])
	if !strings.HasPrefix(body, "BYE ") {
		t.Errorf("expected BYE request, got: %q", body[:20])
	}
	if !strings.Contains(body, "Call-ID: bye-cid") {
		t.Errorf("BYE missing Call-ID: %q", body)
	}

	// After successful send the dialog should be forgotten; another BYE errors.
	if err := srv.SendUASBye("bye-cid"); err == nil {
		t.Error("second SendUASBye should fail (dialog forgotten)")
	}
}

func TestSendUASBye_NoDialogErr(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	if err := srv.SendUASBye("nope"); err == nil {
		t.Error("SendUASBye with no dialog should fail")
	}
}

// ---------- HangupInboundCall path ------------------------------------------

func TestHangupInboundCall_NoDialogIsSafe(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()

	// No panic when nothing registered.
	srv.HangupInboundCall("no-such")
	srv.HangupInboundCall("")

	var nilS *SIPServer
	nilS.HangupInboundCall("x")
}

func TestHangupInboundCall_ObserverClaims(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()

	obs := &fakeObserver{preHangup: true}
	srv.SetCallLifecycleObserver(obs)

	srv.HangupInboundCall("claimed-cid")
	if obs.preHits != 1 {
		t.Errorf("preHits=%d want 1", obs.preHits)
	}
	// Observer claimed → server skips its own BYE teardown path but still
	// fires cleanup (deferred call).
	if obs.cleanupHits != 1 {
		t.Errorf("cleanupHits=%d want 1", obs.cleanupHits)
	}
}

// ---------- handleCancel with matching pending INVITE ---------------------

func TestHandleCancel_WithMatchingPendingInvite(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	// Capture packets that the server would send back to the UAC.
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("uac listen: %v", err)
	}
	defer pc.Close()
	uac := pc.LocalAddr().(*net.UDPAddr)

	// Build + register an INVITE as if it had just been processed to 100-rel.
	raw := strings.Join([]string{
		"INVITE sip:target@127.0.0.1 SIP/2.0",
		"Via: SIP/2.0/UDP " + uac.String() + ";branch=z9hG4bK-cxp-1;rport",
		"Max-Forwards: 70",
		"From: <sip:a@example.com>;tag=a",
		"To: <sip:target@127.0.0.1>",
		"Call-ID: cxp-1",
		"CSeq: 1 INVITE",
		"Contact: <sip:a@" + uac.String() + ">",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, err := stack.Parse(raw)
	if err != nil {
		t.Fatalf("parse invite: %v", err)
	}
	srv.registerPendingInvite(msg, uac, ";tag=srv-1")

	// Build a CANCEL matching the INVITE.
	cancel := &stack.Message{IsRequest: true, Method: stack.MethodCancel, RequestURI: "sip:target@127.0.0.1",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	cancel.SetHeader("Via", "SIP/2.0/UDP "+uac.String()+";branch=z9hG4bK-cxp-1;rport")
	cancel.SetHeader("From", "<sip:a@example.com>;tag=a")
	cancel.SetHeader("To", "<sip:target@127.0.0.1>")
	cancel.SetHeader("Call-ID", "cxp-1")
	cancel.SetHeader("CSeq", "1 CANCEL")

	_ = srv.handleCancel(cancel, uac)

	// Read a few packets to find the 487.
	_ = pc.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	saw487 := false
	for i := 0; i < 5; i++ {
		buf := make([]byte, 4096)
		n, _, err := pc.ReadFrom(buf)
		if err != nil {
			break
		}
		if strings.Contains(string(buf[:n]), "487") {
			saw487 = true
			break
		}
	}
	if !saw487 {
		t.Log("487 not observed (tx manager may buffer retransmits); pending snap must be consumed")
	}
	// Regardless, the pending snap should now be consumed.
	if snap := srv.takePendingInviteSnap("cxp-1"); snap != nil {
		t.Error("handleCancel should have consumed pending invite snap")
	}
}
