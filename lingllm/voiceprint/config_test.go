// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceprint

import (
	"testing"
	"time"
)

func TestNewConfigFromEnv(t *testing.T) {
	tests := []struct {
		name       string
		getEnvFunc func(string) string
		want       *Config
	}{
		{
			name: "default values",
			getEnvFunc: func(key string) string {
				return ""
			},
			want: &Config{
				Enabled:             false,
				BaseURL:             "http://localhost:8005",
				APIKey:              "",
				Timeout:             30 * time.Second,
				ConnectTimeout:      10 * time.Second,
				MaxRetries:          3,
				RetryInterval:       1 * time.Second,
				SimilarityThreshold: 0.6,
				MaxCandidates:       10,
				CacheEnabled:        true,
				CacheTTL:            5 * time.Minute,
				LogEnabled:          true,
				LogLevel:            "info",
			},
		},
		{
			name: "custom values",
			getEnvFunc: func(key string) string {
				switch key {
				case "VOICEPRINT_ENABLED":
					return "true"
				case "VOICEPRINT_BASE_URL":
					return "http://custom:9000"
				case "VOICEPRINT_API_KEY":
					return "custom-key"
				case "VOICEPRINT_TIMEOUT":
					return "60s"
				case "VOICEPRINT_MAX_RETRIES":
					return "5"
				case "VOICEPRINT_SIMILARITY_THRESHOLD":
					return "0.8"
				case "VOICEPRINT_MAX_CANDIDATES":
					return "20"
				case "VOICEPRINT_CACHE_ENABLED":
					return "false"
				case "VOICEPRINT_LOG_LEVEL":
					return "debug"
				default:
					return ""
				}
			},
			want: &Config{
				Enabled:             true,
				BaseURL:             "http://custom:9000",
				APIKey:              "custom-key",
				Timeout:             60 * time.Second,
				ConnectTimeout:      10 * time.Second,
				MaxRetries:          5,
				RetryInterval:       1 * time.Second,
				SimilarityThreshold: 0.8,
				MaxCandidates:       20,
				CacheEnabled:        false,
				CacheTTL:            5 * time.Minute,
				LogEnabled:          true,
				LogLevel:            "debug",
			},
		},
		{
			name: "service url fallback",
			getEnvFunc: func(key string) string {
				if key == "VOICEPRINT_SERVICE_URL" {
					return "http://service:8005"
				}
				return ""
			},
			want: &Config{
				BaseURL:             "http://service:8005",
				Timeout:             30 * time.Second,
				ConnectTimeout:      10 * time.Second,
				MaxRetries:          3,
				RetryInterval:       1 * time.Second,
				SimilarityThreshold: 0.6,
				MaxCandidates:       10,
				CacheEnabled:        true,
				CacheTTL:            5 * time.Minute,
				LogEnabled:          true,
				LogLevel:            "info",
			},
		},
		{
			name: "invalid duration",
			getEnvFunc: func(key string) string {
				if key == "VOICEPRINT_TIMEOUT" {
					return "invalid"
				}
				return ""
			},
			want: &Config{
				BaseURL:             "http://localhost:8005",
				Timeout:             30 * time.Second,
				ConnectTimeout:      10 * time.Second,
				MaxRetries:          3,
				RetryInterval:       1 * time.Second,
				SimilarityThreshold: 0.6,
				MaxCandidates:       10,
				CacheEnabled:        true,
				CacheTTL:            5 * time.Minute,
				LogEnabled:          true,
				LogLevel:            "info",
			},
		},
		{
			name: "invalid integer",
			getEnvFunc: func(key string) string {
				if key == "VOICEPRINT_MAX_RETRIES" {
					return "invalid"
				}
				return ""
			},
			want: &Config{
				BaseURL:             "http://localhost:8005",
				Timeout:             30 * time.Second,
				ConnectTimeout:      10 * time.Second,
				MaxRetries:          3,
				RetryInterval:       1 * time.Second,
				SimilarityThreshold: 0.6,
				MaxCandidates:       10,
				CacheEnabled:        true,
				CacheTTL:            5 * time.Minute,
				LogEnabled:          true,
				LogLevel:            "info",
			},
		},
		{
			name: "invalid float",
			getEnvFunc: func(key string) string {
				if key == "VOICEPRINT_SIMILARITY_THRESHOLD" {
					return "invalid"
				}
				return ""
			},
			want: &Config{
				BaseURL:             "http://localhost:8005",
				Timeout:             30 * time.Second,
				ConnectTimeout:      10 * time.Second,
				MaxRetries:          3,
				RetryInterval:       1 * time.Second,
				SimilarityThreshold: 0.6,
				MaxCandidates:       10,
				CacheEnabled:        true,
				CacheTTL:            5 * time.Minute,
				LogEnabled:          true,
				LogLevel:            "info",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewConfigFromEnv(tt.getEnvFunc)

			if got.Enabled != tt.want.Enabled {
				t.Errorf("Enabled = %v, want %v", got.Enabled, tt.want.Enabled)
			}
			if got.BaseURL != tt.want.BaseURL {
				t.Errorf("BaseURL = %s, want %s", got.BaseURL, tt.want.BaseURL)
			}
			if got.APIKey != tt.want.APIKey {
				t.Errorf("APIKey = %s, want %s", got.APIKey, tt.want.APIKey)
			}
			if got.Timeout != tt.want.Timeout {
				t.Errorf("Timeout = %v, want %v", got.Timeout, tt.want.Timeout)
			}
			if got.MaxRetries != tt.want.MaxRetries {
				t.Errorf("MaxRetries = %d, want %d", got.MaxRetries, tt.want.MaxRetries)
			}
			if got.SimilarityThreshold != tt.want.SimilarityThreshold {
				t.Errorf("SimilarityThreshold = %f, want %f", got.SimilarityThreshold, tt.want.SimilarityThreshold)
			}
			if got.MaxCandidates != tt.want.MaxCandidates {
				t.Errorf("MaxCandidates = %d, want %d", got.MaxCandidates, tt.want.MaxCandidates)
			}
			if got.CacheEnabled != tt.want.CacheEnabled {
				t.Errorf("CacheEnabled = %v, want %v", got.CacheEnabled, tt.want.CacheEnabled)
			}
			if got.LogLevel != tt.want.LogLevel {
				t.Errorf("LogLevel = %s, want %s", got.LogLevel, tt.want.LogLevel)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "disabled service",
			config: &Config{
				Enabled: false,
			},
			wantErr: false,
		},
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
			name: "missing base url",
			config: &Config{
				Enabled:       true,
				APIKey:        "test-key",
				MaxCandidates: 10,
			},
			wantErr: true,
		},
		{
			name: "missing api key",
			config: &Config{
				Enabled:       true,
				BaseURL:       "http://localhost:8005",
				MaxCandidates: 10,
			},
			wantErr: true,
		},
		{
			name: "invalid similarity threshold - too low",
			config: &Config{
				Enabled:             true,
				BaseURL:             "http://localhost:8005",
				APIKey:              "test-key",
				SimilarityThreshold: -0.1,
				MaxCandidates:       10,
			},
			wantErr: true,
		},
		{
			name: "invalid similarity threshold - too high",
			config: &Config{
				Enabled:             true,
				BaseURL:             "http://localhost:8005",
				APIKey:              "test-key",
				SimilarityThreshold: 1.1,
				MaxCandidates:       10,
			},
			wantErr: true,
		},
		{
			name: "invalid max candidates",
			config: &Config{
				Enabled:             true,
				BaseURL:             "http://localhost:8005",
				APIKey:              "test-key",
				SimilarityThreshold: 0.6,
				MaxCandidates:       0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config == nil {
		t.Error("DefaultConfig() returned nil")
	}

	// Should have default values
	if config.BaseURL == "" {
		t.Error("DefaultConfig() BaseURL is empty")
	}
	if config.Timeout == 0 {
		t.Error("DefaultConfig() Timeout is zero")
	}
}

func TestGetEnv(t *testing.T) {
	// Test with non-existent key
	result := getEnv("NONEXISTENT_KEY_12345")
	if result != "" {
		t.Errorf("getEnv() for nonexistent key = %s, want empty string", result)
	}
}

func TestConfig_AllFields(t *testing.T) {
	config := &Config{
		Enabled:             true,
		BaseURL:             "http://test:8005",
		APIKey:              "test-key",
		Timeout:             30 * time.Second,
		ConnectTimeout:      10 * time.Second,
		MaxRetries:          3,
		RetryInterval:       1 * time.Second,
		SimilarityThreshold: 0.6,
		MaxCandidates:       10,
		CacheEnabled:        true,
		CacheTTL:            5 * time.Minute,
		LogEnabled:          true,
		LogLevel:            "info",
	}

	if err := config.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	// Verify all fields are set
	if config.Enabled != true {
		t.Error("Enabled field not set correctly")
	}
	if config.BaseURL != "http://test:8005" {
		t.Error("BaseURL field not set correctly")
	}
	if config.APIKey != "test-key" {
		t.Error("APIKey field not set correctly")
	}
	if config.Timeout != 30*time.Second {
		t.Error("Timeout field not set correctly")
	}
	if config.ConnectTimeout != 10*time.Second {
		t.Error("ConnectTimeout field not set correctly")
	}
	if config.MaxRetries != 3 {
		t.Error("MaxRetries field not set correctly")
	}
	if config.RetryInterval != 1*time.Second {
		t.Error("RetryInterval field not set correctly")
	}
	if config.SimilarityThreshold != 0.6 {
		t.Error("SimilarityThreshold field not set correctly")
	}
	if config.MaxCandidates != 10 {
		t.Error("MaxCandidates field not set correctly")
	}
	if config.CacheEnabled != true {
		t.Error("CacheEnabled field not set correctly")
	}
	if config.CacheTTL != 5*time.Minute {
		t.Error("CacheTTL field not set correctly")
	}
	if config.LogEnabled != true {
		t.Error("LogEnabled field not set correctly")
	}
	if config.LogLevel != "info" {
		t.Error("LogLevel field not set correctly")
	}
}
