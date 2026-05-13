// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/rtcp"
	pionwebrtc "github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

// Participant owns one WebSocket + one pion PeerConnection. The same
// PC handles both the peer's uploads (OnTrack → fan out) and its
// downloads (TrackLocalStaticRTP added per subscribed source). Every
// public method is safe to call concurrently from the signaling and
// media goroutines.
type Participant struct {
	id       string
	identity string
	name     string
	metadata string

	room   *Room
	cfg    *Config
	logger *zap.Logger

	permissions Permissions
	joinedAt    time.Time

	ws *wsConn
	pc *pionwebrtc.PeerConnection

	mu sync.RWMutex
	// published tracks from this peer, keyed by the TrackRemote.ID().
	// The key is what other peers call out in trackUnpublished events.
	published map[string]*PublishedTrack
	// subscriptions[sourceParticipantID][trackID] = the RTPSender we
	// gave this peer so we can RemoveTrack on unpublish.
	subscriptions map[string]map[string]*pionwebrtc.RTPSender

	// negotiating serialises server-initiated offers (renegotiation) so
	// we never pipeline two offers and confuse the pion state machine.
	negotiating chan struct{}

	// negotiated flips to true the first time HandleOffer succeeds.
	// Subscriptions queued before this point are deferred (see
	// deferSubscription / flushDeferredSubscriptions) because adding
	// tracks to the PC before the client has produced its first SDP
	// offer would cause the answer to contain m-lines the offer did
	// not, which browsers reject.
	negotiated   atomic.Bool
	pendingSubs  []pendingSub
	pendingSubMu sync.Mutex

	closed atomic.Bool
}

// pendingSub remembers one (source, pub) pair that wants to be sent
// down to this peer once the initial SDP handshake completes.
type pendingSub struct {
	sourceID string
	pub      *PublishedTrack
}

// PublishedTrack is one media track this peer has published to the
// room, plus the simulcast forwarder fanning it out. Each published
// track has exactly one forwarder even if the browser sent 3
// simulcast layers.
type PublishedTrack struct {
	TrackID  string
	Kind     string
	StreamID string
	Codec    string
	Source   string // "camera"/"microphone"/"screen" — advisory; from track.StreamID or client-declared

	Forwarder *SimulcastForwarder
	muted     atomic.Bool
	// recording, if non-nil, captures this published audio track to
	// the configured store. Only set for "audio" kind when
	// Config.EnableRecording is true. Closed when the participant
	// tears down (see Participant.Close).
	recording *recordingSession
}

// Muted returns whether forwarding is currently suppressed (client
// asked the SFU to stop, or the track is gone but we haven't torn
// down yet).
func (p *PublishedTrack) Muted() bool { return p.muted.Load() }

// ===== Lifecycle =====

// newParticipant wires up the pion PeerConnection but does not yet
// perform the first SDP handshake — that happens when the client sends
// its initial MsgOffer.
func newParticipant(id string, claims AccessTokenClaims, ws *wsConn, room *Room) (*Participant, error) {
	perms := DefaultPermissions()
	if claims.Permissions != nil {
		perms = *claims.Permissions
	}
	name := strings.TrimSpace(claims.Name)
	if name == "" {
		name = claims.Identity
	}

	p := &Participant{
		id:            id,
		identity:      claims.Identity,
		name:          name,
		metadata:      claims.Metadata,
		room:          room,
		cfg:           room.cfg,
		logger:        room.logger.With(zap.String("participant", id), zap.String("identity", claims.Identity)),
		permissions:   perms,
		joinedAt:      time.Now(),
		ws:            ws,
		published:     make(map[string]*PublishedTrack),
		subscriptions: make(map[string]map[string]*pionwebrtc.RTPSender),
		negotiating:   make(chan struct{}, 1),
	}

	pc, err := room.api.NewPeerConnection(pionwebrtc.Configuration{
		ICEServers:           room.cfg.ICEServers,
		ICETransportPolicy:   pionwebrtc.ICETransportPolicyAll,
		BundlePolicy:         pionwebrtc.BundlePolicyMaxBundle,
		RTCPMuxPolicy:        pionwebrtc.RTCPMuxPolicyRequire,
		SDPSemantics:         pionwebrtc.SDPSemanticsUnifiedPlan,
	})
	if err != nil {
		return nil, fmt.Errorf("sfu: new peer connection: %w", err)
	}
	p.pc = pc

	// Wire handlers. Order matters — we register OnTrack before the
	// first SetRemoteDescription so pion doesn't miss the first frame.
	pc.OnTrack(p.onTrack)
	pc.OnICECandidate(p.onICECandidate)
	pc.OnICEConnectionStateChange(p.onICEConnectionStateChange)
	pc.OnConnectionStateChange(p.onConnectionStateChange)

	return p, nil
}

