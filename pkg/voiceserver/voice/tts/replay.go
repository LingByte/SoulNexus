package tts

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"errors"
)

// ReplayService adapts a prefetch PCM channel to Service so Pipeline can play
// audio that was synthesized ahead of the Speak call.
type ReplayService struct {
	ch <-chan []byte
}

// NewReplayService wraps a read-only PCM channel (closed when synthesis ends).
func NewReplayService(ch <-chan []byte) Service {
	return &ReplayService{ch: ch}
}

func (r *ReplayService) SynthesizeStream(ctx context.Context, _ string, onPCMChunk func([]byte) error) error {
	if r == nil || r.ch == nil {
		return errors.New("voice/tts: nil replay channel")
	}
	if onPCMChunk == nil {
		return errors.New("voice/tts: nil onPCMChunk")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case pcm, ok := <-r.ch:
			if !ok {
				return nil
			}
			if len(pcm) == 0 {
				continue
			}
			if len(pcm)&1 != 0 {
				pcm = pcm[:len(pcm)-1]
				if len(pcm) == 0 {
					continue
				}
			}
			if err := onPCMChunk(pcm); err != nil {
				return err
			}
		}
	}
}
