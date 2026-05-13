// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

//go:build rnnoise

package rnnoise

/*
// Explicit -I/-L so builds work without rnnoise.pc (common on macOS after
// ./configure && make install). We omit /opt/local (MacPorts): passing a
// non-existent -L triggers ld warnings on typical Homebrew-only systems.
#cgo darwin CFLAGS: -I/usr/local/include -I/opt/homebrew/include
#cgo darwin LDFLAGS: -L/usr/local/lib -L/opt/homebrew/lib
#cgo linux CFLAGS: -I/usr/include -I/usr/local/include
#cgo linux LDFLAGS: -L/usr/lib -L/usr/local/lib
#cgo LDFLAGS: -lrnnoise

#include <rnnoise.h>
#include <stdlib.h>
*/
import "C"

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"unsafe"
)

// Denoiser wraps a librnnoise DenoiseState. RNNoise is single-threaded
// per state — callers MUST NOT share one Denoiser between concurrent
// goroutines. The pkg/voice/asr pipeline uses one per call which is
// the natural granularity (a single ASR feed runs sequentially).
type Denoiser struct {
	st *C.DenoiseState
}

// New creates a denoiser using librnnoise's default built-in model.
// Custom models (rnnoise_model_from_file) are deliberately not exposed —
// the default model handles the speech-vs-noise balance most callers
// want, and adding model loading raises ownership/lifetime questions
// (the model must outlive every state created from it) that would force
// a more complex API on every consumer.
func New() (*Denoiser, error) {
	st := C.rnnoise_create(nil)
	if st == nil {
		return nil, errors.New("rnnoise: rnnoise_create returned nil")
	}
	return &Denoiser{st: st}, nil
}

// Close releases the native state. Idempotent.
func (d *Denoiser) Close() {
	if d == nil || d.st == nil {
		return
	}
	C.rnnoise_destroy(d.st)
	d.st = nil
}

// FrameSamples returns rnnoise_get_frame_size() — typically 480 at 48 kHz.
// We expose the real C value (rather than hard-coding 480) so future
// librnnoise revs that change the frame size keep working.
func FrameSamples() int { return int(C.rnnoise_get_frame_size()) }

// FrameBytes is the PCM16 frame size: 2 * FrameSamples.
func FrameBytes() int { return FrameSamples() * 2 }

// ProcessPCM16LE denoises exactly one frame of little-endian int16 PCM
// at 48 kHz. The input must be FrameBytes() long; output is the same
// size. RNNoise expects float ∈ [-1,1] internally — we convert at the
// boundary and clip on the way out to guard against rare overshoots.
func (d *Denoiser) ProcessPCM16LE(frame []byte) ([]byte, error) {
	if d == nil || d.st == nil {
		return nil, errors.New("rnnoise: closed denoiser")
	}
	n := FrameSamples()
	want := n * 2
	if len(frame) != want {
		return nil, fmt.Errorf("rnnoise: want %d bytes per frame, got %d", want, len(frame))
	}

	in := make([]float32, n)
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		v := int16(binary.LittleEndian.Uint16(frame[i*2:]))
		in[i] = float32(v) / 32768.0
	}

	pIn := (*C.float)(unsafe.Pointer(&in[0]))
	pOut := (*C.float)(unsafe.Pointer(&out[0]))
	C.rnnoise_process_frame(d.st, pOut, pIn)

	outBytes := make([]byte, want)
	for i := 0; i < n; i++ {
		s := out[i]
		if s > 1 {
			s = 1
		} else if s < -1 {
			s = -1
		}
		v := int16(math.Round(float64(s) * 32767.0))
		binary.LittleEndian.PutUint16(outBytes[i*2:], uint16(v))
	}
	return outBytes, nil
}

// Process runs denoising over arbitrary 48 kHz PCM16 LE buffers.
// Whole frames are denoised; any trailing bytes shorter than one frame
// are copied through unchanged. Callers that need 100% sample coverage
// must buffer until they have a full frame before calling.
func (d *Denoiser) Process(pcm48k []byte) []byte {
	if d == nil || len(pcm48k) == 0 {
		return pcm48k
	}
	fb := FrameBytes()
	full := (len(pcm48k) / fb) * fb
	out := make([]byte, 0, len(pcm48k))
	for i := 0; i < full; i += fb {
		denoised, err := d.ProcessPCM16LE(pcm48k[i : i+fb])
		if err != nil {
			out = append(out, pcm48k[i:i+fb]...)
			continue
		}
		out = append(out, denoised...)
	}
	if full < len(pcm48k) {
		out = append(out, pcm48k[full:]...)
	}
	return out
}
