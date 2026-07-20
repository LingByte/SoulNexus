// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceattach

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	dialogaudio "github.com/LingByte/SoulNexus/pkg/dialog/audio"
	dialogrealtime "github.com/LingByte/SoulNexus/pkg/dialog/realtime"
	"github.com/LingByte/SoulNexus/pkg/dialog/tenantcfg"
	"github.com/LingByte/lingllm/realtime"
	"go.uber.org/zap"
)

const nativeOmniTurnWatch = 8 * time.Second

// WelcomeResolved is a welcome plan for native realtime session lifecycle.
type WelcomeResolved struct {
	AssistantText string
	WillPlay      bool
}

// nativeRealtimeSession owns post-attach lifecycle for StreamingPort realtime:
// TriggerProactiveGreeting and silent-turn watchdog.
type nativeRealtimeSession struct {
	callID           string
	lg               *zap.Logger
	assistantWelcome string
	hadWelcomeClip   bool
	welcomePlaying   func() bool
	mediaCtx         context.Context

	agentMu sync.Mutex
	agent   realtime.Agent

	instrMu         sync.Mutex
	instructionBase string // prompt without dynamic noise block
	lastNoiseLevel  dialogaudio.NoiseLevel

	welcomeDone atomic.Bool
	agentReady  atomic.Bool
	proactive   atomic.Bool

	turnHadAudio    atomic.Bool
	turnNudgeSent   atomic.Bool
	omniWatchMu     sync.Mutex
	omniWatchCancel context.CancelFunc
}

var nativeRTSessions sync.Map // callID → *nativeRealtimeSession

// beginNativeRealtimeSession registers per-call lifecycle state looked up by
// buildNativeRealtimeAgent / adapter.
func beginNativeRealtimeSession(
	callID string,
	env tenantcfg.VoiceEnv,
	lg *zap.Logger,
	plan WelcomeResolved,
	welcomePlaying func() bool,
	mediaCtx context.Context,
) *nativeRealtimeSession {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return nil
	}
	s := &nativeRealtimeSession{
		callID:           callID,
		lg:               lg,
		assistantWelcome: plan.AssistantText,
		hadWelcomeClip:   plan.WillPlay,
		welcomePlaying:   welcomePlaying,
		mediaCtx:         mediaCtx,
	}
	if strings.TrimSpace(s.assistantWelcome) == "" {
		s.assistantWelcome = tenantcfg.ResolvedAssistantWelcome(env)
	}
	nativeRTSessions.Store(callID, s)
	dialogaudio.SetCallNoiseListener(callID, s.onNoiseLevel)
	return s
}

func lookupNativeRealtimeSession(callID string) *nativeRealtimeSession {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return nil
	}
	v, ok := nativeRTSessions.Load(callID)
	if !ok {
		return nil
	}
	return v.(*nativeRealtimeSession)
}

func endNativeRealtimeSession(callID string) {
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return
	}
	dialogaudio.SetCallNoiseListener(callID, nil)
	dialogaudio.GlobalCallNoise.Clear(callID)
	if v, ok := nativeRTSessions.LoadAndDelete(callID); ok {
		s := v.(*nativeRealtimeSession)
		s.clearOmniTurnWatch()
	}
}

// setInstructionBase stores the static session prompt (transfer hints may still be inside).
func (s *nativeRealtimeSession) setInstructionBase(base string) {
	if s == nil {
		return
	}
	s.instrMu.Lock()
	s.instructionBase = dialogaudio.StripNoiseHint(base)
	s.instrMu.Unlock()
}

func (s *nativeRealtimeSession) onNoiseLevel(level dialogaudio.NoiseLevel, snrDB float64) {
	if s == nil {
		return
	}
	s.instrMu.Lock()
	if level == s.lastNoiseLevel {
		s.instrMu.Unlock()
		return
	}
	s.lastNoiseLevel = level
	base := s.instructionBase
	s.instrMu.Unlock()

	agent := s.agentOrNil()
	if agent == nil || strings.TrimSpace(base) == "" {
		return
	}
	full := dialogaudio.ApplyNoiseHint(base, dialogaudio.NoiseSessionHint(level))
	if err := agent.UpdateInstructions(full); err != nil {
		if s.lg != nil {
			s.lg.Debug("native realtime: noise adapt UpdateInstructions failed",
				zap.String("call_id", s.callID),
				zap.String("level", level.String()),
				zap.Float64("snr_db", snrDB),
				zap.Error(err))
		}
		return
	}
	if s.lg != nil {
		s.lg.Info("native realtime: adapted instructions for acoustic noise",
			zap.String("call_id", s.callID),
			zap.String("level", level.String()),
			zap.Float64("snr_db", snrDB))
	}
}

