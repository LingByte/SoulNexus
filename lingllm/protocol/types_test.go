package protocol

import (
	"encoding/json"
	"testing"
)

func TestMessageRoles(t *testing.T) {
	if string(RoleUser) != "user" {
		t.Errorf("RoleUser should be 'user'")
	}
	if string(RoleAssistant) != "assistant" {
		t.Errorf("RoleAssistant should be 'assistant'")
	}
	if string(RoleSystem) != "system" {
		t.Errorf("RoleSystem should be 'system'")
	}
	if string(RoleTool) != "tool" {
		t.Errorf("RoleTool should be 'tool'")
	}
}

func TestNewMessage(t *testing.T) {
	msg := Message{
		Role:    RoleUser,
		Content: "Hello",
	}

	if msg.Role != RoleUser {
		t.Errorf("Expected role %s, got %s", RoleUser, msg.Role)
	}
	if msg.Content != "Hello" {
		t.Errorf("Expected content 'Hello', got '%s'", msg.Content)
	}
}

func TestToolChoiceValues(t *testing.T) {
	if string(ToolChoiceAuto) != "auto" {
		t.Errorf("ToolChoiceAuto should be 'auto'")
	}
	if string(ToolChoiceRequired) != "required" {
		t.Errorf("ToolChoiceRequired should be 'required'")
	}
	if string(ToolChoiceNone) != "none" {
		t.Errorf("ToolChoiceNone should be 'none'")
	}
}

func TestChatRequest(t *testing.T) {
	req := ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: RoleUser, Content: "Hello"},
		},
	}

	if req.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", req.Model)
	}
	if len(req.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(req.Messages))
	}
}

func TestChatResponse(t *testing.T) {
	resp := ChatResponse{
		Model: "gpt-4",
		Choices: []Choice{
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "Hello back",
				},
			},
		},
	}

	if resp.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", resp.Model)
	}
	if len(resp.Choices) != 1 {
		t.Errorf("Expected 1 choice, got %d", len(resp.Choices))
	}
}

func TestChatResponseFirstContent(t *testing.T) {
	resp := ChatResponse{
		Choices: []Choice{
			{
				Message: Message{
					Role:    RoleAssistant,
					Content: "First response",
				},
			},
		},
	}

	content := resp.FirstContent()
	if content != "First response" {
		t.Errorf("Expected 'First response', got '%s'", content)
	}
}

func TestChatResponseFirstContentEmpty(t *testing.T) {
	resp := ChatResponse{
		Choices: []Choice{},
	}

	content := resp.FirstContent()
	if content != "" {
		t.Errorf("Expected empty string, got '%s'", content)
	}
}

func TestTool(t *testing.T) {
	tool := Tool{
		Name:        "calculator",
		Description: "A simple calculator",
		Parameters: map[string]interface{}{
			"type": "object",
		},
	}

	if tool.Name != "calculator" {
		t.Errorf("Expected name 'calculator', got '%s'", tool.Name)
	}
	if tool.Description != "A simple calculator" {
		t.Errorf("Expected description 'A simple calculator', got '%s'", tool.Description)
	}
}

func TestChoice(t *testing.T) {
	choice := Choice{
		Index: 0,
		Message: Message{
			Role:    RoleAssistant,
			Content: "Response",
		},
		FinishReason: "stop",
	}

	if choice.Index != 0 {
		t.Errorf("Expected index 0, got %d", choice.Index)
	}
	if choice.Message.Content != "Response" {
		t.Errorf("Expected content 'Response', got '%s'", choice.Message.Content)
	}
	if choice.FinishReason != "stop" {
		t.Errorf("Expected finish reason 'stop', got '%s'", choice.FinishReason)
	}
}

