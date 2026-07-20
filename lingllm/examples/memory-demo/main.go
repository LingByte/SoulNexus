package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/LingByte/lingllm/chain"
	"github.com/LingByte/lingllm/examples/exutil"
	"github.com/LingByte/lingllm/memory"
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
	c := chain.New("memory-demo", chain.NewModelNode("llm", client))

	ctx := context.Background()

	fmt.Println("=== LingLLM Memory System Demo ===")

	// Demo 1: Working Memory
	fmt.Println("Demo 1: Multi-turn Conversation with Memory")
	fmt.Println("─────────────────────────────────────")
	demoMultiTurnConversation(ctx, c, *model)

	// Demo 2: Memory Context in Prompt
	fmt.Println("\nDemo 2: Using Memory Context in Prompts")
	fmt.Println("─────────────────────────────────────")
	demoMemoryContextPrompt(ctx, c, *model)

	// Demo 3: Memory Statistics
	fmt.Println("\nDemo 3: Memory Statistics and Analysis")
	fmt.Println("─────────────────────────────────────")
	demoMemoryStatistics()
}

// Demo 1: Multi-turn conversation with memory
func demoMultiTurnConversation(ctx context.Context, c *chain.NodeChain, model string) {
	// Create working memory
	wm := memory.NewWorkingMemory("conversation-1")

	// Simulate a multi-turn conversation
	conversations := []struct {
		role    protocol.MessageRole
		content string
	}{
		{protocol.RoleUser, "My name is Alice and I work as a software engineer"},
		{protocol.RoleAssistant, "Nice to meet you, Alice! I'll remember that you're a software engineer."},
		{protocol.RoleUser, "I'm currently working on a Go project"},
		{protocol.RoleAssistant, "Great! Go is an excellent language for building scalable systems."},
		{protocol.RoleUser, "Can you help me with memory management in Go?"},
	}

	fmt.Println("Conversation History:")
	for i, conv := range conversations {
		wm.AddMessage(conv.role, conv.content)
		fmt.Printf("[%s]: %s\n", conv.role, conv.content)

		// For the last user message, get LLM response
		if i == len(conversations)-1 && conv.role == protocol.RoleUser {
			fmt.Println("\nGenerating response with memory context...")

			// Build request with memory context
			messages := wm.GetMessages()
			req := protocol.ChatRequest{
				Model:     model,
				Messages:  messages,
				MaxTokens: 200,
			}

			e2eStart := time.Now()
			resp, err := c.Invoke(ctx, req)
			if err != nil {
				log.Printf("Error: %v", err)
				return
			}
			exutil.LogChat("multi-turn", resp, e2eStart)

			response := resp.FirstContent()
			fmt.Printf("[Assistant]: %s\n", response)

			// Add to memory
			wm.AddMessage(protocol.RoleAssistant, response)
		}
	}

	// Show memory stats
	fmt.Println("\n--- Working Memory Stats ---")
	stats := wm.GetStats()
	fmt.Printf("Round ID: %s\n", stats.RoundID)
	fmt.Printf("Messages: %d\n", stats.MessageCount)
	fmt.Printf("Thoughts: %d\n", stats.ThoughtCount)
	fmt.Printf("Actions: %d\n", stats.ActionCount)
	fmt.Printf("Observations: %d\n", stats.ObservationCount)
	fmt.Printf("Duration: %v\n", stats.Duration)
}

