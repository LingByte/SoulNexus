package realtime

import (
	"encoding/json"
)

// Tool is an OpenAI-style function tool definition for session.update (Qwen-Omni-Realtime).
type Tool struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// ToolHandler executes a tool on the WS read path. Must return quickly; heavy work
// (dial transfer, DB) should be delegated to another goroutine by the caller.
type ToolHandler func(name string, args map[string]any) string

// ToolsForSession builds the `tools` array for DashScope session.update.
func ToolsForSession(tools []Tool) []map[string]any {
	if len(tools) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		if t.Name == "" {
			continue
		}
		params := t.Parameters
		if len(params) == 0 {
			params = json.RawMessage(`{"type":"object","properties":{}}`)
		}
		fn := map[string]any{
			"name":        t.Name,
			"description": t.Description,
		}
		var paramsObj any
		if err := json.Unmarshal(params, &paramsObj); err == nil {
			fn["parameters"] = paramsObj
		} else {
			fn["parameters"] = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		out = append(out, map[string]any{
			"type":     "function",
			"function": fn,
		})
	}
	return out
}
