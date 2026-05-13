// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rnnoise

import (
	"errors"
	"testing"
)

// On stub builds (no -tags rnnoise) New() must yield ErrUnavailable so
// callers can `errors.Is` it cleanly. On real builds this test is
// skipped — the cgo file's New() succeeds, which is its own happy
// path and tested implicitly by anything using the denoiser.
func TestNew_StubReturnsUnavailable(t *testing.T) {
	d, err := New()
	if err == nil {
		// Real CGO build — denoiser actually constructed. Close it
		// and skip the rest; this branch isn't a stub assertion.
		if d != nil {
			d.Close()
		}
		t.Skip("rnnoise build tag enabled — skipping stub assertion")
	}
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}

// FrameSamples / FrameBytes must be sane on every build so callers can
// pre-size buffers without branching on the tag.
func TestFrameSizesAreReasonable(t *testing.T) {
	if FrameSamples() <= 0 {
		t.Fatalf("FrameSamples()=%d", FrameSamples())
	}
	if FrameBytes() != FrameSamples()*2 {
		t.Fatalf("FrameBytes()=%d, expected %d", FrameBytes(), FrameSamples()*2)
	}
}

// Process is a passthrough on stub. On real builds it denoises whole
// frames and passes through any remainder — we only assert length
// invariants here so the test works under both tags.
func TestProcess_PreservesLength(t *testing.T) {
	d, err := New()
	if err != nil {
		// Stub: Process is a method on the zero Denoiser anyway.
		var z Denoiser
		in := make([]byte, 1234)
		out := z.Process(in)
		if len(out) != len(in) {
			t.Fatalf("stub Process changed length: %d → %d", len(in), len(out))
		}
		return
	}
	defer d.Close()
	in := make([]byte, FrameBytes()*3+7) // 3 full frames + 7 byte tail
	out := d.Process(in)
	if len(out) != len(in) {
		t.Fatalf("Process changed length: %d → %d", len(in), len(out))
	}
}
