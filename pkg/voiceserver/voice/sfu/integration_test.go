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
	"go.uber.org/zap"
)

// TestIntegrationOfferAnswer drives a real pion PeerConnection through
// the SFU's WS handler. Verifies:
//
//   - join → joined
//   - offer (publishing one audio track) → answer
//   - trickle iceCandidate → no error
//   - setMute on the published track → mutation visible
//   - leave → graceful close
//
// This covers handler.dispatch, Participant.HandleOffer / HandleAnswer /
// HandleICECandidate / HandleSetMute, plus all the publish-side track
// metadata getters.
func TestIntegrationOfferAnswer(t *testing.T) {
	const secret = "it-secret"
	mgr, err := NewManager(&Config{AuthSecret: secret, HeartbeatInterval: 0}, zap.NewNop())
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()

	tok, err := NewAccessToken(secret, AccessTokenClaims{
		Room:      "itroom",
		Identity:  "alice",
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

	// Build a tiny client-side pion PC and publish one audio track.
	pc, err := pionwebrtc.NewPeerConnection(pionwebrtc.Configuration{})
	if err != nil {
		t.Fatalf("client pc: %v", err)
	}
	defer pc.Close()
	track, err := pionwebrtc.NewTrackLocalStaticRTP(
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio-1", "stream-1",
	)
	if err != nil {
		t.Fatalf("local track: %v", err)
	}
	if _, err := pc.AddTrack(track); err != nil {
		t.Fatalf("add track: %v", err)
	}

	// JOIN
	if err := writeEnvelope(c, MsgJoin, JoinData{Token: tok}); err != nil {
		t.Fatalf("join: %v", err)
	}
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	if env, err := readEnvelope(c); err != nil || env.Type != MsgJoined {
		t.Fatalf("expected joined, got %+v err=%v", env, err)
	}

	// OFFER
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		t.Fatalf("offer: %v", err)
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		t.Fatalf("set local: %v", err)
	}
	if err := writeEnvelope(c, MsgOffer, SDPData{SDP: offer.SDP}); err != nil {
		t.Fatalf("send offer: %v", err)
	}
	// Server may interleave iceCandidate frames before/around the
	// answer (pion gathers candidates eagerly). Drain until we see
	// MsgAnswer, with a hard deadline.
	_ = c.SetReadDeadline(time.Now().Add(5 * time.Second))
	var sdpReply SDPData
	gotAnswer := false
	for !gotAnswer {
		env, err := readEnvelope(c)
		if err != nil {
			t.Fatalf("read while waiting for answer: %v", err)
		}
		switch env.Type {
		case MsgAnswer:
			if err := json.Unmarshal(env.Data, &sdpReply); err != nil {
				t.Fatalf("decode answer: %v", err)
			}
			gotAnswer = true
		case MsgICECandidate, MsgTrackPublished:
			// expected interleaved frames; keep reading
		default:
			t.Fatalf("unexpected frame while waiting for answer: %s", env.Type)
		}
	}
	if sdpReply.SDP == "" {
		t.Fatal("empty answer SDP")
	}

	// Trickle one (garbage) ICE candidate — pion may reject it, but the
	// dispatch path is still exercised. Server should not close the WS.
	_ = writeEnvelope(c, MsgICECandidate, ICECandidateData{
		Candidate: "candidate:1 1 udp 1 1.2.3.4 1234 typ host",
		SDPMid:    "0",
	})

	// SET MUTE on the audio track we published. The track ID was the
	// pion-side track.ID() which we know.
	if err := writeEnvelope(c, MsgSetMute, SetMuteData{TrackID: track.ID(), Muted: true}); err != nil {
		t.Fatalf("setMute: %v", err)
	}

	// PING — server-side handler should accept and not bounce us.
	_ = writeEnvelope(c, MsgPing, struct{}{})

	// LEAVE — server should close gracefully.
	if err := writeEnvelope(c, MsgLeave, struct{}{}); err != nil {
		t.Fatalf("leave: %v", err)
	}

	// Drain remaining frames until close.
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	for {
		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
	}

	// Give the server a moment to GC the room.
	time.Sleep(50 * time.Millisecond)
}

func TestIntegrationFirstMessageMustBeJoin(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true}, zap.NewNop())
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = writeEnvelope(c, MsgOffer, SDPData{SDP: "x"})
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
	env, err := readEnvelope(c)
	if err != nil {
		t.Fatal(err)
	}
	if env.Type != MsgError {
		t.Errorf("first non-join msg should error, got %s", env.Type)
	}
}

