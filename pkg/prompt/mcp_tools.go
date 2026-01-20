package prompt

import (
	"context"
	"encoding/json"

	"github.com/getkin/kin-openapi/openapi3"
)

type Content interface {
	isContent()
}

// CallToolParams represents tool call parameters
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Meta      *struct {
		ProgressToken ProgressToken `json:"progressToken,omitempty"`
	} `json:"_meta,omitempty"`
}

// CallToolRequest represents a tool call request (conforming to MCP specification)
type CallToolRequest struct {
	Request
	Params CallToolParams `json:"params"`
}

// CallToolResult represents tool call result
type CallToolResult struct {
	Result
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// toolHandler defines the function type for handling tool execution
type toolHandler func(ctx context.Context, req *CallToolRequest) (*CallToolResult, error)

// registeredTool combines a Tool with its handler function
type registeredTool struct {
	Tool    *Tool
	Handler toolHandler
}

// Tool represents an MCP tool.
type Tool struct {
	// Tool name
	Name string `json:"name"`

	// Tool description
	Description string `json:"description,omitempty"`

	// Input parameter schema
	InputSchema *openapi3.Schema `json:"inputSchema"`

	// Raw schema (for custom schemas)
	RawInputSchema json.RawMessage `json:"-"`
}

// ToolListFilter defines a function type for filtering tools based on context.
// The filter receives the request context and all registered tools, and returns
// a filtered list of tools that should be visible to the client.
type ToolListFilter func(ctx context.Context, tools []*Tool) []*Tool

// ListToolsResult represents the result of listing tools
type ListToolsResult struct {
	PaginatedResult
	Tools []Tool `json:"tools"`
}
