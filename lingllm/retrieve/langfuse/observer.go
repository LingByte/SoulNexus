package langfuse

import (
	"context"
	"fmt"
	"time"

	llmretrieve "github.com/LingByte/lingllm/retrieve"
)

type traceState struct {
	traceID   string
	query     string
	meta      map[string]string
	started   time.Time
	stageStart map[string]time.Time
}

type traceKey struct{}

// Observer implements retrieve.Observer and exports spans to Langfuse.
type Observer struct {
	async *AsyncClient
	name  string
}

// NewObserver builds a Langfuse retrieval observer.
func NewObserver(c *Client, traceName string) *Observer {
	if c == nil {
		return nil
	}
	if traceName == "" {
		traceName = "knowledge-retrieval"
	}
	return &Observer{
		async: NewAsync(c),
		name:  traceName,
	}
}

func (o *Observer) BeginRetrieval(ctx context.Context, query string, meta map[string]string) context.Context {
	if o == nil || o.async == nil {
		return ctx
	}
	st := &traceState{
		traceID:    newEventID(),
		query:      query,
		meta:       copyMeta(meta),
		started:    time.Now(),
		stageStart: map[string]time.Time{},
	}
	input := map[string]any{"query": query}
	md := map[string]any{}
	for k, v := range meta {
		md[k] = v
	}
	o.async.Enqueue(newTraceEvent(st.traceID, o.name, input, md))
	return context.WithValue(ctx, traceKey{}, st)
}

func (o *Observer) RecordStage(ctx context.Context, stage string, evt llmretrieve.StageEvent) {
	if o == nil || o.async == nil {
		return
	}
	st, ok := ctx.Value(traceKey{}).(*traceState)
	if !ok || st == nil {
		return
	}
	end := time.Now()
	start := end.Add(-time.Duration(evt.DurationMs) * time.Millisecond)
	md := map[string]any{
		"input_count":  evt.InputCount,
		"output_count": evt.OutputCount,
	}
	for k, v := range evt.Metadata {
		md[k] = v
	}
	o.async.Enqueue(newSpanEvent(st.traceID, stage, start, end, md, evt.Err))
}

func (o *Observer) EndRetrieval(ctx context.Context, docs []*llmretrieve.Document, err error) {
	if o == nil || o.async == nil {
		return
	}
	st, ok := ctx.Value(traceKey{}).(*traceState)
	if !ok || st == nil {
		return
	}
	output := map[string]any{
		"hit_count":   len(docs),
		"duration_ms": time.Since(st.started).Milliseconds(),
	}
	if len(docs) > 0 {
		top := make([]map[string]any, 0, min(5, len(docs)))
		for i, d := range docs {
			if i >= 5 || d == nil {
				break
			}
			item := map[string]any{
				"id":    d.ID,
				"score": d.Score,
			}
			if d.Metadata != nil {
				if v := d.Metadata["model_score"]; v != "" {
					item["model_score"] = v
				}
				if v := d.Metadata["base_score"]; v != "" {
					item["base_score"] = v
				}
				if v := d.Metadata["composite_score"]; v != "" {
					item["composite_score"] = v
				}
			}
			top = append(top, item)
		}
		output["top_hits"] = top
	}
	o.async.Enqueue(traceUpdateEvent(st.traceID, output, err))

	spanStart := st.started
	spanEnd := time.Now()
	rootMD := map[string]any{"query": st.query}
	for k, v := range st.meta {
		rootMD[k] = v
	}
	o.async.Enqueue(newSpanEvent(st.traceID, o.name, spanStart, spanEnd, rootMD, err))

	flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = o.async.Flush(flushCtx)
}

// Flush exports buffered events (for tests/shutdown).
func (o *Observer) Flush(ctx context.Context) error {
	if o == nil || o.async == nil {
		return nil
	}
	return o.async.Flush(ctx)
}

func copyMeta(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// String for debug.
func (o *Observer) String() string {
	if o == nil {
		return "langfuse-observer(nil)"
	}
	return fmt.Sprintf("langfuse-observer(%s)", o.name)
}
