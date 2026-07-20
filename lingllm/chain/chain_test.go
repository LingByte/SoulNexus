package chain

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/LingByte/lingllm/metrics"
	"github.com/LingByte/lingllm/protocol"
)

type stubModel struct {
	chatResp  *protocol.ChatResponse
	chatErr   error
	stream    protocol.ChatStream
	streamErr error
}

func (m *stubModel) Name() string { return "stub" }

func (m *stubModel) Chat(ctx context.Context, req protocol.ChatRequest) (*protocol.ChatResponse, error) {
	if m.chatErr != nil {
		return nil, m.chatErr
	}
	return m.chatResp, nil
}

func (m *stubModel) StreamChat(ctx context.Context, req protocol.ChatRequest) (protocol.ChatStream, error) {
	if m.streamErr != nil {
		return nil, m.streamErr
	}
	return m.stream, nil
}

type stubStream struct {
	chunks []*protocol.ChatStreamChunk
	idx    int
	closed bool
}

func (s *stubStream) Recv() (*protocol.ChatStreamChunk, error) {
	if s.idx >= len(s.chunks) {
		return nil, io.EOF
	}
	chunk := s.chunks[s.idx]
	s.idx++
	return chunk, nil
}

func (s *stubStream) Close() error {
	s.closed = true
	return nil
}

func (s *stubStream) Metrics() metrics.CallMetrics {
	return metrics.CallMetrics{Provider: "stub", Model: "test"}
}

func TestChainInvokeEmpty(t *testing.T) {
	c := New("empty")
	_, err := c.Invoke(context.Background(), protocol.ChatRequest{Model: "m", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}}})
	if err == nil {
		t.Fatal("expected error for empty chain")
	}
}

