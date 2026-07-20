// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

// hotwordStage — sentinel between asrStage and llmStage that
// rewrites KindTextInterim / KindTextFinal frames using a small
// substring-replacement table. Mirrors the legacy HotwordCorrector
// behaviour so native-cascaded transcripts get the same hotword
// corrections the old AttachVoicePipeline path applied.
//
// Design notes
// ============
//
//  1. The stage is provider-neutral: it accepts a TextRewriter
//     interface so the voice wiring can pass any concrete
//     corrector (today: a hotword corrector;
//     tomorrow: a per-tenant DB-driven rewriter, an LLM-based
//     corrector, etc.). Cascaded does NOT import the voice-side
//     corrector type — keeps the engine free of transport/HTTP deps.
//
//  2. Non-text frames pass through unchanged. PCM in particular
//     MUST NOT pay any per-frame cost here, so the stage's hot
//     path is a single switch-on-Kind with no allocation.
//
//  3. Empty rewriter or empty result → original text passes
//     through. The stage never silently swallows transcripts.
//
//  4. Optional onCorrected hook fires once per non-no-op rewrite;
//     useful for observability (count "corrections applied" in
//     metrics) without forcing every tenant to subscribe.

import (
	"context"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

// TextRewriter is the slim interface hotwordStage drives. Correct
// receives one piece of ASR text (interim or final) and returns the
// corrected version. Implementations MUST be safe for concurrent
// invocation (Stage runs callbacks from multiple ASR worker
// goroutines in some configurations).
//
// Returning the input string unchanged is a valid no-op; the stage
// only records / forwards a "corrected" event when raw != corrected.
type TextRewriter interface {
	Correct(text string) string
}

// hotwordStage rewrites text frames in flight.
type hotwordStage struct {
	rewriter    TextRewriter
	onCorrected func(raw, corrected string, isFinal bool)
}

type hotwordStageOption func(*hotwordStage)

// withHotwordObserver registers a hook fired whenever a rewrite
// actually changes the text. Default nil = silent.
func withHotwordObserver(fn func(raw, corrected string, isFinal bool)) hotwordStageOption {
	return func(s *hotwordStage) { s.onCorrected = fn }
}

// NewHotwordStage is the exported entry point for callers outside this
// package (notably pkg/dialog/realtime, which reuses the same stage
// at the tail of its pipeline). Returns pipeline.Stage so callers
// don't depend on the concrete type. Pass a nil rewriter for a
// no-op stage.
func NewHotwordStage(r TextRewriter) pipeline.Stage {
	return newHotwordStage(r)
}

// newHotwordStage builds the stage. Nil rewriter → passthrough.
func newHotwordStage(r TextRewriter, opts ...hotwordStageOption) *hotwordStage {
	s := &hotwordStage{rewriter: r}
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
	return s
}

// Name implements pipeline.Stage.
func (hotwordStage) Name() string { return "hotword" }

// Run implements pipeline.Stage. Forwards every frame; rewrites
// KindTextInterim / KindTextFinal in-place when the rewriter
// changes the payload.
func (s *hotwordStage) Run(
	ctx context.Context,
	in <-chan pipeline.Frame,
	out chan<- pipeline.Frame,
	lg engine.Logger,
) error {
	defer close(out)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case f, ok := <-in:
			if !ok {
				return nil
			}
			f = s.maybeRewrite(f, lg)
			if err := sendOrCancel(ctx, out, f); err != nil {
				return err
			}
		}
	}
}

// maybeRewrite returns either the original frame (no-op fast path)
// or a copy with the corrected text applied. Always preserves all
// other Frame fields so downstream stages see identical metadata.
func (s *hotwordStage) maybeRewrite(f pipeline.Frame, lg engine.Logger) pipeline.Frame {
	if s == nil || s.rewriter == nil {
		return f
	}
	switch f.Kind {
	case pipeline.KindTextInterim, pipeline.KindTextFinal:
	default:
		return f
	}
	raw := strings.TrimSpace(f.Text)
	if raw == "" {
		return f
	}
	corrected := strings.TrimSpace(s.rewriter.Correct(f.Text))
	if corrected == "" || corrected == raw {
		return f
	}
	if s.onCorrected != nil {
		s.onCorrected(raw, corrected, f.Kind == pipeline.KindTextFinal)
	}
	if lg != nil {
		lg.Debug("hotword stage rewrite",
			engine.F("raw", raw),
			engine.F("corrected", corrected),
			engine.F("is_final", f.Kind == pipeline.KindTextFinal),
		)
	}
	out := f
	out.Text = corrected
	return out
}
