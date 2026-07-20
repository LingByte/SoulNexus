// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package tenantcfg

import "context"

// ctxKey is a private type so other packages cannot accidentally
// collide with our context.Value key. Standard "unexported key type"
// idiom from Effective Go.
type ctxKey struct{}

// WithVoiceEnv returns a child context that carries env. Used to
// pass a freshly-loaded VoiceEnv down to downstream attach helpers
// so they can skip a redundant DB load.
//
// nil ctx is replaced with context.Background — callers should never
// pass nil but we tolerate it defensively rather than panic deep in
// the call chain.
func WithVoiceEnv(ctx context.Context, env VoiceEnv) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKey{}, env)
}

// VoiceEnvFromContext returns the VoiceEnv stashed by WithVoiceEnv,
// if any. The second return is false when no env was stashed (the
// caller must then load from the database via Resolve).
//
// nil ctx is safe — returns (zero, false).
func VoiceEnvFromContext(ctx context.Context) (VoiceEnv, bool) {
	if ctx == nil {
		return VoiceEnv{}, false
	}
	v, ok := ctx.Value(ctxKey{}).(VoiceEnv)
	return v, ok
}
