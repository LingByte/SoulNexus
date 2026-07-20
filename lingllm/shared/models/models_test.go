package models

import (
	"testing"
)

func TestOpenAIConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"GPT4o", OpenAIGPT4o},
		{"GPT4oMini", OpenAIGPT4oMini},
		{"GPT4", OpenAIGPT4},
		{"GPT35Turbo", OpenAIGPT35Turbo},
		{"O1", OpenAIO1},
		{"O1Mini", OpenAIO1Mini},
		{"TextEmbedding3Large", OpenAITextEmbedding3Large},
		{"DALLE3", OpenAIDALLE3},
		{"Whisper1", OpenAIWhisper1},
		{"TTS1", OpenAITTS1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestAnthropicConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"Claude3Opus", AnthropicClaude3Opus},
		{"Claude3Sonnet", AnthropicClaude3Sonnet},
		{"Claude35Sonnet", AnthropicClaude35Sonnet},
		{"Claude3Haiku", AnthropicClaude3Haiku},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestGoogleConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"Gemini25Flash", Gemini25Flash},
		{"Gemini25Pro", Gemini25Pro},
		{"Gemini15Flash", Gemini15Flash},
		{"Gemini15Pro", Gemini15Pro},
		{"Gemini20Flash", Gemini20Flash},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestMistralConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"MistralLarge", MistralLarge},
		{"MistralMedium", MistralMedium},
		{"MistralSmall", MistralSmall},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestQwenConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"QwenMax", QwenMax},
		{"QwenPlus", QwenPlus},
		{"QwenTurbo", QwenTurbo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestOllamaConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"OllamaLlama2", OllamaLlama2},
		{"OllamaLlama2_7B", OllamaLlama2_7B},
		{"OllamaLlama2_13B", OllamaLlama2_13B},
		{"OllamaLlama3", OllamaLlama3},
		{"OllamaLlama31", OllamaLlama31},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestCohereConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"CohereCommandA032025", CohereCommandA032025},
		{"CohereCommandLight32024", CohereCommandLight32024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestMetaConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"MetaLlama3_8B", MetaLlama3_8B},
		{"MetaLlama31_8B", MetaLlama31_8B},
		{"MetaLlama2_7B", MetaLlama2_7B},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestDeepseekConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"DeepSeekChat", DeepSeekChat},
		{"DeepSeekReasoner", DeepSeekReasoner},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestXAIConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"XAIGrok4", XAIGrok4},
		{"XAIGrok3", XAIGrok3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestZhipuConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"ZhipuGLM4", ZhipuGLM4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}

func TestMoonshotConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"MoonshotMoonshotV1_8K", MoonshotMoonshotV1_8K},
		{"MoonshotKimiK2", MoonshotKimiK2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Error("constant should not be empty")
			}
		})
	}
}
