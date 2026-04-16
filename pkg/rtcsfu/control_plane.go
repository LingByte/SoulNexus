// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import (
	"fmt"
	"sync"
	"time"
)

type replicaEntry struct {
	node     SFUNode
	lastSeen time.Time
}

// ControlPlane holds static SFU nodes (from config / reload), optional replica registrations,
// sticky room assignments, and routing.
type ControlPlane struct {
	mu                sync.Mutex
	static            []SFUNode
	replicas          map[NodeID]replicaEntry
	reg               *RoomRegistry
	replicaStaleAfter time.Duration // 0 = do not mark replicas unhealthy from last_seen age
}

// NewControlPlane creates a control plane with an initial static node list.
// replicaStaleSeconds: if >0, replicas with no register/touch for longer than this are treated as unhealthy for routing and listed as stale.
func NewControlPlane(static []SFUNode, replicaStaleSeconds int) *ControlPlane {
	var stale time.Duration
	if replicaStaleSeconds > 0 {
		stale = time.Duration(replicaStaleSeconds) * time.Second
	}
	return &ControlPlane{
		static:            cloneNodes(static),
		replicas:          make(map[NodeID]replicaEntry),
		reg:               NewRoomRegistry(),
		replicaStaleAfter: stale,
	}
}

func (p *ControlPlane) replicaIsStale(e replicaEntry, now time.Time) bool {
	if p.replicaStaleAfter <= 0 {
		return false
	}
	return now.Sub(e.lastSeen) > p.replicaStaleAfter
}

func (p *ControlPlane) replicaNodeForRouting(e replicaEntry, now time.Time) SFUNode {
	n := e.node
	if p.replicaIsStale(e, now) {
		n.Healthy = false
	}
	return n
}

// Reload replaces the static node snapshot used for new picks and validity checks.
// Dynamically registered replicas are preserved.
func (p *ControlPlane) Reload(static []SFUNode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.static = cloneNodes(static)
}

// UpsertReplica registers or updates a secondary SFU node (same shape as RTCSFU_NODES items).
// If the id collides with a static node id, the static entry wins for routing; the replica is ignored.
// Updates last_seen to the current time.
func (p *ControlPlane) UpsertReplica(n SFUNode) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.replicas == nil {
		p.replicas = make(map[NodeID]replicaEntry)
	}
	p.replicas[n.ID] = replicaEntry{node: n, lastSeen: time.Now().UTC()}
}

// TouchReplica updates last_seen for an already-registered replica id. Returns false if unknown.
func (p *ControlPlane) TouchReplica(id NodeID) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	e, ok := p.replicas[id]
	if !ok {
		return false
	}
	e.lastSeen = time.Now().UTC()
	p.replicas[id] = e
	return true
}

// RemoveReplica drops a dynamically registered node. Returns false if it was not present.
func (p *ControlPlane) RemoveReplica(id NodeID) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.replicas == nil {
		return false
	}
	if _, ok := p.replicas[id]; !ok {
		return false
	}
	delete(p.replicas, id)
	return true
}

// StaticSnapshot returns a copy of the static node list (config / reload).
func (p *ControlPlane) StaticSnapshot() []SFUNode {
	p.mu.Lock()
	defer p.mu.Unlock()
	return cloneNodes(p.static)
}

// ReplicaSnapshot returns registered replica nodes (excluding ids shadowed by static).
func (p *ControlPlane) ReplicaSnapshot() []SFUNode {
	p.mu.Lock()
	defer p.mu.Unlock()
	seen := make(map[NodeID]struct{}, len(p.static))
	for _, n := range p.static {
		seen[n.ID] = struct{}{}
	}
	out := make([]SFUNode, 0, len(p.replicas))
	for id, e := range p.replicas {
		if _, shadow := seen[id]; shadow {
			continue
		}
		out = append(out, e.node)
	}
	return out
}

// ReplicaRows returns replica nodes with last_seen (UTC unix), excluding ids shadowed by static.
func (p *ControlPlane) ReplicaRows() []ReplicaRow {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	seen := make(map[NodeID]struct{}, len(p.static))
	for _, n := range p.static {
		seen[n.ID] = struct{}{}
	}
	out := make([]ReplicaRow, 0, len(p.replicas))
	for id, e := range p.replicas {
		if _, shadow := seen[id]; shadow {
			continue
		}
		stale := p.replicaIsStale(e, now)
		n := e.node
		if stale {
			n.Healthy = false
		}
		out = append(out, ReplicaRow{
			Node:         n,
			LastSeenUnix: e.lastSeen.Unix(),
			Stale:        stale,
		})
		_ = id
	}
	return out
}

// MergedSnapshot returns static nodes plus replicas (replica ids that duplicate static ids are skipped).
func (p *ControlPlane) MergedSnapshot() []SFUNode {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.mergedLocked(time.Now())
}

func (p *ControlPlane) mergedLocked(now time.Time) []SFUNode {
	out := cloneNodes(p.static)
	seen := make(map[NodeID]struct{}, len(out))
	for _, n := range out {
		seen[n.ID] = struct{}{}
	}
	for id, e := range p.replicas {
		if _, ok := seen[id]; ok {
			continue
		}
		out = append(out, p.replicaNodeForRouting(e, now))
		seen[id] = struct{}{}
	}
	return out
}

// NodeCount returns how many nodes are in the merged routing snapshot.
func (p *ControlPlane) NodeCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.mergedLocked(time.Now()))
}

// Join returns a sticky room assignment, re-routing if the previous node vanished or became ineligible.
func (p *ControlPlane) Join(room RoomID, region RegionID) (RoomAssignment, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	nodes := p.mergedLocked(time.Now())
	if len(nodes) == 0 {
		return RoomAssignment{}, fmt.Errorf("rtcsfu: no SFU nodes configured")
	}
	rt := NewRoomRouter(nodes)
	if a, ok := p.reg.Get(room); ok {
		if n, found := findNode(nodes, a.Node.ID); found && n.Eligible() {
			return RoomAssignment{
				RoomID:   room,
				Node:     n,
				Assigned: a.Assigned,
			}, nil
		}
		p.reg.Remove(room)
	}
	return p.reg.GetOrAssign(room, region, rt)
}

func cloneNodes(nodes []SFUNode) []SFUNode {
	if len(nodes) == 0 {
		return nil
	}
	out := make([]SFUNode, len(nodes))
	copy(out, nodes)
	return out
}

func findNode(nodes []SFUNode, id NodeID) (SFUNode, bool) {
	for _, n := range nodes {
		if n.ID == id {
			return n, true
		}
	}
	return SFUNode{}, false
}
