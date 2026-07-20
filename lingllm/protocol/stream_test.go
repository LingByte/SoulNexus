package protocol

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/LingByte/lingllm/metrics"
)

func TestPipe(t *testing.T) {
	reader, writer := Pipe(2)
	defer reader.Close()

	if err := writer.Send(&ChatStreamChunk{Delta: "a"}, nil); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	chunk, err := reader.Recv()
	if err != nil || chunk.Delta != "a" {
		t.Fatalf("Recv failed: chunk=%+v err=%v", chunk, err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if err := writer.Send(&ChatStreamChunk{Delta: "b"}, nil); err != io.ErrClosedPipe {
		t.Fatalf("expected ErrClosedPipe, got %v", err)
	}
}

func TestPipeSendError(t *testing.T) {
	reader, writer := Pipe(1)
	defer reader.Close()

	sendErr := context.Canceled
	if err := writer.Send(nil, sendErr); err != nil {
		t.Fatalf("Send error failed: %v", err)
	}
	_, err := reader.Recv()
	if err != sendErr {
		t.Fatalf("expected send error, got %v", err)
	}
}

func TestStreamReaderFromArray(t *testing.T) {
	chunks := []*ChatStreamChunk{{Delta: "a"}, {Delta: "b"}}
	reader := StreamReaderFromArray(chunks)

	c1, _ := reader.Recv()
	c2, _ := reader.Recv()
	if c1.Delta != "a" || c2.Delta != "b" {
		t.Fatalf("unexpected chunks: %s %s", c1.Delta, c2.Delta)
	}
	if _, err := reader.Recv(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
	if reader.Metrics().Chunks != 2 {
		t.Errorf("expected 2 chunks in metrics, got %d", reader.Metrics().Chunks)
	}
}

func TestMergeStreamReaders(t *testing.T) {
	r1 := StreamReaderFromArray([]*ChatStreamChunk{{Delta: "a"}})
	r2 := StreamReaderFromArray([]*ChatStreamChunk{{Delta: "b"}})
	merged := MergeStreamReaders(r1, r2)

	c1, _ := merged.Recv()
	c2, _ := merged.Recv()
	if c1.Delta != "a" || c2.Delta != "b" {
		t.Fatalf("unexpected merged chunks: %s %s", c1.Delta, c2.Delta)
	}
	if err := merged.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestCollectStream(t *testing.T) {
	stream := StreamReaderFromArray([]*ChatStreamChunk{
		{Delta: "hello"},
		{Delta: " world"},
	})
	resp, err := CollectStream(context.Background(), stream)
	if err != nil {
		t.Fatalf("CollectStream failed: %v", err)
	}
	if resp.FirstContent() != "hello world" {
		t.Errorf("unexpected content: %s", resp.FirstContent())
	}
}

func TestNewConvertedReader(t *testing.T) {
	upstream := StreamReaderFromArray([]*ChatStreamChunk{
		{Delta: "skip"},
		{Delta: "keep"},
	})
	reader := NewConvertedReader(context.Background(), upstream, func(ctx context.Context, chunk *ChatStreamChunk) (*ChatStreamChunk, error) {
		if chunk.Delta == "skip" {
			return nil, ErrNoValue
		}
		return chunk, nil
	})
	chunk, err := reader.Recv()
	if err != nil || chunk.Delta != "keep" {
		t.Fatalf("converted reader failed: chunk=%+v err=%v", chunk, err)
	}
}

func TestNewConvertedReaderError(t *testing.T) {
	upstream := StreamReaderFromArray([]*ChatStreamChunk{{Delta: "x"}})
	reader := NewConvertedReader(context.Background(), upstream, func(ctx context.Context, chunk *ChatStreamChunk) (*ChatStreamChunk, error) {
		return nil, context.Canceled
	})
	_, err := reader.Recv()
	if err != context.Canceled {
		t.Fatalf("expected canceled, got %v", err)
	}
}

func TestMergeNamedStreamReaders(t *testing.T) {
	sources := map[string]StreamReader{
		"a": StreamReaderFromArray([]*ChatStreamChunk{{Delta: "1"}}),
		"b": StreamReaderFromArray([]*ChatStreamChunk{{Delta: "2"}}),
	}
	merged := MergeNamedStreamReaders(sources)

	seen := 0
	for {
		chunk, err := merged.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}
		if chunk.Delta == "" {
			t.Fatalf("empty chunk")
		}
		seen++
	}
	if seen < 2 {
		t.Fatalf("expected at least 2 chunks including EOF markers, got %d", seen)
	}
}

func TestTransformStream(t *testing.T) {
	reader := StreamReaderFromArray([]*ChatStreamChunk{{Delta: "x"}})
	out := TransformStream(context.Background(), reader, StreamTransformerFunc(func(ctx context.Context, chunk *ChatStreamChunk) (*ChatStreamChunk, error) {
		chunk.Delta = chunk.Delta + "!"
		return chunk, nil
	}))
	chunk, err := out.Recv()
	if err != nil || chunk.Delta != "x!" {
		t.Fatalf("TransformStream failed: chunk=%+v err=%v", chunk, err)
	}
	if out.Metrics().Chunks != 1 {
		t.Errorf("expected 1 chunk in metrics")
	}
}

func TestStreamTransformerFunc(t *testing.T) {
	fn := StreamTransformerFunc(func(ctx context.Context, chunk *ChatStreamChunk) (*ChatStreamChunk, error) {
		return chunk, nil
	})
	chunk, err := fn.Transform(context.Background(), &ChatStreamChunk{Delta: "ok"})
	if err != nil || chunk.Delta != "ok" {
		t.Fatalf("Transform failed: %+v err=%v", chunk, err)
	}
}

type metricsStream struct {
	m metrics.CallMetrics
}

func (s *metricsStream) Recv() (*ChatStreamChunk, error) { return nil, io.EOF }
func (s *metricsStream) Close() error                    { return nil }
func (s *metricsStream) Metrics() metrics.CallMetrics {
	return s.m
}

func TestPipeReaderMetricsFirstAt(t *testing.T) {
	reader, writer := Pipe(1)
	start := time.Now()
	_ = writer.Send(&ChatStreamChunk{Delta: "t"}, nil)
	_, _ = reader.Recv()
	if reader.Metrics().FirstAt.Before(start) {
		t.Error("expected FirstAt to be set")
	}
}

func TestSourceEOFType(t *testing.T) {
	eof := SourceEOF{Source: "test"}
	if eof.Source != "test" {
		t.Errorf("unexpected source: %s", eof.Source)
	}
}

func TestMergedReaderMetrics(t *testing.T) {
	merged := MergeStreamReaders(StreamReaderFromArray([]*ChatStreamChunk{{Delta: "a"}}))
	_, _ = merged.Recv()
	m := merged.Metrics()
	if m.Chunks != 1 {
		t.Errorf("expected 1 chunk, got %d", m.Chunks)
	}
}

func TestConvertedReaderCloseAndMetrics(t *testing.T) {
	upstream := StreamReaderFromArray([]*ChatStreamChunk{{Delta: "a"}})
	reader := NewConvertedReader(context.Background(), upstream, func(ctx context.Context, chunk *ChatStreamChunk) (*ChatStreamChunk, error) {
		return chunk, nil
	})
	_, _ = reader.Recv()
	if err := reader.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if reader.Metrics().Chunks != 1 {
		t.Errorf("expected chunk count 1")
	}
}

func TestNamedMergedReaderMetrics(t *testing.T) {
	merged := MergeNamedStreamReaders(map[string]StreamReader{
		"a": StreamReaderFromArray([]*ChatStreamChunk{{Delta: "1"}}),
	})
	_, _ = merged.Recv()
	if merged.Metrics().Chunks != 1 {
		t.Errorf("expected 1 chunk")
	}
}

func TestTransformedReaderCloseAndMetrics(t *testing.T) {
	reader := StreamReaderFromArray([]*ChatStreamChunk{{Delta: "a"}})
	out := TransformStream(context.Background(), reader, StreamTransformerFunc(func(ctx context.Context, chunk *ChatStreamChunk) (*ChatStreamChunk, error) {
		return chunk, nil
	}))
	_, _ = out.Recv()
	if err := out.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if out.Metrics().Chunks != 1 {
		t.Errorf("expected 1 chunk")
	}
}

func TestPipeWriterCloseTwice(t *testing.T) {
	_, writer := Pipe(1)
	if err := writer.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
}

func TestPipeReaderRecvAfterClose(t *testing.T) {
	reader, writer := Pipe(1)
	reader.Close()
	writer.Close()
	_, err := reader.Recv()
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestCollectStreamSetsEndAt(t *testing.T) {
	stream := &metricsStreamWithModel{chunks: []*ChatStreamChunk{{Delta: "x"}}}
	resp, err := CollectStream(context.Background(), stream)
	if err != nil || resp.FirstContent() != "x" {
		t.Fatalf("CollectStream failed: resp=%+v err=%v", resp, err)
	}
}

type metricsStreamWithModel struct {
	chunks []*ChatStreamChunk
	idx    int
}

func (s *metricsStreamWithModel) Recv() (*ChatStreamChunk, error) {
	if s.idx >= len(s.chunks) {
		return nil, io.EOF
	}
	c := s.chunks[s.idx]
	s.idx++
	return c, nil
}
func (s *metricsStreamWithModel) Close() error { return nil }
func (s *metricsStreamWithModel) Metrics() metrics.CallMetrics {
	return metrics.CallMetrics{Model: "test-model", Provider: "test"}
}

func TestCollectStreamRecvError(t *testing.T) {
	stream := &errorStream{err: context.Canceled}
	_, err := CollectStream(context.Background(), stream)
	if err != context.Canceled {
		t.Fatalf("expected canceled, got %v", err)
	}
}

type errorStream struct {
	err error
}

func (s *errorStream) Recv() (*ChatStreamChunk, error) { return nil, s.err }
func (s *errorStream) Close() error                    { return nil }
func (s *errorStream) Metrics() metrics.CallMetrics    { return metrics.CallMetrics{} }

func TestPipeReaderClosed(t *testing.T) {
	reader, _ := Pipe(1)
	reader.Close()
	_, err := reader.Recv()
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestMergeStreamReadersError(t *testing.T) {
	bad := &errorStream{err: errors.New("boom")}
	good := StreamReaderFromArray([]*ChatStreamChunk{{Delta: "a"}})
	merged := MergeStreamReaders(bad, good)
	_, err := merged.Recv()
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom error, got %v", err)
	}
}

func TestTransformStreamEOF(t *testing.T) {
	reader := StreamReaderFromArray(nil)
	out := TransformStream(context.Background(), reader, StreamTransformerFunc(func(ctx context.Context, chunk *ChatStreamChunk) (*ChatStreamChunk, error) {
		return chunk, nil
	}))
	_, err := out.Recv()
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestTransformStreamTransformError(t *testing.T) {
	reader := StreamReaderFromArray([]*ChatStreamChunk{{Delta: "x"}})
	out := TransformStream(context.Background(), reader, StreamTransformerFunc(func(ctx context.Context, chunk *ChatStreamChunk) (*ChatStreamChunk, error) {
		return nil, context.Canceled
	}))
	_, err := out.Recv()
	if err != context.Canceled {
		t.Fatalf("expected canceled, got %v", err)
	}
}

func TestNamedMergedReaderCloseError(t *testing.T) {
	merged := MergeNamedStreamReaders(map[string]StreamReader{
		"a": StreamReaderFromArray(nil),
	})
	if err := merged.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
