package sipserver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/models"
	"github.com/LingByte/SoulNexus/pkg/config"
	"github.com/LingByte/SoulNexus/pkg/scriptlisten"
	"github.com/LingByte/SoulNexus/pkg/sip/conversation"
	sipServer "github.com/LingByte/SoulNexus/pkg/sip/server"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/lingstorage-sdk-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

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

// OnInvite upserts SIPCall in ringing state.
func (s *Store) OnInvite(ctx context.Context, p sipServer.InvitePersistParams) {
	if s == nil || s.db == nil || p.CallID == "" {
		return
	}
	now := time.Now()
	dir := p.Direction
	if dir == "" {
		dir = models.SIPCallDirectionInbound
	}
	row, err := models.FindSIPCallByCallID(ctx, s.db, p.CallID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = models.NewSIPCallRinging(
			p.CallID, p.From, p.To, p.CSeqInvite, p.RemoteSig, dir,
			p.RemoteRTP, p.LocalRTP, p.PayloadType, p.Codec, p.ClockRate, now,
		)
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			s.lg.Warn("sippersist invite create", zap.String("call_id", p.CallID), zap.Error(err))
		}
		return
	}
	if err != nil {
		s.lg.Warn("sippersist invite lookup", zap.String("call_id", p.CallID), zap.Error(err))
		return
	}
	_ = s.db.WithContext(ctx).Model(&row).Updates(models.SIPCallInviteRefreshUpdateMap(
		p.From, p.To, p.RemoteSig, p.RemoteRTP, p.LocalRTP, p.Codec, p.PayloadType, p.ClockRate, now,
	)).Error
}

// OnEstablished marks call established (ACK / media start).
func (s *Store) OnEstablished(ctx context.Context, callID string) {
	if s == nil || s.db == nil || callID == "" {
		return
	}
	now := time.Now()
	_ = s.db.WithContext(ctx).Model(&models.SIPCall{}).Where("call_id = ?", callID).Updates(models.SIPCallEstablishedUpdateMap(now)).Error
}

