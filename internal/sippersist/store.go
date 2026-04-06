package sippersist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/sip/conversation"
	lingstorage "github.com/LingByte/lingstorage-sdk-go"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const maxRecordingBytes = 50 * 1024 * 1024

// Store persists SIPCall (including AI dialog JSON in `turns`) and uploads call recordings via config.GlobalStore.
type Store struct {
	db *gorm.DB
	lg *zap.Logger
}

func New(db *gorm.DB, lg *zap.Logger) *Store {
	if lg == nil {
		lg = zap.NewNop()
	}
	return &Store{db: db, lg: lg}
}

// InviteParams fields for a new inbound call row.
type InviteParams struct {
	CallID        string
	From          string
	To            string
	RemoteSig     string
	RemoteRTP     string
	LocalRTP      string
	Codec         string
	PayloadType   uint8
	ClockRate     int
	CSeqInvite    string
	Direction     string
}

// OnInvite upserts SIPCall in ringing state.
func (s *Store) OnInvite(ctx context.Context, p InviteParams) {
	if s == nil || s.db == nil || p.CallID == "" {
		return
	}
	now := time.Now()
	dir := p.Direction
	if dir == "" {
		dir = "inbound"
	}
	var row models.SIPCall
	err := s.db.WithContext(ctx).Where("call_id = ?", p.CallID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		row = models.SIPCall{
			CallID:        p.CallID,
			FromHeader:    p.From,
			ToHeader:      p.To,
			CSeqInvite:    p.CSeqInvite,
			RemoteAddr:    p.RemoteSig,
			Direction:     dir,
			RemoteRTPAddr: p.RemoteRTP,
			LocalRTPAddr:  p.LocalRTP,
			PayloadType:   p.PayloadType,
			Codec:         p.Codec,
			ClockRate:     p.ClockRate,
			State:         "ringing",
			InviteAt:      &now,
		}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			s.lg.Warn("sippersist invite create", zap.String("call_id", p.CallID), zap.Error(err))
		}
		return
	}
	if err != nil {
		s.lg.Warn("sippersist invite lookup", zap.String("call_id", p.CallID), zap.Error(err))
		return
	}
	_ = s.db.WithContext(ctx).Model(&row).Updates(map[string]interface{}{
		"from_header":     p.From,
		"to_header":       p.To,
		"remote_addr":     p.RemoteSig,
		"remote_rtp_addr": p.RemoteRTP,
		"local_rtp_addr":  p.LocalRTP,
		"codec":           p.Codec,
		"payload_type":    p.PayloadType,
		"clock_rate":      p.ClockRate,
		"state":           "ringing",
		"updated_at":      now,
	}).Error
}

// ByeParams finalizes one SIP dialog leg: updates sip_calls (end flags, recording URL), uploads recording via GlobalStore.
type ByeParams struct {
	CallID     string
	RawPayload []byte
	CodecName  string
	// Initiator is "remote" when the process handles an incoming BYE, or "local" when HangupInboundCall tears down the UAS leg first.
	Initiator string

	// Uplink negotiated media (for Opus→WAV). From CallSession.SourceCodec().
	RecordSampleRate   int
	RecordOpusChannels int
}

func endStatusForBye(initiator string, hadSIPAgentTransfer, hadWebSeat bool) string {
	hadXfer := hadSIPAgentTransfer || hadWebSeat
	local := strings.EqualFold(strings.TrimSpace(initiator), "local")
	if hadXfer {
		if local {
			return models.SIPCallEndAfterTransferLocal
		}
		return models.SIPCallEndAfterTransferRemote
	}
	if local {
		return models.SIPCallEndCompletedLocal
	}
	return models.SIPCallEndCompletedRemote
}

// OnEstablished marks call established (ACK / media start).
func (s *Store) OnEstablished(ctx context.Context, callID string) {
	if s == nil || s.db == nil || callID == "" {
		return
	}
	now := time.Now()
	_ = s.db.WithContext(ctx).Model(&models.SIPCall{}).Where("call_id = ?", callID).Updates(map[string]interface{}{
		"state":      "established",
		"ack_at":     now,
		"updated_at": now,
	}).Error
}

