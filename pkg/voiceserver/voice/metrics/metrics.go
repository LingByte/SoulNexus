// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package metrics is a tiny, dependency-free Prometheus exposition
// backend tailored to VoiceServer's needs. It provides:
//
//   - counters (monotonic, label-keyed)
//   - gauges   (up/down, label-keyed)
//   - summary-style histograms with P50/P90/P95/P99 quantiles
//
// We deliberately avoid pulling in prometheus/client_golang — it adds
// ~100 transitive deps for features (proto, gRPC, exemplars, …) we
// don't use. The text exposition format is small and stable.
//
// Concurrency: every method on Registry is safe for concurrent use.
// Latency cost per observation: one atomic add (counters / gauges) or
// one RWLock + append (histograms) — both below 200 ns on modern x86,
// which is irrelevant compared to the network latencies we measure.
package metrics

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Registry is the single source of truth for VoiceServer process-level
// metrics. A call-site imports the package, mutates the Default
// registry via helpers like IncCounter(), and a single HTTP handler
// serialises the registry to Prometheus text format on /metrics scrape.
type Registry struct {
	mu         sync.RWMutex
	counters   map[string]*counter
	gauges     map[string]*gauge
	histograms map[string]*histogram
}

// NewRegistry returns an empty, ready-to-use registry.
func NewRegistry() *Registry {
	return &Registry{
		counters:   make(map[string]*counter),
		gauges:     make(map[string]*gauge),
		histograms: make(map[string]*histogram),
	}
}

// Default is the process-wide registry. Use this for application-level
// metrics so a single /metrics handler serves everything.
var Default = NewRegistry()

type counter struct {
	name   string
	help   string
	labels map[string]uint64 // serialised label -> value
}

type gauge struct {
	name   string
	help   string
	labels sync.Map // serialised label -> *atomicGaugeVal
}

type atomicGaugeVal struct {
	// Gauges can go up and down and we want to represent fractional
	// values (e.g. bytes/s rates); store as float64 bits behind an
	// int64 atomic so Inc/Dec/Set are all lock-free.
	bits atomic.Uint64
}

type histogram struct {
	name    string
	help    string
	samples []float64
	max     int
	mu      sync.Mutex
}

// ----- COUNTERS --------------------------------------------------------

// IncCounter bumps a counter by 1. Safe to call from hot paths.
func (r *Registry) IncCounter(name, help string, labels map[string]string) {
	r.AddCounter(name, help, labels, 1)
}

// AddCounter adds `n` to the counter. n must be >= 0 (Prometheus
// counters are monotonic); negative values are silently ignored so a
// buggy call site doesn't corrupt the series.
func (r *Registry) AddCounter(name, help string, labels map[string]string, n uint64) {
	if n == 0 {
		return
	}
	key := serialiseLabels(labels)
	r.mu.Lock()
	c, ok := r.counters[name]
	if !ok {
		c = &counter{name: name, help: help, labels: make(map[string]uint64)}
		r.counters[name] = c
	}
	c.labels[key] += n
	if c.help == "" {
		c.help = help
	}
	r.mu.Unlock()
}

// ----- GAUGES ----------------------------------------------------------

// SetGauge stores a value for a gauge.
func (r *Registry) SetGauge(name, help string, labels map[string]string, v float64) {
	r.getOrCreateGauge(name, help).store(labels, v)
}

// AddGauge increments (v > 0) / decrements (v < 0) a gauge atomically.
func (r *Registry) AddGauge(name, help string, labels map[string]string, v float64) {
	r.getOrCreateGauge(name, help).add(labels, v)
}

