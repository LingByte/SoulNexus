package retrieve

import "context"

// Stage names for retrieval pipeline observability.
const (
	StageRetrieve = "retrieve"
	StageRerank   = "rerank"
	StageDedup    = "dedup"
)

// StageEvent captures one pipeline step for tracing/metrics.
type StageEvent struct {
	DurationMs  int64
	InputCount  int
	OutputCount int
	Metadata    map[string]string
	Err         error
}

// Observer receives retrieval lifecycle events (Langfuse, logs, etc.).
type Observer interface {
	BeginRetrieval(ctx context.Context, query string, meta map[string]string) context.Context
	RecordStage(ctx context.Context, stage string, evt StageEvent)
	EndRetrieval(ctx context.Context, docs []*Document, err error)
}

// NoopObserver discards observability events.
type NoopObserver struct{}

func (NoopObserver) BeginRetrieval(ctx context.Context, _ string, _ map[string]string) context.Context {
	return ctx
}
func (NoopObserver) RecordStage(context.Context, string, StageEvent) {}
func (NoopObserver) EndRetrieval(context.Context, []*Document, error) {}

type observerContextKey struct{}

// WithObserver attaches an observer to context for nested retriever calls.
func WithObserver(ctx context.Context, obs Observer) context.Context {
	if obs == nil {
		return ctx
	}
	return context.WithValue(ctx, observerContextKey{}, obs)
}

func observerFromContext(ctx context.Context) Observer {
	if ctx == nil {
		return NoopObserver{}
	}
	if obs, ok := ctx.Value(observerContextKey{}).(Observer); ok && obs != nil {
		return obs
	}
	return NoopObserver{}
}

func (r *StrategyRetriever) activeObserver(ctx context.Context) Observer {
	if r != nil && r.cfg.Observer != nil {
		return r.cfg.Observer
	}
	return observerFromContext(ctx)
}
