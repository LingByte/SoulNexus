package models

import (
	"testing"
)

func TestVisionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"VisionOpenAIGPT4o", VisionOpenAIGPT4o},
		{"VisionOpenAIGPT4oMini", VisionOpenAIGPT4oMini},
		{"VisionAnthropicClaude3Opus", VisionAnthropicClaude3Opus},
		{"VisionGoogleGemini15Pro", VisionGoogleGemini15Pro},
		{"VisionMistralPixtralLarge", VisionMistralPixtralLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestReasoningConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"ReasoningOpenAIO1", ReasoningOpenAIO1},
		{"ReasoningOpenAIO3", ReasoningOpenAIO3},
		{"ReasoningDeepSeekReasoner", ReasoningDeepSeekReasoner},
		{"ReasoningGoogleGemini20ProExp", ReasoningGoogleGemini20ProExp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestFastConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"FastOpenAIGPT4oMini", FastOpenAIGPT4oMini},
		{"FastOpenAIGPT35Turbo", FastOpenAIGPT35Turbo},
		{"FastAnthropicClaude35Haiku", FastAnthropicClaude35Haiku},
		{"FastGoogleGemini25FlashLite", FastGoogleGemini25FlashLite},
		{"FastMistralSmall", FastMistralSmall},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestPremiumConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"PremiumOpenAIGPT4o", PremiumOpenAIGPT4o},
		{"PremiumOpenAIGPT5", PremiumOpenAIGPT5},
		{"PremiumAnthropicClaudeOpus46", PremiumAnthropicClaudeOpus46},
		{"PremiumGoogleGemini25Pro", PremiumGoogleGemini25Pro},
		{"PremiumMistralLarge", PremiumMistralLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestCodeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"CodeOpenAIGPT4o", CodeOpenAIGPT4o},
		{"CodeMistralCodestral", CodeMistralCodestral},
		{"CodeDeepSeekCoder", CodeDeepSeekCoder},
		{"CodeMetaLlama31_70B", CodeMetaLlama31_70B},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestEmbeddingConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"EmbeddingOpenAITextEmbedding3Large", EmbeddingOpenAITextEmbedding3Large},
		{"EmbeddingOpenAITextEmbedding3Small", EmbeddingOpenAITextEmbedding3Small},
		{"EmbeddingMistralEmbed", EmbeddingMistralEmbed},
		{"EmbeddingCohereEmbedV4", EmbeddingCohereEmbedV4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestMultilingualConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"MultilingualGoogleGemini25Pro", MultilingualGoogleGemini25Pro},
		{"MultilingualMistralLarge", MultilingualMistralLarge},
		{"MultilingualQwenMax", MultilingualQwenMax},
		{"MultilingualMetaLlama31_70B", MultilingualMetaLlama31_70B},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestLongContextConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"LongContextOpenAIGPT4Turbo", LongContextOpenAIGPT4Turbo},
		{"LongContextOpenAIGPT4o", LongContextOpenAIGPT4o},
		{"LongContextAnthropicClaude35Sonnet", LongContextAnthropicClaude35Sonnet},
		{"LongContextGoogleGemini15Pro", LongContextGoogleGemini15Pro},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestFunctionCallingConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"FunctionCallingOpenAIGPT4o", FunctionCallingOpenAIGPT4o},
		{"FunctionCallingAnthropicClaude35Sonnet", FunctionCallingAnthropicClaude35Sonnet},
		{"FunctionCallingGoogleGemini25Pro", FunctionCallingGoogleGemini25Pro},
		{"FunctionCallingMistralLarge", FunctionCallingMistralLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestDefaultConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"DefaultGeneralPurpose", DefaultGeneralPurpose},
		{"DefaultFast", DefaultFast},
		{"DefaultReasoning", DefaultReasoning},
		{"DefaultVision", DefaultVision},
		{"DefaultCode", DefaultCode},
		{"DefaultEmbedding", DefaultEmbedding},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}
