package realtime

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// Credential-driven Agent factory. The tenant control plane persists JSON
// of the form `{ "provider": "<slug>", ...vendor fields }`. voice attach
// sites call NewAgentFromCredential to materialise an Agent without
// importing any provider package directly — providers self-register via
// blank import at the application entrypoint).

import (
	"fmt"
	"strings"
)

// NewAgentFromCredential resolves a Provider by `cfg["provider"]` and
// constructs an Agent. Returns ErrUnknownProvider when the slug is missing
// or unregistered.
func NewAgentFromCredential(cfg map[string]any, opts Options) (Agent, error) {
	if len(cfg) == 0 {
		return nil, fmt.Errorf("realtime: empty credential config")
	}
	slug := strings.ToLower(strings.TrimSpace(stringField(cfg, "provider")))
	if slug == "" {
		return nil, fmt.Errorf("realtime: missing provider field")
	}
	p := Lookup(slug)
	if p == nil {
		return nil, &ErrUnknownProvider{Provider: slug}
	}
	if opts.OnEvent == nil {
		return nil, fmt.Errorf("realtime: Options.OnEvent is required")
	}
	if opts.InputSampleRate <= 0 {
		opts.InputSampleRate = 16000
	}
	if opts.OutputSampleRate <= 0 {
		opts.OutputSampleRate = 24000
	}
	return p(cfg, opts)
}

// stringField helpers — kept private; provider implementations should use
// their own typed config.
func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
