// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

// llmStage — real LLM as a pipeline.Stage.
//
// Replaces llmStub when WithLLMService is set on the Engine. Sits
// between the ASR stage (KindTextFinal / KindTextInterim) and the TTS
// stage (KindAIText deltas + KindAITextDone end-of-turn marker).
//
// Design constraints
// ==================
//
//  1. NO upward import to pkg/llm. Cascaded stays vendor-neutral.
//
//  2. Primary dispatch is KindTextFinal. KindTextInterim that stays
//     unchanged for partialStableFor also speculative-dispatches so
//     LLM work overlaps ASR hangover (~200–400ms). A later Final with
//     the same text reuses the in-flight turn; a changed text cancels
//     and waits for TextFinal (speculative burned for that utterance).
//
//  3. One generation per turn, serialised. A new dispatch cancels the
//     in-flight ctx. Speculative + Final with identical text does not
//     restart.
//
//  4. LLM errors are observability-only — emit KindAITextDone and wait
//     for the next turn; the pipeline does not abort.

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
)

const (
	// Speculative LLM waits for a short pause that still fits inside
	// typical ASR endWindow (~300ms). 500ms never fired before TextFinal
	// on Volc; burn-after-revoke prevents cancel storms if text grows.
	defaultPartialStableFor = 220 * time.Millisecond
	// Ignore 1–5 rune fragments ("自我", "呃我") that almost always grow.
	defaultPartialMinRunes = 6
)

// looksIncompleteASRPartial rejects mid-phrase partials that are stable only
// because the user is still inhaling between syllables (e.g. 「…上午到下」
// before 「下午」). Speculating on these burns a LLM call and forces
// TextFinal-only for the rest of the utterance.
func looksIncompleteASRPartial(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}
	for _, b := range []string{
		"到下", "到上", "到中", "的一", "了一", "在一", "有一", "是一", "要一",
		"上午到", "下午到",
	} {
		if strings.HasSuffix(text, b) {
			return true
		}
	}
	runes := []rune(text)
	switch runes[len(runes)-1] {
	case '的', '地', '得', '和', '与', '或', '把', '被', '从', '向', '对', '给', '让',
		'到', '在', '是', '有', '要', '会', '能', '想', '一':
		return true
	}
	return false
}

// LLMService is the slim interface llmStage drives.
type LLMService interface {
	StreamReply(ctx context.Context, userText string, onDelta func(text string, isComplete bool) error) (full string, err error)
}

type llmStage struct {
	svc       LLMService
	bufSize   int
	onErrTurn func(turnText string, err error)

	// Latency features (zero = use defaults; negative duration = disable).
	partialStableFor time.Duration
	partialMinRunes  int
}

type llmStageOption func(*llmStage)

func withLLMDeltaBuffer(n int) llmStageOption {
	return func(s *llmStage) {
		if n > 0 {
			s.bufSize = n
		}
	}
}

func withLLMTurnErrorObserver(fn func(turnText string, err error)) llmStageOption {
	return func(s *llmStage) { s.onErrTurn = fn }
}

// withLLMPartialStable configures speculative dispatch on stable ASR partials.
// d < 0 disables; d == 0 keeps the default.
func withLLMPartialStable(d time.Duration, minRunes int) llmStageOption {
	return func(s *llmStage) {
		s.partialStableFor = d
		if minRunes > 0 {
			s.partialMinRunes = minRunes
		}
	}
}

func newLLMStage(svc LLMService, opts ...llmStageOption) *llmStage {
	s := &llmStage{
		svc:              svc,
		bufSize:          256,
		partialStableFor: defaultPartialStableFor,
		partialMinRunes:  defaultPartialMinRunes,
	}
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
	return s
}

func (s *llmStage) effectivePartialStable() time.Duration {
	if s.partialStableFor < 0 {
		return 0
	}
	if s.partialStableFor == 0 {
		return defaultPartialStableFor
	}
	return s.partialStableFor
}

func (s *llmStage) effectivePartialMinRunes() int {
	if s.partialMinRunes > 0 {
		return s.partialMinRunes
	}
	return defaultPartialMinRunes
}

// Name implements pipeline.Stage.
func (llmStage) Name() string { return "llm" }

type llmDelta struct {
	turnID  uint64
	text    string
	isFinal bool
}

type liveTurn struct {
	text        string
	speculative bool
	dispatchAt  time.Time
	firstToken  *atomic.Bool
}

