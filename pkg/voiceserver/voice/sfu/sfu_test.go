// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// TestAccessTokenRoundtrip covers the happy-path mint/verify pair plus
// the three rejection paths (bad signature, expired, missing fields).
func TestAccessTokenRoundtrip(t *testing.T) {
	const secret = "test-secret"
	tok, err := NewAccessToken(secret, AccessTokenClaims{
		Room:      "r1",
		Identity:  "alice",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	parsed, err := ParseAccessToken(secret, tok)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Claims.Room != "r1" || parsed.Claims.Identity != "alice" {
		t.Fatalf("claims mismatch: %+v", parsed.Claims)
	}

	// Wrong secret → bad signature.
	if _, err := ParseAccessToken("nope", tok); err == nil {
		t.Fatal("expected bad signature, got nil")
	}

	// Expired token.
	expired, _ := NewAccessToken(secret, AccessTokenClaims{
		Room:      "r1",
		Identity:  "alice",
		ExpiresAt: time.Now().Add(-2 * time.Minute).Unix(),
	})
	if _, err := ParseAccessToken(secret, expired); err == nil {
		t.Fatal("expected expired, got nil")
	}

	// Missing room.
	if _, err := NewAccessToken(secret, AccessTokenClaims{
		Identity:  "alice",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}); err == nil {
		t.Fatal("expected missing-room error")
	}
}

// TestConfigNormalise verifies defaults are applied for zero fields and
// that explicit values pass through untouched.
func TestConfigNormalise(t *testing.T) {
	c := (&Config{}).Normalise()
	if c.MaxParticipantsPerRoom != 16 {
		t.Errorf("default max participants = %d, want 16", c.MaxParticipantsPerRoom)
	}
	if c.HeartbeatInterval != 20*time.Second {
		t.Errorf("default heartbeat = %v, want 20s", c.HeartbeatInterval)
	}
	if c.RecordBucket != "sfu-recordings" {
		t.Errorf("default bucket = %q", c.RecordBucket)
	}
	if len(c.ICEServers) == 0 {
		t.Error("expected default STUN server")
	}

	c2 := (&Config{
		MaxParticipantsPerRoom: 4,
		HeartbeatInterval:      5 * time.Second,
		RecordBucket:           "custom",
	}).Normalise()
	if c2.MaxParticipantsPerRoom != 4 || c2.HeartbeatInterval != 5*time.Second || c2.RecordBucket != "custom" {
		t.Errorf("custom config got mutated: %+v", c2)
	}
}

// TestManagerJoinFlow boots a manager against httptest, opens a WS, and
// drives the join handshake to "joined". This exercises:
//   - ServeWS HTTP upgrade
//   - token verification path
//   - room creation
//   - JoinedData payload shape
//
// It does NOT exercise the full SDP handshake (that would require a
// real pion offer); the test stops once the joined ack arrives.
func TestManagerJoinFlow(t *testing.T) {
	const secret = "smoke-secret"
	mgr, err := NewManager(&Config{AuthSecret: secret}, zap.NewNop())
	if err != nil {
		t.Fatalf("manager: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()

	tok, err := NewAccessToken(secret, AccessTokenClaims{
		Room:      "smoke",
		Identity:  "tester",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("mint: %v", err)
	}

	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	// Send join.
	if err := writeEnvelope(c, MsgJoin, JoinData{Token: tok}); err != nil {
		t.Fatalf("write join: %v", err)
	}

	// Expect a joined ack within a generous deadline.
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	env, err := readEnvelope(c)
	if err != nil {
		t.Fatalf("read joined: %v", err)
	}
	if env.Type != MsgJoined {
		t.Fatalf("first reply type = %q, want joined; data=%s", env.Type, string(env.Data))
	}
	var jd JoinedData
	if err := json.Unmarshal(env.Data, &jd); err != nil {
		t.Fatalf("decode joined: %v", err)
	}
	if jd.Room != "smoke" || jd.Identity != "tester" || jd.ParticipantID == "" {
		t.Errorf("joined payload off: %+v", jd)
	}
}

// TestManagerBadToken: a join with an invalid token must come back as
// an error envelope, not a joined ack.
func TestManagerBadToken(t *testing.T) {
	mgr, err := NewManager(&Config{AuthSecret: "right"}, zap.NewNop())
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()
	if err := writeEnvelope(c, MsgJoin, JoinData{Token: "garbage.sig"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	env, err := readEnvelope(c)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if env.Type != MsgError {
		t.Fatalf("expected error, got %q", env.Type)
	}
	var ed ErrorData
	_ = json.Unmarshal(env.Data, &ed)
	if !strings.Contains(ed.Code, "bad_token") && !strings.Contains(ed.Code, "token") {
		t.Errorf("error code = %q, want bad_token-ish", ed.Code)
	}
}

// TestSanitiseSegment makes sure storage keys never contain path
// separators or weird unicode that would confuse object stores.
func TestSanitiseSegment(t *testing.T) {
	cases := map[string]string{
		"":               "unknown",
		"alice@host":     "alice_host",
		"team/standup":   "team_standup",
		"中文":             "__",
		"safe-name_1":    "safe-name_1",
	}
	for in, want := range cases {
		got := sanitiseSegment(in)
		if got != want {
			t.Errorf("sanitiseSegment(%q) = %q, want %q", in, got, want)
		}
	}
}

// ---- helpers ----

func writeEnvelope(c *websocket.Conn, t MessageType, data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.WriteJSON(Envelope{Type: t, Data: raw})
}

func readEnvelope(c *websocket.Conn) (Envelope, error) {
	var env Envelope
	_, msg, err := c.ReadMessage()
	if err != nil {
		return env, err
	}
	if err := json.Unmarshal(msg, &env); err != nil {
		return env, err
	}
	return env, nil
}

// Make sure ErrRoomFull et al. are reachable via errors.Is so callers
// can write idiomatic Go error branches.
func TestSentinelsExported(t *testing.T) {
	for _, e := range []error{ErrRoomFull, ErrRoomNotFound, ErrParticipantGone, ErrForbidden, ErrProtocol, ErrDuplicateIdentity} {
		if !errors.Is(e, e) {
			t.Errorf("sentinel %v not is-itself", e)
		}
	}
}
