package tts

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
)

const synthMaxDefault = 1

var (
	synthSem     chan struct{}
	synthSemOnce sync.Once
)

func initSynthSemaphore() {
	n := synthMaxDefault
	if s := strings.TrimSpace(os.Getenv("VOICE_TTS_SYNTH_MAX")); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			n = v
		}
	}
	synthSem = make(chan struct{}, n)
}

func acquireSynth(ctx context.Context) error {
	synthSemOnce.Do(initSynthSemaphore)
	if synthSem == nil {
		return nil
	}
	select {
	case synthSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseSynth() {
	if synthSem == nil {
		return
	}
	select {
	case <-synthSem:
	default:
	}
}

// WithSynthConcurrency limits concurrent calls to inner.SynthesizeStream process-wide.
// Replay/cache hits should wrap the network-backed inner service, not replay adapters.
func WithSynthConcurrency(inner Service) Service {
	if inner == nil {
		return nil
	}
	return ServiceFunc(func(ctx context.Context, text string, onPCMChunk func([]byte) error) error {
		if err := acquireSynth(ctx); err != nil {
			return err
		}
		defer releaseSynth()
		return inner.SynthesizeStream(ctx, text, onPCMChunk)
	})
}
