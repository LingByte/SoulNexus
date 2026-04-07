// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

//go:build !rnnoise

package rnnoise

// Denoiser is a stub when CGO is disabled.
type Denoiser struct{}

// New always fails when CGO is disabled.
func New() (*Denoiser, error) {
	return nil, ErrUnavailable
}

// Close is a no-op on the stub.
func (d *Denoiser) Close() {}

// FrameSamples matches the Xiph RNNoise default when the library is not linked.
func FrameSamples() int { return 480 }

// FrameBytes is 2 * FrameSamples().
func FrameBytes() int { return FrameSamples() * 2 }

// ProcessPCM16LE is not available on stub builds.
func (d *Denoiser) ProcessPCM16LE(_ []byte) ([]byte, error) {
	return nil, ErrUnavailable
}

// Process passes through on stub (no denoising).
func (d *Denoiser) Process(pcm []byte) []byte {
	return pcm
}
