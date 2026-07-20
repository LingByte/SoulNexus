package metrics

import (
	"math"
	"sort"
)

// CounterTotal sums all label variants of a counter.
func (r *Registry) CounterTotal(name string) uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.counters[name]
	if !ok {
		return 0
	}
	var n uint64
	for _, v := range c.labels {
		n += v
	}
	return n
}

// CounterWithLabels returns the counter value for an exact label set.
func (r *Registry) CounterWithLabels(name string, labels map[string]string) uint64 {
	key := serialiseLabels(labels)
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.counters[name]
	if !ok {
		return 0
	}
	return c.labels[key]
}

// GaugeTotal sums all label variants of a gauge.
func (r *Registry) GaugeTotal(name string) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	g, ok := r.gauges[name]
	if !ok {
		return 0
	}
	var sum float64
	g.labels.Range(func(_, val any) bool {
		sum += math.Float64frombits(val.(*atomicGaugeVal).bits.Load())
		return true
	})
	return sum
}

// SummaryQuantiles returns P50/P90/P99 for a summary metric.
func (r *Registry) SummaryQuantiles(name string) map[string]float64 {
	r.mu.RLock()
	h, ok := r.histograms[name]
	r.mu.RUnlock()
	if !ok {
		return nil
	}
	h.mu.Lock()
	samples := append([]float64(nil), h.samples...)
	h.mu.Unlock()
	sort.Float64s(samples)
	n := len(samples)
	if n == 0 {
		return nil
	}
	q := func(p float64) float64 {
		idx := int(math.Ceil(p*float64(n))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= n {
			idx = n - 1
		}
		return samples[idx]
	}
	return map[string]float64{
		"p50": q(0.50),
		"p90": q(0.90),
		"p99": q(0.99),
		"n":   float64(n),
	}
}

// RegistrySnapshot is a JSON-friendly metrics dump for /system/status.
type RegistrySnapshot struct {
	Counters  map[string]uint64            `json:"counters"`
	Gauges    map[string]float64           `json:"gauges"`
	Summaries map[string]map[string]float64  `json:"summaries"`
}

// Snapshot exports selected prometheus-style series for dashboards.
func (r *Registry) Snapshot() RegistrySnapshot {
	names := []string{
		MetricActiveCalls, MetricCallsTotal, MetricASRErrors, MetricTTSErrors,
		MetricBargeInTotal, MetricDialogReconnectTotal,
		MetricE2EFirstByteMs, MetricTTSFirstByteMs, MetricLLMFirstByteMs,
		"http_requests_total", "http_request_errors_total",
		"go_goroutines", "go_threads", "go_memstats_heap_inuse_bytes", "go_memstats_gc_cpu_fraction",
		"process_resident_memory_bytes", "process_open_fds", "process_max_fds",
	}
	out := RegistrySnapshot{
		Counters:  make(map[string]uint64),
		Gauges:    make(map[string]float64),
		Summaries: make(map[string]map[string]float64),
	}
	for _, name := range names {
		if v := r.CounterTotal(name); v > 0 {
			out.Counters[name] = v
		}
		if v := r.GaugeTotal(name); v != 0 {
			out.Gauges[name] = v
		}
		if q := r.SummaryQuantiles(name); len(q) > 0 {
			out.Summaries[name] = q
		}
	}
	return out
}