// OnBye finalizes SIPCall, optionally uploads SN2 recording as stereo WAV (L=user R=AI per-leg decode),
// falling back to legacy mono mix, via config.GlobalStore.
func (s *Store) OnBye(ctx context.Context, p sipServer.ByePersistParams) {
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
	endStatus := models.SIPCallEndStatusForBye(initiator, sipAgent, webSeat)

	now := time.Now()
	durationSec := 0
	var call models.SIPCall
	if err := s.db.WithContext(ctx).Where("call_id = ?", callID).First(&call).Error; err == nil {
		durationSec = models.SIPCallDurationSince(call.AckAt, call.InviteAt, now)
	}
	updates := models.SIPCallByeFinalizeUpdateMap(now, endStatus, sipAgent, webSeat, durationSec)
	if bi := strings.ToLower(strings.TrimSpace(initiator)); bi != "" {
		updates["bye_initiator"] = bi
	}
	if len(raw) > 0 {
		updates["recording_raw_bytes"] = len(raw)
	}

	c := strings.ToLower(codecName)
	bucketOK := config.GlobalStore != nil && config.GlobalConfig != nil &&
		strings.TrimSpace(config.GlobalConfig.Services.Storage.Bucket) != ""
	var wav []byte
	if len(raw) > 0 && bucketOK {
		switch {
		case strings.Contains(c, "pcmu") || strings.Contains(c, "pcma"):
			wav = utils.G711TaggedRecordingToStereoWav(raw, codecName)
			if len(wav) == 0 {
				wav = utils.G711TaggedRecordingToWav(raw, codecName)
			}
		case strings.Contains(c, "g722"):
			wav = utils.G722TaggedRecordingToStereoWav(raw)
			if len(wav) == 0 {
				wav = utils.G722TaggedRecordingToWav(raw)
			}
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
			wav = utils.MixedOpusRecordingToStereoWav(raw, sr, ch)
			if len(wav) == 0 {
				wav = utils.MixedOpusRecordingToWav(raw, sr, ch)
			}
			if len(wav) == 0 && ch == 1 {
				wav = utils.MixedOpusRecordingToStereoWav(raw, sr, 2)
			}
			if len(wav) == 0 && ch == 1 {
				wav = utils.MixedOpusRecordingToWav(raw, sr, 2)
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
				updates["recording_wav_bytes"] = len(wav)
				s.lg.Info("sippersist recording uploaded", zap.String("call_id", callID), zap.String("codec", codecName))
			}
		} else if len(raw) >= 3 && raw[0] == 'S' && raw[1] == 'N' && (raw[2] == '2' || raw[2] == '1') {
			// WAV mux/decode failed or unsupported codec branch — still persist tagged RTP blob for forensics.
			snKey := fmt.Sprintf("sip/recordings/%s_%d.sn2", sanitizeKey(callID), now.Unix())
			res, err := config.GlobalStore.UploadBytes(&lingstorage.UploadBytesRequest{
				Bucket:   config.GlobalConfig.Services.Storage.Bucket,
				Data:     raw,
				Filename: snKey,
				Key:      snKey,
			})
			if err != nil {
				s.lg.Warn("sippersist raw recording upload", zap.String("call_id", callID), zap.Error(err))
			} else if res != nil && res.URL != "" {
				updates["recording_url"] = res.URL
				s.lg.Info("sippersist raw SN recording uploaded (no WAV)", zap.String("call_id", callID), zap.String("codec", codecName), zap.Int("raw_bytes", len(raw)))
			}
		} else if len(raw) > 0 {
			s.lg.Warn("sippersist recording not converted to WAV and not SN1/SN2 tagged",
				zap.String("call_id", callID), zap.String("codec", codecName), zap.Int("raw_bytes", len(raw)))
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
func (s *Store) SaveConversationTurn(ctx context.Context, callID string, t conversation.DialogTurn) {
	if s == nil || s.db == nil || callID == "" {
		return
	}
	userText := strings.TrimSpace(t.ASRText)
	assistantText := strings.TrimSpace(t.LLMText)
	if userText == "" && assistantText == "" {
		return
	}
	now := time.Now()
	if !t.At.IsZero() {
		now = t.At
	}
	turn := models.SIPCallDialogTurn{
		ASRText:      userText,
		LLMText:      assistantText,
		ASRProvider:  t.ASRProvider,
		TTSProvider:  t.TTSProvider,
		LLMModel:     t.LLMModel,
		At:           now,
		Trigger:      t.Trigger,
		ScriptStepID: t.ScriptStepID,
		RouteIntent:  t.RouteIntent,
		LLMFirstMs:   t.LLMFirstMs,
		LLMWallMs:    t.LLMWallMs,
		TTSMs:        t.TTSMs,
		PipelineMs:   t.PipelineMs,
	}

	row, err := models.FindActiveSIPCallByCallID(ctx, s.db, callID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		turnsBytes, jErr := models.MarshalSIPCallTurns([]models.SIPCallDialogTurn{turn})
		if jErr != nil {
			s.lg.Warn("sippersist call turns marshal failed", zap.String("call_id", callID), zap.Error(jErr))
			return
		}
		row = models.NewSIPCallMinimalEstablishedWithFirstTurn(callID, turnsBytes, now)
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			row, err = models.FindActiveSIPCallByCallID(ctx, s.db, callID)
			if err != nil {
				s.lg.Warn("sippersist call create/first for turn", zap.String("call_id", callID), zap.Error(err))
				return
			}
		} else {
			if s.lg != nil {
				s.lg.Info("sippersist call created for first AI turn", zap.String("call_id", callID), zap.Uint("row_id", row.ID))
			}
			scriptlisten.Notify(callID)
			return
		}
	} else if err != nil {
		s.lg.Warn("sippersist call load for turn", zap.String("call_id", callID), zap.Error(err))
		return
	}

	upd, turnCount, uErr := models.SIPCallAppendTurnUpdateMap(row, turn, now)
	if uErr != nil {
		s.lg.Warn("sippersist call turns merge failed", zap.String("call_id", callID), zap.Error(uErr))
		return
	}
	if err := s.db.WithContext(ctx).Model(&row).Updates(upd).Error; err != nil {
		s.lg.Warn("sippersist call turn update failed", zap.String("call_id", callID), zap.Error(err))
		return
	}
	scriptlisten.Notify(callID)
	if s.lg != nil {
		s.lg.Info("sippersist call turn appended",
			zap.String("call_id", callID),
			zap.Int("turn_count", turnCount),
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
