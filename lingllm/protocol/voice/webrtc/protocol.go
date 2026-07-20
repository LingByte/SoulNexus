// Package webrtc terminates 1v1 WebRTC AI voice calls over HTTP SDP signaling.
//
// Browser ── POST /offer (SDP) ──► Server ── dialog WS ──► Dialog App
//
//	◄── SRTP/Opus (TTS) ────       ◄── tts.speak ───
//	── SRTP/Opus (mic) ────►       ── asr.* ────────►
package webrtc

import "encoding/json"

// SDPMessage is the wire shape of offer/answer payloads.
type SDPMessage struct {
	SDP    string `json:"sdp"`
	Type   string `json:"type"`
	CallID string `json:"call_id,omitempty"`
}

// OfferRequest is the JSON body for POST /webrtc/v1/offer.
type OfferRequest struct {
	SDP       string          `json:"sdp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	ApiKey    string          `json:"apiKey,omitempty"`
	ApiSecret string          `json:"apiSecret,omitempty"`
	AgentId   string          `json:"agentId,omitempty"`
}
