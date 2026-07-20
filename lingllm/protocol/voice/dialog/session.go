package dialog

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/lingllm/protocol/voice/asr"
	"github.com/LingByte/lingllm/protocol/voice/tts"
)

// Session is a transport-agnostic voice call session. It runs uplink ASR and
// downlink TTS, emitting events to an external dialog app and accepting commands.
type Session struct {
	cfg Config

	asrPipeline asr.Pipeline
	recognizer  *asr.RecognizerComponent
	ttsPipeline *tts.TTSPipeline
	speaker     *tts.Speaker
	gate        *asr.PlaybackGate
	filter      *asr.SentenceFilter
	denoiser    asr.Denoiser
	audioSender *tts.AudioSender
	segmenter   *tts.TextSegmenterComponent

	started atomic.Bool
	closed  atomic.Bool

	runCtx    context.Context
	runCancel context.CancelFunc

	asrFinalAt atomic.Pointer[time.Time]

	streamUtteranceID string
	streamMu          sync.Mutex

	turnMetaMu sync.Mutex
	turnMeta   map[string]*CommandMeta

	turnAnchorMu sync.Mutex
	turnAnchors  map[string]time.Time

	spokenMu   sync.Mutex
	spokenText map[string]string
}

func (s *Session) wireCallbacks() {
	s.recognizer.SetOnTranscript(func(text string, isFinal bool) {
		if s.closed.Load() {
			return
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}

		emitFinal := isFinal
		if s.filter != nil {
			delta := s.filter.Update(text, isFinal)
			if delta == "" && !isFinal {
				return
			}
			if delta != "" {
				text = delta
				// Sentence-boundary partials (e.g. trailing 。/?) are complete
				// user turns for streaming ASR vendors that never set isFinal=true
				// until the whole session ends.
				if !isFinal {
					emitFinal = true
				}
			}
		}

		ctx := s.runCtx
		if ctx == nil {
			ctx = context.Background()
		}
		s.asrPipeline.ProcessOutput(ctx, text, emitFinal)
	})

	s.recognizer.SetOnError(func(err error, fatal bool) {
		if err == nil {
			return
		}
		s.emit(Event{
			Type:    EvASRError,
			CallID:  s.cfg.CallID,
			Message: err.Error(),
			Fatal:   fatal,
		})
	})

	s.asrPipeline.SetOutputCallback(func(text string, isFinal bool) {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		evType := EvASRPartial
		if isFinal {
			evType = EvASRFinal
			now := time.Now()
			s.asrFinalAt.Store(&now)
		}
		s.emit(Event{Type: evType, CallID: s.cfg.CallID, Text: text})
	})

	s.asrPipeline.SetBargeInCallback(func() {
		if s.filter != nil {
			s.filter.Reset()
		}
		s.resetStreamSegmenter()
		s.speaker.Interrupt()
		s.emit(Event{Type: EvTTSInterrupt, CallID: s.cfg.CallID})
	})

	s.speaker.SetCallbacksWithFirstFrame(
		func(utteranceID, text string, chained bool) {
			// Skip redundant tts.started between LLM-segmented chunks so
			// transports can keep a single playback envelope (xiaozhi-style).
			if chained {
				return
			}
			s.emit(Event{
				Type:        EvTTSStarted,
				CallID:      s.cfg.CallID,
				UtteranceID: utteranceID,
			})
		},
		func(utteranceID string, ok bool, dur time.Duration, ttsFirstMs, e2eFirstMs int, moreQueued bool) {
			// Speaker reports once per utterance when all segments finish.
			if moreQueued {
				return
			}
			s.clearTurnAnchor(utteranceID)
			s.emit(Event{
				Type:        EvTTSEnded,
				CallID:      s.cfg.CallID,
				UtteranceID: utteranceID,
				OK:          ok,
			})
			if s.cfg.OnTurn != nil {
				meta := s.popTurnMeta(utteranceID)
				s.cfg.OnTurn(TurnEvent{
					UtteranceID:      utteranceID,
					LLMText:          s.popSpokenText(utteranceID),
					Meta:             meta,
					DurationMs:       int(dur.Milliseconds()),
					TTSFirstByteMs:   ttsFirstMs,
					E2EFirstByteMs:   e2eFirstMs,
					MoreSpeaksQueued: false,
					OK:               ok,
				})
			}
		},
		func(utteranceID string, ttsFirstMs, e2eFirstMs int) {
			if s.cfg.OnFirstAudio != nil {
				s.cfg.OnFirstAudio(FirstAudioEvent{
					UtteranceID:    utteranceID,
					TTSFirstByteMs: ttsFirstMs,
					E2EFirstByteMs: e2eFirstMs,
				})
			}
		},
	)
}

