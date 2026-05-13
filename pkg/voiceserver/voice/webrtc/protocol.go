// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package webrtc terminates 1v1 WebRTC AI voice calls. The browser (or any
// WebRTC client) sends an SDP offer over HTTP; VoiceServer answers with a
// fully-negotiated audio session whose media plane runs on top of pion:
// ICE for NAT traversal, DTLS-SRTP for transport encryption, Opus with
// inband FEC for packet-loss concealment, NACK + TWCC interceptors for
// retransmission and bandwidth estimation, and a jitter-buffered sample
// builder for de-pickling out-of-order RTP into clean PCM.
//
// The negotiated audio is bridged into the same dialog plane the SIP and
// xiaozhi paths use (pkg/voice/gateway). One ASR/TTS pipeline serves all
// transports; the dialog application is unchanged.
//
// Architecture:
//
//	Browser ──┐                              ┌── ws://dialog/ws/call ──► dialog app
//	          │  HTTP /webrtc/v1/offer (SDP) │
//	          ├─────────────────────────────►│
//	          │  HTTP body = SDP answer      │
//	          │◄─────────────────────────────│
//	          │  ICE → DTLS → SRTP/Opus      │  pkg/voice/asr  → text events
//	          ├─────────────────────────────►│  pkg/voice/tts  ← tts.speak commands
//	          │  SRTP/Opus (TTS reply)       │
//	          │◄─────────────────────────────│
//	          ▼                              ▼
//	     PeerConnection                 voice.Attached
//
// Wire formats on the signaling endpoint:
//
//	POST /webrtc/v1/offer
//	  Body  : application/json {"sdp":"...","type":"offer","payload":{"apiKey":"...","apiSecret":"...","agentId":"73",...}}
//	        (recommended: pass dialog auth/config in payload — voiceserver sets URL query "payload" for the dialog plane to parse.
//	        Legacy: flat apiKey / apiSecret / agentId on the root object still merged when payload is absent.)
//	  Reply : application/json {"sdp":"v=0...","type":"answer","call_id":"wrtc-..."}
//
// The signaling is HTTP POST so any fetch()-capable client works without
// extra runtime; trickle ICE is unnecessary because pion gathers all
// candidates before we return the answer. For browsers that strictly need
// trickle ICE, add an ICE candidate POST endpoint as a follow-up.
package webrtc

import "encoding/json"

// SDPMessage is the wire shape of both offer and answer payloads. The
// `type` field follows the WebRTC spec ("offer" / "answer" / "pranswer" /
// "rollback") so it can be fed into the browser's
// RTCPeerConnection.setRemoteDescription unchanged.
type SDPMessage struct {
	SDP    string `json:"sdp"`
	Type   string `json:"type"`
	CallID string `json:"call_id,omitempty"` // server-assigned on the answer
}

// OfferRequest is the JSON body for POST /webrtc/v1/offer (browser → voiceserver).
// Prefer Payload: arbitrary JSON merged as a single URL query key "payload"
// for the dialog WebSocket (dialog plane parses it). Flat ApiKey / ApiSecret /
// AgentId are still supported when Payload is empty.
type OfferRequest struct {
	SDP       string          `json:"sdp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	ApiKey    string          `json:"apiKey,omitempty"`
	ApiSecret string          `json:"apiSecret,omitempty"`
	AgentId   string          `json:"agentId,omitempty"`
}
