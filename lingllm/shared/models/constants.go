package models

// Vision-capable models across providers.
const (
	// OpenAI vision models
	VisionOpenAIGPT4o        = OpenAIGPT4o
	VisionOpenAIGPT4oMini    = OpenAIGPT4oMini
	VisionOpenAIGPT4Turbo    = OpenAIGPT4Turbo
	VisionOpenAIGPT4Vision   = "gpt-4-vision-preview"
	VisionOpenAIGPT4VisionHD = "gpt-4-turbo-with-vision"

	// Anthropic vision models
	VisionAnthropicClaude3Opus    = AnthropicClaude3Opus
	VisionAnthropicClaude3Sonnet  = AnthropicClaude3Sonnet
	VisionAnthropicClaude35Sonnet = AnthropicClaude35Sonnet

	// Google vision models
	VisionGoogleGemini15Pro   = Gemini15Pro
	VisionGoogleGemini15Flash = Gemini15Flash
	VisionGoogleGemini20Flash = Gemini20Flash

	// Mistral vision models
	VisionMistralPixtralLarge = MistralPixtralLargeLatest
	VisionMistralPixtral12B   = MistralPixtral12BLatest

	// Meta vision models
	VisionMetaLlama32Vision = "llama-3.2-vision"

	// xAI vision models
	VisionXAIGrok2Vision = XAIGrok2Vision
)

// Reasoning-capable models (for complex problem-solving).
const (
	// OpenAI reasoning models
	ReasoningOpenAIO1     = OpenAIO1
	ReasoningOpenAIO1Mini = OpenAIO1Mini
	ReasoningOpenAIO1Pro  = OpenAIO1Pro
	ReasoningOpenAIO3     = OpenAIO3
	ReasoningOpenAIO3Mini = OpenAIO3Mini
	ReasoningOpenAIO3Pro  = OpenAIO3Pro

	// DeepSeek reasoning models
	ReasoningDeepSeekReasoner = DeepSeekReasoner

	// Google reasoning models (Gemini 2.0 Pro Exp)
	ReasoningGoogleGemini20ProExp = Gemini20ProExp
)

// Fast/lightweight models for cost-effective inference.
const (
	// OpenAI lightweight models
	FastOpenAIGPT4oMini  = OpenAIGPT4oMini
	FastOpenAIGPT35Turbo = OpenAIGPT35Turbo
	FastOpenAIGPT41Mini  = OpenAIGPT41Mini
	FastOpenAIGPT41Nano  = OpenAIGPT41Nano

	// Anthropic lightweight models
	FastAnthropicClaude35Haiku = AnthropicClaude35Haiku
	FastAnthropicClaudeHaiku45 = AnthropicClaudeHaiku45

	// Google lightweight models
	FastGoogleGemini25FlashLite = Gemini25FlashLite
	FastGoogleGemini20FlashLite = Gemini20FlashLite
	FastGoogleGemini15Flash8B   = Gemini15Flash8B

	// Mistral lightweight models
	FastMistralSmall       = MistralSmallLatest
	FastMistralMinistral3B = MistralMinistral3BLatest

	// Qwen lightweight models
	FastQwenTurbo = QwenTurbo

	// Meta lightweight models
	FastMetaLlama32_1B = MetaLlama32_1BInstruct
	FastMetaLlama32_3B = MetaLlama32_3BInstruct
)

// High-performance models for complex tasks.
const (
	// OpenAI flagship models
	PremiumOpenAIGPT4o     = OpenAIGPT4o
	PremiumOpenAIGPT4Turbo = OpenAIGPT4Turbo
	PremiumOpenAIGPT5      = OpenAIGPT5
	PremiumOpenAIGPT52Pro  = OpenAIGPT52Pro

	// Anthropic flagship models
	PremiumAnthropicClaudeOpus46   = AnthropicClaudeOpus46
	PremiumAnthropicClaudeSonnet46 = AnthropicClaudeSonnet46
	PremiumAnthropicClaudeOpus41   = AnthropicClaudeOpus41

	// Google flagship models
	PremiumGoogleGemini25Pro    = Gemini25Pro
	PremiumGoogleGemini20ProExp = Gemini20ProExp

	// Mistral flagship models
	PremiumMistralLarge = MistralLargeLatest

	// DeepSeek flagship models
	PremiumDeepSeekV3 = DeepSeekV3

	// Qwen flagship models
	PremiumQwenMax = QwenMax

	// Meta flagship models
	PremiumMetaLlama31_405B = MetaLlama31_405B
)

