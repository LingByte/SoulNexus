// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

// asrStage — real ASR as a pipeline.Stage.
//
// Replaces asrStub when WithASRRecognizer is set on the Engine.
//
// Design constraints
// ==================
//
//  1. NO upward import from pkg/dialog/cascaded to pkg/voice/asr or
//     pkg/recognizer. Cascaded stays transport-and-vendor-neutral so a
//     future Whisper / Azure / Google ASR can plug in via the same
//     ASRRecognizer interface without touching this package.
//     Production wiring at the voice/dialog seam (PR-9d) adapts
//     *sipasr.Pipeline → ASRRecognizer with a 30-line shim.
//
//  2. The ASR's text emission is callback-driven (the recognizer
//     decides when partials / finals fire). The Stage's Run loop is a
//     single goroutine that select-multiplexes:
//       - PCM in → ProcessPCM (synchronous enqueue inside the recogniser).
//       - transcript callback → emit KindTextInterim / KindTextFinal.
//     The callback writes into a buffered transcripts channel; the
//     Run loop drains it.
//
//  3. Errors from the recogniser are observability-only by default.
//     A fatal recogniser error does NOT abort the pipeline (downstream
//     stages may still want to react to e.g. KindBargeIn / KindUserHangup
//     control frames). Callers who want "fatal ASR ⇒ abort call"
//     supply WithASRErrorHandler returning a non-nil error.
//
//  4. The PCM frames are also forwarded downstream unchanged — future
//     stages (e.g. metrics, recorder) may want to observe them. This
//     mirrors the VAD stage's "observer + passthrough" shape.

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/callbinding"
	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

// ASRRecognizer is the slim subset of *sipasr.Pipeline (and any
// equivalent recogniser) that asrStage drives. The interface is
// callback-shaped because every production ASR streams partials +
// finals out-of-band of the audio-feed call.
//
// Lifecycle expectations:
//
//   - SetTextCallback / SetErrorCallback are called ONCE, before the
//     first ProcessPCM. The stage installs both at Run start.
//   - ProcessPCM is called serially from the stage's Run goroutine.
//     Implementations may queue internally; they MUST NOT block on
//     network IO indefinitely (a short-bounded enqueue is fine).
//   - Callbacks may fire from any goroutine. The stage tolerates
//     concurrent callback invocations.
type ASRRecognizer interface {
	// ProcessPCM feeds one chunk of mono PCM16LE samples at the
	// recogniser's expected sample rate. Returns a non-nil error
	// only on terminal failure; transient network blips should be
	// surfaced via SetErrorCallback with fatal=false.
	ProcessPCM(ctx context.Context, pcm []byte) error

	// SetTextCallback registers the partial+final transcript sink.
	// text is the cumulative or sentence-level transcript depending
	// on the recogniser; isFinal=true marks committed boundaries.
	SetTextCallback(cb func(text string, isFinal bool))

	// SetErrorCallback registers the recogniser-error sink.
	// fatal=true means the recogniser session is unusable from this
	// point; the stage logs but continues forwarding control frames.
	SetErrorCallback(cb func(err error, fatal bool))
}

// asrStage drives an ASRRecognizer from KindPCM frames and emits
// KindTextInterim / KindTextFinal frames downstream.
type asrStage struct {
	asr          ASRRecognizer
	bufSize      int
	onErrFatal   func(error)
	isTTSPlaying func() bool
	// Optional: buffer uplink PCM for on-call voiceprint identify_speaker.
	captureCallID string
	captureRate   int
}

// withASRSuppressDuringTTS drops uplink PCM to ASR while AI TTS is playing,
// unless VAD has already signalled barge-in for this utterance.
func withASRSuppressDuringTTS(fn func() bool) asrStageOption {
	return func(s *asrStage) { s.isTTSPlaying = fn }
}

// withASRUtteranceCapture stores uplink PCM on the call so identify_speaker
// can run without the LLM inventing audioBase64.
func withASRUtteranceCapture(callID string, sampleRate int) asrStageOption {
	return func(s *asrStage) {
		s.captureCallID = strings.TrimSpace(callID)
		if sampleRate > 0 {
			s.captureRate = sampleRate
		}
	}
}

