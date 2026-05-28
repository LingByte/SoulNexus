// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package llm

import (
	"encoding/json"
	"strings"
)

// NormalizeOpenAIChatCompletionBody normalizes an OpenAI-compatible chat completion
// request before relaying upstream. It preserves caller fields and only fills gaps.
func NormalizeOpenAIChatCompletionBody(body []byte) []byte {
	var m map[string]json.RawMessage
	if json.Unmarshal(body, &m) != nil || len(m) == 0 {
		return body
	}
	changed := false

	// Newer OpenAI models may use max_completion_tokens; map to max_tokens when missing.
	if _, hasMax := m["max_tokens"]; !hasMax {
		if raw, ok := m["max_completion_tokens"]; ok {
			m["max_tokens"] = raw
			changed = true
		}
	}

	if !changed {
		return body
	}
	out, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return out
}

// applyOpenAICompatGenParams merges generation knobs into an OpenAI-shaped JSON body.
func applyOpenAICompatGenParams(body map[string]any, options *QueryOptions) {
	if body == nil || options == nil {
		return
	}
	if options.MaxTokens > 0 {
		body["max_tokens"] = options.MaxTokens
	}
	if options.Temperature != 0 {
		body["temperature"] = options.Temperature
	}
	if options.TopP != 0 {
		body["top_p"] = options.TopP
	}
	if len(options.LogitBias) > 0 {
		body["logit_bias"] = options.LogitBias
	}
}

// anthropicOutputMaxTokens returns the max output tokens for Anthropic /v1/messages.
// Anthropic requires max_tokens > 0; when unset we use a conservative default.
func anthropicOutputMaxTokens(n int) int {
	if n > 0 {
		return n
	}
	return 1024
}

func dashscopeCompatibleMode(baseURL string) bool {
	u := strings.ToLower(strings.TrimSpace(baseURL))
	return strings.Contains(u, "compatible-mode") || strings.Contains(u, "/v1")
}
