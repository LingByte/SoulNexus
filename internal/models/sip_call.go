package models

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SIP call end_status (set on BYE).
const (
	SIPCallEndCompletedRemote     = "completed_remote"
	SIPCallEndCompletedLocal      = "completed_local"
	SIPCallEndAfterTransferRemote = "after_transfer_remote"
	SIPCallEndAfterTransferLocal  = "after_transfer_local"
)

// SIP call state values used when persisting.
const (
	SIPCallStateRinging     = "ringing"
	SIPCallStateEstablished = "established"
	SIPCallStateEnded       = "ended"
)

const (
	SIPCallDirectionInbound  = "inbound"
	SIPCallDirectionOutbound = "outbound"
)

// SIPCallDialogTurn is one ASR→LLM exchange stored in SIPCall.Turns (JSON array).
type SIPCallDialogTurn struct {
	ASRText     string    `json:"asrText"`
	LLMText     string    `json:"llmText"`
	ASRProvider string    `json:"asrProvider,omitempty"`
	TTSProvider string    `json:"ttsProvider,omitempty"`
	LLMModel    string    `json:"llmModel,omitempty"`
	At          time.Time `json:"at"`
	Trigger       string `json:"trigger,omitempty"`
	ScriptStepID  string `json:"scriptStepId,omitempty"`
	RouteIntent   string `json:"routeIntent,omitempty"`
	LLMFirstMs    int    `json:"llmFirstMs,omitempty"`
	LLMWallMs     int    `json:"llmWallMs,omitempty"`
	TTSMs         int    `json:"ttsMs,omitempty"`
	PipelineMs    int    `json:"pipelineMs,omitempty"`
}

// SIPCall records one SIP call lifecycle (INVITE -> ACK -> BYE) and optional AI dialog turns in JSON.
type SIPCall struct {
	BaseModel
	CallID         string         `json:"callId" gorm:"size:128;uniqueIndex;not null"`
	FromHeader     string         `json:"fromHeader" gorm:"type:text"`
	ToHeader       string         `json:"toHeader" gorm:"type:text"`
	CSeqInvite     string         `json:"cseqInvite" gorm:"size:64"`
	RemoteAddr     string         `json:"remoteAddr" gorm:"size:128;index"`    // ip:port of SIP signaling peer
	Direction      string         `json:"direction" gorm:"size:16;index"`      // inbound | outbound
	RemoteRTPAddr  string         `json:"remoteRtpAddr" gorm:"size:128;index"` // ip:port from SDP
	LocalRTPAddr   string         `json:"localRtpAddr" gorm:"size:128;index"`  // ip:port allocated locally
	PayloadType    uint8          `json:"payloadType" gorm:"index"`
	Codec          string         `json:"codec" gorm:"size:32;index"` // pcmu/pcma/opus/g722...
	ClockRate      int            `json:"clockRate"`
	State          string         `json:"state" gorm:"size:32;index"` // init, ringing, established, ended, failed
	InviteAt       *time.Time     `json:"inviteAt" gorm:"index"`
	AckAt          *time.Time     `json:"ackAt" gorm:"index"`
	ByeAt          *time.Time     `json:"byeAt" gorm:"index"`
	EndedAt        *time.Time     `json:"endedAt" gorm:"index"`
	FailureReason  string         `json:"failureReason" gorm:"type:text"`
	RecordingURL   string         `json:"recordingUrl" gorm:"size:1024"`
	// RecordingRawBytes is SN2 blob size from CallSession.TakeRecording (0 if none).
	RecordingRawBytes int `json:"recordingRawBytes" gorm:"column:recording_raw_bytes;default:0"`
	// RecordingWavBytes is mono WAV byte length after decode/mix upload (0 if not produced).
	RecordingWavBytes int `json:"recordingWavBytes" gorm:"column:recording_wav_bytes;default:0"`
	// ByeInitiator is who tore down the SIP dialog for this row: "local" | "remote" (from BYE / Hangup path).
	ByeInitiator string `json:"byeInitiator" gorm:"column:bye_initiator;size:16"`
	DurationSec    int            `json:"durationSec" gorm:"default:0"`
	EndStatus      string         `json:"endStatus" gorm:"size:64;index"` // EndStatus is one of SIPCallEnd* constants.
	Turns          datatypes.JSON `json:"turns" gorm:"type:json"`         // AI dialog (same row as call)
	TurnCount      int            `json:"turnCount" gorm:"default:0"`
	FirstTurnAt    *time.Time     `json:"firstTurnAt"`
	LastTurnAt     *time.Time     `json:"lastTurnAt"`
	HadSIPTransfer bool           `json:"hadSipTransfer" gorm:"column:had_sip_transfer;default:0"`
	HadWebSeat     bool           `json:"hadWebSeat" gorm:"column:had_web_seat;default:0"`
}