// Demo 2: Using memory context in prompts
func demoMemoryContextPrompt(ctx context.Context, c *chain.NodeChain, model string) {
	wm := memory.NewWorkingMemory("context-demo")

	// Build conversation context
	wm.AddMessage(protocol.RoleUser, "I'm learning Go and interested in concurrency")
	wm.AddThought("The user is interested in Go concurrency")
	wm.AddAction("search", map[string]interface{}{"query": "Go concurrency patterns"})
	wm.AddObservation("Found goroutines, channels, and sync packages")

	wm.AddMessage(protocol.RoleAssistant, "Go has great concurrency support with goroutines and channels")
	wm.AddMessage(protocol.RoleUser, "How do I handle errors in concurrent code?")

	// Generate prompt with memory context
	fmt.Println("Memory Context:")
	fmt.Println(wm.ToPrompt())

	// Make request with memory context
	fmt.Println("\nGenerating response with full memory context...")

	systemPrompt := fmt.Sprintf(`You are a helpful Go programming assistant. 
Here is the conversation context:
%s

Based on this context, answer the user's latest question.`, wm.ToPrompt())

	messages := []protocol.Message{
		{Role: protocol.RoleSystem, Content: systemPrompt},
	}
	messages = append(messages, wm.GetMessages()...)

	req := protocol.ChatRequest{
		Model:     model,
		Messages:  messages,
		MaxTokens: 300,
	}

	e2eStart := time.Now()
	resp, err := c.Invoke(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	exutil.LogChat("memory-context", resp, e2eStart)

	fmt.Printf("Response: %s\n", resp.FirstContent())
}

// Demo 3: Memory statistics and analysis
func demoMemoryStatistics() {
	fmt.Println("Creating memory with sample data...")

	wm := memory.NewWorkingMemory("stats-demo")

	// Add various messages and ReAct chain
	wm.AddMessage(protocol.RoleUser, "Hello, how are you?")
	wm.AddMessage(protocol.RoleAssistant, "I'm doing well, thanks for asking")

	wm.AddThought("The user is asking a greeting")
	wm.AddAction("analyze", map[string]interface{}{"sentiment": "positive"})
	wm.AddObservation("User is in a good mood")

	wm.AddMessage(protocol.RoleUser, "Let's start coding")
	wm.AddMessage(protocol.RoleAssistant, "Great! What would you like to build?")

	// Add temp variables
	wm.SetTempVar("language", "Go")
	wm.SetTempVar("project_type", "web_service")
	wm.SetTempVar("deadline", "2 weeks")

	// Show statistics
	fmt.Println("\n--- Working Memory Statistics ---")
	wmStats := wm.GetStats()
	fmt.Printf("Round ID: %s\n", wmStats.RoundID)
	fmt.Printf("Total Messages: %d\n", wmStats.MessageCount)
	fmt.Printf("Thoughts: %d\n", wmStats.ThoughtCount)
	fmt.Printf("Actions: %d\n", wmStats.ActionCount)
	fmt.Printf("Observations: %d\n", wmStats.ObservationCount)
	fmt.Printf("Temp Variables: %d\n", wmStats.TempVarCount)
	fmt.Printf("Duration: %v\n", wmStats.Duration)

	// Show ReAct chain
	fmt.Println("\n--- ReAct Chain ---")
	chain := wm.GetReActChain()
	fmt.Printf("Thoughts: %d\n", len(chain.Thoughts))
	for i, thought := range chain.Thoughts {
		fmt.Printf("  %d. %s\n", i+1, thought)
	}
	fmt.Printf("Actions: %d\n", len(chain.Actions))
	for i, action := range chain.Actions {
		fmt.Printf("  %d. %s(%v)\n", i+1, action.ToolName, action.Input)
	}
	fmt.Printf("Observations: %d\n", len(chain.Observations))
	for i, obs := range chain.Observations {
		fmt.Printf("  %d. %s\n", i+1, obs)
	}

	// Show temp variables
	fmt.Println("\n--- Temporary Variables ---")
	vars := wm.GetAllTempVars()
	for key, value := range vars {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Show complete context
	fmt.Println("\n--- Complete Context ---")
	ctx := wm.GetContext()
	fmt.Printf("Round ID: %s\n", ctx.RoundID)
	fmt.Printf("Messages: %d\n", len(ctx.Messages))
	fmt.Printf("Thoughts: %d\n", len(ctx.ReActChain.Thoughts))
	fmt.Printf("Actions: %d\n", len(ctx.ReActChain.Actions))
	fmt.Printf("Observations: %d\n", len(ctx.ReActChain.Observations))
	fmt.Printf("Temp Vars: %d\n", len(ctx.TempVars))
}
