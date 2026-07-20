// Basic Ollama local chat demo (sync + stream).
//
// Usage:
//
//	ollama pull llama3.2
//	go run ./examples/ollama
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
	_ "github.com/LingByte/lingllm/protocol/ollama"
)

func main() {
	client, err := protocol.NewClient(protocol.ClientConfig{
		Provider: protocol.ProviderOllama,
		BaseURL:  envOr("OLLAMA_BASE_URL", "http://localhost:11434"),
		APIKey:   os.Getenv("OLLAMA_API_KEY"),
	})
	if err != nil {
		log.Fatalf("create client: %v", err)
	}

	ctx := context.Background()
	prompt := envOr("PROMPT", "Say hello in one short sentence.")
	model := envOr("OLLAMA_MODEL", "llama3.2")

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
