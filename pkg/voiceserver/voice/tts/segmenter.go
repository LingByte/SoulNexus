package tts

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
	"sync"
	"time"
)

// SegmenterConfig tunes how an LLM token stream is sliced into TTS utterances.
//
// The rules, applied in order, emit a segment as soon as one matches:
//
//   1. buffer ends with one of SentenceEnders  → flush immediately
//   2. buffer ends with one of ClauseMarkers and len(buffer) >= MinRunes → flush
//   3. len(buffer) >= MaxRunes                  → flush (forced)
//   4. IdleFlush elapsed since last token and len(buffer) >= MinRunes → flush
//
// OnComplete forces a final flush regardless of length.
type SegmenterConfig struct {
	SentenceEnders []string
	ClauseMarkers  []string
	MinRunes       int
	MaxRunes       int
	IdleFlush      time.Duration
}

// DefaultSegmenterConfig returns a tuned config suited to Chinese + English
// spoken dialog. Callers can override any field.
func DefaultSegmenterConfig() SegmenterConfig {
	return SegmenterConfig{
		SentenceEnders: []string{"。", "！", "？", ".", "!", "?", "\n"},
		ClauseMarkers:  []string{"，", "、", "；", ";", ",", ":", "：", "—"},
		MinRunes:       12,
		MaxRunes:       40,
		IdleFlush:      60 * time.Millisecond,
	}
}

// Segmenter buffers incoming text (typically LLM tokens) and emits
// TTS-ready segments via the OnSegment callback.
//
// It is goroutine-safe; Push/Complete/Reset can be called from any goroutine.
type Segmenter struct {
	cfg SegmenterConfig

	mu        sync.Mutex
	buf       strings.Builder
	idleTimer *time.Timer

	onSegment func(segment string, final bool)
}

// NewSegmenter builds a segmenter with the given config. Missing fields fall
// back to DefaultSegmenterConfig values.
func NewSegmenter(cfg SegmenterConfig, onSegment func(string, bool)) *Segmenter {
	def := DefaultSegmenterConfig()
	if len(cfg.SentenceEnders) == 0 {
		cfg.SentenceEnders = def.SentenceEnders
	}
	if len(cfg.ClauseMarkers) == 0 {
		cfg.ClauseMarkers = def.ClauseMarkers
	}
	if cfg.MinRunes <= 0 {
		cfg.MinRunes = def.MinRunes
	}
	if cfg.MaxRunes <= 0 {
		cfg.MaxRunes = def.MaxRunes
	}
	if cfg.IdleFlush <= 0 {
		cfg.IdleFlush = def.IdleFlush
	}
	if onSegment == nil {
		onSegment = func(string, bool) {}
	}
	return &Segmenter{cfg: cfg, onSegment: onSegment}
}

// Push feeds a piece of text (one LLM token, one chunk, or any substring).
// It may trigger 0..N segment callbacks synchronously.
func (s *Segmenter) Push(piece string) {
	if s == nil || piece == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buf.WriteString(piece)
	cur := s.buf.String()

	// Rule 1: sentence ender.
	if endsWithAny(cur, s.cfg.SentenceEnders) {
		s.flushLocked(false)
		return
	}
	// Rule 2: clause marker + min length.
	if endsWithAny(cur, s.cfg.ClauseMarkers) && runeLen(cur) >= s.cfg.MinRunes {
		s.flushLocked(false)
		return
	}
	// Rule 3: hard max.
	if runeLen(cur) >= s.cfg.MaxRunes {
		s.flushLocked(false)
		return
	}
	// Rule 4: arm idle-flush timer.
	s.armIdleLocked()
}

// Complete forces a final flush (end of LLM stream / end of turn).
func (s *Segmenter) Complete() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
	s.flushLocked(true)
}

// Reset drops any buffered text without emitting.
func (s *Segmenter) Reset() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.buf.Reset()
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
	s.mu.Unlock()
}

// ----- internal -----

func (s *Segmenter) flushLocked(final bool) {
	text := strings.TrimSpace(s.buf.String())
	s.buf.Reset()
	if text == "" {
		return
	}
	s.onSegment(text, final)
}

func (s *Segmenter) armIdleLocked() {
	if s.idleTimer == nil {
		s.idleTimer = time.AfterFunc(s.cfg.IdleFlush, s.idleFire)
		return
	}
	s.idleTimer.Reset(s.cfg.IdleFlush)
}

func (s *Segmenter) idleFire() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if runeLen(s.buf.String()) >= s.cfg.MinRunes {
		s.flushLocked(false)
	}
}

func endsWithAny(s string, suf []string) bool {
	for _, x := range suf {
		if x == "" {
			continue
		}
		if strings.HasSuffix(s, x) {
			return true
		}
	}
	return false
}

func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
