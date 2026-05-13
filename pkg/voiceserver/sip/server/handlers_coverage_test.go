package server

import (
	"net"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// helper: minimal INVITE+SDP raw message for a given Call-ID.
func rawInviteFor(callID string) string {
	body := strings.Join([]string{
		"v=0",
		"o=- 123456 123456 IN IP4 198.51.100.1",
		"s=Session",
		"c=IN IP4 198.51.100.1",
		"t=0 0",
		"m=audio 49170 RTP/AVP 0 8",
		"a=rtpmap:0 PCMU/8000",
		"a=rtpmap:8 PCMA/8000",
	}, "\r\n")
	return strings.Join([]string{
		"INVITE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP client.com:6050;branch=z9hG4bK" + callID,
		"To: <sip:user@domain.com>",
		"From: <sip:caller@client.com>;tag=" + callID,
		"Call-ID: " + callID,
		"CSeq: 1 INVITE",
		"Contact: <sip:caller@client.com>",
		"Content-Type: application/sdp",
		"",
		body,
	}, "\r\n")
}

// ---------- Default OPTIONS handler --------------------------------------

func TestServer_HandleOptions_AnswersAllow(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"OPTIONS sip:server.example SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKopt",
		"From: <sip:probe@a>;tag=1",
		"To: <sip:server@b>",
		"Call-ID: opt-cov",
		"CSeq: 1 OPTIONS",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, err := stack.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	resp := srv.handleOptions(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	if resp == nil {
		t.Fatal("nil response")
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d", resp.StatusCode)
	}
	if resp.GetHeader("Allow") == "" {
		t.Errorf("Allow header missing: %+v", resp.Headers)
	}
}

// ---------- Register flow with no store --------------------------------

func TestServer_HandleRegister_NoStoreStill200(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"REGISTER sip:server.example SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKreg",
		"From: <sip:user@server.example>;tag=1",
		"To: <sip:user@server.example>",
		"Call-ID: reg-cov",
		"CSeq: 1 REGISTER",
		"Contact: <sip:user@10.0.0.1:5060>",
		"Expires: 3600",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, err := stack.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	resp := srv.handleRegister(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})
	if resp == nil {
		t.Fatal("nil response")
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d (expected 200; with no SIPRegisterStore the registrar still echoes 200)", resp.StatusCode)
	}
}

func TestServer_HandleRegister_StaticPasswordReject(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1", RegisterStaticPassword: "topsecret"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"REGISTER sip:server.example SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKreg2",
		"From: <sip:user@server.example>;tag=1",
		"To: <sip:user@server.example>",
		"Call-ID: reg-cov-2",
		"CSeq: 1 REGISTER",
		"Contact: <sip:user@10.0.0.1:5060>",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleRegister(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})
	if resp == nil || resp.StatusCode != 403 {
		t.Errorf("expected 403 Forbidden, got %v", resp)
	}
}

// ---------- INVITE rejected by InviteHandler ---------------------------

func TestServer_HandleInvite_BusinessReject486(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	srv.SetInboundAllowUnknownDID(true)
	srv.SetInviteHandler(&fakeInviteHandler{
		decision: Decision{Accept: false, StatusCode: 486, ReasonPhrase: "Busy Here"},
	})

	msg, _ := stack.Parse(rawInviteFor("invreject1"))
	resp := srv.handleInvite(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6050})
	if resp == nil {
		t.Fatal("nil")
	}
	if resp.StatusCode != 486 {
		t.Errorf("status = %d, want 486", resp.StatusCode)
	}
}

// ---------- INFO + DTMFSink flow --------------------------------------

