// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

// ErrAlreadyAttached is returned when Attach is invoked twice on the
// same Engine instance. One Engine serves exactly one call; reusing
// the same instance for a second call indicates a caller bug.
var ErrAlreadyAttached = errors.New("dialog/cascaded: engine already attached")

// Engine is the native cascaded engine. It runs a pipeline.Pipeline
// (ASR → LLM → TTS) over the audio stream surfaced by a MediaPort.
//
// PR-8c: the stage implementations are passthrough stubs; see doc.go.
// The lifecycle plumbing (attach / drain / detach / idempotency) is
// production-shaped and will not change when real providers land.
type Engine struct {
	cfg    engine.Config
	stages []pipeline.Stage

	// PR-9b — optional VAD wiring. When detector is non-nil we
	// prepend a vadStage to the chain at Attach time and the engine
	// maintains the ttsPlaying bit observed by that stage.
	vadDetector    BargeInDetector
	bargeInHandler func()
	bargeInGrace   time.Duration

	// PR-9c.1 — optional real ASR. When set, replaces the asrStub
	// in the default stage list. nil → stub remains (test-friendly
	// default).
	asrRecognizer ASRRecognizer

	// PR-9c.2 — optional real LLM. When set, replaces the llmStub.
	llmService LLMService

	// PR-9c.3 — optional real TTS. When set, replaces the ttsStub.
	ttsService TTSService

	// PR-9g — optional ASR text rewriter (hotword corrector). When
	// set, a hotwordStage is inserted between asrStage and llmStage
	// so KindTextInterim/KindTextFinal go through Correct() before
	// reaching the LLM. nil → no rewriting (default).
	textRewriter TextRewriter

	// PR-9h — optional turn persister. When set, a persistStage is
	// appended to the tail of the pipeline so completed turns get
	// observed end-to-end and handed to the persister
	// (RecordDialogTurn from the voice side, typically). nil → no
	// persistence (test default).
	turnPersister TurnPersister

	attached   atomic.Bool // single-shot Attach guard
	detachOnce sync.Once   // idempotent Detach guard

	// State filled in by Attach for Detach to consume.
	cancel  context.CancelFunc
	done    chan struct{}
	pipeErr error
}

// Option mutates an Engine during construction. Variadic options keep
// the common-case `cascaded.New(cfg)` call site clean while letting
// production wiring inject VAD / providers without growing the
// constructor signature.
type Option func(*Engine)

// WithVADDetector installs a barge-in detector. When non-nil a vadStage
// is prepended to the pipeline and the engine maintains the
// "tts playing" bit the stage queries.
//
// Pass nil to explicitly disable (default).
func WithVADDetector(d BargeInDetector) Option {
	return func(e *Engine) { e.vadDetector = d }
}

// WithBargeInHandler registers the callback invoked when vadStage
// fires positive. Typically: drain the MediaPort's queued AI PCM and
// abort in-flight TTS. The cascaded engine doesn't know how to do
// either action itself (the port owns the queue; future TTS stages
// own their own ctx) so the voice wiring supplies the closure.
//
// No-op when no detector is configured.
func WithBargeInHandler(fn func()) Option {
	return func(e *Engine) { e.bargeInHandler = fn }
}

// WithBargeInGrace suppresses VAD barge-in for a short window after TTS
// starts (matches gateway voicedialog behaviour).
func WithBargeInGrace(d time.Duration) Option {
	return func(e *Engine) { e.bargeInGrace = d }
}

// WithASRRecognizer installs a real ASR recogniser. When non-nil the
// engine swaps the default asrStub for an asrStage wrapping this
// recogniser. The recogniser's PCM sample rate must match the
// MediaPort's bridge rate (or upstream resampling must be wired in
// the adapter — see PR-9d).
//
// Pass nil to keep the stub (the default).
func WithASRRecognizer(asr ASRRecognizer) Option {
	return func(e *Engine) { e.asrRecognizer = asr }
}

// WithLLMService installs a real LLM service. When non-nil the engine
// swaps the default llmStub for an llmStage wrapping this service.
// The service is responsible for picking the right call shape per
// provider (streaming vs single-shot) — see pkg/dialog/cascaded/
// llm_stage.go for the LLMService contract.
//
// Pass nil to keep the stub.
func WithLLMService(svc LLMService) Option {
	return func(e *Engine) { e.llmService = svc }
}

// WithTTSService installs a real TTS service. When non-nil the engine
// swaps the default ttsStub for a ttsStage wrapping this service. The
// stage drives Speak/Finalize per LLM turn; see pkg/dialog/cascaded/
// tts_stage.go for the buffering / barge-in contract.
//
// Pass nil to keep the stub.
func WithTTSService(svc TTSService) Option {
	return func(e *Engine) { e.ttsService = svc }
}

