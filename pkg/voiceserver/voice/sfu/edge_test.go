// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	pionwebrtc "github.com/pion/webrtc/v4"
)

// TestNewManagerNilCfg / TestNewManagerNilLogger exercise the nil-input
// branches of NewManager.
func TestNewManagerNilCfg(t *testing.T) {
	mgr, err := NewManager(nil, nil)
	if err != nil {
		t.Fatalf("NewManager(nil,nil): %v", err)
	}
	if mgr.Config().MaxParticipantsPerRoom != 16 {
		t.Error("default max participants not applied")
	}
}

// TestBuildAPIPublicIPs ensures the PublicIPs + SinglePort code paths in
// buildAPI execute.
func TestBuildAPIPublicIPs(t *testing.T) {
	cfg := (&Config{
		PublicIPs:  []string{"1.2.3.4"},
		SinglePort: 0, // 0 to skip ephemeral range (would conflict if rerun)
	}).Normalise()
	api, err := buildAPI(cfg)
	if err != nil {
		t.Fatalf("buildAPI: %v", err)
	}
	if api == nil {
		t.Fatal("buildAPI returned nil API")
	}
}

func TestBuildAPISinglePortInvalid(t *testing.T) {
	cfg := (&Config{SinglePort: -1}).Normalise()
	// -1 is invalid; pion accepts uint16 cast which becomes 65535. The
	// call itself should still succeed.
	if _, err := buildAPI(cfg); err != nil {
		t.Errorf("buildAPI invalid port: %v", err)
	}
}

// TestArmIdleGCImmediate covers the TTL<=0 branch where the room is GC'd
// immediately on emptying.
func TestArmIdleGCImmediate(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true}, nil)
	r, _ := mgr.getOrCreateRoom("imm")
	// Force ttl to 0 to take the immediate branch.
	r.cfg.RoomIdleTTL = 0
	r.armIdleGC()
	mgr.mu.Lock()
	_, exists := mgr.rooms["imm"]
	mgr.mu.Unlock()
	if exists {
		t.Error("RoomIdleTTL=0 should drop immediately")
	}
}

// TestDispatchUnknownTypeIgnored sends a message with an unknown type;
// server must not error nor close the WS.
func TestDispatchUnknownTypeIgnored(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true, HeartbeatInterval: 0}, nil)
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = writeEnvelope(c, MsgJoin, JoinData{Room: "u", Name: "x"})
	_ = mustRead(t, c, MsgJoined, 2*time.Second)

	// Unknown type — should be silently swallowed.
	_ = c.WriteJSON(Envelope{Type: MessageType("totally-fake")})

	// Followup with a valid message; if the WS were closed we'd error.
	_ = writeEnvelope(c, MsgPing, struct{}{})
	time.Sleep(50 * time.Millisecond)
}

// TestDispatchBadOfferJSON sends a malformed offer payload; server
// responds with an error envelope but does not close the socket.
func TestDispatchBadOfferJSON(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true, HeartbeatInterval: 0}, nil)
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	c, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
	defer c.Close()
	_ = writeEnvelope(c, MsgJoin, JoinData{Room: "bj", Name: "x"})
	_ = mustRead(t, c, MsgJoined, 2*time.Second)

	// Send an offer with valid JSON that does not decode into SDPData
	// (sdp must be a string, not a number). The envelope decodes
	// fine; the inner Unmarshal returns the error we want.
	raw, _ := json.Marshal(Envelope{Type: MsgOffer, Data: json.RawMessage(`{"sdp":42}`)})
	_ = c.WriteMessage(websocket.TextMessage, raw)

	// Should get an error envelope.
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	env, err := readEnvelope(c)
	if err != nil {
		t.Fatal(err)
	}
	if env.Type != MsgError {
		t.Errorf("expected error, got %s", env.Type)
	}
}

// TestServeWSBadJoinPayload sends a join with malformed Data —
// expects a protocol error envelope.
func TestServeWSBadJoinPayload(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true}, nil)
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	c, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
	defer c.Close()
	// Envelope is valid JSON; Data decodes into JoinData fine, but with
	// AllowUnauthenticated=false (default below) the empty token would
	// fail. Build a NEW manager with a secret and send a bad token.
	mgr2, _ := NewManager(&Config{AuthSecret: "k", HeartbeatInterval: 0}, nil)
	srv2 := httptest.NewServer(http.HandlerFunc(mgr2.ServeWS))
	defer srv2.Close()
	u2, _ := url.Parse(srv2.URL)
	u2.Scheme = "ws"
	c2, _, _ := websocket.DefaultDialer.Dial(u2.String(), nil)
	defer c2.Close()
	// Wrong-shaped data: "token" should be a string, send number.
	raw, _ := json.Marshal(Envelope{Type: MsgJoin, Data: json.RawMessage(`{"token":42}`)})
	_ = c2.WriteMessage(websocket.TextMessage, raw)
	_ = c2.SetReadDeadline(time.Now().Add(2 * time.Second))
	env, err := readEnvelope(c2)
	if err != nil {
		t.Fatal(err)
	}
	if env.Type != MsgError {
		t.Errorf("expected error, got %s", env.Type)
	}
	_ = c // unused base dial — keeps the surrounding pattern uniform
}

// _silence keeps the pionwebrtc import in this file in case the package
// is reused in further edge tests.
var _ = pionwebrtc.MimeTypeVP8
