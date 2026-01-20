package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/internal/models"
)

// ProviderType LLM 提供者类型
type ProviderType string

const (
	ProviderTypeOpenAI ProviderType = "openai" // OpenAI 兼容的 API
	ProviderTypeCoze   ProviderType = "coze"   // Coze API
	ProviderTypeOllama ProviderType = "ollama" // Ollama API
)

// NewLLMProvider 根据配置创建 LLM 提供者
// 这是统一的工厂函数，根据 credential 中的 LLMProvider 字段选择提供者
func NewLLMProvider(ctx context.Context, credential *models.UserCredential, systemPrompt string) (LLMProvider, error) {
	providerType := strings.ToLower(strings.TrimSpace(credential.LLMProvider))

	// 如果未指定提供者，默认使用 OpenAI
	if providerType == "" {
		providerType = string(ProviderTypeOpenAI)
	}

	switch providerType {
	case string(ProviderTypeCoze):
		// Coze API
		// Bot ID 可以从 LLMApiURL 中获取，或者从配置中获取
		// 格式：LLMApiURL 可以是 "bot_id" 或者 JSON 格式的配置
		botID := ""
		userID := ""
		baseURL := ""

		// 尝试从 LLMApiURL 解析配置
		if credential.LLMApiURL != "" {
			// 检查是否是 JSON 格式
			var config CozeConfig
			if err := json.Unmarshal([]byte(credential.LLMApiURL), &config); err == nil {
				// 是 JSON 格式
				botID = config.BotID
				userID = config.UserID
				baseURL = config.BaseURL
			} else {
				// 不是 JSON，直接作为 Bot ID
				botID = credential.LLMApiURL
			}
		}

		// 如果 Bot ID 仍然为空，尝试从其他配置获取
		if botID == "" {
			// 可以尝试从 LLMConfig 或其他字段获取
			// 这里暂时返回错误，要求必须提供 Bot ID
			return nil, fmt.Errorf("Coze provider requires botID, please set it in LLMApiURL field (format: 'bot_id' or JSON: '{\"botId\":\"...\",\"userId\":\"...\"}')")
		}

		// 如果 userID 为空，使用默认值或从 credential 获取
		if userID == "" {
			// 可以尝试从 UserID 或其他地方获取
			userID = fmt.Sprintf("user_%d", credential.UserID)
		}

		// 创建 Coze 提供者
		if baseURL != "" {
			return NewCozeProvider(ctx, credential.LLMApiKey, botID, userID, systemPrompt, baseURL)
		}
		return NewCozeProvider(ctx, credential.LLMApiKey, botID, userID, systemPrompt)

	case string(ProviderTypeOllama):
		// Ollama API
		// baseURL 可以从 LLMApiURL 中获取，默认为 http://localhost:11434/v1
		baseURL := credential.LLMApiURL
		// Ollama 通常不需要 API Key，但为了兼容性可以传入空字符串
		apiKey := credential.LLMApiKey
		if apiKey == "" {
			apiKey = "ollama" // 占位符，Ollama 不需要真实的 API Key
		}
		return NewOllamaProvider(ctx, apiKey, baseURL, systemPrompt), nil

	default:
		// 所有其他提供者（包括 openai, zhipu, deepseek, qwen 等）都使用 OpenAI 兼容的方式处理
		// Ensure we have a valid base URL, default to OpenAI's API if not provided
		baseURL := credential.LLMApiURL
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return NewOpenAIProvider(ctx, credential.LLMApiKey, baseURL, systemPrompt), nil
	}
}

// NewLLMProviderFromConfig 从配置创建 LLM 提供者（用于测试或直接配置）
func NewLLMProviderFromConfig(ctx context.Context, providerType string, apiKey, baseURL, systemPrompt string, extraConfig map[string]string) (LLMProvider, error) {
	providerType = strings.ToLower(strings.TrimSpace(providerType))

	switch providerType {
	case string(ProviderTypeCoze):
		botID := ""
		userID := ""
		cozeBaseURL := ""

		if extraConfig != nil {
			botID = extraConfig["botId"]
			userID = extraConfig["userId"]
			cozeBaseURL = extraConfig["baseUrl"]
		}

		// 如果 baseURL 参数不为空，且不是 JSON，则作为 Bot ID
		if botID == "" && baseURL != "" {
			botID = baseURL
		}

		if botID == "" {
			return nil, fmt.Errorf("Coze provider requires botID")
		}

		if userID == "" {
			userID = "default_user"
		}

		if cozeBaseURL != "" {
			return NewCozeProvider(ctx, apiKey, botID, userID, systemPrompt, cozeBaseURL)
		}
		return NewCozeProvider(ctx, apiKey, botID, userID, systemPrompt)

	case string(ProviderTypeOllama):
		// Ollama API
		// baseURL 参数如果为空，使用默认值 http://localhost:11434/v1
		// Ollama 通常不需要 API Key，但为了兼容性可以传入空字符串
		if apiKey == "" {
			apiKey = "ollama" // 占位符，Ollama 不需要真实的 API Key
		}
		return NewOllamaProvider(ctx, apiKey, baseURL, systemPrompt), nil

	default:
		// 所有其他提供者（包括 openai, zhipu, deepseek, qwen 等）都使用 OpenAI 兼容的方式处理
		// Ensure we have a valid base URL, default to OpenAI's API if not provided
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		return NewOpenAIProvider(ctx, apiKey, baseURL, systemPrompt), nil
	}
}
