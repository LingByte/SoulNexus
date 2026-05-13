package server

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// ---------- extra_handlers (NOTIFY / UPDATE / MESSAGE) ------------------

func TestServer_HandleNotify_Returns200(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"NOTIFY sip:a@b SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKnotify",
		"From: <sip:a@b>;tag=1",
		"To: <sip:a@b>;tag=2",
		"Call-ID: notify-1",
		"CSeq: 1 NOTIFY",
		"Subscription-State: active",
		"Event: message-summary",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleNotify(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})
	if resp == nil || resp.StatusCode != 200 {
		t.Errorf("NOTIFY: %v", resp)
	}
}

func TestServer_HandleUpdate_Returns200(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"UPDATE sip:a@b SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKupdate",
		"From: <sip:a@b>;tag=1",
		"To: <sip:a@b>;tag=2",
		"Call-ID: update-1",
		"CSeq: 1 UPDATE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleUpdate(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})
	if resp == nil || resp.StatusCode != 200 {
		t.Errorf("UPDATE: %v", resp)
	}
}

func TestServer_HandleMessage_EmptyBody200(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"MESSAGE sip:a@b SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKmsg",
		"From: <sip:a@b>;tag=1",
		"To: <sip:a@b>",
		"Call-ID: msg-empty",
		"CSeq: 1 MESSAGE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleMessage(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})
	if resp == nil || resp.StatusCode != 200 {
		t.Errorf("empty MESSAGE: %v", resp)
	}
}

func TestServer_HandleMessage_NonEmptyBody415(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"MESSAGE sip:a@b SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKmsg2",
		"From: <sip:a@b>;tag=1",
		"To: <sip:a@b>",
		"Call-ID: msg-body",
		"CSeq: 1 MESSAGE",
		"Content-Type: text/plain",
		"Content-Length: 5",
		"",
		"hello",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleMessage(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})
	if resp == nil || resp.StatusCode != 415 {
		t.Errorf("MESSAGE with body should be 415: %v", resp)
	}
}

// ---------- Digest auth flow --------------------------------------------

func TestServer_INVITE_WithDigestRequired_Returns401(t *testing.T) {
	srv := New(Config{
		LocalIP:        "127.0.0.1",
		DigestRealm:    "voiceserver",
		DigestUser:     "alice",
		DigestPassword: "secret",
	})
	defer srv.Stop()
	srv.SetInboundAllowUnknownDID(true)

	msg, _ := stack.Parse(rawInviteFor("digest-1"))
	resp := srv.handleInvite(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6050})
	if resp == nil {
		t.Fatal("nil response")
	}
	if resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
	if !strings.Contains(strings.ToLower(resp.GetHeader("WWW-Authenticate")), "digest") {
		t.Errorf("WWW-Authenticate missing digest: %q", resp.GetHeader("WWW-Authenticate"))
	}
}

func TestNewSIPDigest_NilOnEmpty(t *testing.T) {
	if newSIPDigest("", "u", "p") != nil {
		t.Error("empty realm → nil")
	}
	if newSIPDigest("r", "", "p") != nil {
		t.Error("empty user → nil")
	}
	if newSIPDigest("r", "u", "") != nil {
		t.Error("empty pass → nil")
	}
	if newSIPDigest("r", "u", "p") == nil {
		t.Error("all-set should produce digest")
	}
}

func TestDigest_HelpersDirect(t *testing.T) {
	// md5hex / digestHA1 / digestExpectResponse / parseDigestAuth — purely functional
	if got := md5hex("abc"); got != "900150983cd24fb0d6963f7d28e17f72" {
		t.Errorf("md5hex: %q", got)
	}
	ha1 := digestHA1("alice", "voiceserver", "secret")
	if len(ha1) != 32 {
		t.Errorf("digestHA1 length = %d", len(ha1))
	}
	auth := map[string]string{"nonce": "nonceX", "qop": "", "nc": "", "cnonce": ""}
	resp := digestExpectResponse(auth, "INVITE", "sip:user@dom", ha1)
	if len(resp) != 32 {
		t.Errorf("expectResponse length = %d", len(resp))
	}

	// parseDigestAuth: parses Authorization header into kv map
	kv := parseDigestAuth(`Digest username="alice", realm="voiceserver", nonce="abc", uri="sip:foo", response="def"`)
	if kv["username"] != "alice" || kv["nonce"] != "abc" {
		t.Errorf("parseDigestAuth: %v", kv)
	}
	// Empty / non-digest → empty map
	if got := parseDigestAuth(""); len(got) != 0 {
		t.Errorf("empty parseDigestAuth: %v", got)
	}
	if got := parseDigestAuth("Bearer abc"); len(got) != 0 {
		t.Errorf("non-digest parseDigestAuth: %v", got)
	}
}

