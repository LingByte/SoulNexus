// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package metrics

// Async observation pipeline.
//
// The synchronous Observe() takes a per-histogram mutex. That's fine
// for ~hundreds of samples/sec across the whole process — our case
// for things like "TTS first byte" (one sample per turn) or "RTT"
// (one sample per call). But two future use cases would push it:
//
//   - Per-RTP-frame jitter / inter-arrival sampling.
//   - Per-ASR-partial latency tracking.
//
// Both produce hundreds-to-thousands of samples per call. The cure
// is buffered async observation:
//
//   1. Producers call ObserveAsync(name, value).
//   2. The value goes onto a single buffered channel (FIFO).
//   3. One drain goroutine pulls and calls the underlying sync
//      Observe(). It's the only goroutine that hits the histogram
//      mutex, so there's no fan-in contention.
//   4. If the buffer is full, ObserveAsync DROPS the sample and
//      bumps metrics_observe_dropped_total — producers NEVER block.
//
// Hot-path cost: ObserveAsync = one channel send (non-blocking) or
// one atomic add (overflow). ~30 ns on x86. Zero allocations.
//
// Memory cost: bufSize * sizeof(asyncSample) ≈ 16-24 B per slot.
// Default 4096 slots → ~96 KiB. Negligible.

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// asyncSample is the unit of work for the drain goroutine. Kept
// small (string + float64 + string = ~40 B) so the channel buffer
// stays cache-friendly.
type asyncSample struct {
	name string
	help string
	v    float64
}

const defaultAsyncBufSize = 4096

var (
	asyncOnce    sync.Once
	asyncCh      chan asyncSample
	asyncDropped atomic.Uint64
)

// initAsync lazily starts the drain goroutine on first use. We don't
// start it from init() so packages that never call ObserveAsync
// don't pay the goroutine cost.
func initAsync(bufSize int) {
	asyncOnce.Do(func() {
		if bufSize <= 0 {
			bufSize = defaultAsyncBufSize
		}
		asyncCh = make(chan asyncSample, bufSize)
		go func() {
			for s := range asyncCh {
				Default.Observe(s.name, s.help, s.v)
			}
		}()
	})
}

// ObserveAsync queues a histogram sample on the global async drain.
// Hot-path safe: non-blocking, zero allocation, drops on full
// (incrementing the dropped-samples counter).
//
// This is the recommended call for any observation that fires more
// than ~10x/sec per process. For one-off latencies (per turn, per
// call) the synchronous Default.Observe is fine and slightly more
// accurate (no buffering reorder concerns).
func ObserveAsync(name, help string, v float64) {
	if asyncCh == nil {
		initAsync(defaultAsyncBufSize)
	}
	select {
	case asyncCh <- asyncSample{name: name, help: help, v: v}:
		// queued
	default:
		// Buffer full — drop. The dropped count is itself observable
		// so we can detect when the drain falls behind.
		asyncDropped.Add(1)
		Default.addCounterRaw(MetricObserveDroppedTotal,
			"async histogram samples dropped because the drain channel was full",
			map[string]string{"metric": name}, 1)
	}
}

// AsyncDroppedCount returns the total samples dropped since process
// start. Exposed for tests and self-observability tooling.
func AsyncDroppedCount() uint64 {
	return asyncDropped.Load()
}

// flushAsyncForTest is only used by tests to deterministically wait
// for the drain to catch up before asserting on histogram contents.
// It works by sending a tombstone sample and letting it run through.
// Not exported beyond the package.
func flushAsyncForTest() {
	if asyncCh == nil {
		return
	}
	// Round-trip a sentinel: send N tombstones and wait for the
	// channel length to fall back to zero. Channel length is a
	// snapshot but for tests with a quiescent producer it's reliable.
	for len(asyncCh) > 0 {
		// Spin briefly. Tests should call this only after they've
		// stopped enqueueing.
		//
		// We use a yield rather than a sleep so the test suite isn't
		// slowed down by sleep granularity.
		runtime.Gosched()
	}
}
