package tts

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"

	"github.com/LingByte/lingllm/synthesizer"
)

// FromSynthesisEngine wraps synthesizer.AudioSynthesisEngine as TTSService.
func FromSynthesisEngine(engine synthesizer.AudioSynthesisEngine) TTSService {
	if engine == nil {
		return nil
	}
	return &synthesisAdapter{engine: engine}
}

type synthesisAdapter struct {
	engine synthesizer.AudioSynthesisEngine
}

func (a *synthesisAdapter) Synthesize(ctx context.Context, text string, onPCMChunk func([]byte) error) error {
	if a == nil || a.engine == nil {
		return fmt.Errorf("tts: nil synthesis engine")
	}
	h := &streamHandler{fn: onPCMChunk}
	return a.engine.Synthesize(ctx, h, text)
}

type streamHandler struct {
	fn  func([]byte) error
	err error
}

func (h *streamHandler) OnMessage(data []byte) {
	if h == nil || h.fn == nil || len(data) == 0 || h.err != nil {
		return
	}
	if err := h.fn(data); err != nil {
		h.err = err
	}
}

func (h *streamHandler) OnTimestamp(_ synthesizer.SentenceTimestamp) {}
