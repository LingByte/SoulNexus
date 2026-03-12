package voiceprint

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	// 测试默认配置
	config := DefaultConfig()
	config.Enabled = true
	config.BaseURL = "http://localhost:8005"
	config.APIKey = "test-api-key"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if !client.IsEnabled() {
		t.Error("Client should be enabled")
	}

	// 测试无效配置
	invalidConfig := &Config{
		Enabled: true,
		BaseURL: "", // 无效的BaseURL
	}

	_, err = NewClient(invalidConfig)
	if err == nil {
		t.Error("Should fail with invalid config")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Enabled:             true,
				BaseURL:             "http://localhost:8005",
				APIKey:              "test-key",
				SimilarityThreshold: 0.6,
				MaxCandidates:       10,
			},
			wantErr: false,
		},
		{
			name: "disabled service",
			config: &Config{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "missing base url",
			config: &Config{
				Enabled: true,
				APIKey:  "test-key",
			},
			wantErr: true,
		},
		{
			name: "missing api key",
			config: &Config{
				Enabled: true,
				BaseURL: "http://localhost:8005",
			},
			wantErr: true,
		},
		{
			name: "invalid similarity threshold",
			config: &Config{
				Enabled:             true,
				BaseURL:             "http://localhost:8005",
				APIKey:              "test-key",
				SimilarityThreshold: 1.5,
			},
			wantErr: true,
		},
		{
			name: "invalid max candidates",
			config: &Config{
				Enabled:       true,
				BaseURL:       "http://localhost:8005",
				APIKey:        "test-key",
				MaxCandidates: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientMethods(t *testing.T) {
	// 创建测试客户端（服务未启用）
	config := DefaultConfig()
	config.Enabled = false

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// 测试服务未启用的情况
	_, err = client.HealthCheck(ctx)
	if err != ErrServiceDisabled {
		t.Errorf("Expected ErrServiceDisabled, got %v", err)
	}

	_, err = client.RegisterVoiceprint(ctx, &RegisterRequest{})
	if err != ErrServiceDisabled {
		t.Errorf("Expected ErrServiceDisabled, got %v", err)
	}

	_, err = client.IdentifyVoiceprint(ctx, &IdentifyRequest{})
	if err != ErrServiceDisabled {
		t.Errorf("Expected ErrServiceDisabled, got %v", err)
	}

	_, err = client.DeleteVoiceprint(ctx, "test")
	if err != ErrServiceDisabled {
		t.Errorf("Expected ErrServiceDisabled, got %v", err)
	}
}

func TestRequestValidation(t *testing.T) {
	config := DefaultConfig()
	config.Enabled = true
	config.BaseURL = "http://localhost:8005"
	config.APIKey = "test-key"

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// 测试注册请求验证
	tests := []struct {
		name    string
		req     *RegisterRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &RegisterRequest{
				SpeakerID:   "test_speaker",
				AssistantID: "test_assistant",
				AudioData:   []byte("RIFF....WAVE...."), // 简单的WAV头部模拟
			},
			wantErr: false,
		},
		{
			name: "missing speaker id",
			req: &RegisterRequest{
				AssistantID: "test_assistant",
				AudioData:   []byte("RIFF....WAVE...."),
			},
			wantErr: true,
		},
		{
			name: "missing audio data",
			req: &RegisterRequest{
				SpeakerID:   "test_speaker",
				AssistantID: "test_assistant",
			},
			wantErr: true,
		},
		{
			name: "invalid audio format",
			req: &RegisterRequest{
				SpeakerID:   "test_speaker",
				AssistantID: "test_assistant",
				AudioData:   []byte("invalid audio data"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.validateRegisterRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRegisterRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetConfidenceLevel(t *testing.T) {
	config := DefaultConfig()
	client, _ := NewClient(config)

	tests := []struct {
		score    float64
		expected string
	}{
		{0.9, "very_high"},
		{0.7, "high"},
		{0.5, "medium"},
		{0.3, "low"},
		{0.1, "very_low"},
	}

	for _, tt := range tests {
		result := client.getConfidenceLevel(tt.score)
		if result != tt.expected {
			t.Errorf("getConfidenceLevel(%f) = %s, want %s", tt.score, result, tt.expected)
		}
	}
}

// 基准测试
func BenchmarkClientCreation(b *testing.B) {
	config := DefaultConfig()
	config.Enabled = true
	config.BaseURL = "http://localhost:8005"
	config.APIKey = "test-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := NewClient(config)
		if err != nil {
			b.Fatal(err)
		}
		client.Close()
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	config := DefaultConfig()
	config.Enabled = true
	config.BaseURL = "http://localhost:8005"
	config.APIKey = "test-key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.Validate()
	}
}
