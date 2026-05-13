// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

//go:build !rnnoise

package rnnoise

// Denoiser is a stub when the rnnoise build tag is off. The real
// implementation lives in denoiser_cgo.go and links librnnoise.
//
// All methods are safe to call on the stub — Process is a passthrough
// so calling code can be denoise-aware without forcing CGO/librnnoise
// onto every developer. New() returns ErrUnavailable so production
// deployments can detect the missing dependency at startup and decide
// whether to hard-fail or proceed without denoising.
type Denoiser struct{}

// New always fails on stub builds.
func New() (*Denoiser, error) { return nil, ErrUnavailable }

// Close is a no-op on the stub.
func (d *Denoiser) Close() {}

// FrameSamples matches the Xiph RNNoise default (480 samples @ 48 kHz)
// even on the stub so callers sizing buffers don't branch on build tags.
func FrameSamples() int { return 480 }

// FrameBytes is FrameSamples * 2 (PCM16).
func FrameBytes() int { return FrameSamples() * 2 }

// ProcessPCM16LE on the stub returns ErrUnavailable.
func (d *Denoiser) ProcessPCM16LE(_ []byte) ([]byte, error) {
	return nil, ErrUnavailable
}

// Process is a passthrough on the stub — caller gets the same audio back.
func (d *Denoiser) Process(pcm []byte) []byte { return pcm }
