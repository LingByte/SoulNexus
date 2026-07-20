package chain

import (
	"context"
	"fmt"
	"io"

	"github.com/LingByte/lingllm/metrics"
	"github.com/LingByte/lingllm/prompt"
	"github.com/LingByte/lingllm/protocol"
)

// Runnable represents a composable unit that can be invoked or streamed.
type Runnable interface {
	Invoke(ctx context.Context, input protocol.ChatRequest) (*protocol.ChatResponse, error)
	Stream(ctx context.Context, input protocol.ChatRequest) (protocol.ChatStream, error)
	Collect(ctx context.Context, reader protocol.StreamReader) (*protocol.ChatResponse, error)
	Transform(ctx context.Context, reader protocol.StreamReader) (protocol.StreamReader, error)
}

// Node represents a single node in a processing chain.
type Node interface {
	Runnable
	Name() string
}

// NodeChain represents a sequence of nodes that can be chained together.
// This is the legacy API, prefer using NewChain for new code.
type NodeChain struct {
	nodes []Node
	name  string
}

// NewNodeChain creates a new node chain with the given nodes.
func NewNodeChain(name string, nodes ...Node) *NodeChain {
	return &NodeChain{
		nodes: nodes,
		name:  name,
	}
}

// Name returns the chain's name.
func (c *NodeChain) Name() string {
	return c.name
}

