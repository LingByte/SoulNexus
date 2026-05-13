// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package persist

// CallMediaStats is a periodic (or terminal) snapshot of media-quality
// metrics for one call. Each transport pushes whatever it knows:
//
//   - SIP    — packets sent/received from siprtp.Session counters
//   - xiaozhi — bytes in/out, since opus / pcm has no RTCP
//   - WebRTC — RTT, jitter, packet loss, bytes (from pion's GetStats)
//
// The `transport` column lets dashboards filter by class. Multiple rows
// per call are expected — the ones with `final=true` are end-of-call
// summaries, the rest are mid-call samples (5 s cadence is a reasonable
// default).

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// CallMediaStats is one row of `call_media_stats`.
type CallMediaStats struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`

	CallID    string    `json:"callId" gorm:"size:128;index;not null"`
	Transport string    `json:"transport" gorm:"size:32;index"` // sip/xiaozhi/webrtc
	At        time.Time `json:"at" gorm:"index"`
	Final     bool      `json:"final" gorm:"index"`

	// Codec / topology snapshot at sample time.
	Codec      string `json:"codec" gorm:"size:32"`
	ClockRate  int    `json:"clockRate"`
	Channels   int    `json:"channels"`
	RemoteAddr string `json:"remoteAddr" gorm:"size:128"`

	// Counters (cumulative since call start unless noted).
	PacketsSent     uint64 `json:"packetsSent"`
	PacketsReceived uint64 `json:"packetsReceived"`
	BytesSent       uint64 `json:"bytesSent"`
	BytesReceived   uint64 `json:"bytesReceived"`
	PacketsLost     uint64 `json:"packetsLost"`
	NACKsSent       uint64 `json:"nacksSent"`     // we asked for retransmits
	NACKsReceived   uint64 `json:"nacksReceived"` // peer asked us to retransmit

	// Quality metrics (instantaneous).
	RTTMs       int     `json:"rttMs"`       // round-trip time
	JitterMs    int     `json:"jitterMs"`    // RFC 3550 jitter
	LossRate    float64 `json:"lossRate"`    // 0.0 – 1.0
	AudioLevel  int     `json:"audioLevel"`  // RFC 6464, dBov ∈ [-127, 0]
	BitrateKbps int     `json:"bitrateKbps"` // estimated outbound bitrate

	// Free-form note (e.g. "ice pair: host:54321 → srflx:54321").
	Note string `json:"note" gorm:"type:text"`
}

// TableName overrides GORM's default pluralisation.
func (CallMediaStats) TableName() string { return "call_media_stats" }

// AppendCallMediaStats inserts a stats sample. Silent no-op when db is nil.
func AppendCallMediaStats(ctx context.Context, db *gorm.DB, row *CallMediaStats) error {
	if db == nil {
		return nil
	}
	if row == nil {
		return errors.New("persist: nil stats row")
	}
	if strings.TrimSpace(row.CallID) == "" {
		return errors.New("persist: empty call_id")
	}
	if row.At.IsZero() {
		row.At = time.Now()
	}
	row.At = row.At.UTC()
	return db.WithContext(ctx).Create(row).Error
}

// LatestCallMediaStats returns the most recent sample for one call, or
// gorm.ErrRecordNotFound. Used when a single "current state" snapshot
// is enough (e.g. live-call dashboards).
func LatestCallMediaStats(ctx context.Context, db *gorm.DB, callID string) (CallMediaStats, error) {
	var row CallMediaStats
	if db == nil {
		return row, errors.New("persist: nil db")
	}
	err := db.WithContext(ctx).
		Where("call_id = ?", strings.TrimSpace(callID)).
		Order("at DESC, id DESC").
		First(&row).Error
	return row, err
}
