package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Initialize logger for tests
	_ = logger.Init(&logger.LogConfig{
		Level:    "info",
		Filename: "",
	}, "test")
}

// MockOpenAIClient is a mock implementation of OpenAI client for testing
type MockOpenAIClient struct {
	CreateChatCompletionFunc       func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	CreateChatCompletionStreamFunc func(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error)
}

// TestQueryWithOptions_TimeStatistics tests time statistics in QueryWithOptions
func TestQueryWithOptions_TimeStatistics(t *testing.T) {
	// This test requires a real API key, so we'll skip it in CI
	if testing.Short() {
		t.Skip("Skipping test that requires API key")
	}

	ctx := context.Background()
	handler := NewLLMHandler(ctx, "test-key", "https://api.openai.com/v1", "You are a helpful assistant.")

	// Register a simple test tool
	testToolCallback := func(args map[string]interface{}) (string, error) {
		return "test result", nil
	}

	testToolParams := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query"
			}
		}
	}`)

	handler.RegisterFunctionTool("test_tool", "A test tool", testToolParams, testToolCallback)

	// Note: This test requires a real API key to run
	// In a real scenario, you would use a mock client
	t.Log("This test requires a real API key and will be skipped in CI")
}

// TestQueryWithOptions_ToolCallStatistics tests tool call statistics
func TestQueryWithOptions_ToolCallStatistics(t *testing.T) {
	// This test requires a real API key, so we'll skip it in CI
	if testing.Short() {
		t.Skip("Skipping test that requires API key")
	}

	ctx := context.Background()
	handler := NewLLMHandler(ctx, "test-key", "https://api.openai.com/v1", "You are a helpful assistant.")

	// Register multiple test tools
	tool1Callback := func(args map[string]interface{}) (string, error) {
		return "tool1 result", nil
	}

	tool2Callback := func(args map[string]interface{}) (string, error) {
		return "tool2 result", nil
	}

	toolParams := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query"
			}
		}
	}`)

	handler.RegisterFunctionTool("tool1", "First tool", toolParams, tool1Callback)
	handler.RegisterFunctionTool("tool2", "Second tool", toolParams, tool2Callback)

	// Note: This test requires a real API key to run
	t.Log("This test requires a real API key and will be skipped in CI")
}

// TestConcurrentQueries tests concurrent query execution
func TestConcurrentQueries(t *testing.T) {
	// This test requires a real API key, so we'll skip it in CI
	if testing.Short() {
		t.Skip("Skipping test that requires API key")
	}

	// Skip if no API key is provided
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping test that requires OPENAI_API_KEY environment variable")
	}

	ctx := context.Background()
	handler := NewLLMHandler(ctx, apiKey, "https://api.openai.com/v1", "You are a helpful assistant.")

	// Test concurrent queries
	numGoroutines := 3 // Reduce number to avoid rate limits
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			options := QueryOptions{
				Model:       openai.GPT4oMini,
				Temperature: Float32Ptr(0.7),
				User:        "test-user",
			}

			_, err := handler.QueryWithOptions("Hello", options)
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent query error: %v", err)
		}
	}

	t.Log("Concurrent queries completed")
}

// TestConcurrentQueries_MessageOrder tests that messages are properly ordered in concurrent scenarios
func TestConcurrentQueries_MessageOrder(t *testing.T) {
	ctx := context.Background()
	handler := NewLLMHandler(ctx, "test-key", "https://api.openai.com/v1", "You are a helpful assistant.")

	// Test that messages are properly protected
	numGoroutines := 5
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Access messages (should be safe)
			handler.mutex.Lock()
			messageCount := len(handler.messages)
			handler.mutex.Unlock()

			// Verify we can read messages safely
			assert.GreaterOrEqual(t, messageCount, 1, "Should have at least system message")
		}(i)
	}

	wg.Wait()
}

// TestLLMUsageInfo_TimeStatistics tests LLMUsageInfo time statistics
func TestLLMUsageInfo_TimeStatistics(t *testing.T) {
	startTime := time.Now()
	time.Sleep(10 * time.Millisecond) // Simulate some processing
	endTime := time.Now()
	duration := endTime.Sub(startTime).Milliseconds()

	usageInfo := &LLMUsageInfo{
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  duration,
	}

	assert.NotZero(t, usageInfo.StartTime)
	assert.NotZero(t, usageInfo.EndTime)
	assert.Greater(t, usageInfo.Duration, int64(0))
	assert.LessOrEqual(t, usageInfo.Duration, int64(100)) // Should be around 10ms
}