// OnBye finalizes SIPCall, optionally uploads PCMU/PCMA or Opus (length-prefixed RTP payloads) as WAV via config.GlobalStore.
func (s *Store) OnBye(ctx context.Context, p ByeParams) {
	if s == nil || s.db == nil || p.CallID == "" {
		return
	}
	callID := p.CallID
	raw := p.RawPayload
	codecName := p.CodecName
	initiator := p.Initiator
	if initiator == "" {
		initiator = "remote"
	}

	sipAgent, webSeat := conversation.TakeInboundTransferFlags(callID)
	endStatus := endStatusForBye(initiator, sipAgent, webSeat)

	now := time.Now()
	updates := map[string]interface{}{
		"state":            "ended",
		"bye_at":           now,
		"ended_at":         now,
		"updated_at":       now,
		"end_status":       endStatus,
		"had_sip_transfer": sipAgent,
		"had_web_seat":     webSeat,
	}

	var call models.SIPCall
	if err := s.db.WithContext(ctx).Where("call_id = ?", callID).First(&call).Error; err == nil {
		var start time.Time
		if call.AckAt != nil {
			start = *call.AckAt
		} else if call.InviteAt != nil {
			start = *call.InviteAt
		}
		if !start.IsZero() {
			sec := int(now.Sub(start).Seconds())
			if sec < 0 {
				sec = 0
			}
			updates["duration_sec"] = sec
		}
	}

	c := strings.ToLower(codecName)
	bucketOK := config.GlobalStore != nil && config.GlobalConfig != nil &&
		strings.TrimSpace(config.GlobalConfig.Services.Storage.Bucket) != ""
	var wav []byte
	if len(raw) > 0 && bucketOK {
		switch {
		case strings.Contains(c, "pcmu") || strings.Contains(c, "pcma"):
			wav = G711TaggedRecordingToWav(raw, codecName)
		case strings.Contains(c, "opus"):
			sr := p.RecordSampleRate
			if sr <= 0 {
				sr = 48000
			}
			ch := p.RecordOpusChannels
			if ch < 1 {
				ch = 1
			}
			if ch > 2 {
				ch = 2
			}
			wav = MixedOpusRecordingToWav(raw, sr, ch)
			if len(wav) == 0 && ch == 1 {
				wav = MixedOpusRecordingToWav(raw, sr, 2)
			}
			if len(wav) == 0 {
				s.lg.Warn("sippersist opus recording decode failed or empty",
					zap.String("call_id", callID),
					zap.Int("raw_bytes", len(raw)),
					zap.Int("opus_ch", ch),
					zap.Int("sample_rate", sr),
				)
			}
		default:
			s.lg.Warn("sippersist recording skipped (codec not supported for WAV upload)",
				zap.String("call_id", callID), zap.String("codec", codecName), zap.Int("raw_bytes", len(raw)))
		}
		if len(wav) > 0 {
			key := fmt.Sprintf("sip/recordings/%s_%d.wav", sanitizeKey(callID), now.Unix())
			res, err := config.GlobalStore.UploadBytes(&lingstorage.UploadBytesRequest{
				Bucket:   config.GlobalConfig.Services.Storage.Bucket,
				Data:     wav,
				Filename: key,
				Key:      key,
			})
			if err != nil {
				s.lg.Warn("sippersist recording upload", zap.String("call_id", callID), zap.Error(err))
			} else if res != nil && res.URL != "" {
				updates["recording_url"] = res.URL
				s.lg.Info("sippersist recording uploaded", zap.String("call_id", callID), zap.String("codec", codecName))
			}
		}
	} else if len(raw) > 0 && !bucketOK {
		s.lg.Warn("sippersist recording not uploaded (GlobalStore or storage bucket not configured)",
			zap.String("call_id", callID), zap.String("codec", codecName))
	}

	if err := s.db.WithContext(ctx).Model(&models.SIPCall{}).Where("call_id = ?", callID).Updates(updates).Error; err != nil {
		s.lg.Warn("sippersist bye update", zap.String("call_id", callID), zap.Error(err))
	}

}

