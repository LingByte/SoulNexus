// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

import (
	"context"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

// asrStub is a placeholder for the future ASR stage. Today it just
// drains incoming KindPCM frames and emits a single KindTextFinal
// "stub-transcript" turn when the input channel closes — enough for
// downstream stages (llm/tts) to fire under test without a real
// recognizer wired in.
//
// Real ASR will replace this with a streaming recognizer adapter
// (recognizer.ASRClient pushing interim + final transcripts) in a
// follow-up PR. The Stage interface contract guarantees that swap
// is local: no other stage in the chain needs to know.
type asrStub struct{}

func (asrStub) Name() string { return "asr" }

func (asrStub) Run(ctx context.Context, in <-chan pipeline.Frame, out chan<- pipeline.Frame, lg engine.Logger) error {
	defer close(out)
	frames := 0
	for {
		select {
		case <-ctx.Done():
			lg.Debug("asr stub: ctx done", engine.F("frames_in", frames))
			return ctx.Err()
		case f, ok := <-in:
			if !ok {
				lg.Debug("asr stub: input closed; emitting stub transcript",
					engine.F("frames_in", frames))
				if frames == 0 {
					// No audio at all → no transcript. Real ASR
					// behaves the same.
					return nil
				}
				select {
				case out <- pipeline.Frame{
					Kind:   pipeline.KindTextFinal,
					Text:   "stub-transcript",
					TurnID: "stub-turn-1",
				}:
				case <-ctx.Done():
					return ctx.Err()
				}
				return nil
			}
			if f.Kind == pipeline.KindPCM {
				frames++
			}
		}
	}
}

// llmStub mirrors what a streaming LLM stage will do: react to one
// KindTextFinal by emitting one or more KindAIText fragments followed
// by KindAITextDone. The stub emits a deterministic two-fragment
// response so tests can assert on the order.
type llmStub struct{}

func (llmStub) Name() string { return "llm" }

func (llmStub) Run(ctx context.Context, in <-chan pipeline.Frame, out chan<- pipeline.Frame, lg engine.Logger) error {
	defer close(out)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case f, ok := <-in:
			if !ok {
				return nil
			}
			if f.Kind != pipeline.KindTextFinal {
				// passthrough non-final frames
				if err := sendOrCancel(ctx, out, f); err != nil {
					return err
				}
				continue
			}
			lg.Info("llm stub: dispatching synthetic reply",
				engine.F("user_text", f.Text),
				engine.F("turn_id", f.TurnID))
			fragments := []string{"stub-reply-1 ", "stub-reply-2"}
			for _, frag := range fragments {
				if err := sendOrCancel(ctx, out, pipeline.Frame{
					Kind:   pipeline.KindAIText,
					Text:   frag,
					TurnID: f.TurnID,
				}); err != nil {
					return err
				}
			}
			if err := sendOrCancel(ctx, out, pipeline.Frame{
				Kind:   pipeline.KindAITextDone,
				TurnID: f.TurnID,
			}); err != nil {
				return err
			}
		}
	}
}

// ttsStub absorbs AI text frames and produces nothing (a real TTS
// stage would emit synthesised PCM here). KindAITextDone closes the
// current "speak" cycle. We keep the stage so the pipeline shape
// (ASR → LLM → TTS) is visible end-to-end; swapping to a real
// synthesizer.NewStreamingFromCredential is a follow-up PR.
type ttsStub struct{}

func (ttsStub) Name() string { return "tts" }

func (ttsStub) Run(ctx context.Context, in <-chan pipeline.Frame, out chan<- pipeline.Frame, lg engine.Logger) error {
	defer close(out)
	textFrames := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case f, ok := <-in:
			if !ok {
				lg.Debug("tts stub: input closed",
					engine.F("text_frames", textFrames))
				return nil
			}
			switch f.Kind {
			case pipeline.KindAIText:
				textFrames++
				lg.Debug("tts stub: would synthesize",
					engine.F("text", f.Text),
					engine.F("turn_id", f.TurnID))
			case pipeline.KindAITextDone:
				lg.Debug("tts stub: turn complete",
					engine.F("turn_id", f.TurnID),
					engine.F("text_frames", textFrames))
				textFrames = 0
			}
			// No output emission (real TTS would emit KindPCM here).
			// We pass control frames through so any future post-TTS
			// stage (e.g. recorder) can see them.
			if f.IsControl() {
				if err := sendOrCancel(ctx, out, f); err != nil {
					return err
				}
			}
		}
	}
}

// sendOrCancel writes f to out unless ctx is cancelled first. Hoisted
// because three stages need the same select pattern.
func sendOrCancel(ctx context.Context, out chan<- pipeline.Frame, f pipeline.Frame) error {
	select {
	case out <- f:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// defaultStages returns the canonical Stage list for the cascaded
// pipeline (ASR → LLM → TTS). Exposed package-private for tests and
// for the engine to use when constructing its Pipeline.
func defaultStages() []pipeline.Stage {
	return []pipeline.Stage{asrStub{}, llmStub{}, ttsStub{}}
}
