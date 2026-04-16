// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

// Options configures the embedded SFU engine (production limits and WebSocket behavior).
type Options struct {
	ICEServers      []webrtc.ICEServer
	MaxRooms        int // 0 = unlimited
	MaxPeersPerRoom int // 0 = unlimited
	WSReadTimeout   time.Duration
	WSPingInterval  time.Duration
}

// Engine holds SFU rooms for this process.
type Engine struct {
	opt Options
	mu  sync.Mutex
	ice []webrtc.ICEServer
	// room id -> room
	room map[string]*Room
}

// NewEngine creates an SFU engine. ICEServers is copied.
func NewEngine(opt Options) *Engine {
	ice := append([]webrtc.ICEServer(nil), opt.ICEServers...)
	return &Engine{
		opt:  opt,
		ice:  ice,
		room: make(map[string]*Room),
	}
}

func (e *Engine) roomOf(id string) (*Room, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	r, ok := e.room[id]
	if !ok {
		if e.opt.MaxRooms > 0 && len(e.room) >= e.opt.MaxRooms {
			return nil, ErrMaxRooms
		}
		r = newRoom(id)
		e.room[id] = r
		metricRoomsCurrent.Inc()
	}
	return r, nil
}

func (e *Engine) maybeTrimEmptyRoom(roomID string) {
	e.mu.Lock()
	if r, ok := e.room[roomID]; ok && r.PeerCount() == 0 {
		delete(e.room, roomID)
		metricRoomsCurrent.Dec()
	}
	e.mu.Unlock()
}

func (e *Engine) deleteRoomIfEmpty(roomID string, r *Room) {
	e.mu.Lock()
	if cur, ok := e.room[roomID]; ok && cur == r && cur.PeerCount() == 0 {
		delete(e.room, roomID)
		metricRoomsCurrent.Dec()
	}
	e.mu.Unlock()
}

// PeersInRoom returns how many SFU signaling peers are connected in roomID (0 if room absent).
func (e *Engine) PeersInRoom(roomID string) int {
	e.mu.Lock()
	r, ok := e.room[roomID]
	e.mu.Unlock()
	if !ok || r == nil {
		return 0
	}
	return r.PeerCount()
}

// Stats returns approximate live counts (for status API).
func (e *Engine) Stats() (rooms int, peers int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	rooms = len(e.room)
	for _, r := range e.room {
		peers += r.PeerCount()
	}
	return rooms, peers
}

// HandleConn runs the signaling loop until the WebSocket closes.
func (e *Engine) HandleConn(conn *websocket.Conn, roomID, peerID string) {
	r, err := e.roomOf(roomID)
	if err != nil {
		if errors.Is(err, ErrMaxRooms) {
			RecordPeerRejected("max_rooms")
		}
		_ = conn.Close()
		return
	}
	p, err := newPeer(r, peerID, conn, e.ice, e.opt.WSReadTimeout, e.opt.WSPingInterval)
	if err != nil {
		e.maybeTrimEmptyRoom(roomID)
		_ = conn.Close()
		return
	}
	if !r.tryRegisterPeer(p, e.opt.MaxPeersPerRoom) {
		RecordPeerRejected("room_full")
		p.Close()
		e.maybeTrimEmptyRoom(roomID)
		_ = conn.Close()
		return
	}
	metricPeerJoins.Inc()
	metricPeersCurrent.Inc()
	defer metricPeersCurrent.Dec()
	defer func() {
		p.Close()
		left := r.removePeer(peerID)
		if left == 0 {
			e.deleteRoomIfEmpty(roomID, r)
		}
	}()
	p.run()
}