// SaveConversationTurn appends one ASR→LLM turn onto sip_calls.turns for callID (creates a minimal call row if missing).
func (s *Store) SaveConversationTurn(ctx context.Context, callID, userText, assistantText, asrProvider, llmModel, ttsProvider string) {
	if s == nil || s.db == nil || callID == "" {
		return
	}
	userText = strings.TrimSpace(userText)
	assistantText = strings.TrimSpace(assistantText)
	if userText == "" && assistantText == "" {
		return
	}
	now := time.Now()
	turn := models.SIPCallDialogTurn{
		ASRText:     userText,
		LLMText:     assistantText,
		ASRProvider: asrProvider,
		TTSProvider: ttsProvider,
		LLMModel:    llmModel,
		At:          now,
	}

	var row models.SIPCall
	err := s.db.WithContext(ctx).
		Where("call_id = ? AND is_deleted = ?", callID, models.SoftDeleteStatusActive).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		turnsBytes, jErr := json.Marshal([]models.SIPCallDialogTurn{turn})
		if jErr != nil {
			s.lg.Warn("sippersist call turns marshal failed", zap.String("call_id", callID), zap.Error(jErr))
			return
		}
		row = models.SIPCall{
			CallID:      callID,
			State:       "established",
			Turns:       datatypes.JSON(turnsBytes),
			TurnCount:   1,
			FirstTurnAt: &now,
			LastTurnAt:  &now,
		}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			if err := s.db.WithContext(ctx).
				Where("call_id = ? AND is_deleted = ?", callID, models.SoftDeleteStatusActive).
				First(&row).Error; err != nil {
				s.lg.Warn("sippersist call create/first for turn", zap.String("call_id", callID), zap.Error(err))
				return
			}
		} else {
			if s.lg != nil {
				s.lg.Info("sippersist call created for first AI turn", zap.String("call_id", callID), zap.Uint("row_id", row.ID))
			}
			return
		}
	} else if err != nil {
		s.lg.Warn("sippersist call load for turn", zap.String("call_id", callID), zap.Error(err))
		return
	}

	var turnList []models.SIPCallDialogTurn
	if len(row.Turns) > 0 {
		if uErr := json.Unmarshal(row.Turns, &turnList); uErr != nil {
			s.lg.Warn("sippersist call turns unmarshal failed", zap.String("call_id", callID), zap.Error(uErr))
			turnList = nil
		}
	}
	turnList = append(turnList, turn)
	turnsBytes, jErr := json.Marshal(turnList)
	if jErr != nil {
		s.lg.Warn("sippersist call turns marshal failed", zap.String("call_id", callID), zap.Error(jErr))
		return
	}
	upd := map[string]interface{}{
		"turns":        datatypes.JSON(turnsBytes),
		"turn_count":   len(turnList),
		"last_turn_at": now,
		"updated_at":   now,
	}
	if row.FirstTurnAt == nil || row.FirstTurnAt.IsZero() {
		upd["first_turn_at"] = now
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(upd).Error; err != nil {
		s.lg.Warn("sippersist call turn update failed", zap.String("call_id", callID), zap.Error(err))
		return
	}
	if s.lg != nil {
		s.lg.Info("sippersist call turn appended",
			zap.String("call_id", callID),
			zap.Int("turn_count", len(turnList)),
		)
	}
}

func sanitizeKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "call"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := b.String()
	if len(out) > 120 {
		return out[:120]
	}
	return out
}

// MaxRecordingBytes exported for session package cap alignment.
func MaxRecordingBytes() int { return maxRecordingBytes }