// asrStageOption controls per-instance behaviour. Kept package-private
// because all current options are reachable via Engine-level Options;
// promotion to a public type happens if/when a third caller appears.
type asrStageOption func(*asrStage)

// withASRTranscriptBuffer caps the in-flight transcript queue. A
// burst of partials beyond this depth is dropped (oldest-first),
// trading partial fidelity for hard memory bounds. Default 64 →
// ~10s of 6Hz partial output before drops.
func withASRTranscriptBuffer(n int) asrStageOption {
	return func(s *asrStage) {
		if n > 0 {
			s.bufSize = n
		}
	}
}

// withASRFatalErrorHandler installs a callback invoked when the
// recogniser reports a fatal error. The default (nil) is "log and
// continue"; supply a non-nil handler that returns a non-nil error
// to make the stage abort the pipeline by returning that error from
// Run.
func withASRFatalErrorHandler(fn func(error)) asrStageOption {
	return func(s *asrStage) { s.onErrFatal = fn }
}

func newASRStage(asr ASRRecognizer, opts ...asrStageOption) *asrStage {
	s := &asrStage{
		asr:     asr,
		bufSize: 64,
	}
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
	return s
}

// Name implements pipeline.Stage.
func (asrStage) Name() string { return "asr" }

// transcriptEvent is the internal callback→Run channel payload. We
// don't reuse pipeline.Frame here because the callback may run BEFORE
// the Run loop has established sendOrCancel semantics — buffering as
// a plain struct keeps the recogniser-side hot path allocation-free
// (one struct per partial, no Kind-tag math).
type transcriptEvent struct {
	text    string
	isFinal bool
}