func (SIPCall) TableName() string {
	return constants.SIP_CALL_TABLE_NAME
}

// ActiveSIPCalls limits to non–soft-deleted rows.
func ActiveSIPCalls(db *gorm.DB) *gorm.DB {
	return db.Model(&SIPCall{}).Where("is_deleted = ?", SoftDeleteStatusActive)
}

// ListSIPCallsPage lists active calls; list view omits turns JSON for payload size.
func ListSIPCallsPage(db *gorm.DB, page, size int, callID, state string) ([]SIPCall, int64, error) {
	q := ActiveSIPCalls(db)
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
	offset := (page - 1) * size
	var list []SIPCall
	if err := q.Order("id DESC").Offset(offset).Limit(size).Omit("turns").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetActiveSIPCallByID returns one active row by primary key (includes turns).
func GetActiveSIPCallByID(db *gorm.DB, id uint) (SIPCall, error) {
	var row SIPCall
	err := ActiveSIPCalls(db).Where("id = ?", id).First(&row).Error
	return row, err
}

// FindSIPCallByCallID returns a row by SIP Call-ID (any soft-delete state; used on invite upsert).
func FindSIPCallByCallID(ctx context.Context, db *gorm.DB, callID string) (SIPCall, error) {
	var row SIPCall
	err := db.WithContext(ctx).Where("call_id = ?", callID).First(&row).Error
	return row, err
}

// FindActiveSIPCallByCallID returns an active row by SIP Call-ID.
func FindActiveSIPCallByCallID(ctx context.Context, db *gorm.DB, callID string) (SIPCall, error) {
	var row SIPCall
	err := db.WithContext(ctx).
		Where("call_id = ? AND is_deleted = ?", callID, SoftDeleteStatusActive).
		First(&row).Error
	return row, err
}

// NewSIPCallRinging builds a row for first insert on INVITE.
func NewSIPCallRinging(callID, from, to, cseqInvite, remoteAddr, direction, remoteRTP, localRTP string, payloadType uint8, codec string, clockRate int, inviteAt time.Time) SIPCall {
	dir := strings.TrimSpace(direction)
	if dir == "" {
		dir = SIPCallDirectionInbound
	}
	return SIPCall{
		CallID:        callID,
		FromHeader:    from,
		ToHeader:      to,
		CSeqInvite:    cseqInvite,
		RemoteAddr:    remoteAddr,
		Direction:     dir,
		RemoteRTPAddr: remoteRTP,
		LocalRTPAddr:  localRTP,
		PayloadType:   payloadType,
		Codec:         codec,
		ClockRate:     clockRate,
		State:         SIPCallStateRinging,
		InviteAt:      &inviteAt,
	}
}

// SIPCallInviteRefreshUpdateMap updates media / headers when INVITE is seen again for same Call-ID.
func SIPCallInviteRefreshUpdateMap(from, to, remoteAddr, remoteRTP, localRTP, codec string, payloadType uint8, clockRate int, now time.Time) map[string]interface{} {
	return map[string]interface{}{
		"from_header":     from,
		"to_header":       to,
		"remote_addr":     remoteAddr,
		"remote_rtp_addr": remoteRTP,
		"local_rtp_addr":  localRTP,
		"codec":           codec,
		"payload_type":    payloadType,
		"clock_rate":      clockRate,
		"state":           SIPCallStateRinging,
		"updated_at":      now,
	}
}

// SIPCallEstablishedUpdateMap marks ACK / media start.
func SIPCallEstablishedUpdateMap(now time.Time) map[string]interface{} {
	return map[string]interface{}{
		"state":      SIPCallStateEstablished,
		"ack_at":     now,
		"updated_at": now,
	}
}

// SIPCallEndStatusForBye derives end_status from BYE initiator and transfer flags.
func SIPCallEndStatusForBye(initiator string, hadSIPAgentTransfer, hadWebSeat bool) string {
	hadXfer := hadSIPAgentTransfer || hadWebSeat
	local := strings.EqualFold(strings.TrimSpace(initiator), "local")
	if hadXfer {
		if local {
			return SIPCallEndAfterTransferLocal
		}
		return SIPCallEndAfterTransferRemote
	}
	if local {
		return SIPCallEndCompletedLocal
	}
	return SIPCallEndCompletedRemote
}

// SIPCallDurationSince returns seconds from start (ACK preferred, else INVITE) to end; non-positive → 0.
func SIPCallDurationSince(ackAt, inviteAt *time.Time, end time.Time) int {
	var start time.Time
	if ackAt != nil && !ackAt.IsZero() {
		start = *ackAt
	} else if inviteAt != nil && !inviteAt.IsZero() {
		start = *inviteAt
	}
	if start.IsZero() {
		return 0
	}
	sec := int(end.Sub(start).Seconds())
	if sec < 0 {
		return 0
	}
	return sec
}

// SIPCallByeFinalizeUpdateMap builds the update map for BYE (without recording_url; set separately if needed).
func SIPCallByeFinalizeUpdateMap(now time.Time, endStatus string, hadSIPTransfer, hadWebSeat bool, durationSec int) map[string]interface{} {
	return map[string]interface{}{
		"state":            SIPCallStateEnded,
		"bye_at":           now,
		"ended_at":         now,
		"updated_at":       now,
		"end_status":       endStatus,
		"had_sip_transfer": hadSIPTransfer,
		"had_web_seat":     hadWebSeat,
		"duration_sec":     durationSec,
	}
}

// UnmarshalSIPCallTurns parses turns JSON; empty → nil slice, no error.
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

// MarshalSIPCallTurns encodes turns for storage.
func MarshalSIPCallTurns(turns []SIPCallDialogTurn) (datatypes.JSON, error) {
	b, err := json.Marshal(turns)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// NewSIPCallMinimalEstablishedWithFirstTurn creates a row when the first AI turn arrives before INVITE persist.
func NewSIPCallMinimalEstablishedWithFirstTurn(callID string, turns datatypes.JSON, now time.Time) SIPCall {
	return SIPCall{
		CallID:      callID,
		State:       SIPCallStateEstablished,
		Turns:       turns,
		TurnCount:   1,
		FirstTurnAt: &now,
		LastTurnAt:  &now,
	}
}

// SIPCallAppendTurnUpdateMap appends one dialog turn and returns GORM Updates fields plus new turn_count.
func SIPCallAppendTurnUpdateMap(row SIPCall, newTurn SIPCallDialogTurn, now time.Time) (map[string]interface{}, int, error) {
	turnList, err := UnmarshalSIPCallTurns(row.Turns)
	if err != nil {
		return nil, 0, err
	}
	turnList = append(turnList, newTurn)
	turnsBytes, err := MarshalSIPCallTurns(turnList)
	if err != nil {
		return nil, 0, err
	}
	n := len(turnList)
	upd := map[string]interface{}{
		"turns":        turnsBytes,
		"turn_count":   n,
		"last_turn_at": now,
		"updated_at":   now,
	}
	if row.FirstTurnAt == nil || row.FirstTurnAt.IsZero() {
		upd["first_turn_at"] = now
	}
	return upd, n, nil
}

// SelectSIPCallTurnsByCallID loads id, call_id, turns, turn_count for script polling.
func SelectSIPCallTurnsByCallID(db *gorm.DB, callID string) (SIPCall, error) {
	var row SIPCall
	err := db.Select("id", "call_id", "turns", "turn_count").
		Where("call_id = ?", callID).
		Order("id DESC").
		First(&row).Error
	return row, err
}
