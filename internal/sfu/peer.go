// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type wsEnvelope struct {
	Type      string                   `json:"type"`
	SDP       string                   `json:"sdp,omitempty"`
	Error     string                   `json:"error,omitempty"`
	Candidate *webrtc.ICECandidateInit `json:"candidate,omitempty"`
}

// Peer is one participant's PeerConnection to the SFU plus its signaling socket.
type Peer struct {
	room *Room
	id   string
	conn *websocket.Conn
	pc   *webrtc.PeerConnection

	wsReadTimeout  time.Duration
	wsPingInterval time.Duration

	writeMu sync.Mutex
	negMu   sync.Mutex

	fwdMu sync.Mutex
	fwd   map[string]struct{}

	subqMu    sync.Mutex
	pending   []pendingSub
	renoTimer *time.Timer
	needsReno bool
}

type pendingSub struct {
	uf    *upstreamFanout
	owner string
	tr    *webrtc.TrackRemote
}

func newPeer(room *Room, id string, conn *websocket.Conn, ice []webrtc.ICEServer, wsRead, wsPing time.Duration) (*Peer, error) {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{ICEServers: ice})
	if err != nil {
		return nil, err
	}
	p := &Peer{
		room:           room,
		id:             id,
		conn:           conn,
		pc:             pc,
		wsReadTimeout:  wsRead,
		wsPingInterval: wsPing,
		fwd:            make(map[string]struct{}),
	}
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		j := c.ToJSON()
		_ = p.writeJSON(wsEnvelope{Type: "ice", Candidate: &j})
	})
	pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		p.room.publisherTrackPublished(p, track)
	})
	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		if s == webrtc.PeerConnectionStateConnected {
			p.flushSubscriptions()
			p.room.attachAllUpstreamsToSubscriber(p)
			p.scheduleRenegotiate()
		}
	})
	return p, nil
}

func (p *Peer) writeJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	return p.conn.WriteMessage(websocket.TextMessage, b)
}

func (p *Peer) queueSubscription(uf *upstreamFanout, owner string, tr *webrtc.TrackRemote) {
	key := upstreamKey(owner, tr.ID())
	p.subqMu.Lock()
	defer p.subqMu.Unlock()
	for _, x := range p.pending {
		if upstreamKey(x.owner, x.tr.ID()) == key {
			return
		}
	}
	p.pending = append(p.pending, pendingSub{uf: uf, owner: owner, tr: tr})
}

func (p *Peer) flushSubscriptions() {
	p.subqMu.Lock()
	items := p.pending
	p.pending = nil
	p.subqMu.Unlock()
	for _, it := range items {
		key := upstreamKey(it.owner, it.tr.ID())
		p.doSubscribe(it.uf, it.owner, it.tr, key)
	}
}

func (p *Peer) subscribeUpstream(uf *upstreamFanout, owner string, remote *webrtc.TrackRemote) {
	key := upstreamKey(owner, remote.ID())
	if p.pc.ConnectionState() != webrtc.PeerConnectionStateConnected {
		p.queueSubscription(uf, owner, remote)
		return
	}
	p.doSubscribe(uf, owner, remote, key)
}

func (p *Peer) doSubscribe(uf *upstreamFanout, owner string, remote *webrtc.TrackRemote, key string) {
	p.fwdMu.Lock()
	if _, ok := p.fwd[key]; ok {
		p.fwdMu.Unlock()
		return
	}
	p.fwd[key] = struct{}{}
	p.fwdMu.Unlock()

	cap := remote.Codec().RTPCodecCapability
	tid := "fwd-" + owner + "-" + remote.ID()
	local, err := webrtc.NewTrackLocalStaticRTP(cap, tid, remote.StreamID())
	if err != nil {
		p.dropFwdKey(key)
		return
	}
	sender, err := p.pc.AddTrack(local)
	if err != nil {
		p.dropFwdKey(key)
		return
	}
	uf.addLeg(p, sender, local)
	if pub := p.room.PeerByID(owner); pub != nil && remote.Kind() == webrtc.RTPCodecTypeVideo {
		startSubscriberRTCPRelay(sender, pub, remote)
		burstKeyframeRequestsToPublisher(pub, remote, 4)
	}
	p.scheduleRenegotiate()
}

