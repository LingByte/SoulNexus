// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

// ttsStage — real TTS as a pipeline.Stage.
//
// Replaces ttsStub when WithTTSService is set on the Engine. Sits at
// the tail of the pipeline (VAD → ASR → LLM → TTS) and converts
// KindAIText deltas into KindPCM frames the engine forwards to the
// MediaPort.
//
// Design constraints
// ==================
//
//  1. NO upward import to pkg/voice/tts. Cascaded stays neutral; the
//     media adapter (PR-9d) wraps *siptts.Pipeline by translating
//     its construction-time SendPCMFrame closure into a per-Speak
//     onPCM callback. See pkg/voice/tts/pipeline.go for the
//     production-side shape.
//
//  2. Buffer-then-speak via pkg/voice/tts.Segmenter (lingllm-aligned):
//     first chunk may break on comma (≥ FirstMinRunes) or FirstMaxRunes
//     for low latency; tiny greetings like「您好，」are held so they do
//     not burn a whole TTS round-trip. Later chunks flush only on
//     sentence punctuation (or emergency RestForceMaxRunes).
//     KindAITextDone runs Complete + Finalize.
//
//  3. Barge-in handling: a KindBargeIn frame from the VAD stage
//     cancels the in-flight Speak via ctx, clears queued speak jobs,
//     and resets the sentence buffer. The engine clears ttsPlaying on
//     KindBargeIn in the output bridge (no synthetic KindAITextDone).
//
//  4. PCM emission: TTSService.Speak invokes onPCM zero or more
//     times per call. Each invocation produces one KindPCM frame
//     downstream. The stage forwards via a buffered channel back to
//     its main select loop so the synthesis goroutine never blocks
//     on out-channel send.

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
	siptts "github.com/LingByte/SoulNexus/pkg/voice/tts"
)

// TTSService is the slim interface ttsStage drives. The single Speak
// method covers any synthesizer: the media adapter for
// *siptts.Pipeline maintains the construction-time SendPCMFrame
// callback by stashing a per-Speak onPCM closure in an atomic.Value,
// so production wiring needs ~30 LOC of glue without changing the
// underlying TTS engine.
//
// Lifecycle:
//
//   - Speak is called serially from the stage Run goroutine. The
//     stage cancels via ctx before issuing a new Speak when a
//     barge-in or new turn arrives.
//
//   - onPCM is invoked from the Speak goroutine; the callback MUST
//     return ctx.Err() (or a wrapped form) when the caller cancels
//     mid-synthesis so providers can short-circuit.
//
//   - Sample rate / frame duration are decided by the implementation;
//     the stage doesn't try to reframe. Frames flow through to the
//     MediaPort which is responsible for RTP-time pacing.
type TTSService interface {
	// Speak synthesizes text and emits PCM frames via onPCM. Each
	// frame is the raw PCM payload (mono signed 16-bit LE samples)
	// at the synthesizer's configured sample rate.
	//
	// ctx cancellation: implementations MUST honour ctx and abort
	// mid-synthesis when ctx fires (returning ctx.Err() is fine —
	// the stage filters context.Canceled out of error reporting).
	Speak(ctx context.Context, text string, onPCM func(pcm []byte) error) error

	// Finalize is invoked at end-of-turn (after the last KindAIText
	// of a turn AND after KindAITextDone arrives). The expected
	// behaviour is to flush any synthesizer-internal residual audio
	// (e.g. siptts.Pipeline's sub-frame tail). Implementations
	// without residual state may make this a no-op.
	Finalize(ctx context.Context, onPCM func(pcm []byte) error) error
}

type ttsStage struct {
	svc TTSService
	// pcmBuffer caps the in-flight PCM channel size. Default 64 ≈
	// 1.3s of 20ms-frame audio buffering, well above any reasonable
	// stage drain latency.
	pcmBuffer int
	segCfg    siptts.SegmenterConfig
}

type ttsStageOption func(*ttsStage)

func withTTSPCMBuffer(n int) ttsStageOption {
	return func(s *ttsStage) {
		if n > 0 {
			s.pcmBuffer = n
		}
	}
}

