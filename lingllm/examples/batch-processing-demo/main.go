package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/LingByte/lingllm/chain"
	"github.com/LingByte/lingllm/examples/exutil"
	"github.com/LingByte/lingllm/protocol"
	_ "github.com/LingByte/lingllm/protocol/openai"
)

func main() {
	apiKey := flag.String("apikey", "", "API key for the LLM provider")
	model := flag.String("model", "gpt-4", "Model name")
	baseURL := flag.String("base_url", "", "Base URL for the API")
	flag.Parse()

	if *apiKey == "" {
		log.Fatal("apikey is required")
	}

	// Create the LLM client
	client, err := protocol.NewClient(protocol.ClientConfig{
		Provider: "openai",
		APIKey:   *apiKey,
		BaseURL:  *baseURL,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create a simple chain with one model node
	c := chain.New("batch-demo", chain.NewModelNode("llm", client))

	ctx := context.Background()

	fmt.Println("=== LingLLM Batch Processing Demo ===")

	// Demo 1: 批进批出 (Batch In, Batch Out)
	fmt.Println("Demo 1: 批进批出 (Batch In, Batch Out)")
	fmt.Println("─────────────────────────────────────")
	demoBatchInBatchOut(ctx, c, *model)

	// Demo 2: 批进流出 (Batch In, Stream Out)
	fmt.Println("\nDemo 2: 批进流出 (Batch In, Stream Out)")
	fmt.Println("─────────────────────────────────────")
	demoBatchInStreamOut(ctx, c, *model)

	// Demo 3: 流进批出 (Stream In, Batch Out)
	fmt.Println("Demo 3: 流进批出 (Stream In, Batch Out)")
	fmt.Println("─────────────────────────────────────")
	demoStreamInBatchOut(ctx, c, *model)

	// Demo 4: 流进流出 (Stream In, Stream Out)
	fmt.Println("Demo 4: 流进流出 (Stream In, Stream Out)")
	fmt.Println("─────────────────────────────────────")
	demoStreamInStreamOut(ctx, c, *model)
}

// Demo 1: 批进批出 - 同时处理多个请求，返回所有响应
// 场景: 批量翻译、批量分类、批量摘要等
func demoBatchInBatchOut(ctx context.Context, c *chain.NodeChain, model string) {
	fmt.Println("场景: 批量翻译文本")
	fmt.Println("输入: 3 个英文句子")
	fmt.Println("输出: 3 个中文翻译")

	// 准备批量请求
	requests := []protocol.ChatRequest{
		{
			Model: model,
			Messages: []protocol.Message{
				{Role: protocol.RoleUser, Content: "Translate to Chinese: Hello, how are you?"},
			},
			MaxTokens: 100,
		},
		{
			Model: model,
			Messages: []protocol.Message{
				{Role: protocol.RoleUser, Content: "Translate to Chinese: The weather is beautiful today."},
			},
			MaxTokens: 100,
		},
		{
			Model: model,
			Messages: []protocol.Message{
				{Role: protocol.RoleUser, Content: "Translate to Chinese: Thank you for your help."},
			},
			MaxTokens: 100,
		},
	}

	// 执行批处理
	fmt.Println("Processing batch of 3 requests...")
	e2eStart := time.Now()
	responses, err := c.InvokeBatch(ctx, requests)
	if err != nil {
		log.Printf("InvokeBatch error: %v", err)
		return
	}
	exutil.LogBatch("batch-in-batch-out", responses, e2eStart)

	// 输出结果
	fmt.Printf("Received %d responses:\n", len(responses))
	for i, resp := range responses {
		fmt.Printf("\n[%d] %s\n", i+1, resp.FirstContent())
	}

	fmt.Printf("\n✓ 批进批出完成: 处理了 %d 个请求，获得 %d 个响应\n", len(requests), len(responses))
}

// Demo 2: 批进流出 - 处理多个请求，流式返回所有响应
// 场景: 实时处理多个查询、逐步显示批处理结果等
func demoBatchInStreamOut(ctx context.Context, c *chain.NodeChain, model string) {
	requests := []protocol.ChatRequest{
		{
			Model: model,
			Messages: []protocol.Message{
				{Role: protocol.RoleUser, Content: "Generate a creative marketing slogan for: Coffee Shop"},
			},
			MaxTokens: 50,
		},
		{
			Model: model,
			Messages: []protocol.Message{
				{Role: protocol.RoleUser, Content: "Generate a creative marketing slogan for: Fitness App"},
			},
			MaxTokens: 50,
		},
		{
			Model: model,
			Messages: []protocol.Message{
				{Role: protocol.RoleUser, Content: "Generate a creative marketing slogan for: Travel Agency"},
			},
			MaxTokens: 50,
		},
	}

	// 执行流式批处理
	fmt.Println("Processing batch with streaming output...")
	e2eStart := time.Now()
	stream, err := c.StreamBatch(ctx, requests)
	if err != nil {
		log.Printf("StreamBatch error: %v", err)
		return
	}

	// 实时输出流式结果
	fmt.Println("Streaming responses:")
	count := 0
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Stream error: %v", err)
			break
		}
		fmt.Print(chunk.Delta)
		count++
	}
	stream.Close()
	exutil.LogStream("batch-in-stream-out", stream, e2eStart)

	fmt.Printf("\n\n✓ 批进流出完成: 处理了 %d 个请求，流式返回了 %d 个块\n", len(requests), count)
}

