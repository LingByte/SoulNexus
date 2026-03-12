package llm

import (
	"context"
	"encoding/json"
	"strings"
)

// ProviderType LLM 提供者类型
type ProviderType string

const (
	ProviderTypeOpenAI ProviderType = "openai" // OpenAI 兼容的 API
	ProviderTypeCoze   ProviderType = "coze"   // Coze API
	ProviderTypeOllama ProviderType = "ollama" // Ollama API
)

// NewLLMProvider 根据配置创建 LLM 提供者
func NewLLMProvider(ctx context.Context, provider, apiKey, apiUrl, systemPrompt string) (LLMProvider, error) {
	providerType := strings.ToLower(strings.TrimSpace(provider))
	if providerType == "" {
		providerType = string(ProviderTypeOpenAI)
	}
	switch providerType {
	case string(ProviderTypeCoze):
		botID := ""
		userID := ""
		baseURL := ""
		if apiUrl != "" {
			var config CozeConfig
			if err := json.Unmarshal([]byte(apiUrl), &config); err == nil {
				botID = config.BotID
				userID = config.UserID
				baseURL = config.BaseURL
			} else {
				botID = apiUrl
			}
		}
		if baseURL != "" {
			return NewCozeProvider(ctx, apiKey, botID, userID, systemPrompt, baseURL)
		}
		return NewCozeProvider(ctx, apiKey, botID, userID, systemPrompt)
	case string(ProviderTypeOllama):
		baseURL := apiUrl
		apiKey := apiKey
		if apiKey == "" {
			apiKey = "ollama"
		}
		return NewOllamaProvider(ctx, apiKey, baseURL, systemPrompt), nil
	default:
		// Ensure we have a valid base URL, default to OpenAI's API if not provided
		baseURL := apiUrl
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return NewOpenAIProvider(ctx, apiKey, baseURL, systemPrompt), nil
	}
}
