// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	pionwebrtc "github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

// Manager owns all rooms in the process. The embedding HTTP handler
// goes through the manager for every join so rooms are lazily
// created and garbage collected. Safe for concurrent use.
type Manager struct {
	cfg      *Config
	logger   *zap.Logger
	api      *pionwebrtc.API
	webhook  *webhookEmitter
	upgrader *websocket.Upgrader

	mu     sync.Mutex
	rooms  map[string]*Room
	closed atomic.Bool
}

// NewManager builds the pion API once and stashes it for reuse across
// rooms. Fails if codec/interceptor registration fails.
func NewManager(cfg *Config, logger *zap.Logger) (*Manager, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	cfg.Normalise()
	if logger == nil {
		logger = zap.NewNop()
	}
	api, err := buildAPI(cfg)
	if err != nil {
		return nil, err
	}
	return &Manager{
		cfg:      cfg,
		logger:   logger,
		api:      api,
		webhook:  newWebhookEmitter(cfg, logger),
		upgrader: newWSUpgrader(cfg.AllowedOrigins),
		rooms:    make(map[string]*Room),
	}, nil
}

// Close tears down the manager. All rooms are dropped, every active
// participant gets disconnected, and the webhook goroutine is told to
// exit so no goroutines leak. Idempotent. Subsequent ServeWS calls
// respond with HTTP 503.
func (m *Manager) Close() {
	if !m.closed.CompareAndSwap(false, true) {
		return
	}
	m.mu.Lock()
	rooms := make([]*Room, 0, len(m.rooms))
	for _, r := range m.rooms {
		rooms = append(rooms, r)
	}
	m.rooms = make(map[string]*Room)
	m.mu.Unlock()
	for _, r := range rooms {
		for _, p := range r.Participants() {
			p.Close("manager_shutdown")
		}
	}
	if m.webhook != nil {
		m.webhook.shutdown()
	}
}

// Config exposes the effective (post-normalise) configuration. Used by
// the HTTP handler to read HeartbeatInterval etc. without a separate
// pointer indirection for every callsite.
func (m *Manager) Config() *Config { return m.cfg }

// getOrCreateRoom returns an existing Room by name or creates a new
// one, subject to Config.MaxRooms. Returns ErrTooManyRooms if the cap
// would be exceeded by creating a new room. Existing-room lookups
// always succeed regardless of the cap so peers can keep joining a
// room that's already at the limit.
func (m *Manager) getOrCreateRoom(name string) (*Room, error) {
	if m.closed.Load() {
		return nil, ErrManagerClosed
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.rooms[name]; ok {
		return r, nil
	}
	// MaxRooms < 0 == unlimited; ==0 should have been normalised to
	// 1024 by Config.Normalise so we only hit the cap branch for >0.
	if m.cfg.MaxRooms > 0 && len(m.rooms) >= m.cfg.MaxRooms {
		return nil, ErrTooManyRooms
	}
	r := newRoom(name, m)
	m.rooms[name] = r
	m.webhook.emit(Event{Type: EventRoomStarted, Room: name, Timestamp: time.Now().UnixMilli()})
	return r, nil
}

// dropRoom removes a room from the manager map. Called by Room.maybeGC
// once the idle timer fires with no participants.
func (m *Manager) dropRoom(name string) {
	m.mu.Lock()
	delete(m.rooms, name)
	m.mu.Unlock()
	m.webhook.emit(Event{Type: EventRoomEnded, Room: name, Timestamp: time.Now().UnixMilli()})
}

// Room is N participants that can see/hear each other's published
// tracks. One Room per distinct `room` claim in access tokens.
type Room struct {
	name    string
	manager *Manager
	cfg     *Config
	logger  *zap.Logger
	api     *pionwebrtc.API

	mu           sync.RWMutex
	participants map[string]*Participant
	// identityIndex prevents two simultaneous sessions from the same
	// Identity from joining; the second join replaces the first (older
	// session gets kicked with "duplicate_identity") so mobile →
	// desktop handoffs work without a manual leave.
	identityIndex map[string]string // identity → participantID

	idleTimer *time.Timer
}

func newRoom(name string, m *Manager) *Room {
	return &Room{
		name:          name,
		manager:       m,
		cfg:           m.cfg,
		logger:        m.logger.With(zap.String("room", name)),
		api:           m.api,
		participants:  make(map[string]*Participant),
		identityIndex: make(map[string]string),
	}
}

// Name is the room identifier used in token claims and webhook events.
func (r *Room) Name() string { return r.name }

// Participants returns a snapshot list. Safe to iterate without
// holding a lock.
func (r *Room) Participants() []*Participant {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Participant, 0, len(r.participants))
	for _, p := range r.participants {
		out = append(out, p)
	}
	return out
}