func TestChainInvokeSingleNode(t *testing.T) {
	model := &stubModel{chatResp: &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Role: protocol.RoleAssistant, Content: "hello"}}},
	}}
	c := New("single", NewModelNode("model", model))
	resp, err := c.Invoke(context.Background(), protocol.ChatRequest{
		Model:    "gpt-4",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	if resp.FirstContent() != "hello" {
		t.Errorf("unexpected content: %s", resp.FirstContent())
	}
}

func TestChainInvokeMultiNode(t *testing.T) {
	first := &stubModel{chatResp: &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "step1"}}},
	}}
	second := &stubModel{chatResp: &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "step2"}}},
	}}
	c := New("multi", NewModelNode("first", first), NewModelNode("second", second))
	resp, err := c.Invoke(context.Background(), protocol.ChatRequest{
		Model:    "gpt-4",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	if resp.FirstContent() != "step2" {
		t.Errorf("expected step2, got %s", resp.FirstContent())
	}
}

func TestChainInvokeNodeError(t *testing.T) {
	model := &stubModel{chatErr: errors.New("boom")}
	c := New("err", NewModelNode("model", model))
	_, err := c.Invoke(context.Background(), protocol.ChatRequest{
		Model:    "gpt-4",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChainStreamSingleNode(t *testing.T) {
	stream := &stubStream{chunks: []*protocol.ChatStreamChunk{{Delta: "hi"}}}
	model := &stubModel{stream: stream}
	c := New("stream", NewModelNode("model", model))
	got, err := c.Stream(context.Background(), protocol.ChatRequest{
		Model:    "gpt-4",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	chunk, err := got.Recv()
	if err != nil || chunk.Delta != "hi" {
		t.Fatalf("unexpected chunk: %+v err=%v", chunk, err)
	}
}

func TestChainStreamMultiNode(t *testing.T) {
	first := &stubModel{chatResp: &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "mid"}}},
	}}
	stream := &stubStream{chunks: []*protocol.ChatStreamChunk{{Delta: "final"}}}
	second := &stubModel{stream: stream}
	c := New("stream-multi", NewModelNode("first", first), NewModelNode("second", second))
	got, err := c.Stream(context.Background(), protocol.ChatRequest{
		Model:    "gpt-4",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	chunk, _ := got.Recv()
	if chunk.Delta != "final" {
		t.Errorf("expected final, got %s", chunk.Delta)
	}
}

func TestChainCollectAndTransform(t *testing.T) {
	reader := protocol.StreamReaderFromArray([]*protocol.ChatStreamChunk{
		{Delta: "hello"},
		{Delta: " world"},
	})
	model := &stubModel{}
	c := New("collect", NewModelNode("model", model))

	resp, err := c.Collect(context.Background(), reader)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if resp.FirstContent() != "hello world" {
		t.Errorf("unexpected content: %s", resp.FirstContent())
	}

	reader2 := protocol.StreamReaderFromArray([]*protocol.ChatStreamChunk{{Delta: "x"}})
	out, err := c.Transform(context.Background(), reader2)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	chunk, _ := out.Recv()
	if chunk.Delta != "x" {
		t.Errorf("unexpected delta: %s", chunk.Delta)
	}
}

func TestProcessorNode(t *testing.T) {
	node := NewProcessorNode("proc", func(ctx context.Context, resp *protocol.ChatResponse) (*protocol.ChatResponse, error) {
		resp.Choices[0].Message.Content = "processed"
		return resp, nil
	})
	_, err := node.Invoke(context.Background(), protocol.ChatRequest{Model: "m"})
	if err == nil {
		t.Fatal("expected invoke error for processor node")
	}

	reader := protocol.StreamReaderFromArray([]*protocol.ChatStreamChunk{{Delta: "raw"}})
	collected, err := node.Collect(context.Background(), reader)
	if err != nil || collected.FirstContent() != "processed" {
		t.Fatalf("Collect failed: resp=%+v err=%v", collected, err)
	}

	if _, err := node.Stream(context.Background(), protocol.ChatRequest{}); err == nil {
		t.Fatal("expected stream error")
	}
	if _, err := node.Transform(context.Background(), reader); err == nil {
		t.Fatal("expected transform error")
	}
}

func TestStreamProcessorNode(t *testing.T) {
	node := NewStreamProcessorNode("sp", StreamProcessorFunc(func(ctx context.Context, chunk *protocol.ChatStreamChunk) (*protocol.ChatStreamChunk, error) {
		chunk.Delta = "!" + chunk.Delta
		return chunk, nil
	}))

	if _, err := node.Invoke(context.Background(), protocol.ChatRequest{}); err == nil {
		t.Fatal("expected invoke error")
	}
	if _, err := node.Stream(context.Background(), protocol.ChatRequest{}); err == nil {
		t.Fatal("expected stream error")
	}
	if _, err := node.Collect(context.Background(), protocol.StreamReaderFromArray(nil)); err == nil {
		t.Fatal("expected collect error")
	}

	reader := protocol.StreamReaderFromArray([]*protocol.ChatStreamChunk{{Delta: "hi"}})
	out, err := node.Transform(context.Background(), reader)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	chunk, _ := out.Recv()
	if chunk.Delta != "!hi" {
		t.Errorf("expected !hi, got %s", chunk.Delta)
	}
}

func TestProcessingStream(t *testing.T) {
	upstream := &stubStream{chunks: []*protocol.ChatStreamChunk{{Delta: "a"}}}
	ps := NewProcessingStream(upstream, StreamProcessorFunc(func(ctx context.Context, chunk *protocol.ChatStreamChunk) (*protocol.ChatStreamChunk, error) {
		return chunk, nil
	}))
	chunk, err := ps.Recv()
	if err != nil || chunk.Delta != "a" {
		t.Fatalf("Recv failed: chunk=%+v err=%v", chunk, err)
	}
	if _, err := ps.Recv(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
	if err := ps.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if ps.Metrics().Provider != "stub" {
		t.Errorf("unexpected metrics provider: %s", ps.Metrics().Provider)
	}
}

func TestBuilder(t *testing.T) {
	model := &stubModel{chatResp: &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "built"}}},
	}}
	c := NewBuilder("built").AddModel("m", model).Build()

	if c.Name() != "built" {
		t.Errorf("unexpected chain name: %s", c.Name())
	}
	resp, err := c.Invoke(context.Background(), protocol.ChatRequest{
		Model:    "gpt-4",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil || resp.FirstContent() != "built" {
		t.Fatalf("built chain failed: resp=%+v err=%v", resp, err)
	}
}

func TestChainCollectEmpty(t *testing.T) {
	c := New("empty")
	_, err := c.Collect(context.Background(), protocol.StreamReaderFromArray(nil))
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = c.Transform(context.Background(), protocol.StreamReaderFromArray(nil))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBuilderAddProcessor(t *testing.T) {
	model := &stubModel{chatResp: &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "built"}}},
	}}
	c := NewBuilder("proc-chain").
		AddModel("m", model).
		AddProcessor("p", func(ctx context.Context, resp *protocol.ChatResponse) (*protocol.ChatResponse, error) {
			resp.Choices[0].Message.Content = "processed"
			return resp, nil
		}).
		Build()
	if c.Name() != "proc-chain" {
		t.Errorf("unexpected name: %s", c.Name())
	}
}

func TestNodeNames(t *testing.T) {
	if NewProcessorNode("p", nil).Name() != "p" {
		t.Error("processor name mismatch")
	}
	if NewStreamProcessorNode("s", nil).Name() != "s" {
		t.Error("stream processor name mismatch")
	}
}

