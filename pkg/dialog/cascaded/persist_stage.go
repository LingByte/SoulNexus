// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

// persistStage observes turn boundaries and reports completed turns
// to a TurnPersister. It is a transparent pass-through stage placed
// at the tail of the cascaded pipeline so it sees every frame the
// downstream MediaPort consumer would see, but mutates none of them.
//
// What "a completed turn" means in our pipeline
// =============================================
//
//   KindTextFinal   — user-side ASR commit. Marks the start of a new
//                     turn record (accumulator reset). The Text
//                     field is the LLM input.
//   KindAIText      — one or more assistant-side delta fragments
//                     produced by llmStage. We concatenate them
//                     into the AIText field and record the
//                     wall-time of the first one for LLMFirstMs.
//   KindAITextDone  — assistant-side end-of-turn marker. We flush
//                     the accumulator into a TurnRecord and call
//                     PersistTurn. The TurnRecord is fire-and-forget
//                     from the stage's perspective — the persister
//                     is responsible for any background dispatch.
//
// Concurrency
// ===========
//
//   - persistStage owns its accumulator strictly inside Run; no
//     external concurrent access. Frames flow through one channel.
//   - PersistTurn is invoked from the Run goroutine; the persister
//     SHOULD return quickly (e.g. push to a buffered chan or fire a
//     goroutine internally) so it doesn't stall the pipeline.

import (
	"context"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
	"github.com/LingByte/SoulNexus/pkg/utils/audutil"
)

// TurnRecord is the minimal payload the persistStage hands to the
// persister. Voice adapters translate it into the heavier
// DialogTurn / DB row shape; cascaded itself stays storage-neutral.
type TurnRecord struct {
	// UserText is the ASR-final transcript that triggered the turn.
	UserText string
	// AIText is the concatenation of every KindAIText delta in the
	// turn (LLM-side, NOT post-TTS-normalised).
	AIText string
	// LLMFirstMs is the wall-time from KindTextFinal to the first
	// KindAIText delta. Zero when no delta arrived (e.g. provider
	// error before any token).
	LLMFirstMs int
	// LLMWallMs is the wall-time from KindTextFinal to
	// KindAITextDone. Approximates QueryStream's duration; this
	// includes any TTS-side back-pressure if it stalls the
	// upstream stream.
	LLMWallMs int
	// PipelineMs equals LLMWallMs in the current Stage topology;
	// kept as a separate field because the legacy callers persist
	// both for analytics. Future TTS-aware stages can populate
	// it from the actual end-of-audio timestamp.
	PipelineMs int
	// TTSFirstByteMs is the wall-time from KindTextFinal to the
	// first synthesized KindPCM (pipeline TTS TTFB after ASR commit).
	// Zero when no PCM arrived before KindAITextDone.
	TTSFirstByteMs int
	// E2EFirstByteMs is user-perceived latency: last energetic uplink
	// speech → first synthesized PCM about to leave toward the client.
	// Includes VAD/ASR endpointing after the user stops talking.
	// Falls back to TTSFirstByteMs when speech-end could not be timed.
	E2EFirstByteMs int
	// CompletedAt is the wall-clock time KindAITextDone arrived.
	CompletedAt time.Time
}

// speechRMSFloor: PCM16 RMS below this is treated as silence / CN for
// E2E speech-end anchoring (telephony mic levels; not barge threshold).
const speechRMSFloor = 120.0

// TurnPersister sinks completed turns. nil is treated as a no-op
// stage (passthrough only). Implementations MUST NOT block the
// caller; defer DB IO to a goroutine or buffered channel inside.
type TurnPersister interface {
	PersistTurn(ctx context.Context, rec TurnRecord)
}

// TurnPersisterFunc adapts a plain function into a TurnPersister.
type TurnPersisterFunc func(ctx context.Context, rec TurnRecord)

// PersistTurn implements TurnPersister.
func (f TurnPersisterFunc) PersistTurn(ctx context.Context, rec TurnRecord) {
	if f != nil {
		f(ctx, rec)
	}
}

