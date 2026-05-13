// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"fmt"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	"github.com/pion/interceptor/pkg/nack"
	"github.com/pion/interceptor/pkg/twcc"
	pionwebrtc "github.com/pion/webrtc/v4"
)

// buildAPI builds a process-wide pion API with the codec set and
// interceptor chain we need for multi-party SFU forwarding.
//
// Differences from pkg/voice/webrtc/engine.go (the 1:1 AI-bridge):
//
//   - Register VP8 in addition to Opus. VP8 is the most broadly
//     compatible video codec without paying royalties (H.264 via openh264
//     has licensing constraints) and simulcast just works with it.
//   - Keep the intervalpli interceptor active — when we receive a new
//     subscriber we want pion to schedule PLIs for the publisher so the
//     new watcher can join mid-frame without waiting for the next
//     natural keyframe.
//
// UDPMux / SinglePort / PublicIPs mirror the webrtc package for uniform
// deployment semantics.
func buildAPI(cfg *Config) (*pionwebrtc.API, error) {
	m := &pionwebrtc.MediaEngine{}

	// Opus. channels=2 in SDP is the browser-accepted idiom even for
	// mono content; pion downmixes transparently. inband FEC is on.
	opus := pionwebrtc.RTPCodecParameters{
		RTPCodecCapability: pionwebrtc.RTPCodecCapability{
			MimeType:    pionwebrtc.MimeTypeOpus,
			ClockRate:   48000,
			Channels:    2,
			SDPFmtpLine: "minptime=10;useinbandfec=1;usedtx=0",
			RTCPFeedback: []pionwebrtc.RTCPFeedback{
				{Type: "nack"},
				{Type: "transport-cc"},
			},
		},
		PayloadType: 111,
	}
	if err := m.RegisterCodec(opus, pionwebrtc.RTPCodecTypeAudio); err != nil {
		return nil, fmt.Errorf("sfu: register opus: %w", err)
	}

	// VP8. The RTX pairing is important for NACK retransmissions — pion
	// registers the matching apt= line automatically.
	vp8 := pionwebrtc.RTPCodecParameters{
		RTPCodecCapability: pionwebrtc.RTPCodecCapability{
			MimeType:  pionwebrtc.MimeTypeVP8,
			ClockRate: 90000,
			RTCPFeedback: []pionwebrtc.RTCPFeedback{
				{Type: "goog-remb"},
				{Type: "transport-cc"},
				{Type: "ccm", Parameter: "fir"},
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
			},
		},
		PayloadType: 96,
	}
	if err := m.RegisterCodec(vp8, pionwebrtc.RTPCodecTypeVideo); err != nil {
		return nil, fmt.Errorf("sfu: register vp8: %w", err)
	}

	// Simulcast extensions. Browsers attach rid / mid / repaired-rid to
	// outbound RTP when they split a track into layers. Without these
	// extensions registered, pion would refuse the SDP.
	for _, ext := range []string{
		"urn:ietf:params:rtp-hdrext:sdes:mid",
		"urn:ietf:params:rtp-hdrext:sdes:rtp-stream-id",
		"urn:ietf:params:rtp-hdrext:sdes:repaired-rtp-stream-id",
	} {
		if err := m.RegisterHeaderExtension(pionwebrtc.RTPHeaderExtensionCapability{URI: ext}, pionwebrtc.RTPCodecTypeVideo); err != nil {
			return nil, fmt.Errorf("sfu: register %s: %w", ext, err)
		}
	}

	ir := &interceptor.Registry{}
	if err := pionwebrtc.RegisterDefaultInterceptors(m, ir); err != nil {
		return nil, fmt.Errorf("sfu: default interceptors: %w", err)
	}
	// NACK generator + responder = packet-loss recovery in both
	// directions. Needed for video; cheap for audio.
	nackGen, err := nack.NewGeneratorInterceptor()
	if err != nil {
		return nil, fmt.Errorf("sfu: nack gen: %w", err)
	}
	ir.Add(nackGen)
	nackResp, err := nack.NewResponderInterceptor()
	if err != nil {
		return nil, fmt.Errorf("sfu: nack resp: %w", err)
	}
	ir.Add(nackResp)
	// Transport-Wide Congestion Control — required by browser video
	// senders and the REMB replacement standard.
	twccGen, err := twcc.NewSenderInterceptor()
	if err != nil {
		return nil, fmt.Errorf("sfu: twcc: %w", err)
	}
	ir.Add(twccGen)
	// PLI scheduler keeps new video subscribers sane by forcing a
	// keyframe on subscription.
	pli, err := intervalpli.NewReceiverInterceptor()
	if err == nil {
		ir.Add(pli)
	}

	se := pionwebrtc.SettingEngine{}
	if len(cfg.PublicIPs) > 0 {
		se.SetNAT1To1IPs(cfg.PublicIPs, pionwebrtc.ICECandidateTypeHost)
	}
	if cfg.SinglePort > 0 {
		if err := se.SetEphemeralUDPPortRange(uint16(cfg.SinglePort), uint16(cfg.SinglePort)); err != nil {
			return nil, fmt.Errorf("sfu: single-port: %w", err)
		}
	}
	se.SetICETimeouts(15*time.Second, 25*time.Second, 2*time.Second)

	api := pionwebrtc.NewAPI(
		pionwebrtc.WithMediaEngine(m),
		pionwebrtc.WithInterceptorRegistry(ir),
		pionwebrtc.WithSettingEngine(se),
	)
	return api, nil
}

// iceServersForClient converts pion's ICEServer slice into the JSON
// shape that browsers accept natively through RTCPeerConnection
// constructors. pion's Credential field is interface{} for forward-
// compatibility with OAuth credentials; we only project the string
// case to the wire because browsers can't consume the struct form.
// Non-string credentials are silently omitted (empty string in the
// output) so a server-side misconfiguration doesn't leak a struct
// literal into client logs.
func iceServersForClient(servers []pionwebrtc.ICEServer) []ICEServerForClient {
	out := make([]ICEServerForClient, 0, len(servers))
	for _, s := range servers {
		cred := ""
		if v, ok := s.Credential.(string); ok {
			cred = v
		}
		out = append(out, ICEServerForClient{
			URLs:       s.URLs,
			Username:   s.Username,
			Credential: cred,
		})
	}
	return out
}