// ParticipantByID returns the participant matching id, or nil.
func (r *Room) ParticipantByID(id string) *Participant {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.participants[id]
}

// AddParticipant admits a new participant. Enforces MaxParticipants
// and the duplicate-identity rule. Returns (participant, nil) on
// success.
func (r *Room) AddParticipant(claims AccessTokenClaims, ws *wsConn) (*Participant, error) {
	r.mu.Lock()
	if len(r.participants) >= r.cfg.MaxParticipantsPerRoom {
		r.mu.Unlock()
		return nil, ErrRoomFull
	}
	// Kick an earlier session with the same identity (older wins the
	// race on reconnect storms by virtue of being already in the map;
	// newer wins here because the reconnect is definitionally the
	// freshest intent).
	var evict *Participant
	if prevID, ok := r.identityIndex[claims.Identity]; ok {
		evict = r.participants[prevID]
	}
	id := newParticipantID()
	r.mu.Unlock()

	if evict != nil {
		evict.Close("duplicate_identity")
	}

	p, err := newParticipant(id, claims, ws, r)
	if err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.participants[id] = p
	r.identityIndex[claims.Identity] = id
	if r.idleTimer != nil {
		r.idleTimer.Stop()
		r.idleTimer = nil
	}
	// Snapshot peer list BEFORE releasing lock so the joined reply and
	// the participantJoined broadcast race-free.
	peers := make([]*Participant, 0, len(r.participants)-1)
	for pid, other := range r.participants {
		if pid != id {
			peers = append(peers, other)
		}
	}
	r.mu.Unlock()

	// Send joined reply with self identity + existing peers.
	peerInfos := make([]ParticipantInfo, 0, len(peers))
	for _, pr := range peers {
		peerInfos = append(peerInfos, pr.Info())
	}
	_ = ws.sendJSON(Envelope{Type: MsgJoined}, JoinedData{
		Room:          r.name,
		ParticipantID: id,
		Identity:      claims.Identity,
		Participants:  peerInfos,
		ICEServers:    iceServersForClient(r.cfg.ICEServers),
		ServerTime:    time.Now().UnixMilli(),
	})

	// Auto-subscribe the new peer to everyone else's published tracks
	// so there's no need for the client to manually request each one.
	// This is the LiveKit-style "subscribe to everything by default"
	// pattern; a future extension can add selective subscription via
	// a permission flag.
	//
	// deferSubscription (vs. SubscribeTo) is critical here: the new
	// peer has not yet sent its first offer, so AddTrack'ing now would
	// cause the upcoming answer to contain m-lines the client never
	// described. We queue the subscriptions; HandleOffer flushes them
	// once the initial handshake completes.
	for _, other := range peers {
		other.mu.RLock()
		pubs := make([]*PublishedTrack, 0, len(other.published))
		for _, pub := range other.published {
			pubs = append(pubs, pub)
		}
		other.mu.RUnlock()
		for _, pub := range pubs {
			p.deferSubscription(other.id, pub)
		}
	}

	// Notify existing peers that someone joined.
	info := p.Info()
	r.broadcast(id, Envelope{Type: MsgParticipantJoined}, ParticipantJoinedData{Participant: info})

	r.manager.webhook.emit(Event{
		Type:          EventParticipantJoined,
		Room:          r.name,
		ParticipantID: id,
		Identity:      claims.Identity,
		Timestamp:     time.Now().UnixMilli(),
	})

	return p, nil
}

