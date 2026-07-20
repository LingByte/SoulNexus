// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package realtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

// agentStage is the head Stage of the realtime pipeline. It owns one
// Agent for the entire call: caller PCM flows in via the standard
// pipeline.Frame channel; Agent events fan out as downstream
// pipeline.Frames.
//
// Lifecycle:
//
//   - Run builds the Agent via the supplied AgentBuilder, calls
//     Start, then loops on three channels (input PCM, agent events,
//     ctx).
//   - Cancellation, ctx-Done, in-channel close, and a fatal agent
//     error all converge on the same teardown path: stop pushing
//     audio, Close the agent, drain pending events, return.
//   - The stage is single-shot — building it twice is permitted but
//     concurrent Run calls on the same instance are NOT supported.
type agentStage struct {
	builder AgentBuilder

	// emittedTurnEnd guards against emitting two KindAITextDone for
	// the same turn (one from EventAssistantText{Final:true}, one
	// from EventAssistantTurnEnd). Reset to false on the first
	// frame of a new assistant turn.
	emittedTurnEnd atomic.Bool

	// activeTurn tracks whether we're inside an assistant turn for
	// the purpose of emitting a synthetic KindAITextDone on
	// EventSessionClose without an explicit turn-end event.
	activeTurn atomic.Bool
}

// newAgentStage returns a Stage wired to the given AgentBuilder.
// The Stage is named "realtime_agent" — short enough for metrics
// labels and distinct from cascaded stage names.
func newAgentStage(b AgentBuilder) *agentStage {
	return &agentStage{builder: b}
}

// Name implements pipeline.Stage.
func (s *agentStage) Name() string { return "realtime_agent" }

// Run implements pipeline.Stage. The loop body is documented
// inline; the high-level shape is:
//
//	for {
//	    select {
//	    case <-ctx.Done():       teardown
//	    case f, ok := <-in:      push pcm to agent / forward control
//	    case ev, ok := <-events: translate to Frame, send out
//	    }
//	}
//
// Errors are surfaced via the return value; pipeline.Wait gathers
// them. Non-fatal agent errors are logged and swallowed so the
// realtime path matches the cascaded path's resilience.
func (s *agentStage) Run(
	ctx context.Context,
	in <-chan pipeline.Frame,
	out chan<- pipeline.Frame,
	lg engine.Logger,
) error {
	defer close(out)
	if lg == nil {
		lg = engine.NopLogger{}
	}
	if s.builder == nil {
		return errors.New("dialog/realtime: nil AgentBuilder")
	}

	sink := newChanSink(64)
	defer sink.Close()

	agent, err := s.builder.Build(sink)
	if err != nil {
		return fmt.Errorf("dialog/realtime: build agent: %w", err)
	}
	if agent == nil {
		return errors.New("dialog/realtime: AgentBuilder returned nil agent")
	}

	// Close the agent on any exit path. Builder ownership transfers
	// to the stage at this point; no caller should hold the agent
	// past Run's return.
	closed := atomic.Bool{}
	closeAgent := func() {
		if closed.CompareAndSwap(false, true) {
			_ = agent.Close()
		}
	}
	defer closeAgent()

	if err := agent.Start(ctx); err != nil {
		return fmt.Errorf("dialog/realtime: start agent: %w", err)
	}

	emit := func(f pipeline.Frame) bool {
		if f.EmittedAt.IsZero() {
			f.EmittedAt = time.Now()
		}
		select {
		case out <- f:
			return true
		case <-ctx.Done():
			return false
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case f, ok := <-in:
			if !ok {
				// Upstream closed — drain remaining agent events
				// up to a short deadline so a final
				// KindAITextDone doesn't get lost. Close the
				// agent first so it stops producing.
				closeAgent()
				s.drain(ctx, sink.events(), emit, 200*time.Millisecond)
				return nil
			}
			switch f.Kind {
			case pipeline.KindPCM:
				if len(f.PCM.Data) == 0 {
					continue
				}
				// MediaPort bridge rate may be 8 kHz (G.711); Omni expects 16 kHz.
				pcm := normalizePCMRate(f.PCM.Data, f.PCM.SampleRate, AgentInputRate)
				if len(pcm) == 0 {
					continue
				}
				if perr := agent.PushAudio(pcm); perr != nil {
					lg.Warn("realtime agent: push audio",
						engine.F("err", perr.Error()))
				}
			case pipeline.KindBargeIn:
				// Externally-detected barge-in (e.g. a future
				// in-engine VAD stage placed upstream). Forward
				// to the agent so it stops generating, then pass
				// the frame through unchanged for downstream
				// players to drain their queues.
				if cerr := agent.Cancel(); cerr != nil {
					lg.Debug("realtime agent: cancel on barge-in",
						engine.F("err", cerr.Error()))
				}
				if !emit(f) {
					return ctx.Err()
				}
			case pipeline.KindUserHangup:
				// Forward then teardown. Closing the agent here
				// (instead of waiting for ctx) gives the WS a
				// chance to send goodbye frames.
				if !emit(f) {
					return ctx.Err()
				}
				closeAgent()
				s.drain(ctx, sink.events(), emit, 200*time.Millisecond)
				return nil
			default:
				// Unknown / non-audio frames pass through
				// unchanged so a future composite source (e.g.
				// "play this WAV first") doesn't get filtered.
				if !emit(f) {
					return ctx.Err()
				}
			}

		case ev, ok := <-sink.events():
			if !ok {
				return nil
			}
			frames := s.translate(ev)
			if ev.Kind == EventUserSpeechStarted {
				// Server-VAD barge: cancel the model immediately (same as
				// inbound KindBargeIn). Downstream KindBargeIn flushes pacers
				// and drains transport output.
				if cerr := agent.Cancel(); cerr != nil {
					lg.Debug("realtime agent: cancel on speech started",
						engine.F("err", cerr.Error()))
				}
			}
			for _, fr := range frames {
				if !emit(fr) {
					return ctx.Err()
				}
			}
			if ev.Kind == EventError && ev.Fatal {
				return fmt.Errorf("dialog/realtime: fatal agent error: %w", ev.Err)
			}
		}
	}
}

