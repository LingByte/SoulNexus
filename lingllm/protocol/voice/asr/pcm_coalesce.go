package asr

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"
	"sync"
)

// PCMCoalesceComponent buffers small uplink chunks into recognizer-friendly frames.
// Default target is 20 ms of PCM16 mono at the configured sample rate.
type PCMCoalesceComponent struct {
	mu       sync.Mutex
	minBytes int
	buf      []byte
	flushFn  func([]byte) error
}

// NewPCMCoalesceComponent creates a coalescer. sampleRateHz is the PCM rate
// after decode (e.g. 16000). minMs is the minimum buffer duration (default 20).
func NewPCMCoalesceComponent(sampleRateHz, minMs int) *PCMCoalesceComponent {
	if sampleRateHz <= 0 {
		sampleRateHz = 16000
	}
	if minMs <= 0 {
		minMs = 20
	}
	minBytes := (sampleRateHz * minMs / 1000) * 2
	if minBytes < 2 {
		minBytes = 320
	}
	return &PCMCoalesceComponent{
		minBytes: minBytes,
		buf:      make([]byte, 0, minBytes*4),
	}
}

// Name returns the component identifier.
func (c *PCMCoalesceComponent) Name() string { return "pcm_coalesce" }

// Process accumulates PCM; emits when minBytes reached.
func (c *PCMCoalesceComponent) Process(ctx context.Context, data interface{}) (interface{}, bool, error) {
	pcm, ok := data.([]byte)
	if !ok {
		return nil, false, fmt.Errorf("%w: expected []byte, got %T", ErrInvalidDataType, data)
	}
	if len(pcm) == 0 {
		return nil, true, nil
	}

	c.mu.Lock()
	c.buf = append(c.buf, pcm...)
	out := c.takeReadyLocked()
	c.mu.Unlock()

	if len(out) > 0 {
		return out, true, nil
	}
	// Hold until minBytes; stop the chain for this uplink chunk.
	return nil, false, nil
}

func (c *PCMCoalesceComponent) takeReadyLocked() []byte {
	if len(c.buf) < c.minBytes {
		return nil
	}
	out := make([]byte, len(c.buf))
	copy(out, c.buf)
	c.buf = c.buf[:0]
	return out
}

// Flush emits any remaining buffered PCM (call on utterance end / session close).
func (c *PCMCoalesceComponent) Flush() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.buf) == 0 {
		return nil
	}
	out := make([]byte, len(c.buf))
	copy(out, c.buf)
	c.buf = c.buf[:0]
	return out
}
