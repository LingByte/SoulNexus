// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package xiaozhi

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewServer_RealtimeRequiresFactory(t *testing.T) {
	_, err := NewServer(ServerConfig{
		Mode:   ModeRealtime,
		Logger: zap.NewNop(),
	})
	if err == nil {
		t.Fatal("expected error without RealtimeFactory")
	}
}

func TestNewServer_PipelineRequiresDialog(t *testing.T) {
	_, err := NewServer(ServerConfig{
		Mode:           ModePipeline,
		SessionFactory: stubFactory{},
		Logger:         zap.NewNop(),
	})
	if err == nil {
		t.Fatal("expected error without DialogWSURL")
	}
}

func TestNormalizeMode(t *testing.T) {
	if normalizeMode("realtime") != ModeRealtime {
		t.Fatalf("got %q", normalizeMode("realtime"))
	}
	if normalizeMode("omni") != ModeRealtime {
		t.Fatalf("got %q", normalizeMode("omni"))
	}
	if normalizeMode("") != ModePipeline {
		t.Fatalf("got %q", normalizeMode(""))
	}
}
