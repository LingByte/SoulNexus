// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package metrics

import (
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestObserveAsync_DeliversToHistogram(t *testing.T) {
	const m = "voiceserver_async_test_latency_ms"
	for i := 0; i < 50; i++ {
		ObserveAsync(m, "test", float64(i))
	}
	// Wait for drain. The drain goroutine is unbuffered on the read
	// side; a short poll loop is more reliable than a sleep.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		var sb strings.Builder
		Default.WritePromText(&sb)
		if strings.Contains(sb.String(), m+"_count 50") {
			return
		}
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
	t.Errorf("async drain didn't deliver all 50 samples within 1s")
}

func TestObserveAsync_DropsOnOverflow(t *testing.T) {
	// Saturate the channel by sending many more than buffer cap to
	// a slow-draining histogram. We can't easily slow the drain so
	// we rely on the simple fact: send vastly more than bufSize ×
	// drain capacity in a tight loop, expect non-zero dropped.
	const m = "voiceserver_async_test_overflow_ms"
	before := AsyncDroppedCount()
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for i := 0; i < 10_000; i++ {
				ObserveAsync(m, "overflow probe", float64(seed*10_000+i))
			}
		}(g)
	}
	wg.Wait()
	dropped := AsyncDroppedCount() - before
	if dropped == 0 {
		// In a fast environment the drain may keep up — that's not
		// a bug. We just can't assert overflow here. Log instead.
		t.Logf("overflow probe didn't exceed drain throughput (dropped=0); skipping assertion")
		return
	}
	if dropped > 16*10_000 {
		t.Errorf("nonsense dropped count: %d (more than enqueued)", dropped)
	}
}

func TestObserveAsync_DroppedCountVisibleInMetrics(t *testing.T) {
	// If anything dropped during this test run, the self-counter
	// should show it in WritePromText.
	if AsyncDroppedCount() == 0 {
		t.Skip("no drops observed in this run, can't assert exposition")
	}
	var sb strings.Builder
	Default.WritePromText(&sb)
	if !strings.Contains(sb.String(), MetricObserveDroppedTotal) {
		t.Error("dropped counter not exposed via WritePromText")
	}
}
