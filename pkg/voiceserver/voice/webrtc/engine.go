// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package webrtc

// engine.go centralises the pion API construction. Building an `API` is the
// expensive, configuration-heavy step in pion: registering codecs (with the
// exact RTP feedback list every browser expects), wiring the interceptor
// chain (NACK / TWCC / RTCP report intervals), and tuning the ICE settings
// for our deployment topology.
//
// A single API is built once at server start and shared across every
// PeerConnection. This is both faster (no per-call re-registration) and
// safer — interceptor state machines that span the lifetime of the process
// (e.g. RTCP receiver reports) work correctly only when shared.

import (
	"fmt"
	"strings"
	"time"

	"github.com/pion/ice/v4"
	"github.com/pion/interceptor"
	"github.com/pion/interceptor/pkg/intervalpli"
	"github.com/pion/interceptor/pkg/nack"
	"github.com/pion/interceptor/pkg/twcc"
	pionwebrtc "github.com/pion/webrtc/v4"
)

// EngineConfig configures the per-process pion API used by every call.
type EngineConfig struct {
	// ICEServers is the STUN / TURN server list passed to every peer.
	// At minimum supply a public STUN server (e.g.
	// stun:stun.l.google.com:19302); add a TURN server for clients
	// behind symmetric NATs.
	ICEServers []pionwebrtc.ICEServer

	// PublicIPs lists addresses reachable from the public internet that
	// should be advertised as ICE host candidates. Required when the
	// process runs in a container / behind a 1:1 NAT — pion otherwise
	// only sees the private bind address. Empty = no NAT 1:1.
	PublicIPs []string

	// UDPMux, when set, multiplexes all ICE UDP traffic onto a single
	// listening socket. Recommended in production: only one UDP port has
	// to be opened on the firewall regardless of concurrent calls.
	// nil = pion ephemerally allocates per call.
	UDPMux ice.UDPMux

	// SinglePort, if >0 and UDPMux is nil, makes pion bind one fixed UDP
	// port for ICE instead of an ephemeral range. Easier to firewall.
	SinglePort int
}

// BuildAPI constructs a *pionwebrtc.API with our codec + interceptor
// configuration baked in. The returned API is goroutine-safe and intended
// to be reused across every PeerConnection in the process.
func BuildAPI(cfg EngineConfig) (*pionwebrtc.API, error) {
	m := &pionwebrtc.MediaEngine{}
	if err := registerOpusWithFEC(m); err != nil {
		return nil, fmt.Errorf("webrtc: register opus: %w", err)
	}

	ir := &interceptor.Registry{}
	// Default RTCP / sender / receiver report interceptors. These keep the
	// remote bandwidth estimator fed and let pion emit the standard
	// receiver reports the browser expects.
	if err := pionwebrtc.RegisterDefaultInterceptors(m, ir); err != nil {
		return nil, fmt.Errorf("webrtc: default interceptors: %w", err)
	}
	// NACK generator/responder: when an inbound RTP packet is missing,
	// pion sends an RTCP NACK so the peer retransmits. Combined with
	// Opus inband FEC this gives us two layers of loss recovery.
	nackGen, err := nack.NewGeneratorInterceptor()
	if err != nil {
		return nil, fmt.Errorf("webrtc: nack generator: %w", err)
	}
	ir.Add(nackGen)
	nackResp, err := nack.NewResponderInterceptor()
	if err != nil {
		return nil, fmt.Errorf("webrtc: nack responder: %w", err)
	}
	ir.Add(nackResp)
	// Transport-Wide Congestion Control. Audio-only sessions don't need
	// rate adaptation as urgently as video, but TWCC still gives us
	// per-packet RTT and loss telemetry that we'll surface in the future
	// for adaptive Opus bitrate / DTX decisions.
	twccGen, err := twcc.NewSenderInterceptor()
	if err != nil {
		return nil, fmt.Errorf("webrtc: twcc: %w", err)
	}
	ir.Add(twccGen)
	// PLI scheduler is a no-op for audio (PLI is video-only) but is
	// cheap and keeps the registry symmetrical with future video work.
	pliInt, err := intervalpli.NewReceiverInterceptor()
	if err == nil {
		ir.Add(pliInt)
	}

	// SettingEngine governs the IO surface — bind addresses, NAT 1:1,
	// UDP mux, ICE timers. These are the knobs that matter on real
	// production deployments.
	se := pionwebrtc.SettingEngine{}
	if len(cfg.PublicIPs) > 0 {
		// Tells pion: I'm bound on a private address but I'm reachable
		// from the public internet via these IPs; advertise them as
		// host candidates. This is what makes single-host deployments
		// work without needing a TURN relay for every call.
		se.SetNAT1To1IPs(cfg.PublicIPs, pionwebrtc.ICECandidateTypeHost)
	}
	if cfg.UDPMux != nil {
		se.SetICEUDPMux(cfg.UDPMux)
	} else if cfg.SinglePort > 0 {
		// Pin ICE to one UDP port. Firewall config trivialises.
		if err := se.SetEphemeralUDPPortRange(uint16(cfg.SinglePort), uint16(cfg.SinglePort)); err != nil {
			return nil, fmt.Errorf("webrtc: single-port: %w", err)
		}
	}
	// Loosen ICE disconnect timers: a 2-second WiFi blip shouldn't tear
	// down an AI call; the application-layer timeouts (dialog idle) take
	// over after a real outage.
	se.SetICETimeouts(15*time.Second, 25*time.Second, 2*time.Second)

	api := pionwebrtc.NewAPI(
		pionwebrtc.WithMediaEngine(m),
		pionwebrtc.WithInterceptorRegistry(ir),
		pionwebrtc.WithSettingEngine(se),
	)
	return api, nil
}

