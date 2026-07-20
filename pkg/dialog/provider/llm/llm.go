// Package llm defines the LLM (Large Language Model) provider
// interface used by cascaded dialog engines. Realtime multimodal
// providers (Qwen-Omni, GPT-4o realtime) implement
// pkg/dialog/provider/multimodal.Realtime instead.
package llm

import (
	"context"
)

// Provider is the factory abstraction for one LLM vendor.
type Provider interface {
	Name() string

	// Chat opens a streaming chat session. The engine pushes
	// messages and consumes streamed deltas (text + optional tool
	// calls). Many vendors are stateless per-turn — implementations
	// MAY internally re-send the full history on each Chat call.
	Chat(ctx context.Context, req ChatRequest) (Stream, error)
}

// ChatRequest is the input to one chat turn.
type ChatRequest struct {
	// Model is a vendor-specific model identifier (e.g. "gpt-4o",
	// "qwen-max", "ERNIE-Bot-4"). Empty = vendor default.
	Model string

	// Messages is the conversation so far, oldest first. Engines
	// build this from persisted turns + system prompt.
	Messages []Message

	// Tools advertises function-call tools to the model. Vendors
	// that don't support function calling ignore this field.
	Tools []ToolSchema

	// Temperature controls sampling randomness; 0 = vendor default.
	Temperature float64

	// MaxTokens caps output length; 0 = vendor default.
	MaxTokens int
}

// Role is the speaker of a message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is one entry in the conversation history.
type Message struct {
	Role    Role
	Content string

	// ToolCallID is set on RoleTool messages (the response to a
	// previous tool call from the assistant).
	ToolCallID string

	// ToolCalls is set on RoleAssistant messages when the model
	// requested one or more tool invocations.
	ToolCalls []ToolCall
}

// ToolSchema is the JSON-Schema-shaped declaration sent to the model.
type ToolSchema struct {
	Name        string
	Description string
	// Parameters is a JSON Schema document (vendor-specific
	// dialect — most accept OpenAI-style). Stored as raw bytes so
	// this package stays JSON-library-free.
	Parameters []byte
}

// ToolCall is one function-call invocation requested by the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments []byte // JSON object, vendor-specified
}

// Stream is one in-progress chat completion.
type Stream interface {
	// Delta returns a receive-only channel of streaming events.
	// Closed when the model finishes (either normal stop or error).
	Delta() <-chan Delta

	// Close cancels in-flight generation and releases resources.
	// Idempotent.
	Close() error
}

// Delta is one streaming event from the model.
type Delta struct {
	// Text is the next text fragment (may be empty when the event
	// is only a tool call).
	Text string

	// ToolCall is non-nil when the model requested a tool call.
	// The engine should dispatch and feed the result back via a
	// follow-up Chat request with the tool response.
	ToolCall *ToolCall

	// Done indicates this is the final event. After Done=true the
	// Delta channel will be closed.
	Done bool

	// Err is set on the final event when the stream ended in error.
	Err error
}
