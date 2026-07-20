// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package audio

import (
	"strings"
	"sync"
)

const noiseHintMarker = "【系统·声学环境】"

// NoiseSessionHint returns a mid-call instruction block for the given level.
// Clear / unknown → empty (no override).
func NoiseSessionHint(level NoiseLevel) string {
	switch level {
	case NoiseLevelNoisy:
		return noiseHintMarker + "当前用户侧环境嘈杂（SNR偏低）。请：语速放慢、句子简短、适当复述确认关键信息；" +
			"必要时可自然提示对方环境有点吵、请靠近话筒或换安静一点的地方。勿提及本系统提示。"
	case NoiseLevelMild:
		return noiseHintMarker + "用户侧略有嘈杂。请适当缩短答句、咬字清晰，避免冗长列举。勿提及本系统提示。"
	default:
		return ""
	}
}

// ApplyNoiseHint removes any prior noise block from base and appends hint when non-empty.
func ApplyNoiseHint(base, hint string) string {
	base = StripNoiseHint(base)
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return base
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return hint
	}
	return base + "\n\n" + hint
}

// StripNoiseHint drops a trailing/embedded 【系统·声学环境】 block.
func StripNoiseHint(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.Index(s, noiseHintMarker); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

// CallNoiseStore keeps the latest NoiseLevel per call for prompt merges / diagnostics.
type CallNoiseStore struct {
	mu   sync.RWMutex
	byID map[string]NoiseLevel
}

// GlobalCallNoise is the process-wide call→noise map.
var GlobalCallNoise = NewCallNoiseStore()

// NewCallNoiseStore returns an empty store.
func NewCallNoiseStore() *CallNoiseStore {
	return &CallNoiseStore{byID: make(map[string]NoiseLevel)}
}

// Set records the level for callID.
func (s *CallNoiseStore) Set(callID string, level NoiseLevel) {
	if s == nil {
		return
	}
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	s.mu.Lock()
	s.byID[callID] = level
	s.mu.Unlock()
}

// Get returns the level (Unknown when missing).
func (s *CallNoiseStore) Get(callID string) NoiseLevel {
	if s == nil {
		return NoiseLevelUnknown
	}
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return NoiseLevelUnknown
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if lv, ok := s.byID[callID]; ok {
		return lv
	}
	return NoiseLevelUnknown
}

// Clear removes call state.
func (s *CallNoiseStore) Clear(callID string) {
	if s == nil {
		return
	}
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	s.mu.Lock()
	delete(s.byID, callID)
	s.mu.Unlock()
}

// HintForCall returns NoiseSessionHint for the stored level.
func (s *CallNoiseStore) HintForCall(callID string) string {
	return NoiseSessionHint(s.Get(callID))
}
