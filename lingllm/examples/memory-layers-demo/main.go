package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/LingByte/lingllm/examples/exutil"
	"github.com/LingByte/lingllm/memory"
	"github.com/LingByte/lingllm/protocol"
	_ "github.com/LingByte/lingllm/protocol/openai"
)

func main() {
	apiKey := flag.String("apikey", "", "API key for the LLM provider")
	model := flag.String("model", "gpt-4", "Model name")
	baseURL := flag.String("base_url", "", "Base URL for the API")
	dataDir := flag.String("data_dir", "./memory_data", "Directory for L2 persistence")
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

	ctx := context.Background()

	// Create L2 short-term memory with persistence
	stm := memory.NewShortTermMemory(3, 24*time.Hour)
	if err := stm.BindPersistence(*dataDir, "demo-user"); err != nil {
		log.Fatalf("Failed to bind persistence: %v", err)
	}

	// Run interactive conversation
	interactiveConversation(ctx, client, *model, stm)

	// Show final L2 state
	fmt.Println("\n=== Final L2 State ===")
	showL2State(stm)
}

// interactiveConversation runs an interactive conversation with L1+L2 memory
func interactiveConversation(ctx context.Context, client protocol.ChatModel, model string, stm *memory.ShortTermMemory) {
	scanner := bufio.NewScanner(os.Stdin)
	roundNum := 1
	var wm *memory.WorkingMemory

	// Initialize first round
	wm = memory.NewWorkingMemory(fmt.Sprintf("round-%d", roundNum))

	// Show L2 context if available
	l2Context := stm.BuildContextString(3)
	if l2Context != "" {
		fmt.Println("📚 L2 Context (from previous rounds):")
		fmt.Println(l2Context)
		fmt.Println()
	}

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		// Handle special commands
		if input == "exit" {
			break
		}

		if input == "status" {
			fmt.Println("\nL1 State:")
			showL1State(wm)
			fmt.Println("\nL2 State:")
			showL2State(stm)
			fmt.Println()
			continue
		}

		if input == "" {
			continue
		}

		// Add L2 context to L1 if this is a new round (first message)
		if wm.GetStats().MessageCount == 0 {
			l2Context := stm.BuildContextString(3)
			if l2Context != "" {
				wm.AddMessage(protocol.RoleSystem, "Previous conversation context:\n"+l2Context)
			}
		}

		// Add user message to L1
		wm.AddMessage(protocol.RoleUser, input)

		// Get LLM response
		fmt.Print("Assistant: ")
		messages := wm.GetMessages()
		req := protocol.ChatRequest{
			Model:     model,
			Messages:  messages,
			MaxTokens: 300,
		}

		e2eStart := time.Now()
		resp, err := client.Chat(ctx, req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		exutil.LogChat("conversation", resp, e2eStart)

		response := resp.FirstContent()
		fmt.Printf("%s\n\n", response)
		wm.AddMessage(protocol.RoleAssistant, response)

		// Check if L1 capacity exceeded (4 user turns)
		// Count only user messages (not system or assistant messages)
		userMessageCount := 0
		for _, msg := range wm.GetMessages() {
			if msg.Role == protocol.RoleUser {
				userMessageCount++
			}
		}
		if userMessageCount >= 4 {
			fmt.Println("─────────────────────────────────────────────────────────────")
			fmt.Printf("L1 capacity exceeded (%d user messages). Summarizing to L2...\n", userMessageCount)

			// Generate comprehensive summary using LLM
			summary, err := generateComprehensiveSummary(ctx, client, model, wm)
			if err != nil {
				log.Printf("Error generating summary: %v", err)
				continue
			}

			// Add summary to L2
			evicted, err := stm.AddRoundSummary(summary)
			if err != nil {
				log.Printf("Error adding summary: %v", err)
				continue
			}

			fmt.Printf("✓ Round %s summarized and added to L2\n", wm.GetRoundID())
			if evicted != nil {
				fmt.Printf("Evicted round %s (L2 capacity exceeded)\n", evicted.RoundID)
			}

			// Show L2 state
			fmt.Println("\n L2 State:")
			showL2State(stm)

			// Clear L1 and start new round
			fmt.Println("\nClearing L1 for next round...")
			wm.Clear()
			roundNum++
			wm = memory.NewWorkingMemory(fmt.Sprintf("round-%d", roundNum))

			// Show L2 context for next round
			l2Context := stm.BuildContextString(3)
			if l2Context != "" {
				fmt.Println("\n� L2 Context (for next round):")
				fmt.Println(l2Context)
			}
			fmt.Println("─────────────────────────────────────────────────────────────")
		}
	}

	// Final summary if L1 has messages
	if wm.GetStats().MessageCount > 0 {
		fmt.Println("\n─────────────────────────────────────────────────────────────")
		fmt.Println("Saving final round to L2...")
		summary, err := generateComprehensiveSummary(ctx, client, model, wm)
		if err != nil {
			log.Printf("Error generating summary: %v", err)
		} else {
			evicted, err := stm.AddRoundSummary(summary)
			if err != nil {
				log.Printf("Error adding summary: %v", err)
			} else {
				fmt.Printf("✓ Round %s summarized and added to L2\n", wm.GetRoundID())
				if evicted != nil {
					fmt.Printf("Evicted round %s (L2 capacity exceeded)\n", evicted.RoundID)
				}
			}
		}
		fmt.Println("─────────────────────────────────────────────────────────────")
	}
}

