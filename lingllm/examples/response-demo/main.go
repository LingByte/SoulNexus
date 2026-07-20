// Basic OpenAI-compatible gateway chat demo (sync + stream).
//
// Usage:
//
//	export GATEWAY_API_KEY=...
//	export GATEWAY_BASE_URL=https://your-gateway.example/v1
//	go run ./examples/gateway
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/LingByte/lingllm/examples/exutil"
	"github.com/LingByte/lingllm/protocol"
	_ "github.com/LingByte/lingllm/protocol/response"
)

func main() {
	apiKey := os.Getenv("GATEWAY_API_KEY")
	baseURL := os.Getenv("GATEWAY_BASE_URL")
	if apiKey == "" || baseURL == "" {
		log.Fatal("GATEWAY_API_KEY and GATEWAY_BASE_URL are required")
	}

	client, err := protocol.NewClient(protocol.ClientConfig{
		Provider: protocol.ProviderOpenAIResponse,
		APIKey:   apiKey,
		BaseURL:  baseURL,
	})
	if err != nil {
		log.Fatalf("create client: %v", err)
	}

	ctx := context.Background()
	prompt := envOr("PROMPT", "Say hello in one short sentence.")
	model := envOr("GATEWAY_MODEL", "claude")

	req := protocol.ChatRequest{
		Model:    model,
		Messages: []protocol.Message{protocol.UserMessage(prompt)},
	}

	e2eStart := time.Now()
	resp, err := client.Chat(ctx, req)
	if err != nil {
		log.Fatalf("chat: %v", err)
	}
	fmt.Println("--- Chat ---")
	fmt.Println(resp.FirstContent())
	exutil.LogChat("chat", resp, e2eStart)

	e2eStart = time.Now()
	stream, err := client.StreamChat(ctx, req)
	if err != nil {
		log.Fatalf("stream chat: %v", err)
	}
	defer stream.Close()

	fmt.Println("--- Stream ---")
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("recv: %v", err)
		}
		fmt.Print(chunk.Delta)
	}
	fmt.Println()
	exutil.LogStream("stream", stream, e2eStart)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
