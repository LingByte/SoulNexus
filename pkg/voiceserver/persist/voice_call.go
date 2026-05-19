// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package persist contains GORM-backed domain entities and repository helpers
// for the voice control plane:
//
//   - VoiceCall — one call lifecycle row (init → ringing → established →
//     ended/failed) for SIP / xiaozhi (WS) / WebRTC. Carries codec / RTP
//     topology, recording metadata (URL / size / format / layout / duration),
//     and a JSON column of ASR↔LLM dialog turns for offline review.
//   - SIPUser — registrar-facing online state for a UA (used to look up an
//     online callee by AOR / username at INVITE time).
//   - CallEvent — per-call timeline entries.
//   - CallMediaStats — per-call media-quality samples.
//
// Historical note
//
// This file (and the `voice_call` table) replaces the earlier split between
// `sip_calls` (lifecycle) and `call_recording` (recording artefact). The
// pcmRecorder always produces exactly one stereo WAV per call, so a 1:1
// merge is sufficient and removes a join from the hot path. Old deployments
// that still have `sip_calls` / `call_recording` tables should migrate data
// out-of-band; this package no longer manages those tables.
//
// Storage portability
//
// All entities are plain GORM models. The default driver is SQLite (file
// `./ling.db`) so a fresh checkout boots without any external DB server.
// MySQL / Postgres are also supported via `pkg/utils.InitDatabase`. Fields
// use vendor-neutral types (no MySQL-only `JSON`; we use
// `gorm.io/datatypes.JSON` which maps to TEXT on SQLite and JSON on
// MySQL/Postgres).
package persist

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Direction values stored on VoiceCall.Direction.
const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
)

// VoiceCall lifecycle state values.
const (
	VoiceCallStateInit        = "init"
	VoiceCallStateRinging     = "ringing"
	VoiceCallStateEstablished = "established"
	VoiceCallStateEnded       = "ended"
	VoiceCallStateFailed      = "failed"
)

// EndStatus describes why a call ended; populated on the BYE / failure path.
const (
	VoiceCallEndUnknown         = "unknown"
	VoiceCallEndDeclined        = "declined"
	VoiceCallEndBusy            = "busy"
	VoiceCallEndCancelled       = "cancelled"
	VoiceCallEndNormalClearing  = "normal_clearing"
	VoiceCallEndTransportError  = "transport_error"
	VoiceCallEndServerError     = "server_error"
	VoiceCallEndCompletedRemote = "completed_remote"
	VoiceCallEndCompletedLocal  = "completed_local"
)

// Recording channel layouts (stamped on VoiceCall.RecordingLayout).
const (
	RecordingLayoutMono     = "mono"
	RecordingLayoutStereoLR = "stereo-l-r" // L=caller R=ai
	RecordingLayoutCaller   = "caller-only"
	RecordingLayoutAIOnly   = "ai-only"
)

// VoiceCallDialogTurn is one ASR→LLM exchange stored inside VoiceCall.Turns.
//
// The struct is intentionally small: it holds the user-visible text on both
// sides and a few latency markers useful for tuning the pipeline. Provider
// names live in the same record so a single call row carries enough context
// to reconstruct who handled what when.
type VoiceCallDialogTurn struct {
	ASRText     string    `json:"asrText"`
	LLMText     string    `json:"llmText"`
	ASRProvider string    `json:"asrProvider,omitempty"`
	TTSProvider string    `json:"ttsProvider,omitempty"`
	LLMModel    string    `json:"llmModel,omitempty"`
	At          time.Time `json:"at"`
	// LLMFirstMs is time from ASR final to first LLM token (ms).
	LLMFirstMs int `json:"llmFirstMs,omitempty"`
	// LLMWallMs is total LLM wall-clock time (ms).
	LLMWallMs int `json:"llmWallMs,omitempty"`
	// TTSMs is total TTS wall-clock playback time (ms).
	TTSMs int `json:"ttsMs,omitempty"`
	// TTSFirstByteMs is the time from invoking TTS Speak() to the first
	// PCM frame leaving the pipeline (TTS engine's cold-start / TTFB).
	// 0 when the Speak failed before producing any audio.
	TTSFirstByteMs int `json:"ttsFirstByteMs,omitempty"`
	// E2EFirstByteMs is the user-perceived end-to-end latency:
	// wall-clock milliseconds from the most recent ASR final to the
	// first audible PCM byte of the AI's response. Only recorded on
	// the FIRST Speak that follows a given ASR final; later
	// sentence-segmented Speaks within the same turn record 0 here
	// because the user has already heard the AI start talking.
	E2EFirstByteMs int `json:"e2eFirstByteMs,omitempty"`
}

