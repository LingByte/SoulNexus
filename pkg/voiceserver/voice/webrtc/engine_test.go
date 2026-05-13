// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package webrtc

import (
	"strings"
	"testing"

	pionwebrtc "github.com/pion/webrtc/v4"
)

var (
	pcConfigEmpty = pionwebrtc.Configuration{}
	audioKind     = pionwebrtc.RTPCodecTypeAudio
	recvonlyInit  = pionwebrtc.RTPTransceiverInit{Direction: pionwebrtc.RTPTransceiverDirectionRecvonly}
)

func TestParseICEServers_StunOnly(t *testing.T) {
	out, err := ParseICEServers("stun:stun.l.google.com:19302")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || len(out[0].URLs) != 1 ||
		out[0].URLs[0] != "stun:stun.l.google.com:19302" {
		t.Fatalf("got %+v", out)
	}
	if out[0].Username != "" || out[0].Credential != nil {
		t.Fatalf("stun should not carry creds: %+v", out[0])
	}
}

func TestParseICEServers_TurnWithCreds(t *testing.T) {
	out, err := ParseICEServers("turn:turn.example.com:3478?username=alice&credential=secret")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("len: %d", len(out))
	}
	got := out[0]
	if got.URLs[0] != "turn:turn.example.com:3478" {
		t.Fatalf("url: %s", got.URLs[0])
	}
	if got.Username != "alice" || got.Credential != "secret" {
		t.Fatalf("creds: u=%q c=%v", got.Username, got.Credential)
	}
}

func TestParseICEServers_MixedAndWhitespace(t *testing.T) {
	out, err := ParseICEServers(" stun:s.example:19302 , turns:t.example:5349?username=u&credential=c ")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len: %d", len(out))
	}
	if out[0].URLs[0] != "stun:s.example:19302" {
		t.Fatalf("first: %s", out[0].URLs[0])
	}
	if !strings.HasPrefix(out[1].URLs[0], "turns:") {
		t.Fatalf("second: %s", out[1].URLs[0])
	}
}

func TestParseICEServers_RejectsBadScheme(t *testing.T) {
	if _, err := ParseICEServers("http://nope"); err == nil {
		t.Fatal("expected error for non-stun/turn scheme")
	}
}

func TestBuildAPI_OpusFECRegistered(t *testing.T) {
	api, err := BuildAPI(EngineConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if api == nil {
		t.Fatal("nil api")
	}
	// Indirect verification: the API can construct a peer connection
	// with our codec list, which means the MediaEngine has Opus.
	pc, err := api.NewPeerConnection(pcConfigEmpty)
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	// Add a recv-only audio transceiver and make an offer; the SDP must
	// contain `useinbandfec=1` and `transport-cc` for our config to be
	// considered correct.
	if _, err := pc.AddTransceiverFromKind(audioKind, recvonlyInit); err != nil {
		t.Fatal(err)
	}
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		t.Fatal(err)
	}
	sdp := strings.ToLower(offer.SDP)
	if !strings.Contains(sdp, "useinbandfec=1") {
		t.Errorf("offer missing useinbandfec=1:\n%s", offer.SDP)
	}
	if !strings.Contains(sdp, "transport-cc") {
		t.Errorf("offer missing transport-cc feedback:\n%s", offer.SDP)
	}
	if !strings.Contains(sdp, "nack") {
		t.Errorf("offer missing nack feedback:\n%s", offer.SDP)
	}
}
