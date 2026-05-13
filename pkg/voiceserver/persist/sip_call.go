// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package persist contains GORM-backed domain entities and repository helpers
// for the SIP / voice control plane:
//
//   - SIPCall  — one SIP dialog lifecycle row (init → ringing → established →
//     ended/failed). Carries codec/RTP topology, recording metadata, and a
//     JSON column of ASR↔LLM dialog turns for offline review.
//   - SIPUser  — registrar-facing online state for a UA (used to look up an
//     online callee by AOR / username at INVITE time).
//
// Storage portability
//
// All entities are plain GORM models. The default driver is SQLite (file
// `./ling.db`) so a fresh checkout boots without any external DB server.
// MySQL / Postgres / Postgres-flavored DSNs are also supported via the
// existing `pkg/utils.InitDatabase` switch — fields use vendor-neutral types
// (no MySQL-only `JSON`; we use `gorm.io/datatypes.JSON` which maps to TEXT
// on SQLite and JSON on MySQL/Postgres).
//
// On absence of a database
//
// LingEchoX historically stored ASR/LLM dialog state as JSON inside the same
// SQL column (`turns`). We adopt the same pattern here — a single JSON blob
// per call, appended in-place — because:
//
//   1. It avoids a second table and the join cost on read;
//   2. It is identical at rest whether the backing store is SQLite or MySQL;
//   3. It survives schema rolls (no per-turn migration needed).
//
// If at some point a no-DB mode is requested, a thin file-backed Store can be
// added that serialises these same structs as `*.json` per call without
// touching call-site code, since all callers go through the helper functions
// defined here (FindByCallID / Create / AppendTurn / …).
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

// Direction values stored on SIPCall.Direction.
const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
)

// SIPCall lifecycle state values (mirrors LingEchoX's vocabulary so logs and
// dashboards built against either project read the same).
const (
	SIPCallStateInit        = "init"
	SIPCallStateRinging     = "ringing"
	SIPCallStateEstablished = "established"
	SIPCallStateEnded       = "ended"
	SIPCallStateFailed      = "failed"
)

// EndStatus describes why a call ended; populated on the BYE / failure path.
const (
	SIPCallEndUnknown         = "unknown"
	SIPCallEndDeclined        = "declined"
	SIPCallEndBusy            = "busy"
	SIPCallEndCancelled       = "cancelled"
	SIPCallEndNormalClearing  = "normal_clearing"
	SIPCallEndTransportError  = "transport_error"
	SIPCallEndServerError     = "server_error"
	SIPCallEndCompletedRemote = "completed_remote"
	SIPCallEndCompletedLocal  = "completed_local"
)

// SIPCallDialogTurn is one ASR→LLM exchange stored inside SIPCall.Turns.
//
// The struct is intentionally small: it holds the user-visible text on both
// sides and a few latency markers useful for tuning the pipeline. Provider
// names live in the same record so a single call row carries enough context
// to reconstruct who handled what when.
type SIPCallDialogTurn struct {
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

// SIPCall is one row of the `sip_calls` table.
//
// Despite the name, this table now holds calls from ALL transports
// (SIP / xiaozhi / WebRTC) — the `Transport` column distinguishes them.
// The historical name `sip_calls` is preserved for backward compat with
// existing deployments; the Go type is also kept as `SIPCall` to avoid
// a churn-heavy rename across hundreds of call sites. Treat the
// physical table name as an implementation detail and the `Transport`
// column as the source of truth for transport classification.
//
// CallID is the unique key per row:
//
//	• SIP      — the SIP Call-ID header value
//	• xiaozhi  — `xz-{nano}` minted at hello time
//	• WebRTC   — `wrtc-{nano}` minted at signaling time
type SIPCall struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt,omitempty" gorm:"autoUpdateTime"`

	CallID     string `json:"callId" gorm:"size:128;uniqueIndex;not null"`
	Direction  string `json:"direction" gorm:"size:16;index"`
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

	// Recording.
	RecordingURL      string `json:"recordingUrl" gorm:"size:1024"`
	RecordingWavBytes int    `json:"recordingWavBytes" gorm:"default:0"`

	// AI dialog turns appended in-place as JSON.
	Turns       datatypes.JSON `json:"turns" gorm:"type:json"`
	TurnCount   int            `json:"turnCount" gorm:"default:0"`
	FirstTurnAt *time.Time     `json:"firstTurnAt"`
	LastTurnAt  *time.Time     `json:"lastTurnAt"`
}

