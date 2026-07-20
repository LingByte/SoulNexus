// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package websocket provides a minimal hub for workflow execution log streaming.
//
// Multi-replica: this hub is process-local. For HA, fan-out via Redis pub/sub
// (see docs/distributed.md). Broadcast drops when the buffer is full so request
// paths never block.
package websocket

import (
	"sync"
	"time"
)

// Message is a hub envelope used by workflow log streaming.
type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	From      string      `json:"from,omitempty"`
	To        string      `json:"to,omitempty"`
	Group     string      `json:"group,omitempty"`
}

// Handler receives drained broadcast messages (e.g. WS write pump).
type Handler func(msg *Message)

// Hub manages optional workflow log fan-out.
type Hub struct {
	broadcast chan *Message
	mu        sync.RWMutex
	handlers  []Handler
	started   bool
}

// NewHub creates a hub with a buffered broadcast channel.
func NewHub() *Hub {
	return &Hub{broadcast: make(chan *Message, 256)}
}

// Subscribe registers a consumer. Call Start once so messages are drained.
func (h *Hub) Subscribe(fn Handler) {
	if h == nil || fn == nil {
		return
	}
	h.mu.Lock()
	h.handlers = append(h.handlers, fn)
	h.mu.Unlock()
}

// Start begins draining the broadcast channel to subscribers (idempotent).
func (h *Hub) Start() {
	if h == nil {
		return
	}
	h.mu.Lock()
	if h.started {
		h.mu.Unlock()
		return
	}
	h.started = true
	if h.broadcast == nil {
		h.broadcast = make(chan *Message, 256)
	}
	ch := h.broadcast
	h.mu.Unlock()
	go func() {
		for msg := range ch {
			h.mu.RLock()
			hs := append([]Handler(nil), h.handlers...)
			h.mu.RUnlock()
			for _, fn := range hs {
				fn(msg)
			}
		}
	}()
}

// GetBroadcastChannel returns the broadcast send channel.
func (h *Hub) GetBroadcastChannel() chan<- *Message {
	if h == nil {
		ch := make(chan *Message)
		close(ch)
		return ch
	}
	if h.broadcast == nil {
		h.broadcast = make(chan *Message, 256)
	}
	return h.broadcast
}

// Broadcast enqueues a message without blocking callers when the buffer is full.
func (h *Hub) Broadcast(msgType string, data interface{}) {
	if h == nil {
		return
	}
	if h.broadcast == nil {
		h.broadcast = make(chan *Message, 256)
	}
	msg := &Message{Type: msgType, Data: data, Timestamp: time.Now().UnixMilli()}
	select {
	case h.broadcast <- msg:
	default:
		// Drop rather than block workflow / request paths.
	}
}
