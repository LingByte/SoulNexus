package xiaozhi

import (
	"context"
	"strings"

	"github.com/LingByte/lingllm/media"
	"github.com/LingByte/lingllm/realtime"
)

// RealtimeAgentFactory builds a started realtime.Agent for one xiaozhi session.
type RealtimeAgentFactory interface {
	NewAgent(ctx context.Context, callID string, onEvent func(realtime.Event)) (realtime.Agent, int, int, error)
}

func (s *wsSession) handleHelloRealtime(ctx context.Context) {
	if s.cfg.RealtimeFactory == nil {
		s.writeText(MakeError("realtime unavailable", true))
		return
	}

	agent, inSR, outSR, err := s.cfg.RealtimeFactory.NewAgent(ctx, s.callID, s.onRealtimeEvent)
	if err != nil {
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
}

func (s *wsSession) onRealtimeEvent(ev realtime.Event) {
	if s.closed.Load() {
		return
	}
	switch ev.Type {
	case realtime.EventUserTranscript:
		if !ev.Final {
			return
		}
		s.writeText(MakeSTTReply(s.sessionID, ev.Text))
	case realtime.EventUserSpeechStarted:
		// Model server VAD barge-in while assistant audio is playing.
		if !s.ttsActive.Load() {
			return
		}
		s.rtReplyBusy.Store(false)
		if s.rtAgent != nil {
			_ = s.rtAgent.Cancel()
		}
		if s.rtOut != nil {
			s.rtOut.interrupt(true)
		}
	case realtime.EventAssistantText:
		// Mute uplink only while text is streaming (before first audio chunk).
		if ev.Text != "" && !s.ttsActive.Load() {
			s.rtReplyBusy.Store(true)
		}
		if ev.Final {
			full := strings.TrimSpace(ev.Text)
			if full == "" {
				full = strings.TrimSpace(s.turnLLMText)
			}
			if full != "" {
				s.writeText(MakeLLMReply(full))
			}
			s.turnLLMText = ""
		} else if ev.Text != "" {
			s.turnLLMText += ev.Text
		}
	case realtime.EventAssistantAudio:
		if len(ev.AudioPC) == 0 {
			return
		}
		// Resume uplink so the model's server VAD can detect user barge-in.
		s.rtReplyBusy.Store(false)
		if !s.ttsActive.Load() {
			s.ttsActive.Store(true)
			s.writeText(MakeTTSStateReplyFrames(s.sessionID, "start", s.outFormat, s.ttsWireFrameMs))
		}
		pcm := ev.AudioPC
		if s.rtOutSR > 0 && s.outSR > 0 && s.rtOutSR != s.outSR {
			resampled, err := media.ResamplePCM(pcm, s.rtOutSR, s.outSR)
			if err != nil {
				return
			}
			pcm = resampled
		}
		if s.rtOut != nil {
			s.rtOut.push(pcm)
		}
	case realtime.EventAssistantTurnEnd:
		if s.rtOut != nil {
			_ = s.rtOut.endTurn()
		}
		if s.ttsActive.CompareAndSwap(true, false) {
			s.writeText(MakeTTSStateReply(s.sessionID, "stop", s.outFormat))
		}
		s.rtReplyBusy.Store(false)
	case realtime.EventError:
		msg := "realtime error"
		if ev.Err != nil {
			msg = ev.Err.Error()
		}
		s.writeText(MakeError(msg, ev.Fatal))
		if ev.Fatal {
			s.teardown("realtime-error")
		}
	case realtime.EventSessionClose:
		if !s.closed.Load() {
			s.teardown("realtime-session-close")
		}
	}
}

func (s *wsSession) handleAudioRealtime(ctx context.Context, frame []byte) {
	if s.rtAgent == nil || !s.listening.Load() {
		return
	}
	// Block uplink only during assistant text streaming (before audio). Once TTS
	// starts, mic is forwarded for server-side VAD barge-in.
	if s.rtReplyBusy.Load() {
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
	if s.rtInSR > 0 && s.inSR > 0 && s.rtInSR != s.inSR {
		resampled, err := media.ResamplePCM(pcm, s.inSR, s.rtInSR)
		if err != nil {
			return
		}
		pcm = resampled
	}
	_ = s.rtAgent.PushAudio(pcm)
}
