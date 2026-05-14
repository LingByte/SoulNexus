package session

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
)

// CodecHandler converts one matched sdp.Codec into the negotiated SDP codec
// line and the media.CodecConfig used by the encoder/decoder pipeline.
//
// Implementations should not mutate offer. Return ok=false to signal the
// handler does not want this codec (e.g. wrong clock rate, unsupported
// channel count) so the negotiator can try the next candidate.
type CodecHandler func(offer sdp.Codec) (neg sdp.Codec, src media.CodecConfig, ok bool)

// CodecNegotiator is a pluggable registry of codec handlers used by
// NegotiateOffer to pick the best supported codec from a remote SDP offer.
//
// Register custom codecs (e.g. G.729, AMR-WB, future video codecs) via
// Register and adjust preference order with SetPreference.
type CodecNegotiator struct {
	mu         sync.RWMutex
	handlers   map[string]CodecHandler // lowercase codec name → handler
	preference []string                // lowercase codec names in priority order
}

// NewCodecNegotiator returns an empty negotiator with no codecs registered.
// Use DefaultCodecNegotiator for the standard audio set (pcma/pcmu/g722/opus).
func NewCodecNegotiator() *CodecNegotiator {
	return &CodecNegotiator{handlers: make(map[string]CodecHandler)}
}

// Register adds or replaces a codec handler.
// name is canonicalised (lowercased and trimmed).
func (n *CodecNegotiator) Register(name string, h CodecHandler) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || h == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.handlers[name] = h
}

// Unregister removes a codec handler (no-op if not present).
func (n *CodecNegotiator) Unregister(name string) {
	name = strings.ToLower(strings.TrimSpace(name))
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.handlers, name)
}

// SetPreference sets the priority list (earlier = preferred). Unknown names
// are ignored. Codecs not present in the list fall back to registration order.
func (n *CodecNegotiator) SetPreference(names ...string) {
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, strings.ToLower(strings.TrimSpace(name)))
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.preference = out
}

// Has reports whether the codec is registered.
func (n *CodecNegotiator) Has(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	n.mu.RLock()
	defer n.mu.RUnlock()
	_, ok := n.handlers[name]
	return ok
}

// Negotiate picks the first supported codec from offer according to the
// configured preference order.
func (n *CodecNegotiator) Negotiate(offer []sdp.Codec) (src media.CodecConfig, neg sdp.Codec, err error) {
	if len(offer) == 0 {
		return media.CodecConfig{}, sdp.Codec{}, fmt.Errorf("sip/session: empty offer")
	}
	n.mu.RLock()
	prefRank := make(map[string]int, len(n.preference))
	for i, name := range n.preference {
		prefRank[name] = i
	}
	handlers := make(map[string]CodecHandler, len(n.handlers))
	for k, v := range n.handlers {
		handlers[k] = v
	}
	n.mu.RUnlock()

	sorted := make([]sdp.Codec, len(offer))
	copy(sorted, offer)
	sort.SliceStable(sorted, func(i, j int) bool {
		ci := strings.ToLower(strings.TrimSpace(sorted[i].Name))
		cj := strings.ToLower(strings.TrimSpace(sorted[j].Name))
		ri, okI := prefRank[ci]
		rj, okJ := prefRank[cj]
		if !okI {
			ri = 1 << 20
		}
		if !okJ {
			rj = 1 << 20
		}
		return ri < rj
	})

	for _, c := range sorted {
		name := strings.ToLower(strings.TrimSpace(c.Name))
		h, ok := handlers[name]
		if !ok {
			continue
		}
		if neg, src, ok := h(c); ok {
			return src, neg, nil
		}
	}
	return media.CodecConfig{}, sdp.Codec{}, fmt.Errorf("sip/session: no supported codec in offer")
}

// DefaultCodecNegotiator returns a negotiator preloaded with pcma/pcmu/g722/opus
// and the traditional telephony preference order (A-law first for PSTN interop).
func DefaultCodecNegotiator() *CodecNegotiator {
	n := NewCodecNegotiator()
	n.Register("pcma", handlerG711("pcma", 8))
	n.Register("pcmu", handlerG711("pcmu", 8))
	n.Register("g722", handlerG722)
	n.Register("opus", handlerOpus)
	n.SetPreference("pcma", "pcmu", "g722", "opus")
	return n
}

