// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

const p2pRelayBufCap = 120

// P2PBroker relays WebSocket signaling between up to two peers per room (mesh WebRTC).
type P2PBroker struct {
	mu    sync.Mutex
	rooms map[string]*p2pRoster
}

type p2pRoster struct {
	mu    sync.Mutex
	peers map[string]*websocket.Conn
	buf   [][]byte
}

// NewP2PBroker creates an empty broker.
func NewP2PBroker() *P2PBroker {
	return &P2PBroker{rooms: make(map[string]*p2pRoster)}
}

// RoomPeerCount returns how many WebSocket peers are currently in the room (0–2).
func (b *P2PBroker) RoomPeerCount(roomID string) int {
	b.mu.Lock()
	r := b.rooms[roomID]
	b.mu.Unlock()
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.peers)
}

// RoomAllows returns nil if this peer_id may connect; otherwise ErrP2PRoomFull when two others are present.
func (b *P2PBroker) RoomAllows(roomID, peerID string) error {
	if roomID == "" || peerID == "" {
		return fmt.Errorf("sfu: empty room_id or peer_id")
	}
	b.mu.Lock()
	r := b.rooms[roomID]
	b.mu.Unlock()
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.peers) >= 2 {
		if _, exists := r.peers[peerID]; !exists {
			return ErrP2PRoomFull
		}
	}
	return nil
}

func (b *P2PBroker) ensureRoster(roomID string) *p2pRoster {
	b.mu.Lock()
	defer b.mu.Unlock()
	if r, ok := b.rooms[roomID]; ok {
		return r
	}
	r := &p2pRoster{peers: make(map[string]*websocket.Conn)}
	b.rooms[roomID] = r
	return r
}

func (b *P2PBroker) leaveRoom(roomID, peerID string) {
	b.mu.Lock()
	r, ok := b.rooms[roomID]
	b.mu.Unlock()
	if !ok || r == nil {
		return
	}
	r.mu.Lock()
	delete(r.peers, peerID)
	empty := len(r.peers) == 0
	r.mu.Unlock()
	if empty {
		b.mu.Lock()
		delete(b.rooms, roomID)
		b.mu.Unlock()
	}
}

// Handle registers the connection in roomID (max two distinct peer_id values) and relays
// text frames to the other peer. If the other peer is not connected yet, messages are queued
// (capped) until they connect.
func (b *P2PBroker) Handle(roomID, peerID string, conn *websocket.Conn, readLimit int64) error {
	if roomID == "" || peerID == "" {
		return fmt.Errorf("sfu: empty room_id or peer_id")
	}
	rost := b.ensureRoster(roomID)
	rost.mu.Lock()
	if _, exists := rost.peers[peerID]; !exists && len(rost.peers) >= 2 {
		rost.mu.Unlock()
		return ErrP2PRoomFull
	}
	if old := rost.peers[peerID]; old != nil {
		_ = old.Close()
	}
	rost.peers[peerID] = conn
	toFlush := append([][]byte(nil), rost.buf...)
	rost.buf = rost.buf[:0]
	rost.mu.Unlock()
	defer b.leaveRoom(roomID, peerID)

	if readLimit > 0 {
		conn.SetReadLimit(readLimit)
	}
	for _, m := range toFlush {
		if err := conn.WriteMessage(websocket.TextMessage, m); err != nil {
			return err
		}
	}

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		rost.mu.Lock()
		var forwarded bool
		for pid, c := range rost.peers {
			if pid == peerID {
				continue
			}
			if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
				rost.mu.Unlock()
				return err
			}
			forwarded = true
			break
		}
		if !forwarded {
			if len(rost.buf) >= p2pRelayBufCap {
				rost.buf = rost.buf[1:]
			}
			rost.buf = append(rost.buf, append([]byte(nil), data...))
		}
		rost.mu.Unlock()
	}
}