// VoiceCall is one row of the `voice_call` table.
//
// One row per call across all transports (SIP / xiaozhi / WebRTC) — the
// `Transport` column distinguishes them. Recording metadata is folded
// into the same row (pcmRecorder produces exactly one stereo WAV per
// call), so listing recordings is `SELECT … WHERE recording_url <> ''`.
//
// CallID is the unique key per row:
//
//	• SIP      — the SIP Call-ID header value
//	• xiaozhi  — `xz-{nano}` minted at hello time
//	• WebRTC   — `wrtc-{nano}` minted at signaling time
type VoiceCall struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" gorm:"autoUpdateTime"`

	CallID    string `json:"callId" gorm:"size:128;uniqueIndex;not null"`
	Direction string `json:"direction" gorm:"size:16;index"`
	// Transport identifies which media plane carried the call. One of
	// "sip" / "xiaozhi" / "webrtc"; empty rows are legacy / unspecified.
	// Lets dashboards split call volume by transport without parsing
	// CallID prefixes.
	Transport       string `json:"transport" gorm:"size:16;index"`
	RemoteUserAgent string `json:"remoteUserAgent" gorm:"size:256"`
	FromHeader      string `json:"fromHeader" gorm:"type:text"`
	ToHeader        string `json:"toHeader" gorm:"type:text"`
	// FromNumber / ToNumber are the bare digits extracted from From/To URI
	// (e.g. `"alice" <sip:13800138000@host>` → "13800138000"); useful for
	// grouping/listing without re-parsing every row.
	FromNumber string `json:"fromNumber" gorm:"size:64;index"`
	ToNumber   string `json:"toNumber" gorm:"size:64;index"`

	// Signaling / media topology.
	RemoteAddr    string `json:"remoteAddr" gorm:"size:128;index"`
	RemoteRTPAddr string `json:"remoteRtpAddr" gorm:"size:128"`
	LocalRTPAddr  string `json:"localRtpAddr" gorm:"size:128"`
	Codec         string `json:"codec" gorm:"size:32;index"`
	PayloadType   uint8  `json:"payloadType"`
	ClockRate     int    `json:"clockRate" gorm:"column:clock_rate"`

	// Lifecycle.
	State         string     `json:"state" gorm:"size:32;index"`
	InviteAt      *time.Time `json:"inviteAt"`
	AckAt         *time.Time `json:"ackAt"`
	ByeAt         *time.Time `json:"byeAt"`
	EndedAt       *time.Time `json:"endedAt"`
	DurationSec   int        `json:"durationSec" gorm:"default:0"`
	EndStatus     string     `json:"endStatus" gorm:"size:64;index"`
	ByeInitiator  string     `json:"byeInitiator" gorm:"size:16"`
	FailureReason string     `json:"failureReason" gorm:"type:text"`

	// Recording — populated when the pcmRecorder finalises a WAV on
	// teardown. Empty rows mean recording was disabled or the call
	// produced no audio (no inbound / outbound PCM seen).
	RecordingURL        string `json:"recordingUrl" gorm:"size:1024"`
	RecordingKey        string `json:"recordingKey" gorm:"size:512"`
	RecordingFormat     string `json:"recordingFormat" gorm:"size:16"`  // "wav", "opus", …
	RecordingLayout     string `json:"recordingLayout" gorm:"size:32"`  // mono / stereo-l-r / caller-only / ai-only
	RecordingSampleRate int    `json:"recordingSampleRate" gorm:"default:0"`
	RecordingChannels   int    `json:"recordingChannels" gorm:"default:0"`
	RecordingWavBytes   int    `json:"recordingWavBytes" gorm:"default:0"`
	RecordingDurationMs int64  `json:"recordingDurationMs" gorm:"default:0"`
	RecordingHash       string `json:"recordingHash" gorm:"size:128"`
	RecordingNote       string `json:"recordingNote" gorm:"type:text"`

	// AI dialog turns appended in-place as JSON.
	Turns       datatypes.JSON `json:"turns" gorm:"type:json"`
	TurnCount   int            `json:"turnCount" gorm:"default:0"`
	FirstTurnAt *time.Time     `json:"firstTurnAt"`
	LastTurnAt  *time.Time     `json:"lastTurnAt"`
}

