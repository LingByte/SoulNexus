// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package metrics

// Cardinality discipline — read this before adding metrics.
//
// Prometheus / time-series databases store one series per unique
// label combination. Forgetting this and labeling with a Call-ID,
// phone number, or RTP SSRC will produce one new series per call,
// which crashes the TSDB within hours.
//
// This file ships the *soft-defense* layer chosen during the
// 2026-05-17 observability design review:
//
//   1. Each metric MUST declare its allowed label keys via
//      RegisterLabels(metric, keys...). This is the explicit
//      whitelist of what's allowed to vary.
//   2. Calls that pass label keys NOT in the whitelist will:
//        - Drop those keys from the recorded series (so cardinality
//          stays bounded).
//        - Log a warning once per (metric, offending key) pair.
//        - Increment metrics_unknown_label_total{metric, key} so
//          this self-violation is visible on the very same
//          dashboards that consume the metrics.
//   3. Calls to metrics that haven't called RegisterLabels at all
//      go through unchanged. This keeps tests and ad-hoc tools
//      working without forcing a whitelist on every new metric.
//
// Pre-allocated label maps — the constants in this file — are the
// recommended call style. They are zero-alloc and zero-string-build
// at the call site:
//
//   metrics.Default.IncCounter(MetricCallsTotal, "", LabelsCall("websocket", "ok"))
//
// Hot-path rule: code on the per-RTP-frame path MUST NOT call any
// metric function that takes a map literal. Use the helpers below.

import (
	"sync"
	"sync/atomic"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// MetricUnknownLabelTotal counts soft-whitelist violations. Visible
// via /metrics so on-call can spot "someone is shipping a metric the
// declared whitelist doesn't cover" without grepping logs.
const MetricUnknownLabelTotal = "voiceserver_metrics_unknown_label_total"

// MetricObserveDroppedTotal counts samples lost because the async
// Observe buffer was full. If this is non-zero in production the
// drain goroutine isn't keeping up — usually a downstream stall
// rather than a real load issue.
const MetricObserveDroppedTotal = "voiceserver_metrics_observe_dropped_total"

// allowedLabels holds the cardinality whitelist registered via
// RegisterLabels. The empty-map case (metric not registered) is
// deliberately distinguished from registered-with-no-labels so we
// can fall through silently for ad-hoc metrics.
var (
	allowedLabelsMu sync.RWMutex
	allowedLabels   = make(map[string]map[string]struct{})

	// unknownLabelWarnedOnce keeps the warn-log rate-limited per
	// (metric, key) — without this a misuse on a hot path would
	// spam the log at observation rate.
	unknownLabelWarnedOnce sync.Map // key = "metric|labelKey" -> *atomic.Bool
)

// RegisterLabels declares the allowed label keys for a metric.
// Subsequent updates with extra keys will have those keys dropped
// (soft defense). Calling RegisterLabels twice for the same metric
// REPLACES the whitelist (last write wins) — intended for tests.
//
// Safe to call from init().
func RegisterLabels(metric string, keys ...string) {
	if metric == "" {
		return
	}
	set := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		if k != "" {
			set[k] = struct{}{}
		}
	}
	allowedLabelsMu.Lock()
	allowedLabels[metric] = set
	allowedLabelsMu.Unlock()
}

// filterLabels returns a (possibly new) label map containing only
// keys allowed for the given metric. If the metric hasn't been
// registered, the input map is returned unchanged (zero overhead).
// If filtering happened, dropped keys are reported via the soft
// defense pipeline.
//
// This is on the warm path (called once per Inc/Add/Set/Observe).
// We optimise for the common case "all keys allowed": a single
// RLock + map lookup + len-comparison, no allocations.
func filterLabels(metric string, in map[string]string) map[string]string {
	if len(in) == 0 {
		return in
	}
	allowedLabelsMu.RLock()
	allowed, ok := allowedLabels[metric]
	allowedLabelsMu.RUnlock()
	if !ok {
		// Metric not registered — pass through (backwards compat).
		return in
	}

	// Fast path: all keys are allowed.
	allOK := true
	for k := range in {
		if _, present := allowed[k]; !present {
			allOK = false
			break
		}
	}
	if allOK {
		return in
	}

	// Slow path: rebuild with only allowed keys, log dropped ones once.
	out := make(map[string]string, len(in))
	for k, v := range in {
		if _, present := allowed[k]; present {
			out[k] = v
			continue
		}
		reportUnknownLabel(metric, k)
	}
	return out
}

