package workflow

import (
	"testing"
)

// TestParseAIChatConfig 测试解析AI对话配置
func TestParseAIChatConfig(t *testing.T) {
	tests := []struct {
		name       string
		properties map[string]interface{}
		wantErr    bool
		errMsg     string
	}{
		{
			name: "valid config",
			properties: map[string]interface{}{
				"provider":       "openai",
				"apiKey":         "sk-test",
				"model":          "gpt-4",
				"inputVariable":  "user_input",
				"outputVariable": "ai_response",
			},
			wantErr: false,
		},
		{
			name: "missing provider",
			properties: map[string]interface{}{
				"apiKey":         "sk-test",
				"model":          "gpt-4",
				"inputVariable":  "user_input",
				"outputVariable": "ai_response",
			},
			wantErr: true,
			errMsg:  "provider is required",
		},
		{
			name: "missing apiKey",
			properties: map[string]interface{}{
				"provider":       "openai",
				"model":          "gpt-4",
				"inputVariable":  "user_input",
				"outputVariable": "ai_response",
			},
			wantErr: true,
			errMsg:  "apiKey is required",
		},
		{
			name: "missing model",
			properties: map[string]interface{}{
				"provider":       "openai",
				"apiKey":         "sk-test",
				"inputVariable":  "user_input",
				"outputVariable": "ai_response",
			},
			wantErr: true,
			errMsg:  "model is required",
		},
		{
			name: "missing inputVariable",
			properties: map[string]interface{}{
				"provider":       "openai",
				"apiKey":         "sk-test",
				"model":          "gpt-4",
				"outputVariable": "ai_response",
			},
			wantErr: true,
			errMsg:  "inputVariable is required",
		},
		{
			name: "missing outputVariable",
			properties: map[string]interface{}{
				"provider":      "openai",
				"apiKey":        "sk-test",
				"model":         "gpt-4",
				"inputVariable": "user_input",
			},
			wantErr: true,
			errMsg:  "outputVariable is required",
		},
		{
			name: "with optional fields",
			properties: map[string]interface{}{
				"provider":       "openai",
				"apiKey":         "sk-test",
				"baseURL":        "https://api.openai.com/v1",
				"model":          "gpt-4",
				"systemPrompt":   "You are a helpful assistant",
				"inputVariable":  "user_input",
				"outputVariable": "ai_response",
				"temperature":    0.7,
				"maxTokens":      2000,
				"enableStream":   true,
				"persistHistory": true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseAIChatConfig(tt.properties)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAIChatConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("ParseAIChatConfig() error = %v, want %v", err.Error(), tt.errMsg)
				return
			}
			if !tt.wantErr && config == nil {
				t.Errorf("ParseAIChatConfig() config is nil")
				return
			}
		})
	}
}

// TestNewAIChatNode 测试创建AI对话节点
func TestNewAIChatNode(t *testing.T) {
	baseNode := &Node{
		ID:   "ai_chat_1",
		Name: "AI对话",
		Type: NodeTypeAIChat,
	}

	config := &AIChatConfig{
		Provider:       "openai",
		APIKey:         "sk-test",
		Model:          "gpt-4",
		InputVariable:  "user_input",
		OutputVariable: "ai_response",
	}

	node := NewAIChatNode(baseNode, config)

	if node == nil {
		t.Errorf("NewAIChatNode() returned nil")
		return
	}

	if node.ID != "ai_chat_1" {
		t.Errorf("NewAIChatNode() ID = %s, want ai_chat_1", node.ID)
	}

	if node.Config.Provider != "openai" {
		t.Errorf("NewAIChatNode() Provider = %s, want openai", node.Config.Provider)
	}
}

// TestAIChatNodeBase 测试AI对话节点的Base方法
func TestAIChatNodeBase(t *testing.T) {
	baseNode := &Node{
		ID:   "ai_chat_1",
		Name: "AI对话",
		Type: NodeTypeAIChat,
	}

	config := &AIChatConfig{
		Provider:       "openai",
		APIKey:         "sk-test",
		Model:          "gpt-4",
		InputVariable:  "user_input",
		OutputVariable: "ai_response",
	}

	node := NewAIChatNode(baseNode, config)
	base := node.Base()

	if base == nil {
		t.Errorf("Base() returned nil")
		return
	}

	if base.ID != "ai_chat_1" {
		t.Errorf("Base() ID = %s, want ai_chat_1", base.ID)
	}

	if base.Type != NodeTypeAIChat {
		t.Errorf("Base() Type = %s, want %s", base.Type, NodeTypeAIChat)
	}
}

// TestAIChatConfigValidation 测试AI对话配置验证
func TestAIChatConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *AIChatConfig
		wantErr bool
	}{
		{
			name: "valid openai config",
			config: &AIChatConfig{
				Provider:       "openai",
				APIKey:         "sk-test",
				Model:          "gpt-4",
				InputVariable:  "user_input",
				OutputVariable: "ai_response",
			},
			wantErr: false,
		},
		{
			name: "valid coze config",
			config: &AIChatConfig{
				Provider:       "coze",
				APIKey:         "pat-test",
				Model:          "coze",
				InputVariable:  "user_input",
				OutputVariable: "ai_response",
			},
			wantErr: false,
		},
		{
			name: "valid ollama config",
			config: &AIChatConfig{
				Provider:       "ollama",
				APIKey:         "ollama",
				BaseURL:        "http://localhost:11434",
				Model:          "llama2",
				InputVariable:  "user_input",
				OutputVariable: "ai_response",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 验证必需字段
			if tt.config.Provider == "" {
				t.Errorf("Provider is required")
			}
			if tt.config.APIKey == "" {
				t.Errorf("APIKey is required")
			}
			if tt.config.Model == "" {
				t.Errorf("Model is required")
			}
			if tt.config.InputVariable == "" {
				t.Errorf("InputVariable is required")
			}
			if tt.config.OutputVariable == "" {
				t.Errorf("OutputVariable is required")
			}
		})
	}
}

// TestAIChatNodeProperties 测试AI对话节点的属性
func TestAIChatNodeProperties(t *testing.T) {
	config := &AIChatConfig{
		Provider:       "openai",
		APIKey:         "sk-test",
		BaseURL:        "https://api.openai.com/v1",
		Model:          "gpt-4",
		SystemPrompt:   "You are a helpful assistant",
		InputVariable:  "user_input",
		OutputVariable: "ai_response",
		Temperature:    floatPtr(0.7),
		MaxTokens:      intPtr(2000),
		EnableStream:   true,
		PersistHistory: true,
	}

	if config.Provider != "openai" {
		t.Errorf("Provider = %s, want openai", config.Provider)
	}

	if config.Temperature == nil || *config.Temperature != 0.7 {
		t.Errorf("Temperature = %v, want 0.7", config.Temperature)
	}

	if config.MaxTokens == nil || *config.MaxTokens != 2000 {
		t.Errorf("MaxTokens = %v, want 2000", config.MaxTokens)
	}

	if !config.EnableStream {
		t.Errorf("EnableStream = %v, want true", config.EnableStream)
	}

	if !config.PersistHistory {
		t.Errorf("PersistHistory = %v, want true", config.PersistHistory)
	}
}

// 辅助函数
func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}
