package server

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"net"
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

func mkReq(method string) *stack.Message {
	m := &stack.Message{IsRequest: true, Method: method, RequestURI: "sip:server@127.0.0.1",
		Headers: map[string]string{}, HeadersMulti: map[string][]string{}}
	m.SetHeader("Via", "SIP/2.0/UDP 10.0.0.1;branch=z9hG4bK-" + method)
	m.SetHeader("From", "<sip:a@example.com>;tag=x")
	m.SetHeader("To", "<sip:server@127.0.0.1>")
	m.SetHeader("Call-ID", "hx-" + method)
	m.SetHeader("CSeq", "1 " + method)
	return m
}

// handleNotify — refer sub-event path + nil + delegation to presence handler.
func TestHandleNotify_ReferAnd200(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	if resp := s.handleNotify(nil, nil); resp != nil {
		t.Error("nil msg → nil")
	}

	// Event: refer → 200 OK
	n := mkReq("NOTIFY")
	n.SetHeader("Event", "refer")
	resp := s.handleNotify(n, &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5060})
	if resp == nil || resp.StatusCode != 200 {
		t.Errorf("refer-notify: %+v", resp)
	}

	// Event: presence → delegates to handleNotifyPresence (may return various)
	p := mkReq("NOTIFY")
	p.SetHeader("Event", "presence")
	_ = s.handleNotify(p, &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5060})
}

// handleUpdate: always 200 + Content-Length:0
func TestHandleUpdate(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	if resp := s.handleUpdate(nil, nil); resp != nil {
		t.Error("nil msg → nil")
	}
	var nilS *SIPServer
	if resp := nilS.handleUpdate(&stack.Message{}, nil); resp != nil {
		t.Error("nil server → nil")
	}
	m := mkReq("UPDATE")
	resp := s.handleUpdate(m, &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5060})
	if resp == nil || resp.StatusCode != 200 {
		t.Errorf("update: %+v", resp)
	}
}

// handleMessage: empty body → 200, non-empty → 415.
func TestHandleMessage_BodyBranches(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	if resp := s.handleMessage(nil, nil); resp != nil {
		t.Error("nil → nil")
	}

	empty := mkReq("MESSAGE")
	resp := s.handleMessage(empty, &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5060})
	if resp == nil || resp.StatusCode != 200 {
		t.Errorf("empty body: %+v", resp)
	}

	withBody := mkReq("MESSAGE")
	withBody.Body = "hello"
	resp = s.handleMessage(withBody, &net.UDPAddr{IP: net.ParseIP("10.0.0.1"), Port: 5060})
	if resp == nil || resp.StatusCode != 415 {
		t.Errorf("body: %+v", resp)
	}
	if resp.GetHeader("Accept") != "text/plain" {
		t.Errorf("Accept=%q", resp.GetHeader("Accept"))
	}
}

// SetInboundAllowUnknownDID + inboundAllowsUnknownDID round-trip.
func TestInboundAllowUnknownDID_RoundTrip(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()

	if s.inboundAllowsUnknownDID() {
		t.Error("default should be false")
	}
	s.SetInboundAllowUnknownDID(true)
	if !s.inboundAllowsUnknownDID() {
		t.Error("after set true")
	}
	s.SetInboundAllowUnknownDID(false)
	if s.inboundAllowsUnknownDID() {
		t.Error("after set false")
	}

	var nilS *SIPServer
	nilS.SetInboundAllowUnknownDID(true)
	if nilS.inboundAllowsUnknownDID() {
		t.Error("nil server should be false")
	}
}

// resolveInboundDIDBinding default path (no resolver → zero).
func TestResolveInboundDIDBinding_NoResolver(t *testing.T) {
	s := New(Config{LocalIP: "127.0.0.1"})
	defer s.Stop()
	b := s.resolveInboundDIDBinding(mkReq("INVITE"))
	if b.TenantID != 0 || b.TrunkNumberID != 0 {
		t.Errorf("no resolver → zero binding, got %+v", b)
	}
	// With resolver
	s.SetInboundDIDBindingResolver(func(*stack.Message) InboundDIDBinding {
		return InboundDIDBinding{TenantID: 42, TrunkNumberID: 7}
	})
	b2 := s.resolveInboundDIDBinding(mkReq("INVITE"))
	if b2.TenantID != 42 || b2.TrunkNumberID != 7 {
		t.Errorf("got %+v", b2)
	}

	var nilS *SIPServer
	_ = nilS.resolveInboundDIDBinding(mkReq("INVITE"))
}

// addrString / safe / safeI / preview
func TestMisc_Helpers(t *testing.T) {
	// addrString nil + good
	if got := addrString(nil); got != "" {
		t.Errorf("nil addr: %q", got)
	}
	a := &net.UDPAddr{IP: net.ParseIP("10.1.1.1"), Port: 5060}
	if got := addrString(a); got == "" {
		t.Error("non-nil addr empty")
	}

	// safe / safeI nil
	if got := safe(nil, func(m *stack.Message) string { return m.Method }); got != "" {
		t.Errorf("safe(nil): %q", got)
	}
	if got := safeI(nil, func(m *stack.Message) int { return m.StatusCode }); got != 0 {
		t.Errorf("safeI(nil): %d", got)
	}

	// non-nil branch
	m := &stack.Message{Method: "OPTIONS", StatusCode: 200}
	if got := safe(m, func(m *stack.Message) string { return m.Method }); got != "OPTIONS" {
		t.Errorf("safe: %q", got)
	}
	if got := safeI(m, func(m *stack.Message) int { return m.StatusCode }); got != 200 {
		t.Errorf("safeI: %d", got)
	}

	// preview truncation
	if got := preview("", 5); got != "" {
		t.Errorf("preview empty: %q", got)
	}
	if got := preview("hello world", 5); got == "" {
		t.Errorf("preview long: %q", got)
	}
	if got := preview("short", 100); got != "short" {
		t.Errorf("preview short: %q", got)
	}
}

// HangupInboundCall with active call path (real session registered).
func TestHangupInboundCall_WithRegisteredSession(t *testing.T) {
	srv := New(Config{Host: "127.0.0.1", Port: 0, LocalIP: "127.0.0.1"})
	if err := srv.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer srv.Stop()

	// No panic even with no call registered.
	srv.HangupInboundCall("nothing")

	// Observer receives both pre-hangup and cleanup.
	obs := &fakeObserver{preHangup: false}
	srv.SetCallLifecycleObserver(obs)
	srv.HangupInboundCall("no-session-call")
	if obs.preHits == 0 {
		t.Error("pre-hangup should fire even without session")
	}
	if obs.cleanupHits == 0 {
		t.Error("cleanup should fire even without session")
	}
}
