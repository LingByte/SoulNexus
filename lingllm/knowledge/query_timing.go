package knowledge

import "context"

type queryTimingKey struct{}

// QueryTiming accumulates embed vs vector-store latencies for one Query call.
type QueryTiming struct {
	EmbedMs  int64
	SearchMs int64
}

// WithQueryTiming attaches a timing sink to ctx for Qdrant/Milvus Query instrumentation.
func WithQueryTiming(ctx context.Context, t *QueryTiming) context.Context {
	if ctx == nil || t == nil {
		return ctx
	}
	return context.WithValue(ctx, queryTimingKey{}, t)
}

func queryTimingFromContext(ctx context.Context) *QueryTiming {
	if ctx == nil {
		return nil
	}
	t, _ := ctx.Value(queryTimingKey{}).(*QueryTiming)
	return t
}
