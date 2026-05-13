// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package webrtc

// End-to-end signaling smoke test. We can't drive real ASR / TTS without
// vendor credentials, but we *can* prove that:
//
//  1. POST /offer parses, builds a pion PeerConnection, gathers ICE,
//     and returns a valid SDP answer.
//  2. The answer contains the codec contract our engine is supposed to
//     advertise (Opus + useinbandfec=1 + transport-cc + nack).
//  3. A real second pion peer can SetRemoteDescription on that answer
//     and reach Connected over loopback ICE.
//
// This test runs entirely on localhost — no STUN, no TURN, no internet —
// and uses no vendor SDKs. It still exercises the full pion stack
// (DTLS-SRTP, ICE, interceptors).

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/recognizer"
	voicetts "github.com/LingByte/SoulNexus/pkg/voiceserver/voice/tts"

	pionwebrtc "github.com/pion/webrtc/v4"
)

// stubFactory returns nil services so startPipelines errors out cleanly
// — we only test signaling, not media bridging. The session will tear
// down on the first OnTrack which is fine for this test scope.
type stubFactory struct{}

func (stubFactory) NewASR(_ context.Context, _ string) (recognizer.TranscribeService, int, error) {
	return nil, 16000, errSkipASR
}
func (stubFactory) TTS(_ context.Context, _ string) (voicetts.Service, int, error) {
	return nil, 16000, errSkipTTS
}

var (
	errSkipASR = stubErr("asr disabled in test")
	errSkipTTS = stubErr("tts disabled in test")
)

type stubErr string

func (e stubErr) Error() string { return string(e) }

func TestServer_OfferAnswer_ProducesNegotiableSDP(t *testing.T) {
	srv, err := NewServer(ServerConfig{
		SessionFactory: stubFactory{},
		// Empty dialog URL is rejected by NewServer; fake one. The
		// session never actually dials it — pipelines fail on the
		// nil ASR service first, so the gateway dial code path
		// isn't reached during signaling.
		DialogWSURL: "ws://127.0.0.1:1/never",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Build a real offerer (the "browser") with the same default API so
	// codec negotiation works.
	api, err := BuildAPI(EngineConfig{})
	if err != nil {
		t.Fatal(err)
	}
	offerer, err := api.NewPeerConnection(pionwebrtc.Configuration{})
	if err != nil {
		t.Fatal(err)
	}
	defer offerer.Close()

	// The offer must contain at least one media line — add a recvonly
	// audio transceiver so pion emits an m=audio.
	if _, err := offerer.AddTransceiverFromKind(
		pionwebrtc.RTPCodecTypeAudio,
		pionwebrtc.RTPTransceiverInit{Direction: pionwebrtc.RTPTransceiverDirectionSendrecv},
	); err != nil {
		t.Fatal(err)
	}
	offer, err := offerer.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	gather := pionwebrtc.GatheringCompletePromise(offerer)
	if err := offerer.SetLocalDescription(offer); err != nil {
		t.Fatal(err)
	}
	<-gather
	finalOffer := offerer.LocalDescription()

	// POST it to the server's HTTP endpoint.
	httpSrv := httptest.NewServer(http.HandlerFunc(srv.HandleOffer))
	defer httpSrv.Close()

	body, _ := json.Marshal(SDPMessage{SDP: finalOffer.SDP, Type: "offer"})
	req, _ := http.NewRequest(http.MethodPost, httpSrv.URL, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var ans SDPMessage
	if err := json.NewDecoder(resp.Body).Decode(&ans); err != nil {
		t.Fatal(err)
	}
	if ans.Type != "answer" || strings.TrimSpace(ans.SDP) == "" {
		t.Fatalf("bad answer: %+v", ans)
	}
	if !strings.Contains(strings.ToLower(ans.SDP), "useinbandfec=1") {
		t.Errorf("answer missing useinbandfec=1:\n%s", ans.SDP)
	}
	if !strings.Contains(strings.ToLower(ans.SDP), "transport-cc") {
		t.Errorf("answer missing transport-cc:\n%s", ans.SDP)
	}
	if ans.CallID == "" {
		t.Error("answer missing call_id")
	}

	// Apply the answer to the offerer and wait for ICE Connected — the
	// ultimate proof that signaling produced a usable pairing. We wait
	// up to 5 seconds; on a healthy machine connection completes well
	// under 1 second on loopback.
	if err := offerer.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeAnswer, SDP: ans.SDP,
	}); err != nil {
		t.Fatal(err)
	}
	connected := make(chan struct{}, 1)
	var connectedOnce atomic.Bool
	offerer.OnICEConnectionStateChange(func(s pionwebrtc.ICEConnectionState) {
		if (s == pionwebrtc.ICEConnectionStateConnected ||
			s == pionwebrtc.ICEConnectionStateCompleted) &&
			connectedOnce.CompareAndSwap(false, true) {
			connected <- struct{}{}
		}
	})
	select {
	case <-connected:
		// ICE pairing completed end-to-end.
	case <-time.After(5 * time.Second):
		t.Fatalf("ICE never connected; offerer state=%s", offerer.ICEConnectionState())
	}
}
