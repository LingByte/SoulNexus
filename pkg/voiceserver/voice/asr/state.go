package asr

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"strings"
	"sync"
)

// IncrementalState extracts the *newly-appended* text from vendors that emit
// cumulative-running transcripts (QCloud ASR, Volcengine, Xunfei streaming).
//
// Typical flow for a single utterance:
//
//   partial  "你好"          → Update → returns "你好"
//   partial  "你好今天"       → Update → returns "今天"
//   partial  "你好今天天气"    → Update → returns "天气"
//   final    "你好今天天气不错" → Update(final=true) → returns "不错" (and resets)
//
// For vendors that already emit incremental deltas, just pass the delta —
// Update treats a shorter-or-equal text as a "no-op" safely.
type IncrementalState struct {
	mu         sync.Mutex
	lastText   string // last cumulative partial seen in this utterance
	lastFinal  string // last completed final utterance text
	utterances int
}

// NewIncrementalState returns a fresh state.
func NewIncrementalState() *IncrementalState {
	return &IncrementalState{}
}

// Update returns the new delta (may be empty). If isFinal is true, the state
// rolls over to the next utterance and Update returns the final-tail delta
// (usually just whatever is appended after the last partial).
func (s *IncrementalState) Update(text string, isFinal bool) string {
	text = strings.TrimSpace(text)
	if s == nil {
		return text
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var delta string
	switch {
	case text == "":
		delta = ""
	case strings.HasPrefix(text, s.lastText):
		delta = text[len(s.lastText):]
	default:
		// Vendor restarted or sent a different hypothesis — treat whole text
		// as new. This is conservative but avoids dropping content.
		delta = text
	}

	if isFinal {
		s.lastFinal = text
		s.lastText = ""
		s.utterances++
	} else {
		s.lastText = text
	}
	return strings.TrimSpace(delta)
}

// LastFinal returns the most recent completed utterance.
func (s *IncrementalState) LastFinal() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastFinal
}

// Reset clears state for a new call / session.
func (s *IncrementalState) Reset() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.lastText = ""
	s.lastFinal = ""
	s.utterances = 0
	s.mu.Unlock()
}

// Utterances returns how many finals have completed so far.
func (s *IncrementalState) Utterances() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.utterances
}
