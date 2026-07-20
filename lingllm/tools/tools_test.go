package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/LingByte/lingllm/protocol"
)

type testChatModel struct {
	responses []*protocol.ChatResponse
	calls     int
}

func (m *testChatModel) Name() string { return "test" }

func (m *testChatModel) Chat(ctx context.Context, req protocol.ChatRequest) (*protocol.ChatResponse, error) {
	if m.calls >= len(m.responses) {
		return &protocol.ChatResponse{
			Choices: []protocol.Choice{{Message: protocol.Message{Role: protocol.RoleAssistant, Content: "done"}}},
		}, nil
	}
	resp := m.responses[m.calls]
	m.calls++
	return resp, nil
}

func (m *testChatModel) StreamChat(ctx context.Context, req protocol.ChatRequest) (protocol.ChatStream, error) {
	return nil, errors.New("not implemented")
}

func TestNewSimpleToolExecutor(t *testing.T) {
	executor := NewSimpleToolExecutor()
	if executor == nil {
		t.Fatal("NewSimpleToolExecutor returned nil")
	}
	if len(executor.GetTools()) != 0 {
		t.Error("expected empty tools list")
	}
}

func TestRegisterAndExecuteTool(t *testing.T) {
	executor := NewSimpleToolExecutor()
	executor.RegisterTool(CalculatorTool(), func(args json.RawMessage) (string, error) {
		return "Result: 42", nil
	})

	result, err := executor.Execute(context.Background(), "calculate", []byte(`{"expression":"2+2"}`))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result != "Result: 42" {
		t.Errorf("unexpected result: %s", result)
	}

	tools := executor.GetTools()
	if len(tools) != 1 || tools[0].Name != "calculate" {
		t.Errorf("unexpected tools: %+v", tools)
	}
}

func TestExecuteToolNotFound(t *testing.T) {
	executor := NewSimpleToolExecutor()
	_, err := executor.Execute(context.Background(), "missing", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
}

func TestGetToolsMultiple(t *testing.T) {
	executor := NewSimpleToolExecutor()
	executor.RegisterTool(CalculatorTool(), func(json.RawMessage) (string, error) { return "42", nil })
	executor.RegisterTool(WeatherTool(), func(json.RawMessage) (string, error) { return "Sunny", nil })
	if len(executor.GetTools()) != 2 {
		t.Errorf("expected 2 tools, got %d", len(executor.GetTools()))
	}
}

func TestNewToolChain(t *testing.T) {
	tc := NewToolChain(&testChatModel{}, NewSimpleToolExecutor())
	if tc == nil {
		t.Fatal("NewToolChain returned nil")
	}
	if tc.MaxRounds() != 5 {
		t.Errorf("expected maxRounds 5, got %d", tc.MaxRounds())
	}
}

func TestWithMaxRounds(t *testing.T) {
	tc := NewToolChain(&testChatModel{}, NewSimpleToolExecutor()).WithMaxRounds(10)
	if tc.MaxRounds() != 10 {
		t.Errorf("expected maxRounds 10, got %d", tc.MaxRounds())
	}
}

func TestExecuteWithToolsNoToolCalls(t *testing.T) {
	model := &testChatModel{
		responses: []*protocol.ChatResponse{{
			Choices: []protocol.Choice{{Message: protocol.Message{Role: protocol.RoleAssistant, Content: "Hello"}}},
		}},
	}
	tc := NewToolChain(model, NewSimpleToolExecutor())
	resp, err := tc.ExecuteWithTools(context.Background(), protocol.ChatRequest{
		Model:    "test",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ExecuteWithTools failed: %v", err)
	}
	if resp.FirstContent() != "Hello" {
		t.Errorf("unexpected content: %s", resp.FirstContent())
	}
}

func TestExecuteWithToolsWithToolCall(t *testing.T) {
	executor := NewSimpleToolExecutor()
	executor.RegisterTool(CalculatorTool(), func(json.RawMessage) (string, error) {
		return "4", nil
	})

	model := &testChatModel{
		responses: []*protocol.ChatResponse{
			{
				Choices: []protocol.Choice{{
					Message: protocol.Message{
						Role: protocol.RoleAssistant,
						ToolCalls: []protocol.ToolCall{
							{ID: "call_1", Type: "function", Function: protocol.FunctionCall{Name: "calculate", Arguments: []byte("{}")}},
						},
					},
				}},
			},
			{Choices: []protocol.Choice{{Message: protocol.Message{Role: protocol.RoleAssistant, Content: "The answer is 4"}}}},
		},
	}

	tc := NewToolChain(model, executor)
	resp, err := tc.ExecuteWithTools(context.Background(), protocol.ChatRequest{
		Model:    "test",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "calc"}},
	})
	if err != nil {
		t.Fatalf("ExecuteWithTools failed: %v", err)
	}
	if resp.FirstContent() != "The answer is 4" {
		t.Errorf("unexpected content: %s", resp.FirstContent())
	}
}

