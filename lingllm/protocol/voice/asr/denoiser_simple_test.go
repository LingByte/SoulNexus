// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package asr

import (
	"context"
	"testing"
)

func TestNewSimpleDenoiser(t *testing.T) {
	tests := []struct {
		name    string
		config  *SimpleDenoiserConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid config",
			config: &SimpleDenoiserConfig{
				AECEnable:     true,
				AGCEnable:     true,
				SampleRate:    16000,
				Channels:      1,
				BitsPerSample: 16,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			denoiser, err := NewSimpleDenoiser(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSimpleDenoiser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && denoiser == nil {
				t.Error("NewSimpleDenoiser() returned nil denoiser")
				return
			}
			if denoiser != nil {
				denoiser.Close()
			}
		})
	}
}

func TestSimpleDenoiser_Process(t *testing.T) {
	denoiser, err := NewSimpleDenoiser(&SimpleDenoiserConfig{
		AECEnable:     true,
		AGCEnable:     true,
		SampleRate:    16000,
		Channels:      1,
		BitsPerSample: 16,
	})
	if err != nil {
		t.Fatalf("NewSimpleDenoiser() error = %v", err)
	}
	defer denoiser.Close()

	tests := []struct {
		name    string
		input   []byte
		wantLen int
	}{
		{
			name:    "empty input",
			input:   []byte{},
			wantLen: 0,
		},
		{
			name:    "valid input",
			input:   make([]byte, 1024),
			wantLen: 1024,
		},
		{
			name:    "small input",
			input:   make([]byte, 2),
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := denoiser.Process(tt.input)
			if len(output) != tt.wantLen {
				t.Errorf("Process() output length = %d, want %d", len(output), tt.wantLen)
			}
		})
	}
}

func TestSimpleDenoiser_Close(t *testing.T) {
	denoiser, err := NewSimpleDenoiser(nil)
	if err != nil {
		t.Fatalf("NewSimpleDenoiser() error = %v", err)
	}

	err = denoiser.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestNewSimpleDenoiserComponent(t *testing.T) {
	tests := []struct {
		name    string
		config  *SimpleDenoiserConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid config",
			config: &SimpleDenoiserConfig{
				AECEnable:     true,
				AGCEnable:     true,
				SampleRate:    16000,
				Channels:      1,
				BitsPerSample: 16,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component, err := NewSimpleDenoiserComponent(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSimpleDenoiserComponent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && component == nil {
				t.Error("NewSimpleDenoiserComponent() returned nil component")
				return
			}
			if component != nil {
				component.Close()
			}
		})
	}
}

func TestSimpleDenoiserComponent_Name(t *testing.T) {
	component, err := NewSimpleDenoiserComponent(nil)
	if err != nil {
		t.Fatalf("NewSimpleDenoiserComponent() error = %v", err)
	}
	defer component.Close()

	if got := component.Name(); got != "simple_denoiser" {
		t.Errorf("Name() = %s, want simple_denoiser", got)
	}
}

func TestSimpleDenoiserComponent_Process(t *testing.T) {
	component, err := NewSimpleDenoiserComponent(nil)
	if err != nil {
		t.Fatalf("NewSimpleDenoiserComponent() error = %v", err)
	}
	defer component.Close()

	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
	}{
		{
			name:    "valid PCM data",
			data:    make([]byte, 1024),
			wantErr: false,
		},
		{
			name:    "empty PCM data",
			data:    []byte{},
			wantErr: false,
		},
		{
			name:    "invalid data type",
			data:    "invalid",
			wantErr: true,
		},
		{
			name:    "nil data",
			data:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, ok, err := component.Process(context.Background(), tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Process() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !ok {
				t.Error("Process() ok = false, want true")
			}
			if !tt.wantErr && output == nil {
				t.Error("Process() output = nil, want non-nil")
			}
		})
	}
}

func TestSimpleDenoiserComponent_Close(t *testing.T) {
	component, err := NewSimpleDenoiserComponent(nil)
	if err != nil {
		t.Fatalf("NewSimpleDenoiserComponent() error = %v", err)
	}

	err = component.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func BenchmarkSimpleDenoiser_Process(b *testing.B) {
	denoiser, _ := NewSimpleDenoiser(&SimpleDenoiserConfig{
		AECEnable:     true,
		AGCEnable:     true,
		SampleRate:    16000,
		Channels:      1,
		BitsPerSample: 16,
	})
	defer denoiser.Close()

	input := make([]byte, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		denoiser.Process(input)
	}
}
