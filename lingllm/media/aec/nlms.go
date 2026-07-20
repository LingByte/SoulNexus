// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package aec implements far-end reference echo cancellation (NLMS).
// Unlike denoise "simple" amplitude cut, this uses playback PCM as a
// reference to cancel loudspeaker echo from the near-end (mic) stream.
package aec

import (
	"math"
	"sync"
)

// Canceller is a normalized LMS adaptive echo canceller for mono PCM16 LE.
type Canceller struct {
	mu       sync.Mutex
	filter   []float64
	farHist  []float64
	histIdx  int
	muStep   float64
	eps      float64
	frameCap int
}

// Config tunes filter length and adaptation.
type Config struct {
	SampleRate int
	// FilterMs is adaptive filter memory in milliseconds (default 64ms).
	FilterMs int
	// Mu is NLMS step size (default 0.4).
	Mu float64
}

// New builds an NLMS canceller. SampleRate must match near/far PCM.
func New(cfg Config) *Canceller {
	sr := cfg.SampleRate
	if sr <= 0 {
		sr = 16000
	}
	ms := cfg.FilterMs
	if ms <= 0 {
		ms = 64
	}
	n := sr * ms / 1000
	if n < 64 {
		n = 64
	}
	mu := cfg.Mu
	if mu <= 0 {
		mu = 0.4
	}
	return &Canceller{
		filter:   make([]float64, n),
		farHist:  make([]float64, n),
		muStep:   mu,
		eps:      1e-6,
		frameCap: sr / 25, // ~40ms safety
	}
}

// ProcessFar feeds far-end (playback / TTS) reference samples.
func (c *Canceller) ProcessFar(pcm []byte) {
	if c == nil || len(pcm) < 2 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	n := len(c.farHist)
	for i := 0; i+1 < len(pcm); i += 2 {
		s := float64(int16(pcm[i]) | int16(pcm[i+1])<<8)
		c.farHist[c.histIdx%n] = s
		c.histIdx++
	}
}

// ProcessNear cancels echo from near-end (mic) PCM and returns cleaned PCM16.
func (c *Canceller) ProcessNear(pcm []byte) []byte {
	if c == nil || len(pcm) < 2 {
		return pcm
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]byte, len(pcm)& ^1)
	n := len(c.filter)
	for i := 0; i+1 < len(pcm); i += 2 {
		near := float64(int16(pcm[i]) | int16(pcm[i+1])<<8)
		var y, power float64
		for j := 0; j < n; j++ {
			idx := (c.histIdx - 1 - j + n*1024) % n
			x := c.farHist[idx]
			y += c.filter[j] * x
			power += x * x
		}
		err := near - y
		norm := c.muStep / (power + c.eps)
		for j := 0; j < n; j++ {
			idx := (c.histIdx - 1 - j + n*1024) % n
			c.filter[j] += norm * err * c.farHist[idx]
		}
		if err > 32767 {
			err = 32767
		} else if err < -32768 {
			err = -32768
		}
		v := int16(math.Round(err))
		out[i] = byte(v)
		out[i+1] = byte(v >> 8)
	}
	return out
}

// Close releases resources (no-op for pure Go).
func (c *Canceller) Close() error { return nil }