// Run implements pipeline.Stage.
func (s *llmStage) Run(ctx context.Context, in <-chan pipeline.Frame, out chan<- pipeline.Frame, lg engine.Logger) error {
	defer close(out)
	if s.svc == nil {
		lg.Debug("llm stage: nil service; passthrough mode")
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

	deltas := make(chan llmDelta, s.bufSize)

	var (
		genMu      sync.Mutex
		genCancel  context.CancelFunc
		genWG      sync.WaitGroup
		currentTID uint64
		live       *liveTurn
	)

	cancelInFlight := func() {
		genMu.Lock()
		c := genCancel
		genCancel = nil
		live = nil
		genMu.Unlock()
		if c != nil {
			c()
		}
	}

	dispatchTurn := func(parent context.Context, userText string, speculative bool) {
		userText = strings.TrimSpace(userText)
		if userText == "" {
			return
		}
		cancelInFlight()

		genCtx, cancel := context.WithCancel(parent)
		ft := &atomic.Bool{}
		lt := &liveTurn{
			text:        userText,
			speculative: speculative,
			dispatchAt:  time.Now(),
			firstToken:  ft,
		}
		genMu.Lock()
		currentTID++
		tid := currentTID
		genCancel = cancel
		live = lt
		genMu.Unlock()

		lg.Info("llm stage: dispatch turn",
			engine.F("turn_id", tid),
			engine.F("turn_text", userText),
			engine.F("runes", len([]rune(userText))),
			engine.F("speculative", speculative))

		genWG.Add(1)
		go func() {
			defer genWG.Done()
			defer cancel()
			var sawTerminal bool
			var firstTokenLogged atomic.Bool
			dispatchAt := lt.dispatchAt
			replyCtx := genCtx
			if speculative {
				replyCtx = WithSpeculativeLLM(genCtx)
			}
			full, err := s.svc.StreamReply(replyCtx, userText, func(text string, isComplete bool) error {
				if genCtx.Err() != nil {
					return genCtx.Err()
				}
				if !firstTokenLogged.Load() && strings.TrimSpace(text) != "" {
					if firstTokenLogged.CompareAndSwap(false, true) {
						ft.Store(true)
						lg.Info("llm stage: first token",
							engine.F("turn_id", tid),
							engine.F("llm_first_token_ms", time.Since(dispatchAt).Milliseconds()),
							engine.F("preview", truncateRunes(text, 48)))
					}
				}
				if isComplete {
					sawTerminal = true
				}
				select {
				case deltas <- llmDelta{turnID: tid, text: text, isFinal: isComplete}:
				case <-genCtx.Done():
					return genCtx.Err()
				}
				return nil
			})
			if !sawTerminal {
				select {
				case deltas <- llmDelta{turnID: tid, text: "", isFinal: true}:
				case <-genCtx.Done():
				}
			}
			wallMs := time.Since(dispatchAt).Milliseconds()
			canceled := errors.Is(err, context.Canceled) ||
				errors.Is(genCtx.Err(), context.Canceled) ||
				(err != nil && strings.Contains(err.Error(), "context canceled"))
			if err != nil && canceled {
				lg.Debug("llm stage: turn canceled",
					engine.F("turn_id", tid),
					engine.F("turn_text", userText),
					engine.F("llm_wall_ms", wallMs),
					engine.F("speculative", speculative))
			} else if err != nil {
				lg.Warn("llm stage: turn failed",
					engine.F("err", err.Error()),
					engine.F("turn_id", tid),
					engine.F("turn_text", userText),
					engine.F("llm_wall_ms", wallMs))
				if s.onErrTurn != nil {
					s.onErrTurn(userText, err)
				}
			} else {
				lg.Info("llm stage: turn complete",
					engine.F("turn_id", tid),
					engine.F("llm_wall_ms", wallMs),
					engine.F("reply_runes", len([]rune(full))),
					engine.F("reply_preview", truncateRunes(full, 64)))
			}
		}()
	}

	confirmOrDispatchFinal := func(parent context.Context, userText string) {
		userText = strings.TrimSpace(userText)
		if userText == "" {
			return
		}
		genMu.Lock()
		lt := live
		tid := currentTID
		same := lt != nil && lt.text == userText
		elapsed := time.Duration(0)
		if same {
			elapsed = time.Since(lt.dispatchAt)
			lt.speculative = false
		}
		genMu.Unlock()

		if same {
			lg.Info("llm stage: final confirms speculative",
				engine.F("turn_id", tid),
				engine.F("turn_text", userText),
				engine.F("elapsed_ms", elapsed.Milliseconds()))
			return
		}
		dispatchTurn(parent, userText, false)
	}

	defer func() {
		cancelInFlight()
		genWG.Wait()
	}()

	fwd := make(chan pipeline.Frame, 512)
	fwdErr := make(chan error, 1)
	var fwdWG sync.WaitGroup
	fwdWG.Add(1)
	go func() {
		defer fwdWG.Done()
		for f := range fwd {
			if err := sendOrCancel(ctx, out, f); err != nil {
				select {
				case fwdErr <- err:
				default:
				}
				return
			}
		}
	}()
	defer func() {
		close(fwd)
		fwdWG.Wait()
	}()
	forward := func(f pipeline.Frame) error {
		select {
		case fwd <- f:
			return nil
		case err := <-fwdErr:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	var (
		lastInterim        string
		stableTimer        *time.Timer
		stableCh           <-chan time.Time
		stablePending      string
		speculativeBurned  bool // after a revoked speculative, wait for TextFinal only
	)
	stopStable := func() {
		if stableTimer != nil {
			stableTimer.Stop()
			stableTimer = nil
		}
		stableCh = nil
		stablePending = ""
	}
	armStable := func(text string) {
		if speculativeBurned {
			return
		}
		d := s.effectivePartialStable()
		if d <= 0 {
			return
		}
		if len([]rune(text)) < s.effectivePartialMinRunes() {
			stopStable()
			return
		}
		if looksIncompleteASRPartial(text) {
			stopStable()
			return
		}
		stopStable()
		stablePending = text
		stableTimer = time.NewTimer(d)
		stableCh = stableTimer.C
	}

	inOpen := true
	for {
		var inCh <-chan pipeline.Frame
		if inOpen {
			inCh = in
		}
		select {
		case <-ctx.Done():
			stopStable()
			return ctx.Err()
		case err := <-fwdErr:
			stopStable()
			return err
		case <-stableCh:
			text := stablePending
			stopStable()
			if text == "" || speculativeBurned {
				continue
			}
			genMu.Lock()
			already := live != nil && live.text == text
			genMu.Unlock()
			if already {
				continue
			}
			lg.Info("llm stage: speculative from stable partial",
				engine.F("turn_text", text),
				engine.F("runes", len([]rune(text))))
			dispatchTurn(ctx, text, true)
		case f, ok := <-inCh:
			if !ok {
				inOpen = false
				stopStable()
				break
			}
			switch f.Kind {
			case pipeline.KindTextFinal:
				stopStable()
				confirmOrDispatchFinal(ctx, f.Text)
				lastInterim = ""
				speculativeBurned = false
				if err := forward(f); err != nil {
					return err
				}
			case pipeline.KindTextInterim:
				text := strings.TrimSpace(f.Text)
				if text != "" && text != lastInterim {
					lastInterim = text
					genMu.Lock()
					lt := live
					changedUnderSpec := lt != nil && lt.speculative && lt.text != text
					genMu.Unlock()
					if changedUnderSpec {
						cancelInFlight()
						speculativeBurned = true
						stopStable()
						lg.Info("llm stage: speculative burned until final",
							engine.F("from", truncateRunes(lt.text, 32)),
							engine.F("to", truncateRunes(text, 32)))
					} else {
						armStable(text)
					}
				}
				if err := forward(f); err != nil {
					return err
				}
			case pipeline.KindBargeIn:
				if err := forward(f); err != nil {
					return err
				}
			default:
				if err := forward(f); err != nil {
					return err
				}
			}
		case d := <-deltas:
			genMu.Lock()
			active := d.turnID == currentTID
			genMu.Unlock()
			if !active {
				continue
			}
			if d.isFinal {
				if t := strings.TrimSpace(d.text); t != "" {
					if err := forward(pipeline.Frame{Kind: pipeline.KindAIText, Text: t}); err != nil {
						return err
					}
				}
				if err := forward(pipeline.Frame{Kind: pipeline.KindAITextDone}); err != nil {
					return err
				}
			} else if t := d.text; t != "" {
				if err := forward(pipeline.Frame{Kind: pipeline.KindAIText, Text: t}); err != nil {
					return err
				}
			}
		}
		if !inOpen {
			genDone := make(chan struct{})
			go func() {
				genWG.Wait()
				close(genDone)
			}()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case err := <-fwdErr:
					return err
				case d := <-deltas:
					if err := s.emitDeltaForward(ctx, forward, d, &genMu, &currentTID); err != nil {
						return err
					}
				case <-genDone:
					for {
						select {
						case d := <-deltas:
							if err := s.emitDeltaForward(ctx, forward, d, &genMu, &currentTID); err != nil {
								return err
							}
						default:
							return nil
						}
					}
				}
			}
		}
	}
}

func (s *llmStage) emitDeltaForward(
	ctx context.Context,
	forward func(pipeline.Frame) error,
	d llmDelta,
	mu *sync.Mutex,
	currentTID *uint64,
) error {
	mu.Lock()
	active := d.turnID == *currentTID
	mu.Unlock()
	if !active {
		return nil
	}
	if d.isFinal {
		if t := strings.TrimSpace(d.text); t != "" {
			if err := forward(pipeline.Frame{
				Kind: pipeline.KindAIText,
				Text: t,
			}); err != nil {
				return err
			}
		}
		return forward(pipeline.Frame{
			Kind: pipeline.KindAITextDone,
		})
	}
	if t := d.text; t != "" {
		return forward(pipeline.Frame{
			Kind: pipeline.KindAIText,
			Text: t,
		})
	}
	return nil
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
