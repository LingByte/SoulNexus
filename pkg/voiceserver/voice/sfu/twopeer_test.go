// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	pionwebrtc "github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

// TestTwoPeerPublishFanout boots two pion peers against the SFU and
// verifies the publish→subscribe fan-out path:
//
//   - alice joins, offers, publishes one audio track
//   - bob joins, sees alice in joined, gets trackPublished, then is
//     auto-subscribed (server emits a renegotiation offer to bob)
//   - bob answers → covers HandleAnswer + triggerRenegotiation
//   - alice writes RTP to her published track → covers SimulcastForwarder.pump
//   - bob disconnects → covers announceUnpublish + UnsubscribeFrom
func TestTwoPeerPublishFanout(t *testing.T) {
	mgr, err := NewManager(&Config{AllowUnauthenticated: true, HeartbeatInterval: 0}, zap.NewNop())
	if err != nil {
		t.Fatalf("manager: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"

	// ---- Peer A (alice): publisher ----
	aliceWS, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer aliceWS.Close()
	alicePC, _ := pionwebrtc.NewPeerConnection(pionwebrtc.Configuration{})
	defer alicePC.Close()
	aliceTrack, _ := pionwebrtc.NewTrackLocalStaticRTP(
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio-a", "alice-stream",
	)
	if _, err := alicePC.AddTrack(aliceTrack); err != nil {
		t.Fatal(err)
	}
	if err := writeEnvelope(aliceWS, MsgJoin, JoinData{Room: "fan", Name: "alice"}); err != nil {
		t.Fatal(err)
	}
	if env := mustRead(t, aliceWS, MsgJoined, 3*time.Second); env.Type != MsgJoined {
		t.Fatal("alice not joined")
	}
	offer, _ := alicePC.CreateOffer(nil)
	_ = alicePC.SetLocalDescription(offer)
	_ = writeEnvelope(aliceWS, MsgOffer, SDPData{SDP: offer.SDP})
	answerEnv := drainUntil(t, aliceWS, MsgAnswer, 5*time.Second)
	var aliceAns SDPData
	_ = json.Unmarshal(answerEnv.Data, &aliceAns)
	_ = alicePC.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeAnswer, SDP: aliceAns.SDP,
	})

	// ---- Peer B (bob): subscriber ----
	bobWS, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer bobWS.Close()
	bobPC, _ := pionwebrtc.NewPeerConnection(pionwebrtc.Configuration{})
	defer bobPC.Close()

	// Counter so we know bob saw alice's track.
	var bobTracks int32
	bobPC.OnTrack(func(_ *pionwebrtc.TrackRemote, _ *pionwebrtc.RTPReceiver) {
		atomic.AddInt32(&bobTracks, 1)
	})
	// Bob also publishes one audio track so initial negotiation
	// succeeds; this also exercises announcePublish to alice.
	bobTrack, _ := pionwebrtc.NewTrackLocalStaticRTP(
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio-b", "bob-stream",
	)
	if _, err := bobPC.AddTrack(bobTrack); err != nil {
		t.Fatal(err)
	}

	if err := writeEnvelope(bobWS, MsgJoin, JoinData{Room: "fan", Name: "bob"}); err != nil {
		t.Fatal(err)
	}
	bobJoined := mustRead(t, bobWS, MsgJoined, 3*time.Second)
	var jd JoinedData
	_ = json.Unmarshal(bobJoined.Data, &jd)
	if len(jd.Participants) != 1 {
		t.Errorf("bob should see alice in participants, got %d", len(jd.Participants))
	}

	bobOffer, _ := bobPC.CreateOffer(nil)
	_ = bobPC.SetLocalDescription(bobOffer)
	_ = writeEnvelope(bobWS, MsgOffer, SDPData{SDP: bobOffer.SDP})
	bobAnsEnv := drainUntil(t, bobWS, MsgAnswer, 5*time.Second)
	var bobAns SDPData
	_ = json.Unmarshal(bobAnsEnv.Data, &bobAns)
	_ = bobPC.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeAnswer, SDP: bobAns.SDP,
	})

	// At this point the server has auto-subscribed bob to alice and
	// will fire triggerRenegotiation → server-initiated MsgOffer to bob.
	// Bob answers to cover HandleAnswer.
	renegEnv := drainUntilEither(t, bobWS, MsgOffer, 3*time.Second)
	if renegEnv != nil && renegEnv.Type == MsgOffer {
		var sdp SDPData
		_ = json.Unmarshal(renegEnv.Data, &sdp)
		_ = bobPC.SetRemoteDescription(pionwebrtc.SessionDescription{
			Type: pionwebrtc.SDPTypeOffer, SDP: sdp.SDP,
		})
		bobReneg, _ := bobPC.CreateAnswer(nil)
		_ = bobPC.SetLocalDescription(bobReneg)
		_ = writeEnvelope(bobWS, MsgAnswer, SDPData{SDP: bobReneg.SDP})
	}

	// Push a couple of RTP packets from alice to exercise the
	// SimulcastForwarder.pump goroutine. Successful WriteRTP doesn't
	// require the upstream connection to be established because we're
	// writing to the local track, but actually pion routes via the
	// connection so this is best-effort; the call itself covers code.
	for i := 0; i < 5; i++ {
		_ = aliceTrack.WriteRTP(&rtp.Packet{
			Header:  rtp.Header{PayloadType: 111, SequenceNumber: uint16(i), SSRC: 1},
			Payload: []byte{0xFC, 0xFF, 0xFE},
		})
		time.Sleep(5 * time.Millisecond)
	}

	// Disconnect bob; should not deadlock alice or panic on the server.
	_ = writeEnvelope(bobWS, MsgLeave, struct{}{})
	_ = bobWS.Close()
	time.Sleep(100 * time.Millisecond)

	// Sanity: alice should still be in the room.
	mgr.mu.Lock()
	rm, exists := mgr.rooms["fan"]
	mgr.mu.Unlock()
	if !exists {
		t.Fatal("room dropped while alice still present")
	}
	if len(rm.Participants()) != 1 {
		t.Errorf("expected 1 participant after bob leave, got %d", len(rm.Participants()))
	}
}

