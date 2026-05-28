// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package sessionctx holds per-call dialog payload used for agent-aware TTS.
package sessionctx

import (
	"encoding/json"
	"strings"
	"sync"
)

// DialogPayload is the JSON object browsers and hardware pass as ?payload=
// or WebRTC offer.payload.
type DialogPayload struct {
	APIKey       string `json:"apiKey"`
	APISecret    string `json:"apiSecret"`
	AgentID      string `json:"agentId"`
	Speaker      string `json:"speaker"`
	Language     string `json:"language"`
	TtsProvider  string `json:"ttsProvider"`
	VoiceCloneID *int   `json:"voiceCloneId"`
}

// ParseDialogPayload unmarshals payload JSON. Returns nil when empty or invalid.
func ParseDialogPayload(raw []byte) *DialogPayload {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var p DialogPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil
	}
	if strings.TrimSpace(p.APIKey) == "" || strings.TrimSpace(p.APISecret) == "" || strings.TrimSpace(p.AgentID) == "" {
		return nil
	}
	return &p
}

// TTSSpecRegistry stores per-call dialog payload for agent-aware TTS.
type TTSSpecRegistry struct {
	mu   sync.RWMutex
	spec map[string]*DialogPayload
}

func NewTTSSpecRegistry() *TTSSpecRegistry {
	return &TTSSpecRegistry{spec: make(map[string]*DialogPayload)}
}

func (r *TTSSpecRegistry) Put(callID string, p *DialogPayload) {
	if r == nil || strings.TrimSpace(callID) == "" || p == nil {
		return
	}
	r.mu.Lock()
	r.spec[callID] = p
	r.mu.Unlock()
}

func (r *TTSSpecRegistry) Get(callID string) *DialogPayload {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.spec[callID]
}

func (r *TTSSpecRegistry) Delete(callID string) {
	if r == nil || strings.TrimSpace(callID) == "" {
		return
	}
	r.mu.Lock()
	delete(r.spec, callID)
	r.mu.Unlock()
}

// DefaultRegistry is populated by xiaozhi/webrtc handlers when DB-backed TTS is enabled.
var DefaultRegistry = NewTTSSpecRegistry()
