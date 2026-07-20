// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package realtime

import (
	"errors"
	"fmt"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
)

// ErrNoBuilder is returned when the package-level AgentBuilder has
// not been installed before the factory is asked to Build an Engine.
// Production wiring MUST call SetAgentBuilder during
// bridge initialisation; otherwise Build short-circuits to this
// error and the legacy realtime path keeps serving traffic.
var ErrNoBuilder = errors.New("dialog/realtime: AgentBuilder not configured")

// BuilderProvider returns the AgentBuilder for the supplied call.
// Production wiring closes over the per-call tenant configuration
// (RealtimeConfigRaw + cred lookup) and returns a builder that
// constructs one Agent for that call. nil = "this call is not
// eligible for native routing" (the factory then returns
// ErrNoBuilder so the caller can fall back to the legacy path).
type BuilderProvider func(cfg engine.Config) AgentBuilder

var (
	builderMu       sync.RWMutex
	builderProvider BuilderProvider
)

// SetBuilderProvider installs the call-scoped AgentBuilder factory.
// Calling SetBuilderProvider(nil) unwires the factory — useful for
// tests that need to assert "factory falls back when unwired".
func SetBuilderProvider(p BuilderProvider) {
	builderMu.Lock()
	builderProvider = p
	builderMu.Unlock()
}

func loadBuilder(cfg engine.Config) AgentBuilder {
	builderMu.RLock()
	p := builderProvider
	builderMu.RUnlock()
	if p == nil {
		return nil
	}
	return p(cfg)
}

// factory implements engine.Factory. Stateless apart from the
// package-level builder provider (which is set once at voice bridge
// init).
type factory struct{}

// Build constructs a realtime.Engine. Only engine.ModeRealtime is accepted.
// Production wiring installs BuilderProvider at bootstrap (voiceattach).
func (factory) Build(cfg engine.Config) (engine.Engine, error) {
	if cfg.Mode != engine.ModeRealtime {
		return nil, fmt.Errorf("dialog/realtime: factory called with mode %q, want %q",
			string(cfg.Mode), string(engine.ModeRealtime))
	}
	b := loadBuilder(cfg)
	if b == nil {
		return nil, ErrNoBuilder
	}
	return New(cfg, b), nil
}

// NewFactory returns the realtime engine factory.
func NewFactory() engine.Factory { return factory{} }