// Invoke executes all nodes in sequence synchronously.
func (c *NodeChain) Invoke(ctx context.Context, input protocol.ChatRequest) (*protocol.ChatResponse, error) {
	if len(c.nodes) == 0 {
		return nil, fmt.Errorf("chain has no nodes")
	}

	var result *protocol.ChatResponse
	var err error

	for i, node := range c.nodes {
		if i == 0 {
			result, err = node.Invoke(ctx, input)
		} else {
			// Check if this is a processor node
			if procNode, ok := node.(*ProcessorNode); ok {
				// For processor nodes, pass the previous result for processing
				// The processor modifies the result in place
				result, err = procNode.ProcessResult(ctx, result)
			} else {
				nextReq := FollowUpRequest(input, result)
				result, err = node.Invoke(ctx, nextReq)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("chain node %d (%s) failed: %w", i, node.Name(), err)
		}
	}

	return result, nil
}

// Stream executes all nodes in sequence with streaming on the final node.
func (c *NodeChain) Stream(ctx context.Context, input protocol.ChatRequest) (protocol.ChatStream, error) {
	if len(c.nodes) == 0 {
		return nil, fmt.Errorf("chain has no nodes")
	}

	var result *protocol.ChatResponse
	var err error

	for i, node := range c.nodes {
		if i < len(c.nodes)-1 {
			if i == 0 {
				result, err = node.Invoke(ctx, input)
			} else {
				result, err = node.Invoke(ctx, FollowUpRequest(input, result))
			}
			if err != nil {
				return nil, fmt.Errorf("chain node %d (%s) failed: %w", i, node.Name(), err)
			}
		} else {
			streamInput := input
			if len(c.nodes) > 1 {
				streamInput = FollowUpRequest(input, result)
			}
			return node.Stream(ctx, streamInput)
		}
	}

	return nil, fmt.Errorf("chain has no final node")
}

// Collect reads all chunks from a stream into a response.
func (c *NodeChain) Collect(ctx context.Context, reader protocol.StreamReader) (*protocol.ChatResponse, error) {
	if len(c.nodes) == 0 {
		return nil, fmt.Errorf("chain has no nodes")
	}
	return c.nodes[len(c.nodes)-1].Collect(ctx, reader)
}

// Transform applies a transformation to each chunk in a stream.
func (c *NodeChain) Transform(ctx context.Context, reader protocol.StreamReader) (protocol.StreamReader, error) {
	if len(c.nodes) == 0 {
		return nil, fmt.Errorf("chain has no nodes")
	}
	return c.nodes[len(c.nodes)-1].Transform(ctx, reader)
}

func FollowUpRequest(input protocol.ChatRequest, result *protocol.ChatResponse) protocol.ChatRequest {
	messages := append(input.Messages, protocol.Message{
		Role:    protocol.RoleAssistant,
		Content: result.FirstContent(),
	})
	return protocol.ChatRequest{
		Model:       input.Model,
		Messages:    messages,
		MaxTokens:   input.MaxTokens,
		Temperature: input.Temperature,
		TopP:        input.TopP,
		Stop:        input.Stop,
		Metadata:    input.Metadata,
	}
}

// ModelNode wraps a ChatModel as a Node.
type ModelNode struct {
	model protocol.ChatModel
	name  string
}

// NewModelNode creates a new ModelNode.
func NewModelNode(name string, model protocol.ChatModel) *ModelNode {
	return &ModelNode{
		model: model,
		name:  name,
	}
}

// Name returns the node's name.
func (n *ModelNode) Name() string {
	return n.name
}

// Invoke calls the model synchronously.
func (n *ModelNode) Invoke(ctx context.Context, input protocol.ChatRequest) (*protocol.ChatResponse, error) {
	return n.model.Chat(ctx, input)
}

// Stream calls the model with streaming.
func (n *ModelNode) Stream(ctx context.Context, input protocol.ChatRequest) (protocol.ChatStream, error) {
	return n.model.StreamChat(ctx, input)
}

// Collect reads all chunks from a stream into a response.
func (n *ModelNode) Collect(ctx context.Context, reader protocol.StreamReader) (*protocol.ChatResponse, error) {
	return CollectStream(ctx, reader)
}

// Transform applies a pass-through transformation to each chunk.
func (n *ModelNode) Transform(ctx context.Context, reader protocol.StreamReader) (protocol.StreamReader, error) {
	s, err := TransformStream(reader, StreamTransformerFunc(func(ctx context.Context, chunk *protocol.ChatStreamChunk) (*protocol.ChatStreamChunk, error) {
		return chunk, nil
	}))
	return s, err
}

// ProcessorFunc processes a ChatResponse.
type ProcessorFunc func(ctx context.Context, resp *protocol.ChatResponse) (*protocol.ChatResponse, error)

// ProcessorNode wraps a processor function as a Node.
type ProcessorNode struct {
	fn   ProcessorFunc
	name string
}

// NewProcessorNode creates a new ProcessorNode.
func NewProcessorNode(name string, fn ProcessorFunc) *ProcessorNode {
	return &ProcessorNode{
		fn:   fn,
		name: name,
	}
}

// Name returns the node's name.
func (n *ProcessorNode) Name() string {
	return n.name
}

// Invoke executes the processor function.
// Note: ProcessorNode should not be used as the first node in a chain.
// Use ProcessResult instead when you have a previous response to process.
func (n *ProcessorNode) Invoke(ctx context.Context, input protocol.ChatRequest) (*protocol.ChatResponse, error) {
	return nil, fmt.Errorf("processor node %s cannot be invoked directly; use it after a model node", n.name)
}

// ProcessResult processes an existing response (for use in chains).
func (n *ProcessorNode) ProcessResult(ctx context.Context, input *protocol.ChatResponse) (*protocol.ChatResponse, error) {
	return n.fn(ctx, input)
}

// Stream is not supported for processor nodes.
func (n *ProcessorNode) Stream(ctx context.Context, input protocol.ChatRequest) (protocol.ChatStream, error) {
	return nil, fmt.Errorf("processor node %s does not support streaming", n.name)
}

// Collect reads all chunks from a stream and applies the processor.
func (n *ProcessorNode) Collect(ctx context.Context, reader protocol.StreamReader) (*protocol.ChatResponse, error) {
	resp, err := CollectStream(ctx, reader)
	if err != nil {
		return nil, err
	}
	return n.fn(ctx, resp)
}

// Transform is not supported for processor nodes.
func (n *ProcessorNode) Transform(ctx context.Context, reader protocol.StreamReader) (protocol.StreamReader, error) {
	return nil, fmt.Errorf("processor node %s does not support streaming transform", n.name)
}

// StreamProcessor processes chunks from a stream.
type StreamProcessor interface {
	ProcessChunk(ctx context.Context, chunk *protocol.ChatStreamChunk) (*protocol.ChatStreamChunk, error)
}

// StreamProcessorFunc is a function that processes stream chunks.
type StreamProcessorFunc func(ctx context.Context, chunk *protocol.ChatStreamChunk) (*protocol.ChatStreamChunk, error)

// ProcessChunk implements StreamProcessor.
func (f StreamProcessorFunc) ProcessChunk(ctx context.Context, chunk *protocol.ChatStreamChunk) (*protocol.ChatStreamChunk, error) {
	return f(ctx, chunk)
}

// StreamProcessorNode wraps a stream processor as a Node.
type StreamProcessorNode struct {
	processor StreamProcessor
	name      string
}

// NewStreamProcessorNode creates a new StreamProcessorNode.
func NewStreamProcessorNode(name string, processor StreamProcessor) *StreamProcessorNode {
	return &StreamProcessorNode{
		processor: processor,
		name:      name,
	}
}

// Name returns the node's name.
func (n *StreamProcessorNode) Name() string {
	return n.name
}

// Invoke is not supported for stream processor nodes.
func (n *StreamProcessorNode) Invoke(ctx context.Context, input protocol.ChatRequest) (*protocol.ChatResponse, error) {
	return nil, fmt.Errorf("stream processor node %s does not support sync invoke", n.name)
}

// Stream is not supported without an upstream stream.
func (n *StreamProcessorNode) Stream(ctx context.Context, input protocol.ChatRequest) (protocol.ChatStream, error) {
	return nil, fmt.Errorf("stream processor node %s requires an upstream stream", n.name)
}

// Collect is not supported for stream processor nodes.
func (n *StreamProcessorNode) Collect(ctx context.Context, reader protocol.StreamReader) (*protocol.ChatResponse, error) {
	return nil, fmt.Errorf("stream processor node %s does not support collect", n.name)
}

// Transform applies the stream processor to each chunk.
func (n *StreamProcessorNode) Transform(ctx context.Context, reader protocol.StreamReader) (protocol.StreamReader, error) {
	return NewProcessingStream(reader, n.processor), nil
}

// ProcessingStream wraps a stream and applies a processor to each chunk.
type ProcessingStream struct {
	upstream  protocol.StreamReader
	processor StreamProcessor
	closed    bool
}

// NewProcessingStream creates a new ProcessingStream.
func NewProcessingStream(upstream protocol.StreamReader, processor StreamProcessor) *ProcessingStream {
	return &ProcessingStream{
		upstream:  upstream,
		processor: processor,
	}
}

// Recv receives and processes the next chunk.
func (s *ProcessingStream) Recv() (*protocol.ChatStreamChunk, error) {
	if s.closed {
		return nil, io.EOF
	}

	chunk, err := s.upstream.Recv()
	if err != nil {
		if err == io.EOF {
			s.closed = true
		}
		return nil, err
	}

	processed, err := s.processor.ProcessChunk(context.Background(), chunk)
	if err != nil {
		return nil, err
	}

	return processed, nil
}

// Close closes the stream.
func (s *ProcessingStream) Close() error {
	s.closed = true
	return s.upstream.Close()
}

// Metrics returns the upstream metrics.
func (s *ProcessingStream) Metrics() metrics.CallMetrics {
	return s.upstream.Metrics()
}

// Builder provides a fluent API for building chains.
type Builder struct {
	nodes []Node
	name  string
}

// NewBuilder creates a new Builder.
func NewBuilder(name string) *Builder {
	return &Builder{
		name:  name,
		nodes: make([]Node, 0),
	}
}

// AddModel adds a ChatModel node to the chain.
func (b *Builder) AddModel(name string, model protocol.ChatModel) *Builder {
	b.nodes = append(b.nodes, NewModelNode(name, model))
	return b
}

// AddProcessor adds a processor node to the chain.
func (b *Builder) AddProcessor(name string, fn ProcessorFunc) *Builder {
	b.nodes = append(b.nodes, NewProcessorNode(name, fn))
	return b
}

// AddNode adds a custom node to the chain.
func (b *Builder) AddNode(node Node) *Builder {
	b.nodes = append(b.nodes, node)
	return b
}

// Build creates the chain.
func (b *Builder) Build() *NodeChain {
	return NewNodeChain(b.name, b.nodes...)
}

// New is an alias for NewNodeChain for backwards compatibility.
// Deprecated: Use NewNodeChain for new code.
func New(name string, nodes ...Node) *NodeChain {
	return NewNodeChain(name, nodes...)
}

// --- Generic Chain API ---

// GenericChain represents a sequence of operations with type-safe input and output.
type GenericChain[I, O any] struct {
	name     string
	steps    []GenericStep
	compiled bool
}

// GenericStep is a single step in a generic chain.
type GenericStep interface {
	execute(ctx context.Context, input any) (any, error)
	stepName() string
}

// NewGenericChain creates a new generic chain.
func NewGenericChain[I, O any](name string) *GenericChain[I, O] {
	return &GenericChain[I, O]{
		name:  name,
		steps: make([]GenericStep, 0),
	}
}

// AppendPrompt adds a prompt template step.
func (c *GenericChain[I, O]) AppendPrompt(name string, template PromptTemplate) *GenericChain[I, O] {
	c.steps = append(c.steps, &genericPromptStep{
		n:        name,
		template: template,
	})
	return c
}

// AppendModel adds a chat model step.
func (c *GenericChain[I, O]) AppendModel(name string, model protocol.ChatModel) *GenericChain[I, O] {
	c.steps = append(c.steps, &genericModelStep{
		n:     name,
		model: model,
	})
	return c
}

// AppendLambda adds a transformation lambda.
func (c *GenericChain[I, O]) AppendLambda(name string, fn func(context.Context, any) (any, error)) *GenericChain[I, O] {
	c.steps = append(c.steps, &genericLambdaStep{
		n:  name,
		fn: fn,
	})
	return c
}

// Compile validates the chain.
func (c *GenericChain[I, O]) Compile(ctx context.Context) error {
	if len(c.steps) == 0 {
		return fmt.Errorf("chain %s has no steps", c.name)
	}
	c.compiled = true
	return nil
}

// Invoke executes the chain.
func (c *GenericChain[I, O]) Invoke(ctx context.Context, input I) (O, error) {
	var zero O
	if !c.compiled {
		if err := c.Compile(ctx); err != nil {
			return zero, err
		}
	}

	var current any = input
	var err error

	for i, step := range c.steps {
		current, err = step.execute(ctx, current)
		if err != nil {
			return zero, fmt.Errorf("chain step %d (%s) failed: %w", i, step.stepName(), err)
		}
	}

	result, ok := current.(O)
	if !ok {
		return zero, fmt.Errorf("chain output type mismatch: got %T, want %T", current, zero)
	}

	return result, nil
}

type genericPromptStep struct {
	n        string
	template PromptTemplate
}

func (s *genericPromptStep) stepName() string { return s.n }

func (s *genericPromptStep) execute(ctx context.Context, input any) (any, error) {
	data, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("prompt expects map[string]any, got %T", input)
	}
	return s.template.Render(data)
}