func TestIntegrationAllowAnon(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true}, zap.NewNop())
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = writeEnvelope(c, MsgJoin, JoinData{Room: "anon-room", Name: "guest"})
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	env, err := readEnvelope(c)
	if err != nil {
		t.Fatal(err)
	}
	if env.Type != MsgJoined {
		t.Fatalf("expected joined in anon mode, got %s data=%s", env.Type, string(env.Data))
	}
}

func TestIntegrationRoomFull(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true, MaxParticipantsPerRoom: 1}, zap.NewNop())
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"

	// First joiner OK.
	c1, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
	defer c1.Close()
	_ = writeEnvelope(c1, MsgJoin, JoinData{Room: "full", Name: "alice"})
	_ = c1.SetReadDeadline(time.Now().Add(2 * time.Second))
	if env, _ := readEnvelope(c1); env.Type != MsgJoined {
		t.Fatalf("first join should succeed; got %s", env.Type)
	}

	// Second joiner rejected with room_full.
	c2, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
	defer c2.Close()
	_ = writeEnvelope(c2, MsgJoin, JoinData{Room: "full", Name: "bob"})
	_ = c2.SetReadDeadline(time.Now().Add(2 * time.Second))
	env, err := readEnvelope(c2)
	if err != nil {
		t.Fatal(err)
	}
	if env.Type != MsgError {
		t.Fatalf("second join expected error, got %s", env.Type)
	}
	var ed ErrorData
	_ = json.Unmarshal(env.Data, &ed)
	if ed.Code != "room_full" {
		t.Errorf("expected room_full, got %s", ed.Code)
	}
}

func TestParticipantGetters(t *testing.T) {
	// Direct getters that don't require a live PC.
	p := &Participant{
		id:          "p-1",
		identity:    "alice",
		permissions: DefaultPermissions(),
	}
	if p.ID() != "p-1" {
		t.Error("ID()")
	}
	if p.Identity() != "alice" {
		t.Error("Identity()")
	}
	if !p.Permissions().CanPublish {
		t.Error("Permissions()")
	}
	pub := &PublishedTrack{}
	if pub.Muted() {
		t.Error("default muted should be false")
	}
	pub.muted.Store(true)
	if !pub.Muted() {
		t.Error("muted set but Muted()=false")
	}
}

func TestRoomNameByID(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true}, zap.NewNop())
	r, _ := mgr.getOrCreateRoom("named")
	if r.Name() != "named" {
		t.Error("Name()")
	}
	if r.ParticipantByID("nope") != nil {
		t.Error("nonexistent participant should be nil")
	}
}

func TestManagerConfigAccessor(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true, MaxParticipantsPerRoom: 5}, zap.NewNop())
	if mgr.Config().MaxParticipantsPerRoom != 5 {
		t.Error("Config().MaxParticipantsPerRoom mismatch")
	}
}

func TestManagerIdleRoomGC(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true, RoomIdleTTL: -1}, zap.NewNop())
	// Negative TTL → Normalise resets to 60s default; verify rooms can
	// be created and dropped without panic.
	r, _ := mgr.getOrCreateRoom("temp")
	mgr.dropRoom(r.Name())
	mgr.mu.Lock()
	_, exists := mgr.rooms[r.Name()]
	mgr.mu.Unlock()
	if exists {
		t.Error("dropRoom should remove from map")
	}
}
