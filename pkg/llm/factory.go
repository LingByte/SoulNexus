package llm

import (
	"context"
	"strings"
)

const (
	ProviderOpenAI    = "openai"
	ProviderOllama    = "ollama"
	ProviderAlibaba   = "alibaba"
	ProviderAnthropic = "anthropic"
	ProviderLMStudio  = "lmstudio"
	ProviderCoze      = "coze"
)

func normalizeProvider(provider string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	switch p {
	case "", ProviderOpenAI:
		return ProviderOpenAI
	case ProviderOllama:
		return ProviderOllama
	case ProviderAlibaba:
		return ProviderAlibaba
	case ProviderAnthropic:
		return ProviderAnthropic
	case ProviderLMStudio:
		return ProviderLMStudio
	case ProviderCoze:
		return ProviderCoze
	default:
		return ProviderOpenAI
	}
}

// NewProviderHandler creates an LLM handler by provider type.
// Note: in Ling, non-OpenAI providers currently use OpenAI-compatible chat API shape.
func NewProviderHandler(ctx context.Context, provider string, llmOptions *LLMOptions) (LLMHandler, error) {
	if llmOptions == nil {
		llmOptions = &LLMOptions{}
	}
	selected := normalizeProvider(provider)
	if strings.TrimSpace(llmOptions.Provider) != "" {
		selected = normalizeProvider(llmOptions.Provider)
	}

	opts := *llmOptions

	switch selected {
	case ProviderOllama:
		return NewOllamaHandler(ctx, &opts)
	case ProviderAlibaba:
		return NewAlibabaHandler(ctx, &opts)
	case ProviderAnthropic:
		return NewAnthropicHandler(ctx, &opts)
	case ProviderLMStudio:
		return NewLMStudioHandler(ctx, &opts)
	case ProviderCoze:
		return NewCozeHandler(ctx, &opts)
	default:
		return newOpenAICompatibleHandler(ctx, &opts, LLM_OPENAI)
	}
}

// NewLLMProvider provides a SoulNexus-like factory signature for Ling.
func NewLLMProvider(ctx context.Context, provider, apiKey, apiURL, systemPrompt string) (LLMHandler, error) {
	return NewProviderHandler(ctx, provider, &LLMOptions{
		Provider:     provider,
		ApiKey:       apiKey,
		BaseURL:      apiURL,
		SystemPrompt: systemPrompt,
	})
}

