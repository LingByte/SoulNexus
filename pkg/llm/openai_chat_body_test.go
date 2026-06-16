// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package llm

import (
	"encoding/json"
	"testing"
)

func TestPatchOpenAIChatCompletionBody_maxCompletionTokensAlias(t *testing.T) {
	in := []byte(`{"model":"gpt-4o","max_completion_tokens":32,"messages":[{"role":"user","content":"hi"}]}`)
	out := PatchOpenAIChatCompletionBody(in)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["max_completion_tokens"]; !ok {
		t.Fatal("expected max_completion_tokens preserved")
	}
	v, ok := m["max_tokens"].(float64)
	if !ok || int(v) != 32 {
		t.Fatalf("max_tokens=%v", m["max_tokens"])
	}
}

func TestPatchOpenAIChatCompletionBody_preservesExistingMaxTokens(t *testing.T) {
	in := []byte(`{"model":"qwen-plus","max_tokens":32,"temperature":0.2,"stream":false}`)
	out := PatchOpenAIChatCompletionBody(in)
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if int(m["max_tokens"].(float64)) != 32 {
		t.Fatalf("max_tokens=%v", m["max_tokens"])
	}
	if m["temperature"].(float64) != 0.2 {
		t.Fatalf("temperature=%v", m["temperature"])
	}
}

func TestAnthropicOutputMaxTokens(t *testing.T) {
	if anthropicOutputMaxTokens(32) != 32 {
		t.Fatal("expected exact user limit")
	}
	if anthropicOutputMaxTokens(0) != 1024 {
		t.Fatal("expected default when unset")
	}
}