func TestProcessorNodeCollectError(t *testing.T) {
	node := NewProcessorNode("p", func(ctx context.Context, resp *protocol.ChatResponse) (*protocol.ChatResponse, error) {
		return resp, nil
	})
	_, err := node.Collect(context.Background(), &errorStreamReader{})
	if err == nil {
		t.Fatal("expected collect error")
	}
}

type errorStreamReader struct{}

func (e *errorStreamReader) Recv() (*protocol.ChatStreamChunk, error) { return nil, context.Canceled }
func (e *errorStreamReader) Close() error                             { return nil }
func (e *errorStreamReader) Metrics() metrics.CallMetrics             { return metrics.CallMetrics{} }

func TestChainStreamEmpty(t *testing.T) {
	c := New("empty")
	_, err := c.Stream(context.Background(), protocol.ChatRequest{
		Model: "m", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestChainStreamNodeError(t *testing.T) {
	model := &stubModel{chatErr: context.Canceled}
	c := New("err", NewModelNode("m", model), NewModelNode("m2", &stubModel{stream: &stubStream{}}))
	_, err := c.Stream(context.Background(), protocol.ChatRequest{
		Model: "m", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProcessingStreamProcessorError(t *testing.T) {
	upstream := &stubStream{chunks: []*protocol.ChatStreamChunk{{Delta: "a"}}}
	ps := NewProcessingStream(upstream, StreamProcessorFunc(func(ctx context.Context, chunk *protocol.ChatStreamChunk) (*protocol.ChatStreamChunk, error) {
		return nil, context.Canceled
	}))
	_, err := ps.Recv()
	if err != context.Canceled {
		t.Fatalf("expected canceled, got %v", err)
	}
}

func TestProcessingStreamClosed(t *testing.T) {
	ps := &ProcessingStream{closed: true}
	_, err := ps.Recv()
	if err == nil {
		t.Fatal("expected EOF on closed stream")
	}
}

func TestStreamProcessorFunc(t *testing.T) {
	fn := StreamProcessorFunc(func(ctx context.Context, chunk *protocol.ChatStreamChunk) (*protocol.ChatStreamChunk, error) {
		return chunk, nil
	})
	chunk, err := fn.ProcessChunk(context.Background(), &protocol.ChatStreamChunk{Delta: "x"})
	if err != nil || chunk.Delta != "x" {
		t.Fatalf("ProcessChunk failed: %+v err=%v", chunk, err)
	}
}

func TestFollowUpRequest(t *testing.T) {
	input := protocol.ChatRequest{
		Model: "gpt-4",
		Messages: []protocol.Message{
			{Role: protocol.RoleUser, Content: "hello"},
			{Role: protocol.RoleAssistant, Content: "hi"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
		TopP:        0.9,
		Stop:        []string{"END"},
		Metadata:    map[string]string{"key": "value"},
	}
	result := &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "response"}}},
	}

	followUp := FollowUpRequest(input, result)

	if followUp.Model != "gpt-4" {
		t.Errorf("model mismatch: %s", followUp.Model)
	}
	if len(followUp.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(followUp.Messages))
	}
	if followUp.Messages[2].Content != "response" {
		t.Errorf("last message content mismatch: %s", followUp.Messages[2].Content)
	}
	if followUp.MaxTokens != 100 {
		t.Errorf("max tokens mismatch: %d", followUp.MaxTokens)
	}
	if followUp.Temperature != 0.7 {
		t.Errorf("temperature mismatch: %f", followUp.Temperature)
	}
	if followUp.TopP != 0.9 {
		t.Errorf("top_p mismatch: %f", followUp.TopP)
	}
	if len(followUp.Stop) != 1 || followUp.Stop[0] != "END" {
		t.Errorf("stop mismatch: %v", followUp.Stop)
	}
	if followUp.Metadata["key"] != "value" {
		t.Errorf("metadata mismatch: %v", followUp.Metadata)
	}
}

func TestChainInvokeMultiNodeWithProcessor(t *testing.T) {
	first := &stubModel{chatResp: &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "step1"}}},
	}}
	second := &stubModel{chatResp: &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "step2"}}},
	}}
	c := New("multi-proc",
		NewModelNode("first", first),
		NewProcessorNode("proc", func(ctx context.Context, resp *protocol.ChatResponse) (*protocol.ChatResponse, error) {
			resp.Choices[0].Message.Content = resp.FirstContent() + "-processed"
			return resp, nil
		}),
		NewModelNode("second", second),
	)
	resp, err := c.Invoke(context.Background(), protocol.ChatRequest{
		Model:    "gpt-4",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	if resp.FirstContent() != "step2" {
		t.Errorf("expected step2, got %s", resp.FirstContent())
	}
}

func TestChainStreamMultiNodeFinal(t *testing.T) {
	first := &stubModel{chatResp: &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "mid"}}},
	}}
	stream := &stubStream{chunks: []*protocol.ChatStreamChunk{{Delta: "final"}}}
	second := &stubModel{stream: stream}
	c := New("stream-multi-final",
		NewModelNode("first", first),
		NewModelNode("second", second),
	)
	got, err := c.Stream(context.Background(), protocol.ChatRequest{
		Model:    "gpt-4",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}
	chunk, _ := got.Recv()
	if chunk.Delta != "final" {
		t.Errorf("expected final, got %s", chunk.Delta)
	}
}