// registerOpusWithFEC declares the Opus codec to the MediaEngine with the
// exact rtcp-fb / fmtp lines browsers expect. The `useinbandfec=1` fmtp
// is what unlocks Opus's per-packet forward error correction — receivers
// can reconstruct one lost packet from the next packet's FEC payload.
func registerOpusWithFEC(m *pionwebrtc.MediaEngine) error {
	// Browsers historically allocate dynamic payload type 111 for Opus;
	// using the same value avoids a re-mapping step inside the user
	// agent. Channels=2 in the SDP line is browser tradition even
	// though we treat audio as mono internally — Opus stereo packets
	// downmix correctly in our decoder.
	codec := pionwebrtc.RTPCodecParameters{
		RTPCodecCapability: pionwebrtc.RTPCodecCapability{
			MimeType:    pionwebrtc.MimeTypeOpus,
			ClockRate:   48000,
			Channels:    2,
			SDPFmtpLine: "minptime=10;useinbandfec=1;usedtx=0",
			RTCPFeedback: []pionwebrtc.RTCPFeedback{
				{Type: "nack"},                 // packet loss recovery
				{Type: "transport-cc"},         // TWCC bandwidth feedback
				{Type: "ccm", Parameter: "fir"}, // benign for audio; harmless
			},
		},
		PayloadType: 111,
	}
	return m.RegisterCodec(codec, pionwebrtc.RTPCodecTypeAudio)
}

// ParseICEServers turns a comma-separated env-var spec into a slice of
// ICEServer entries. Format per entry:
//
//	scheme://host:port[?username=U&credential=C]
//
// where scheme is one of stun, turn, turns. Credentials only apply to
// turn:/turns:. Whitespace around entries is trimmed.
func ParseICEServers(spec string) ([]pionwebrtc.ICEServer, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	out := make([]pionwebrtc.ICEServer, 0)
	for _, raw := range strings.Split(spec, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		entry, err := parseOneICEServer(raw)
		if err != nil {
			return nil, fmt.Errorf("webrtc: ice server %q: %w", raw, err)
		}
		out = append(out, entry)
	}
	return out, nil
}

func parseOneICEServer(raw string) (pionwebrtc.ICEServer, error) {
	// Split URL from optional ?username=...&credential=... query string
	// without pulling in net/url (the URL is non-standard — pion accepts
	// stun: / turn: schemes that net/url parses oddly).
	url, query := raw, ""
	if i := strings.IndexByte(raw, '?'); i >= 0 {
		url, query = raw[:i], raw[i+1:]
	}
	if !strings.HasPrefix(url, "stun:") &&
		!strings.HasPrefix(url, "turn:") &&
		!strings.HasPrefix(url, "turns:") {
		return pionwebrtc.ICEServer{}, fmt.Errorf("scheme must be stun:/turn:/turns:")
	}
	srv := pionwebrtc.ICEServer{URLs: []string{url}}
	for _, kv := range strings.Split(query, "&") {
		if kv == "" {
			continue
		}
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		switch strings.ToLower(k) {
		case "username":
			srv.Username = v
		case "credential":
			srv.Credential = v
		}
	}
	return srv, nil
}