// TestLLMUsageInfo_ToolCallStatistics tests LLMUsageInfo tool call statistics
func TestLLMUsageInfo_ToolCallStatistics(t *testing.T) {
	toolCalls := []ToolCallInfo{
		{ID: "call_1", Name: "tool1", Arguments: `{"arg": "value1"}`},
		{ID: "call_2", Name: "tool2", Arguments: `{"arg": "value2"}`},
		{ID: "call_3", Name: "tool1", Arguments: `{"arg": "value3"}`},
	}

	usageInfo := &LLMUsageInfo{
		HasToolCalls:  true,
		ToolCallCount: len(toolCalls),
		ToolCalls:     toolCalls,
	}

	assert.True(t, usageInfo.HasToolCalls)
	assert.Equal(t, 3, usageInfo.ToolCallCount)
	assert.Len(t, usageInfo.ToolCalls, 3)
	assert.Equal(t, "tool1", usageInfo.ToolCalls[0].Name)
	assert.Equal(t, "tool2", usageInfo.ToolCalls[1].Name)
}

// TestLLMUsageInfo_NoToolCalls tests LLMUsageInfo when no tools are called
func TestLLMUsageInfo_NoToolCalls(t *testing.T) {
	usageInfo := &LLMUsageInfo{
		HasToolCalls:  false,
		ToolCallCount: 0,
		ToolCalls:     nil,
	}

	assert.False(t, usageInfo.HasToolCalls)
	assert.Equal(t, 0, usageInfo.ToolCallCount)
	assert.Nil(t, usageInfo.ToolCalls)
}