type genericModelStep struct {
	n     string
	model protocol.ChatModel
}

func (s *genericModelStep) stepName() string { return s.n }

func (s *genericModelStep) execute(ctx context.Context, input any) (any, error) {
	req, ok := input.(protocol.ChatRequest)
	if !ok {
		if msgs, ok := input.([]protocol.Message); ok {
			req = protocol.ChatRequest{Messages: msgs}
		} else {
			return nil, fmt.Errorf("model expects ChatRequest or []Message, got %T", input)
		}
	}
	return s.model.Chat(ctx, req)
}

type genericLambdaStep struct {
	n  string
	fn func(context.Context, any) (any, error)
}

func (s *genericLambdaStep) stepName() string { return s.n }

func (s *genericLambdaStep) execute(ctx context.Context, input any) (any, error) {
	return s.fn(ctx, input)
}

// PromptTemplate defines a template that renders messages.
type PromptTemplate interface {
	Render(data map[string]any) ([]protocol.Message, error)
}

// SimplePrompt creates a prompt template from system and user templates.
type SimplePrompt struct {
	systemTemplate string
	userTemplate   string
}

// NewSimplePrompt creates a simple prompt with system and user messages.
func NewSimplePrompt(system, user string) *SimplePrompt {
	return &SimplePrompt{systemTemplate: system, userTemplate: user}
}