func TestExecuteWithToolsMaxRoundsExceeded(t *testing.T) {
	model := &testChatModel{
		responses: []*protocol.ChatResponse{
			{
				Choices: []protocol.Choice{{
					Message: protocol.Message{
						Role: protocol.RoleAssistant,
						ToolCalls: []protocol.ToolCall{
							{ID: "call_1", Type: "function", Function: protocol.FunctionCall{Name: "calculate", Arguments: []byte("{}")}},
						},
					},
				}},
			},
			{
				Choices: []protocol.Choice{{
					Message: protocol.Message{
						Role: protocol.RoleAssistant,
						ToolCalls: []protocol.ToolCall{
							{ID: "call_2", Type: "function", Function: protocol.FunctionCall{Name: "calculate", Arguments: []byte("{}")}},
						},
					},
				}},
			},
		},
	}
	executor := NewSimpleToolExecutor()
	executor.RegisterTool(CalculatorTool(), func(json.RawMessage) (string, error) { return "ok", nil })

	tc := NewToolChain(model, executor).WithMaxRounds(1)
	_, err := tc.ExecuteWithTools(context.Background(), protocol.ChatRequest{
		Model:    "test",
		Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "calc"}},
	})
	if err == nil {
		t.Fatal("expected max rounds error")
	}
}

func TestMakeToolAndPresets(t *testing.T) {
	tool := MakeTool("test", "desc", map[string]interface{}{"type": "object"})
	if tool.Name != "test" || tool.Description != "desc" {
		t.Errorf("unexpected tool: %+v", tool)
	}
	for _, preset := range []struct {
		name string
		fn   func() protocol.Tool
	}{
		{"calculate", CalculatorTool},
		{"get_weather", WeatherTool},
		{"web_search", SearchTool},
	} {
		t.Run(preset.name, func(t *testing.T) {
			tool := preset.fn()
			if tool.Name != preset.name || tool.Description == "" {
				t.Errorf("unexpected preset tool: %+v", tool)
			}
		})
	}
}

func TestReActToolCallParser(t *testing.T) {
	parser := &ReActToolCallParser{}
	response := &protocol.ChatResponse{
		Choices: []protocol.Choice{{
			Message: protocol.Message{
				Role: protocol.RoleAssistant,
				Content: `Thought: calc
Action: calculate
Action Input: {"expression":"2+2"}`,
			},
		}},
	}
	calls, err := parser.Parse(context.Background(), response)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(calls) != 1 || calls[0].Function.Name != "calculate" {
		t.Errorf("unexpected calls: %+v", calls)
	}
}

func TestReActToolCallParserNoChoices(t *testing.T) {
	parser := &ReActToolCallParser{}
	_, err := parser.Parse(context.Background(), &protocol.ChatResponse{})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestJSONToolCallParser(t *testing.T) {
	parser := &JSONToolCallParser{}
	response := &protocol.ChatResponse{
		Choices: []protocol.Choice{{
			Message: protocol.Message{
				Role:    protocol.RoleAssistant,
				Content: `{"tool_calls": [{"name": "calculate", "input": {"expression": "2+2"}}]}`,
			},
		}},
	}
	calls, err := parser.Parse(context.Background(), response)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(calls) != 1 || calls[0].Function.Name != "calculate" {
		t.Errorf("unexpected calls: %+v", calls)
	}
}

func TestJSONToolCallParserInvalidJSON(t *testing.T) {
	parser := &JSONToolCallParser{}
	response := &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "not json"}}},
	}
	_, err := parser.Parse(context.Background(), response)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestToolCallExtractor(t *testing.T) {
	extractor := NewToolCallExtractor()
	response := &protocol.ChatResponse{
		Choices: []protocol.Choice{{
			Message: protocol.Message{
				Content: `{"tool_calls": [{"name": "calculate", "input": {"expression": "2+2"}}]}`,
			},
		}},
	}
	calls, err := extractor.Extract(context.Background(), response)
	if err != nil || len(calls) != 1 {
		t.Fatalf("Extract failed: err=%v calls=%+v", err, calls)
	}
}

