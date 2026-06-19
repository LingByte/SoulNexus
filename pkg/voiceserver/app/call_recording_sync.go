// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	svcmodels "github.com/LingByte/SoulNexus/internal/models/server"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/persist"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/sessionctx"
	"gorm.io/gorm"
)

// SyncVoiceCallToCallRecording copies a finished xiaozhi/hardware call from
// voice_call into call_recordings so DeviceDetail / analytics can list it.
// Idempotent on session_id (= call_id). Best-effort: logs and returns nil
// when tenant context cannot be resolved.
func SyncVoiceCallToCallRecording(db *gorm.DB, callID string) error {
	if db == nil {
		return nil
	}
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return nil
	}

	existing, err := svcmodels.FindCallRecordingBySessionID(db, callID)
	if err == nil && existing != nil {
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	row, err := persist.FindVoiceCallByCallID(context.Background(), db, callID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if row.Transport != "" && row.Transport != "xiaozhi" {
		return nil
	}
	if strings.TrimSpace(row.RecordingURL) == "" && row.TurnCount == 0 {
		return nil
	}

	mac := strings.TrimSpace(row.FromHeader)
	if mac == "" {
		mac = strings.TrimSpace(row.FromNumber)
	}
	if mac == "" {
		logger.Info(fmt.Sprintf("[call-recording-sync] skip call=%s: no device id", callID))
		return nil
	}

	dev, err := svcmodels.GetDeviceByMacAddress(db, mac)
	if err != nil || dev == nil {
		logger.Info(fmt.Sprintf("[call-recording-sync] skip call=%s device=%s: not registered", callID, mac))
		return nil
	}

	agentID := uint(0)
	if dev.AgentID != nil && *dev.AgentID > 0 {
		agentID = *dev.AgentID
	}
	if spec := sessionctx.DefaultRegistry.Get(callID); spec != nil {
		if id, perr := strconv.ParseUint(strings.TrimSpace(spec.AgentID), 10, 32); perr == nil && id > 0 {
			agentID = uint(id)
		}
	}
	if agentID == 0 {
		logger.Info(fmt.Sprintf("[call-recording-sync] skip call=%s device=%s: no agent", callID, mac))
		return nil
	}

	turns, err := persist.UnmarshalVoiceCallTurns(row.Turns)
	if err != nil {
		return fmt.Errorf("unmarshal turns: %w", err)
	}

	start, end := callTimeRange(&row)
	details := voiceTurnsToConversation(callID, turns, start, end)
	metrics := voiceTurnsToTimingMetrics(turns, start, end)

	rec := &svcmodels.CallRecording{
		GroupID:      dev.GroupID,
		CreatedBy:    dev.CreatedBy,
		AgentID:      agentID,
		DeviceID:     dev.ID,
		MacAddress:   mac,
		SessionID:    callID,
		StorageURL:   normalizeRecordingURL(row.RecordingURL, row.RecordingKey),
		AudioFormat:  defaultIfEmpty(row.RecordingFormat, "wav"),
		AudioSize:    int64(row.RecordingWavBytes),
		Duration:     recordingDurationSec(&row),
		SampleRate:   defaultIfZero(row.RecordingSampleRate, 16000),
		Channels:     defaultIfZero(row.RecordingChannels, 2),
		CallType:     "voice",
		CallStatus:   mapVoiceCallStatus(row.EndStatus, row.FailureReason),
		StartTime:    start,
		EndTime:      end,
		AnalysisStatus: "pending",
	}
	if len(turns) > 0 {
		rec.LLMModel = turns[len(turns)-1].LLMModel
		rec.TTSProvider = turns[len(turns)-1].TTSProvider
		rec.ASRProvider = turns[len(turns)-1].ASRProvider
	}
	if err := rec.SetConversationDetails(details); err != nil {
		return err
	}
	if err := rec.SetTimingMetrics(metrics); err != nil {
		return err
	}
	if err := svcmodels.CreateCallRecording(db, rec); err != nil {
		return err
	}
	logger.Info(fmt.Sprintf("[call-recording-sync] created call=%s device=%s agent=%d recording=%t turns=%d",
		callID, mac, agentID, rec.StorageURL != "", len(turns)))
	return nil
}

func callTimeRange(row *persist.VoiceCall) (start, end time.Time) {
	if row.InviteAt != nil && !row.InviteAt.IsZero() {
		start = row.InviteAt.UTC()
	} else {
		start = row.CreatedAt.UTC()
	}
	if row.EndedAt != nil && !row.EndedAt.IsZero() {
		end = row.EndedAt.UTC()
	} else if row.ByeAt != nil && !row.ByeAt.IsZero() {
		end = row.ByeAt.UTC()
	} else {
		end = time.Now().UTC()
	}
	if end.Before(start) {
		end = start
	}
	return start, end
}

func recordingDurationSec(row *persist.VoiceCall) int {
	if row.RecordingDurationMs > 0 {
		sec := int(row.RecordingDurationMs / 1000)
		if sec > 0 {
			return sec
		}
	}
	if row.DurationSec > 0 {
		return row.DurationSec
	}
	start, end := callTimeRange(row)
	if d := end.Sub(start); d > 0 {
		return int(d.Seconds())
	}
	return 0
}

func mapVoiceCallStatus(endStatus, failureReason string) string {
	switch strings.ToLower(strings.TrimSpace(endStatus)) {
	case persist.VoiceCallEndNormalClearing,
		persist.VoiceCallEndCompletedRemote,
		persist.VoiceCallEndCompletedLocal:
		return "completed"
	case persist.VoiceCallEndTransportError, persist.VoiceCallEndServerError:
		return "error"
	}
	r := strings.ToLower(failureReason)
	if strings.Contains(r, "error") || strings.Contains(r, "fail") {
		return "error"
	}
	if strings.Contains(r, "closed") || strings.Contains(r, "bye") || strings.Contains(r, "hangup") {
		return "completed"
	}
	return "interrupted"
}

func normalizeRecordingURL(url, key string) string {
	url = strings.TrimSpace(url)
	if url != "" && strings.HasPrefix(url, "http") {
		return url
	}
	key = strings.Trim(strings.TrimSpace(key), "/")
	if key != "" {
		return "/api/device/recordings/" + key
	}
	if url != "" {
		return url
	}
	return ""
}

func voiceTurnsToConversation(callID string, turns []persist.VoiceCallDialogTurn, start, end time.Time) *svcmodels.ConversationDetails {
	details := &svcmodels.ConversationDetails{
		SessionID: callID,
		StartTime: start,
		EndTime:   end,
	}
	turnID := 1
	for _, t := range turns {
		if asr := strings.TrimSpace(t.ASRText); asr != "" {
			at := t.At.UTC()
			if at.IsZero() {
				at = start
			}
			details.Turns = append(details.Turns, svcmodels.ConversationTurn{
				TurnID:    turnID,
				Timestamp: at,
				Type:      "user",
				Content:   asr,
				StartTime: at,
				EndTime:   at,
			})
			turnID++
			details.UserTurns++
		}
		if llm := strings.TrimSpace(t.LLMText); llm != "" {
			at := t.At.UTC()
			if at.IsZero() {
				at = start
			}
			turn := svcmodels.ConversationTurn{
				TurnID:    turnID,
				Timestamp: at,
				Type:      "ai",
				Content:   llm,
				StartTime: at,
				EndTime:   at,
			}
			if t.LLMWallMs > 0 {
				d := int64(t.LLMWallMs)
				turn.LLMDuration = &d
			}
			if t.TTSMs > 0 {
				d := int64(t.TTSMs)
				turn.TTSDuration = &d
			}
			if t.E2EFirstByteMs > 0 {
				d := int64(t.E2EFirstByteMs)
				turn.TotalDelay = &d
				turn.ResponseDelay = &d
			}
			details.Turns = append(details.Turns, turn)
			turnID++
			details.AITurns++
		}
	}
	details.TotalTurns = len(details.Turns)
	return details
}

func voiceTurnsToTimingMetrics(turns []persist.VoiceCallDialogTurn, start, end time.Time) *svcmodels.TimingMetrics {
	metrics := &svcmodels.TimingMetrics{
		SessionDuration: end.Sub(start).Milliseconds(),
	}
	for _, t := range turns {
		if t.E2EFirstByteMs > 0 {
			metrics.TotalDelays = append(metrics.TotalDelays, int64(t.E2EFirstByteMs))
		}
		if t.LLMFirstMs > 0 {
			metrics.ResponseDelays = append(metrics.ResponseDelays, int64(t.LLMFirstMs))
			metrics.LLMCalls++
			metrics.LLMTotalTime += int64(t.LLMWallMs)
		}
		if t.TTSMs > 0 {
			metrics.TTSCalls++
			metrics.TTSTotalTime += int64(t.TTSMs)
		}
	}
	aggregateDelays := func(values []int64) (avg, min, max int64) {
		if len(values) == 0 {
			return 0, 0, 0
		}
		min, max = values[0], values[0]
		var sum int64
		for _, v := range values {
			sum += v
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}
		return sum / int64(len(values)), min, max
	}
	if metrics.LLMCalls > 0 {
		metrics.LLMAverageTime = metrics.LLMTotalTime / int64(metrics.LLMCalls)
	}
	if metrics.TTSCalls > 0 {
		metrics.TTSAverageTime = metrics.TTSTotalTime / int64(metrics.TTSCalls)
	}
	metrics.AverageTotalDelay, metrics.MinTotalDelay, metrics.MaxTotalDelay = aggregateDelays(metrics.TotalDelays)
	metrics.AverageResponseDelay, metrics.MinResponseDelay, metrics.MaxResponseDelay = aggregateDelays(metrics.ResponseDelays)
	return metrics
}

func defaultIfEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func defaultIfZero(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}
