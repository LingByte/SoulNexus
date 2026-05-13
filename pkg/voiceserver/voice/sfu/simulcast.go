// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"github.com/pion/rtp"
	pionwebrtc "github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

// preferredLayerOrder is how a SimulcastForwarder picks which RID to
// forward when multiple simulcast layers are published. Highest first:
// "f" (full), "h" (half), "q" (quarter). When none of these RIDs are
// present (non-simulcast stream) we simply forward whatever layer
// OnTrack delivered.
var preferredLayerOrder = []string{"f", "h", "q"}

// SimulcastForwarder multiplexes 1..N incoming simulcast layers from a
// publisher onto one outbound TrackLocalStaticRTP that is AddTrack'd to
// every subscriber. It picks the highest available layer at any moment
// and switches atomically when that layer disappears (e.g. browser
// downgrades under bandwidth pressure).
//
// Key properties
//
//   - Thread-safe: layers can be added/removed from any goroutine
//   - Lock-free forwarding hot path: one atomic load per RTP packet
//   - Key-frame preserving: layer switches are guarded so we only
//     promote on a new keyframe (for video). For audio there is only
//     one layer; the forwarder degenerates to a passthrough.
//
// Field ownership (mu)
//
//   - `layers` keys/values only change with mu held
//   - `active` is swapped atomically; forwarding goroutines read it
//     without mu
type SimulcastForwarder struct {
	kind   string                          // "audio" | "video"
	local  *pionwebrtc.TrackLocalStaticRTP // single downstream track we write to
	logger *zap.Logger

	mu     sync.Mutex
	layers map[string]*layer // keyed by RID ("" for non-simulcast)
	// active is the currently-forwarded layer RID. Empty = none picked.
	// Read lock-free on the write path.
	active atomic.Value // string

	// payloadSink, if set, receives a copy of every RTP payload that
	// gets forwarded on the currently-active layer. Used by the
	// recording subsystem for audio-track capture. Set once before any
	// Attach so we don't need synchronisation on the hot path.
	payloadSink func(payload []byte)

	// muted, when true, causes pump() to skip WriteRTP on the
	// downstream track AND skip the payloadSink call. This is the
	// server-enforced half of MsgSetMute: even if the publisher's
	// browser misbehaves and keeps sending RTP, the SFU drops it.
	muted atomic.Bool
}

// SetPayloadSink wires a callback that gets called with each forwarded
// RTP packet's payload (only on the active simulcast layer, so we don't
// double-record). Must be set before the first Attach for race safety.
func (f *SimulcastForwarder) SetPayloadSink(fn func(payload []byte)) {
	f.payloadSink = fn
}

// SetMuted toggles server-side packet suppression. Called by
// Participant.HandleSetMute. Lock-free atomic so the forwarding hot
// path stays branch-cheap.
func (f *SimulcastForwarder) SetMuted(muted bool) { f.muted.Store(muted) }

// Muted reports the current muted state.
func (f *SimulcastForwarder) Muted() bool { return f.muted.Load() }

type layer struct {
	rid    string
	remote *pionwebrtc.TrackRemote
	// seenKeyframe goes true once we see one video keyframe; until then
	// we refuse to promote this layer as `active` because subscribers
	// would stall waiting for one.
	seenKeyframe atomic.Bool
}

// NewSimulcastForwarder allocates the downstream TrackLocalStaticRTP
// and wires logging. `codec` is the codec capability the published
// track will use (opus or vp8).
func NewSimulcastForwarder(trackID, streamID, kind string, codec pionwebrtc.RTPCodecCapability, logger *zap.Logger) (*SimulcastForwarder, error) {
	local, err := pionwebrtc.NewTrackLocalStaticRTP(codec, trackID, streamID)
	if err != nil {
		return nil, err
	}
	f := &SimulcastForwarder{
		kind:   kind,
		local:  local,
		logger: logger.With(zap.String("track", trackID), zap.String("kind", kind)),
		layers: make(map[string]*layer),
	}
	f.active.Store("")
	return f, nil
}

// Local returns the downstream track that should be AddTrack'd to
// subscriber PeerConnections.
func (f *SimulcastForwarder) Local() *pionwebrtc.TrackLocalStaticRTP { return f.local }

// Attach registers one incoming simulcast layer and spawns a read
// pump. When that layer becomes (or remains) the best choice, its
// packets get forwarded to the local downstream track. When a higher-
// priority layer is added later, this layer is silently demoted — the
// read pump keeps draining so the RTP buffer doesn't back up, but its
// packets are not forwarded.
//
// readRTCPFn is invoked with any RTCP packets that should be sent back
// upstream to the publisher. Callers typically wire this into
// Participant.sendRTCP so PLI / NACK etc. reach the publisher PC.
func (f *SimulcastForwarder) Attach(remote *pionwebrtc.TrackRemote, onRTCP func(pionwebrtc.RTCPFeedback)) {
	rid := remote.RID()
	l := &layer{rid: rid, remote: remote}
	f.mu.Lock()
	f.layers[rid] = l
	// If this is the only layer, pick it immediately. For audio that's
	// always the case; for video a higher layer may arrive later and
	// supersede via rebalance().
	if f.active.Load().(string) == "" {
		f.active.Store(rid)
	} else {
		f.rebalanceLocked()
	}
	f.mu.Unlock()

	go f.pump(l)
}

