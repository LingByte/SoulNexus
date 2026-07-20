// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package asr

import (
	"testing"
)

func TestNewDenoiserFactory(t *testing.T) {
	factory := NewDenoiserFactory()
	if factory == nil {
		t.Error("NewDenoiserFactory() returned nil")
	}
}

func TestDenoiserFactory_CreateDenoiser_None(t *testing.T) {
	factory := NewDenoiserFactory()
	component, err := factory.CreateDenoiser(DenoiserTypeNone, nil)
	if err != nil {
		t.Errorf("CreateDenoiser(None) error = %v", err)
	}
	if component != nil {
		t.Errorf("CreateDenoiser(None) returned non-nil component, want nil")
	}
}

func TestDenoiserFactory_CreateDenoiser_Simple(t *testing.T) {
	factory := NewDenoiserFactory()
	config := &SimpleDenoiserConfig{
		AECEnable:     true,
		AGCEnable:     true,
		SampleRate:    16000,
		Channels:      1,
		BitsPerSample: 16,
	}

	component, err := factory.CreateDenoiser(DenoiserTypeSimple, config)
	if err != nil {
		t.Errorf("CreateDenoiser(Simple) error = %v", err)
		return
	}
	if component == nil {
		t.Error("CreateDenoiser(Simple) returned nil component")
		return
	}

	simpleComponent, ok := component.(*SimpleDenoiserComponent)
	if !ok {
		t.Errorf("CreateDenoiser(Simple) returned wrong type: %T", component)
		return
	}

	defer simpleComponent.Close()

	if simpleComponent.Name() != "simple_denoiser" {
		t.Errorf("Name() = %s, want simple_denoiser", simpleComponent.Name())
	}
}

func TestDenoiserFactory_CreateDenoiser_SimpleWithNilConfig(t *testing.T) {
	factory := NewDenoiserFactory()
	component, err := factory.CreateDenoiser(DenoiserTypeSimple, nil)
	if err != nil {
		t.Errorf("CreateDenoiser(Simple, nil) error = %v", err)
		return
	}
	if component == nil {
		t.Error("CreateDenoiser(Simple, nil) returned nil component")
		return
	}

	simpleComponent, ok := component.(*SimpleDenoiserComponent)
	if !ok {
		t.Errorf("CreateDenoiser(Simple, nil) returned wrong type: %T", component)
		return
	}

	defer simpleComponent.Close()
}

func TestDenoiserFactory_CreateDenoiser_SimpleWithInvalidConfig(t *testing.T) {
	factory := NewDenoiserFactory()
	component, err := factory.CreateDenoiser(DenoiserTypeSimple, "invalid")
	if err == nil {
		t.Error("CreateDenoiser(Simple, invalid) expected error, got nil")
	}
	if component != nil {
		t.Errorf("CreateDenoiser(Simple, invalid) returned non-nil component")
	}
}

func TestDenoiserFactory_CreateDenoiser_RNNoise(t *testing.T) {
	factory := NewDenoiserFactory()
	component, err := factory.CreateDenoiser(DenoiserTypeRNNoise, nil)

	// RNNoise 可能不可用 (取决于 build tag)
	if err != nil && isRNNoiseAvailable() {
		t.Errorf("CreateDenoiser(RNNoise) error = %v, but RNNoise is available", err)
	}

	if err == nil && !isRNNoiseAvailable() {
		t.Error("CreateDenoiser(RNNoise) succeeded, but RNNoise is not available")
	}

	if component != nil && isRNNoiseAvailable() {
		// 如果 RNNoise 可用，验证组件不为 nil
		if component == nil {
			t.Error("CreateDenoiser(RNNoise) returned nil component when available")
		}
	}
}

func TestDenoiserFactory_CreateDenoiser_Unknown(t *testing.T) {
	factory := NewDenoiserFactory()
	component, err := factory.CreateDenoiser("unknown", nil)
	if err == nil {
		t.Error("CreateDenoiser(unknown) expected error, got nil")
	}
	if component != nil {
		t.Errorf("CreateDenoiser(unknown) returned non-nil component")
	}
}

func TestDenoiserFactory_GetAvailableDenoiserTypes(t *testing.T) {
	factory := NewDenoiserFactory()
	types := factory.GetAvailableDenoiserTypes()

	if len(types) == 0 {
		t.Error("GetAvailableDenoiserTypes() returned empty list")
		return
	}

	// 检查是否包含 none 和 simple
	hasNone := false
	hasSimple := false
	hasRNNoise := false

	for _, t := range types {
		switch t {
		case DenoiserTypeNone:
			hasNone = true
		case DenoiserTypeSimple:
			hasSimple = true
		case DenoiserTypeRNNoise:
			hasRNNoise = true
		}
	}

	if !hasNone {
		t.Error("GetAvailableDenoiserTypes() missing DenoiserTypeNone")
	}
	if !hasSimple {
		t.Error("GetAvailableDenoiserTypes() missing DenoiserTypeSimple")
	}

	// RNNoise 的可用性取决于 build tag
	if hasRNNoise && !isRNNoiseAvailable() {
		t.Error("GetAvailableDenoiserTypes() includes RNNoise but it's not available")
	}
	if !hasRNNoise && isRNNoiseAvailable() {
		t.Error("GetAvailableDenoiserTypes() missing RNNoise but it's available")
	}
}

func TestDenoiserType_String(t *testing.T) {
	tests := []struct {
		name     string
		denoiser DenoiserType
		want     string
	}{
		{
			name:     "none",
			denoiser: DenoiserTypeNone,
			want:     "none",
		},
		{
			name:     "simple",
			denoiser: DenoiserTypeSimple,
			want:     "simple",
		},
		{
			name:     "rnnoise",
			denoiser: DenoiserTypeRNNoise,
			want:     "rnnoise",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(tt.denoiser); got != tt.want {
				t.Errorf("DenoiserType string = %s, want %s", got, tt.want)
			}
		})
	}
}