func (s *Session) enqueueSegment(seg tts.TextSegment) {
	text := tts.SanitizeSpeech(seg.Text)
	if text == "" {
		return
	}
	anchor := s.turnE2EAnchor(seg.PlayID)
	s.appendSpokenText(seg.PlayID, text)
	if !s.speaker.Enqueue(text, seg.PlayID, anchor) {
		s.emit(Event{
			Type:        EvTTSEnded,
			CallID:      s.cfg.CallID,
			UtteranceID: seg.PlayID,
			OK:          false,
		})
	}
}

func (s *Session) resetStreamSegmenter() {
	s.streamMu.Lock()
	s.streamUtteranceID = ""
	s.streamMu.Unlock()
	if s.segmenter != nil {
		s.segmenter.Reset()
	}
}

func (s *Session) handleStreamChunk(cmd Command, end bool) {
	if s.segmenter == nil {
		return
	}
	utter := strings.TrimSpace(cmd.UtteranceID)
	if utter == "" {
		return
	}

	s.streamMu.Lock()
	if utter != s.streamUtteranceID {
		prev := s.streamUtteranceID
		if prev != "" {
			s.speaker.Interrupt()
			s.clearTurnAnchor(prev)
		}
		if s.segmenter != nil {
			s.segmenter.Reset()
		}
		s.streamUtteranceID = utter
	}
	s.streamMu.Unlock()

	if cmd.Meta != nil {
		s.storeTurnMeta(utter, cmd.Meta)
	}
	s.segmenter.SetPlayID(utter)

	text := cmd.Text
	if text != "" {
		ctx := s.runCtx
		if ctx == nil {
			ctx = context.Background()
		}
		_, _, _ = s.segmenter.Process(ctx, text)
	}
	if end {
		s.segmenter.OnComplete()
		s.streamMu.Lock()
		s.streamUtteranceID = ""
		s.streamMu.Unlock()
	}
}

