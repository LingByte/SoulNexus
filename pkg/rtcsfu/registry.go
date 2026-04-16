// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import (
	"fmt"
	"sync"
	"time"
)

// RoomRegistry keeps sticky room→SFU assignments and supports explicit migration.
type RoomRegistry struct {
	mu    sync.RWMutex
	rooms map[RoomID]RoomAssignment
	// nodeRooms inverted index for draining / ops
	nodeRooms map[NodeID]map[RoomID]struct{}
}

// NewRoomRegistry creates an empty registry.
func NewRoomRegistry() *RoomRegistry {
	return &RoomRegistry{
		rooms:     make(map[RoomID]RoomAssignment),
		nodeRooms: make(map[NodeID]map[RoomID]struct{}),
	}
}

// Get returns the current assignment if any.
func (reg *RoomRegistry) Get(room RoomID) (RoomAssignment, bool) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	a, ok := reg.rooms[room]
	return a, ok
}

// GetOrAssign returns the existing assignment or picks via router and stores it.
func (reg *RoomRegistry) GetOrAssign(room RoomID, clientRegion RegionID, rt *RoomRouter) (RoomAssignment, error) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if a, ok := reg.rooms[room]; ok {
		return a, nil
	}
	node, err := rt.Pick(RoomRouteRequest{RoomID: room, ClientRegion: clientRegion})
	if err != nil {
		return RoomAssignment{}, err
	}
	a := RoomAssignment{RoomID: room, Node: node, Assigned: time.Now().UTC()}
	reg.rooms[room] = a
	m := reg.nodeRooms[node.ID]
	if m == nil {
		m = make(map[RoomID]struct{})
		reg.nodeRooms[node.ID] = m
	}
	m[room] = struct{}{}
	return a, nil
}

// Migrate moves a room to a new node (e.g. draining). The target must match want.ID.
func (reg *RoomRegistry) Migrate(room RoomID, want SFUNode) error {
	if !want.Eligible() {
		return fmt.Errorf("rtcsfu: target node %q not eligible", want.ID)
	}
	reg.mu.Lock()
	defer reg.mu.Unlock()
	old, ok := reg.rooms[room]
	if ok {
		reg.dropFromNode(old.Node.ID, room)
	}
	a := RoomAssignment{RoomID: room, Node: want, Assigned: time.Now().UTC()}
	reg.rooms[room] = a
	m := reg.nodeRooms[want.ID]
	if m == nil {
		m = make(map[RoomID]struct{})
		reg.nodeRooms[want.ID] = m
	}
	m[room] = struct{}{}
	return nil
}

// Remove deletes a room mapping (e.g. room destroyed).
func (reg *RoomRegistry) Remove(room RoomID) {
	reg.mu.Lock()
	defer reg.mu.Unlock()
	if old, ok := reg.rooms[room]; ok {
		reg.dropFromNode(old.Node.ID, room)
		delete(reg.rooms, room)
	}
}

// RoomsOnNode lists rooms currently assigned to the node.
func (reg *RoomRegistry) RoomsOnNode(id NodeID) []RoomID {
	reg.mu.RLock()
	defer reg.mu.RUnlock()
	m := reg.nodeRooms[id]
	if len(m) == 0 {
		return nil
	}
	out := make([]RoomID, 0, len(m))
	for r := range m {
		out = append(out, r)
	}
	return out
}

func (reg *RoomRegistry) dropFromNode(id NodeID, room RoomID) {
	m := reg.nodeRooms[id]
	if m == nil {
		return
	}
	delete(m, room)
	if len(m) == 0 {
		delete(reg.nodeRooms, id)
	}
}