// generateComprehensiveSummary uses LLM to summarize the entire L1 conversation
func generateComprehensiveSummary(ctx context.Context, client protocol.ChatModel, model string, wm *memory.WorkingMemory) (*memory.RoundSummary, error) {
	stats := wm.GetStats()

	// Build conversation text for summarization
	var conversationText strings.Builder
	conversationText.WriteString("Please summarize the following conversation concisely, capturing the main topics, key points, and important information:\n\n")

	for _, msg := range wm.GetMessages() {
		if msg.Role == protocol.RoleUser {
			conversationText.WriteString("User: " + msg.Content + "\n\n")
		} else if msg.Role == protocol.RoleAssistant {
			conversationText.WriteString("Assistant: " + msg.Content + "\n\n")
		}
	}

	conversationText.WriteString("\nProvide a concise summary that captures the essence of this conversation.")

	// Call LLM to generate summary
	req := protocol.ChatRequest{
		Model: model,
		Messages: []protocol.Message{
			{
				Role:    protocol.RoleUser,
				Content: conversationText.String(),
			},
		},
		MaxTokens: 500,
	}

	e2eStart := time.Now()
	resp, err := client.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	exutil.LogChat("summary", resp, e2eStart)

	summaryText := resp.FirstContent()

	// Create round summary
	return &memory.RoundSummary{
		RoundID:      stats.RoundID,
		Summary:      summaryText,
		KeyPoints:    []string{},
		Messages:     stats.MessageCount,
		Thoughts:     stats.ThoughtCount,
		Actions:      stats.ActionCount,
		Observations: stats.ObservationCount,
		Timestamp:    time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}, nil
}

// showL1State displays the current state of L1 working memory
func showL1State(wm *memory.WorkingMemory) {
	stats := wm.GetStats()
	fmt.Printf("  Round ID: %s\n", stats.RoundID)
	fmt.Printf("  Messages: %d\n", stats.MessageCount)
	fmt.Printf("  Thoughts: %d\n", stats.ThoughtCount)
	fmt.Printf("  Actions: %d\n", stats.ActionCount)
	fmt.Printf("  Observations: %d\n", stats.ObservationCount)
	fmt.Printf("  Duration: %v\n", stats.Duration)
}

// showL2State displays the current state of L2 short-term memory
func showL2State(stm *memory.ShortTermMemory) {
	stats := stm.GetStats()
	fmt.Printf("  Stored Rounds: %d/%d\n", stats.StoredRounds, stats.MaxRounds)
	fmt.Printf("  TTL: %v\n", stats.TTL)

	if stats.StoredRounds > 0 {
		fmt.Printf("  Stored Rounds:\n")
		summaries := stm.GetAllSummaries()
		for i, summary := range summaries {
			fmt.Printf("    %d. %s (at %s)\n", i+1, summary.RoundID, summary.Timestamp.Format("15:04:05"))
			fmt.Printf("       Summary: %s\n", summary.Summary)
			if len(summary.KeyPoints) > 0 {
				fmt.Printf("       Key Points: %v\n", summary.KeyPoints)
			}
		}
	}
}