func newTTSStage(svc TTSService, opts ...ttsStageOption) *ttsStage {
	s := &ttsStage{
		svc:       svc,
		pcmBuffer: 256, // ~5s of 20ms frames; keep ahead of WebRTC pacer depth
		segCfg:    siptts.PipelineSegmenterConfigFromEnv(),
	}
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
	return s
}

// Name implements pipeline.Stage.
func (ttsStage) Name() string { return "tts" }

// pcmEvent is the synthesis-goroutine → Run-loop carrier. The turnID
// lets the stage discard PCM from a barge-in-cancelled turn that
// happens to land in the buffer just after we started a new turn.
type pcmEvent struct {
	turnID   uint64
	data     []byte
	terminal bool
	// turnDone marks terminal events from a Finalize/end-of-turn job.
	turnDone bool
}

// Run implements pipeline.Stage.
func (s *ttsStage) Run(ctx context.Context, in <-chan pipeline.Frame, out chan<- pipeline.Frame, lg engine.Logger) error {
	defer close(out)
	if s.svc == nil {
		// Nil service → passthrough.
		lg.Debug("tts stage: nil service; passthrough mode")
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case f, ok := <-in:
				if !ok {
					return nil
				}
				if err := sendOrCancel(ctx, out, f); err != nil {
					return err
				}
			}
		}
	}

	pcmCh := make(chan pcmEvent, s.pcmBuffer)

	type speakJob struct {
		text     string
		turnDone bool
	}
	jobCh := make(chan speakJob, 64)

	var (
		genMu      sync.Mutex
		genCancel  context.CancelFunc
		currentTID uint64
		seg        *siptts.Segmenter
	)

	cancelInFlight := func() {
		genMu.Lock()
		c := genCancel
		genCancel = nil
		genMu.Unlock()
		if c != nil {
			c()
		}
	}

	// clearPendingJobs drops speak jobs queued by the segmenter but not yet
	// consumed by the worker. cancelInFlight() alone only aborts the job
	// currently running; leftover jobs would keep synthesizing interrupted
	// AI text and flip ttsPlaying back on, causing a second barge-in that
	// cancels the user's new LLM turn.
	clearPendingJobs := func() {
		for {
			select {
			case <-jobCh:
			default:
				return
			}
		}
	}

	// Single worker goroutine: consumes speakJobs serially so
	// Speak() and the subsequent Finalize() never race against
	// each other. cancelInFlight() cancels whatever the worker is
	// currently doing without tearing down the worker itself.
	//
	// Note: nativeCascadedTTS serializes Speak via speakMu (shared
	// Pipeline residual), so we cannot overlap segment N+1 synth
	// with N. FirstMinRunes coalesces tiny「您好，」instead.
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		for job := range jobCh {
			text := strings.TrimSpace(job.text)
			if text == "" && !job.turnDone {
				continue
			}
			genCtx, cancel := context.WithCancel(ctx)
			// Keep the same turnID across sentence segments of one LLM turn.
			// Bumping per Speak raced the Run loop: buffered PCM from segment
			// N was dropped when segment N+1 started → Linphone heard only the
			// last few words while the stereo recorder still had full audio.
			genMu.Lock()
			tid := currentTID
			genCancel = cancel
			genMu.Unlock()

			speakAt := time.Now()
			var firstPCMLogged atomic.Bool
			if text != "" {
				lg.Info("tts stage: speak start",
					engine.F("turn_id", tid),
					engine.F("text", truncateRunes(text, 64)),
					engine.F("runes", len([]rune(text))),
					engine.F("turn_done", job.turnDone))
			}

			emit := func(pcm []byte) error {
				if genCtx.Err() != nil {
					return genCtx.Err()
				}
				if text != "" && !firstPCMLogged.Load() && len(pcm) > 0 {
					if firstPCMLogged.CompareAndSwap(false, true) {
						lg.Info("tts stage: first pcm",
							engine.F("turn_id", tid),
							engine.F("tts_first_pcm_ms", time.Since(speakAt).Milliseconds()),
							engine.F("pcm_bytes", len(pcm)))
					}
				}
				cp := make([]byte, len(pcm))
				copy(cp, pcm)
				select {
				case pcmCh <- pcmEvent{turnID: tid, data: cp}:
				case <-genCtx.Done():
					return genCtx.Err()
				}
				return nil
			}
			if text != "" {
				if err := s.svc.Speak(genCtx, text, emit); err != nil && !errors.Is(err, context.Canceled) {
					lg.Warn("tts stage: speak failed",
						engine.F("err", err.Error()),
						engine.F("text", text),
						engine.F("tts_wall_ms", time.Since(speakAt).Milliseconds()))
				} else if err == nil {
					lg.Info("tts stage: speak complete",
						engine.F("turn_id", tid),
						engine.F("tts_wall_ms", time.Since(speakAt).Milliseconds()),
						engine.F("had_first_pcm", firstPCMLogged.Load()))
				} else {
					lg.Info("tts stage: speak canceled",
						engine.F("turn_id", tid),
						engine.F("tts_wall_ms", time.Since(speakAt).Milliseconds()),
						engine.F("text", truncateRunes(text, 48)))
				}
			}
			if job.turnDone {
				if err := s.svc.Finalize(genCtx, emit); err != nil && !errors.Is(err, context.Canceled) {
					lg.Warn("tts stage: finalize failed",
						engine.F("err", err.Error()))
				}
			}
			// Emit terminal regardless so the main loop knows
			// this job is done.
			select {
			case pcmCh <- pcmEvent{turnID: tid, terminal: true, turnDone: job.turnDone}:
			case <-ctx.Done():
			}
			cancel()
			genMu.Lock()
			if genCancel != nil {
				// Only clear if we're still the current owner;
				// a barge-in may have already swapped us out.
				genCancel = nil
			}
			genMu.Unlock()
		}
	}()

	dispatchSpeak := func(text string, turnDone bool) {
		job := speakJob{text: text, turnDone: turnDone}
		select {
		case jobCh <- job:
		case <-ctx.Done():
		default:
			// Worker is still synthesizing a prior segment; drop the
			// oldest queued job so a barge-in recovery cannot wedge
			// the stage with a full jobCh.
			var dropped speakJob
			select {
			case dropped = <-jobCh:
				lg.Warn("tts stage: speak job dropped (queue full)",
					engine.F("dropped_runes", len([]rune(dropped.text))),
					engine.F("dropped_preview", truncateRunes(dropped.text, 32)))
			default:
			}
			select {
			case jobCh <- job:
			case <-ctx.Done():
			}
		}
	}
	var lastSegWasFinal bool
	seg = siptts.NewSegmenter(s.segCfg, func(text string, final bool) {
		if strings.TrimSpace(text) == "" && !final {
			return
		}
		if strings.TrimSpace(text) != "" {
			lg.Info("tts stage: segment enqueue",
				engine.F("text", truncateRunes(text, 64)),
				engine.F("runes", len([]rune(text))),
				engine.F("final", final),
				engine.F("first_max_runes", s.segCfg.FirstMaxRunes))
		}
		dispatchSpeak(text, final)
		lastSegWasFinal = final
	})

	jobChClosed := false
	closeJobs := func() {
		if !jobChClosed {
			jobChClosed = true
			close(jobCh)
		}
	}
	defer func() {
		cancelInFlight()
		closeJobs()
		<-workerDone
	}()

	inOpen := true
	// pendingTurnDone tracks whether the LLM has already delivered
	// KindAITextDone but we deferred the synthesizer Finalize until
	// the residual sentence buffer was flushed.
	pendingTurnDone := false
	needNewTurnID := true

	for {
		var inCh <-chan pipeline.Frame
		if inOpen {
			inCh = in
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case f, ok := <-inCh:
			if !ok {
				inOpen = false
				break
			}
			switch f.Kind {
			case pipeline.KindAIText:
				if needNewTurnID {
					// New LLM turn — reset first-segment state so comma /
					// FirstMax early flush works again (otherwise turn 2+
					// stays in "rest" mode and waits for a full sentence).
					seg.Reset()
					genMu.Lock()
					currentTID++
					genMu.Unlock()
					needNewTurnID = false
				}
				seg.Push(f.Text)
				// passthrough so observers (UI / metrics) still see deltas
				if err := sendOrCancel(ctx, out, f); err != nil {
					return err
				}
			case pipeline.KindAITextDone:
				pendingTurnDone = true
				seg.Complete()
				if !lastSegWasFinal {
					dispatchSpeak("", true)
				}
				lastSegWasFinal = false
				needNewTurnID = true
				seg.Reset()
				// Defer KindAITextDone until synthesizer Finalize
				// completes so engine ttsPlaying and persist/UI see
				// a turn boundary after audible output starts/finishes.
			case pipeline.KindBargeIn:
				// User started talking — dump residual text + cancel
				// in-flight Speak. Forward the barge-in frame so
				// downstream observers (recorder, metrics) see it.
				seg.Reset()
				pendingTurnDone = false
				lastSegWasFinal = false
				needNewTurnID = true
				cancelInFlight()
				clearPendingJobs()
				// Invalidate any PCM still buffered from the aborted turn.
				genMu.Lock()
				currentTID++
				genMu.Unlock()
			drainPCM:
				for {
					select {
					case <-pcmCh:
					default:
						break drainPCM
					}
				}
				if err := sendOrCancel(ctx, out, f); err != nil {
					return err
				}
			default:
				if err := sendOrCancel(ctx, out, f); err != nil {
					return err
				}
			}
		case ev := <-pcmCh:
			if ev.terminal && ev.turnDone && pendingTurnDone {
				pendingTurnDone = false
				if err := sendOrCancel(ctx, out, pipeline.Frame{Kind: pipeline.KindAITextDone}); err != nil {
					return err
				}
			}
			if err := s.handlePCMEvent(ctx, out, ev, &genMu, &currentTID); err != nil {
				return err
			}
		}
		if !inOpen {
			// Drain phase: close the job queue so the worker
			// finishes pending Speak/Finalize calls, then wait
			// for it to exit while still pumping PCM frames so
			// we don't drop trailing audio.
			closeJobs()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case ev := <-pcmCh:
					if ev.terminal && ev.turnDone && pendingTurnDone {
						pendingTurnDone = false
						if err := sendOrCancel(ctx, out, pipeline.Frame{Kind: pipeline.KindAITextDone}); err != nil {
							return err
						}
					}
					if err := s.handlePCMEvent(ctx, out, ev, &genMu, &currentTID); err != nil {
						return err
					}
				case <-workerDone:
					for {
						select {
						case ev := <-pcmCh:
							if ev.terminal && ev.turnDone && pendingTurnDone {
								pendingTurnDone = false
								if err := sendOrCancel(ctx, out, pipeline.Frame{Kind: pipeline.KindAITextDone}); err != nil {
									return err
								}
							}
							if err := s.handlePCMEvent(ctx, out, ev, &genMu, &currentTID); err != nil {
								return err
							}
						default:
							if pendingTurnDone {
								pendingTurnDone = false
								if err := sendOrCancel(ctx, out, pipeline.Frame{Kind: pipeline.KindAITextDone}); err != nil {
									return err
								}
							}
							return nil
						}
					}
				}
			}
		}
	}
}

// handlePCMEvent emits one PCM frame downstream, filtering stale-turn
// payloads via turnID. terminal events are dropped (they exist only
// to signal the synthesis goroutine has finished — the engine's own
// KindAITextDone tracking is driven by the upstream pipeline frame
// passthrough, not by this signal).
func (s *ttsStage) handlePCMEvent(
	ctx context.Context,
	out chan<- pipeline.Frame,
	ev pcmEvent,
	mu *sync.Mutex,
	currentTID *uint64,
) error {
	mu.Lock()
	active := ev.turnID == *currentTID
	mu.Unlock()
	if !active {
		return nil
	}
	if ev.terminal {
		return nil
	}
	if len(ev.data) == 0 {
		return nil
	}
	return sendOrCancel(ctx, out, pipeline.Frame{
		Kind: pipeline.KindPCM,
		PCM: engine.PCMFrame{
			Data:        ev.data,
			Synthesized: true,
		},
	})
}
