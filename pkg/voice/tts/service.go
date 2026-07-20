package tts

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"

	"github.com/LingByte/lingllm/synthesizer"
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

// FromSynthesisEngine wraps a lingllm AudioSynthesisEngine so it can be
// fed into the Pipeline. Engines with native SynthesizeStream (Aliyun
// Qwen-TTS realtime) are wired directly; others use batch Synthesize.
func FromSynthesisEngine(engine synthesizer.AudioSynthesisEngine) Service {
	if engine == nil {
		return nil
	}
	if native, ok := engine.(nativeStreamEngine); ok {
		return nativeStreamAdapter{native: native}
	}
	return &synthesisAdapter{engine: engine}
}

type nativeStreamEngine interface {
	SynthesizeStream(ctx context.Context, text string, callback func([]byte) error) error
}

type nativeStreamAdapter struct {
	native nativeStreamEngine
}

func (a nativeStreamAdapter) SynthesizeStream(ctx context.Context, text string, onPCMChunk func([]byte) error) error {
	if a.native == nil {
		return fmt.Errorf("voice/tts: nil native stream engine")
	}
	return a.native.SynthesizeStream(ctx, text, onPCMChunk)
}

type synthesisAdapter struct {
	engine synthesizer.AudioSynthesisEngine
}

func (a *synthesisAdapter) SynthesizeStream(ctx context.Context, text string, onPCMChunk func([]byte) error) error {
	if a == nil || a.engine == nil {
		return fmt.Errorf("voice/tts: nil synthesis engine")
	}
	h := &streamHandler{fn: onPCMChunk}
	return a.engine.Synthesize(ctx, h, text)
}

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