// Demo 3: 流进批出 - 从多个流收集数据，返回完整响应
// 场景: 聚合多个数据源、合并流式结果等
func demoStreamInBatchOut(ctx context.Context, c *chain.NodeChain, model string) {
	// 创建多个流式请求
	requests := []protocol.ChatRequest{
		{
			Model: model,
			Messages: []protocol.Message{
				{Role: protocol.RoleUser, Content: "List 3 benefits of exercise"},
			},
			MaxTokens: 100,
		},
		{
			Model: model,
			Messages: []protocol.Message{
				{Role: protocol.RoleUser, Content: "List 3 benefits of reading"},
			},
			MaxTokens: 100,
		},
		{
			Model: model,
			Messages: []protocol.Message{
				{Role: protocol.RoleUser, Content: "List 3 benefits of meditation"},
			},
			MaxTokens: 100,
		},
	}

	// 创建流
	fmt.Println("Creating 3 streams...")
	streams := make([]protocol.StreamReader, len(requests))
	for i, req := range requests {
		stream, err := c.Stream(ctx, req)
		if err != nil {
			log.Printf("Stream creation error: %v", err)
			return
		}
		streams[i] = stream
	}

	// 收集所有流
	fmt.Println("Collecting streams into responses...")
	e2eStart := time.Now()
	responses, err := c.CollectBatch(ctx, streams)
	if err != nil {
		log.Printf("CollectBatch error: %v", err)
		return
	}
	exutil.LogBatch("stream-in-batch-out", responses, e2eStart)

	// 输出聚合结果
	fmt.Printf("Collected %d responses:\n", len(responses))
	for i, resp := range responses {
		fmt.Printf("\n[%d] %s\n", i+1, resp.FirstContent())
	}

	fmt.Printf("\n✓ 流进批出完成: 收集了 %d 个流，获得 %d 个完整响应\n", len(streams), len(responses))
}

// Demo 4: 流进流出 - 对多个流进行转换，返回转换后的流
// 场景: 流处理管道、实时数据转换等
func demoStreamInStreamOut(ctx context.Context, c *chain.NodeChain, model string) {
	// 创建多个流式请求
	requests := []protocol.ChatRequest{
		{
			Model: model,
			Messages: []protocol.Message{
				{Role: protocol.RoleUser, Content: "Write a haiku about spring"},
			},
			MaxTokens: 50,
		},
		{
			Model: model,
			Messages: []protocol.Message{
				{Role: protocol.RoleUser, Content: "Write a haiku about summer"},
			},
			MaxTokens: 50,
		},
		{
			Model: model,
			Messages: []protocol.Message{
				{Role: protocol.RoleUser, Content: "Write a haiku about autumn"},
			},
			MaxTokens: 50,
		},
	}

	// 创建流
	fmt.Println("Creating 3 streams...")
	e2eStart := time.Now()
	streams := make([]protocol.StreamReader, len(requests))
	for i, req := range requests {
		stream, err := c.Stream(ctx, req)
		if err != nil {
			log.Printf("Stream creation error: %v", err)
			return
		}
		streams[i] = stream
	}

	// 转换流
	fmt.Println("Transforming streams...")
	transformedStreams, err := c.TransformBatch(ctx, streams)
	if err != nil {
		log.Printf("TransformBatch error: %v", err)
		return
	}

	// 从转换后的流读取数据
	fmt.Printf("Reading from %d transformed streams:\n", len(transformedStreams))
	for i, stream := range transformedStreams {
		fmt.Printf("\n[Stream %d]:\n", i+1)
		count := 0
		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Stream error: %v", err)
				break
			}
			fmt.Print(chunk.Delta)
			count++
		}
		stream.Close()
		fmt.Printf("\n(received %d chunks)", count)
	}
	exutil.LogE2E("stream-in-stream-out", e2eStart)

	fmt.Printf("\n\n✓ 流进流出完成: 转换了 %d 个流，并读取了所有数据\n", len(transformedStreams))
}
