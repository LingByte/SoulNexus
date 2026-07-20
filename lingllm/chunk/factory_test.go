package chunk

import (
	"context"
	"testing"
)

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

func TestNewDefaultFactory(t *testing.T) {
	factory := NewDefaultFactory()

	if factory == nil {
		t.Fatal("expected factory, got nil")
	}

	if len(factory.factories) == 0 {
		t.Fatal("expected factories to be registered")
	}
}

func TestDefaultFactory_Register(t *testing.T) {
	factory := NewDefaultFactory()

	// Test registering a proper factory
	llmFactory := &LLMChunkerFactory{}
	err := factory.Register(llmFactory)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDefaultFactory_RegisterNil(t *testing.T) {
	factory := NewDefaultFactory()

	err := factory.Register(nil)
	if err != ErrInvalidConfig {
		t.Errorf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestDefaultFactory_Create(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "rules_structured",
	}

	chunker, err := factory.Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chunker == nil {
		t.Fatal("expected chunker, got nil")
	}

	if chunker.Provider() != "rules_structured" {
		t.Errorf("expected provider 'rules_structured', got '%s'", chunker.Provider())
	}
}

func TestDefaultFactory_CreateNilConfig(t *testing.T) {
	factory := NewDefaultFactory()

	_, err := factory.Create(context.Background(), nil)
	if err != ErrInvalidConfig {
		t.Errorf("expected ErrInvalidConfig, got %v", err)
	}
}

func TestDefaultFactory_CreateEmptyProvider(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "",
	}

	_, err := factory.Create(context.Background(), cfg)
	if err != ErrProviderNotFound {
		t.Errorf("expected ErrProviderNotFound, got %v", err)
	}
}

func TestDefaultFactory_CreateUnknownProvider(t *testing.T) {
	factory := NewDefaultFactory()

	cfg := &Config{
		Provider: "unknown_provider",
	}

	_, err := factory.Create(context.Background(), cfg)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestDefaultFactory_List(t *testing.T) {
	factory := NewDefaultFactory()

	providers := factory.List()
	if len(providers) == 0 {
		t.Fatal("expected at least one provider")
	}

	expectedProviders := map[string]bool{
		"llm":              true,
		"rules_structured": true,
		"rules_table_kv":   true,
		"router":           true,
	}

	for _, provider := range providers {
		if !expectedProviders[provider] {
			t.Errorf("unexpected provider: %s", provider)
		}
	}
}

func TestDefaultFactory_Supports(t *testing.T) {
	factory := NewDefaultFactory()

	tests := []struct {
		provider string
		expected bool
	}{
		{"llm", true},
		{"rules_structured", true},
		{"rules_table_kv", true},
		{"router", true},
		{"unknown", false},
		{"", false},
	}

	for _, test := range tests {
		if factory.Supports(test.provider) != test.expected {
			t.Errorf("Supports(%s): expected %v, got %v", test.provider, test.expected, factory.Supports(test.provider))
		}
	}
}

func TestLLMChunkerFactory(t *testing.T) {
	factory := &LLMChunkerFactory{}

	if factory.Name() != "llm" {
		t.Errorf("expected name 'llm', got '%s'", factory.Name())
	}

	if !factory.Supports("llm") {
		t.Error("expected to support 'llm'")
	}

	if factory.Supports("other") {
		t.Error("expected not to support 'other'")
	}
}

func TestStructuredChunkerFactory(t *testing.T) {
	factory := &StructuredChunkerFactory{}

	if factory.Name() != "rules_structured" {
		t.Errorf("expected name 'rules_structured', got '%s'", factory.Name())
	}

	if !factory.Supports("rules_structured") {
		t.Error("expected to support 'rules_structured'")
	}
}

func TestTableKVChunkerFactory(t *testing.T) {
	factory := &TableKVChunkerFactory{}

	if factory.Name() != "rules_table_kv" {
		t.Errorf("expected name 'rules_table_kv', got '%s'", factory.Name())
	}

	if !factory.Supports("rules_table_kv") {
		t.Error("expected to support 'rules_table_kv'")
	}
}

func TestRouterChunkerFactory(t *testing.T) {
	factory := &RouterChunkerFactory{}

	if factory.Name() != "router" {
		t.Errorf("expected name 'router', got '%s'", factory.Name())
	}

	if !factory.Supports("router") {
		t.Error("expected to support 'router'")
	}
}

func TestGetFactory(t *testing.T) {
	factory1 := GetFactory()
	factory2 := GetFactory()

	if factory1 != factory2 {
		t.Error("expected same factory instance (singleton)")
	}
}

func TestGlobalCreate(t *testing.T) {
	cfg := &Config{
		Provider: "rules_structured",
	}

	chunker, err := Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if chunker == nil {
		t.Fatal("expected chunker, got nil")
	}
}

func TestGlobalRegister(t *testing.T) {
	factory := &StructuredChunkerFactory{}
	err := Register(factory)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGlobalList(t *testing.T) {
	providers := List()

	if len(providers) == 0 {
		t.Fatal("expected at least one provider")
	}
}

func TestGlobalSupports(t *testing.T) {
	if !Supports("rules_structured") {
		t.Error("expected to support 'rules_structured'")
	}

	if Supports("unknown") {
		t.Error("expected not to support 'unknown'")
	}
}
