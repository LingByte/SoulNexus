// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package sfu

import (
	"testing"

	"github.com/pion/rtp"
	pionwebrtc "github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

func TestNewSimulcastForwarder(t *testing.T) {
	fwd, err := NewSimulcastForwarder("track-1", "peer-1", "audio",
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		zap.NewNop())
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if fwd.Local() == nil {
		t.Fatal("local track nil")
	}
	if active, _ := fwd.active.Load().(string); active != "" {
		t.Errorf("initial active = %q, want empty", active)
	}
}

func TestSimulcastForwarderSetMuted(t *testing.T) {
	fwd, _ := NewSimulcastForwarder("t", "p", "audio",
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		zap.NewNop())
	if fwd.Muted() {
		t.Error("default Muted=false")
	}
	fwd.SetMuted(true)
	if !fwd.Muted() {
		t.Error("SetMuted(true) didn't stick")
	}
	fwd.SetMuted(false)
	if fwd.Muted() {
		t.Error("SetMuted(false) didn't reset")
	}
}

func TestSimulcastForwarderSetPayloadSink(t *testing.T) {
	fwd, err := NewSimulcastForwarder("t", "p", "audio",
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	called := false
	fwd.SetPayloadSink(func(payload []byte) { called = true })
	// Manually invoke since we have no real remote
	if fwd.payloadSink == nil {
		t.Fatal("sink not stored")
	}
	fwd.payloadSink([]byte{0})
	if !called {
		t.Error("sink not called")
	}
}

func TestSimulcastForwarderDetachEmpty(t *testing.T) {
	fwd, err := NewSimulcastForwarder("t", "p", "video",
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeVP8, ClockRate: 90000},
		zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	// Detach when no layers — must not panic and active stays empty.
	fwd.Detach("nope")
	if active, _ := fwd.active.Load().(string); active != "" {
		t.Errorf("active after empty detach = %q", active)
	}
}

func TestSimulcastRebalanceAudioImmediate(t *testing.T) {
	fwd, err := NewSimulcastForwarder("t", "p", "audio",
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	// Inject a fake "" layer and verify rebalance picks it (audio path
	// has no keyframe requirement).
	fwd.mu.Lock()
	fwd.layers[""] = &layer{rid: ""}
	fwd.rebalanceLocked()
	fwd.mu.Unlock()
	if active, _ := fwd.active.Load().(string); active != "" {
		t.Errorf("expected empty-RID active, got %q", active)
	}

	// Add an "f" layer; audio doesn't require keyframe so it should be
	// promoted on rebalance.
	fwd.mu.Lock()
	fwd.layers["f"] = &layer{rid: "f"}
	fwd.rebalanceLocked()
	fwd.mu.Unlock()
	if active, _ := fwd.active.Load().(string); active != "f" {
		t.Errorf("expected f active, got %q", active)
	}
}

func TestSimulcastRebalanceVideoKeyframeGated(t *testing.T) {
	fwd, _ := NewSimulcastForwarder("t", "p", "video",
		pionwebrtc.RTPCodecCapability{MimeType: pionwebrtc.MimeTypeVP8, ClockRate: 90000},
		zap.NewNop())
	fwd.mu.Lock()
	fwd.layers["f"] = &layer{rid: "f"} // no keyframe yet
	fwd.rebalanceLocked()
	fwd.mu.Unlock()
	if active, _ := fwd.active.Load().(string); active != "" {
		t.Errorf("video w/o keyframe should NOT be promoted, got %q", active)
	}

	// Mark keyframe; rebalance should pick "f".
	fwd.mu.Lock()
	fwd.layers["f"].seenKeyframe.Store(true)
	fwd.rebalanceLocked()
	fwd.mu.Unlock()
	if active, _ := fwd.active.Load().(string); active != "f" {
		t.Errorf("video keyframe seen → active=f, got %q", active)
	}
}

func TestIsVP8Keyframe(t *testing.T) {
	// Empty payload.
	if isVP8Keyframe(&rtp.Packet{Payload: nil}) {
		t.Error("empty payload should not be keyframe")
	}
	// Minimal non-extended descriptor + keyframe (P-bit=0).
	pkt := &rtp.Packet{Payload: []byte{0x10 /*desc, X=0*/, 0x00 /*payload header, P=0*/, 0x9d, 0x01, 0x2a}}
	if !isVP8Keyframe(pkt) {
		t.Error("expected keyframe true for P-bit=0 payload")
	}
	// P-bit=1 → interframe.
	pkt2 := &rtp.Packet{Payload: []byte{0x10, 0x01}}
	if isVP8Keyframe(pkt2) {
		t.Error("P-bit=1 should be interframe")
	}
	// Extended descriptor with PictureID 1 byte.
	pkt3 := &rtp.Packet{Payload: []byte{
		0x80,       // desc byte0: X=1
		0x80,       // desc byte1: I=1
		0x42,       // PID 1 byte (high bit=0)
		0x00,       // VP8 payload header: P=0
		0x9d, 0x01,
	}}
	if !isVP8Keyframe(pkt3) {
		t.Error("extended descriptor with PID, P=0 should be keyframe")
	}
}
