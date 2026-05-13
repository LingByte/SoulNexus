package server

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// ---------- parseContactUDPAddr branches -----------------------------------

func TestParseContactUDPAddr_AllBranches(t *testing.T) {
	src := &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5060}

	// Not a sip: URI → fall through to src.
	if got := parseContactUDPAddr("nonsense", src); got == nil || got.Port != 5060 {
		t.Errorf("non-sip fallback: %v", got)
	}

	// sip: URI without @ → fall back to src.
	if got := parseContactUDPAddr("<sip:abc>", src); got == nil || got.Port != 5060 {
		t.Errorf("no-@ fallback: %v", got)
	}

	// sip:user@host (no port) → IP + default 6050
	if got := parseContactUDPAddr("<sip:alice@10.0.0.2>", src); got == nil || got.IP.String() != "10.0.0.2" || got.Port != 6050 {
		t.Errorf("no-port default: %v", got)
	}

	// sip:user@host:port happy
	if got := parseContactUDPAddr("<sip:alice@10.0.0.2:5080>", src); got == nil || got.Port != 5080 {
		t.Errorf("happy: %v", got)
	}

	// sip:user@host:port;transport=udp (semicolon strip)
	if got := parseContactUDPAddr("<sip:alice@10.0.0.2:5060;transport=udp>", src); got == nil || got.Port != 5060 {
		t.Errorf(";params: %v", got)
	}

	// Nil src + no-@ URI → nil
	if got := parseContactUDPAddr("<sip:nohost>", nil); got != nil {
		t.Errorf("nil src + no-@ should be nil, got %v", got)
	}

	// Bad port in URI → fallback to 6050 (with parsed IP)
	if got := parseContactUDPAddr("<sip:alice@10.0.0.2:notanum>", src); got == nil || got.Port != 6050 {
		t.Errorf("bad port → 6050, got %v", got)
	}
}

// ---------- InboundCalledPartyUser ----------------------------------------

