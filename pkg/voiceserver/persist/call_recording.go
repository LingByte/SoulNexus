// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package persist

// CallRecording is one piece of recorded media from one call. Promoting
// recordings to their own table (vs SIPCall.RecordingURL on the call row)
// has three benefits:
//
//  1. A call can produce multiple recordings — segmented (rotated WAVs),
//     per-channel (caller / AI separately), or different formats (WAV
//     for offline review, opus for storage). One row per artefact keeps
//     the schema clean.
//  2. Recordings carry their own size / duration / hash. That belongs on
//     the recording, not the call.
//  3. Auditing: a `recordings` table is the obvious place for retention
//     policy enforcement (e.g. delete > 90 days) and for storage-bucket
//     reconciliation jobs.

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Recording channel layouts.
const (
	RecordingLayoutMono     = "mono"
	RecordingLayoutStereoLR = "stereo-l-r" // L=caller R=ai
	RecordingLayoutCaller   = "caller-only"
	RecordingLayoutAIOnly   = "ai-only"
)

// CallRecording is one row of `call_recording`.
type CallRecording struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`

	CallID    string `json:"callId" gorm:"size:128;index;not null"`
	Transport string `json:"transport" gorm:"size:32;index"` // sip/xiaozhi/webrtc

	// Storage location. Bucket is the Store bucket name; Key is the path
	// within the bucket. URL is a convenience denormalisation when the
	// Store can produce a public link.
	Bucket string `json:"bucket" gorm:"size:128;index"`
	Key    string `json:"key" gorm:"size:512"`
	URL    string `json:"url" gorm:"size:1024"`

	// Format and layout (e.g. wav / mono / 16 kHz, or wav / stereo-l-r / 16 kHz).
	Format     string `json:"format" gorm:"size:16"`
	Layout     string `json:"layout" gorm:"size:32"`
	SampleRate int    `json:"sampleRate"`
	Channels   int    `json:"channels"`

	// Size and duration (filled at flush time).
	Bytes      int   `json:"bytes"`
	DurationMs int64 `json:"durationMs"`

	// Optional content hash for dedup / integrity (e.g. sha256 hex).
	Hash string `json:"hash" gorm:"size:128"`

	// Free-form note.
	Note string `json:"note" gorm:"type:text"`
}

// TableName overrides GORM's default pluralisation. We use the
// singular form `call_recording` per VoiceServer convention; SoulNexus
// historically used the plural `call_recording` but VoiceServer's
// schema is independent and the user asked for singular here.
func (CallRecording) TableName() string { return "call_recording" }

// AppendCallRecording inserts a recording row. Silent no-op when db is nil.
func AppendCallRecording(ctx context.Context, db *gorm.DB, row *CallRecording) error {
	if db == nil {
		return nil
	}
	if row == nil {
		return errors.New("persist: nil recording row")
	}
	if strings.TrimSpace(row.CallID) == "" {
		return errors.New("persist: empty call_id")
	}
	return db.WithContext(ctx).Create(row).Error
}

// ListCallRecordings returns every recording for one call, oldest first.
func ListCallRecordings(ctx context.Context, db *gorm.DB, callID string) ([]CallRecording, error) {
	if db == nil {
		return nil, errors.New("persist: nil db")
	}
	var rows []CallRecording
	err := db.WithContext(ctx).
		Where("call_id = ?", strings.TrimSpace(callID)).
		Order("id ASC").
		Find(&rows).Error
	return rows, err
}
