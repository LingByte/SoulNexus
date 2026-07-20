package protocol

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/lingllm/metrics"
)

// StreamWriter provides a write interface for streaming data.
type StreamWriter interface {
	// Send sends a chunk to the stream.
	Send(chunk *ChatStreamChunk, err error) error
	// Close closes the stream.
	Close() error
}

// Pipe creates a linked pair of StreamReader and StreamWriter.
// The buffer size controls how many chunks can be buffered.
func Pipe(bufferSize int) (StreamReader, StreamWriter) {
	ch := make(chan *ChatStreamChunk, bufferSize)
	errCh := make(chan error, 1)

	return &pipeReader{ch: ch, errCh: errCh},
		&pipeWriter{ch: ch, errCh: errCh}
}

type pipeReader struct {
	ch      chan *ChatStreamChunk
	errCh   chan error
	closed  bool
	metrics metrics.CallMetrics
}

func (r *pipeReader) Recv() (*ChatStreamChunk, error) {
	if r.closed {
		return nil, io.EOF
	}

	select {
	case err := <-r.errCh:
		r.closed = true
		return nil, err
	case chunk, ok := <-r.ch:
		if !ok {
			r.closed = true
			select {
			case err := <-r.errCh:
				return nil, err
			default:
				return nil, io.EOF
			}
		}
		if r.metrics.FirstAt.IsZero() && chunk != nil {
			r.metrics.FirstAt = time.Now()
		}
		if chunk != nil {
			r.metrics.Chunks++
		}
		return chunk, nil
	case err := <-r.errCh:
		r.closed = true
		return nil, err
	}
}

func (r *pipeReader) Close() error {
	r.closed = true
	return nil
}

func (r *pipeReader) Metrics() metrics.CallMetrics {
	return r.metrics
}

type pipeWriter struct {
	ch     chan *ChatStreamChunk
	errCh  chan error
	closed bool
	mu     sync.Mutex
}

func (w *pipeWriter) Send(chunk *ChatStreamChunk, err error) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return io.ErrClosedPipe
	}

	if err != nil {
		w.errCh <- err
		w.closed = true
		close(w.ch)
		return nil
	}

	w.ch <- chunk
	return nil
}

func (w *pipeWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true
	close(w.ch)
	return nil
}

// StreamReaderFromArray wraps a slice as a stream.
// Useful for testing or converting static data to streams.
func StreamReaderFromArray(chunks []*ChatStreamChunk) StreamReader {
	return &arrayReader{chunks: chunks, index: 0}
}

type arrayReader struct {
	chunks  []*ChatStreamChunk
	index   int
	closed  bool
	metrics metrics.CallMetrics
}

func (r *arrayReader) Recv() (*ChatStreamChunk, error) {
	if r.closed || r.index >= len(r.chunks) {
		r.closed = true
		return nil, io.EOF
	}

	chunk := r.chunks[r.index]
	r.index++

	if r.metrics.FirstAt.IsZero() && chunk != nil {
		r.metrics.FirstAt = time.Now()
	}
	if chunk != nil {
		r.metrics.Chunks++
	}

	return chunk, nil
}

func (r *arrayReader) Close() error {
	r.closed = true
	return nil
}

func (r *arrayReader) Metrics() metrics.CallMetrics {
	return r.metrics
}

// MergeStreamReaders combines multiple streams into one.
// Reads from each stream in order until all are exhausted.
func MergeStreamReaders(readers ...StreamReader) StreamReader {
	return &mergedReader{
		readers: readers,
		current: 0,
		metrics: metrics.CallMetrics{},
	}
}

type mergedReader struct {
	readers []StreamReader
	current int
	closed  bool
	metrics metrics.CallMetrics
	mu      sync.Mutex
}

func (r *mergedReader) Recv() (*ChatStreamChunk, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed || r.current >= len(r.readers) {
		r.closed = true
		return nil, io.EOF
	}

	for r.current < len(r.readers) {
		chunk, err := r.readers[r.current].Recv()
		if err == io.EOF {
			r.current++
			continue
		}
		if err != nil {
			return nil, err
		}

		if r.metrics.FirstAt.IsZero() && chunk != nil {
			r.metrics.FirstAt = time.Now()
		}
		if chunk != nil {
			r.metrics.Chunks++
		}

		return chunk, nil
	}

	r.closed = true
	return nil, io.EOF
}