func (p *SimplePrompt) Render(data map[string]any) ([]protocol.Message, error) {
	var messages []protocol.Message

	if p.systemTemplate != "" {
		tpl, err := prompt.NewTemplate("system", p.systemTemplate)
		if err != nil {
			return nil, err
		}
		content, err := tpl.Render(data)
		if err != nil {
			return nil, err
		}
		messages = append(messages, protocol.Message{Role: protocol.RoleSystem, Content: content})
	}

	if p.userTemplate != "" {
		tpl, err := prompt.NewTemplate("user", p.userTemplate)
		if err != nil {
			return nil, err
		}
		content, err := tpl.Render(data)
		if err != nil {
			return nil, err
		}
		messages = append(messages, protocol.Message{Role: protocol.RoleUser, Content: content})
	}

	return messages, nil
}

// --- Stream Utilities ---

// StreamTransformer transforms chunks.
type StreamTransformer interface {
	Transform(ctx context.Context, chunk *protocol.ChatStreamChunk) (*protocol.ChatStreamChunk, error)
}

// StreamTransformerFunc is a function adapter for StreamTransformer.
type StreamTransformerFunc func(ctx context.Context, chunk *protocol.ChatStreamChunk) (*protocol.ChatStreamChunk, error)

func (f StreamTransformerFunc) Transform(ctx context.Context, chunk *protocol.ChatStreamChunk) (*protocol.ChatStreamChunk, error) {
	return f(ctx, chunk)
}

// CollectStream collects all chunks into a response.
func CollectStream(ctx context.Context, reader protocol.StreamReader) (*protocol.ChatResponse, error) {
	var content string
	for {
		chunk, err := reader.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		content += chunk.Delta
	}
	return &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Role: protocol.RoleAssistant, Content: content}}},
	}, nil
}