func TestDigest_GcRemovesExpired(t *testing.T) {
	d := newSIPDigest("r", "u", "p")
	if d == nil {
		t.Fatal("nil digest")
	}
	d.mu.Lock()
	d.nonces["fresh"] = digestNonce{expires: time.Now().Add(60 * time.Second)}
	d.nonces["stale"] = digestNonce{expires: time.Now().Add(-60 * time.Second)}
	d.mu.Unlock()
	d.gc()
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.nonces["fresh"]; !ok {
		t.Error("fresh nonce removed")
	}
	if _, ok := d.nonces["stale"]; ok {
		t.Error("stale nonce not removed")
	}
}

func TestDigest_IssueNonce_ReturnsHex(t *testing.T) {
	d := newSIPDigest("r", "u", "p")
	if d == nil {
		t.Fatal("nil")
	}
	n1 := d.issueNonce()
	n2 := d.issueNonce()
	if n1 == "" || n2 == "" || n1 == n2 {
		t.Errorf("issueNonce: n1=%q n2=%q", n1, n2)
	}
}

// ---------- IP CIDR allow-list -----------------------------------------

func TestParseOneIPCIDR_AllBranches(t *testing.T) {
	if n, err := parseOneIPCIDR("10.0.0.0/8"); err != nil || n == nil {
		t.Errorf("CIDR: err=%v n=%v", err, n)
	}
	if n, err := parseOneIPCIDR("127.0.0.1"); err != nil || n == nil {
		t.Errorf("ipv4 single: err=%v n=%v", err, n)
	}
	if n, err := parseOneIPCIDR("::1"); err != nil || n == nil {
		t.Errorf("ipv6 single: err=%v n=%v", err, n)
	}
	if _, err := parseOneIPCIDR("not-an-ip"); err == nil {
		t.Error("garbage must error")
	}
	if _, err := parseOneIPCIDR("10.0.0.0/99"); err == nil {
		t.Error("bad CIDR mask must error")
	}
}

func TestParseIPCIDRList_SkipsInvalid(t *testing.T) {
	got := parseIPCIDRList("10.0.0.0/8, , garbage, 192.168.0.0/16")
	if len(got) != 2 {
		t.Errorf("want 2 valid CIDRs, got %d", len(got))
	}
}

func TestIPAllowed_NilOrEmptyAllowsAny(t *testing.T) {
	if !ipAllowed(nil, net.ParseIP("1.2.3.4")) {
		t.Error("nil allow-list permits any")
	}
	if !ipAllowed([]*net.IPNet{}, net.ParseIP("1.2.3.4")) {
		t.Error("empty allow-list permits any")
	}
}

func TestIPAllowed_NilIPRejected(t *testing.T) {
	nets := parseIPCIDRList("10.0.0.0/8")
	if ipAllowed(nets, nil) {
		t.Error("nil IP must reject when allow-list set")
	}
}

func TestIPAllowed_MatchAndMiss(t *testing.T) {
	nets := parseIPCIDRList("10.0.0.0/8")
	if !ipAllowed(nets, net.ParseIP("10.5.6.7")) {
		t.Error("10.5.6.7 should match 10.0.0.0/8")
	}
	if ipAllowed(nets, net.ParseIP("192.168.1.1")) {
		t.Error("192.168.1.1 should not match")
	}
}

func TestInviteRate_Allow_NoLimit(t *testing.T) {
	var rs inviteRateState
	if !rs.allow(net.ParseIP("1.2.3.4"), 0, 0) {
		t.Error("0 perSec should always allow")
	}
}

func TestInviteRate_Allow_ZeroIPRejects(t *testing.T) {
	var rs inviteRateState
	if rs.allow(nil, 1.0, 1) {
		t.Error("nil IP must reject when limited")
	}
}

func TestInviteRate_Allow_BurstThenThrottle(t *testing.T) {
	var rs inviteRateState
	ip := net.ParseIP("1.2.3.4")
	allowed := 0
	for i := 0; i < 20; i++ {
		if rs.allow(ip, 0.001, 5) {
			allowed++
		}
	}
	// burst of 5 → first 5 allowed, rest throttled (rate is 0.001/sec → effectively 0 within test window)
	if allowed != 5 {
		t.Errorf("expected 5 burst allowances, got %d", allowed)
	}
}

