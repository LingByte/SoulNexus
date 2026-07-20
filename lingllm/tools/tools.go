package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/lingllm/protocol"
)

// ToolCall represents a tool invocation by the model.
// Deprecated: Use protocol.ToolCall instead.
type ToolCall = protocol.ToolCall

// FunctionCall represents a function call with name and arguments.
// Deprecated: Use protocol.FunctionCall instead.
type FunctionCall = protocol.FunctionCall

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error,omitempty"`
}

// ToolExecutor executes tools and returns results.
type ToolExecutor interface {
	Execute(ctx context.Context, toolName string, args json.RawMessage) (string, error)
	GetTools() []protocol.Tool
}

// SimpleToolExecutor is a basic tool executor using a map of functions.
type SimpleToolExecutor struct {
	tools map[string]protocol.Tool
	funcs map[string]func(json.RawMessage) (string, error)
}

// NewSimpleToolExecutor creates a new simple tool executor.
func NewSimpleToolExecutor() *SimpleToolExecutor {
	return &SimpleToolExecutor{
		tools: make(map[string]protocol.Tool),
		funcs: make(map[string]func(json.RawMessage) (string, error)),
	}
}

// RegisterTool registers a tool with its implementation.
func (e *SimpleToolExecutor) RegisterTool(tool protocol.Tool, fn func(json.RawMessage) (string, error)) {
	e.tools[tool.Name] = tool
	e.funcs[tool.Name] = fn
}

// Execute runs a tool by name.
func (e *SimpleToolExecutor) Execute(ctx context.Context, toolName string, args json.RawMessage) (string, error) {
	fn, ok := e.funcs[toolName]
	if !ok {
		return "", fmt.Errorf("tool %s not found", toolName)
	}
	return fn(args)
}

// GetTools returns all registered tools.
func (e *SimpleToolExecutor) GetTools() []protocol.Tool {
	tools := make([]protocol.Tool, 0, len(e.tools))
	for _, tool := range e.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ToolChain manages tool calling in a conversation.
type ToolChain struct {
	executor  ToolExecutor
	model     protocol.ChatModel
	maxRounds int
}

// NewToolChain creates a new tool chain.
func NewToolChain(model protocol.ChatModel, executor ToolExecutor) *ToolChain {
	return &ToolChain{
		executor:  executor,
		model:     model,
		maxRounds: 5,
	}
}

// WithMaxRounds sets the maximum number of tool calling rounds.
func (tc *ToolChain) WithMaxRounds(maxRounds int) *ToolChain {
	tc.maxRounds = maxRounds
	return tc
}

// MaxRounds returns the configured maximum tool calling rounds.
func (tc *ToolChain) MaxRounds() int {
	return tc.maxRounds
}

// ExecuteWithTools runs a chat request with tool support.
func (tc *ToolChain) ExecuteWithTools(ctx context.Context, req protocol.ChatRequest) (*protocol.ChatResponse, error) {
	req.Tools = tc.executor.GetTools()
	if len(req.Tools) > 0 {
		req.ToolChoice = protocol.ToolChoiceAuto
	}

	messages := req.Messages
	rounds := 0

	for rounds < tc.maxRounds {
		resp, err := tc.model.Chat(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("model call failed: %w", err)
		}

		// Extract tool calls from response
		toolCalls := tc.extractToolCalls(resp)

		if len(toolCalls) == 0 {
			return resp, nil
		}

		// Execute tool calls
		toolResults := make([]protocol.Message, 0, len(toolCalls))
		for _, call := range toolCalls {
			result, execErr := tc.executor.Execute(ctx, call.Function.Name, call.Function.Arguments)
			if execErr != nil {
				result = fmt.Sprintf("Error: %v", execErr)
			}
			toolResults = append(toolResults, protocol.Message{
				Role:       protocol.RoleTool,
				Content:    result,
				ToolCallID: call.ID,
			})
		}

		// Add assistant message with tool calls to history
		assistantMsg := protocol.Message{
			Role:      protocol.RoleAssistant,
			Content:   resp.FirstContent(),
			ToolCalls: toolCalls,
		}
		messages = append(messages, assistantMsg)

		// Add tool results to history
		messages = append(messages, toolResults...)

		// Prepare next request
		req = protocol.ChatRequest{
			Model:       req.Model,
			Messages:    messages,
			Tools:       tc.executor.GetTools(),
			ToolChoice:  protocol.ToolChoiceAuto,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			TopP:        req.TopP,
			Stop:        req.Stop,
			Metadata:    req.Metadata,
		}
		rounds++
	}

	return nil, fmt.Errorf("max tool calling rounds (%d) exceeded", tc.maxRounds)
}

// extractToolCalls extracts tool calls from a response.
func (tc *ToolChain) extractToolCalls(resp *protocol.ChatResponse) []protocol.ToolCall {
	var toolCalls []protocol.ToolCall

	for _, choice := range resp.Choices {
		// Check Message.ToolCalls (OpenAI/Anthropic format)
		if len(choice.Message.ToolCalls) > 0 {
			toolCalls = append(toolCalls, choice.Message.ToolCalls...)
		}

		// Check Message.Content for ReAct-style tool calls
		if choice.Message.Content != "" && len(choice.Message.ToolCalls) == 0 {
			// Try to parse ReAct-style format: Action: tool_name\nAction Input: {...}
			calls := tc.parseReActStyle(choice.Message.Content)
			toolCalls = append(toolCalls, calls...)
		}
	}

	return toolCalls
}

// parseReActStyle parses tool calls from ReAct-style content.
func (tc *ToolChain) parseReActStyle(content string) []protocol.ToolCall {
	var toolCalls []protocol.ToolCall
	lines := strings.Split(content, "\n")

	var currentAction, currentInput string
	var callID int

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Action:") {
			currentAction = strings.TrimPrefix(line, "Action:")
			currentAction = strings.TrimSpace(currentAction)
		} else if strings.HasPrefix(line, "Action Input:") {
			currentInput = strings.TrimPrefix(line, "Action Input:")
			currentInput = strings.TrimSpace(currentInput)

			if currentAction != "" && currentInput != "" {
				toolCalls = append(toolCalls, protocol.ToolCall{
					ID:   fmt.Sprintf("call_%d", callID),
					Type: "function",
					Function: protocol.FunctionCall{
						Name:      currentAction,
						Arguments: json.RawMessage(currentInput),
					},
				})
				callID++
				currentAction = ""
				currentInput = ""
			}
		}
	}

	return toolCalls
}

