// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package encoder

// PCM scratch-buffer pools shared across all codec encoders / decoders.
//
// Why this exists: Opus encode + decode runs at 50 frames/sec per
// direction per call. Each frame previously allocated 3-5 fresh slices
// (pcmBuffer, opusScratch, frame, raw, stereo). At 100 concurrent
// calls that is ~30K allocations / sec just for codec scratch space —
// noticeable both in GC pause time and in cache locality.
//
// The pools below give back to the heap once a buffer is done. The
// only constraint is that pooled buffers must NOT escape into the
// returned MediaPacket — only buffers that the caller is guaranteed to
// drop after the encode call returns may be pooled. The escaping
// outputs (decodedData, payload) are still freshly allocated per call.
//
// We key by exact len-or-greater so callers always get a buffer at
// least as large as they asked for. Smaller pooled buffers in the
// pool's free list are skipped (re-pooled) on demand. We don't
// bucket-by-power-of-two because Opus frame sizes are predictable
// (frameSize * channels for encode, fixed maxSamplesPerCh for decode)
// — the same closure asks for the same size every call, so a single
// pooled buffer per closure-arity is hit 99% of the time.

import "sync"

// pcm16Pool is the sync.Pool for []int16 scratch buffers used during
// Opus encode/decode. Stored value is *[]int16 to avoid the boxing
// allocation a plain []int16 entry would incur (sync.Pool stores any).
var pcm16Pool = sync.Pool{
	New: func() any {
		// Sized for one 20 ms Opus frame at 48 kHz stereo (1920 samples).
		// Most callers ask for ≤ this, so we hit the pool path; larger
		// requesters get a fresh make() via getInt16Slice.
		buf := make([]int16, 0, 1920)
		return &buf
	},
}

// byteScratchPool feeds the small []byte scratch areas Opus encoders
// use to receive packetised output (RFC 6716 caps a single packet at
// ~1275 bytes; we round up to 4 KB so even 120 ms super-frames fit).
var byteScratchPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 4000)
		return &buf
	},
}

// getInt16Slice borrows a []int16 of length n from the pool. The
// returned slice has len == n; the underlying capacity may be larger,
// but callers should treat the contents as undefined and overwrite
// every position they intend to read. The slice MUST be returned via
// putInt16Slice once it is no longer needed AND has not escaped into
// any retained data structure.
func getInt16Slice(n int) []int16 {
	bp := pcm16Pool.Get().(*[]int16)
	if cap(*bp) >= n {
		*bp = (*bp)[:n]
		return *bp
	}
	// Existing buffer too small — drop it (back to GC) and allocate
	// fresh. The new buffer will be put back by the caller and from
	// then on this size class is cached.
	*bp = make([]int16, n)
	return *bp
}

// putInt16Slice returns a slice borrowed via getInt16Slice. nil-safe.
func putInt16Slice(buf []int16) {
	if cap(buf) == 0 {
		return
	}
	// Reset length to 0; cap is preserved so the next caller's
	// getInt16Slice(n) sees the full backing array.
	buf = buf[:0]
	pcm16Pool.Put(&buf)
}

// getByteScratch borrows a []byte at least n bytes long for Opus
// encode output. Same lifetime rules as getInt16Slice.
func getByteScratch(n int) []byte {
	bp := byteScratchPool.Get().(*[]byte)
	if cap(*bp) >= n {
		*bp = (*bp)[:n]
		return *bp
	}
	*bp = make([]byte, n)
	return *bp
}

// putByteScratch returns a buffer borrowed via getByteScratch.
func putByteScratch(buf []byte) {
	if cap(buf) == 0 {
		return
	}
	buf = buf[:0]
	byteScratchPool.Put(&buf)
}