// TableName overrides GORM's default pluralization to use `sip_calls`.
func (SIPCall) TableName() string { return "sip_calls" }

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

// MarshalSIPCallTurns serialises a slice of dialog turns into the column type.
func MarshalSIPCallTurns(turns []SIPCallDialogTurn) (datatypes.JSON, error) {
	b, err := json.Marshal(turns)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(b), nil
}

// UnmarshalSIPCallTurns reads the JSON column back into Go structs. An empty
// column returns (nil, nil) — callers can treat that as "no turns yet".
func UnmarshalSIPCallTurns(j datatypes.JSON) ([]SIPCallDialogTurn, error) {
	if len(j) == 0 {
		return nil, nil
	}
	var turns []SIPCallDialogTurn
	if err := json.Unmarshal(j, &turns); err != nil {
		return nil, err
	}
	return turns, nil
}

// ApplyRTPMedia stamps codec/RTP topology onto a call row in-memory; useful
// when the SIP layer learns the negotiated codec after Ringing.
func (c *SIPCall) ApplyRTPMedia(remoteIP string, remotePort int, localIP string, localPort int, codec string, pt uint8, clock int) {
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

// FindSIPCallByCallID returns the row for callID, or gorm.ErrRecordNotFound.
func FindSIPCallByCallID(ctx context.Context, db *gorm.DB, callID string) (SIPCall, error) {
	var row SIPCall
	if db == nil {
		return row, errors.New("persist: nil db")
	}
	err := db.WithContext(ctx).Where("call_id = ?", callID).First(&row).Error
	return row, err
}

// CreateSIPCall inserts a new call row. The caller should set CallID, From*,
// To*, Direction, State, InviteAt before calling. CreatedAt/UpdatedAt are
// stamped by GORM.
func CreateSIPCall(ctx context.Context, db *gorm.DB, row *SIPCall) error {
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

// UpdateSIPCallStateByCallID flips state and an arbitrary fields map onto a
// call row identified by callID. Returns rows-affected.
func UpdateSIPCallStateByCallID(ctx context.Context, db *gorm.DB, callID string, fields map[string]any) (int64, error) {
	if db == nil {
		return 0, errors.New("persist: nil db")
	}
	if strings.TrimSpace(callID) == "" {
		return 0, errors.New("persist: empty call_id")
	}
	if len(fields) == 0 {
		return 0, nil
	}
	res := db.WithContext(ctx).Model(&SIPCall{}).Where("call_id = ?", callID).Updates(fields)
	return res.RowsAffected, res.Error
}

// AppendSIPCallTurn appends a dialog turn to the call row's JSON column and
// updates TurnCount / FirstTurnAt / LastTurnAt. If no row exists yet for
// callID, the function creates a minimal "established" row carrying the first
// turn — this lets dialog observers persist turns even before the SIP layer
// has filled in topology metadata (it will be back-filled at ACK / BYE time).
func AppendSIPCallTurn(ctx context.Context, db *gorm.DB, callID string, turn SIPCallDialogTurn) (int, error) {
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
	row, err := FindSIPCallByCallID(ctx, db, callID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		turns := []SIPCallDialogTurn{turn}
		blob, mErr := MarshalSIPCallTurns(turns)
		if mErr != nil {
			return 0, mErr
		}
		row = SIPCall{
			CallID:      callID,
			State:       SIPCallStateEstablished,
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
	existing, uErr := UnmarshalSIPCallTurns(row.Turns)
	if uErr != nil {
		return 0, uErr
	}
	existing = append(existing, turn)
	blob, mErr := MarshalSIPCallTurns(existing)
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
	if err := db.WithContext(ctx).Model(&SIPCall{}).Where("id = ?", row.ID).Updates(upd).Error; err != nil {
		return 0, err
	}
	return len(existing), nil
}

// ListSIPCallsPage returns a single page of calls ordered by id DESC. Filters
// are applied conjunctively; empty filters are ignored. The Turns column is
// omitted from the row payload to keep list responses small — fetch a single
// row via FindSIPCallByCallID to read turns.
func ListSIPCallsPage(ctx context.Context, db *gorm.DB, page, size int, callID, state string) ([]SIPCall, int64, error) {
	if db == nil {
		return nil, 0, errors.New("persist: nil db")
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 500 {
		size = 50
	}
	q := db.WithContext(ctx).Model(&SIPCall{})
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
	var list []SIPCall
	if err := q.Order("id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Omit("turns").
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}