// WithTextRewriter installs a hotword corrector / generic text
// rewriter. When non-nil a hotwordStage is inserted right after the
// asrStage so KindTextInterim / KindTextFinal frames flow through
// Correct() before reaching the LLM. Pass nil to disable.
func WithTextRewriter(r TextRewriter) Option {
	return func(e *Engine) { e.textRewriter = r }
}

// WithTurnPersister installs a turn observer at the tail of the
// pipeline. The persister sees one TurnRecord per completed turn
// (KindTextFinal → ... → KindAITextDone). Pass nil to disable.
func WithTurnPersister(p TurnPersister) Option {
	return func(e *Engine) { e.turnPersister = p }
}

// New builds an Engine for the supplied Config. Apply Options for
// VAD / providers / future hooks.
func New(cfg engine.Config, opts ...Option) *Engine {
	e := &Engine{
		cfg:    cfg,
		stages: defaultStages(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

// Mode reports the engine mode this instance was built under. Either
// engine.ModeCascaded (legacy-bridge replacement) or
// engine.ModeCascadedNative (parallel native path behind a feature
// flag). Falls back to ModeCascaded when cfg.Mode is unset so older
// callers that build the engine via cascaded.New(engine.Config{})
// without a Mode keep working.
func (e *Engine) Mode() engine.Mode {
	switch e.cfg.Mode {
	case engine.ModeCascaded, engine.ModeCascadedNative:
		return e.cfg.Mode
	default:
		return engine.ModeCascaded
	}
}

// Attach binds the engine to one MediaPort. Returns quickly; the
// pipeline runs in goroutines owned by the engine until Detach (or
// ctx cancellation) tears them down.
//
// Concurrency contract (matches engine.Engine):
//
//   - Attach is called once per Engine instance. Calling it twice
//     returns ErrAlreadyAttached.
//   - The returned Detach is safe to call from any goroutine and is
//     idempotent — subsequent calls are no-ops.
//   - Cancelling ctx is equivalent to invoking Detach.
func (e *Engine) Attach(ctx context.Context, port engine.MediaPort, lg engine.Logger) (engine.Detach, error) {
	if !e.attached.CompareAndSwap(false, true) {
		return nil, ErrAlreadyAttached
	}
	if port == nil {
		return nil, fmt.Errorf("dialog/cascaded: nil MediaPort")
	}
	if lg == nil {
		lg = engine.NopLogger{}
	}
	lg = lg.With(
		engine.F("engine", "cascaded"),
		engine.F("call_id", e.cfg.CallID),
		engine.F("tenant_id", e.cfg.TenantID),
	)
	lg.Info("cascaded engine: attaching",
		engine.F("stages", len(e.stages)))

	// Engine-owned context. ctx-cancel from the caller AND Detach
	// both trip this; whichever fires first wins.
	engCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.done = make(chan struct{})

	// PR-9b — assemble the active stage list. When a VAD detector is
	// configured we prepend vadStage and maintain the "tts playing"
	// bit downstream-of-engine. ttsPlaying flips:
	//   - true  on the first KindPCM frame of an AI turn (the engine
	//           sees AI-PCM in the output stream).
	//   - false on KindAITextDone (the LLM/TTS turn boundary).
	// The bit is read from inside the vadStage's Run loop so it must
	// be safe to read concurrently — atomic.Bool fits.
	var ttsPlaying atomic.Bool
	var ttsStartedAtNS atomic.Int64
	// Start from defaults and apply per-provider swaps. Working on a
	// copy keeps e.stages immutable across multiple (hypothetical)
	// Attach calls and avoids cross-Engine aliasing if Engines are
	// ever pooled.
	stages := append([]pipeline.Stage(nil), e.stages...)
	// swapByName replaces the first stage whose Name() matches; if
	// none matches, it prepends — defensive against future edits to
	// defaultStages() that might remove a slot.
	swapByName := func(name string, replacement pipeline.Stage) {
		for i, st := range stages {
			if st.Name() == name {
				stages[i] = replacement
				return
			}
		}
		stages = append([]pipeline.Stage{replacement}, stages...)
	}
	if e.asrRecognizer != nil {
		asrOpts := []asrStageOption{
			withASRSuppressDuringTTS(func() bool { return ttsPlaying.Load() }),
		}
		if cid := strings.TrimSpace(e.cfg.CallID); cid != "" {
			asrOpts = append(asrOpts, withASRUtteranceCapture(cid, port.SampleRate()))
		}
		swapByName("asr", newASRStage(e.asrRecognizer, asrOpts...))
		lg.Info("cascaded engine: asr stage replaced with real recognizer")
	}
	if e.llmService != nil {
		swapByName("llm", newLLMStage(e.llmService))
		lg.Info("cascaded engine: llm stage replaced with real service")
	}
	if e.ttsService != nil {
		swapByName("tts", newTTSStage(e.ttsService))
		lg.Info("cascaded engine: tts stage replaced with real service")
	}
	if e.textRewriter != nil {
		// Insert hotword stage right after ASR. We splice it in so
		// the stage list stays semantically: VAD? → ASR → hotword
		// → LLM → TTS.
		hs := newHotwordStage(e.textRewriter)
		out := make([]pipeline.Stage, 0, len(stages)+1)
		inserted := false
		for _, st := range stages {
			out = append(out, st)
			if !inserted && st.Name() == "asr" {
				out = append(out, hs)
				inserted = true
			}
		}
		if !inserted {
			// No asr stage somehow — push hotword to the front so
			// it still affects any text frames produced by an
			// alternative source.
			out = append([]pipeline.Stage{hs}, out...)
		}
		stages = out
		lg.Info("cascaded engine: hotword stage enabled")
	}
	if e.vadDetector != nil {
		userBargeIn := e.bargeInHandler
		vs := newVADStage(
			e.vadDetector,
			ttsPlaying.Load,
			func() {
				// Match voice attach: stop treating AI playback as active
				// immediately so VAD/ASR do not stay gated after interrupt.
				ttsPlaying.Store(false)
				ttsStartedAtNS.Store(0)
				if userBargeIn != nil {
					userBargeIn()
				}
			},
			e.bargeInGrace,
			func() time.Time {
				ns := ttsStartedAtNS.Load()
				if ns <= 0 {
					return time.Time{}
				}
				return time.Unix(0, ns)
			},
		)
		// Insert at the FRONT so VAD sees PCM before ASR consumes it.
		stages = append([]pipeline.Stage{vs}, stages...)
		lg.Info("cascaded engine: vad stage enabled", engine.F("stages", len(stages)))
	}
	if e.turnPersister != nil {
		// Append at the TAIL — persistStage is a pass-through
		// observer; placing it last guarantees it sees frames in
		// their final downstream form.
		stages = append(stages, newPersistStage(e.turnPersister, nil))
		lg.Info("cascaded engine: persist stage enabled")
	}

	pipe, err := pipeline.New("cascaded", stages, pipeline.WithChannelDepth(128))
	if err != nil {
		return nil, fmt.Errorf("dialog/cascaded: build pipeline: %w", err)
	}

	// PCM-in bridge: MediaPort.InputPCM() emits engine.PCMFrame; the
	// pipeline expects pipeline.Frame{Kind: KindPCM, PCM: ...}. We
	// wrap each frame and write into a dedicated channel that is
	// closed when the transport closes its side.
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

	// PCM-out bridge: any KindPCM frame emerging from the pipeline
	// goes back to the caller via SendOutputPCM. The stubs don't
	// emit any (TTS is a stub), but the wiring is production-shaped.
	go func() {
		defer close(e.done)
		var stageErr error
		for {
			select {
			case <-engCtx.Done():
				stageErr = pipeline.Wait(errs)
				e.pipeErr = stageErr
				return
			case f, ok := <-out:
				if !ok {
					stageErr = pipeline.Wait(errs)
					e.pipeErr = stageErr
					return
				}
				// Maintain the TTS-playing bit consumed by vadStage:
				//   - any synthesized PCM frame implies a turn is in
				//     progress.
				//   - KindAITextDone closes the turn so VAD stops
				//     gating speech as barge-in.
				// We track these BEFORE the kind filter so the bit
				// stays accurate even if non-PCM frames are passed
				// through to a future post-pipeline observer.
				switch f.Kind {
				case pipeline.KindPCM:
					if f.PCM.Synthesized {
						if !ttsPlaying.Load() {
							ttsStartedAtNS.Store(time.Now().UnixNano())
						}
						ttsPlaying.Store(true)
					}
				case pipeline.KindBargeIn, pipeline.KindAITextDone:
					ttsPlaying.Store(false)
					ttsStartedAtNS.Store(0)
				}
				if f.Kind != pipeline.KindPCM || !f.PCM.Synthesized {
					continue
				}
				if sendErr := port.SendOutputPCM(f.PCM); sendErr != nil {
					lg.Warn("cascaded engine: SendOutputPCM failed; halting",
						engine.F("err", sendErr.Error()))
					cancel()
				}
			}
		}
	}()

	return e.detach, nil
}

// detach is the engine.Detach handle returned by Attach. Idempotent
// via sync.Once; safe to call from any goroutine.
//
// Behaviour:
//
//   - First call: cancels the engine context, waits for pipeline
//     drain, returns the joined stage error (or nil).
//   - Subsequent calls: instant no-op, return nil.
//   - ctx parameter bounds how long we wait for stages to drain;
//     when ctx fires before drain completes, returns ctx.Err() (the
//     pipeline error, if any, is still recorded internally and
//     surfaces on subsequent calls).
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