func (r *mergedReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed = true
	var lastErr error
	for _, reader := range r.readers {
		if err := reader.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (r *mergedReader) Metrics() metrics.CallMetrics {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.metrics
}

// StreamReaderWithConvert transforms stream elements.
// Return ErrNoValue from the convert function to skip an element.
var ErrNoValue = io.EOF

type convertedReader struct {
	upstream  StreamReader
	converter func(ctx context.Context, chunk *ChatStreamChunk) (*ChatStreamChunk, error)
	closed    bool
	metrics   metrics.CallMetrics
	ctx       context.Context
}

// NewConvertedReader creates a stream that transforms chunks.
func NewConvertedReader(ctx context.Context, upstream StreamReader, converter func(context.Context, *ChatStreamChunk) (*ChatStreamChunk, error)) StreamReader {
	return &convertedReader{
		upstream:  upstream,
		converter: converter,
		ctx:       ctx,
		metrics:   metrics.CallMetrics{},
	}
}

func (r *convertedReader) Recv() (*ChatStreamChunk, error) {
	if r.closed {
		return nil, io.EOF
	}

	for {
		chunk, err := r.upstream.Recv()
		if err != nil {
			r.closed = true
			return nil, err
		}

		converted, err := r.converter(r.ctx, chunk)
		if err == ErrNoValue {
			// Skip this chunk and get the next one
			continue
		}
		if err != nil {
			r.closed = true
			return nil, err
		}

		if r.metrics.FirstAt.IsZero() && converted != nil {
			r.metrics.FirstAt = time.Now()
		}
		if converted != nil {
			r.metrics.Chunks++
		}

		return converted, nil
	}
}

func (r *convertedReader) Close() error {
	r.closed = true
	return r.upstream.Close()
}

func (r *convertedReader) Metrics() metrics.CallMetrics {
	return r.metrics
}

// CollectStream reads all chunks from a stream and returns them as a single response.
func CollectStream(ctx context.Context, stream StreamReader) (*ChatResponse, error) {
	var sb strings.Builder
	var chunks int

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		sb.WriteString(chunk.Delta)
		chunks++
	}

	metrics := stream.Metrics()
	metrics.EndAt = time.Now()

	return &ChatResponse{
		Model: metrics.Model,
		Choices: []Choice{{
			Message: Message{Role: RoleAssistant, Content: sb.String()},
		}},
		Metrics: metrics,
	}, nil
}

// SourceEOF indicates that a named source has ended.
type SourceEOF struct {
	Source string
}

// MergeNamedStreamReaders combines multiple named streams.
// Emits SourceEOF when each named source ends.
func MergeNamedStreamReaders(sources map[string]StreamReader) StreamReader {
	return &namedMergedReader{
		sources: sources,
		active:  len(sources),
		metrics: metrics.CallMetrics{},
	}
}

type namedMergedReader struct {
	sources map[string]StreamReader
	active  int
	closed  bool
	metrics metrics.CallMetrics
	mu      sync.Mutex
}

func (r *namedMergedReader) Recv() (*ChatStreamChunk, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed || r.active == 0 {
		r.closed = true
		return nil, io.EOF
	}

	// Try to read from any active source
	for name, reader := range r.sources {
		if reader == nil {
			continue
		}

		chunk, err := reader.Recv()
		if err == io.EOF {
			// Source exhausted
			r.sources[name] = nil
			r.active--
			// Return a marker chunk indicating source EOF
			return &ChatStreamChunk{
				Delta: "[EOF:" + name + "]",
			}, nil
		}
		if err != nil {
			return nil, err
		}

		if r.metrics.FirstAt.IsZero() && chunk != nil {
			r.metrics.FirstAt = time.Now()
		}
		if chunk != nil {
			r.metrics.Chunks++
		}

		return chunk, nil
	}

	r.closed = true
	return nil, io.EOF
}

func (r *namedMergedReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed = true
	var lastErr error
	for _, reader := range r.sources {
		if reader != nil {
			if err := reader.Close(); err != nil {
				lastErr = err
			}
		}
	}
	return lastErr
}

func (r *namedMergedReader) Metrics() metrics.CallMetrics {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.metrics
}

// StreamTransformer transforms stream chunks.
type StreamTransformer interface {
	Transform(ctx context.Context, chunk *ChatStreamChunk) (*ChatStreamChunk, error)
}

// StreamTransformerFunc is a function that transforms stream chunks.
type StreamTransformerFunc func(ctx context.Context, chunk *ChatStreamChunk) (*ChatStreamChunk, error)

// Transform implements StreamTransformer.
func (f StreamTransformerFunc) Transform(ctx context.Context, chunk *ChatStreamChunk) (*ChatStreamChunk, error) {
	return f(ctx, chunk)
}

// TransformStream applies a transformer to each chunk in a stream.
func TransformStream(ctx context.Context, reader StreamReader, transformer StreamTransformer) StreamReader {
	return &transformedReader{
		reader:      reader,
		transformer: transformer,
		ctx:         ctx,
		metrics:     metrics.CallMetrics{},
	}
}

type transformedReader struct {
	reader      StreamReader
	transformer StreamTransformer
	ctx         context.Context
	closed      bool
	metrics     metrics.CallMetrics
	mu          sync.Mutex
}

func (r *transformedReader) Recv() (*ChatStreamChunk, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, io.EOF
	}

	chunk, err := r.reader.Recv()
	if err == io.EOF {
		r.closed = true
		return nil, io.EOF
	}
	if err != nil {
		return nil, err
	}

	transformed, err := r.transformer.Transform(r.ctx, chunk)
	if err != nil {
		return nil, err
	}

	if r.metrics.FirstAt.IsZero() && transformed != nil {
		r.metrics.FirstAt = time.Now()
	}
	if transformed != nil {
		r.metrics.Chunks++
	}

	return transformed, nil
}

func (r *transformedReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.closed = true
	return r.reader.Close()
}

func (r *transformedReader) Metrics() metrics.CallMetrics {
	r.mu.Lock()
	defer r.mu.Unlock()

	metrics := r.metrics
	if sourceMetrics := r.reader.Metrics(); sourceMetrics.Chunks > 0 {
		metrics.Model = sourceMetrics.Model
		metrics.Provider = sourceMetrics.Provider
	}
	return metrics
}
