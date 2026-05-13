package tts

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"

	"github.com/LingByte/SoulNexus/pkg/synthesizer"
)

// Service is the minimal streaming TTS contract the Pipeline consumes.
//
// SynthesizeStream synthesizes `text` and invokes `onPCMChunk` for each chunk
// of PCM16 little-endian mono samples as they arrive. The chunk size is
// vendor-specific; Pipeline reframes them to a fixed frame size.
//
// Implementations MUST return when ctx is cancelled.
type Service interface {
	SynthesizeStream(ctx context.Context, text string, onPCMChunk func([]byte) error) error
}

// ServiceFunc is a convenience adapter.
type ServiceFunc func(ctx context.Context, text string, onPCMChunk func([]byte) error) error

// SynthesizeStream implements Service.
func (f ServiceFunc) SynthesizeStream(ctx context.Context, text string, onPCMChunk func([]byte) error) error {
	if f == nil {
		return fmt.Errorf("voice/tts: nil ServiceFunc")
	}
	return f(ctx, text, onPCMChunk)
}

// FromSynthesisService wraps a pkg/synthesizer.SynthesisService so it can be
// fed into the Pipeline. The upstream service emits `synthesizer.SynthesisHandler`
// callbacks on every chunk; we forward OnMessage → onPCMChunk.
//
// Note: the underlying service is expected to produce PCM16LE mono at a
// sample rate the caller has already configured (e.g. via NewQcloudTTSConfig
// with format="pcm" and the desired rate). The Pipeline can resample on the
// downstream side if it differs from the output bridge.
func FromSynthesisService(svc synthesizer.SynthesisService) Service {
	if svc == nil {
		return nil
	}
	return &synthesisAdapter{svc: svc}
}

type synthesisAdapter struct {
	svc synthesizer.SynthesisService
}

func (a *synthesisAdapter) SynthesizeStream(ctx context.Context, text string, onPCMChunk func([]byte) error) error {
	if a == nil || a.svc == nil {
		return fmt.Errorf("voice/tts: nil synthesizer service")
	}
	h := &streamHandler{fn: onPCMChunk}
	return a.svc.Synthesize(ctx, h, text)
}

// streamHandler adapts a push-callback into synthesizer.SynthesisHandler.
type streamHandler struct {
	fn  func([]byte) error
	err error
}

func (h *streamHandler) OnMessage(data []byte) {
	if h == nil || h.fn == nil || len(data) == 0 {
		return
	}
	if h.err != nil {
		return
	}
	if err := h.fn(data); err != nil {
		h.err = err
	}
}

func (h *streamHandler) OnTimestamp(_ synthesizer.SentenceTimestamp) {}
