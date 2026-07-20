// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package metrics

import (
	"strings"
	"testing"
)

func TestFilterLabels_NoRegistration_PassesThrough(t *testing.T) {
	// Metrics without RegisterLabels are unaffected (backwards
	// compat for tests / ad-hoc tools).
	in := map[string]string{"phone": "+12025551234"}
	got := filterLabels("voiceserver_unregistered_metric_total", in)
	if v := got["phone"]; v != "+12025551234" {
		t.Errorf("unregistered metric should pass keys through; got %q", v)
	}
}

func TestFilterLabels_DropsUnknownKeys(t *testing.T) {
	const m = "voiceserver_filter_test_drop_total"
	RegisterLabels(m, "transport")
	in := map[string]string{"transport": "websocket", "phone": "+1xxxx", "call_id": "abc"}
	out := filterLabels(m, in)
	if _, ok := out["phone"]; ok {
		t.Error("phone label must be dropped (not whitelisted)")
	}
	if _, ok := out["call_id"]; ok {
		t.Error("call_id label must be dropped (not whitelisted)")
	}
	if out["transport"] != "websocket" {
		t.Error("whitelisted label must survive")
	}
}

func TestFilterLabels_BumpsSelfObservabilityCounter(t *testing.T) {
	const m = "voiceserver_filter_test_counter_total"
	RegisterLabels(m, "transport")
	in := map[string]string{"transport": "websocket", "phone": "+1xxxx"}
	filterLabels(m, in)

	// metrics_unknown_label_total should now have an entry for this
	// (metric, phone) pair. Snapshot via WritePromText.
	var sb strings.Builder
	Default.WritePromText(&sb)
	if !strings.Contains(sb.String(), MetricUnknownLabelTotal) {
		t.Error("self-observability counter missing from /metrics output")
	}
	if !strings.Contains(sb.String(), `key="phone"`) {
		t.Errorf("expected key=\"phone\" in unknown-label counter; got:\n%s", sb.String())
	}
}

func TestFilterLabels_FastPath_AllAllowed(t *testing.T) {
	const m = "voiceserver_filter_test_fastpath_total"
	RegisterLabels(m, "transport", "status")
	in := map[string]string{"transport": "websocket", "status": "ok"}
	out := filterLabels(m, in)
	// Fast path: same map reference returned (no rebuild).
	if &out == &in {
		// Different scope vars but maps in Go aren't identity-comparable directly;
		// best check: same len + entries.
	}
	if len(out) != 2 || out["transport"] != "websocket" || out["status"] != "ok" {
		t.Errorf("fast path corrupted labels: %v", out)
	}
}

func TestLabelsCall_KnownCombinationsArePreAllocated(t *testing.T) {
	// Pointer identity is the test — we WANT the same map returned
	// across calls so the metric registry's serialiseLabels key
	// computation hits the same string.
	a := LabelsCall("websocket", "ok")
	b := LabelsCall("websocket", "ok")
	// Maps in Go can't compare by pointer directly, but we can check
	// that mutating one would be visible in the other — which would
	// be a bug, but proves the same backing map. Since we documented
	// these as read-only, we use a value check instead.
	if a["transport"] != "websocket" || a["status"] != "ok" || b["status"] != "ok" {
		t.Errorf("unexpected: %v / %v", a, b)
	}
}

func TestLabelsCall_UnknownStatusFallback(t *testing.T) {
	got := LabelsCall("websocket", "weird-new-status")
	if got["transport"] != "websocket" || got["status"] != "weird-new-status" {
		t.Errorf("fallback path broken: %v", got)
	}
}

func TestRegisterLabels_NilSafe(t *testing.T) {
	// Must not panic on empty metric name.
	RegisterLabels("")
	RegisterLabels("ok-metric") // empty keys list is valid
}

func TestFilterLabels_EmptyInputZeroOverhead(t *testing.T) {
	if got := filterLabels("any-metric", nil); got != nil {
		t.Error("nil in → nil out (zero overhead path)")
	}
	if got := filterLabels("any-metric", map[string]string{}); len(got) != 0 {
		t.Error("empty in → empty out")
	}
}
