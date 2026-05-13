// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"time"

	pionwebrtc "github.com/pion/webrtc/v4"
)

// Config is everything the SFU needs to serve a WS upgrade request.
// Fields left at zero get sensible defaults applied by Normalise().
type Config struct {
	// AuthSecret is the HMAC-SHA256 secret used to verify access
	// tokens presented by WebSocket clients. Required. The business
	// backend signs tokens with the same secret (see AccessToken).
	AuthSecret string

	// AllowUnauthenticated skips token verification entirely. Only useful
	// for local development; never enable in production.
	AllowUnauthenticated bool

	// MaxParticipantsPerRoom caps how many peers can be in one room.
	// Defaults to 16. Hitting the cap rejects the join with
	// ErrRoomFull.
	MaxParticipantsPerRoom int

	// MaxRooms caps how many distinct rooms can exist concurrently in
	// this Manager. 0 = use default (1024). -1 = unlimited. Joins that
	// would push beyond the cap are rejected with ErrTooManyRooms.
	MaxRooms int

	// AllowedOrigins is a whitelist matched against the Origin header
	// during WebSocket upgrade. Wildcard "*" allows any origin (the
	// previous always-allow behaviour). Empty list also allows any
	// origin — explicit so operators upgrading from an older config
	// don't accidentally lock themselves out. Matching is exact,
	// case-insensitive on scheme+host+port. Recommended for production:
	// ["https://app.example.com"].
	AllowedOrigins []string

	// RoomIdleTTL is how long a room sticks around with zero
	// participants before being garbage-collected. Defaults to 60s.
	// Set to 0 to GC immediately on empty.
	RoomIdleTTL time.Duration

	// HeartbeatInterval is how often the server sends a ping on each
	// WS connection. Defaults to 20s. Clients that miss 2 pings are
	// disconnected.
	HeartbeatInterval time.Duration

	// ICEServers are forwarded to every PeerConnection (and echoed back
	// to clients in the joined message so browsers can match). At least
	// one STUN server is recommended; a TURN server is required for
	// clients behind symmetric NAT.
	ICEServers []pionwebrtc.ICEServer

	// PublicIPs advertises NAT-1:1 host candidates. Required on
	// containerised deployments where the private bind address is not
	// reachable from the public internet. Leave empty otherwise.
	PublicIPs []string

	// SinglePort, if >0, pins all ICE UDP traffic to one port so only
	// one firewall rule is needed. When 0 pion allocates ephemeral
	// ports per peer.
	SinglePort int

	// EnableRecording turns on per-participant audio recording. When
	// true, RecordBucket is used as the storage bucket and the
	// recording lifecycle fires webhook events on finish. Video is
	// not currently recorded (Opus decode exists in the codebase; a
	// VP8 decode path does not, keeping the feature scoped).
	EnableRecording bool

	// RecordBucket is the pkg/stores bucket for recordings. Defaults
	// to "sfu-recordings".
	RecordBucket string

	// WebhookURL receives POSTed JSON events when set. Each request is
	// signed with X-SFU-Signature = hex(HMAC_SHA256(AuthSecret, body)).
	WebhookURL string

	// WebhookTimeout caps each webhook HTTP call. Defaults to 5s.
	WebhookTimeout time.Duration
}

// Normalise fills in defaults for zero-valued fields. Returns the same
// Config for chaining convenience.
func (c *Config) Normalise() *Config {
	if c.MaxParticipantsPerRoom <= 0 {
		c.MaxParticipantsPerRoom = 16
	}
	if c.MaxRooms == 0 {
		c.MaxRooms = 1024
	}
	// -1 stays as-is and is interpreted as "unlimited" by Manager.
	if c.RoomIdleTTL == 0 {
		c.RoomIdleTTL = 60 * time.Second
	}
	if c.HeartbeatInterval == 0 {
		c.HeartbeatInterval = 20 * time.Second
	}
	if c.RecordBucket == "" {
		c.RecordBucket = "sfu-recordings"
	}
	if c.WebhookTimeout == 0 {
		c.WebhookTimeout = 5 * time.Second
	}
	if len(c.ICEServers) == 0 {
		c.ICEServers = []pionwebrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		}
	}
	return c
}