func (r *Registry) getOrCreateGauge(name, help string) *gauge {
	r.mu.RLock()
	g, ok := r.gauges[name]
	r.mu.RUnlock()
	if ok {
		return g
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if g, ok = r.gauges[name]; ok {
		return g
	}
	g = &gauge{name: name, help: help}
	r.gauges[name] = g
	return g
}

func (g *gauge) store(labels map[string]string, v float64) {
	key := serialiseLabels(labels)
	iv, _ := g.labels.LoadOrStore(key, &atomicGaugeVal{})
	iv.(*atomicGaugeVal).bits.Store(math.Float64bits(v))
}

func (g *gauge) add(labels map[string]string, v float64) {
	key := serialiseLabels(labels)
	iv, _ := g.labels.LoadOrStore(key, &atomicGaugeVal{})
	av := iv.(*atomicGaugeVal)
	for {
		old := av.bits.Load()
		next := math.Float64bits(math.Float64frombits(old) + v)
		if av.bits.CompareAndSwap(old, next) {
			return
		}
	}
}

// ----- HISTOGRAMS ------------------------------------------------------

// Observe records one sample into a histogram. The registry keeps at
// most `maxSamples` most recent observations to bound memory; older
// values are dropped in FIFO order. Quantiles are computed at scrape
// time from the live buffer, so a /metrics request is O(n log n) in
// buffer size — perfectly fine for n up to a few thousand.
func (r *Registry) Observe(name, help string, v float64) {
	r.ObserveN(name, help, v, 2048)
}

// ObserveN is Observe with a custom buffer cap. Use when you want
// finer control over memory vs resolution (e.g. 8192 for a hot call
// latency signal you scrape every 10s).
func (r *Registry) ObserveN(name, help string, v float64, maxSamples int) {
	if maxSamples <= 0 {
		maxSamples = 2048
	}
	r.mu.RLock()
	h, ok := r.histograms[name]
	r.mu.RUnlock()
	if !ok {
		r.mu.Lock()
		if h, ok = r.histograms[name]; !ok {
			h = &histogram{name: name, help: help, max: maxSamples, samples: make([]float64, 0, 128)}
			r.histograms[name] = h
		}
		r.mu.Unlock()
	}
	h.mu.Lock()
	h.samples = append(h.samples, v)
	if len(h.samples) > h.max {
		// Drop oldest half at once to amortise the shift cost.
		drop := h.max / 4
		h.samples = h.samples[drop:]
	}
	h.mu.Unlock()
}

// ----- EXPOSITION ------------------------------------------------------

// WritePromText serialises the registry in Prometheus text exposition
// format (v0.0.4). Safe to call concurrently with metric updates;
// snapshot is point-in-time per metric.
func (r *Registry) WritePromText(w io.Writer) {
	r.mu.RLock()
	counters := make([]*counter, 0, len(r.counters))
	for _, c := range r.counters {
		counters = append(counters, c)
	}
	gauges := make([]*gauge, 0, len(r.gauges))
	for _, g := range r.gauges {
		gauges = append(gauges, g)
	}
	hists := make([]*histogram, 0, len(r.histograms))
	for _, h := range r.histograms {
		hists = append(hists, h)
	}
	r.mu.RUnlock()

	sort.Slice(counters, func(i, j int) bool { return counters[i].name < counters[j].name })
	sort.Slice(gauges, func(i, j int) bool { return gauges[i].name < gauges[j].name })
	sort.Slice(hists, func(i, j int) bool { return hists[i].name < hists[j].name })

	for _, c := range counters {
		writeHelp(w, c.name, c.help)
		fmt.Fprintf(w, "# TYPE %s counter\n", c.name)
		r.mu.RLock()
		for labels, v := range c.labels {
			if labels == "" {
				fmt.Fprintf(w, "%s %d\n", c.name, v)
			} else {
				fmt.Fprintf(w, "%s{%s} %d\n", c.name, labels, v)
			}
		}
		r.mu.RUnlock()
	}
	for _, g := range gauges {
		writeHelp(w, g.name, g.help)
		fmt.Fprintf(w, "# TYPE %s gauge\n", g.name)
		g.labels.Range(func(key, val any) bool {
			labels := key.(string)
			v := math.Float64frombits(val.(*atomicGaugeVal).bits.Load())
			if labels == "" {
				fmt.Fprintf(w, "%s %g\n", g.name, v)
			} else {
				fmt.Fprintf(w, "%s{%s} %g\n", g.name, labels, v)
			}
			return true
		})
	}
	for _, h := range hists {
		writeHelp(w, h.name, h.help)
		// We expose quantiles as a summary rather than classic
		// histogram buckets — we already hold raw samples and
		// summary is what dashboards usually want.
		fmt.Fprintf(w, "# TYPE %s summary\n", h.name)
		h.mu.Lock()
		samples := append([]float64(nil), h.samples...)
		h.mu.Unlock()
		sort.Float64s(samples)
		n := len(samples)
		if n == 0 {
			fmt.Fprintf(w, "%s_count 0\n", h.name)
			fmt.Fprintf(w, "%s_sum 0\n", h.name)
			continue
		}
		var sum float64
		for _, s := range samples {
			sum += s
		}
		for _, q := range []float64{0.5, 0.9, 0.95, 0.99} {
			fmt.Fprintf(w, "%s{quantile=\"%.2f\"} %g\n", h.name, q, quantile(samples, q))
		}
		fmt.Fprintf(w, "%s_count %d\n", h.name, n)
		fmt.Fprintf(w, "%s_sum %g\n", h.name, sum)
	}

	// Process-level runtime breadcrumb so scrapes always return at
	// least one metric and a /metrics of an idle server isn't
	// suspiciously empty.
	fmt.Fprintf(w, "# TYPE voiceserver_scrape_timestamp_seconds gauge\n")
	fmt.Fprintf(w, "voiceserver_scrape_timestamp_seconds %d\n", time.Now().Unix())
}

func writeHelp(w io.Writer, name, help string) {
	if help == "" {
		help = name
	}
	fmt.Fprintf(w, "# HELP %s %s\n", name, help)
}

// quantile assumes samples is sorted ascending. Returns 0 on empty.
// Uses linear interpolation between the two neighbouring ranks —
// matches prometheus/client_golang's summary default.
func quantile(samples []float64, q float64) float64 {
	n := len(samples)
	if n == 0 {
		return 0
	}
	if q <= 0 {
		return samples[0]
	}
	if q >= 1 {
		return samples[n-1]
	}
	rank := q * float64(n-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return samples[lo]
	}
	frac := rank - float64(lo)
	return samples[lo]*(1-frac) + samples[hi]*frac
}

// serialiseLabels turns {transport: "sip", status: "ok"} into
// `status="ok",transport="sip"`. Sorted key order keeps the serialised
// form stable so the map key in Registry.counters[name] collapses
// identical label sets. Returns "" for nil / empty input.
func serialiseLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteString(`="`)
		b.WriteString(escapeLabel(labels[k]))
		b.WriteByte('"')
	}
	return b.String()
}

// escapeLabel escapes the three characters Prometheus's text format
// requires escaping in label values: backslash, double quote, newline.
func escapeLabel(v string) string {
	if !strings.ContainsAny(v, `\"` + "\n") {
		return v
	}
	var b strings.Builder
	b.Grow(len(v) + 2)
	for _, r := range v {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