// DefaultNegotiator is the process-wide default used by NegotiateOffer.
// Replace with SetDefaultNegotiator before any call has been negotiated.
var (
	defaultNegotiator   = DefaultCodecNegotiator()
	defaultNegotiatorMu sync.RWMutex
)

// SetDefaultNegotiator overrides the package-level negotiator used by NegotiateOffer.
func SetDefaultNegotiator(n *CodecNegotiator) {
	if n == nil {
		return
	}
	defaultNegotiatorMu.Lock()
	defer defaultNegotiatorMu.Unlock()
	defaultNegotiator = n
}

// NegotiateOffer picks the first supported audio codec from a remote SDP offer.
// Thin wrapper over the process-wide default CodecNegotiator for backward compat.
func NegotiateOffer(offer []sdp.Codec) (src media.CodecConfig, neg sdp.Codec, err error) {
	defaultNegotiatorMu.RLock()
	n := defaultNegotiator
	defaultNegotiatorMu.RUnlock()
	return n.Negotiate(offer)
}

// ---------- Built-in handlers for the default audio codecs ----------------

func handlerG711(name string, bitDepth int) CodecHandler {
	return func(c sdp.Codec) (sdp.Codec, media.CodecConfig, bool) {
		ch := c.Channels
		if ch < 1 {
			ch = 1
		}
		neg := sdp.Codec{PayloadType: c.PayloadType, Name: name, ClockRate: c.ClockRate, Channels: ch}
		src := media.CodecConfig{
			Codec:         name,
			SampleRate:    c.ClockRate,
			Channels:      1,
			BitDepth:      bitDepth,
			PayloadType:   c.PayloadType,
			FrameDuration: "20ms",
		}
		return neg, src, true
	}
}

func handlerG722(c sdp.Codec) (sdp.Codec, media.CodecConfig, bool) {
	neg := sdp.Codec{PayloadType: c.PayloadType, Name: "g722", ClockRate: 8000, Channels: 1}
	src := media.CodecConfig{
		Codec:         "g722",
		SampleRate:    16000,
		Channels:      1,
		BitDepth:      16,
		PayloadType:   c.PayloadType,
		FrameDuration: "20ms",
	}
	return neg, src, true
}

func handlerOpus(c sdp.Codec) (sdp.Codec, media.CodecConfig, bool) {
	decodeCh := c.Channels
	if decodeCh < 1 {
		decodeCh = 1
	}
	if decodeCh > 2 {
		decodeCh = 2
	}
	neg := sdp.Codec{PayloadType: c.PayloadType, Name: "opus", ClockRate: c.ClockRate, Channels: decodeCh}
	src := media.CodecConfig{
		Codec:              "opus",
		SampleRate:         c.ClockRate,
		Channels:           1,
		OpusDecodeChannels: decodeCh,
		BitDepth:           16,
		PayloadType:        c.PayloadType,
		FrameDuration:      "20ms",
	}
	return neg, src, true
}

// ---------- PCM bridge helpers (used by MediaLeg and tests) --------------

// InternalPCMSampleRate picks the MediaSession internal PCM bridge rate so decode/encode
// avoids unnecessary resampling (e.g. keep G.711 at 8 kHz, Opus at negotiated clock,
// G.722 at 16 kHz PCM per RFC 3551).
func InternalPCMSampleRate(src media.CodecConfig) int {
	name := strings.ToLower(strings.TrimSpace(src.Codec))
	switch name {
	case "pcmu", "pcma":
		if src.SampleRate > 0 {
			return src.SampleRate
		}
		return 8000
	case "g722":
		return 16000
	case "opus":
		switch src.SampleRate {
		case 8000, 12000, 16000, 24000, 48000:
			return src.SampleRate
		}
		if src.SampleRate > 36000 {
			return 48000
		}
		if src.SampleRate > 20000 {
			return 24000
		}
		if src.SampleRate > 14000 {
			return 16000
		}
		if src.SampleRate > 10000 {
			return 12000
		}
		if src.SampleRate > 0 {
			return 8000
		}
		return 48000
	default:
		if src.SampleRate > 0 {
			return src.SampleRate
		}
		return 16000
	}
}

func telephoneEventPT(offer []sdp.Codec, matchClock int) uint8 {
	c, ok := sdp.PickTelephoneEventFromOffer(offer, matchClock)
	if !ok {
		return 0
	}
	return c.PayloadType
}