// translate maps one Agent Event into zero or more pipeline.Frames.
// Returning a slice keeps the call site at Run a single switch arm
// even when one event fans out (e.g. final assistant text emits
// KindAIText + KindAITextDone).
func (s *agentStage) translate(ev Event) []pipeline.Frame {
	switch ev.Kind {
	case EventUserTranscript:
		text := ev.Text
		if text == "" {
			return nil
		}
		kind := pipeline.KindTextInterim
		if ev.Final {
			kind = pipeline.KindTextFinal
			s.emittedTurnEnd.Store(false)
		}
		return []pipeline.Frame{{Kind: kind, Text: text}}

	case EventUserSpeechStarted:
		return []pipeline.Frame{{Kind: pipeline.KindBargeIn}}

	case EventAssistantText:
		s.activeTurn.Store(true)
		out := make([]pipeline.Frame, 0, 2)
		if ev.Text != "" {
			out = append(out, pipeline.Frame{Kind: pipeline.KindAIText, Text: ev.Text})
		}
		if ev.Final && s.emittedTurnEnd.CompareAndSwap(false, true) {
			out = append(out, pipeline.Frame{Kind: pipeline.KindAITextDone})
			s.activeTurn.Store(false)
		}
		return out

	case EventAssistantAudio:
		if len(ev.Audio) == 0 {
			return nil
		}
		s.activeTurn.Store(true)
		return []pipeline.Frame{{
			Kind: pipeline.KindPCM,
			PCM: engine.PCMFrame{
				Data:        ev.Audio,
				SampleRate:  ev.SampleRate,
				Synthesized: true,
			},
		}}

	case EventAssistantTurnEnd:
		if s.emittedTurnEnd.CompareAndSwap(false, true) {
			s.activeTurn.Store(false)
			return []pipeline.Frame{{Kind: pipeline.KindAITextDone}}
		}
		return nil

	case EventSessionClose:
		// Synthesise a turn-end if we exited mid-turn so
		// downstream observers (persistStage etc.) flush their
		// accumulators. Real teardown happens in Run.
		if s.activeTurn.Load() && s.emittedTurnEnd.CompareAndSwap(false, true) {
			s.activeTurn.Store(false)
			return []pipeline.Frame{{Kind: pipeline.KindAITextDone}}
		}
		return nil

	case EventError:
		// Non-fatal errors don't generate frames; fatal errors
		// flush the in-flight turn (same logic as session close)
		// and Run handles the actual return.
		if ev.Fatal && s.activeTurn.Load() && s.emittedTurnEnd.CompareAndSwap(false, true) {
			s.activeTurn.Store(false)
			return []pipeline.Frame{{Kind: pipeline.KindAITextDone}}
		}
		return nil

	case EventUserSpeechEnded:
		// Server VAD silence — informational, no downstream
		// frame today. A future intent-detection stage may want
		// this; revisit then.
		return nil
	}
	return nil
}

// drain pulls remaining events from the agent for up to maxWait so a
// final assistant transcript or KindAITextDone doesn't get dropped
// when input closes mid-turn. Stops early on ctx-Done.
func (s *agentStage) drain(
	ctx context.Context,
	events <-chan Event,
	emit func(pipeline.Frame) bool,
	maxWait time.Duration,
) {
	deadline := time.NewTimer(maxWait)
	defer deadline.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-deadline.C:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			for _, fr := range s.translate(ev) {
				if !emit(fr) {
					return
				}
			}
		}
	}
}

// chanSink is the default EventSink: a buffered channel paired with
// a one-shot close. Producers call Emit; the stage reads from
// events(). Closed sinks reject new events without blocking — this
// is the "drop instead of stall" contract documented on EventSink.
type chanSink struct {
	ch     chan Event
	mu     sync.Mutex
	closed bool
}

func newChanSink(buf int) *chanSink {
	if buf <= 0 {
		buf = 1
	}
	return &chanSink{ch: make(chan Event, buf)}
}

func (s *chanSink) Emit(ev Event) bool {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return false
	}
	ch := s.ch
	s.mu.Unlock()
	select {
	case ch <- ev:
		return true
	default:
		// Buffer full — drop silently. Realtime providers emit
		// hundreds of audio chunks per turn; back-pressuring
		// them risks WS read-timeouts. Logging is the caller's
		// responsibility.
		return false
	}
}

func (s *chanSink) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.ch)
	s.mu.Unlock()
}

func (s *chanSink) events() <-chan Event { return s.ch }
