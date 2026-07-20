package providers

import (
	"context"
	"fmt"
	"strings"

	"github.com/LingByte/lingllm/protocol"
	_ "github.com/LingByte/lingllm/protocol/anthropic"
	_ "github.com/LingByte/lingllm/protocol/ollama"
	_ "github.com/LingByte/lingllm/protocol/openai"
)

// ChatLLM is a multi-turn lingllm protocol session for voice dialog paths.
type ChatLLM interface {
	Query(text, model string) (string, error)
	QueryWithOptions(text string, options LLMQueryOptions) (string, error)
	QueryStream(text string, options LLMQueryOptions, callback func(segment string, isComplete bool) error) (string, error)
	RegisterFunctionTool(name, description string, parameters interface{}, callback LLMFunctionToolCallback)
	RegisterFunctionToolDefinition(def *LLMFunctionToolDefinition)
	GetFunctionTools() []interface{}
	ListFunctionTools() []string
	GetLastUsage() (LLMUsage, bool)
	ResetMessages()
	// SeedMessages replaces chat history (excluding system prompt) for text/IM resume.
	SeedMessages(msgs []LLMMessage)
	SetSystemPrompt(systemPrompt string)
	GetMessages() []LLMMessage
	// LastToolTrace returns tool calls from the most recent QueryWithOptions turn.
	LastToolTrace() []LLMToolCall
	Interrupt()
	Hangup()
}

// LLMQueryOptions controls a single LLM turn.
type LLMQueryOptions struct {
	Model               string
	MaxTokens           *int
	MaxCompletionTokens *int
	Temperature         *float32
	TopP                *float32
	Stop                []string
	Stream              bool
	EnableJSONOutput    bool
	// KnowledgeCallID enables server-side search_knowledge_base before the model replies.
	KnowledgeCallID string
	// InvocationCallID attributes the AI invocation log without triggering KB enrich.
	// Prefer this for voice-session / debug / embed (cascaded already injects KB itself).
	InvocationCallID string
	// InvocationSource overrides default source (voice / api), e.g. assistant_debug_text.
	InvocationSource string
	// DisableTools omits function tools from this turn (text debug / stream fallback).
	DisableTools bool
	// MaxToolRounds caps tool-chain iterations (0 = default 10 for voice, text dialogs use 12 via ctx).
	MaxToolRounds int
	// Context cancels the LLM/tool chain (voice barge-in / turn timeout).
	// When nil, Query* still applies a bounded timeout so turns cannot hang forever.
	Context context.Context
}

// LLMUsage is token accounting from the last call.
type LLMUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// LLMMessage is a simplified chat history entry.
type LLMMessage struct {
	Role      string
	Content   string
	ToolCalls []LLMToolCall
}

// LLMToolCall describes one tool invocation in history.
type LLMToolCall struct {
	ID       string
	Type     string
	Function LLMFunctionCall
}

// LLMFunctionCall holds function name and JSON arguments.
type LLMFunctionCall struct {
	Name      string
	Arguments string
}

// NewChatLLM builds a lingllm protocol session for openai, ollama, or anthropic.
func NewChatLLM(ctx context.Context, provider, apiKey, baseURL, systemPrompt string) (ChatLLM, error) {
	model, err := newProtocolClient(provider, apiKey, baseURL)
	if err != nil {
		return nil, err
	}
	return newChatLLMSession(ctx, provider, model, systemPrompt), nil
}

func newProtocolClient(provider, apiKey, baseURL string) (protocol.ChatModel, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "" {
		p = "openai"
	}
	switch p {
	case "openai":
		if strings.TrimSpace(baseURL) == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return protocol.NewClient(protocol.ClientConfig{
			Provider: protocol.ProviderOpenAI,
			APIKey:   apiKey,
			BaseURL:  baseURL,
		})
	case "ollama":
		if strings.TrimSpace(baseURL) == "" {
			baseURL = "http://localhost:11434"
		}
		baseURL = strings.TrimSuffix(baseURL, "/")
		if strings.HasSuffix(baseURL, "/v1") {
			baseURL = strings.TrimSuffix(baseURL, "/v1")
		}
		if strings.TrimSpace(apiKey) == "" {
			apiKey = "ollama"
		}
		return protocol.NewClient(protocol.ClientConfig{
			Provider: protocol.ProviderOllama,
			APIKey:   apiKey,
			BaseURL:  baseURL,
		})
	case "anthropic":
		return protocol.NewClient(protocol.ClientConfig{
			Provider: protocol.ProviderAnthropic,
			APIKey:   apiKey,
			BaseURL:  baseURL,
		})
	default:
		return nil, fmt.Errorf("unsupported LLM provider %q (supported: openai, ollama, anthropic)", provider)
	}
}