func TestProviderConstants(t *testing.T) {
	if string(ProviderOpenAI) != "openai" {
		t.Errorf("ProviderOpenAI should be 'openai'")
	}
	if string(ProviderAnthropic) != "anthropic" {
		t.Errorf("ProviderAnthropic should be 'anthropic'")
	}
	if string(ProviderOllama) != "ollama" {
		t.Errorf("ProviderOllama should be 'ollama'")
	}
	if string(ProviderOpenAIResponse) != "openai-response" {
		t.Errorf("ProviderOpenAIResponse should be 'openai-response'")
	}
}

func TestClientConfig(t *testing.T) {
	config := ClientConfig{
		Provider: ProviderOpenAI,
		APIKey:   "test-key",
		BaseURL:  "https://api.openai.com/v1",
	}

	if config.Provider != ProviderOpenAI {
		t.Errorf("Expected provider %s, got %s", ProviderOpenAI, config.Provider)
	}
	if config.APIKey != "test-key" {
		t.Errorf("Expected API key 'test-key', got '%s'", config.APIKey)
	}
}
func TestValidate(t *testing.T) {
	var nilReq *ChatRequest
	if err := nilReq.Validate(); err == nil {
		t.Fatal("expected error for nil request")
	}

	req := ChatRequest{}
	if err := req.Validate(); err == nil {
		t.Fatal("expected error for missing model")
	}

	req = ChatRequest{Model: "gpt-4"}
	if err := req.Validate(); err == nil {
		t.Fatal("expected error for missing messages")
	}

	req = ChatRequest{
		Model:    "gpt-4",
		Messages: []Message{{Content: "hi"}},
	}
	if err := req.Validate(); err == nil {
		t.Fatal("expected error for missing role")
	}

	req = ChatRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: RoleUser}},
	}
	if err := req.Validate(); err == nil {
		t.Fatal("expected error for missing content")
	}

	req = ChatRequest{
		Model:    "gpt-4",
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request, got %v", err)
	}

	req = ChatRequest{
		Model: "gpt-4",
		Messages: []Message{
			{Role: RoleUser, Content: "hi"},
			{Role: RoleAssistant, ToolCalls: []ToolCall{{ID: "c1", Type: "function", Function: FunctionCall{Name: "t", Arguments: json.RawMessage(`{}`)}}}},
			{Role: RoleTool, ToolCallID: "c1", Content: "ok"},
		},
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid tool-turn request, got %v", err)
	}
}

func TestMessageHelpers(t *testing.T) {
	if UserMessage("u").Role != RoleUser {
		t.Error("UserMessage failed")
	}
	if SystemMessage("s").Role != RoleSystem {
		t.Error("SystemMessage failed")
	}
	if AssistantMessage("a").Role != RoleAssistant {
		t.Error("AssistantMessage failed")
	}
	tool := ToolMessage("result", "call_1")
	if tool.Role != RoleTool || tool.ToolCallID != "call_1" {
		t.Errorf("ToolMessage failed: %+v", tool)
	}
}

func TestNewChatRequestAndWithMethods(t *testing.T) {
	req := NewChatRequest("gpt-4", UserMessage("hi")).
		WithMaxTokens(100).
		WithTemperature(0.7).
		WithTopP(0.9).
		WithStop("END").
		WithMetadata("k", "v")

	if req.Model != "gpt-4" || req.MaxTokens != 100 || req.Temperature != 0.7 || req.TopP != 0.9 {
		t.Errorf("unexpected request fields: %+v", req)
	}
	if len(req.Stop) != 1 || req.Metadata["k"] != "v" {
		t.Errorf("unexpected stop/metadata: %+v", req)
	}
}

func TestFirstMessage(t *testing.T) {
	resp := ChatResponse{Choices: []Choice{{Message: Message{Content: "x"}}}}
	if resp.FirstMessage().Content != "x" {
		t.Error("FirstMessage failed")
	}
	empty := ChatResponse{}
	if empty.FirstMessage() != nil {
		t.Error("expected nil FirstMessage for empty choices")
	}
}