// ID returns the server-assigned participant identifier. Different
// sessions from the same Identity get different IDs.
func (p *Participant) ID() string { return p.id }

// Identity returns the Identity claim from the access token.
func (p *Participant) Identity() string { return p.identity }

// Permissions returns a read-only copy of this peer's permissions.
func (p *Participant) Permissions() Permissions { return p.permissions }

// Info returns the ParticipantInfo shape other peers see.
func (p *Participant) Info() ParticipantInfo {
	p.mu.RLock()
	tracks := make([]TrackInfo, 0, len(p.published))
	for _, t := range p.published {
		tracks = append(tracks, TrackInfo{
			TrackID: t.TrackID,
			Kind:    t.Kind,
			Source:  t.Source,
			Muted:   t.Muted(),
			Codec:   t.Codec,
		})
	}
	perms := p.permissions
	p.mu.RUnlock()
	return ParticipantInfo{
		ParticipantID: p.id,
		Identity:      p.identity,
		Name:          p.name,
		Metadata:      p.metadata,
		Tracks:        tracks,
		Permissions:   &perms,
		JoinedAt:      p.joinedAt.UnixMilli(),
	}
}

// Close tears the PeerConnection down and notifies the Room so peers
// see the MsgParticipantLeft. Idempotent.
func (p *Participant) Close(reason string) {
	if !p.closed.CompareAndSwap(false, true) {
		return
	}
	p.logger.Info("participant closing", zap.String("reason", reason))
	// Finalise recordings BEFORE closing the PC so any tail packets
	// already flushed through SetPayloadSink make it into the WAV.
	p.mu.Lock()
	for _, pub := range p.published {
		if pub.recording != nil {
			go pub.recording.Close()
		}
	}
	p.mu.Unlock()
	if p.pc != nil {
		_ = p.pc.Close()
	}
	if p.ws != nil {
		p.ws.close()
	}
	p.room.removeParticipant(p, reason)
}

// ===== Publish path (client → SFU) =====

// onTrack fires when pion receives a new inbound track from this
// peer's upload. For simulcast video the same (publisher, trackID)
// pair fires OnTrack once per RID. We demultiplex by TrackID+MSID,
// attaching the newly-arrived layer to the existing forwarder when
// appropriate.
func (p *Participant) onTrack(remote *pionwebrtc.TrackRemote, _ *pionwebrtc.RTPReceiver) {
	if p.closed.Load() {
		return
	}
	// PublishedTrack identity key: we collapse simulcast layers that
	// share MSID (media stream identifier) + kind into one logical
	// track. For non-simulcast tracks MSID == remote.ID().
	logicalID := remote.ID()
	kind := remote.Kind().String()
	p.logger.Info("track received",
		zap.String("trackId", logicalID),
		zap.String("kind", kind),
		zap.String("rid", remote.RID()),
		zap.String("codec", remote.Codec().MimeType),
	)

	p.mu.Lock()
	pub, ok := p.published[logicalID]
	if !ok {
		fwd, err := NewSimulcastForwarder(logicalID, p.id, kind, remote.Codec().RTPCodecCapability, p.logger)
		if err != nil {
			p.mu.Unlock()
			p.logger.Error("new forwarder", zap.Error(err))
			return
		}
		pub = &PublishedTrack{
			TrackID:   logicalID,
			Kind:      kind,
			StreamID:  p.id,
			Codec:     remote.Codec().MimeType,
			Source:    kind, // advisory
			Forwarder: fwd,
		}
		// Wire per-track audio recording. We only record audio because
		// VP8 → container conversion is out of scope for the SFU; if a
		// caller wants composed video they can run a recorder bot that
		// subscribes like a normal participant.
		if kind == pionKindAudio && p.cfg.EnableRecording {
			pub.recording = newRecordingSession(p.cfg, p.logger, p.room.manager.webhook,
				p.room.name, p.id, p.identity, logicalID)
			fwd.SetPayloadSink(pub.recording.sinkPayload())
		}
		p.published[logicalID] = pub
	}
	p.mu.Unlock()

	// Forward RTP onto the simulcast forwarder. PLI / NACK forwarding
	// upstream is handled by readRTCP per subscriber, not by an
	// Attach-time callback.
	pub.Forwarder.Attach(remote, nil)

	// Only announce logical track once (first attach, not per-RID).
	if !ok {
		// Fan out to every other participant.
		p.room.announcePublish(p, pub)
	}
}

// ===== Subscribe path (SFU → client) =====