// Start activates the session and emits call.started.
func (s *Session) Start(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if !s.started.CompareAndSwap(false, true) {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.runCtx, s.runCancel = context.WithCancel(ctx)

	if s.audioSender != nil {
		if err := s.audioSender.Start(s.runCtx); err != nil {
			return err
		}
	}
	if err := s.ttsPipeline.Start(s.runCtx); err != nil {
		return err
	}
	s.ttsPipeline.Warmup(s.runCtx)
	s.speaker.Start(s.runCtx)
	if err := s.recognizer.Start(); err != nil {
		return fmt.Errorf("dialog: asr start: %w", err)
	}

	s.emit(Event{
		Type:   EvCallStarted,
		CallID: s.cfg.CallID,
		From:   s.cfg.Meta.From,
		To:     s.cfg.Meta.To,
		Codec:  s.cfg.Meta.Codec,
		PCMHz:  s.cfg.Meta.PCMHz,
	})
	return nil
}

// ProcessAudio feeds one uplink audio chunk (encoded or PCM per session config).
func (s *Session) ProcessAudio(ctx context.Context, data []byte) error {
	if s == nil || s.closed.Load() || !s.started.Load() {
		return nil
	}
	if ctx == nil {
		ctx = s.runCtx
	}
	_, err := s.asrPipeline.Process(ctx, data)
	return err
}

// PushDTMF forwards a DTMF digit as an event to the dialog app.
func (s *Session) PushDTMF(digit string, end bool) {
	s.emit(Event{
		Type:   EvDTMF,
		CallID: s.cfg.CallID,
		Digit:  digit,
		End:    end,
	})
}

// ForwardTransferRequest notifies the dialog app of a transfer request.
func (s *Session) ForwardTransferRequest(target string) {
	s.emit(Event{
		Type:   EvTransferRequest,
		CallID: s.cfg.CallID,
		Target: target,
	})
}

// HandleCommand processes a dialog-plane command.
func (s *Session) HandleCommand(cmd Command) {
	if s == nil || s.closed.Load() {
		return
	}
	if cmd.CallID != "" && cmd.CallID != s.cfg.CallID {
		return
	}

	switch cmd.Type {
	case CmdTTSSpeak:
		text := tts.SanitizeSpeech(cmd.Text)
		if text == "" {
			return
		}
		if cmd.Meta != nil {
			s.storeTurnMeta(cmd.UtteranceID, cmd.Meta)
		}
		anchor := s.turnE2EAnchor(cmd.UtteranceID)
		s.appendSpokenText(cmd.UtteranceID, text)
		if !s.speaker.Enqueue(text, cmd.UtteranceID, anchor) {
			s.emit(Event{
				Type:        EvTTSEnded,
				CallID:      s.cfg.CallID,
				UtteranceID: cmd.UtteranceID,
				OK:          false,
			})
		}
	case CmdTTSStream:
		end := cmd.StreamEnd
		s.handleStreamChunk(cmd, end)
	case CmdTTSStreamEnd:
		s.handleStreamChunk(cmd, true)
	case CmdTTSInterrupt:
		s.resetStreamSegmenter()
		s.clearAllTurnAnchors()
		s.speaker.Interrupt()
	case CmdHangup:
		reason := cmd.Reason
		if reason == "" {
			reason = "dialog-hangup"
		}
		s.Close(reason)
		if s.cfg.OnHangup != nil {
			s.cfg.OnHangup(reason)
		}
	}
}

// Close tears down the session and emits call.ended.
func (s *Session) Close(reason string) {
	if s == nil || s.closed.Swap(true) {
		return
	}
	s.emit(Event{Type: EvCallEnded, CallID: s.cfg.CallID, Reason: reason})

	s.speaker.Stop()
	if s.audioSender != nil {
		_ = s.audioSender.Close()
	}
	_ = s.ttsPipeline.Close()
	_ = s.asrPipeline.Close()
	_ = s.recognizer.Close()
	if s.gate != nil {
		s.gate.Reset()
	}
	if closer, ok := s.denoiser.(interface{ Close() error }); ok {
		_ = closer.Close()
	}

	if s.runCancel != nil {
		s.runCancel()
	}
}

// IsTTSPlaying reports whether downlink TTS is active.
func (s *Session) IsTTSPlaying() bool {
	return s.speaker != nil && s.speaker.IsPlaying()
}

func (s *Session) emit(ev Event) {
	if s.cfg.OnEvent != nil {
		s.cfg.OnEvent(ev)
	}
}

func (s *Session) storeTurnMeta(utteranceID string, meta *CommandMeta) {
	if utteranceID == "" || meta == nil {
		return
	}
	s.turnMetaMu.Lock()
	if s.turnMeta == nil {
		s.turnMeta = make(map[string]*CommandMeta)
	}
	s.turnMeta[utteranceID] = meta
	s.turnMetaMu.Unlock()
}

func (s *Session) popTurnMeta(utteranceID string) *CommandMeta {
	if utteranceID == "" {
		return nil
	}
	s.turnMetaMu.Lock()
	defer s.turnMetaMu.Unlock()
	if s.turnMeta == nil {
		return nil
	}
	meta := s.turnMeta[utteranceID]
	delete(s.turnMeta, utteranceID)
	return meta
}

func (s *Session) turnE2EAnchor(utteranceID string) *time.Time {
	if utteranceID == "" {
		return nil
	}
	s.turnAnchorMu.Lock()
	defer s.turnAnchorMu.Unlock()
	if s.turnAnchors == nil {
		s.turnAnchors = make(map[string]time.Time)
	}
	if t, ok := s.turnAnchors[utteranceID]; ok {
		tt := t
		return &tt
	}
	if t := s.asrFinalAt.Load(); t != nil {
		s.turnAnchors[utteranceID] = *t
		s.asrFinalAt.Store(nil)
		tt := *t
		return &tt
	}
	return nil
}

func (s *Session) clearTurnAnchor(utteranceID string) {
	if utteranceID == "" {
		return
	}
	s.turnAnchorMu.Lock()
	delete(s.turnAnchors, utteranceID)
	s.turnAnchorMu.Unlock()
}

func (s *Session) clearAllTurnAnchors() {
	s.turnAnchorMu.Lock()
	s.turnAnchors = make(map[string]time.Time)
	s.turnAnchorMu.Unlock()
}

func (s *Session) appendSpokenText(utteranceID, text string) {
	if utteranceID == "" || text == "" {
		return
	}
	s.spokenMu.Lock()
	if s.spokenText == nil {
		s.spokenText = make(map[string]string)
	}
	if prev := s.spokenText[utteranceID]; prev != "" {
		s.spokenText[utteranceID] = prev + text
	} else {
		s.spokenText[utteranceID] = text
	}
	s.spokenMu.Unlock()
}

func (s *Session) popSpokenText(utteranceID string) string {
	if utteranceID == "" {
		return ""
	}
	s.spokenMu.Lock()
	defer s.spokenMu.Unlock()
	if s.spokenText == nil {
		return ""
	}
	text := s.spokenText[utteranceID]
	delete(s.spokenText, utteranceID)
	return text
}