func TestToolCallExtractorReActFallback(t *testing.T) {
	extractor := NewToolCallExtractor()
	response := &protocol.ChatResponse{
		Choices: []protocol.Choice{{
			Message: protocol.Message{
				Content: "Action: calculate\nAction Input: {\"expression\":\"2+2\"}",
			},
		}},
	}
	calls, err := extractor.Extract(context.Background(), response)
	if err != nil || len(calls) != 1 {
		t.Fatalf("Extract fallback failed: err=%v calls=%+v", err, calls)
	}
}

func TestToolCallExtractorNoToolCalls(t *testing.T) {
	extractor := NewToolCallExtractor()
	response := &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "plain text"}}},
	}
	_, err := extractor.Extract(context.Background(), response)
	if err == nil {
		t.Fatal("expected error when no tool calls found")
	}
}

func TestAddParser(t *testing.T) {
	extractor := NewToolCallExtractor().AddParser(&ReActToolCallParser{})
	if len(extractor.parsers) != 3 {
		t.Errorf("expected 3 parsers, got %d", len(extractor.parsers))
	}
}

func TestExecuteWithToolsToolError(t *testing.T) {
	executor := NewSimpleToolExecutor()
	executor.RegisterTool(CalculatorTool(), func(json.RawMessage) (string, error) {
		return "", context.Canceled
	})

	model := &testChatModel{
		responses: []*protocol.ChatResponse{
			{
				Choices: []protocol.Choice{{
					Message: protocol.Message{
						Role: protocol.RoleAssistant,
						ToolCalls: []protocol.ToolCall{
							{ID: "call_1", Type: "function", Function: protocol.FunctionCall{Name: "calculate", Arguments: []byte("{}")}},
						},
					},
				}},
			},
			{Choices: []protocol.Choice{{Message: protocol.Message{Content: "done"}}}},
		},
	}

	tc := NewToolChain(model, executor)
	resp, err := tc.ExecuteWithTools(context.Background(), protocol.ChatRequest{
		Model: "test", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FirstContent() != "done" {
		t.Errorf("unexpected content: %s", resp.FirstContent())
	}
}

func TestJSONToolCallParserSkipsInvalidEntries(t *testing.T) {
	parser := &JSONToolCallParser{}
	response := &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{
			Content: `{"tool_calls": ["bad", {"name": "calculate", "input": {"expression": "1"}}]}`,
		}}},
	}
	calls, err := parser.Parse(context.Background(), response)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(calls) != 1 || calls[0].Function.Name != "calculate" {
		t.Fatalf("unexpected calls: %+v", calls)
	}
}

func TestExecuteWithToolsModelError(t *testing.T) {
	tc := NewToolChain(&errorChatModel{}, NewSimpleToolExecutor())
	_, err := tc.ExecuteWithTools(context.Background(), protocol.ChatRequest{
		Model: "test", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected model error")
	}
}

type errorChatModel struct{}

func (m *errorChatModel) Name() string { return "err" }
func (m *errorChatModel) Chat(ctx context.Context, req protocol.ChatRequest) (*protocol.ChatResponse, error) {
	return nil, context.Canceled
}
func (m *errorChatModel) StreamChat(ctx context.Context, req protocol.ChatRequest) (protocol.ChatStream, error) {
	return nil, context.Canceled
}

func TestJSONToolCallParserNoChoices(t *testing.T) {
	parser := &JSONToolCallParser{}
	_, err := parser.Parse(context.Background(), &protocol.ChatResponse{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestJSONToolCallParserMissingToolCalls(t *testing.T) {
	parser := &JSONToolCallParser{}
	response := &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: `{"other": true}`}}},
	}
	_, err := parser.Parse(context.Background(), response)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReActToolCallParserEmptyActions(t *testing.T) {
	parser := &ReActToolCallParser{}
	response := &protocol.ChatResponse{
		Choices: []protocol.Choice{{Message: protocol.Message{Content: "Thought: nothing"}}},
	}
	calls, err := parser.Parse(context.Background(), response)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("expected no calls, got %+v", calls)
	}
}

func TestExecuteWithToolsEmptyToolResults(t *testing.T) {
	model := &testChatModel{
		responses: []*protocol.ChatResponse{
			{Choices: []protocol.Choice{{Message: protocol.Message{Role: protocol.RoleAssistant, ToolCallID: "x", Content: "has-content"}}}},
		},
	}
	tc := NewToolChain(model, NewSimpleToolExecutor())
	resp, err := tc.ExecuteWithTools(context.Background(), protocol.ChatRequest{
		Model: "test", Messages: []protocol.Message{{Role: protocol.RoleUser, Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FirstContent() != "has-content" {
		t.Errorf("unexpected response: %s", resp.FirstContent())
	}
}
