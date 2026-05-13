// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package sfu implements a minimal but production-oriented Selective
// Forwarding Unit (SFU) for multi-party realtime calls.
//
// Architecture at a glance
//
//	browser ────WS /sfu/ws?token=… ───► signaling.go ◄──► Room
//	  ▲                                                    │
//	  │           RTP (Opus / VP8 simulcast)               │
//	  └──────────── pion PeerConnection ◄──────────────────┘
//
// A Room owns a set of Participants. Each Participant is a single
// bidirectional pion PeerConnection that the participant both publishes
// tracks to and subscribes other participants' tracks from. When a new
// track appears in the room, the Room fans it out to every other
// participant by minting a TrackLocalStaticRTP and driving pion's
// renegotiation flow ("offer coming from server" → client answer).
//
// What is included
//
//   - WebSocket signaling with a small typed JSON protocol (protocol.go)
//   - HMAC-signed access tokens carrying room + identity + permissions
//     (auth.go) — the server never minted its own, the business backend
//     does, sharing a single secret
//   - Audio (Opus) and video (VP8) forwarding with VP8 simulcast — the
//     SFU passes the highest layer it has for every subscriber. Layer
//     selection can be driven by client messages later.
//   - ICE restart on failure, per-participant heartbeat, idle-room GC
//   - Per-participant audio recording (Opus → decoded PCM16 mono WAV),
//     uploaded through pkg/stores, matching the SIP / xiaozhi / 1:1
//     WebRTC recordings so dashboards see one shape for all transports.
//   - Outbound webhooks (room.started, participant.joined/left,
//     recording.finished) with HMAC-SHA256 signature headers.
//
// What is intentionally out of scope (MVP ship cuts)
//
//   - Data channels, SVC, redundant audio encoding
//   - Persistent room state (Redis) — everything lives in memory;
//     horizontal scaling needs an external coordinator added later.
//   - TURN server provisioning — callers supply TURN URLs in the
//     engine config or environment; we only use them.
package sfu