func TestInboundCalledPartyUser(t *testing.T) {
	if got := InboundCalledPartyUser(nil); got != "" {
		t.Error("nil msg → empty")
	}
	// Prefers Request-URI
	m := &stack.Message{IsRequest: true, RequestURI: "sip:bob@example.com",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	m.SetHeader("To", "<sip:alice@example.com>")
	if got := InboundCalledPartyUser(m); got != "bob" {
		t.Errorf("got %q want bob", got)
	}
	// Falls back to To when RequestURI user is empty/missing
	m2 := &stack.Message{IsRequest: true, RequestURI: "",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	m2.SetHeader("To", "<sip:carol@example.com>")
	if got := InboundCalledPartyUser(m2); got != "carol" {
		t.Errorf("got %q want carol", got)
	}
	// Unparseable → ""
	m3 := &stack.Message{IsRequest: true, RequestURI: "garbage",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	if got := InboundCalledPartyUser(m3); got != "" {
		t.Errorf("unparseable got %q", got)
	}
}

// ---------- presentityKeyFromSubscribe ------------------------------------

func TestPresentityKeyFromSubscribe(t *testing.T) {
	if got := presentityKeyFromSubscribe(nil); got != "" {
		t.Error("nil → empty")
	}
	// RequestURI wins
	m := &stack.Message{IsRequest: true, RequestURI: "SIP:Bob@Example.COM",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	if got := presentityKeyFromSubscribe(m); got != "sip:bob@example.com" {
		t.Errorf("RequestURI path: %q", got)
	}
	// Falls back to To header and strips <…>
	m2 := &stack.Message{IsRequest: true, RequestURI: "",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	m2.SetHeader("To", "\"Alice\" <sip:alice@example.com>;tag=x")
	if got := presentityKeyFromSubscribe(m2); got != "sip:alice@example.com" {
		t.Errorf("To path: %q", got)
	}
}

// ---------- digestExpectResponse: qop=auth vs classic --------------------

func TestDigestExpectResponse_BothPaths(t *testing.T) {
	ha1 := digestHA1("alice", "example.com", "secret")

	// qop=auth path exercises nc/cnonce.
	authMap := map[string]string{
		"nonce":  "abc123",
		"qop":    "auth",
		"nc":     "00000001",
		"cnonce": "cn",
	}
	hAuth := digestExpectResponse(authMap, "INVITE", "sip:bob@example.com", ha1)
	if len(hAuth) != 32 {
		t.Errorf("qop=auth hash: %q", hAuth)
	}

	// classic (no qop) path
	classic := map[string]string{"nonce": "abc123"}
	hClassic := digestExpectResponse(classic, "INVITE", "sip:bob@example.com", ha1)
	if hClassic == hAuth {
		t.Error("classic and qop=auth should differ")
	}
}

// ---------- verifyINVITE: round-trip matching digest ----------------------

func TestVerifyINVITE_RoundTrip(t *testing.T) {
	d := newSIPDigest("example.com", "alice", "secret")
	if d == nil {
		t.Fatal("digest builder returned nil")
	}
	nonce := d.issueNonce()

	// Build a matching digest response.
	ha1 := digestHA1("alice", "example.com", "secret")
	ha2 := md5hex("INVITE:sip:bob@example.com")
	response := md5hex(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))

	req := &stack.Message{IsRequest: true, Method: "INVITE", RequestURI: "sip:bob@example.com",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	req.SetHeader("Authorization",
		fmt.Sprintf(`Digest username="alice", realm="example.com", nonce="%s", uri="sip:bob@example.com", response="%s"`, nonce, response))

	if !d.verifyINVITE(req) {
		t.Error("matching digest should verify")
	}
	// Replay should fail (nonce consumed).
	if d.verifyINVITE(req) {
		t.Error("nonce should be single-use")
	}

	// Missing Authorization → false
	req2 := &stack.Message{IsRequest: true, Method: "INVITE",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	if d.verifyINVITE(req2) {
		t.Error("missing Authorization should fail")
	}

	// Wrong user → false
	wrongUser := &stack.Message{IsRequest: true, Method: "INVITE",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	wrongUser.SetHeader("Authorization",
		`Digest username="bob", realm="example.com", nonce="x", uri="sip:x@x", response="y"`)
	if d.verifyINVITE(wrongUser) {
		t.Error("wrong user should fail")
	}

	// Unknown nonce → false
	unknownNonce := &stack.Message{IsRequest: true, Method: "INVITE",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	unknownNonce.SetHeader("Authorization",
		fmt.Sprintf(`Digest username="alice", realm="example.com", nonce="%s", uri="sip:x@x", response="%s"`, "never-issued", response))
	if d.verifyINVITE(unknownNonce) {
		t.Error("unknown nonce should fail")
	}

	// Nil cases
	var nilD *sipDigestAuth
	if nilD.verifyINVITE(req) {
		t.Error("nil digest should fail")
	}
	if d.verifyINVITE(nil) {
		t.Error("nil msg should fail")
	}
}

// ---------- newSIPDigest nil-on-empty --------------------------------------

func TestNewSIPDigest_EmptyReturnsNil(t *testing.T) {
	if newSIPDigest("", "u", "p") != nil {
		t.Error("empty realm should yield nil")
	}
	if newSIPDigest("r", "", "p") != nil {
		t.Error("empty user should yield nil")
	}
	if newSIPDigest("r", "u", "") != nil {
		t.Error("empty pass should yield nil")
	}
	d := newSIPDigest("r", "u", "p")
	if d == nil {
		t.Fatal("valid digest should not be nil")
	}
	d.gc() // no expired yet
	var nilD *sipDigestAuth
	nilD.gc() // must not panic
}

// ---------- handleRefer paths ---------------------------------------------

func TestHandleRefer_Paths(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	// Missing Call-ID → 400
	bad := &stack.Message{IsRequest: true, Method: stack.MethodRefer,
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	bad.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK-rf-1")
	bad.SetHeader("CSeq", "1 REFER")
	if resp := srv.handleRefer(bad, &net.UDPAddr{IP: net.ParseIP("10.0.0.1")}); resp == nil || resp.StatusCode != 400 {
		t.Errorf("missing Call-ID → 400, got %+v", resp)
	}

	// Call-ID set but no active session → 481
	no := &stack.Message{IsRequest: true, Method: stack.MethodRefer,
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	no.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK-rf-2")
	no.SetHeader("Call-ID", "rf-no-session")
	no.SetHeader("CSeq", "1 REFER")
	no.SetHeader("Refer-To", "sip:c@x")
	if resp := srv.handleRefer(no, &net.UDPAddr{IP: net.ParseIP("10.0.0.1")}); resp == nil || resp.StatusCode != 481 {
		t.Errorf("no session → 481, got %+v", resp)
	}

	// Nil msg → nil
	if resp := srv.handleRefer(nil, nil); resp != nil {
		t.Error("nil msg → nil")
	}
}

// ---------- handleReInvite: basic guards -----------------------------------

func TestHandleReInvite_NilGuards(t *testing.T) {
	srv := New(Config{LocalIP: "127.0.0.1"})
	defer srv.Stop()
	if resp := srv.handleReInvite(nil, nil, nil); resp != nil {
		t.Error("nil msg → nil")
	}
	if resp := srv.handleReInvite(&stack.Message{}, nil, nil); resp != nil {
		t.Error("nil addr → nil")
	}
	if resp := srv.handleReInvite(&stack.Message{}, &net.UDPAddr{}, nil); resp != nil {
		t.Error("nil cs → nil")
	}
	var nilS *SIPServer
	if resp := nilS.handleReInvite(&stack.Message{}, &net.UDPAddr{}, nil); resp != nil {
		t.Error("nil server → nil")
	}
}

// ---------- isPrivateIPv4 branches -----------------------------------------

func TestIsPrivateIPv4(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"::1", false}, // IPv6
	}
	for _, c := range cases {
		t.Run(c.ip, func(t *testing.T) {
			got := isPrivateIPv4(net.ParseIP(c.ip))
			if got != c.want {
				t.Errorf("%s: got %v want %v", c.ip, got, c.want)
			}
		})
	}
	// nil IP
	if isPrivateIPv4(nil) {
		t.Error("nil IP must not be private")
	}
}

// ---------- challenge401 + digest path --------------------------------------

func TestDigest_Challenge401(t *testing.T) {
	d := newSIPDigest("example.com", "alice", "secret")
	req := &stack.Message{IsRequest: true, Method: "INVITE", RequestURI: "sip:alice@example.com",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	req.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK-dg")
	req.SetHeader("From", "<sip:a@example.com>;tag=x")
	req.SetHeader("To", "<sip:alice@example.com>")
	req.SetHeader("Call-ID", "dg-1")
	req.SetHeader("CSeq", "1 INVITE")

	resp, err := d.challenge401(req)
	if err != nil {
		t.Fatalf("challenge401: %v", err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("status=%d", resp.StatusCode)
	}
	if !strings.Contains(resp.GetHeader("WWW-Authenticate"), `realm="example.com"`) {
		t.Errorf("WWW-Authenticate=%q", resp.GetHeader("WWW-Authenticate"))
	}

	var nilD *sipDigestAuth
	if _, err := nilD.challenge401(req); err == nil {
		t.Error("nil digest → error")
	}
}