// deferSubscription queues a subscription to be applied after the
// participant's initial offer/answer has completed. Calling SubscribeTo
// before that point would AddTrack onto a PC whose first SDP exchange
// hasn't happened yet — the resulting answer would expose m-lines the
// browser's offer never described, which most browsers reject.
func (p *Participant) deferSubscription(sourceID string, pub *PublishedTrack) {
	if p.closed.Load() || !p.permissions.CanSubscribe {
		return
	}
	if p.negotiated.Load() {
		// Initial negotiation already done; just subscribe directly.
		p.SubscribeTo(sourceID, pub)
		return
	}
	p.pendingSubMu.Lock()
	p.pendingSubs = append(p.pendingSubs, pendingSub{sourceID: sourceID, pub: pub})
	p.pendingSubMu.Unlock()
}

// flushDeferredSubscriptions executes every queued subscription against
// the now-negotiated PC. Run from a goroutine to keep the calling
// HandleOffer hot path snappy and let pion finish whatever queueing it
// does after SetLocalDescription returns.
func (p *Participant) flushDeferredSubscriptions() {
	p.pendingSubMu.Lock()
	pending := p.pendingSubs
	p.pendingSubs = nil
	p.pendingSubMu.Unlock()
	for _, sub := range pending {
		p.SubscribeTo(sub.sourceID, sub.pub)
	}
}

// SubscribeTo attaches the given PublishedTrack from `source` to this
// participant's PC. Triggers a renegotiation so pion emits an
// OnNegotiationNeeded which we convert to a signaling offer.
//
// Must only be called after the participant has completed its initial
// offer/answer. Callers during AddParticipant should use
// deferSubscription instead.
func (p *Participant) SubscribeTo(sourceID string, pub *PublishedTrack) {
	if p.closed.Load() || !p.permissions.CanSubscribe {
		return
	}
	p.mu.Lock()
	if _, exists := p.subscriptions[sourceID][pub.TrackID]; exists {
		p.mu.Unlock()
		return
	}
	sender, err := p.pc.AddTrack(pub.Forwarder.Local())
	if err != nil {
		p.mu.Unlock()
		p.logger.Warn("subscribe addTrack", zap.String("source", sourceID), zap.String("track", pub.TrackID), zap.Error(err))
		return
	}
	if p.subscriptions[sourceID] == nil {
		p.subscriptions[sourceID] = make(map[string]*pionwebrtc.RTPSender)
	}
	p.subscriptions[sourceID][pub.TrackID] = sender
	p.mu.Unlock()

	// Drain RTCP from the sender so pion's congestion control can react.
	go p.readRTCP(sender, sourceID, pub)

	// Ask pion to renegotiate. For the FIRST subscribe that happens
	// before the client has offered, we skip the explicit offer — the
	// track will be carried by the client's own answer implicitly.
	p.triggerRenegotiation()
}

// UnsubscribeFrom reverses a SubscribeTo. Fired when a publisher
// disappears from the room.
func (p *Participant) UnsubscribeFrom(sourceID, trackID string) {
	p.mu.Lock()
	sender, ok := p.subscriptions[sourceID][trackID]
	if !ok {
		p.mu.Unlock()
		return
	}
	delete(p.subscriptions[sourceID], trackID)
	if len(p.subscriptions[sourceID]) == 0 {
		delete(p.subscriptions, sourceID)
	}
	p.mu.Unlock()
	if sender != nil {
		_ = p.pc.RemoveTrack(sender)
	}
	p.triggerRenegotiation()
}

// readRTCP drains feedback from one downstream sender and forwards
// keyframe requests (PLI/FIR) to the original publisher so the
// upstream encoder produces a fresh keyframe. This is how a
// subscriber's "I joined mid-stream, I need a keyframe" becomes a
// round trip all the way back to the browser that owns the camera.
func (p *Participant) readRTCP(sender *pionwebrtc.RTPSender, sourceID string, pub *PublishedTrack) {
	buf := make([]byte, 1500)
	for !p.closed.Load() {
		n, _, err := sender.Read(buf)
		if err != nil {
			return
		}
		pkts, err := rtcp.Unmarshal(buf[:n])
		if err != nil {
			continue
		}
		for _, pkt := range pkts {
			switch pkt.(type) {
			case *rtcp.PictureLossIndication, *rtcp.FullIntraRequest:
				// Forward upstream: request a keyframe from the publisher.
				if source := p.room.ParticipantByID(sourceID); source != nil && source.pc != nil {
					// Find the media SSRC from the published track and
					// emit a PLI targeting it. RemoteTrack SSRCs are
					// rid-specific; emitting on each is benign pion-side.
					for _, recv := range source.pc.GetReceivers() {
						for _, t := range recv.Tracks() {
							if t.ID() != pub.TrackID {
								continue
							}
							_ = source.pc.WriteRTCP([]rtcp.Packet{
								&rtcp.PictureLossIndication{MediaSSRC: uint32(t.SSRC())},
							})
						}
					}
				}
			}
		}
	}
}