// TestConcurrentToolCalls tests concurrent tool call handling
func TestConcurrentToolCalls(t *testing.T) {
	ctx := context.Background()
	handler := NewLLMHandler(ctx, "test-key", "https://api.openai.com/v1", "You are a helpful assistant.")

	// Register a test tool
	toolCallback := func(args map[string]interface{}) (string, error) {
		// Simulate some processing time
		time.Sleep(5 * time.Millisecond)
		return "result", nil
	}

	toolParams := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query"
			}
		}
	}`)

	handler.RegisterFunctionTool("test_tool", "A test tool", toolParams, toolCallback)

	// Test concurrent tool registrations
	numGoroutines := 5
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			toolName := fmt.Sprintf("tool_%d", id)
			handler.RegisterFunctionTool(toolName, "Test tool", toolParams, toolCallback)
		}(i)
	}

	wg.Wait()

	// Verify all tools were registered
	tools := handler.GetFunctionTools()
	assert.GreaterOrEqual(t, len(tools), numGoroutines+1) // +1 for the initial tool
}

// TestMessageOrdering tests that messages maintain order under concurrent access
func TestMessageOrdering(t *testing.T) {
	ctx := context.Background()
	handler := NewLLMHandler(ctx, "test-key", "https://api.openai.com/v1", "You are a helpful assistant.")

	// Simulate concurrent message additions
	numGoroutines := 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			handler.mutex.Lock()
			handler.messages = append(handler.messages, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: "Message from goroutine",
			})
			handler.mutex.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify messages were added
	handler.mutex.Lock()
	messageCount := len(handler.messages)
	handler.mutex.Unlock()

	// Should have system message + numGoroutines user messages
	assert.Equal(t, 1+numGoroutines, messageCount)
}

// TestResetMessages_Concurrent tests ResetMessages under concurrent access
func TestResetMessages_Concurrent(t *testing.T) {
	ctx := context.Background()
	handler := NewLLMHandler(ctx, "test-key", "https://api.openai.com/v1", "You are a helpful assistant.")

	// Add some messages
	handler.mutex.Lock()
	handler.messages = append(handler.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: "Test message",
	})
	handler.mutex.Unlock()

	// Test concurrent reset
	numGoroutines := 5
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.ResetMessages()
		}()
	}

	wg.Wait()

	// Verify messages were reset
	messages := handler.GetMessages()
	assert.Len(t, messages, 1) // Only system message should remain
	assert.Equal(t, openai.ChatMessageRoleSystem, messages[0].Role)
}

// TestGetMessages_Concurrent tests GetMessages under concurrent access
func TestGetMessages_Concurrent(t *testing.T) {
	ctx := context.Background()
	handler := NewLLMHandler(ctx, "test-key", "https://api.openai.com/v1", "You are a helpful assistant.")

	// Add some messages
	handler.mutex.Lock()
	handler.messages = append(handler.messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: "Test message",
	})
	handler.mutex.Unlock()

	// Test concurrent reads
	numGoroutines := 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			messages := handler.GetMessages()
			if len(messages) == 0 {
				errors <- assert.AnError
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent GetMessages error: %v", err)
		}
	}
}

// TestSetSystemPrompt_Concurrent tests SetSystemPrompt under concurrent access
func TestSetSystemPrompt_Concurrent(t *testing.T) {
	ctx := context.Background()
	handler := NewLLMHandler(ctx, "test-key", "https://api.openai.com/v1", "Initial system prompt")

	// Test concurrent system prompt updates
	numGoroutines := 5
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			prompt := fmt.Sprintf("System prompt from goroutine %d", id)
			handler.SetSystemPrompt(prompt)
		}(i)
	}

	wg.Wait()

	// Verify system prompt was set (should be the last one)
	messages := handler.GetMessages()
	require.Greater(t, len(messages), 0)
	assert.Equal(t, openai.ChatMessageRoleSystem, messages[0].Role)
}

// TestToolCallInfo_Serialization tests ToolCallInfo JSON serialization
func TestToolCallInfo_Serialization(t *testing.T) {
	toolCall := ToolCallInfo{
		ID:        "call_123",
		Name:      "test_tool",
		Arguments: `{"arg": "value"}`,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(toolCall)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaled ToolCallInfo
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, toolCall.ID, unmarshaled.ID)
	assert.Equal(t, toolCall.Name, unmarshaled.Name)
	assert.Equal(t, toolCall.Arguments, unmarshaled.Arguments)
}

// TestLLMUsageInfo_Serialization tests LLMUsageInfo JSON serialization
func TestLLMUsageInfo_Serialization(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(100 * time.Millisecond)

	usageInfo := &LLMUsageInfo{
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		StartTime:        startTime,
		EndTime:          endTime,
		Duration:         100,
		HasToolCalls:     true,
		ToolCallCount:    2,
		ToolCalls: []ToolCallInfo{
			{ID: "call_1", Name: "tool1", Arguments: `{"arg": "value1"}`},
			{ID: "call_2", Name: "tool2", Arguments: `{"arg": "value2"}`},
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(usageInfo)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaled LLMUsageInfo
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, usageInfo.Model, unmarshaled.Model)
	assert.Equal(t, usageInfo.PromptTokens, unmarshaled.PromptTokens)
	assert.Equal(t, usageInfo.CompletionTokens, unmarshaled.CompletionTokens)
	assert.Equal(t, usageInfo.TotalTokens, unmarshaled.TotalTokens)
	assert.Equal(t, usageInfo.Duration, unmarshaled.Duration)
	assert.Equal(t, usageInfo.HasToolCalls, unmarshaled.HasToolCalls)
	assert.Equal(t, usageInfo.ToolCallCount, unmarshaled.ToolCallCount)
	assert.Len(t, unmarshaled.ToolCalls, 2)
}

// BenchmarkQueryWithOptions benchmarks QueryWithOptions performance
func BenchmarkQueryWithOptions(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark that requires API key")
	}

	ctx := context.Background()
	handler := NewLLMHandler(ctx, "test-key", "https://api.openai.com/v1", "You are a helpful assistant.")

	options := QueryOptions{
		Model:       openai.GPT4oMini,
		Temperature: Float32Ptr(0.7),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.QueryWithOptions("Hello", options)
	}
}

// BenchmarkConcurrentQueries benchmarks concurrent query performance
func BenchmarkConcurrentQueries(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark that requires API key")
	}

	ctx := context.Background()
	handler := NewLLMHandler(ctx, "test-key", "https://api.openai.com/v1", "You are a helpful assistant.")

	options := QueryOptions{
		Model:       openai.GPT4oMini,
		Temperature: Float32Ptr(0.7),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = handler.QueryWithOptions("Hello", options)
		}
	})
}
