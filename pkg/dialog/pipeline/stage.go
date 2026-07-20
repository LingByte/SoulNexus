package pipeline

import (
	"context"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
)

// Stage is one node of the pipeline. Each Stage owns one goroutine
// (started by Pipeline.Run) that reads from in, optionally produces
// frames into out, and exits when in is closed or ctx is cancelled.
//
// Implementation contract:
//
//   - Run MUST close out before returning. This is what propagates
//     EOF down the chain.
//   - Run MUST drain in (read until closed) even if it has nothing
//     to do — leaving frames in the buffer would block upstream.
//   - Run MAY be blocking inside provider calls (asr.Push / llm.Chat
//     / tts.Speak), but MUST honour ctx cancellation promptly.
//   - Run MUST NOT panic on individual frame errors; return a
//     wrapped error when terminating early.
//
// Naming: Stage names appear in metrics labels and log fields. Use
// short snake_case identifiers ("vad", "asr", "llm", "tts").
type Stage interface {
	// Name returns the stage identifier (stable; used in metrics).
	Name() string

	// Run is the goroutine entry. Returns nil on clean shutdown,
	// non-nil on early failure. The framework closes out for you
	// only on panic; in normal flow each Stage closes its own out.
	Run(ctx context.Context, in <-chan Frame, out chan<- Frame, lg engine.Logger) error
}

// StageFunc adapts a plain function to Stage, useful for ad-hoc
// transforms in tests.
type StageFunc struct {
	StageName string
	Fn        func(ctx context.Context, in <-chan Frame, out chan<- Frame, lg engine.Logger) error
}

func (s StageFunc) Name() string { return s.StageName }

func (s StageFunc) Run(ctx context.Context, in <-chan Frame, out chan<- Frame, lg engine.Logger) error {
	if s.Fn == nil {
		// Default: pass-through.
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case f, ok := <-in:
				if !ok {
					return nil
				}
				select {
				case out <- f:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
	return s.Fn(ctx, in, out, lg)
}

// Compile-time check.
var _ Stage = StageFunc{}