// ===== Negotiation plumbing =====

// HandleOffer processes a client-initiated offer (first negotiation or
// new publication). Returns the SDP answer to send back.
func (p *Participant) HandleOffer(sdp string) (string, error) {
	if p.closed.Load() {
		return "", ErrParticipantGone
	}
	if err := p.pc.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeOffer,
		SDP:  sdp,
	}); err != nil {
		return "", fmt.Errorf("sfu: set remote offer: %w", err)
	}
	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		return "", fmt.Errorf("sfu: create answer: %w", err)
	}
	gatherComplete := pionwebrtc.GatheringCompletePromise(p.pc)
	if err := p.pc.SetLocalDescription(answer); err != nil {
		return "", fmt.Errorf("sfu: set local answer: %w", err)
	}
	// Wait up to 1s for full ICE gathering — yields a "complete"
	// answer (no trickle needed) for simple deployments. Anything
	// missed by the deadline still reaches the client via trickled
	// MsgICECandidate frames. Trimmed from 3s to 1s so a slow gather
	// (CGNAT, IPv6 dual-stack) doesn't make a user's leave/mute click
	// feel laggy — the dispatch goroutine is blocked here.
	select {
	case <-gatherComplete:
	case <-time.After(time.Second):
	}
	local := p.pc.LocalDescription()
	if local == nil {
		return "", errors.New("sfu: no local description")
	}
	// First successful answer → flush any subscriptions queued while the
	// peer was still mid-handshake. Done in a goroutine so this RPC
	// returns immediately and the answer reaches the client before the
	// renegotiation offers we are about to emit.
	if p.negotiated.CompareAndSwap(false, true) {
		go p.flushDeferredSubscriptions()
	}
	return local.SDP, nil
}

// HandleAnswer processes a client answer to a server-initiated offer
// (renegotiation after new publications).
func (p *Participant) HandleAnswer(sdp string) error {
	if p.closed.Load() {
		return ErrParticipantGone
	}
	err := p.pc.SetRemoteDescription(pionwebrtc.SessionDescription{
		Type: pionwebrtc.SDPTypeAnswer,
		SDP:  sdp,
	})
	if err != nil {
		return fmt.Errorf("sfu: set remote answer: %w", err)
	}
	// Release negotiating slot so a queued renegotiation can proceed.
	select {
	case <-p.negotiating:
	default:
	}
	return nil
}

// HandleICECandidate applies a trickled candidate from the client.
func (p *Participant) HandleICECandidate(cand ICECandidateData) error {
	if p.closed.Load() {
		return ErrParticipantGone
	}
	return p.pc.AddICECandidate(pionwebrtc.ICECandidateInit{
		Candidate:     cand.Candidate,
		SDPMid:        ptrString(cand.SDPMid),
		SDPMLineIndex: cand.SDPMLineIndex,
	})
}

// HandleSetMute updates the muted flag on a published track AND tells
// the simulcast forwarder to drop packets server-side. The browser is
// expected to also stop sending — the forwarder gate is defence in
// depth against a misbehaving sender. We also broadcast the new state
// so other participants' UIs can render a muted indicator.
func (p *Participant) HandleSetMute(trackID string, muted bool) {
	p.mu.RLock()
	pub, ok := p.published[trackID]
	p.mu.RUnlock()
	if !ok {
		return
	}
	pub.muted.Store(muted)
	if pub.Forwarder != nil {
		pub.Forwarder.SetMuted(muted)
	}
	// Re-announce the track with the new muted state so peers learn.
	p.room.broadcast(p.id, Envelope{Type: MsgTrackPublished}, TrackPublishedData{
		ParticipantID: p.id,
		Track: TrackInfo{
			TrackID: pub.TrackID,
			Kind:    pub.Kind,
			Source:  pub.Source,
			Muted:   muted,
			Codec:   pub.Codec,
		},
	})
}