// Detach removes a layer (e.g. browser stopped publishing a RID due to
// bandwidth). If the removed layer was the active one, rebalance picks
// the next best.
func (f *SimulcastForwarder) Detach(rid string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.layers, rid)
	if f.active.Load().(string) == rid {
		f.active.Store("")
		f.rebalanceLocked()
	}
}

// rebalanceLocked must be called with f.mu held. Picks the highest
// preferredLayerOrder layer that has seen a keyframe; falls back to
// the first arrived layer if nothing has keyframed yet.
func (f *SimulcastForwarder) rebalanceLocked() {
	for _, rid := range preferredLayerOrder {
		if l, ok := f.layers[rid]; ok {
			if f.kind == "audio" || l.seenKeyframe.Load() {
				f.active.Store(rid)
				return
			}
		}
	}
	// Non-simulcast path: one anonymous layer keyed with empty RID.
	if l, ok := f.layers[""]; ok {
		if f.kind == "audio" || l.seenKeyframe.Load() {
			f.active.Store("")
			return
		}
	}
	// Nothing eligible — drop active. A layer will call rebalance again
	// after its first keyframe.
	f.active.Store("")
}

// pump reads packets from one RTP layer and forwards them to the
// downstream track when that layer is currently `active`. Exits when
// the layer errors (publisher unpublished or PC died).
func (f *SimulcastForwarder) pump(l *layer) {
	// Pre-allocated buffer sized for typical RTP MTU + header. pion's
	// ReadRTP allocates internally so we can't avoid alloc; at least
	// keep this out of the hot path.
	for {
		pkt, _, err := l.remote.ReadRTP()
		if err != nil {
			if errors.Is(err, io.EOF) {
				f.logger.Debug("layer ended", zap.String("rid", l.rid))
			} else {
				f.logger.Debug("layer read error", zap.String("rid", l.rid), zap.Error(err))
			}
			f.Detach(l.rid)
			return
		}

		// Track keyframes for video to unblock layer promotion.
		if f.kind == "video" && !l.seenKeyframe.Load() && isVP8Keyframe(pkt) {
			l.seenKeyframe.Store(true)
			f.mu.Lock()
			f.rebalanceLocked()
			f.mu.Unlock()
		}

		// Forward only when this layer is the chosen one. Cheap lock-
		// free check on the hot path.
		if active, _ := f.active.Load().(string); active != l.rid {
			continue
		}
		// Server-enforced mute: drop the packet outright. We also skip
		// the recording sink — a muted speaker shouldn't end up in the
		// recording even if their browser keeps sending RTP.
		if f.muted.Load() {
			continue
		}
		if f.payloadSink != nil {
			f.payloadSink(pkt.Payload)
		}
		if err := f.local.WriteRTP(pkt); err != nil {
			// Writing to a local track only fails when the track is
			// closed, which happens during teardown. Bail quietly.
			return
		}
	}
}

// isVP8Keyframe returns true when the RTP payload starts with a VP8
// keyframe descriptor. VP8 keyframes are identifiable by the `P` bit
// in the VP8 payload header being 0 (inverse bit — 0 means keyframe).
//
// Layout (simplified):
//
//	| VP8 payload descriptor (1–6 bytes) | VP8 payload |
//
// First byte of the VP8 payload has bit 0 (LSB) == 0 for keyframes.
// We parse just enough of the descriptor to locate the payload.
func isVP8Keyframe(pkt *rtp.Packet) bool {
	p := pkt.Payload
	if len(p) == 0 {
		return false
	}
	// VP8 payload descriptor byte 0: X|R|N|S|R|PID
	b0 := p[0]
	offset := 1
	// X bit → extended descriptor follows
	if b0&0x80 != 0 {
		if len(p) < 2 {
			return false
		}
		b1 := p[1]
		offset++
		// I bit = PictureID present (1 or 2 bytes)
		if b1&0x80 != 0 {
			if len(p) < offset+1 {
				return false
			}
			if p[offset]&0x80 != 0 {
				offset += 2
			} else {
				offset++
			}
		}
		// L bit = TL0PICIDX (1 byte)
		if b1&0x40 != 0 {
			offset++
		}
		// T bit or K bit = TID/KEYIDX (1 byte)
		if b1&0x30 != 0 {
			offset++
		}
	}
	if len(p) <= offset {
		return false
	}
	// VP8 payload header byte: size0|H|VER|P
	return p[offset]&0x01 == 0
}
