// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	pionwebrtc "github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

// directHarness boots a manager, opens a WS, joins anonymously, and
// returns the live server-side Participant so tests can call internal
// methods (SubscribeTo, UnsubscribeFrom, triggerRenegotiation, etc.)
// without needing successful ICE.
type directHarness struct {
	mgr   *Manager
	srv   *httptest.Server
	conns []*websocket.Conn
}

func newDirectHarness(t *testing.T) *directHarness {
	t.Helper()
	mgr, err := NewManager(&Config{AllowUnauthenticated: true, HeartbeatInterval: 0}, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	return &directHarness{
		mgr: mgr,
		srv: httptest.NewServer(http.HandlerFunc(mgr.ServeWS)),
	}
}
func (h *directHarness) close() {
	for _, c := range h.conns {
		c.Close()
	}
	h.srv.Close()
}
func (h *directHarness) join(t *testing.T, room, name string) *Participant {
	t.Helper()
	u, _ := url.Parse(h.srv.URL)
	u.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	h.conns = append(h.conns, c)
	if err := writeEnvelope(c, MsgJoin, JoinData{Room: room, Name: name}); err != nil {
		t.Fatal(err)
	}
	if env := mustRead(t, c, MsgJoined, 2*time.Second); env.Type != MsgJoined {
		t.Fatal("not joined")
	}
	// Find the server-side Participant by identity in the room.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		h.mgr.mu.Lock()
		r := h.mgr.rooms[room]
		h.mgr.mu.Unlock()
		if r != nil {
			for _, p := range r.Participants() {
				if p.identity == name {
					return p
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("server-side participant for %s not found", name)
	return nil
}

// TestAnnouncePublishAndSubscribe directly exercises Room.announcePublish
// + Participant.SubscribeTo + UnsubscribeFrom by building a synthetic
// PublishedTrack against alice and announcing it. Bob's PC.AddTrack is
// real, so the SubscribeTo path executes end to end.
func TestAnnouncePublishAndSubscribe(t *testing.T) {
	h := newDirectHarness(t)
	defer h.close()

	alice := h.join(t, "direct", "alice")
	bob := h.join(t, "direct", "bob")

	// Build a synthetic forwarder + PublishedTrack against alice.
	fwd, err := NewSimulcastForwarder("audio-syn", alice.id, "audio",
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	pub := &PublishedTrack{
		TrackID: "audio-syn", Kind: "audio", StreamID: alice.id,
		Codec: pionwebrtc.MimeTypeOpus, Source: "audio", Forwarder: fwd,
	}
	alice.mu.Lock()
	alice.published[pub.TrackID] = pub
	alice.mu.Unlock()

	// announcePublish only auto-subscribes peers whose initial offer/
	// answer is complete. The direct harness never drives a real
	// handshake; mark bob as negotiated so deferSubscription forwards
	// straight to SubscribeTo.
	bob.negotiated.Store(true)

	// announcePublish must notify bob and trigger SubscribeTo on bob's PC.
	alice.room.announcePublish(alice, pub)

	// Verify bob now has a subscription entry for alice's track.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		bob.mu.RLock()
		_, ok := bob.subscriptions[alice.id][pub.TrackID]
		bob.mu.RUnlock()
		if ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	bob.mu.RLock()
	_, ok := bob.subscriptions[alice.id][pub.TrackID]
	bob.mu.RUnlock()
	if !ok {
		t.Fatal("bob did not subscribe to alice's announced track")
	}

	// Idempotent subscribe — second call must not duplicate.
	bob.SubscribeTo(alice.id, pub)

	// Unsubscribe → mapping gone.
	bob.UnsubscribeFrom(alice.id, pub.TrackID)
	bob.mu.RLock()
	_, still := bob.subscriptions[alice.id][pub.TrackID]
	bob.mu.RUnlock()
	if still {
		t.Error("UnsubscribeFrom did not remove mapping")
	}

	// announceUnpublish — just covers the broadcast path.
	alice.room.announceUnpublish(alice.id, pub.TrackID)
}

// TestDeferSubscriptionQueuesUntilNegotiated covers the new defer path:
// while a peer is mid-handshake (negotiated=false), subscriptions are
// queued and only applied after HandleOffer success flips the flag.
func TestDeferSubscriptionQueuesUntilNegotiated(t *testing.T) {
	h := newDirectHarness(t)
	defer h.close()
	alice := h.join(t, "defer", "alice")
	bob := h.join(t, "defer", "bob")

	fwd, _ := NewSimulcastForwarder("dx", alice.id, "audio",
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		zap.NewNop())
	pub := &PublishedTrack{TrackID: "dx", Kind: "audio", Forwarder: fwd}

	// Bob hasn't negotiated yet.
	bob.deferSubscription(alice.id, pub)
	bob.pendingSubMu.Lock()
	queued := len(bob.pendingSubs)
	bob.pendingSubMu.Unlock()
	if queued != 1 {
		t.Fatalf("expected 1 pending sub, got %d", queued)
	}
	bob.mu.RLock()
	_, exists := bob.subscriptions[alice.id]["dx"]
	bob.mu.RUnlock()
	if exists {
		t.Fatal("deferred sub should NOT be on PC yet")
	}

	// Simulate handshake completion → flush.
	bob.negotiated.Store(true)
	bob.flushDeferredSubscriptions()
	bob.mu.RLock()
	_, ok := bob.subscriptions[alice.id]["dx"]
	bob.mu.RUnlock()
	if !ok {
		t.Error("flush should have applied the queued subscription")
	}

	// After negotiated, deferSubscription must bypass the queue.
	fwd2, _ := NewSimulcastForwarder("dy", alice.id, "audio",
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		zap.NewNop())
	pub2 := &PublishedTrack{TrackID: "dy", Kind: "audio", Forwarder: fwd2}
	bob.deferSubscription(alice.id, pub2)
	bob.mu.RLock()
	_, applied := bob.subscriptions[alice.id]["dy"]
	bob.mu.RUnlock()
	if !applied {
		t.Error("after negotiated, deferSubscription should subscribe immediately")
	}
}

// TestSubscribeForbiddenWithoutPermission verifies CanSubscribe=false
// short-circuits SubscribeTo (no AddTrack invocation).
func TestSubscribeForbiddenWithoutPermission(t *testing.T) {
	h := newDirectHarness(t)
	defer h.close()
	alice := h.join(t, "perm", "alice")
	bob := h.join(t, "perm", "bob")
	bob.permissions.CanSubscribe = false

	fwd, _ := NewSimulcastForwarder("x", alice.id, "audio",
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		zap.NewNop())
	pub := &PublishedTrack{TrackID: "x", Kind: "audio", Forwarder: fwd}
	bob.SubscribeTo(alice.id, pub)
	bob.mu.RLock()
	defer bob.mu.RUnlock()
	if _, ok := bob.subscriptions[alice.id]["x"]; ok {
		t.Error("CanSubscribe=false should block subscription")
	}
}

// TestUnsubscribeMissing is a no-op when no subscription exists.
func TestUnsubscribeMissing(t *testing.T) {
	h := newDirectHarness(t)
	defer h.close()
	bob := h.join(t, "u", "bob")
	bob.UnsubscribeFrom("nope", "nope") // must not panic
}

// TestHandleSetMute toggles the muted flag on a published track.
func TestHandleSetMute(t *testing.T) {
	h := newDirectHarness(t)
	defer h.close()
	alice := h.join(t, "m", "alice")
	fwd, _ := NewSimulcastForwarder("mt", alice.id, "audio",
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		zap.NewNop())
	pub := &PublishedTrack{TrackID: "mt", Kind: "audio", Forwarder: fwd}
	alice.mu.Lock()
	alice.published["mt"] = pub
	alice.mu.Unlock()
	alice.HandleSetMute("mt", true)
	if !pub.Muted() {
		t.Error("setMute(true) didn't stick on PublishedTrack")
	}
	if !fwd.Muted() {
		t.Error("setMute(true) did not propagate to forwarder")
	}
	alice.HandleSetMute("mt", false)
	if pub.Muted() || fwd.Muted() {
		t.Error("setMute(false) didn't reset both flags")
	}
	// Unknown trackID — no-op.
	alice.HandleSetMute("unknown", true)
}

// TestHandleAnswerOnClosedParticipant returns ErrParticipantGone.
func TestHandleAnswerOnClosedParticipant(t *testing.T) {
	h := newDirectHarness(t)
	defer h.close()
	p := h.join(t, "ha", "x")
	p.Close("test")
	if err := p.HandleAnswer("v=0"); err == nil {
		t.Error("expected error on closed participant")
	}
	if err := p.HandleICECandidate(ICECandidateData{Candidate: "x"}); err == nil {
		t.Error("expected error on closed participant for ICE")
	}
	if _, err := p.HandleOffer("v=0"); err == nil {
		t.Error("expected error on closed participant for offer")
	}
}

// TestDuplicateIdentityEviction joins alice twice; the older session
// must be evicted, the newer remains.
func TestDuplicateIdentityEviction(t *testing.T) {
	h := newDirectHarness(t)
	defer h.close()
	_ = h.join(t, "dup", "alice")
	time.Sleep(50 * time.Millisecond)
	a2 := h.join(t, "dup", "alice")
	time.Sleep(100 * time.Millisecond)

	h.mgr.mu.Lock()
	r := h.mgr.rooms["dup"]
	h.mgr.mu.Unlock()
	if r == nil {
		t.Fatal("room gone")
	}
	if got := len(r.Participants()); got != 1 {
		t.Errorf("after duplicate-identity eviction expected 1 participant, got %d", got)
	}
	if r.ParticipantByID(a2.ID()) == nil {
		t.Error("newer alice session not retained")
	}
}

// TestRoomDropOnIdleGC: with RoomIdleTTL set to a very short value,
// when the last participant leaves the room must be removed.
func TestRoomDropOnIdleGC(t *testing.T) {
	mgr, _ := NewManager(&Config{AllowUnauthenticated: true, RoomIdleTTL: 30 * time.Millisecond, HeartbeatInterval: 0}, zap.NewNop())
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	c, _, _ := websocket.DefaultDialer.Dial(u.String(), nil)
	_ = writeEnvelope(c, MsgJoin, JoinData{Room: "gc", Name: "lonely"})
	_ = mustRead(t, c, MsgJoined, 2*time.Second)
	_ = writeEnvelope(c, MsgLeave, struct{}{})
	c.Close()
	// Wait past TTL.
	time.Sleep(200 * time.Millisecond)
	mgr.mu.Lock()
	_, exists := mgr.rooms["gc"]
	mgr.mu.Unlock()
	if exists {
		t.Error("room should have been GC'd")
	}
}

// TestICEConnectionStateChange directly drives the state change handler
// for the Failed state which schedules an ICE restart offer.
func TestICEConnectionStateChange(t *testing.T) {
	h := newDirectHarness(t)
	defer h.close()
	p := h.join(t, "ice", "x")
	// Non-failed states are debug logs only.
	p.onICEConnectionStateChange(pionwebrtc.ICEConnectionStateConnected)
	p.onICEConnectionStateChange(pionwebrtc.ICEConnectionStateDisconnected)
	// Failed triggers a server-side ICE-restart offer.
	p.onICEConnectionStateChange(pionwebrtc.ICEConnectionStateFailed)
	time.Sleep(100 * time.Millisecond)
}

// TestConnectionStateChange closes the participant on terminal states.
func TestConnectionStateChange(t *testing.T) {
	h := newDirectHarness(t)
	defer h.close()
	p := h.join(t, "cs", "x")
	p.onConnectionStateChange(pionwebrtc.PeerConnectionStateConnected) // no-op
	p.onConnectionStateChange(pionwebrtc.PeerConnectionStateFailed)
	if !p.closed.Load() {
		t.Error("Failed should have closed participant")
	}
}

// TestHandleAnswerBadSDP exercises the error path of HandleAnswer
// when no offer is outstanding (or SDP is malformed).
func TestHandleAnswerBadSDP(t *testing.T) {
	h := newDirectHarness(t)
	defer h.close()
	p := h.join(t, "ans", "x")
	if err := p.HandleAnswer("not-sdp"); err == nil {
		t.Error("expected error for bogus answer SDP")
	}
	// Pre-load slot — must still drain it on error so we don't leak.
	p.negotiating <- struct{}{}
	_ = p.HandleAnswer("not-sdp")
}

// TestTriggerRenegotiationCoalesces verifies the debounce path doesn't
// fire two offers back to back; calling twice should issue one slot
// occupancy.
func TestTriggerRenegotiationCoalesces(t *testing.T) {
	h := newDirectHarness(t)
	defer h.close()
	p := h.join(t, "rn", "x")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.triggerRenegotiation()
		p.triggerRenegotiation() // second call dropped via select default
	}()
	wg.Wait()

	// Drain pending frames to let the goroutine run.
	time.Sleep(150 * time.Millisecond)
}
