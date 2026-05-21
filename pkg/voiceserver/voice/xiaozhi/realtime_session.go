// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package xiaozhi

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/media"
	"github.com/LingByte/SoulNexus/pkg/realtime"
	"go.uber.org/zap"
)

// RealtimeAgentFactory builds a started realtime.Agent for one xiaozhi session.
type RealtimeAgentFactory interface {
	NewAgent(ctx context.Context, callID string, onEvent func(realtime.Event)) (realtime.Agent, int, int, error)
}

// handleHelloRealtime negotiates codecs and brings up a pkg/realtime agent
// instead of ASR + gateway + TTS.
func (s *session) handleHelloRealtime(ctx context.Context) {
	if s.cfg.RealtimeFactory == nil {
		s.log.Error("xiaozhi: nil RealtimeFactory")
		s.writeText(MakeError("realtime unavailable", true))
		return
	}

	agent, inSR, outSR, err := s.cfg.RealtimeFactory.NewAgent(ctx, s.callID, s.onRealtimeEvent)
	if err != nil {
		s.log.Error("xiaozhi: realtime agent", zap.Error(err))
		s.writeText(MakeError("realtime init failed", true))
		return
	}
	s.rtAgent = agent
	s.rtInSR = inSR
	s.rtOutSR = outSR
	s.rtOut = newRealtimeOutPacer(s)

	s.writeText(MakeWelcomeReply(s.sessionID, AudioParams{
		Format:        s.outFormat,
		SampleRate:    s.outSR,
		Channels:      1,
		FrameDuration: s.ttsWireFrameMs,
		BitDepth:      16,
	}))

	if s.cfg.OnSessionStart != nil {
		s.cfg.OnSessionStart(ctx, s.callID, s.deviceID)
	}
	if s.cfg.RecorderFactory != nil {
		s.rec = s.cfg.RecorderFactory(s.callID, s.inFormat, s.inSR)
	}
	if s.cfg.PersisterFactory != nil {
		remoteAddr := ""
		if s.conn != nil && s.conn.RemoteAddr() != nil {
			remoteAddr = s.conn.RemoteAddr().String()
		}
		fromHdr := s.deviceID
		if fromHdr == "" {
			fromHdr = remoteAddr
		}
		s.persister = s.cfg.PersisterFactory(ctx, s.callID, fromHdr, "xiaozhi", remoteAddr)
		if s.persister != nil {
			s.persister.OnAccept(ctx, s.inFormat, s.inSR, remoteAddr)
			detail, _ := json.Marshal(map[string]any{
				"mode":           ModeRealtime,
				"in_format":      s.inFormat,
				"in_sample_rate": s.inSR,
				"out_format":     s.outFormat,
				"frame_ms":       s.inFrameMs,
				"device_id":      s.deviceID,
				"realtime_in_sr": inSR,
				"realtime_out_sr": outSR,
			})
			s.persister.OnEvent(ctx, "xiaozhi.hello", "info", detail)
		}
	}
	logger.Info(fmt.Sprintf("[xiaozhi] call=%s device=%s connected: mode=realtime in=%s/%dHz out=%s/%dHz rt=%d→%dHz",
		s.callID, s.deviceID, s.inFormat, s.inSR, s.outFormat, s.outSR, inSR, outSR))
}

func (s *session) onRealtimeEvent(ev realtime.Event) {
	if s.closed.Load() {
		return
	}
	switch ev.Type {
	case realtime.EventUserTranscript:
		if !ev.Final {
			return
		}
		text := ev.Text
		s.writeText(MakeSTTReply(s.sessionID, text))
		if s.persister != nil {
			s.persister.OnASRFinal(context.Background(), text)
		}
	case realtime.EventUserSpeechStarted:
		if s.rtAgent != nil {
			_ = s.rtAgent.Cancel()
		}
		if s.rtOut != nil {
			s.rtOut.interrupt(true)
		}
	case realtime.EventAssistantAudio:
		if len(ev.AudioPC) == 0 {
			return
		}
		if !s.ttsActive.Load() {
			s.ttsActive.Store(true)
			s.writeText(MakeTTSStateReplyFrames(s.sessionID, "start", s.outFormat, s.ttsWireFrameMs))
		}
		pcm := ev.AudioPC
		if s.rtOutSR > 0 && s.outSR > 0 && s.rtOutSR != s.outSR {
			resampled, err := media.ResamplePCM(pcm, s.rtOutSR, s.outSR)
			if err != nil {
				s.log.Debug("xiaozhi: realtime resample out", zap.Error(err))
				return
			}
			pcm = resampled
		}
		if s.rec != nil {
			s.rec.WriteAI(pcm)
		}
		if s.rtOut != nil {
			s.rtOut.push(pcm)
		}
	case realtime.EventAssistantTurnEnd:
		if s.rtOut != nil {
			s.rtOut.endTurn()
		}
		if s.ttsActive.CompareAndSwap(true, false) {
			s.writeText(MakeTTSStateReply(s.sessionID, "stop", s.outFormat))
		}
	case realtime.EventError:
		s.log.Warn("xiaozhi: realtime error",
			zap.String("vendor", ev.Vendor),
			zap.Error(ev.Err),
			zap.Bool("fatal", ev.Fatal))
		if ev.Fatal {
			s.teardown("realtime-error")
		}
	case realtime.EventSessionClose:
		if !s.closed.Load() {
			s.teardown("realtime-session-close")
		}
	}
}

func (s *session) handleAudioRealtime(ctx context.Context, frame []byte) {
	if s.rtAgent == nil || !s.listening.Load() {
		return
	}
	var pcm []byte
	if s.opusDec != nil {
		out, err := s.opusDec(&media.AudioPacket{Payload: frame})
		if err != nil || len(out) == 0 {
			return
		}
		ap, _ := out[0].(*media.AudioPacket)
		if ap == nil || len(ap.Payload) == 0 {
			return
		}
		pcm = ap.Payload
	} else {
		pcm = frame
	}
	if s.rec != nil {
		s.rec.WriteCaller(pcm)
	}
	if s.rtInSR > 0 && s.inSR > 0 && s.rtInSR != s.inSR {
		resampled, err := media.ResamplePCM(pcm, s.inSR, s.rtInSR)
		if err != nil {
			s.log.Debug("xiaozhi: realtime resample in", zap.Error(err))
			return
		}
		pcm = resampled
	}
	if err := s.rtAgent.PushAudio(pcm); err != nil {
		s.log.Debug("xiaozhi: realtime push", zap.Error(err))
	}
}
