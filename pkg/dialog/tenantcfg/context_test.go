// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package tenantcfg

import (
	"context"
	"testing"
)

func TestWithVoiceEnv_RoundTrip(t *testing.T) {
	env := VoiceEnv{VoiceMode: "pipeline", ASRProvider: "qcloud"}
	ctx := WithVoiceEnv(context.Background(), env)
	got, ok := VoiceEnvFromContext(ctx)
	if !ok {
		t.Fatal("VoiceEnvFromContext reported not present after WithVoiceEnv")
	}
	if got.VoiceMode != "pipeline" || got.ASRProvider != "qcloud" {
		t.Errorf("round-trip env = %+v, want VoiceMode=pipeline ASRProvider=qcloud", got)
	}
}

func TestVoiceEnvFromContext_AbsentByDefault(t *testing.T) {
	if _, ok := VoiceEnvFromContext(context.Background()); ok {
		t.Error("plain context should not carry a VoiceEnv")
	}
}

func TestVoiceEnvFromContext_NilContextSafe(t *testing.T) {
	// Use a variable so `go vet` doesn't flag the nil literal —
	// we are explicitly contract-testing nil tolerance.
	var ctx context.Context
	if _, ok := VoiceEnvFromContext(ctx); ok {
		t.Error("nil ctx should return ok=false, not panic")
	}
}

func TestWithVoiceEnv_NilContextDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("WithVoiceEnv(nil, ...) panicked: %v", r)
		}
	}()
	var nilCtx context.Context
	ctx := WithVoiceEnv(nilCtx, VoiceEnv{VoiceMode: "realtime"})
	if got, ok := VoiceEnvFromContext(ctx); !ok || got.VoiceMode != "realtime" {
		t.Errorf("WithVoiceEnv(nil) round-trip failed: ok=%v env=%+v", ok, got)
	}
}

func TestWithVoiceEnv_OverridesPrevious(t *testing.T) {
	ctx := WithVoiceEnv(context.Background(), VoiceEnv{VoiceMode: "pipeline"})
	ctx = WithVoiceEnv(ctx, VoiceEnv{VoiceMode: "realtime"})
	got, ok := VoiceEnvFromContext(ctx)
	if !ok {
		t.Fatal("env not present after second WithVoiceEnv")
	}
	if got.VoiceMode != "realtime" {
		t.Errorf("VoiceMode = %q, want realtime (most recent wins)", got.VoiceMode)
	}
}
