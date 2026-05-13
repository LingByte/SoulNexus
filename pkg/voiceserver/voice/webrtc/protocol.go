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
//	  Body  : application/json {"sdp":"v=0...","type":"offer"}
//	  Reply : application/json {"sdp":"v=0...","type":"answer","call_id":"wrtc-..."}
//
// The signaling is HTTP POST so any fetch()-capable client works without
// extra runtime; trickle ICE is unnecessary because pion gathers all
// candidates before we return the answer. For browsers that strictly need
// trickle ICE, add an ICE candidate POST endpoint as a follow-up.
package webrtc

// SDPMessage is the wire shape of both offer and answer payloads. The
// `type` field follows the WebRTC spec ("offer" / "answer" / "pranswer" /
// "rollback") so it can be fed into the browser's
// RTCPeerConnection.setRemoteDescription unchanged.
type SDPMessage struct {
	SDP    string `json:"sdp"`
	Type   string `json:"type"`
	CallID string `json:"call_id,omitempty"` // server-assigned on the answer
}
