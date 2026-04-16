// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu

import (
	"slices"
	"time"
)

// VideoLayer identifies a simulcast/SVC layer target for subscription or publish caps.
type VideoLayer int

const (
	LayerLow VideoLayer = iota
	LayerMid
	LayerHigh
)

// LastN returns up to n participant IDs from candidates.
// Candidates should be ordered by priority (e.g. active speaker score descending).
func LastN(candidates []UserID, n int) []UserID {
	if n <= 0 || len(candidates) == 0 {
		return nil
	}
	if len(candidates) <= n {
		cp := slices.Clone(candidates)
		return cp
	}
	return slices.Clone(candidates[:n])
}

// LayerSelector applies hysteresis when moving between simulcast layers.
type LayerSelector struct {
	Current       VideoLayer
	MinHold       time.Duration
	lastSwitch    time.Time
	lowBelowBps   int
	midBelowBps   int
	// High is implied: >= midBelowBps
}

// NewLayerSelector builds a selector with default thresholds (tunable).
// Below lowBelowBps -> LayerLow; below midBelowBps -> LayerMid; else LayerHigh.
func NewLayerSelector(minHold time.Duration, lowBelowBps, midBelowBps int) *LayerSelector {
	if lowBelowBps <= 0 {
		lowBelowBps = 250_000
	}
	if midBelowBps <= 0 {
		midBelowBps = 900_000
	}
	if minHold <= 0 {
		minHold = 2 * time.Second
	}
	return &LayerSelector{
		Current:     LayerHigh,
		MinHold:     minHold,
		lowBelowBps: lowBelowBps,
		midBelowBps: midBelowBps,
	}
}

func (s *LayerSelector) targetLayer(availableDownlinkBps int) VideoLayer {
	if availableDownlinkBps < s.lowBelowBps {
		return LayerLow
	}
	if availableDownlinkBps < s.midBelowBps {
		return LayerMid
	}
	return LayerHigh
}

// Update returns the layer to use after applying hold time to reduce oscillation.
func (s *LayerSelector) Update(now time.Time, availableDownlinkBps int) VideoLayer {
	want := s.targetLayer(availableDownlinkBps)
	if want == s.Current {
		if s.lastSwitch.IsZero() {
			s.lastSwitch = now
		}
		return s.Current
	}
	if s.lastSwitch.IsZero() {
		s.Current = want
		s.lastSwitch = now
		return s.Current
	}
	if now.Sub(s.lastSwitch) < s.MinHold {
		return s.Current
	}
	s.Current = want
	s.lastSwitch = now
	return s.Current
}
