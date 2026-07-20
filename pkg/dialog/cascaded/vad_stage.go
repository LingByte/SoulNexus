// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

// vadStage — barge-in detection as a pipeline.Stage.
//
// Architecture
// ============
//
// Until PR-9b, barge-in detection lived inside the legacy voice attach
// helpers (attach helpers held their
// own *sipvad.Detector and reacted to user PCM directly). That coupled
// VAD to the legacy 3-layer attach code and made the native cascaded
// engine — which routes audio through pipeline.Stage hand-offs —
// barge-in-blind.
//
// vadStage moves VAD into the stage chain. It is the first stage in
// the cascaded pipeline (VAD → ASR → LLM → TTS) so it observes the
// raw caller PCM as it enters the pipeline. The stage is pure
// observer + passthrough: every frame it receives is forwarded
// unchanged downstream.
//
// "TTS playing" state
// -------------------
//
// VAD only matters during synthesized AI playback (otherwise every
// utterance from a non-muted caller would be a "barge-in"). But the
// stage chain is linear and VAD sits BEFORE the TTS-emitting stages,
// so it can't observe AI output directly. The engine wires an
// isTTSPlaying predicate at construction time; the engine itself
// maintains the bit by inspecting the pipeline's output stream
// (KindPCM out → playing=true; KindAITextDone → playing=false).
//
// Detection action
// ----------------
//
// When CheckBargeIn fires positive AND isTTSPlaying() reports true,
// the stage:
//
//  1. Invokes the configured onBargeIn callback (typically:
//     TriggerBargeIn on the streaming port — drains queued AI PCM
//     and clears the playback state).
//  2. Emits a KindBargeIn control frame into the downstream stages.
//     Future TTS stages will observe this and abort the in-flight
//     synthesis (today's ttsStub is a no-op so this is forward-
//     compatible plumbing).
//  3. Still forwards the original PCM frame so ASR keeps seeing the
//     user's audio uninterrupted (otherwise the barge-in utterance
//     itself would be lost).
//
// Nil safety: if either detector or isTTSPlaying is nil the stage
// degrades to a transparent passthrough. This is the production-shape
// default for cascaded.New (no options) so existing tests keep
// passing without configuring VAD.

import (
	"context"
	"math"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

// BargeInDetector is the minimal interface the VAD stage needs.
// Implemented by *pkg/vad.Detector in production; tests provide
// a deterministic fake. Kept as an interface so cascaded does NOT
// import transport vad in a way that would invert layering
// the rest of the package carefully avoids).
type BargeInDetector interface {
	// CheckBargeIn returns true when pcm crosses the speech-energy
	// threshold AND synthPlaying is true. pcm is mono signed 16-bit
	// little-endian samples at the bridge sample rate.
	CheckBargeIn(pcm []byte, synthPlaying bool) bool
}

// vadStage observes incoming PCM frames, runs the detector against
// each one, and triggers the configured barge-in handler when speech
// is detected during TTS playback.
type vadStage struct {
	detector     BargeInDetector
	isTTSPlaying func() bool
	onBargeIn    func()
	bargeInGrace time.Duration
	ttsStartedAt func() time.Time
}

// newVADStage builds a VAD stage. Any nil dependency degrades to a
// passthrough — see the package doc for rationale.
func newVADStage(detector BargeInDetector, isTTSPlaying func() bool, onBargeIn func(), grace time.Duration, ttsStartedAt func() time.Time) *vadStage {
	return &vadStage{
		detector:     detector,
		isTTSPlaying: isTTSPlaying,
		onBargeIn:    onBargeIn,
		bargeInGrace: grace,
		ttsStartedAt: ttsStartedAt,
	}
}

// Name implements pipeline.Stage. The stable identifier "vad" is the
// label used in metrics and log fields throughout the engine.
func (vadStage) Name() string { return "vad" }

// Run implements pipeline.Stage. See the package-level doc for the
// observer-passthrough contract.
func (s *vadStage) Run(ctx context.Context, in <-chan pipeline.Frame, out chan<- pipeline.Frame, lg engine.Logger) error {
	defer close(out)
	// passthrough fast-path: when the stage is not actually
	// configured (no detector / no playing predicate) we still
	// shuttle frames through so removing the stage and inserting
	// this no-op is behaviourally equivalent.
	disabled := s.detector == nil || s.isTTSPlaying == nil
	if disabled {
		lg.Debug("vad stage: passthrough mode (no detector or predicate)")
	}
	// One KindBargeIn per TTS playback episode. Trailing synthesized PCM
	// can briefly re-arm ttsPlaying after the first interrupt; without
	// debouncing, downstream LLM would see a second barge-in and cancel
	// the user's new turn.
	bargeInSent := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case f, ok := <-in:
			if !ok {
				return nil
			}
			if disabled || f.Kind != pipeline.KindPCM {
				if err := sendOrCancel(ctx, out, f); err != nil {
					return err
				}
				continue
			}
			playing := s.isTTSPlaying()
			if !playing {
				bargeInSent = false
			}
			allowBargeIn := playing && !bargeInSent
			if allowBargeIn && s.bargeInGrace > 0 && s.ttsStartedAt != nil {
				if started := s.ttsStartedAt(); !started.IsZero() && time.Since(started) < s.bargeInGrace {
					allowBargeIn = false
				}
			}
			if allowBargeIn && s.detector.CheckBargeIn(f.PCM.Data, true) {
				bargeInSent = true
				rms := pcmRMS16LE(f.PCM.Data)
				graceMs := int64(0)
				sinceTTSMs := int64(0)
				if s.bargeInGrace > 0 {
					graceMs = s.bargeInGrace.Milliseconds()
				}
				if s.ttsStartedAt != nil {
					if started := s.ttsStartedAt(); !started.IsZero() {
						sinceTTSMs = time.Since(started).Milliseconds()
					}
				}
				lg.Info("vad stage: barge-in detected during tts playback",
					engine.F("pcm_bytes", len(f.PCM.Data)),
					engine.F("rms", int(rms)),
					engine.F("grace_ms", graceMs),
					engine.F("ms_since_tts_start", sinceTTSMs))
				if s.onBargeIn != nil {
					// Run the user-supplied callback inline. The
					// callback is expected to be cheap (drains a
					// queue, flips an atomic); long-running work
					// belongs in a separate goroutine.
					s.onBargeIn()
				}
				if err := sendOrCancel(ctx, out, pipeline.Frame{
					Kind:   pipeline.KindBargeIn,
					TurnID: f.TurnID,
				}); err != nil {
					return err
				}
			}
			// Always pass the PCM through — ASR must continue to see
			// the caller's audio even during a barge-in (the barge-in
			// utterance IS the next user turn).
			if err := sendOrCancel(ctx, out, f); err != nil {
				return err
			}
		}
	}
}

func pcmRMS16LE(pcm []byte) float64 {
	if len(pcm) < 2 {
		return 0
	}
	var sum float64
	n := 0
	for i := 0; i+1 < len(pcm); i += 2 {
		s := int16(pcm[i]) | int16(pcm[i+1])<<8
		sum += float64(s) * float64(s)
		n++
	}
	if n == 0 {
		return 0
	}
	return math.Sqrt(sum / float64(n))
}
