package models

// OpenAI-compatible gateway aliases (aggregators, proxies, private deployments).
const (
	// OpenAI family
	GatewayGPT4o      = OpenAIGPT4o
	GatewayGPT4oMini  = OpenAIGPT4oMini
	GatewayGPT41      = OpenAIGPT41
	GatewayGPT41Mini  = OpenAIGPT41Mini
	GatewayGPT41Nano  = OpenAIGPT41Nano
	GatewayGPT4Turbo  = OpenAIGPT4Turbo
	GatewayGPT4       = OpenAIGPT4
	GatewayGPT35Turbo = OpenAIGPT35Turbo
	GatewayGPT5       = OpenAIGPT5
	GatewayGPT52      = OpenAIGPT52
	GatewayO1         = OpenAIO1
	GatewayO1Mini     = OpenAIO1Mini
	GatewayO3         = OpenAIO3
	GatewayO3Mini     = OpenAIO3Mini
	GatewayO4Mini     = OpenAIO4Mini

	// Anthropic family
	GatewayClaudeOpus46   = AnthropicClaudeOpus46
	GatewayClaudeSonnet46 = AnthropicClaudeSonnet46
	GatewayClaudeHaiku45  = AnthropicClaudeHaiku45
	GatewayClaudeOpus41   = AnthropicClaudeOpus41
	GatewayClaudeSonnet4  = AnthropicClaudeSonnet4
	GatewayClaude37Sonnet = AnthropicClaude37Sonnet
	GatewayClaude35Sonnet = AnthropicClaude35Sonnet
	GatewayClaude35Haiku  = AnthropicClaude35Haiku
	GatewayClaude3Opus    = AnthropicClaude3Opus
	GatewayClaude3Sonnet  = AnthropicClaude3Sonnet
	GatewayClaude3Haiku   = AnthropicClaude3Haiku

	// DeepSeek family
	GatewayDeepSeekChat     = DeepSeekChat
	GatewayDeepSeekReasoner = DeepSeekReasoner
	GatewayDeepSeekCoder    = DeepSeekCoder
	GatewayDeepSeekV3       = DeepSeekV3

	// Google Gemini family
	GatewayGemini25Pro       = Gemini25Pro
	GatewayGemini25Flash     = Gemini25Flash
	GatewayGemini25FlashLite = Gemini25FlashLite
	GatewayGemini20Flash     = Gemini20Flash
	GatewayGemini20FlashLite = Gemini20FlashLite
	GatewayGemini15Pro       = Gemini15Pro
	GatewayGemini15Flash     = Gemini15Flash
	GatewayGemini15Flash8B   = Gemini15Flash8B

	// Mistral family
	GatewayMistralLarge     = MistralLargeLatest
	GatewayMistralSmall     = MistralSmallLatest
	GatewayMistralMedium    = MistralMediumLatest
	GatewayMistralCodestral = MistralCodestralLatest
	GatewayMistralNemo      = MistralOpenNemo

	// Qwen family
	GatewayQwenMax   = QwenMax
	GatewayQwenPlus  = QwenPlus
	GatewayQwenTurbo = QwenTurbo
	GatewayQwen3Max  = Qwen3Max

	// xAI Grok family
	GatewayGrok3     = XAIGrok3
	GatewayGrok3Mini = XAIGrok3Mini
	GatewayGrok4     = XAIGrok4

	// Zhipu / Moonshot
	GatewayGLM4Plus  = ZhipuGLM4Plus
	GatewayGLM4Flash = ZhipuGLM4Flash
	GatewayKimiK2    = MoonshotKimiK2
)
