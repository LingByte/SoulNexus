package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/LingByte/lingllm/metrics"
)

// MessageRole defines the semantic role of a chat message.
type MessageRole string

const (
	RoleSystem    MessageRole = "system"
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleTool      MessageRole = "tool"
)

// Message represents a single turn in a chat conversation.
type Message struct {
	Role       MessageRole `json:"role"`
	Content    string      `json:"content"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
}

// ToolCall represents a tool invocation request.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall represents a function call with name and arguments.
type FunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Tool defines a tool/function that the model can call.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ToolChoice controls whether the model must, may, or must not call tools.
type ToolChoice string

const (
	ToolChoiceAuto     ToolChoice = "auto"
	ToolChoiceRequired ToolChoice = "required"
	ToolChoiceNone     ToolChoice = "none"
)

// ChatRequest captures a provider-agnostic chat generation request.
type ChatRequest struct {
	// Model is the target model identifier (e.g. "gpt-4o", "claude-3-opus").
	Model string `json:"model"`
	// Messages is the ordered conversation history.
	Messages []Message `json:"messages"`
	// MaxTokens limits generated tokens (provider specific defaults apply when zero).
	MaxTokens int `json:"max_tokens,omitempty"`
	// Temperature controls randomness (0–2 typical for OpenAI-compatible models).
	Temperature float32 `json:"temperature,omitempty"`
	// TopP enables nucleus sampling (provider specific).
	TopP float32 `json:"top_p,omitempty"`
	// Stop provides stop sequences.
	Stop []string `json:"stop,omitempty"`
	// Tools are functions the model can call.
	Tools []Tool `json:"tools,omitempty"`
	// ToolChoice controls tool calling behavior.
	ToolChoice ToolChoice `json:"tool_choice,omitempty"`
	// Metadata is an optional provider-specific bag for future extensions.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// TokenUsage reports token accounting from the provider.
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Choice represents a single candidate completion.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// ChatResponse is a normalized chat completion result.
type ChatResponse struct {
	ID        string              `json:"id"`
	Model     string              `json:"model"`
	CreatedAt time.Time           `json:"created_at"`
	Choices   []Choice            `json:"choices"`
	Usage     TokenUsage          `json:"usage"`
	Metrics   metrics.CallMetrics `json:"metrics,omitempty"`
}

// ChatModel defines the minimal capability surface for text-only chat models.
type ChatModel interface {
	Name() string
	// Chat return common chat response
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	// StreamChat returns a streaming handle that yields incremental deltas.
	StreamChat(ctx context.Context, req ChatRequest) (ChatStream, error)
}

// Note: ChatModel implementations should accept ChatRequest by value for consistency with the interface.
// If your implementation needs a pointer, dereference it at the call site.

// ChatStreamChunk represents an incremental delta from a streaming chat call.
type ChatStreamChunk struct {
	Index        int         `json:"index"`
	Role         MessageRole `json:"role"`
	Delta        string      `json:"delta"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

// ChatStream provides pull-based access to streaming deltas.
// Recv returns io.EOF when the stream ends.
type ChatStream interface {
	Recv() (*ChatStreamChunk, error)
	Close() error
	Metrics() metrics.CallMetrics
}

// StreamReader is an alias for ChatStream used by stream utilities and chains.
type StreamReader = ChatStream

// ErrInvalidRequest is returned when validation fails.
var ErrInvalidRequest = errors.New("invalid chat request")

// Validate ensures a ChatRequest contains minimally required fields.
func (r *ChatRequest) Validate() error {
	if r == nil {
		return fmt.Errorf("%w: request is nil", ErrInvalidRequest)
	}
	if r.Model == "" {
		return fmt.Errorf("%w: model is required", ErrInvalidRequest)
	}
	if len(r.Messages) == 0 {
		return fmt.Errorf("%w: at least one message is required", ErrInvalidRequest)
	}
	for i, m := range r.Messages {
		if m.Role == "" {
			return fmt.Errorf("%w: message %d missing role", ErrInvalidRequest, i)
		}
		// Assistant tool-call turns may have empty content; tool results carry content separately.
		if m.Role == RoleAssistant && len(m.ToolCalls) > 0 {
			continue
		}
		if m.Content == "" {
			return fmt.Errorf("%w: message %d missing content", ErrInvalidRequest, i)
		}
	}
	return nil
}

// Helper constructors for common message types

// UserMessage creates a user message.
func UserMessage(content string) Message {
	return Message{Role: RoleUser, Content: content}
}

// SystemMessage creates a system message.
func SystemMessage(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

// AssistantMessage creates an assistant message.
func AssistantMessage(content string) Message {
	return Message{Role: RoleAssistant, Content: content}
}

// ToolMessage creates a tool message.
func ToolMessage(content string, toolCallID string) Message {
	return Message{Role: RoleTool, Content: content, ToolCallID: toolCallID}
}

// NewChatRequest creates a ChatRequest with the given model and messages.
func NewChatRequest(model string, messages ...Message) *ChatRequest {
	return &ChatRequest{
		Model:    model,
		Messages: messages,
	}
}

// WithMaxTokens sets the max tokens for the request.
func (r *ChatRequest) WithMaxTokens(maxTokens int) *ChatRequest {
	r.MaxTokens = maxTokens
	return r
}

// WithTemperature sets the temperature for the request.
func (r *ChatRequest) WithTemperature(temp float32) *ChatRequest {
	r.Temperature = temp
	return r
}

// WithTopP sets the top_p for the request.
func (r *ChatRequest) WithTopP(topP float32) *ChatRequest {
	r.TopP = topP
	return r
}

// WithStop sets the stop sequences for the request.
func (r *ChatRequest) WithStop(stop ...string) *ChatRequest {
	r.Stop = stop
	return r
}

// WithMetadata sets metadata for the request.
func (r *ChatRequest) WithMetadata(key, value string) *ChatRequest {
	if r.Metadata == nil {
		r.Metadata = make(map[string]string)
	}
	r.Metadata[key] = value
	return r
}

// FirstMessage returns the first choice's message if available.
func (r *ChatResponse) FirstMessage() *Message {
	if len(r.Choices) == 0 {
		return nil
	}
	return &r.Choices[0].Message
}

// FirstContent returns the first choice's message content if available.
func (r *ChatResponse) FirstContent() string {
	if msg := r.FirstMessage(); msg != nil {
		return msg.Content
	}
	return ""
}