// Code generation specialized models.
const (
	// OpenAI code models
	CodeOpenAIGPT4o      = OpenAIGPT4o
	CodeOpenAIGPT4Turbo  = OpenAIGPT4Turbo
	CodeOpenAIGPT35Turbo = OpenAIGPT35Turbo

	// Mistral code models
	CodeMistralCodestral     = MistralCodestralLatest
	CodeMistralCodestral2501 = MistralCodestral2501

	// DeepSeek code models
	CodeDeepSeekCoder         = DeepSeekCoder
	CodeDeepSeekCoderInstruct = DeepSeekCoderInstruct

	// Meta code models
	CodeMetaLlama31_70B = MetaLlama31_70B
)

// Embedding models for semantic search and similarity.
const (
	// OpenAI embedding models
	EmbeddingOpenAITextEmbedding3Large = OpenAITextEmbedding3Large
	EmbeddingOpenAITextEmbedding3Small = OpenAITextEmbedding3Small
	EmbeddingOpenAITextEmbeddingAda002 = OpenAITextEmbeddingAda002

	// Mistral embedding models
	EmbeddingMistralEmbed          = MistralEmbed
	EmbeddingMistralCodestralEmbed = MistralCodestralEmbed

	// Cohere embedding models
	EmbeddingCohereEmbedV4             = CohereEmbedV4
	EmbeddingCohereEmbedEnglishV3      = CohereEmbedEnglishV3
	EmbeddingCohereEmbedMultilingualV3 = CohereEmbedMultilingualV3
)

// Multilingual models with strong language support.
const (
	// Google multilingual models
	MultilingualGoogleGemini25Pro = Gemini25Pro
	MultilingualGoogleGemini15Pro = Gemini15Pro

	// Mistral multilingual models
	MultilingualMistralLarge = MistralLargeLatest

	// Qwen multilingual models
	MultilingualQwenMax = QwenMax

	// Meta multilingual models
	MultilingualMetaLlama31_70B = MetaLlama31_70B

	// Cohere multilingual models
	MultilingualCohereCommandRPlus = CohereCommandRPlus082024
)

// Long context window models (100k+ tokens).
const (
	// OpenAI long context models
	LongContextOpenAIGPT4Turbo = OpenAIGPT4Turbo
	LongContextOpenAIGPT4o     = OpenAIGPT4o

	// Anthropic long context models
	LongContextAnthropicClaude35Sonnet = AnthropicClaude35Sonnet
	LongContextAnthropicClaudeOpus41   = AnthropicClaudeOpus41

	// Google long context models
	LongContextGoogleGemini15Pro = Gemini15Pro

	// Zhipu long context models
	LongContextZhipuGLM4Long = ZhipuGLM4Long
)

// Models with function calling support.
const (
	// OpenAI function calling models
	FunctionCallingOpenAIGPT4o      = OpenAIGPT4o
	FunctionCallingOpenAIGPT4Turbo  = OpenAIGPT4Turbo
	FunctionCallingOpenAIGPT35Turbo = OpenAIGPT35Turbo

	// Anthropic function calling models
	FunctionCallingAnthropicClaude35Sonnet = AnthropicClaude35Sonnet
	FunctionCallingAnthropicClaudeOpus41   = AnthropicClaudeOpus41

	// Google function calling models
	FunctionCallingGoogleGemini25Pro = Gemini25Pro
	FunctionCallingGoogleGemini15Pro = Gemini15Pro

	// Mistral function calling models
	FunctionCallingMistralLarge = MistralLargeLatest

	// Qwen function calling models
	FunctionCallingQwenMax = QwenMax
)

// Recommended default models by use case.
const (
	// Default general-purpose model
	DefaultGeneralPurpose = OpenAIGPT4o

	// Default fast model
	DefaultFast = OpenAIGPT4oMini

	// Default reasoning model
	DefaultReasoning = OpenAIO1

	// Default vision model
	DefaultVision = OpenAIGPT4o

	// Default code model
	DefaultCode = OpenAIGPT4o

	// Default embedding model
	DefaultEmbedding = OpenAITextEmbedding3Small
)
