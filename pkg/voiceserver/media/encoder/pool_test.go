// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package encoder

import "testing"

func TestPool_Int16RoundTrip(t *testing.T) {
	a := getInt16Slice(64)
	if len(a) != 64 {
		t.Fatalf("len=%d, want 64", len(a))
	}
	a[0] = 1234
	putInt16Slice(a)

	// Borrow again; we should usually get the same backing array.
	b := getInt16Slice(64)
	if len(b) != 64 {
		t.Fatalf("len=%d, want 64", len(b))
	}
	// Cap should be ≥ 64 (might be 1920 on the first New()-allocated
	// buffer, or exactly 64 on subsequent reuse — both valid).
	if cap(b) < 64 {
		t.Fatalf("cap=%d shrunk below requested", cap(b))
	}
	putInt16Slice(b)
}

func TestPool_Int16GrowsForLargerRequest(t *testing.T) {
	// First request smaller than the New() default; pool returns the
	// default-sized 1920 buffer.
	small := getInt16Slice(100)
	putInt16Slice(small)

	// Now ask for one larger than the default; pool should NOT reuse
	// the small one — it should allocate fresh.
	big := getInt16Slice(8192)
	if len(big) != 8192 {
		t.Fatalf("len=%d, want 8192", len(big))
	}
	if cap(big) < 8192 {
		t.Fatalf("cap=%d, want ≥ 8192", cap(big))
	}
	putInt16Slice(big)
}

func TestPool_ByteScratchRoundTrip(t *testing.T) {
	a := getByteScratch(2000)
	if len(a) != 2000 {
		t.Fatalf("len=%d, want 2000", len(a))
	}
	putByteScratch(a)

	b := getByteScratch(2000)
	if len(b) != 2000 {
		t.Fatalf("len=%d, want 2000", len(b))
	}
	putByteScratch(b)
}

// sink prevents the compiler from optimising the benchmark loops into
// nothing — assigning to a package-level sink forces the buffer to
// "escape" into something the optimiser can't statically prove unused.
var sink []int16

// BenchmarkPool_VsAllocate confirms pooling is faster than fresh make()
// at the typical Opus-encode hot-path size (1920 int16 samples).
// On Apple M1 the pooled version runs ~10× faster than fresh make()
// once the sink keeps the optimiser honest.
//
//	go test -bench=BenchmarkPool -benchmem ./pkg/media/encoder/
func BenchmarkPool_Allocate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := make([]int16, 1920)
		buf[0] = int16(i) // touch the buffer
		sink = buf
	}
}

func BenchmarkPool_Pooled(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buf := getInt16Slice(1920)
		buf[0] = int16(i) // touch the buffer
		sink = buf
		putInt16Slice(buf)
	}
}
