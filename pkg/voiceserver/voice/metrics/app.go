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

// CallStarted increments the active-calls gauge and the calls_total
// counter for the given transport. Call at the moment the session
// becomes "live" (ASR/TTS wired + dialog plane connected).
func CallStarted(transport string) {
	Default.AddGauge(MetricActiveCalls,
		"currently-active calls broken down by transport",
		map[string]string{"transport": transport}, 1)
}

// CallEnded mirrors CallStarted. status is a short classification like
// "ok", "dialog-hangup", "ice-failed", "pipeline-error" — use the same
// vocabulary you use in call_events.kind so dashboards line up.
func CallEnded(transport, status string) {
	Default.AddGauge(MetricActiveCalls,
		"currently-active calls broken down by transport",
		map[string]string{"transport": transport}, -1)
	Default.IncCounter(MetricCallsTotal,
		"total calls handled since process start, by transport + end status",
		map[string]string{"transport": transport, "status": status})
}

// ASRError bumps the ASR error counter. Called from the recognizer
// error callback in the gateway client.
func ASRError(transport string) {
	Default.IncCounter(MetricASRErrors,
		"total recognizer errors since process start, by transport",
		map[string]string{"transport": transport})
}

// TTSError bumps the TTS error counter. Called when Speak returns an
// error or is interrupted / drained before producing any audio.
func TTSError(transport string) {
	Default.IncCounter(MetricTTSErrors,
		"total synthesis errors since process start, by transport",
		map[string]string{"transport": transport})
}

// BargeIn counts how often the VAD interrupted the AI's TTS because
// the user started talking. Good predictor of conversation health — a
// high rate usually means the AI is too verbose or VAD is too twitchy.
func BargeIn(transport string) {
	Default.IncCounter(MetricBargeInTotal,
		"total barge-in (user interrupted AI) events",
		map[string]string{"transport": transport})
}

// DialogReconnect counts reconnect attempts to the dialog plane
// regardless of outcome. A growing counter means the dialog app is
// flaky; pair with the ok/fail counters for success rate.
func DialogReconnect(transport, outcome string) {
	Default.IncCounter(MetricDialogReconnectTotal,
		"dialog-plane WebSocket reconnect attempts, by outcome",
		map[string]string{"transport": transport, "outcome": outcome})
}

// ObserveE2EFirstByte records the user-perceived latency from ASR
// final to first audible AI byte. Only meaningful values (>0) should
// be passed — 0 means "no ASR final preceded this turn" which
// shouldn't skew the distribution.
func ObserveE2EFirstByte(ms int) {
	if ms <= 0 {
		return
	}
	Default.Observe(MetricE2EFirstByteMs,
		"user-perceived latency: ASR final -> first TTS byte (ms)",
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
