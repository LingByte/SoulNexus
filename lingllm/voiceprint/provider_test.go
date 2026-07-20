// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceprint

import (
	"testing"
)

func TestNewFactory(t *testing.T) {
	factory := NewFactory()
	if factory == nil {
		t.Error("NewFactory returned nil")
	}
}

func TestCreateProvider(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name    string
		config  *ProviderConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "unsupported provider",
			config: &ProviderConfig{
				Provider: "unknown",
			},
			wantErr: true,
		},
		{
			name: "http provider with minimal config",
			config: &ProviderConfig{
				Provider: ProviderHTTP,
				Options: map[string]interface{}{
					"base_url": "http://localhost:8080",
					"api_key":  "test-key",
				},
			},
			wantErr: false,
		},
		{
			name: "xunfei without options",
			config: &ProviderConfig{
				Provider: ProviderXunfei,
			},
			wantErr: true,
		},
		{
			name: "xunfei without api_key",
			config: &ProviderConfig{
				Provider: ProviderXunfei,
				Options: map[string]interface{}{
					"api_secret": "secret",
					"app_id":     "app123",
				},
			},
			wantErr: true,
		},
		{
			name: "xunfei without api_secret",
			config: &ProviderConfig{
				Provider: ProviderXunfei,
				Options: map[string]interface{}{
					"api_key": "key",
					"app_id":  "app123",
				},
			},
			wantErr: true,
		},
		{
			name: "xunfei without app_id",
			config: &ProviderConfig{
				Provider: ProviderXunfei,
				Options: map[string]interface{}{
					"api_key":    "key",
					"api_secret": "secret",
				},
			},
			wantErr: true,
		},
		{
			name: "valid xunfei config",
			config: &ProviderConfig{
				Provider: ProviderXunfei,
				Options: map[string]interface{}{
					"api_key":    "test-key",
					"api_secret": "test-secret",
					"app_id":     "test-app",
				},
			},
			wantErr: false,
		},
		{
			name: "valid xunfei config with base_url",
			config: &ProviderConfig{
				Provider: ProviderXunfei,
				Options: map[string]interface{}{
					"api_key":    "test-key",
					"api_secret": "test-secret",
					"app_id":     "test-app",
					"base_url":   "https://custom.api.com/v1",
				},
			},
			wantErr: false,
		},
		{
			name: "volcengine without options",
			config: &ProviderConfig{
				Provider: ProviderVolcengine,
			},
			wantErr: true,
		},
		{
			name: "volcengine without access_key",
			config: &ProviderConfig{
				Provider: ProviderVolcengine,
				Options: map[string]interface{}{
					"secret_key": "secret",
				},
			},
			wantErr: true,
		},
		{
			name: "volcengine without secret_key",
			config: &ProviderConfig{
				Provider: ProviderVolcengine,
				Options: map[string]interface{}{
					"access_key": "key",
				},
			},
			wantErr: true,
		},
		{
			name: "valid volcengine config",
			config: &ProviderConfig{
				Provider: ProviderVolcengine,
				Options: map[string]interface{}{
					"access_key": "test-access",
					"secret_key": "test-secret",
				},
			},
			wantErr: false,
		},
		{
			name: "valid volcengine config with region",
			config: &ProviderConfig{
				Provider: ProviderVolcengine,
				Options: map[string]interface{}{
					"access_key": "test-access",
					"secret_key": "test-secret",
					"region":     "cn-south-1",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProvider(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && provider == nil {
				t.Error("CreateProvider() returned nil provider")
			}
			if !tt.wantErr && provider != nil {
				_ = provider.Close()
			}
		})
	}
}

func TestXunfeiProviderAdapter(t *testing.T) {
	config := &XunfeiConfig{
		APIKey:    "test-key",
		APISecret: "test-secret",
		AppID:     "test-app",
	}

	client, err := NewXunfeiClient(config, nil)
	if err != nil {
		t.Fatalf("NewXunfeiClient() error = %v", err)
	}

	adapter := &XunfeiProviderAdapter{client: client}

	// Test that adapter implements VoiceprintProvider interface
	var _ VoiceprintProvider = adapter

	// Test Close
	err = adapter.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestProviderConfig(t *testing.T) {
	config := &ProviderConfig{
		Provider: ProviderXunfei,
		Options: map[string]interface{}{
			"api_key":    "test-key",
			"api_secret": "test-secret",
			"app_id":     "test-app",
		},
	}

	if config.Provider != ProviderXunfei {
		t.Errorf("Provider = %s, want %s", config.Provider, ProviderXunfei)
	}

	if config.Options["api_key"] != "test-key" {
		t.Error("api_key not set correctly")
	}
}
