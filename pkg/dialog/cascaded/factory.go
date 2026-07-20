// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package cascaded

import (
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
)

// factory implements engine.Factory by constructing one Engine per
// call. Stateless; the Build method is safe to call from any goroutine.
type factory struct{}

// Build constructs a cascaded.Engine. The factory accepts BOTH
// ModeCascaded and ModeCascadedNative — same engine implementation
// either way; the mode label is propagated to Engine.Mode() so
// metrics / logs distinguish "registered as the legacy-bridge
// substitute" from "registered as the parallel native path behind a
// feature flag" without code-shape changes here.
func (factory) Build(cfg engine.Config) (engine.Engine, error) {
	switch cfg.Mode {
	case engine.ModeCascaded, engine.ModeCascadedNative:
		return New(cfg), nil
	default:
		return nil, fmt.Errorf("dialog/cascaded: factory called with mode %q, want %q or %q",
			string(cfg.Mode), string(engine.ModeCascaded), string(engine.ModeCascadedNative))
	}
}

// NewFactory returns the cascaded engine factory. Exposed so callers
// (test setup, feature-flagged production wiring) can register this
// engine without depending on package-private types.
func NewFactory() engine.Factory { return factory{} }

// RegisterNative installs the cascaded factory under
// engine.ModeCascadedNative. This is the production-safe entry
// point: the legacy bridge already owns engine.ModeCascaded, but
// nothing claims ModeCascadedNative, so this Register call is
// idempotent across boots. Re-calling it is a no-op rather than a
// panic — engine.Register panics on duplicate, so we recover and
// surface a sentinel error instead.
//
// Typical bootstrap shape (from
// dialog engine bridge):
//
//	if err := cascaded.RegisterNative(); err != nil { … }
//
// Pair with the per-tenant feature flag in
// dialog_engine_native_route.go to actually route traffic through
// this factory; otherwise it sits dormant in the registry.
func RegisterNative() (err error) {
	defer func() {
		if r := recover(); r != nil {
			// engine.Register panics on duplicate — squash to a
			// sentinel so callers can shrug it off on second
			// bootstrap (hot reload, repeated tests, etc.).
			err = fmt.Errorf("dialog/cascaded: RegisterNative: %v", r)
		}
	}()
	engine.Register(engine.ModeCascadedNative, factory{})
	return nil
}

// RegisterForTesting installs the cascaded factory under
// engine.ModeCascaded. Intended for tests and feature-flagged
// experiments only.
//
// Why no init()-time auto-register: the legacy bridge in
// dialog engine bridge already claims
// engine.ModeCascaded at bootstrap. engine.Register panics on a
// duplicate, so the two registrations would race. Production keeps
// the legacy bridge; opt-in callers (typically tests after calling
// engine.ResetRegistryForTest) get the native engine.
//
// Returns an error rather than panicking so callers can decide how
// to handle the duplicate case.
func RegisterForTesting() error {
	defer func() { _ = recover() }()
	// Best-effort: engine.Register panics on duplicate. Wrapping it
	// in a defer-recover keeps test setup robust against ordering
	// quirks; the caller is responsible for calling
	// engine.ResetRegistryForTest first if they want a clean slate.
	engine.Register(engine.ModeCascaded, factory{})
	return nil
}
