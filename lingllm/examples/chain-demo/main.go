package main

import (
	"context"
	"fmt"
	"time"

	"github.com/LingByte/lingllm/chain"
	"github.com/LingByte/lingllm/examples/exutil"
	"github.com/LingByte/lingllm/metrics"
	"github.com/LingByte/lingllm/protocol"
)

func main() {
	fmt.Println("=== Chain Compose Demo ===")

	// Example 1: Simple Translation Chain
	fmt.Println("1. Translation Chain (Compose API)")
	translationChain()

	// Example 2: Chain with Processor
	fmt.Println("\n2. Chain with Post-Processing")
	chainWithProcessor()

	// Example 3: Legacy Chain (compatible with old API)
	fmt.Println("\n3. Legacy Chain API")
	legacyChain()
}

// mockModel implements protocol.ChatModel for demo
type mockModel struct {
	name    string
	replies []string
}

func (m *mockModel) Name() string { return m.name }

func (m *mockModel) Chat(ctx context.Context, req protocol.ChatRequest) (*protocol.ChatResponse, error) {
	reply := "Hello!"
	if len(m.replies) > 0 {
		reply = m.replies[0]
		m.replies = m.replies[1:]
	}
	return &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Role: protocol.RoleAssistant, Content: reply}}},
	}, nil
}

func (m *mockModel) StreamChat(ctx context.Context, req protocol.ChatRequest) (protocol.ChatStream, error) {
	return &mockStream{}, nil
}

type mockStream struct{}

func (s *mockStream) Recv() (*protocol.ChatStreamChunk, error) {
	return nil, fmt.Errorf("stream ended")
}
func (s *mockStream) Close() error                 { return nil }
func (s *mockStream) Metrics() metrics.CallMetrics { return metrics.CallMetrics{} }

func translationChain() {
	// Create a translation chain: Input(map[string]any) -> Output(string)

	// 1. Create the chain
	transChain := chain.NewGenericChain[map[string]any, string]("translation")

	// 2. Add prompt template
	promptTpl := chain.NewSimplePrompt(
		"You are a professional translator. Translate the following text to {{target_lang}}.",
		"Text to translate: {{text}}",
	)
	transChain.AppendPrompt("translator-prompt", promptTpl)

	// 3. Add chat model
	model := &mockModel{
		name:    "claude",
		replies: []string{"This is a professional translation"},
	}
	transChain.AppendModel("translator-model", model)

	// 4. Add lambda to extract result
	transChain.AppendLambda("extractor", func(ctx context.Context, resp any) (any, error) {
		response, ok := resp.(*protocol.ChatResponse)
		if !ok {
			return "", fmt.Errorf("expected *ChatResponse, got %T", resp)
		}
		return response.Choices[0].Message.Content, nil
	})

	// 5. Compile
	ctx := context.Background()
	if err := transChain.Compile(ctx); err != nil {
		fmt.Printf("Compile error: %v\n", err)
		return
	}

	// 6. Execute
	input := map[string]any{
		"text":        "Hello, how are you?",
		"target_lang": "Chinese",
	}

	e2eStart := time.Now()
	result, err := transChain.Invoke(ctx, input)
	if err != nil {
		fmt.Printf("Invoke error: %v\n", err)
		return
	}
	exutil.LogE2E("translation", e2eStart)

	fmt.Printf("Input: %v\n", input["text"])
	fmt.Printf("Output: %s\n", result)
}

func chainWithProcessor() {
	// Create a chain with post-processing: Input -> Model -> Process -> Output

	chatChain := chain.NewGenericChain[map[string]any, string]("processor-demo")

	// Add template
	chatChain.AppendPrompt("chat-prompt", chain.NewSimplePrompt(
		"You are a helpful assistant.",
		"User question: {{question}}",
	))

	// Add model
	model := &mockModel{
		name:    "claude",
		replies: []string{"The weather is sunny today."},
	}
	chatChain.AppendModel("chat-model", model)

	// Add processor lambda (transform response)
	chatChain.AppendLambda("formatter", func(ctx context.Context, resp any) (any, error) {
		response, ok := resp.(*protocol.ChatResponse)
		if !ok {
			return "", fmt.Errorf("expected *ChatResponse, got %T", resp)
		}
		content := response.Choices[0].Message.Content
		return fmt.Sprintf("[Answer] %s", content), nil
	})

	ctx := context.Background()
	chatChain.Compile(ctx)

	input := map[string]any{"question": "What's the weather?"}
	e2eStart := time.Now()
	result, err := chatChain.Invoke(ctx, input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	exutil.LogE2E("processor", e2eStart)

	fmt.Printf("Result: %s\n", result)
}

func legacyChain() {
	// Legacy chain API for backwards compatibility
	model := &mockModel{
		name:    "claude",
		replies: []string{"Response from claude"},
	}

	// Use legacy New() API
	c := chain.New("legacy-demo", chain.NewModelNode("model", model))

	ctx := context.Background()
	e2eStart := time.Now()
	resp, err := c.Invoke(ctx, protocol.ChatRequest{
		Model:    "claude",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "Hello"}},
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	exutil.LogE2E("legacy", e2eStart)

	fmt.Printf("Legacy Response: %s\n", resp.FirstContent())
}
