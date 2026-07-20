// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package metrics

import (
	"net/http"
)

// Metric name constants. Kept in one place so dashboards can grep for a
// single source of truth. Names follow Prometheus convention:
// `<namespace>_<subsystem>_<name>_<unit>`.
const (
	// Calls.
	MetricActiveCalls = "voiceserver_active_calls"
	MetricCallsTotal  = "voiceserver_calls_total"

	// Recognizer / synthesizer errors.
	MetricASRErrors = "voiceserver_asr_errors_total"
	MetricTTSErrors = "voiceserver_tts_errors_total"

	// User-interrupts-AI events.
	MetricBargeInTotal = "voiceserver_barge_in_total"

	// Latencies (milliseconds).
	MetricE2EFirstByteMs = "voiceserver_e2e_first_byte_ms"
	MetricTTSFirstByteMs = "voiceserver_tts_first_byte_ms"
	MetricLLMFirstByteMs = "voiceserver_llm_first_byte_ms"

	// Dialog plane.
	MetricDialogReconnectTotal = "voiceserver_dialog_reconnect_total"
)

// init declares the cardinality whitelist for all app-level metrics
// defined in this file. Adding a new label key requires updating
// this list — that's the whole point of the soft defense.
func init() {
	RegisterLabels(MetricActiveCalls, "transport")
	RegisterLabels(MetricCallsTotal, "transport", "status")
	RegisterLabels(MetricASRErrors, "transport")
	RegisterLabels(MetricTTSErrors, "transport")
	RegisterLabels(MetricBargeInTotal, "transport")
	RegisterLabels(MetricDialogReconnectTotal, "transport", "outcome")
	RegisterLabels(MetricUnknownLabelTotal, "metric", "key")
	RegisterLabels(MetricObserveDroppedTotal, "metric")
}

// labelTransport returns the pre-allocated map for a single
// "transport" label. Hot-path safe (no allocation on known values).
func labelTransport(transport string) map[string]string {
	switch transport {
	case "websocket", "ws":
		return LabelsTransportWebSocket
	case "webrtc":
		return LabelsTransportWebRTC
	}
	return map[string]string{"transport": transport}
}

// CallStarted increments the active-calls gauge and the calls_total
// counter for the given transport. Call at the moment the session
// becomes "live" (ASR/TTS wired + dialog plane connected).
func CallStarted(transport string) {
	Default.AddGauge(MetricActiveCalls,
		"currently-active calls broken down by transport",
		labelTransport(transport), 1)
}

// CallEnded mirrors CallStarted. status is a short classification like
// "ok", "dialog-hangup", "ice-failed", "pipeline-error" — use the same
// vocabulary you use in call_events.kind so dashboards line up.
func CallEnded(transport, status string) {
	Default.AddGauge(MetricActiveCalls,
		"currently-active calls broken down by transport",
		labelTransport(transport), -1)
	Default.IncCounter(MetricCallsTotal,
		"total calls handled since process start, by transport + end status",
		LabelsCall(transport, status))
}

// ASRError bumps the ASR error counter. Called from the recognizer
// error callback in the gateway client.
func ASRError(transport string) {
	Default.IncCounter(MetricASRErrors,
		"total recognizer errors since process start, by transport",
		labelTransport(transport))
}

// TTSError bumps the TTS error counter. Called when Speak returns an
// error or is interrupted / drained before producing any audio.
func TTSError(transport string) {
	Default.IncCounter(MetricTTSErrors,
		"total synthesis errors since process start, by transport",
		labelTransport(transport))
}

// BargeIn counts how often the VAD interrupted the AI's TTS because
// the user started talking. Good predictor of conversation health — a
// high rate usually means the AI is too verbose or VAD is too twitchy.
func BargeIn(transport string) {
	Default.IncCounter(MetricBargeInTotal,
		"total barge-in (user interrupted AI) events",
		labelTransport(transport))
}

// DialogReconnect counts reconnect attempts to the dialog plane
// regardless of outcome. A growing counter means the dialog app is
// flaky; pair with the ok/fail counters for success rate.
func DialogReconnect(transport, outcome string) {
	Default.IncCounter(MetricDialogReconnectTotal,
		"dialog-plane WebSocket reconnect attempts, by outcome",
		LabelsDialogOutcome(transport, outcome))
}

// ObserveE2EFirstByte records user-perceived latency from end of
// user speech to first AI audio byte toward the client. Only
// meaningful values (>0) should be passed.
func ObserveE2EFirstByte(ms int) {
	if ms <= 0 {
		return
	}
	Default.Observe(MetricE2EFirstByteMs,
		"user-perceived latency: speech end -> first TTS byte to client (ms)",
		float64(ms))
}

// ObserveTTSFirstByte records Speak -> first PCM frame latency (ms).
// Measures the TTS engine's cold-start / TTFB across all turns.
func ObserveTTSFirstByte(ms int) {
	if ms <= 0 {
		return
	}
	Default.Observe(MetricTTSFirstByteMs,
		"TTS time-to-first-byte: Speak() -> first PCM frame (ms)",
		float64(ms))
}

// ObserveLLMFirstByte records the dialog app's reported time to first
// LLM token (ms). Comes from CommandMeta.LLMFirstMs on tts.speak.
func ObserveLLMFirstByte(ms int) {
	if ms <= 0 {
		return
	}
	Default.Observe(MetricLLMFirstByteMs,
		"LLM time-to-first-token as reported by the dialog plane (ms)",
		float64(ms))
}

// Handler returns an http.Handler that writes the Default registry in
// Prometheus text exposition format. Mount at /metrics — no auth by
// default; add middleware if the listener is internet-exposed.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		Default.WritePromText(w)
	})
}