func (p *Peer) dropFwdKey(key string) {
	p.fwdMu.Lock()
	delete(p.fwd, key)
	p.fwdMu.Unlock()
}

func (p *Peer) scheduleRenegotiate() {
	p.negMu.Lock()
	defer p.negMu.Unlock()
	p.needsReno = true
	if p.renoTimer != nil {
		p.renoTimer.Stop()
	}
	p.renoTimer = time.AfterFunc(80*time.Millisecond, func() {
		_ = p.flushRenegotiate()
	})
}

func (p *Peer) flushRenegotiate() error {
	p.negMu.Lock()
	if !p.needsReno {
		p.negMu.Unlock()
		return nil
	}
	p.needsReno = false
	if p.pc.SignalingState() != webrtc.SignalingStateStable {
		p.needsReno = true
		p.negMu.Unlock()
		return nil
	}
	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		p.needsReno = true
		p.negMu.Unlock()
		return err
	}
	if err := p.pc.SetLocalDescription(offer); err != nil {
		p.needsReno = true
		p.negMu.Unlock()
		return err
	}
	p.negMu.Unlock()
	return p.writeJSON(wsEnvelope{Type: "server_offer", SDP: offer.SDP})
}

func (p *Peer) handleClientOffer(sdp string) error {
	p.negMu.Lock()
	defer p.negMu.Unlock()
	if err := p.pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}); err != nil {
		return err
	}
	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		return err
	}
	if err := p.pc.SetLocalDescription(answer); err != nil {
		return err
	}
	return p.writeJSON(wsEnvelope{Type: "answer", SDP: answer.SDP})
}

func (p *Peer) handleClientAnswer(sdp string) error {
	p.negMu.Lock()
	err := p.pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdp})
	p.negMu.Unlock()
	if err != nil {
		return err
	}
	go func() {
		time.Sleep(15 * time.Millisecond)
		_ = p.flushRenegotiate()
	}()
	return nil
}

func (p *Peer) handleICE(c *webrtc.ICECandidateInit) error {
	return p.pc.AddICECandidate(*c)
}

func (p *Peer) run() {
	stop := make(chan struct{})
	defer close(stop)
	if p.wsPingInterval > 0 {
		go func() {
			t := time.NewTicker(p.wsPingInterval)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					deadline := time.Now().Add(5 * time.Second)
					p.writeMu.Lock()
					_ = p.conn.WriteControl(websocket.PingMessage, nil, deadline)
					p.writeMu.Unlock()
				}
			}
		}()
	}
	p.conn.SetPongHandler(func(string) error {
		if p.wsReadTimeout > 0 {
			return p.conn.SetReadDeadline(time.Now().Add(2 * p.wsReadTimeout))
		}
		return nil
	})
	for {
		if p.wsReadTimeout > 0 {
			_ = p.conn.SetReadDeadline(time.Now().Add(p.wsReadTimeout))
		}
		_, data, err := p.conn.ReadMessage()
		if err != nil {
			return
		}
		var env wsEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		switch env.Type {
		case "client_offer":
			if err := p.handleClientOffer(env.SDP); err != nil {
				_ = p.writeJSON(wsEnvelope{Type: "error", Error: err.Error()})
			}
		case "client_answer":
			if err := p.handleClientAnswer(env.SDP); err != nil {
				_ = p.writeJSON(wsEnvelope{Type: "error", Error: err.Error()})
			}
		case "ice":
			if env.Candidate != nil {
				_ = p.handleICE(env.Candidate)
			}
		}
	}
}

func (p *Peer) Close() {
	p.negMu.Lock()
	if p.renoTimer != nil {
		p.renoTimer.Stop()
	}
	p.negMu.Unlock()
	_ = p.pc.Close()
}