// ---------- forgetUASDialog / SendUASBye nil-safety + flow ------------

func TestForgetUASDialog_NoOp(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	srv.forgetUASDialog("nonexistent")
	srv.ForgetUASDialog("nonexistent") // exported alias
}

func TestSendUASBye_NoDialogReturnsError(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	if err := srv.SendUASBye("no-such-call"); err == nil {
		t.Error("BYE without remembered dialog must error")
	}
}

// ---------- Presence (SUBSCRIBE) -------------------------------------

func TestServer_HandleSubscribe_PresenceEvent(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"SUBSCRIBE sip:peer@dom SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKsub",
		"From: <sip:watcher@dom>;tag=1",
		"To: <sip:peer@dom>",
		"Call-ID: sub-1",
		"CSeq: 1 SUBSCRIBE",
		"Event: presence",
		"Expires: 60",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleSubscribe(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	if resp == nil {
		t.Fatal("nil response")
	}
	// 200 OK accept or 489 Bad Event — both are valid behaviours
	if resp.StatusCode != 200 && resp.StatusCode != 489 && resp.StatusCode != 481 {
		t.Errorf("SUBSCRIBE: %v", resp)
	}
}

func TestServer_HandlePublishPresence_AnyResponse(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"PUBLISH sip:user@dom SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKpub",
		"From: <sip:user@dom>;tag=1",
		"To: <sip:user@dom>",
		"Call-ID: pub-1",
		"CSeq: 1 PUBLISH",
		"Event: presence",
		"Expires: 60",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handlePublishPresence(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	if resp == nil {
		t.Fatal("nil response from PUBLISH")
	}
}

func TestServer_HandleCancel_OrphanReturns481(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"CANCEL sip:user@dom SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKcancel",
		"From: <sip:caller@dom>;tag=1",
		"To: <sip:user@dom>",
		"Call-ID: cancel-orphan",
		"CSeq: 1 CANCEL",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handleCancel(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	// May return 481 Call/Transaction Does Not Exist or 200 OK depending on absorbed retransmit logic
	if resp == nil {
		t.Fatal("nil")
	}
	if resp.StatusCode != 481 && resp.StatusCode != 200 {
		t.Errorf("CANCEL orphan: %v", resp)
	}
}

// ---------- compat callOwnedByBusinessRouting / attachInboundCallToBusiness

func TestCompat_CallOwnedByBusinessRouting_AlwaysFalse(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	if srv.callOwnedByBusinessRouting("any") {
		t.Error("default should be false")
	}
}

func TestCompat_AttachInboundCallToBusiness_NoOp(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	srv.attachInboundCallToBusiness("c1") // must not panic
}

// ---------- TrunkCapacityTracker -------------------------------------

func TestTrunkCapacityTracker_AcquireReleaseInbound(t *testing.T) {
	tr := NewTrunkCapacityTracker()
	if tr == nil {
		t.Fatal("nil tracker")
	}
	// First call within limit succeeds.
	if !tr.TryAcquireInbound("c1", 7, 2) {
		t.Error("first acquire should succeed")
	}
	if !tr.TryAcquireInbound("c2", 7, 2) {
		t.Error("second acquire within limit should succeed")
	}
	// Third hits limit
	if tr.TryAcquireInbound("c3", 7, 2) {
		t.Error("third acquire should be rejected (limit=2)")
	}
	// Release frees a slot
	tr.ReleaseInbound("c1")
	if !tr.TryAcquireInbound("c4", 7, 2) {
		t.Error("after release, new acquire should succeed")
	}
	// Idempotent release (unknown ID safe)
	tr.ReleaseInbound("nope")
}

// ---------- Existing-pre setters all reachable ------------------------

func TestServer_AllExistingSetters_NilSafe(t *testing.T) {
	var s *SIPServer
	s.SetRegisterStore(nil)
	s.SetCallPersist(nil)
	s.SetInboundDIDBindingResolver(nil)
	s.SetInboundCapacityGate(nil)
	s.SetInboundCapacityRelease(nil)
	s.SetVoiceDialogWSLookup(nil)
	if s.lookupVoiceDialogWS("c") != "" {
		t.Error("nil server lookup must be empty")
	}
}

func TestServer_AllExistingSetters_Roundtrip(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()

	srv.SetRegisterStore(nil)
	srv.SetCallPersist(nil)
	srv.SetInboundDIDBindingResolver(func(*stack.Message) InboundDIDBinding { return InboundDIDBinding{TenantID: 1} })
	srv.SetInboundCapacityGate(func(callID, called string) (bool, int, string) { return true, 0, "" })
	srv.SetInboundCapacityRelease(func(callID string) {})
	srv.SetVoiceDialogWSLookup(func(callID string) string { return "ws://x/" + callID })
	if srv.lookupVoiceDialogWS("c1") != "ws://x/c1" {
		t.Error("voice dialog lookup not stored")
	}
}

// ---------- Direct unit tests for small registrar helpers --------------

func TestRandomBranch(t *testing.T) {
	a, b := randomBranch(), randomBranch()
	if a == "" || b == "" || a == b {
		t.Errorf("randomBranch: %q vs %q", a, b)
	}
	// randomBranch returns hex bytes; the SIP magic-cookie prefix is added by callers.
	if len(a) < 8 {
		t.Errorf("randomBranch too short: %q", a)
	}
}

func TestRandomHexBranch(t *testing.T) {
	a := randomHexBranch()
	if len(a) == 0 {
		t.Error("randomHexBranch empty")
	}
}

func TestRegistrationKey(t *testing.T) {
	if got := registrationKey("alice", "Example.com"); got != "alice@example.com" {
		t.Errorf("got %q", got)
	}
}

func TestParseURIUserHost(t *testing.T) {
	cases := []struct {
		in           string
		wantUser     string
		wantHost     string
		wantOK       bool
	}{
		{"sip:alice@example.com", "alice", "example.com", true},
		{"sip:alice@example.com:5060", "alice", "example.com", true},
		{"<sip:alice@example.com>", "alice", "example.com", true},
		{"<sip:alice@example.com;transport=udp>", "alice", "example.com", true},
		{"alice@example.com", "", "", false}, // bare (no sip:) is not accepted
		{"sip:example.com", "", "example.com", false},        // no user
		{"", "", "", false},
		{"garbage", "", "", false},
	}
	for _, c := range cases {
		u, h, ok := parseURIUserHost(c.in)
		if ok != c.wantOK {
			t.Errorf("%q: ok=%v want %v", c.in, ok, c.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if u != c.wantUser || strings.ToLower(h) != strings.ToLower(c.wantHost) {
			t.Errorf("%q: u=%q h=%q want %q,%q", c.in, u, h, c.wantUser, c.wantHost)
		}
	}
}

func TestParseExpiresRegister(t *testing.T) {
	mk := func(headers ...string) *stack.Message {
		m := &stack.Message{Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
		for i := 0; i+1 < len(headers); i += 2 {
			m.SetHeader(headers[i], headers[i+1])
		}
		return m
	}
	if s, ok := parseExpiresRegister(mk("Expires", "60")); !ok || s != 60 {
		t.Errorf("explicit expires: s=%d ok=%v", s, ok)
	}
	if s, ok := parseExpiresRegister(mk("Contact", "<sip:u@h>;expires=120")); !ok || s != 120 {
		t.Errorf("contact-expires: s=%d ok=%v", s, ok)
	}
	// missing both → implementation may default-on; just exercise the path
	_, _ = parseExpiresRegister(mk())
}

func TestParseContactUDPAddr_Basics(t *testing.T) {
	src := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 6050}
	if a := parseContactUDPAddr("<sip:user@10.0.0.1:5060>", src); a == nil || a.Port != 5060 {
		t.Errorf("contact addr: %v", a)
	}
	// Garbage falls back to source address (not nil)
	if a := parseContactUDPAddr("not a contact", src); a == nil {
		t.Errorf("garbage contact should fall back to src, got nil")
	}
	if a := parseContactUDPAddr("", src); a == nil {
		t.Errorf("empty contact should fall back to src, got nil")
	}
}

// ---------- StartAckHandler exposed alias -------------------------------

func TestServer_StartAckHandler_NilCallNoOp(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"ACK sip:user@a SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKack-noop",
		"From: <sip:c@a>;tag=1",
		"To: <sip:u@a>;tag=2",
		"Call-ID: ack-noop",
		"CSeq: 1 ACK",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.StartAckHandler(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	if resp != nil {
		t.Errorf("ACK should produce no response, got %v", resp)
	}
}

// ---------- SendSIP without endpoint --------------------------------

func TestSendSIP_NoEndpointError(t *testing.T) {
	var s *SIPServer
	if err := s.SendSIP(&stack.Message{}, &net.UDPAddr{}); err == nil {
		t.Error("nil server SendSIP must error")
	}
}

// ---------- Presence SUBSCRIBE → NOTIFY + expire ---------------------

func TestServer_HandleSubscribe_PresenceCreateAndPrune(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()

	subRaw := strings.Join([]string{
		"SUBSCRIBE sip:peer@dom SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKsubpres",
		"From: <sip:watcher@dom>;tag=1",
		"To: <sip:peer@dom>",
		"Call-ID: subpres-1",
		"CSeq: 1 SUBSCRIBE",
		"Event: presence",
		"Contact: <sip:watcher@10.0.0.1:5060>",
		"Expires: 60",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(subRaw)
	resp := srv.handleSubscribe(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	if resp == nil {
		t.Fatal("nil")
	}
	// Re-subscribe with Expires: 0 to terminate (exercise pruneSubByCallID branch)
	subEndRaw := strings.Replace(subRaw, "Expires: 60", "Expires: 0", 1)
	msgEnd, _ := stack.Parse(subEndRaw)
	srv.handleSubscribe(msgEnd, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
}

// Presence helper: parse Expires (covers parseSubscribeExpires explicit branches)
func TestParseSubscribeExpires_Variants(t *testing.T) {
	mk := func(headers ...string) *stack.Message {
		m := &stack.Message{Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
		for i := 0; i+1 < len(headers); i += 2 {
			m.SetHeader(headers[i], headers[i+1])
		}
		return m
	}
	if v := parseSubscribeExpires(mk("Expires", "120")); v != 120 {
		t.Errorf("explicit Expires: %d", v)
	}
	if v := parseSubscribeExpires(mk("Expires", "")); v == 0 {
		t.Errorf("default expires should be > 0, got %d", v)
	}
	if v := parseSubscribeExpires(mk("Expires", "garbage")); v == 0 {
		t.Errorf("garbage expires falls back to default, got %d", v)
	}
}

// ---------- Inbound capacity flow with gates ---------------------------

func TestServer_INVITE_RejectedByCapacityGate(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	srv.SetInboundAllowUnknownDID(true)

	srv.SetInboundCapacityGate(func(callID, called string) (bool, int, string) {
		return false, 503, "Service Unavailable"
	})

	msg, _ := stack.Parse(rawInviteFor("cap-rejected-1"))
	resp := srv.handleInvite(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6050})
	if resp == nil || resp.StatusCode != 503 {
		t.Errorf("capacity-gate reject: %v", resp)
	}
}

// ---------- INVITE allow-list rejection ---------------------------

func TestServer_INVITE_AllowCIDRsBlocksOutsiders(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1", InviteAllowCIDRs: []string{"10.0.0.0/8"}})
	defer srv.Stop()
	srv.SetInboundAllowUnknownDID(true)

	msg, _ := stack.Parse(rawInviteFor("allow-blocked-1"))
	resp := srv.handleInvite(msg, &net.UDPAddr{IP: net.IPv4(192, 168, 1, 1), Port: 6050})
	if resp == nil || resp.StatusCode != 403 {
		t.Errorf("allow-cidr block: %v", resp)
	}
}

// ---------- Various extra calls ----------------------------------------

func TestServer_HandlePrack_Orphan(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	raw := strings.Join([]string{
		"PRACK sip:user@dom SIP/2.0",
		"Via: SIP/2.0/UDP 10.0.0.1;branch=z9hG4bKprack-orphan",
		"From: <sip:caller@dom>;tag=1",
		"To: <sip:user@dom>;tag=2",
		"Call-ID: prack-orphan",
		"CSeq: 1 PRACK",
		"RAck: 1 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}, "\r\n")
	msg, _ := stack.Parse(raw)
	resp := srv.handlePrack(msg, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6000})
	// Orphan PRACK either gets 481 or absorbed silently.
	if resp != nil && resp.StatusCode != 481 && resp.StatusCode != 200 {
		t.Errorf("PRACK orphan: %v", resp)
	}
}

func TestTrunkCapacityTracker_OutboundIndependent(t *testing.T) {
	tr := NewTrunkCapacityTracker()
	if !tr.TryAcquireOutbound("o1", 7, 1) {
		t.Error("outbound acquire failed")
	}
	if tr.TryAcquireOutbound("o2", 7, 1) {
		t.Error("outbound limit exceeded")
	}
	tr.ReleaseOutbound("o1")
	if !tr.TryAcquireOutbound("o3", 7, 1) {
		t.Error("outbound after release failed")
	}
	tr.ReleaseOutbound("nope") // safe
}
