// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package persist

// CallEvent is an append-only timeline of one call. Every transport
// (SIP / xiaozhi / WebRTC) writes the same shape of rows so an operator
// can reconstruct what happened — when ICE connected, when ASR fired,
// when the dialog plane disconnected — without grepping logs.
//
// Why a separate table from SIPCall.Turns?
//
//   - Turns is dialog content (user said X, assistant said Y) and is
//     bounded and small per call. Events are infra-level signals, can
//     be 50+ per call, and benefit from being indexed by (call_id, at).
//   - Operators query events by kind / level across calls (e.g. "show
//     me every ice_failed in the last hour"); a JSON column inside
//     SIPCall would force a full table scan for that.
//   - Events should survive even if the SIPCall row update fails
//     (disk full, race, …). A separate insert is atomic.

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CallEventLevel values. Kept short so they index well and read cleanly
// in dashboards.
const (
	EventLevelInfo  = "info"
	EventLevelWarn  = "warn"
	EventLevelError = "error"
)

// Common CallEvent.Kind values. Transports MAY write additional kinds —
// this list is the well-known vocabulary the standard tooling understands.
const (
	EventKindCallStarted    = "call.started"
	EventKindCallAccepted   = "call.accepted"
	EventKindCallTerminated = "call.terminated"
	EventKindASRPartial     = "asr.partial"
	EventKindASRFinal       = "asr.final"
	EventKindASRError       = "asr.error"
	EventKindTTSStart       = "tts.start"
	EventKindTTSEnd         = "tts.end"
	EventKindTTSInterrupt   = "tts.interrupt"
	// EventKindE2EFirstByte is emitted once per turn, on the first TTS
	// Speak that follows an ASR final. Detail carries latency markers
	// (asr_final_ms, tts_ttfb_ms, e2e_ms, utter). Dashboards use this
	// to plot the user-perceived "stopped talking → AI starts talking"
	// response time across all transports.
	EventKindE2EFirstByte = "e2e.first_byte"
	EventKindDialogConnect  = "dialog.connect"
	EventKindDialogHangup   = "dialog.hangup"
	EventKindICEConnected   = "ice.connected"
	EventKindICEFailed      = "ice.failed"
	EventKindICEDisconnect  = "ice.disconnect"
	EventKindDTLSConnected  = "dtls.connected"
	EventKindMediaCodec     = "media.codec"
	EventKindMediaRecording = "media.recording"
	EventKindXiaozhiHello   = "xiaozhi.hello"
	EventKindXiaozhiListen  = "xiaozhi.listen"
	EventKindXiaozhiAbort   = "xiaozhi.abort"
)

// CallEvent is one row of `call_events`. Detail is a free-form JSON blob
// (vendor-neutral via datatypes.JSON) carrying whatever payload the
// emitter found useful — keep it small (< 4 KB) for SQLite friendliness.
type CallEvent struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`

	CallID string         `json:"callId" gorm:"size:128;index;not null"`
	At     time.Time      `json:"at" gorm:"index"`
	Kind   string         `json:"kind" gorm:"size:64;index;not null"`
	Level  string         `json:"level" gorm:"size:16;index"`
	Detail datatypes.JSON `json:"detail,omitempty" gorm:"type:json"`
}

// TableName overrides GORM's default pluralisation.
func (CallEvent) TableName() string { return "call_events" }

// AppendCallEvent inserts one event row. Returns nil silently if db is
// nil so callers can use a zero-value db without conditional checks.
//
// callID and kind are required; level defaults to "info"; at defaults
// to time.Now(); detail may be nil.
func AppendCallEvent(ctx context.Context, db *gorm.DB, callID, kind, level string, at time.Time, detail datatypes.JSON) error {
	if db == nil {
		return nil
	}
	if strings.TrimSpace(callID) == "" {
		return errors.New("persist: empty call_id")
	}
	if strings.TrimSpace(kind) == "" {
		return errors.New("persist: empty kind")
	}
	if strings.TrimSpace(level) == "" {
		level = EventLevelInfo
	}
	if at.IsZero() {
		at = time.Now()
	}
	return db.WithContext(ctx).Create(&CallEvent{
		CallID: callID,
		At:     at.UTC(),
		Kind:   strings.TrimSpace(kind),
		Level:  level,
		Detail: detail,
	}).Error
}

// ListCallEventsByCall returns every event for one call, oldest first.
// Useful for rendering a per-call timeline view.
func ListCallEventsByCall(ctx context.Context, db *gorm.DB, callID string) ([]CallEvent, error) {
	if db == nil {
		return nil, errors.New("persist: nil db")
	}
	var rows []CallEvent
	err := db.WithContext(ctx).
		Where("call_id = ?", strings.TrimSpace(callID)).
		Order("at ASC, id ASC").
		Find(&rows).Error
	return rows, err
}