// reportUnknownLabel logs once per (metric,key) and bumps the
// self-observability counter. We bypass the whitelist for that
// counter so it always reaches the registry.
func reportUnknownLabel(metric, key string) {
	cacheKey := metric + "|" + key
	v, _ := unknownLabelWarnedOnce.LoadOrStore(cacheKey, &atomic.Bool{})
	flag := v.(*atomic.Bool)
	if flag.CompareAndSwap(false, true) {
		if logger.Lg != nil {
			logger.Lg.Warn("metrics: dropping unknown label (cardinality defense)",
				zap.String("metric", metric),
				zap.String("dropped_key", key))
		}
	}
	// Increment the self-observability counter directly on the
	// registry (no filtering, no recursion) — that's why
	// addCounterRaw exists.
	Default.addCounterRaw(MetricUnknownLabelTotal,
		"label keys dropped because they weren't declared via RegisterLabels",
		map[string]string{"metric": metric, "key": key}, 1)
}

// ----- Pre-allocated label maps (the call-site-safe path) ----------

// These are deliberately defined as VARS not consts because Go maps
// can't be const. They MUST NOT be mutated at runtime — treat them
// as read-only. Each is sized for one common label combination so
// hot-path callers don't have to allocate.
//
// If you need a label combination not listed here, either:
//   (a) prefer using one of these and accept the loss of dimension, or
//   (b) add it to this file with a comment explaining the use site.
// DO NOT inline `map[string]string{...}` literals in hot paths.

// LabelsTransportWebSocket / LabelsTransportWebRTC are the two transports
// we use today. The whitelist for any metric labelled by transport
// should be: RegisterLabels(metric, "transport").
var (
	LabelsTransportWebSocket = map[string]string{"transport": "websocket"}
	LabelsTransportWebRTC = map[string]string{"transport": "webrtc"}
)

// LabelsCall composes a 2-key label set for the common (transport,
// status) shape used by voiceserver_calls_total. We pre-build the
// known combinations rather than allocating per-call. Add more
// statuses here if dashboards need to slice on them.
//
// Return type is map[string]string to fit the existing API; pointer
// identity is preserved across calls so map-key dedupe inside the
// registry stays cheap.
func LabelsCall(transport, status string) map[string]string {
	switch transport + "|" + status {
	case "websocket|ok":
		return labelsCallWebSocketOK
	case "websocket|error":
		return labelsCallWebSocketError
	case "webrtc|ok":
		return labelsCallWebRTCOK
	case "webrtc|error":
		return labelsCallWebRTCError
	}
	// Cold path for combinations we didn't pre-allocate. Acceptable
	// because call-end happens at most ~1 Hz/leg.
	return map[string]string{"transport": transport, "status": status}
}

var (
	labelsCallWebSocketOK    = map[string]string{"transport": "websocket", "status": "ok"}
	labelsCallWebSocketError = map[string]string{"transport": "websocket", "status": "error"}
	labelsCallWebRTCOK       = map[string]string{"transport": "webrtc", "status": "ok"}
	labelsCallWebRTCError    = map[string]string{"transport": "webrtc", "status": "error"}
)

// LabelsDialogOutcome is used by DialogReconnect — bounded set of
// outcomes per the original API contract.
func LabelsDialogOutcome(transport, outcome string) map[string]string {
	switch transport + "|" + outcome {
	case "websocket|ok":
		return labelsDialogWebSocketOk
	case "websocket|fail":
		return labelsDialogWebSocketFail
	case "webrtc|ok":
		return labelsDialogWebRTCOk
	case "webrtc|fail":
		return labelsDialogWebRTCFail
	}
	return map[string]string{"transport": transport, "outcome": outcome}
}

var (
	labelsDialogWebSocketOk   = map[string]string{"transport": "websocket", "outcome": "ok"}
	labelsDialogWebSocketFail = map[string]string{"transport": "websocket", "outcome": "fail"}
	labelsDialogWebRTCOk      = map[string]string{"transport": "webrtc", "outcome": "ok"}
	labelsDialogWebRTCFail    = map[string]string{"transport": "webrtc", "outcome": "fail"}
)
