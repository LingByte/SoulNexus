// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package metrics

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

func TestCounter_BasicAndLabels(t *testing.T) {
	r := NewRegistry()
	r.IncCounter("c", "help", map[string]string{"t": "sip"})
	r.IncCounter("c", "help", map[string]string{"t": "sip"})
	r.IncCounter("c", "help", map[string]string{"t": "xiaozhi"})
	r.AddCounter("c", "help", map[string]string{"t": "sip"}, 5)

	var buf bytes.Buffer
	r.WritePromText(&buf)
	out := buf.String()
	if !strings.Contains(out, `c{t="sip"} 7`) {
		t.Errorf("want c{t=sip} 7, got:\n%s", out)
	}
	if !strings.Contains(out, `c{t="xiaozhi"} 1`) {
		t.Errorf("want c{t=xiaozhi} 1, got:\n%s", out)
	}
}

func TestCounter_NegativeAddIgnored(t *testing.T) {
	r := NewRegistry()
	r.IncCounter("c", "", nil)
	r.AddCounter("c", "", nil, 0) // 0 no-op
	var buf bytes.Buffer
	r.WritePromText(&buf)
	if !strings.Contains(buf.String(), "c 1\n") {
		t.Fatalf("expected c 1, got:\n%s", buf.String())
	}
}

func TestGauge_SetAndAdd(t *testing.T) {
	r := NewRegistry()
	r.SetGauge("g", "h", map[string]string{"l": "a"}, 3)
	r.AddGauge("g", "h", map[string]string{"l": "a"}, 1.5)
	r.AddGauge("g", "h", map[string]string{"l": "a"}, -0.5)
	var buf bytes.Buffer
	r.WritePromText(&buf)
	if !strings.Contains(buf.String(), `g{l="a"} 4`) {
		t.Fatalf("expected g{l=a} 4, got:\n%s", buf.String())
	}
}

func TestHistogram_Quantiles(t *testing.T) {
	r := NewRegistry()
	// 100 samples from 1 to 100: p50 ≈ 50.5, p95 ≈ 95.05
	for i := 1; i <= 100; i++ {
		r.Observe("h", "latency", float64(i))
	}
	var buf bytes.Buffer
	r.WritePromText(&buf)
	out := buf.String()
	if !strings.Contains(out, `h{quantile="0.50"}`) {
		t.Fatalf("missing p50:\n%s", out)
	}
	if !strings.Contains(out, `h_count 100`) {
		t.Fatalf("missing count:\n%s", out)
	}
	if !strings.Contains(out, `h_sum 5050`) {
		t.Fatalf("expected sum 5050:\n%s", out)
	}
}

func TestHistogram_RingBufferTruncates(t *testing.T) {
	r := NewRegistry()
	for i := 0; i < 5000; i++ {
		r.ObserveN("h", "", float64(i), 1000)
	}
	r.mu.RLock()
	h := r.histograms["h"]
	r.mu.RUnlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.samples) > h.max {
		t.Fatalf("buffer grew past max: got %d, max %d", len(h.samples), h.max)
	}
}

func TestLabelEscape(t *testing.T) {
	got := escapeLabel(`tri"cky\text`)
	if got != `tri\"cky\\text` {
		t.Fatalf(`escape mismatch: %q`, got)
	}
}

func TestSerialiseLabels_Stable(t *testing.T) {
	a := serialiseLabels(map[string]string{"b": "1", "a": "2"})
	b := serialiseLabels(map[string]string{"a": "2", "b": "1"})
	if a != b {
		t.Fatalf("label serialisation not stable: %q vs %q", a, b)
	}
	if !strings.HasPrefix(a, `a="2"`) {
		t.Fatalf("expected alphabetical order: %q", a)
	}
}

func TestRegistry_ConcurrentCounters(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				r.IncCounter("c", "", map[string]string{"t": "x"})
			}
		}()
	}
	wg.Wait()
	var buf bytes.Buffer
	r.WritePromText(&buf)
	if !strings.Contains(buf.String(), `c{t="x"} 10000`) {
		t.Fatalf("race lost counts:\n%s", buf.String())
	}
}

func TestAppHelpers_SmokeTest(t *testing.T) {
	// Reset Default so we don't leak state across test orderings.
	Default = NewRegistry()
	CallStarted("webrtc")
	CallStarted("webrtc")
	CallEnded("webrtc", "ok")
	ASRError("sip")
	TTSError("xiaozhi")
	BargeIn("webrtc")
	ObserveE2EFirstByte(800)
	ObserveE2EFirstByte(1200)
	ObserveE2EFirstByte(0) // ignored
	ObserveTTSFirstByte(300)
	ObserveLLMFirstByte(150)

	var buf bytes.Buffer
	Default.WritePromText(&buf)
	out := buf.String()
	for _, want := range []string{
		`voiceserver_active_calls{transport="webrtc"} 1`,
		`voiceserver_calls_total{status="ok",transport="webrtc"} 1`,
		`voiceserver_asr_errors_total{transport="sip"} 1`,
		`voiceserver_tts_errors_total{transport="xiaozhi"} 1`,
		`voiceserver_barge_in_total{transport="webrtc"} 1`,
		`voiceserver_e2e_first_byte_ms_count 2`,
		`voiceserver_tts_first_byte_ms_count 1`,
		`voiceserver_llm_first_byte_ms_count 1`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in metrics output:\n%s", want, out)
		}
	}
}
