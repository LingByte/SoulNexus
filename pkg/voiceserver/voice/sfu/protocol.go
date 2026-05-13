// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import "encoding/json"

// Signaling protocol — tiny typed JSON message set exchanged on the
// /sfu/ws WebSocket. Keep it additive: old clients must survive new
// server messages they don't recognise, and vice versa. That's enforced
// by the `Type` field being the only required field at the envelope
// level; everything else lives inside Data which is typed per message.
//
// Directions
//
//	c→s  client to server
//	s→c  server to client
//	both either direction
//
// Message types
//
//	join           c→s   client joins the room (first message after connect)
//	joined         s→c   server ack, carries peer list + self identity
//	offer          both  SDP offer (client publish or server renegotiation)
//	answer         both  SDP answer
//	iceCandidate   both  trickle ICE candidate
//	participantJoined s→c  someone else joined
//	participantLeft   s→c  someone left
//	trackPublished    s→c  a remote track is now available to subscribe
//	trackUnpublished  s→c  a remote track disappeared
//	setMute        c→s   mute/unmute a local track (server stops forwarding)
//	iceRestart     s→c   server forces an ICE restart (renegotiation)
//	leave          c→s   graceful disconnect
//	error          s→c   transport-level error with code+message
//	ping / pong    both  keepalive (server initiates, 2 missed → kick)

type MessageType string

const (
	MsgJoin              MessageType = "join"
	MsgJoined            MessageType = "joined"
	MsgOffer             MessageType = "offer"
	MsgAnswer            MessageType = "answer"
	MsgICECandidate      MessageType = "iceCandidate"
	MsgParticipantJoined MessageType = "participantJoined"
	MsgParticipantLeft   MessageType = "participantLeft"
	MsgTrackPublished    MessageType = "trackPublished"
	MsgTrackUnpublished  MessageType = "trackUnpublished"
	MsgSetMute           MessageType = "setMute"
	MsgICERestart        MessageType = "iceRestart"
	MsgLeave             MessageType = "leave"
	MsgError             MessageType = "error"
	MsgPing              MessageType = "ping"
	MsgPong              MessageType = "pong"
)

// Envelope wraps every message. Data is opaque to the transport and
// decoded per-Type by the signaling handler.
type Envelope struct {
	Type MessageType     `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
	// RequestID is optional; when the client sets it on a message, the
	// server echoes it on the reply so client-side callbacks can be
	// correlated without tracking SDP payloads.
	RequestID string `json:"requestId,omitempty"`
}

// JoinData is the payload of MsgJoin. The access token is the sole
// source of truth for room / identity / permissions — Room and Name in
// this struct are purely advisory (used for logging on bad tokens).
type JoinData struct {
	Token string `json:"token"`
	Room  string `json:"room,omitempty"`
	Name  string `json:"name,omitempty"`
	// SDKVersion is optional client identification for server-side
	// compatibility heuristics.
	SDKVersion string `json:"sdkVersion,omitempty"`
}

// JoinedData is the server's reply to a successful MsgJoin. It tells the
// client who it is (server-authoritative, trimmed from the token) and
// which peers are already in the room. ICEServers mirrors the SFU
// config so the browser doesn't need to be re-configured per
// deployment.
type JoinedData struct {
	Room         string               `json:"room"`
	ParticipantID string              `json:"participantId"`
	Identity     string               `json:"identity"`
	Participants []ParticipantInfo    `json:"participants"`
	ICEServers   []ICEServerForClient `json:"iceServers"`
	// ServerTime is a UTC millisecond timestamp for clients that want to
	// measure clock skew.
	ServerTime int64 `json:"serverTime"`
}

// ParticipantInfo summarises one peer in the room from another peer's
// point of view. Excludes PeerConnection internals.
type ParticipantInfo struct {
	ParticipantID string       `json:"participantId"`
	Identity      string       `json:"identity"`
	Name          string       `json:"name,omitempty"`
	Tracks        []TrackInfo  `json:"tracks"`
	Metadata      string       `json:"metadata,omitempty"`
	Permissions   *Permissions `json:"permissions,omitempty"`
	JoinedAt      int64        `json:"joinedAt"`
}

// TrackInfo describes one media track published by a participant. The
// SFU emits one per unique track-id per source peer; subscribers receive
// trackPublished on join and on every subsequent new publication.
type TrackInfo struct {
	TrackID string `json:"trackId"`
	Kind    string `json:"kind"` // "audio" | "video"
	Source  string `json:"source,omitempty"`
	Muted   bool   `json:"muted,omitempty"`
	Codec   string `json:"codec,omitempty"`
}

// ICEServerForClient is a JSON-friendly mirror of pion's ICEServer for
// safe return to browser clients. We flatten the "URLs" field (JSON
// browsers expect a single string "urls" OR an array — we always emit
// an array for consistency).
type ICEServerForClient struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// SDPData carries an SDP offer or answer.
type SDPData struct {
	SDP string `json:"sdp"`
	// TargetParticipantID is used when the server sends an offer to a
	// subscriber as part of renegotiation after a new publication; it
	// identifies the "source" participant whose tracks triggered the
	// renegotiation. Clients can safely ignore it.
	TargetParticipantID string `json:"targetParticipantId,omitempty"`
}

// ICECandidateData carries one trickled candidate.
type ICECandidateData struct {
	Candidate     string `json:"candidate"`
	SDPMid        string `json:"sdpMid,omitempty"`
	SDPMLineIndex *uint16 `json:"sdpMLineIndex,omitempty"`
}

// ParticipantJoinedData / ParticipantLeftData / TrackPublishedData
// notify every other peer in the room about membership changes. These
// are derivable from a combination of other messages but having them as
// first-class events keeps clients simple.
type ParticipantJoinedData struct {
	Participant ParticipantInfo `json:"participant"`
}

type ParticipantLeftData struct {
	ParticipantID string `json:"participantId"`
	Reason        string `json:"reason,omitempty"`
}

type TrackPublishedData struct {
	ParticipantID string    `json:"participantId"`
	Track         TrackInfo `json:"track"`
}

type TrackUnpublishedData struct {
	ParticipantID string `json:"participantId"`
	TrackID       string `json:"trackId"`
}

// SetMuteData is client→server: "please stop forwarding my track with
// this trackId". The server enforces by dropping packets server-side;
// the client is still expected to stop sending (saves upstream
// bandwidth).
type SetMuteData struct {
	TrackID string `json:"trackId"`
	Muted   bool   `json:"muted"`
}

// ErrorData is the payload of MsgError. Code is a short machine-friendly
// identifier (e.g. "room_full", "bad_token"); Message is human-readable.
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