func TestServer_HandleInfo_DTMFRoutedToSink(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	ds := &fakeDTMFSink{}
	srv.SetDTMFSink(ds)

	infoRaw := strings.Join([]string{
		"INFO sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKinfo",
		"From: <sip:caller@client.com>;tag=1",
		"To: <sip:user@domain.com>;tag=2",
		"Call-ID: info-cov-1",
		"CSeq: 100 INFO",
		"Content-Type: application/dtmf-relay",
		"Content-Length: 22",
		"",
		"Signal=5\r\nDuration=160",
	}, "\r\n")
	msg, err := stack.Parse(infoRaw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	resp := srv.handleInfo(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	if resp == nil || resp.StatusCode != 200 {
		t.Errorf("INFO response: %v", resp)
	}
	got := ds.all()
	if len(got) != 1 || got[0] != "info-cov-1:5" {
		t.Errorf("dtmf events = %v, want [info-cov-1:5]", got)
	}
}

// ---------- BYE without prior INVITE -----------------------------------

func TestServer_HandleBye_NoPriorCallStill200(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"BYE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKbye-orphan",
		"From: <sip:caller@client.com>;tag=1",
		"To: <sip:user@domain.com>;tag=2",
		"Call-ID: bye-orphan-1",
		"CSeq: 200 BYE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleBye(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	if resp == nil || resp.StatusCode != 200 {
		t.Errorf("orphan BYE response: %v", resp)
	}
}

// ---------- Cancel + observer pre-hangup ------------------------------

func TestServer_HandleBye_ObserverClaimsTeardown(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	obs := &fakeObserver{preHangup: true}
	srv.SetCallLifecycleObserver(obs)

	raw := strings.Join([]string{
		"BYE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKbye-claim",
		"From: <sip:caller@client.com>;tag=1",
		"To: <sip:user@domain.com>;tag=2",
		"Call-ID: bye-claim-1",
		"CSeq: 1 BYE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleBye(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	if resp == nil || resp.StatusCode != 200 {
		t.Errorf("BYE observer-claim response: %v", resp)
	}
	if obs.preHits != 1 {
		t.Errorf("observer.preHits = %d", obs.preHits)
	}
}

// ---------- ACK on unknown Call-ID --------------------------------------

func TestServer_HandleAck_UnknownCall_Noop(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"ACK sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKack-orphan",
		"From: <sip:caller@client.com>;tag=1",
		"To: <sip:user@domain.com>;tag=2",
		"Call-ID: ack-orphan",
		"CSeq: 1 ACK",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleAck(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	if resp != nil {
		t.Errorf("ACK should produce no response, got %v", resp)
	}
}

// ---------- INVITE missing Call-ID → 400 -------------------------------

func TestServer_HandleInvite_MissingCallID(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"INVITE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKnocallid",
		"From: <sip:caller@client.com>;tag=1",
		"To: <sip:user@domain.com>",
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleInvite(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	if resp == nil || resp.StatusCode != 400 {
		t.Errorf("missing Call-ID INVITE: %v", resp)
	}
}

// ---------- INVITE bad SDP → 488 ----------------------------------------

func TestServer_HandleInvite_BadSDP_488(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	srv.SetInboundAllowUnknownDID(true)
	raw := strings.Join([]string{
		"INVITE sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKbadsdp",
		"From: <sip:caller@client.com>;tag=1",
		"To: <sip:user@domain.com>",
		"Call-ID: badsdp-1",
		"CSeq: 1 INVITE",
		"Content-Type: application/sdp",
		"Content-Length: 5",
		"",
		"hello",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleInvite(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	if resp == nil || resp.StatusCode != 488 {
		t.Errorf("bad sdp: %v", resp)
	}
}

// ---------- handleRefer rejects when call not present -----------------

func TestServer_HandleRefer_NoCall_481(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"REFER sip:user@domain.com SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKrefer-orphan",
		"From: <sip:caller@client.com>;tag=1",
		"To: <sip:user@domain.com>;tag=2",
		"Call-ID: refer-orphan",
		"CSeq: 1 REFER",
		"Refer-To: <sip:transferee@target>",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleRefer(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	if resp == nil || resp.StatusCode != 481 {
		t.Errorf("orphan REFER: %v", resp)
	}
}

// ---------- Listen address accessor -----------------------------------

func TestServer_ListenAddr_BeforeStart(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", LocalIP: "127.0.0.1"})
	defer srv.Stop()
	host, port := srv.ListenAddr()
	if host != "127.0.0.1" {
		t.Errorf("host = %q", host)
	}
	if port != 0 {
		t.Errorf("port should be 0 before Start(), got %d", port)
	}
}

func TestServer_ListenAddr_NilSafe(t *testing.T) {
	var s *SIPServer
	host, port := s.ListenAddr()
	if host != "" || port != 0 {
		t.Errorf("nil server ListenAddr = %q,%d", host, port)
	}
}

// ---------- Start/Stop lifecycle --------------------------------------

func TestServer_StartStop(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	host, port := srv.ListenAddr()
	if host == "" || port == 0 {
		t.Errorf("after Start: host=%q port=%d", host, port)
	}
	if err := srv.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}
	// Idempotent Stop
	if err := srv.Stop(); err != nil {
		t.Errorf("second Stop: %v", err)
	}
}

func TestServer_NilLifecycle(t *testing.T) {
	var s *SIPServer
	if err := s.Stop(); err != nil {
		t.Errorf("nil Stop: %v", err)
	}
	if err := s.Start(); err == nil {
		t.Error("nil Start should error")
	}
}

// ---------- HangupInboundCall guard clauses ---------------------------

func TestServer_HangupInboundCall_Empty(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	srv.HangupInboundCall("")     // empty ID
	srv.HangupInboundCall("nope") // unknown ID
	var nilSrv *SIPServer
	nilSrv.HangupInboundCall("anything")
}

// ---------- GetCallSession / RemoveCallSession / RegisterCallSession nil-safe

func TestServer_CallSessionAPIs_NilSafe(t *testing.T) {
	var s *SIPServer
	if s.GetCallSession("c") != nil {
		t.Error("nil server GetCallSession must be nil")
	}
	s.RemoveCallSession("c")
	s.RegisterCallSession("c", nil)

	good := New(Config{LocalIP: "127.0.0.1"})
	defer good.Stop()
	if good.GetCallSession("") != nil {
		t.Error("empty id GetCallSession must be nil")
	}
	good.RemoveCallSession("") // empty no-op
	good.RegisterCallSession("", nil)
}
