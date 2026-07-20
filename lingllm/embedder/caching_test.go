package embedder

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type countingEmbedder struct {
	calls atomic.Int64
	dim   int
}

func (c *countingEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	c.calls.Add(1)
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = make([]float32, c.dim)
		out[i][0] = float32(len(texts[i]))
	}
	return out, nil
}

func (c *countingEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	vecs, err := c.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (c *countingEmbedder) Dimension() int  { return c.dim }
func (c *countingEmbedder) Name() string    { return "counting" }
func (c *countingEmbedder) Provider() string { return "test" }
func (c *countingEmbedder) Close() error    { return nil }

func TestCachingEmbedder_Hit(t *testing.T) {
	inner := &countingEmbedder{dim: 4}
	c := NewCachingEmbedder(inner, 16, time.Minute)
	ctx := context.Background()

	v1, err := c.Embed(ctx, []string{"云间有哪些服务"})
	if err != nil || len(v1) != 1 {
		t.Fatalf("first embed: %v %#v", err, v1)
	}
	v2, err := c.Embed(ctx, []string{"云间有哪些服务"})
	if err != nil || len(v2) != 1 {
		t.Fatalf("second embed: %v %#v", err, v2)
	}
	if inner.calls.Load() != 1 {
		t.Fatalf("expected 1 inner call, got %d", inner.calls.Load())
	}
	hits, miss := c.Stats()
	if hits < 1 || miss < 1 {
		t.Fatalf("stats hits=%d miss=%d", hits, miss)
	}
}