// TableName overrides GORM's default pluralization. The physical table is
// `voice_call` (singular) — one row per call across all transports.
func (VoiceCall) TableName() string { return "voice_call" }

// ---------- helpers --------------------------------------------------------

// ExtractSIPUserPart returns the user part of a SIP URI inside a From/To
// header. Examples:
//
//	`"Bob" <sip:13800138000@host>;tag=xyz` → "13800138000"
//	`<sip:alice@example.com>`              → "alice"
//	`sip:bob@host`                         → "bob"
//
// It is conservative: anything it cannot confidently parse is returned as-is
// (after stripping spaces). The result is intended for display / index keys,
// not for routing decisions.
func ExtractSIPUserPart(header string) string {
	s := strings.TrimSpace(header)
	if s == "" {
		return ""
	}
	// Strip display name and angle brackets if present.
	if i := strings.Index(s, "<"); i >= 0 {
		if j := strings.Index(s[i:], ">"); j > 0 {
			s = s[i+1 : i+j]
		}
	}
	// Drop scheme.
	for _, sch := range []string{"sip:", "sips:", "tel:"} {
		if strings.HasPrefix(s, sch) {
			s = s[len(sch):]
			break
		}
	}
	// Drop host/params after first '@' or ';'.
	if i := strings.IndexAny(s, "@;"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// MarshalVoiceCallTurns serialises a slice of dialog turns into the column type.
func MarshalVoiceCallTurns(turns []VoiceCallDialogTurn) (datatypes.JSON, error) {
	b, err := json.Marshal(turns)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

// UnmarshalVoiceCallTurns reads the JSON column back into Go structs. An empty
// column returns (nil, nil) — callers can treat that as "no turns yet".
func UnmarshalVoiceCallTurns(j datatypes.JSON) ([]VoiceCallDialogTurn, error) {
	if len(j) == 0 {
		return nil, nil
	}
	var turns []VoiceCallDialogTurn
	if err := json.Unmarshal(j, &turns); err != nil {
		return nil, err
	}
	return turns, nil
}

// ApplyRTPMedia stamps codec/RTP topology onto a call row in-memory; useful
// when the SIP layer learns the negotiated codec after Ringing.
func (c *VoiceCall) ApplyRTPMedia(remoteIP string, remotePort int, localIP string, localPort int, codec string, pt uint8, clock int) {
	if c == nil || remoteIP == "" || remotePort <= 0 {
		return
	}
	c.RemoteRTPAddr = net.JoinHostPort(remoteIP, strconv.Itoa(remotePort))
	if localIP != "" && localPort > 0 {
		c.LocalRTPAddr = net.JoinHostPort(localIP, strconv.Itoa(localPort))
	}
	c.Codec = strings.ToLower(strings.TrimSpace(codec))
	c.PayloadType = pt
	c.ClockRate = clock
}

// ---------- repository functions ------------------------------------------

// FindVoiceCallByCallID returns the row for callID, or gorm.ErrRecordNotFound.
func FindVoiceCallByCallID(ctx context.Context, db *gorm.DB, callID string) (VoiceCall, error) {
	var row VoiceCall
	if db == nil {
		return row, errors.New("persist: nil db")
	}
	err := db.WithContext(ctx).Where("call_id = ?", callID).First(&row).Error
	return row, err
}

// CreateVoiceCall inserts a new call row. The caller should set CallID, From*,
// To*, Direction, State, InviteAt before calling. CreatedAt/UpdatedAt are
// stamped by GORM.
func CreateVoiceCall(ctx context.Context, db *gorm.DB, row *VoiceCall) error {
	if db == nil {
		return errors.New("persist: nil db")
	}
	if row == nil {
		return errors.New("persist: nil call row")
	}
	if strings.TrimSpace(row.CallID) == "" {
		return errors.New("persist: empty call_id")
	}
	return db.WithContext(ctx).Create(row).Error
}

// UpdateVoiceCallStateByCallID flips state and an arbitrary fields map onto a
// call row identified by callID. Returns rows-affected.
func UpdateVoiceCallStateByCallID(ctx context.Context, db *gorm.DB, callID string, fields map[string]any) (int64, error) {
	if db == nil {
		return 0, errors.New("persist: nil db")
	}
	if strings.TrimSpace(callID) == "" {
		return 0, errors.New("persist: empty call_id")
	}
	if len(fields) == 0 {
		return 0, nil
	}
	res := db.WithContext(ctx).Model(&VoiceCall{}).Where("call_id = ?", callID).Updates(fields)
	return res.RowsAffected, res.Error
}

// AppendVoiceCallTurn appends a dialog turn to the call row's JSON column and
// updates TurnCount / FirstTurnAt / LastTurnAt. If no row exists yet for
// callID, the function creates a minimal "established" row carrying the first
// turn — this lets dialog observers persist turns even before the SIP layer
// has filled in topology metadata (it will be back-filled at ACK / BYE time).
func AppendVoiceCallTurn(ctx context.Context, db *gorm.DB, callID string, turn VoiceCallDialogTurn) (int, error) {
	if db == nil {
		return 0, errors.New("persist: nil db")
	}
	if strings.TrimSpace(callID) == "" {
		return 0, errors.New("persist: empty call_id")
	}
	now := turn.At
	if now.IsZero() {
		now = time.Now()
	}
	row, err := FindVoiceCallByCallID(ctx, db, callID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		turns := []VoiceCallDialogTurn{turn}
		blob, mErr := MarshalVoiceCallTurns(turns)
		if mErr != nil {
			return 0, mErr
		}
		row = VoiceCall{
			CallID:      callID,
			State:       VoiceCallStateEstablished,
			Turns:       blob,
			TurnCount:   1,
			FirstTurnAt: &now,
			LastTurnAt:  &now,
		}
		if err := db.WithContext(ctx).Create(&row).Error; err != nil {
			return 0, err
		}
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	existing, uErr := UnmarshalVoiceCallTurns(row.Turns)
	if uErr != nil {
		return 0, uErr
	}
	existing = append(existing, turn)
	blob, mErr := MarshalVoiceCallTurns(existing)
	if mErr != nil {
		return 0, mErr
	}
	upd := map[string]any{
		"turns":        blob,
		"turn_count":   len(existing),
		"last_turn_at": now,
	}
	if row.FirstTurnAt == nil || row.FirstTurnAt.IsZero() {
		upd["first_turn_at"] = now
	}
	if err := db.WithContext(ctx).Model(&VoiceCall{}).Where("id = ?", row.ID).Updates(upd).Error; err != nil {
		return 0, err
	}
	return len(existing), nil
}

// ListVoiceCallsPage returns a single page of calls ordered by id DESC. Filters
// are applied conjunctively; empty filters are ignored. The Turns column is
// omitted from the row payload to keep list responses small — fetch a single
// row via FindVoiceCallByCallID to read turns.
func ListVoiceCallsPage(ctx context.Context, db *gorm.DB, page, size int, callID, state string) ([]VoiceCall, int64, error) {
	if db == nil {
		return nil, 0, errors.New("persist: nil db")
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 500 {
		size = 50
	}
	q := db.WithContext(ctx).Model(&VoiceCall{})
	if cid := strings.TrimSpace(callID); cid != "" {
		q = q.Where("call_id = ?", cid)
	}
	if st := strings.TrimSpace(state); st != "" {
		q = q.Where("state = ?", st)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []VoiceCall
	if err := q.Order("id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Omit("turns").
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
