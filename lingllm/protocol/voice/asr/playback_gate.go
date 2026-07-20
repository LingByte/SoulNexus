package asr

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"sync/atomic"
	"time"
)

// PlaybackGate tracks downlink TTS activity for echo suppression and barge-in.
// It treats queued utterances and a configurable post-playback tail as "active"
// so uplink echo does not leak into ASR right after the speaker goes quiet.
type PlaybackGate struct {
	isPlaying   func() bool
	queueDepth  func() int
	tail        time.Duration
	lastActiveN atomic.Int64 // unix nanos
}

// NewPlaybackGate creates a gate. tail is how long after playback ends uplink
// remains suppressed (room echo). 0 disables tail extension.
func NewPlaybackGate(isPlaying func() bool, queueDepth func() int, tail time.Duration) *PlaybackGate {
	return &PlaybackGate{
		isPlaying:  isPlaying,
		queueDepth: queueDepth,
		tail:       tail,
	}
}

// IsStreaming is true while audio frames are actively leaving the TTS pipeline.
func (g *PlaybackGate) IsStreaming() bool {
	if g == nil {
		return false
	}
	if g.isPlaying != nil && g.isPlaying() {
		g.markActive()
		return true
	}
	return false
}

// IsQueued is true when additional utterances wait on the speak queue.
func (g *PlaybackGate) IsQueued() bool {
	if g == nil || g.queueDepth == nil {
		return false
	}
	return g.queueDepth() > 0
}

// IsBargeInWindow is true when user interrupt should be considered: streaming,
// queued, or within the post-playback tail.
func (g *PlaybackGate) IsBargeInWindow() bool {
	if g == nil {
		return false
	}
	if g.IsStreaming() || g.IsQueued() {
		return true
	}
	return g.inTail()
}

// IsEchoSuppressActive is true when uplink should not be fed to ASR (echo tail
// included). Slightly longer than barge-in window when tail > 0.
func (g *PlaybackGate) IsEchoSuppressActive() bool {
	return g.IsBargeInWindow()
}

func (g *PlaybackGate) inTail() bool {
	if g.tail <= 0 {
		return false
	}
	last := g.lastActiveN.Load()
	if last == 0 {
		return false
	}
	return time.Since(time.Unix(0, last)) < g.tail
}

func (g *PlaybackGate) markActive() {
	g.lastActiveN.Store(time.Now().UnixNano())
}

// Reset clears tail memory (e.g. on session teardown).
func (g *PlaybackGate) Reset() {
	if g != nil {
		g.lastActiveN.Store(0)
	}
}
