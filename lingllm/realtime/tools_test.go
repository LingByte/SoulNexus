package realtime

import (
	"encoding/json"
	"testing"
)

func TestToolStruct(t *testing.T) {
	params := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`)
	tool := Tool{
		Name:        "get_weather",
		Description: "Get the weather for a location",
		Parameters:  params,
	}

	if tool.Name != "get_weather" {
		t.Errorf("Name = %s, want 'get_weather'", tool.Name)
	}

	if tool.Description != "Get the weather for a location" {
		t.Errorf("Description = %s, want 'Get the weather for a location'", tool.Description)
	}

	if len(tool.Parameters) == 0 {
		t.Error("Parameters should not be empty")
	}
}

func TestToolHandlerFunc(t *testing.T) {
	handler := func(name string, args map[string]any) string {
		if name == "test_tool" {
			return "success"
		}
		return "failed"
	}

	result := handler("test_tool", map[string]any{})
	if result != "success" {
		t.Errorf("Handler result = %s, want 'success'", result)
	}

	result = handler("unknown_tool", map[string]any{})
	if result != "failed" {
		t.Errorf("Handler result = %s, want 'failed'", result)
	}
}

func TestToolsForSession_Empty(t *testing.T) {
	tools := []Tool{}
	result := ToolsForSession(tools)

	if result != nil {
		t.Errorf("ToolsForSession with empty tools should return nil, got %v", result)
	}
}

func TestToolsForSession_SingleTool(t *testing.T) {
	params := json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`)
	tools := []Tool{
		{
			Name:        "get_weather",
			Description: "Get weather information",
			Parameters:  params,
		},
	}

	result := ToolsForSession(tools)

	if len(result) != 1 {
		t.Errorf("ToolsForSession result length = %d, want 1", len(result))
	}

	tool := result[0]
	if toolType, ok := tool["type"]; !ok || toolType != "function" {
		t.Errorf("Tool type = %v, want 'function'", toolType)
	}

	if fn, ok := tool["function"]; !ok {
		t.Error("Tool should have 'function' field")
	} else {
		fnMap := fn.(map[string]any)
		if fnMap["name"] != "get_weather" {
			t.Errorf("Function name = %v, want 'get_weather'", fnMap["name"])
		}
	}
}

func TestToolsForSession_MultipleTools(t *testing.T) {
	tools := []Tool{
		{
			Name:        "tool1",
			Description: "First tool",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		},
		{
			Name:        "tool2",
			Description: "Second tool",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		},
		{
			Name:        "tool3",
			Description: "Third tool",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		},
	}

	result := ToolsForSession(tools)

	if len(result) != 3 {
		t.Errorf("ToolsForSession result length = %d, want 3", len(result))
	}
}

func TestToolsForSession_SkipsEmptyNames(t *testing.T) {
	tools := []Tool{
		{
			Name:        "valid_tool",
			Description: "Valid tool",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		},
		{
			Name:        "",
			Description: "Tool with empty name",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		},
		{
			Name:        "another_valid",
			Description: "Another valid tool",
			Parameters:  json.RawMessage(`{"type":"object"}`),
		},
	}

	result := ToolsForSession(tools)

	if len(result) != 2 {
		t.Errorf("ToolsForSession result length = %d, want 2 (should skip empty name)", len(result))
	}
}

func TestToolsForSession_DefaultParameters(t *testing.T) {
	tools := []Tool{
		{
			Name:        "tool_no_params",
			Description: "Tool without parameters",
			Parameters:  nil,
		},
	}

	result := ToolsForSession(tools)

	if len(result) != 1 {
		t.Fatalf("ToolsForSession result length = %d, want 1", len(result))
	}

	tool := result[0]
	fn := tool["function"].(map[string]any)
	params := fn["parameters"]

	// Should have default parameters
	if params == nil {
		t.Error("Parameters should not be nil (should have defaults)")
	}
}

func TestToolsForSession_InvalidJSON(t *testing.T) {
	tools := []Tool{
		{
			Name:        "tool_invalid_json",
			Description: "Tool with invalid JSON",
			Parameters:  json.RawMessage(`{invalid json}`),
		},
	}

	result := ToolsForSession(tools)

	if len(result) != 1 {
		t.Fatalf("ToolsForSession result length = %d, want 1", len(result))
	}

	tool := result[0]
	fn := tool["function"].(map[string]any)
	params := fn["parameters"]

	// Should have default parameters when JSON is invalid
	if params == nil {
		t.Error("Parameters should not be nil (should have defaults for invalid JSON)")
	}
}
