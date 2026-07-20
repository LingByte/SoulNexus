package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
)

// Pipeline wires N stages into a chain. Each adjacent pair is
// connected by a buffered channel; the Pipeline owns these channels
// and the goroutines running each stage.
type Pipeline struct {
	name      string
	stages    []Stage
	chanDepth int
}

// Option configures a Pipeline at construction time.
type Option func(*Pipeline)

// WithChannelDepth sets the buffer size of inter-stage channels.
// Default is 32. Higher values smooth bursty stages (e.g. ASR
// emitting many interim results) at the cost of memory.
func WithChannelDepth(n int) Option {
	return func(p *Pipeline) {
		if n > 0 {
			p.chanDepth = n
		}
	}
}

// New constructs a Pipeline. The stages list is run in order;
// stages[0] reads from the source, stages[N-1] writes to the sink.
// Returns an error if stages is empty.
func New(name string, stages []Stage, opts ...Option) (*Pipeline, error) {
	if len(stages) == 0 {
		return nil, errors.New("dialog/pipeline: no stages")
	}
	p := &Pipeline{
		name:      name,
		stages:    stages,
		chanDepth: 32,
	}
	for _, opt := range opts {
		opt(p)
	}
	for i, s := range stages {
		if s == nil {
			return nil, fmt.Errorf("dialog/pipeline: stage %d is nil", i)
		}
		if s.Name() == "" {
			return nil, fmt.Errorf("dialog/pipeline: stage %d has empty Name()", i)
		}
	}
	return p, nil
}

// Name returns the pipeline identifier (e.g. "cascaded", "realtime").
func (p *Pipeline) Name() string { return p.name }

// Stages returns a copy of the stage list. Safe to read after Run.
func (p *Pipeline) Stages() []Stage {
	out := make([]Stage, len(p.stages))
	copy(out, p.stages)
	return out
}

// Run starts all stages in goroutines. source feeds the first stage;
// the returned out channel emits frames from the last stage. errs
// receives one error per stage that returned non-nil; it is closed
// after every stage has finished. Callers should:
//
//	out, errs := p.Run(ctx, src)
//	for frame := range out { ... }
//	for err := range errs { if err != nil { ... } }
//
// or use Wait() helper which collects errors into one error.
func (p *Pipeline) Run(ctx context.Context, source <-chan Frame, lg engine.Logger) (<-chan Frame, <-chan error) {
	if lg == nil {
		lg = engine.NopLogger{}
	}
	depth := p.chanDepth
	chans := make([]chan Frame, len(p.stages)+1)
	chans[0] = make(chan Frame, depth)
	for i := 1; i <= len(p.stages); i++ {
		chans[i] = make(chan Frame, depth)
	}

	// Glue the external source into chans[0] so each stage owns its
	// own channels (closed when stage exits). Sourcing goroutine
	// closes chans[0] when source is exhausted or ctx is done.
	go func() {
		defer close(chans[0])
		if source == nil {
			return
		}
		for {
			select {
			case <-ctx.Done():
				return
			case f, ok := <-source:
				if !ok {
					return
				}
				select {
				case chans[0] <- f:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	errs := make(chan error, len(p.stages))
	var wg sync.WaitGroup
	wg.Add(len(p.stages))

	for i, stage := range p.stages {
		i := i
		stage := stage
		stageLg := lg.With(engine.F("pipeline", p.name), engine.F("stage", stage.Name()))
		go func() {
			defer wg.Done()
			if err := stage.Run(ctx, chans[i], chans[i+1], stageLg); err != nil {
				errs <- fmt.Errorf("stage %s: %w", stage.Name(), err)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errs)
	}()

	return chans[len(p.stages)], errs
}

// Wait collects all errors emitted by the err channel returned from
// Run. Returns nil when every stage finished cleanly, or a joined
// error containing each non-nil stage error.
func Wait(errs <-chan error) error {
	var joined []error
	for e := range errs {
		if e != nil {
			joined = append(joined, e)
		}
	}
	if len(joined) == 0 {
		return nil
	}
	return errors.Join(joined...)
}