// triggerRenegotiation asks pion to create a new offer and sends it
// on the WS. Serialised via p.negotiating so we never send two offers
// back-to-back without waiting for the answer.
func (p *Participant) triggerRenegotiation() {
	select {
	case p.negotiating <- struct{}{}:
	default:
		// An offer is already in flight. The track was just added
		// synchronously to the PC, so the NEXT renegotiation will
		// include it automatically once the current answer arrives.
		return
	}
	go func() {
		// Small debounce so burst-subscribes (e.g. 5 peers joining at
		// once) produce one offer, not five.
		time.Sleep(50 * time.Millisecond)
		offer, err := p.pc.CreateOffer(nil)
		if err != nil {
			p.logger.Warn("create renegotiation offer", zap.Error(err))
			select {
			case <-p.negotiating:
			default:
			}
			return
		}
		if err := p.pc.SetLocalDescription(offer); err != nil {
			p.logger.Warn("set local renegotiation offer", zap.Error(err))
			select {
			case <-p.negotiating:
			default:
			}
			return
		}
		local := p.pc.LocalDescription()
		if local == nil {
			select {
			case <-p.negotiating:
			default:
			}
			return
		}
		if err := p.ws.sendJSON(Envelope{Type: MsgOffer}, SDPData{SDP: local.SDP}); err != nil {
			p.logger.Warn("send renegotiation offer", zap.Error(err))
			select {
			case <-p.negotiating:
			default:
			}
			return
		}
	}()
}

// onICECandidate streams server-gathered candidates to the client as
// they arrive. The final nil candidate is skipped (clients detect end
// via SDP a=end-of-candidates or the gatherComplete promise server-
// side).
func (p *Participant) onICECandidate(c *pionwebrtc.ICECandidate) {
	if c == nil || p.closed.Load() {
		return
	}
	init := c.ToJSON()
	data := ICECandidateData{
		Candidate: init.Candidate,
	}
	if init.SDPMid != nil {
		data.SDPMid = *init.SDPMid
	}
	if init.SDPMLineIndex != nil {
		// pion exposes this as a uint16 in our protocol for symmetry
		// with the wire format browsers emit.
		v := uint16(*init.SDPMLineIndex)
		data.SDPMLineIndex = &v
	}
	_ = p.ws.sendJSON(Envelope{Type: MsgICECandidate}, data)
}

// onICEConnectionStateChange triggers an ICE restart when the
// connection fails. Pion's state machine issues a fresh offer/answer
// with new ICE ufrag/pwd — transparent to higher layers.
func (p *Participant) onICEConnectionStateChange(state pionwebrtc.ICEConnectionState) {
	p.logger.Debug("ice state", zap.String("state", state.String()))
	if state == pionwebrtc.ICEConnectionStateFailed {
		p.logger.Warn("ice failed, issuing restart")
		// Tell the client (informational — some clients display a
		// reconnect spinner during restart) then kick pion.
		_ = p.ws.sendJSON(Envelope{Type: MsgICERestart}, struct{}{})
		// pion's CreateOffer accepts ICERestart option. We serialise
		// through negotiating to avoid racing a pending answer.
		select {
		case p.negotiating <- struct{}{}:
		default:
			return
		}
		go func() {
			// Helper to drain the negotiating slot. We release it on
			// every error path AND when the client never answers within
			// a watchdog window so subsequent restart attempts can
			// proceed instead of being silently skipped.
			release := func() {
				select {
				case <-p.negotiating:
				default:
				}
			}
			offer, err := p.pc.CreateOffer(&pionwebrtc.OfferOptions{ICERestart: true})
			if err != nil {
				p.logger.Warn("ice restart offer", zap.Error(err))
				release()
				return
			}
			if err := p.pc.SetLocalDescription(offer); err != nil {
				p.logger.Warn("ice restart set-local", zap.Error(err))
				release()
				return
			}
			local := p.pc.LocalDescription()
			if local == nil {
				release()
				return
			}
			if err := p.ws.sendJSON(Envelope{Type: MsgOffer}, SDPData{SDP: local.SDP}); err != nil {
				p.logger.Warn("ice restart send offer", zap.Error(err))
				release()
				return
			}
			// Watchdog: if the client never answers we'd keep the slot
			// forever and block future renegotiations. 10s is enough
			// for any real client to round-trip an answer; HandleAnswer
			// races with this and drains the slot via the same select.
			time.AfterFunc(10*time.Second, release)
		}()
	}
}

// onConnectionStateChange closes the participant when the overall PC
// transitions to failed or disconnected for an extended period. We
// trust pion's ICE timers to differentiate transient blips from real
// outages; here we only react to the terminal states.
func (p *Participant) onConnectionStateChange(state pionwebrtc.PeerConnectionState) {
	p.logger.Debug("pc state", zap.String("state", state.String()))
	switch state {
	case pionwebrtc.PeerConnectionStateClosed, pionwebrtc.PeerConnectionStateFailed:
		p.Close("pc_" + state.String())
	}
}

func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