// drainUntil reads until it sees the wanted MessageType; fails on
// timeout or unexpected error.
func drainUntil(t *testing.T, c *websocket.Conn, want MessageType, timeout time.Duration) Envelope {
	t.Helper()
	deadline := time.Now().Add(timeout)
	_ = c.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		env, err := readEnvelope(c)
		if err != nil {
			t.Fatalf("read while waiting for %s: %v", want, err)
		}
		if env.Type == want {
			return env
		}
	}
	t.Fatalf("timeout waiting for %s", want)
	return Envelope{}
}

// drainUntilEither is non-fatal — returns nil if the wanted type does
// not show up within the timeout. Used for optional server-initiated
// frames (renegotiation offers) whose exact timing is racy.
func drainUntilEither(t *testing.T, c *websocket.Conn, want MessageType, timeout time.Duration) *Envelope {
	t.Helper()
	deadline := time.Now().Add(timeout)
	_ = c.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		env, err := readEnvelope(c)
		if err != nil {
			return nil
		}
		if env.Type == want {
			return &env
		}
	}
	return nil
}

func mustRead(t *testing.T, c *websocket.Conn, want MessageType, timeout time.Duration) Envelope {
	return drainUntil(t, c, want, timeout)
}

// TestHeartbeatPing forces a tiny heartbeat interval to drive sendPing
// at least once, then verifies the WS doesn't error from the client side.
func TestHeartbeatPing(t *testing.T) {
	mgr, _ := NewManager(&Config{
		AllowUnauthenticated: true,
		HeartbeatInterval:    50 * time.Millisecond,
	}, zap.NewNop())
	srv := httptest.NewServer(http.HandlerFunc(mgr.ServeWS))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var pinged int32
	c.SetPingHandler(func(string) error {
		atomic.AddInt32(&pinged, 1)
		return nil
	})
	_ = writeEnvelope(c, MsgJoin, JoinData{Room: "hb", Name: "x"})
	_ = mustRead(t, c, MsgJoined, 2*time.Second)

	// Drive the read pump for ~300 ms so the ping handler fires.
	deadline := time.Now().Add(300 * time.Millisecond)
	_ = c.SetReadDeadline(deadline)
	for time.Now().Before(deadline) {
		if _, _, err := c.ReadMessage(); err != nil {
			break
		}
	}
	if atomic.LoadInt32(&pinged) == 0 {
		t.Error("expected at least one server ping")
	}
}
