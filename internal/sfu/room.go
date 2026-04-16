// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"strings"
	"sync"

	"github.com/pion/webrtc/v3"
)

const upstreamKeySep = "\x1e"

// Room is one SFU room: many peers, each publishing tracks fanned out to others.
type Room struct {
	id   string
	mu   sync.RWMutex
	peer map[string]*Peer
	// key = ownerPeerID + sep + trackID
	up map[string]*upstreamFanout
}

func newRoom(id string) *Room {
	return &Room{
		id:   id,
		peer: make(map[string]*Peer),
		up:   make(map[string]*upstreamFanout),
	}
}

func upstreamKey(ownerPeerID, trackID string) string {
	return ownerPeerID + upstreamKeySep + trackID
}

// tryRegisterPeer registers p if the room is under maxPeers (0 = unlimited). Returns false if full.
func (r *Room) tryRegisterPeer(p *Peer, maxPeers int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if maxPeers > 0 && len(r.peer) >= maxPeers {
		return false
	}
	r.peer[p.id] = p
	return true
}

// PeerCount returns the number of connected peers.
func (r *Room) PeerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.peer)
}

// removePeer removes a peer, closes its published upstream fan-outs, and returns remaining peers.
func (r *Room) removePeer(id string) int {
	r.mu.Lock()
	var owned []*upstreamFanout
	var keys []string
	for k, uf := range r.up {
		owner, _, ok := strings.Cut(k, upstreamKeySep)
		if ok && owner == id {
			keys = append(keys, k)
			owned = append(owned, uf)
		}
	}
	for _, k := range keys {
		delete(r.up, k)
	}
	delete(r.peer, id)
	n := len(r.peer)
	r.mu.Unlock()
	for _, uf := range owned {
		uf.Close()
	}
	return n
}

func (r *Room) eachPeer(fn func(*Peer)) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.peer {
		fn(p)
	}
}

// PeerByID returns a connected peer or nil.
func (r *Room) PeerByID(id string) *Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.peer[id]
}

func (r *Room) publisherTrackPublished(pub *Peer, track *webrtc.TrackRemote) {
	key := upstreamKey(pub.id, track.ID())
	r.mu.Lock()
	if _, exists := r.up[key]; exists {
		r.mu.Unlock()
		return
	}
	uf := &upstreamFanout{track: track}
	r.up[key] = uf
	r.mu.Unlock()

	r.eachPeer(func(sub *Peer) {
		if sub.id == pub.id {
			return
		}
		sub.subscribeUpstream(uf, pub.id, track)
	})
}

func (r *Room) attachAllUpstreamsToSubscriber(sub *Peer) {
	r.mu.RLock()
	type item struct {
		owner string
		uf    *upstreamFanout
	}
	var items []item
	for k, uf := range r.up {
		parts := strings.SplitN(k, upstreamKeySep, 2)
		if len(parts) != 2 {
			continue
		}
		owner := parts[0]
		if owner == sub.id {
			continue
		}
		items = append(items, item{owner: owner, uf: uf})
	}
	r.mu.RUnlock()

	for _, it := range items {
		sub.subscribeUpstream(it.uf, it.owner, it.uf.track)
	}
}
