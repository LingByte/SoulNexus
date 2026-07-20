// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package voiceprint

import (
	"testing"
)

func TestNewVolcengineClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *VolcengineConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "missing access_key",
			config: &VolcengineConfig{
				SecretKey: "secret",
			},
			wantErr: true,
		},
		{
			name: "missing secret_key",
			config: &VolcengineConfig{
				AccessKey: "key",
			},
			wantErr: true,
		},
		{
			name: "valid config",
			config: &VolcengineConfig{
				AccessKey: "test-access",
				SecretKey: "test-secret",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewVolcengineClient(tt.config, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVolcengineClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewVolcengineClient() returned nil client")
			}
			if !tt.wantErr && client != nil {
				_ = client.Close()
			}
		})
	}
}

func TestVolcengineConfig_Defaults(t *testing.T) {
	config := &VolcengineConfig{
		AccessKey: "test-access",
		SecretKey: "test-secret",
	}

	client, err := NewVolcengineClient(config, nil)
	if err != nil {
		t.Fatalf("NewVolcengineClient() error = %v", err)
	}
	defer client.Close()

	if client.config.Region != "cn-north-1" {
		t.Errorf("Region = %s, want cn-north-1", client.config.Region)
	}

	if client.config.BaseURL != "https://rtc.volcengineapi.com" {
		t.Errorf("BaseURL = %s, want https://rtc.volcengineapi.com", client.config.BaseURL)
	}
}

func TestVolcengineClient_Close(t *testing.T) {
	config := &VolcengineConfig{
		AccessKey: "test-access",
		SecretKey: "test-secret",
	}

	client, err := NewVolcengineClient(config, nil)
	if err != nil {
		t.Fatalf("NewVolcengineClient() error = %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
