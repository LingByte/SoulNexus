// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package audio

import (
	"strings"
	"sync"
)

// CallNoiseListener receives hysteretic noise-level changes for one call.
type CallNoiseListener func(level NoiseLevel, snrDB float64)

var (
	callNoiseListenerMu sync.RWMutex
	callNoiseListeners  = map[string]CallNoiseListener{}
)

// SetCallNoiseListener registers a per-call adapter (e.g. realtime UpdateInstructions).
// Pass nil to clear.
func SetCallNoiseListener(callID string, fn CallNoiseListener) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callNoiseListenerMu.Lock()
	defer callNoiseListenerMu.Unlock()
	if fn == nil {
		delete(callNoiseListeners, callID)
		return
	}
	callNoiseListeners[callID] = fn
}

func notifyCallNoiseListeners(callID string, level NoiseLevel, snrDB float64) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	callNoiseListenerMu.RLock()
	fn := callNoiseListeners[callID]
	callNoiseListenerMu.RUnlock()
	if fn != nil {
		fn(level, snrDB)
	}
}

func clearCallNoiseListeners(callID string) {
	SetCallNoiseListener(callID, nil)
}
