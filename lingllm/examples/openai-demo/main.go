// Basic OpenAI chat demo (sync + stream).
//
// Usage:
//
//	export OPENAI_API_KEY=sk-...
//	go run ./examples/openai
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/LingByte/lingllm/examples/exutil"
	"github.com/LingByte/lingllm/protocol"
	_ "github.com/LingByte/lingllm/protocol/openai"
)

func main() {
	apiKey := flag.String("apikey", "", "")
	model := flag.String("model", "claude", "")
	baseUrl := flag.String("base_url", "", "")
	flag.Parse()

	client, err := protocol.NewClient(protocol.ClientConfig{
		Provider: protocol.ProviderOpenAI,
		APIKey:   *apiKey,
		BaseURL:  *baseUrl,
	})
	if err != nil {
		log.Fatalf("create client: %v", err)
	}

	ctx := context.Background()
	prompt := "Say hello in one short sentence."

	req := protocol.ChatRequest{
		Model:    *model,
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
