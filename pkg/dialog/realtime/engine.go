// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package realtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/LingByte/SoulNexus/pkg/dialog/cascaded"
	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

// TextRewriter and TurnPersister are aliased from pkg/dialog/cascaded
// so the media adapter can build one corrector / persister and
// hand the same instance to either engine. This keeps the package
// surface small without re-declaring the contracts.
type (
	TextRewriter  = cascaded.TextRewriter
	TurnPersister = cascaded.TurnPersister
	TurnRecord    = cascaded.TurnRecord
)

// ErrAlreadyAttached is returned when Attach is invoked twice on the
// same Engine instance. One Engine serves exactly one call; reuse
// indicates a caller bug.
var ErrAlreadyAttached = errors.New("dialog/realtime: engine already attached")

// Engine is the native realtime engine. It runs a small pipeline
// rooted at agentStage:
//
//	agentStage → [hotword?] → [persist?]
//
// Stage extraction (transfer state machine, welcome gate, AI jitter
// player) lands in PR-10c+. PR-10a wires only the agentStage and
// optional hotword / persist tail so cross-mode reuse with cascaded
// is testable end-to-end.
type Engine struct {
	cfg     engine.Config
	builder AgentBuilder

	// Optional tail stages — same interfaces as cascaded so the media
	// adapter can build one TextRewriter / TurnPersister and feed
	// both engines.
	textRewriter TextRewriter
	persister    TurnPersister
	persistObs   func(pipeline.Frame)

	// pacerCfg drives the optional PCM pacer stage. Zero-value
	// PacerConfig{} is treated as "disabled"; callers opt in via
	// WithPacer. The legacy realtime path always paces, so most
	// voice wiring will pass a populated config here.
	pacerCfg    PacerConfig
	pacerActive bool

	attached   atomic.Bool
	detachOnce sync.Once

	cancel  context.CancelFunc
	done    chan struct{}
	pipeErr error
}

// Option mutates an Engine during construction.
type Option func(*Engine)

// WithTextRewriter installs a hotword corrector. Same shape as the
// cascaded engine's option so callers can pass the same instance to
// both.
func WithTextRewriter(r TextRewriter) Option {
	return func(e *Engine) { e.textRewriter = r }
}

// WithTurnPersister installs a turn observer at the tail of the
// pipeline. The persister sees one TurnRecord per completed turn.
func WithTurnPersister(p TurnPersister) Option {
	return func(e *Engine) { e.persister = p }
}

// WithPersistObserver taps every pipeline frame at the persist stage
// (live debug transcripts, metrics hooks).
func WithPersistObserver(obs func(pipeline.Frame)) Option {
	return func(e *Engine) { e.persistObs = obs }
}

// WithPacer enables the PCM pacer stage between agentStage and the
// tail stages. voice attaches MUST pass a populated config (typically
// SampleRate=bridgeRate, FrameMillis=20, PrebufferFrames=3) so the
// media RTP transport sees a steady 20 ms cadence; without pacing,
// realtime providers' burst output produces "compressed playback"
// for the caller.
func WithPacer(cfg PacerConfig) Option {
	return func(e *Engine) {
		e.pacerCfg = cfg
		e.pacerActive = true
	}
}