func TestProcessingStreamClose(t *testing.T) {
	upstream := &stubStream{chunks: []*protocol.ChatStreamChunk{{Delta: "a"}}}
	ps := NewProcessingStream(upstream, StreamProcessorFunc(func(ctx context.Context, chunk *protocol.ChatStreamChunk) (*protocol.ChatStreamChunk, error) {
		return chunk, nil
	}))
	if err := ps.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if _, err := ps.Recv(); err == nil {
		t.Fatal("expected EOF after close")
	}
}

// --- Batch Processing Tests ---

func TestInvokeBatch(t *testing.T) {
	// Test 批进批出: Multiple requests → Multiple responses
	model := &stubModel{
		chatResp: &protocol.ChatResponse{
			Choices: []protocol.Choice{
				{Message: protocol.Message{Role: protocol.RoleAssistant, Content: "response"}},
			},
		},
	}
	c := New("batch-invoke", NewModelNode("model", model))

	inputs := []protocol.ChatRequest{
		{Model: "gpt-4", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "q1"}}},
		{Model: "gpt-4", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "q2"}}},
		{Model: "gpt-4", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "q3"}}},
	}

	results, err := c.InvokeBatch(context.Background(), inputs)
	if err != nil {
		t.Fatalf("InvokeBatch failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	for i, result := range results {
		if result.FirstContent() != "response" {
			t.Errorf("result %d: expected 'response', got '%s'", i, result.FirstContent())
		}
	}
}

func TestStreamBatch(t *testing.T) {
	// Test 批进流出: Multiple requests → Stream of responses
	stream := &stubStream{chunks: []*protocol.ChatStreamChunk{
		{Delta: "resp"},
	}}
	model := &stubModel{stream: stream}
	c := New("batch-stream", NewModelNode("model", model))

	inputs := []protocol.ChatRequest{
		{Model: "gpt-4", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "q1"}}},
		{Model: "gpt-4", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "q2"}}},
	}

	result, err := c.StreamBatch(context.Background(), inputs)
	if err != nil {
		t.Fatalf("StreamBatch failed: %v", err)
	}

	// Read all chunks
	var chunks []*protocol.ChatStreamChunk
	for {
		chunk, err := result.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
}

func TestCollectBatch(t *testing.T) {
	// Test 流进批出: Multiple streams → Multiple responses
	c := New("batch-collect", NewModelNode("model", &stubModel{}))

	readers := []protocol.StreamReader{
		&stubStream{chunks: []*protocol.ChatStreamChunk{{Delta: "a"}, {Delta: "b"}}},
		&stubStream{chunks: []*protocol.ChatStreamChunk{{Delta: "c"}, {Delta: "d"}}},
	}

	results, err := c.CollectBatch(context.Background(), readers)
	if err != nil {
		t.Fatalf("CollectBatch failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	if results[0].FirstContent() != "ab" {
		t.Errorf("expected 'ab', got '%s'", results[0].FirstContent())
	}

	if results[1].FirstContent() != "cd" {
		t.Errorf("expected 'cd', got '%s'", results[1].FirstContent())
	}
}

func TestTransformBatch(t *testing.T) {
	// Test 流进流出: Multiple streams → Multiple transformed streams
	c := New("batch-transform", NewModelNode("model", &stubModel{}))

	readers := []protocol.StreamReader{
		&stubStream{chunks: []*protocol.ChatStreamChunk{{Delta: "a"}}},
		&stubStream{chunks: []*protocol.ChatStreamChunk{{Delta: "b"}}},
	}

	results, err := c.TransformBatch(context.Background(), readers)
	if err != nil {
		t.Fatalf("TransformBatch failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Verify each stream is readable
	for i, reader := range results {
		chunk, err := reader.Recv()
		if err != nil && err != io.EOF {
			t.Errorf("stream %d: Recv failed: %v", i, err)
		}
		if chunk != nil && chunk.Delta == "" {
			t.Errorf("stream %d: expected non-empty delta", i)
		}
	}
}

func TestModelNodeMetrics(t *testing.T) {
	model := &stubModel{chatResp: &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "hello"}}},
	}}
	node := NewModelNode("test", model)
	if node.Name() != "test" {
		t.Errorf("unexpected node name: %s", node.Name())
	}
}