// MakeTool creates a simple tool definition.
func MakeTool(name, description string, params map[string]interface{}) protocol.Tool {
	return protocol.Tool{
		Name:        name,
		Description: description,
		Parameters:  params,
	}
}

// WeatherTool returns a weather lookup tool definition.
func WeatherTool() protocol.Tool {
	return MakeTool(
		"get_weather",
		"Get the current weather for a location",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "The city and state, e.g. San Francisco, CA",
				},
				"unit": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"celsius", "fahrenheit"},
					"description": "Temperature unit",
				},
			},
			"required": []string{"location"},
		},
	)
}

// CalculatorTool returns a calculator tool definition.
func CalculatorTool() protocol.Tool {
	return MakeTool(
		"calculate",
		"Perform mathematical calculations",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "Mathematical expression to evaluate",
				},
			},
			"required": []string{"expression"},
		},
	)
}

// SearchTool returns a web search tool definition.
func SearchTool() protocol.Tool {
	return MakeTool(
		"web_search",
		"Search the web for information",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results",
					"default":     10,
				},
			},
			"required": []string{"query"},
		},
	)
}

// ToolCallParser extracts tool calls from model responses.
type ToolCallParser interface {
	Parse(ctx context.Context, response *protocol.ChatResponse) ([]ToolCall, error)
}

// ReActToolCallParser parses tool calls from ReAct-style responses.
type ReActToolCallParser struct{}

// Parse extracts tool calls from ReAct-style responses.
func (p *ReActToolCallParser) Parse(ctx context.Context, response *protocol.ChatResponse) ([]ToolCall, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := response.Choices[0].Message.Content
	var toolCalls []ToolCall

	lines := strings.Split(content, "\n")
	var currentAction string
	var currentInput string

	for i, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Action:") {
			currentAction = strings.TrimPrefix(line, "Action:")
			currentAction = strings.TrimSpace(currentAction)
		} else if strings.HasPrefix(line, "Action Input:") {
			currentInput = strings.TrimPrefix(line, "Action Input:")
			currentInput = strings.TrimSpace(currentInput)

			if currentAction != "" && currentInput != "" {
				toolCall := ToolCall{
					ID:   fmt.Sprintf("call_%d", i),
					Type: "function",
					Function: FunctionCall{
						Name:      currentAction,
						Arguments: json.RawMessage(currentInput),
					},
				}
				toolCalls = append(toolCalls, toolCall)

				currentAction = ""
				currentInput = ""
			}
		}
	}

	return toolCalls, nil
}

// JSONToolCallParser parses tool calls from JSON-formatted responses.
type JSONToolCallParser struct{}

// Parse extracts tool calls from JSON-formatted responses.
func (p *JSONToolCallParser) Parse(ctx context.Context, response *protocol.ChatResponse) ([]ToolCall, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := response.Choices[0].Message.Content
	var data map[string]interface{}

	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	toolCallsData, ok := data["tool_calls"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("tool_calls not found in response")
	}

	var toolCalls []ToolCall
	for i, tc := range toolCallsData {
		tcMap, ok := tc.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := tcMap["name"].(string)
		input, _ := tcMap["input"].(map[string]interface{})

		inputBytes, _ := json.Marshal(input)
		toolCalls = append(toolCalls, ToolCall{
			ID:   fmt.Sprintf("call_%d", i),
			Type: "function",
			Function: FunctionCall{
				Name:      name,
				Arguments: inputBytes,
			},
		})
	}

	return toolCalls, nil
}

// ToolCallExtractor extracts tool calls from model responses using various strategies.
type ToolCallExtractor struct {
	parsers []ToolCallParser
}

// NewToolCallExtractor creates a new tool call extractor.
func NewToolCallExtractor() *ToolCallExtractor {
	return &ToolCallExtractor{
		parsers: []ToolCallParser{
			&JSONToolCallParser{},
			&ReActToolCallParser{},
		},
	}
}

// AddParser adds a parser to the extractor.
func (e *ToolCallExtractor) AddParser(parser ToolCallParser) *ToolCallExtractor {
	e.parsers = append(e.parsers, parser)
	return e
}

// Extract tries all parsers to extract tool calls.
func (e *ToolCallExtractor) Extract(ctx context.Context, response *protocol.ChatResponse) ([]ToolCall, error) {
	var lastErr error

	for _, parser := range e.parsers {
		toolCalls, err := parser.Parse(ctx, response)
		if err == nil && len(toolCalls) > 0 {
			return toolCalls, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("no tool calls found in response")
}