// removeParticipant is called by Participant.Close. Cleans up
// subscriptions other peers had into this participant's tracks and
// broadcasts MsgParticipantLeft.
func (r *Room) removeParticipant(p *Participant, reason string) {
	r.mu.Lock()
	if _, ok := r.participants[p.id]; !ok {
		r.mu.Unlock()
		return
	}
	delete(r.participants, p.id)
	if r.identityIndex[p.identity] == p.id {
		delete(r.identityIndex, p.identity)
	}
	empty := len(r.participants) == 0
	r.mu.Unlock()

	// For every track the departing peer had published, tell all
	// subscribers the track is gone. They'll RemoveTrack + renegotiate.
	p.mu.RLock()
	trackIDs := make([]string, 0, len(p.published))
	for id := range p.published {
		trackIDs = append(trackIDs, id)
	}
	p.mu.RUnlock()
	for _, tid := range trackIDs {
		r.announceUnpublish(p.id, tid)
		for _, other := range r.Participants() {
			other.UnsubscribeFrom(p.id, tid)
		}
	}

	r.broadcast(p.id, Envelope{Type: MsgParticipantLeft}, ParticipantLeftData{
		ParticipantID: p.id,
		Reason:        reason,
	})

	r.manager.webhook.emit(Event{
		Type:          EventParticipantLeft,
		Room:          r.name,
		ParticipantID: p.id,
		Identity:      p.identity,
		Reason:        reason,
		Timestamp:     time.Now().UnixMilli(),
	})

	if empty {
		r.armIdleGC()
	}
}

// announcePublish notifies every other participant that `p` just
// published a new track AND auto-subscribes each of them to it.
func (r *Room) announcePublish(p *Participant, pub *PublishedTrack) {
	info := TrackInfo{
		TrackID: pub.TrackID,
		Kind:    pub.Kind,
		Source:  pub.Source,
		Codec:   pub.Codec,
	}
	for _, other := range r.Participants() {
		if other.id == p.id {
			continue
		}
		_ = other.ws.sendJSON(Envelope{Type: MsgTrackPublished}, TrackPublishedData{
			ParticipantID: p.id,
			Track:         info,
		})
		// deferSubscription is a safe superset of SubscribeTo: it
		// routes to immediate AddTrack when `other` has finished its
		// initial handshake (the common case here), and queues
		// otherwise — covering the corner case where one peer publishes
		// before another has answered its joined offer.
		other.deferSubscription(p.id, pub)
	}
	r.manager.webhook.emit(Event{
		Type:          EventTrackPublished,
		Room:          r.name,
		ParticipantID: p.id,
		Identity:      p.identity,
		TrackID:       pub.TrackID,
		TrackKind:     pub.Kind,
		Timestamp:     time.Now().UnixMilli(),
	})
}

// announceUnpublish is the mirror of announcePublish; called during
// participant teardown.
func (r *Room) announceUnpublish(participantID, trackID string) {
	for _, other := range r.Participants() {
		if other.id == participantID {
			continue
		}
		_ = other.ws.sendJSON(Envelope{Type: MsgTrackUnpublished}, TrackUnpublishedData{
			ParticipantID: participantID,
			TrackID:       trackID,
		})
	}
}

// broadcast sends env+payload to every participant except skipID.
func (r *Room) broadcast(skipID string, env Envelope, payload any) {
	for _, p := range r.Participants() {
		if p.id == skipID {
			continue
		}
		_ = p.ws.sendJSON(env, payload)
	}
}

// armIdleGC schedules room deletion after RoomIdleTTL; cancelled if
// anyone joins in the meantime.
func (r *Room) armIdleGC() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.idleTimer != nil {
		return
	}
	// Zero TTL means GC immediately.
	if r.cfg.RoomIdleTTL <= 0 {
		r.manager.dropRoom(r.name)
		return
	}
	r.idleTimer = time.AfterFunc(r.cfg.RoomIdleTTL, func() {
		r.mu.Lock()
		if len(r.participants) > 0 {
			r.mu.Unlock()
			return
		}
		r.mu.Unlock()
		r.manager.dropRoom(r.name)
	})
}

// newParticipantID returns a short hex ID suitable for log correlation
// without leaking identity info. 64 bits of entropy is plenty for a
// room's lifetime uniqueness.
func newParticipantID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