// New builds an Engine bound to the supplied AgentBuilder. The
// builder is invoked once at Attach time; the returned Agent owns
// the underlying transport for the duration of the call.
func New(cfg engine.Config, builder AgentBuilder, opts ...Option) *Engine {
	e := &Engine{cfg: cfg, builder: builder}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

// Mode reports the engine mode this instance was built under. Always
// engine.ModeRealtime — the native realtime engine has no
// "ModeRealtimeNative" variant because there's no legacy bridge to
// coexist with at this layer (the legacy realtime path is owned
// directly by voice, not by a dialog/engine factory).
func (e *Engine) Mode() engine.Mode {
	if e.cfg.Mode == engine.ModeRealtime {
		return engine.ModeRealtime
	}
	return engine.ModeRealtime
}

// Attach binds the engine to one MediaPort. Returns quickly; the
// pipeline runs in goroutines owned by the engine until Detach (or
// ctx cancellation) tears them down.
func (e *Engine) Attach(ctx context.Context, port engine.MediaPort, lg engine.Logger) (engine.Detach, error) {
	if !e.attached.CompareAndSwap(false, true) {
		return nil, ErrAlreadyAttached
	}
	if port == nil {
		return nil, fmt.Errorf("dialog/realtime: nil MediaPort")
	}
	if e.builder == nil {
		return nil, fmt.Errorf("dialog/realtime: nil AgentBuilder")
	}
	if lg == nil {
		lg = engine.NopLogger{}
	}
	lg = lg.With(
		engine.F("engine", "realtime"),
		engine.F("call_id", e.cfg.CallID),
		engine.F("tenant_id", e.cfg.TenantID),
	)

	// Compose stages: agentStage at the head, optional hotword /
	// persist at the tail. The agentStage emits KindTextFinal for
	// user transcripts, KindAIText/KindAITextDone for assistant
	// turns, and KindPCM for synthesised audio — exactly the frame
	// vocabulary the cascaded tail stages already consume.
	stages := []pipeline.Stage{newAgentStage(e.builder)}
	if e.pacerActive {
		// Pacer sits right after agentStage so it smooths the
		// AI-side PCM bursts before any downstream observer
		// (recorder tap, transfer audio gate, MediaPort) sees
		// the stream. Hotword / persist tail stages don't
		// consume KindPCM so their position relative to the
		// pacer doesn't matter.
		stages = append(stages, newPacerStage(e.pacerCfg))
		lg.Info("realtime engine: pacer stage enabled",
			engine.F("sample_rate", e.pacerCfg.SampleRate),
			engine.F("frame_ms", e.pacerCfg.FrameMillis),
			engine.F("prebuffer_frames", e.pacerCfg.PrebufferFrames),
		)
	}
	if e.textRewriter != nil {
		stages = append(stages, cascaded.NewHotwordStage(e.textRewriter))
		lg.Info("realtime engine: hotword stage enabled")
	}
	if e.persister != nil || e.persistObs != nil {
		stages = append(stages, cascaded.NewPersistStageWithObserver(e.persister, e.persistObs))
		lg.Info("realtime engine: persist stage enabled")
	}
	lg.Info("realtime engine: attaching", engine.F("stages", len(stages)))

	pipe, err := pipeline.New("realtime", stages)
	if err != nil {
		return nil, fmt.Errorf("dialog/realtime: build pipeline: %w", err)
	}

	engCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.done = make(chan struct{})

	// PCM-in bridge: identical shape to cascaded.Engine. Caller PCM
	// flows through the agentStage which forwards to Agent.PushAudio.
	source := make(chan pipeline.Frame, 32)
	go func() {
		defer close(source)
		in := port.InputPCM()
		for {
			select {
			case <-engCtx.Done():
				return
			case pcm, ok := <-in:
				if !ok {
					return
				}
				select {
				case source <- pipeline.Frame{Kind: pipeline.KindPCM, PCM: pcm}:
				case <-engCtx.Done():
					return
				}
			}
		}
	}()

	out, errs := pipe.Run(engCtx, source, lg)

	// PCM-out bridge: assistant audio from the agentStage flows to
	// the MediaPort. Non-PCM frames are observed-only at this layer
	// (the persist / hotword stages produce no audio).
	go func() {
		defer close(e.done)
		for {
			select {
			case <-engCtx.Done():
				e.pipeErr = pipeline.Wait(errs)
				return
			case f, ok := <-out:
				if !ok {
					e.pipeErr = pipeline.Wait(errs)
					return
				}
				if f.Kind != pipeline.KindPCM || !f.PCM.Synthesized {
					if f.Kind == pipeline.KindBargeIn {
						if d, ok := port.(interface{ TriggerBargeIn() }); ok {
							d.TriggerBargeIn()
						}
					}
					continue
				}
				if sendErr := port.SendOutputPCM(f.PCM); sendErr != nil {
					lg.Warn("realtime engine: SendOutputPCM failed; halting",
						engine.F("err", sendErr.Error()))
					cancel()
				}
			}
		}
	}()

	return e.detach, nil
}

// detach is idempotent. First call cancels the engine context and
// waits for pipeline drain; subsequent calls are instant no-ops.
func (e *Engine) detach(ctx context.Context) error {
	var err error
	e.detachOnce.Do(func() {
		if e.cancel != nil {
			e.cancel()
		}
		if e.done == nil {
			return
		}
		select {
		case <-e.done:
			err = e.pipeErr
		case <-ctx.Done():
			err = ctx.Err()
		}
	})
	return err
}