// Run implements pipeline.Stage.
//
// Lifecycle:
//
//	register callbacks
//	loop:
//	  ctx done            → return ctx.Err()
//	  in  → KindPCM       → ProcessPCM + passthrough
//	  in  → other Kind    → passthrough
//	  in  closed          → drain transcripts briefly, then exit
//	  cb  → transcript    → emit KindTextInterim / KindTextFinal
//	  cb  → fatal error   → optional abort via onErrFatal
//	on exit               → close out
//
// Returning ctx.Err() bubbles up to pipeline.Pipeline.Wait so callers
// can distinguish "stage finished normally" (nil) from "stage was
// torn down" (ctx.Canceled / DeadlineExceeded).
func (s *asrStage) Run(ctx context.Context, in <-chan pipeline.Frame, out chan<- pipeline.Frame, lg engine.Logger) error {
	defer close(out)
	if s.asr == nil {
		// Nil recogniser → degrade to pure passthrough so removing
		// the stage and inserting this no-op are observationally
		// equivalent. Mirrors vadStage's nil-tolerance rule.
		lg.Debug("asr stage: nil recognizer; passthrough mode")
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

	transcripts := make(chan transcriptEvent, s.bufSize)
	// Cancellation note: the callbacks are owned by the recogniser
	// goroutine which we cannot directly stop. Instead, we close a
	// local done chan on Run exit so callbacks become drop-on-the-
	// floor sends instead of leaking into a nil channel.
	var (
		closeDoneOnce sync.Once
		done          = make(chan struct{})
	)
	closeDone := func() { closeDoneOnce.Do(func() { close(done) }) }
	defer closeDone()

	s.asr.SetTextCallback(func(text string, isFinal bool) {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		// Drop-oldest overflow: prefer keeping the most recent
		// transcript (text changes constantly during a turn; the
		// final value is the most useful).
		select {
		case transcripts <- transcriptEvent{text: text, isFinal: isFinal}:
		case <-done:
		default:
			select {
			case <-transcripts: // discard oldest
			default:
			}
			select {
			case transcripts <- transcriptEvent{text: text, isFinal: isFinal}:
			case <-done:
			}
		}
	})
	s.asr.SetErrorCallback(func(err error, fatal bool) {
		if err == nil {
			return
		}
		lg.Warn("asr stage: recognizer error",
			engine.F("err", err.Error()),
			engine.F("fatal", fatal))
		if fatal && s.onErrFatal != nil {
			s.onErrFatal(err)
		}
	})

	inOpen := true
	allowASRDuringTTS := false
	var (
		utteranceStartedAt   time.Time
		lastPartialAt        time.Time
		lastPartialText      string
		lastContentChangeAt  time.Time // when partial text last grew/changed
	)
	for {
		// in == nil disables the case once input is closed.
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
				// Fall through to the post-select drain check.
				// Using `continue` here would re-enter the outer
				// loop and skip the drain entirely, hanging the
				// stage on a never-closed transcripts channel.
				break
			}
			if f.Kind == pipeline.KindBargeIn {
				allowASRDuringTTS = true
			}
			if f.Kind == pipeline.KindPCM {
				if s.isTTSPlaying != nil && !s.isTTSPlaying() {
					allowASRDuringTTS = false
				}
				skipASR := s.isTTSPlaying != nil && s.isTTSPlaying() && !allowASRDuringTTS
				if !skipASR {
					if s.captureCallID != "" && len(f.PCM.Data) > 0 {
						sr := s.captureRate
						if f.PCM.SampleRate > 0 {
							sr = f.PCM.SampleRate
						}
						callbinding.AppendUserUtterancePCM(s.captureCallID, f.PCM.Data, sr)
					}
					if err := s.asr.ProcessPCM(ctx, f.PCM.Data); err != nil {
						if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
							return err
						}
						lg.Debug("asr stage: ProcessPCM error",
							engine.F("err", err.Error()))
					}
				}
			}
			// Always passthrough (PCM observers downstream + control
			// frames from upstream e.g. KindBargeIn from VAD).
			if err := sendOrCancel(ctx, out, f); err != nil {
				return err
			}
		case t := <-transcripts:
			kind := pipeline.KindTextInterim
			now := time.Now()
			emittedAt := time.Time{}
			if !t.isFinal {
				if utteranceStartedAt.IsZero() {
					utteranceStartedAt = now
				}
				lastPartialAt = now
				trim := strings.TrimSpace(t.text)
				if trim != "" && trim != lastPartialText {
					lastContentChangeAt = now
					lastPartialText = trim
				}
				lg.Debug("asr stage: partial",
					engine.F("text", t.text),
					engine.F("ms_since_utterance_start", now.Sub(utteranceStartedAt).Milliseconds()))
			} else {
				kind = pipeline.KindTextFinal
				var sinceStart, sincePartial int64
				if !utteranceStartedAt.IsZero() {
					sinceStart = now.Sub(utteranceStartedAt).Milliseconds()
				}
				if !lastPartialAt.IsZero() {
					sincePartial = now.Sub(lastPartialAt).Milliseconds()
				}
				// Prefer last content-change as "user finished wording";
				// fall back to last partial / now for E2E anchoring.
				emittedAt = lastContentChangeAt
				if emittedAt.IsZero() {
					emittedAt = lastPartialAt
				}
				if emittedAt.IsZero() {
					emittedAt = now
				}
				lg.Info("asr stage: text final",
					engine.F("text", t.text),
					engine.F("runes", len([]rune(t.text))),
					engine.F("ms_since_utterance_start", sinceStart),
					engine.F("ms_since_last_partial", sincePartial),
					engine.F("ms_since_content_change", now.Sub(emittedAt).Milliseconds()),
					engine.F("last_partial", lastPartialText))
				utteranceStartedAt = time.Time{}
				lastPartialAt = time.Time{}
				lastPartialText = ""
				lastContentChangeAt = time.Time{}
			}
			if err := sendOrCancel(ctx, out, pipeline.Frame{
				Kind:      kind,
				Text:      t.text,
				EmittedAt: emittedAt,
			}); err != nil {
				return err
			}
		}
		// Exit condition: input closed AND no pending transcripts.
		// Drain remaining transcripts non-blockingly, then return.
		// Without this, a final transcript that landed in the buffer
		// just before in.close would be lost.
		if !inOpen {
			for {
				select {
				case t, ok := <-transcripts:
					if !ok {
						return nil
					}
					kind := pipeline.KindTextInterim
					if t.isFinal {
						kind = pipeline.KindTextFinal
					}
					if err := sendOrCancel(ctx, out, pipeline.Frame{
						Kind: kind,
						Text: t.text,
					}); err != nil {
						return err
					}
				default:
					return nil
				}
			}
		}
	}
}
