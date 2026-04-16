// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import "time"

// RoomID identifies a realtime session (conference room).
type RoomID string

// UserID identifies a participant.
type UserID string

// RegionID is a coarse location label (e.g. "cn-east", "us-west") used for affinity.
type RegionID string

// NodeID uniquely identifies one SFU instance.
type NodeID string

// SFUNode is schedulable metadata for one SFU process.
type SFUNode struct {
	ID        NodeID   `json:"id"`
	Region    RegionID `json:"region"`
	SignalURL string   `json:"signal_url"`
	MediaURL  string   `json:"media_url,omitempty"`
	Healthy   bool     `json:"healthy"`
	Draining  bool     `json:"draining,omitempty"`
}

// Eligible reports whether the node may receive new room assignments.
func (n SFUNode) Eligible() bool {
	return n.Healthy && !n.Draining
}

// ReplicaRow is one dynamically registered replica plus last successful register/touch time.
type ReplicaRow struct {
	Node         SFUNode `json:"node"` // Healthy reflects routing view (false when Stale).
	LastSeenUnix int64   `json:"last_seen_unix"`
	Stale        bool    `json:"stale"` // true when last_seen exceeds primary RTCSFU_REPLICA_STALE_SECONDS
}

// RoomRouteRequest carries join-time hints for routing.
type RoomRouteRequest struct {
	RoomID RoomID
	// ClientRegion is the client's preferred region; may be empty for global-only routing.
	ClientRegion RegionID
}

// RoomAssignment binds a room to an SFU for the lifetime of the mapping until migrated.
type RoomAssignment struct {
	RoomID   RoomID
	Node     SFUNode
	Assigned time.Time
}