func (s *nativeRealtimeSession) MarkWelcomeDone() {
	if s == nil {
		return
	}
	s.welcomeDone.Store(true)
	s.maybeProactiveGreeting()
}

func (s *nativeRealtimeSession) onAgentStarted(agent realtime.Agent, _ context.Context) {
	if s == nil || agent == nil {
		return
	}
	s.agentMu.Lock()
	s.agent = agent
	s.agentMu.Unlock()
	s.agentReady.Store(true)

	s.maybeProactiveGreeting()
	// Apply SNR class discovered during welcome / early uplink if any.
	if lv := dialogaudio.GlobalCallNoise.Get(s.callID); lv != dialogaudio.NoiseLevelUnknown {
		s.onNoiseLevel(lv, 0)
	}
}

func (s *nativeRealtimeSession) agentOrNil() realtime.Agent {
	if s == nil {
		return nil
	}
	s.agentMu.Lock()
	defer s.agentMu.Unlock()
	return s.agent
}

func (s *nativeRealtimeSession) maybeProactiveGreeting() {
	if s == nil || !s.agentReady.Load() || !s.welcomeDone.Load() {
		return
	}
	if !s.proactive.CompareAndSwap(false, true) {
		return
	}
	agent := s.agentOrNil()
	if agent == nil {
		return
	}
	dialogrealtime.TriggerProactiveGreeting(
		agent, s.assistantWelcome, s.lg, s.callID,
		true, s.hadWelcomeClip,
	)
}

func (s *nativeRealtimeSession) observeVendorEvent(ev realtime.Event) {
	if s == nil {
		return
	}
	switch ev.Type {
	case realtime.EventUserSpeechStarted:
		if s.welcomePlaying != nil && s.welcomePlaying() {
			return
		}
	case realtime.EventUserTranscript:
		if ev.Final && s.serverVADActive() {
			s.resetUserTurn()
			s.scheduleOmniTurnWatch("user_transcript_final")
		}
	case realtime.EventAssistantAudio:
		if len(ev.AudioPC) > 0 {
			s.turnHadAudio.Store(true)
			s.clearOmniTurnWatch()
		}
	case realtime.EventAssistantText:
		if ev.Final {
			s.clearOmniTurnWatch()
			if s.serverVADActive() && !s.turnHadAudio.Load() {
				s.nudgeOmniResponseIfNeeded("empty_assistant_final")
			}
		}
	case realtime.EventAssistantTurnEnd:
		s.clearOmniTurnWatch()
	}
}

func (s *nativeRealtimeSession) serverVADActive() bool {
	return s != nil
}

func (s *nativeRealtimeSession) resetUserTurn() {
	s.turnHadAudio.Store(false)
	s.turnNudgeSent.Store(false)
}

func (s *nativeRealtimeSession) clearOmniTurnWatch() {
	s.omniWatchMu.Lock()
	if s.omniWatchCancel != nil {
		s.omniWatchCancel()
		s.omniWatchCancel = nil
	}
	s.omniWatchMu.Unlock()
}

func (s *nativeRealtimeSession) scheduleOmniTurnWatch(reason string) {
	if s == nil || !s.serverVADActive() || s.agentOrNil() == nil {
		return
	}
	s.clearOmniTurnWatch()
	watchCtx, cancel := context.WithCancel(context.Background())
	s.omniWatchMu.Lock()
	s.omniWatchCancel = cancel
	s.omniWatchMu.Unlock()
	go func() {
		select {
		case <-watchCtx.Done():
			return
		case <-time.After(nativeOmniTurnWatch):
		}
		if s.turnHadAudio.Load() {
			return
		}
		s.nudgeOmniResponseIfNeeded("watchdog:" + reason)
	}()
}

func (s *nativeRealtimeSession) nudgeOmniResponseIfNeeded(reason string) {
	if s == nil || !s.serverVADActive() || s.turnHadAudio.Load() {
		return
	}
	if !s.turnNudgeSent.CompareAndSwap(false, true) {
		return
	}
	agent := s.agentOrNil()
	starter, ok := agent.(dialogrealtime.ResponseStarter)
	if !ok {
		s.turnNudgeSent.Store(false)
		return
	}
	if err := starter.CreateResponse(""); err != nil {
		s.turnNudgeSent.Store(false)
		if s.lg != nil {
			s.lg.Warn("native realtime: response.create nudge failed",
				zap.String("call_id", s.callID),
				zap.String("reason", reason),
				zap.Error(err))
		}
		return
	}
	if s.lg != nil {
		s.lg.Warn("native realtime: nudging response.create after silent omni turn",
			zap.String("call_id", s.callID),
			zap.String("reason", reason))
	}
}