// persistStage is the pipeline.Stage that builds TurnRecords from
// the frame stream.
type persistStage struct {
	persister TurnPersister
	observe   func(pipeline.Frame)
	nowFn     func() time.Time // injectable for tests
}

func newPersistStage(p TurnPersister, observe func(pipeline.Frame)) *persistStage {
	return &persistStage{persister: p, observe: observe, nowFn: time.Now}
}

// NewPersistStage is the exported entry point for callers outside
// this package (notably pkg/dialog/realtime). Returns pipeline.Stage
// so callers don't depend on the concrete type. Pass a nil persister
// for a no-op observer that still passes frames through unchanged.
func NewPersistStage(p TurnPersister) pipeline.Stage {
	return newPersistStage(p, nil)
}

// NewPersistStageWithObserver is like NewPersistStage but invokes observe
// on every inbound frame before persistence bookkeeping (debug UI taps).
func NewPersistStageWithObserver(p TurnPersister, observe func(pipeline.Frame)) pipeline.Stage {
	return newPersistStage(p, observe)
}

// Name implements pipeline.Stage.
func (persistStage) Name() string { return "persist" }

// Run implements pipeline.Stage.
//
//	in  → KindTextFinal     → start accumulator
//	in  → KindAIText        → append to AIText, mark LLMFirstMs
//	in  → KindAITextDone    → flush TurnRecord
//	in  → any other frame   → passthrough
//	in  closed              → flush any in-flight accumulator then exit
//
// Always passthrough so downstream consumers (TTS, MediaPort) see
// every frame regardless of persistence wiring.
func (s *persistStage) Run(
	ctx context.Context,
	in <-chan pipeline.Frame,
	out chan<- pipeline.Frame,
	lg engine.Logger,
) error {
	defer close(out)

	type acc struct {
		userText      string
		aiText        strings.Builder
		startedAt     time.Time // KindTextFinal (ASR commit) — LLM/TTS pipeline
		speechEndedAt time.Time // last uplink speech — user-perceived E2E
		firstAIAt     time.Time
		firstPCMAt    time.Time
		active        bool
	}
	var a acc
	// lastUserSpeechAt tracks energetic uplink PCM across turns so
	// TextFinal can anchor E2E at "user finished speaking".
	var lastUserSpeechAt time.Time

	flush := func(completedAt time.Time) {
		if !a.active || s.persister == nil {
			a = acc{}
			return
		}
		rec := TurnRecord{
			UserText:    a.userText,
			AIText:      strings.TrimSpace(a.aiText.String()),
			CompletedAt: completedAt,
		}
		if !a.firstAIAt.IsZero() && !a.startedAt.IsZero() {
			rec.LLMFirstMs = int(a.firstAIAt.Sub(a.startedAt).Milliseconds())
		}
		if !a.firstPCMAt.IsZero() && !a.startedAt.IsZero() {
			rec.TTSFirstByteMs = int(a.firstPCMAt.Sub(a.startedAt).Milliseconds())
		}
		if !a.firstPCMAt.IsZero() {
			e2eStart := a.speechEndedAt
			if e2eStart.IsZero() {
				e2eStart = a.startedAt
			}
			if !e2eStart.IsZero() {
				rec.E2EFirstByteMs = int(a.firstPCMAt.Sub(e2eStart).Milliseconds())
				if rec.E2EFirstByteMs < 0 {
					rec.E2EFirstByteMs = 0
				}
			}
		}
		if rec.E2EFirstByteMs <= 0 && rec.TTSFirstByteMs > 0 {
			rec.E2EFirstByteMs = rec.TTSFirstByteMs
		}
		if !completedAt.IsZero() && !a.startedAt.IsZero() {
			rec.LLMWallMs = int(completedAt.Sub(a.startedAt).Milliseconds())
			rec.PipelineMs = rec.LLMWallMs
		}
		s.persister.PersistTurn(ctx, rec)
		a = acc{}
	}

	for {
		select {
		case <-ctx.Done():
			// Don't flush on ctx cancel — the turn was interrupted,
			// not completed. Persisters that want abort-records can
			// register their own ctx observer.
			return ctx.Err()
		case f, ok := <-in:
			if s.observe != nil {
				s.observe(f)
			}
			if !ok {
				// in closed → drain incomplete turn if anything
				// meaningful was accumulated.
				if a.active && a.aiText.Len() > 0 {
					flush(s.nowFn())
				}
				return nil
			}
			switch f.Kind {
			case pipeline.KindTextFinal:
				text := strings.TrimSpace(f.Text)
				now := s.nowFn()
				// User-perceived E2E starts when speech energy drops, not when
				// ASR emits TextFinal (that includes endWindow hangover).
				// EmittedAt (last content-change) only wins when it clearly
				// precedes final — final-time ASR corrections must not yank
				// speechEnd forward to "now" and erase hangover from the metric.
				speechEnd := lastUserSpeechAt
				if !f.EmittedAt.IsZero() {
					finalSkew := now.Sub(f.EmittedAt)
					if speechEnd.IsZero() {
						speechEnd = f.EmittedAt
					} else if finalSkew > 80*time.Millisecond && f.EmittedAt.After(speechEnd) {
						// Quiet / whispered tail: content stopped after energy.
						speechEnd = f.EmittedAt
					}
				}
				if speechEnd.IsZero() {
					speechEnd = now
				}
				if a.active && a.aiText.Len() == 0 {
					// Realtime ASR may revise the user transcript
					// (partial final → full final) before the
					// assistant replies. Update in place instead of
					// flushing a user-only phantom turn.
					a.userText = text
					if a.startedAt.IsZero() {
						a.startedAt = now
					}
					// Prefer the latest speech-end if uplink kept
					// arriving between premature and full finals.
					if speechEnd.After(a.speechEndedAt) {
						a.speechEndedAt = speechEnd
					}
					break
				}
				if a.active {
					flush(now)
				}
				a = acc{
					userText:      text,
					startedAt:     now,
					speechEndedAt: speechEnd,
					active:        true,
				}
			case pipeline.KindAIText:
				if !a.active {
					// AI delta without a preceding user-final —
					// can happen on welcome/auto-prompt paths.
					// Start an accumulator with empty userText so
					// the assistant utterance still gets recorded.
					a = acc{
						startedAt:     s.nowFn(),
						speechEndedAt: s.nowFn(),
						active:        true,
					}
				}
				if a.firstAIAt.IsZero() {
					a.firstAIAt = s.nowFn()
				}
				appendAIText(&a.aiText, f.Text)
			case pipeline.KindAITextDone:
				flush(s.nowFn())
			case pipeline.KindPCM:
				if !f.PCM.Synthesized && len(f.PCM.Data) > 0 {
					if audutil.RMSPCM16LE(f.PCM.Data) >= speechRMSFloor {
						lastUserSpeechAt = s.nowFn()
					}
				}
				// Only AI/TTS output counts toward TTS / E2E first-byte.
				// Uplink caller PCM also flows through the pipeline and
				// would otherwise stamp firstPCMAt ~immediately after
				// KindTextFinal (false "E2E 93ms").
				if a.active && a.firstPCMAt.IsZero() && f.PCM.Synthesized && len(f.PCM.Data) > 0 {
					a.firstPCMAt = s.nowFn()
				}
			}
			if err := sendOrCancel(ctx, out, f); err != nil {
				return err
			}
		}
	}
}

// appendAIText merges assistant fragments. Realtime vendors often
// emit incremental deltas followed by a final chunk that repeats the
// full cumulative transcript — concatenating blindly duplicates text.
func appendAIText(b *strings.Builder, text string) {
	if text == "" || b == nil {
		return
	}
	existing := b.String()
	if existing == "" {
		b.WriteString(text)
		return
	}
	if text == existing {
		return
	}
	trimExisting := strings.TrimSpace(existing)
	trimText := strings.TrimSpace(text)
	if trimText == "" {
		return
	}
	if trimText == trimExisting {
		return
	}
	if strings.HasPrefix(trimText, trimExisting) {
		b.Reset()
		b.WriteString(text)
		return
	}
	if strings.HasPrefix(trimExisting, trimText) {
		return
	}
	b.WriteString(text)
}
