// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package cdr produces Call Detail Records — one structured event
// per completed call, written to a local JSON-Lines log. The shape
// is tuned for downstream consumption by Filebeat / Vector pipelines
// that ship the events into ClickHouse / Loki / S3 for analysis.
//
// Design contract:
//
//   - Application code emits a record ONCE at end-of-call. The hot
//     path (RTP frames, ASR partials) never touches this package.
//   - Writes are buffered through a goroutine + bounded channel. A
//     burst of call-ends never blocks the producer; if the buffer
//     overflows, the record is dropped and `cdr_dropped_total` is
//     incremented in the metrics registry.
//   - The writer rotates files by date (hourly) and size. A finished
//     file is renamed to `<base>-<UTC>.jsonl` so Filebeat sees an
//     atomic, fully-written artifact.
//   - All file I/O happens on the drain goroutine; the call-end
//     goroutine pays only one channel-send.
package cdr

import "time"

// CallRecord is the schema written as one JSON object per line.
//
// Field naming follows snake_case so SQL engines (ClickHouse,
// BigQuery) consume it without an aliasing layer. Optional fields
// use omitempty so the line stays compact when telemetry isn't
// available.
//
// SCHEMA STABILITY: this is a public log format. Adding fields is
// always safe (downstreams ignore unknowns). RENAMING or REMOVING
// fields breaks downstream pipelines — bump SchemaVersion if you
// must.
type CallRecord struct {
	SchemaVersion int    `json:"schema_version"`
	CallID        string `json:"call_id"`
	CorrelationID string `json:"correlation_id,omitempty"` // tenant/trace correlation
	Transport     string `json:"transport"`                // websocket / webrtc
	Scenario      string `json:"scenario,omitempty"`       // outbound campaign / inbound-ai / bridge
	Codec         string `json:"codec,omitempty"`          // pcmu / pcma / opus / g722

	// Time fields. RFC3339 string for human readability; epoch
	// seconds duplicated for fast time-series indexing without a
	// parse round-trip.
	StartTS      time.Time `json:"start_ts"`
	StartEpoch   int64     `json:"start_epoch"`
	EndTS        time.Time `json:"end_ts"`
	EndEpoch     int64     `json:"end_epoch"`
	DurationMs   int64     `json:"duration_ms"`
	AnsweredMs   int64     `json:"answered_ms,omitempty"` // setup time until first answer

	// Termination classification. EndStatus is a short enum that
	// must come from a bounded set so dashboards can group; long-
	// form detail goes in HangupReason / Errors.
	EndStatus     string   `json:"end_status"`              // ok / dialog-hangup / timer-expired / pipeline-error / signaling-error
	HangupReason  string   `json:"hangup_reason,omitempty"` // hangup cause
	HangupBy      string   `json:"hangup_by,omitempty"`     // local / remote
	FinalStatusCode int    `json:"final_status_code,omitempty"`
	Errors        []string `json:"errors,omitempty"` // short error tags accumulated during the call

	// Voice QoS (filled from RTCP at cleanup).
	RTTMsP95           uint32  `json:"rtt_ms_p95,omitempty"`
	JitterRTPUnits     uint32  `json:"jitter_rtp_units,omitempty"`
	PeerLossFraction   float32 `json:"peer_loss_fraction,omitempty"` // 0..1
	PeerCumulativeLost uint32  `json:"peer_cumulative_lost,omitempty"`
	MOSEstimate        float32 `json:"mos_estimate,omitempty"` // 1.0..4.5 (E-Model)

	// Dialog plane / NLU pipeline.
	Turns           int   `json:"turns,omitempty"`
	ASRCharsTotal   int   `json:"asr_chars_total,omitempty"`
	TTSCharsTotal   int   `json:"tts_chars_total,omitempty"`
	LLMTokensTotal  int   `json:"llm_tokens_total,omitempty"`
	E2EFirstByteP95 int64 `json:"e2e_first_byte_ms_p95,omitempty"`
	BargeInCount    int   `json:"barge_in_count,omitempty"`

	// Free-form structured tail for things we don't want to elevate
	// to first-class columns yet. Keep keys short; values must be
	// JSON-encodable (string / number / bool / nested map).
	Extra map[string]any `json:"extra,omitempty"`
}

// CurrentSchemaVersion is bumped whenever a field is renamed or
// removed. Adding new optional fields does NOT require a bump.
const CurrentSchemaVersion = 1

// NewCallRecord returns a record with SchemaVersion + epochs
// pre-filled so call sites only set what they have.
func NewCallRecord(callID, transport string, start time.Time) CallRecord {
	return CallRecord{
		SchemaVersion: CurrentSchemaVersion,
		CallID:        callID,
		Transport:     transport,
		StartTS:       start,
		StartEpoch:    start.Unix(),
	}
}

// Finalize fills end-time fields and computes durations. Call this
// before emitting so all derived fields are consistent.
func (r *CallRecord) Finalize(end time.Time) {
	if r == nil {
		return
	}
	if r.StartTS.IsZero() {
		r.StartTS = end
	}
	if r.StartEpoch == 0 {
		r.StartEpoch = r.StartTS.Unix()
	}
	r.EndTS = end
	r.EndEpoch = end.Unix()
	r.DurationMs = end.Sub(r.StartTS).Milliseconds()
	if r.DurationMs < 0 {
		r.DurationMs = 0
	}
}