// TransformStream applies a transformer to each chunk.
func TransformStream(reader protocol.StreamReader, transformer StreamTransformer) (protocol.StreamReader, error) {
	return &transStream{reader: reader, transformer: transformer}, nil
}

type transStream struct {
	reader      protocol.StreamReader
	transformer StreamTransformer
}

func (s *transStream) Recv() (*protocol.ChatStreamChunk, error) {
	chunk, err := s.reader.Recv()
	if err != nil {
		return nil, err
	}
	return s.transformer.Transform(context.Background(), chunk)
}

func (s *transStream) Close() error {
	return s.reader.Close()
}

func (s *transStream) Metrics() metrics.CallMetrics {
	return s.reader.Metrics()
}

// --- Batch Processing Methods ---

// InvokeBatch executes the chain on multiple requests synchronously (批进批出)
// Returns responses in the same order as inputs
func (c *NodeChain) InvokeBatch(ctx context.Context, inputs []protocol.ChatRequest) ([]*protocol.ChatResponse, error) {
	if len(c.nodes) == 0 {
		return nil, fmt.Errorf("chain has no nodes")
	}

	results := make([]*protocol.ChatResponse, len(inputs))
	for i, input := range inputs {
		resp, err := c.Invoke(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("batch item %d failed: %w", i, err)
		}
		results[i] = resp
	}
	return results, nil
}

// StreamBatch executes the chain on multiple requests with streaming (批进流出)
// Returns a stream of responses
func (c *NodeChain) StreamBatch(ctx context.Context, inputs []protocol.ChatRequest) (protocol.StreamReader, error) {
	if len(c.nodes) == 0 {
		return nil, fmt.Errorf("chain has no nodes")
	}

	// Create a pipe to stream results
	pipe := &batchStreamPipe{
		ch: make(chan *protocol.ChatStreamChunk, 100),
	}

	// Run batch processing in background
	go func() {
		defer close(pipe.ch)
		for i, input := range inputs {
			stream, err := c.Stream(ctx, input)
			if err != nil {
				// Send error as a chunk
				pipe.ch <- &protocol.ChatStreamChunk{
					Delta: fmt.Sprintf("Error processing item %d: %v\n", i, err),
				}
				continue
			}

			// Forward all chunks from this stream
			for {
				chunk, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					pipe.ch <- &protocol.ChatStreamChunk{
						Delta: fmt.Sprintf("Stream error at item %d: %v\n", i, err),
					}
					break
				}
				pipe.ch <- chunk
			}
			stream.Close()

			// Add separator between items
			if i < len(inputs)-1 {
				pipe.ch <- &protocol.ChatStreamChunk{Delta: "\n---\n"}
			}
		}
	}()

	return pipe, nil
}

// CollectBatch collects multiple streams into responses (流进批出)
// Each stream is collected into a complete response
func (c *NodeChain) CollectBatch(ctx context.Context, readers []protocol.StreamReader) ([]*protocol.ChatResponse, error) {
	if len(c.nodes) == 0 {
		return nil, fmt.Errorf("chain has no nodes")
	}

	results := make([]*protocol.ChatResponse, len(readers))
	for i, reader := range readers {
		resp, err := c.Collect(ctx, reader)
		if err != nil {
			return nil, fmt.Errorf("batch item %d collection failed: %w", i, err)
		}
		results[i] = resp
	}
	return results, nil
}

// TransformBatch applies transformations to multiple streams (流进流出)
// Returns transformed streams
func (c *NodeChain) TransformBatch(ctx context.Context, readers []protocol.StreamReader) ([]protocol.StreamReader, error) {
	if len(c.nodes) == 0 {
		return nil, fmt.Errorf("chain has no nodes")
	}

	results := make([]protocol.StreamReader, len(readers))
	for i, reader := range readers {
		transformed, err := c.Transform(ctx, reader)
		if err != nil {
			return nil, fmt.Errorf("batch item %d transform failed: %w", i, err)
		}
		results[i] = transformed
	}
	return results, nil
}

// batchStreamPipe implements protocol.ChatStream for batch streaming
type batchStreamPipe struct {
	ch chan *protocol.ChatStreamChunk
}

func (p *batchStreamPipe) Recv() (*protocol.ChatStreamChunk, error) {
	chunk, ok := <-p.ch
	if !ok {
		return nil, io.EOF
	}
	return chunk, nil
}

func (p *batchStreamPipe) Close() error {
	return nil
}

func (p *batchStreamPipe) Metrics() metrics.CallMetrics {
	return metrics.CallMetrics{}
}
