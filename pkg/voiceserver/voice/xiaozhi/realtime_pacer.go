// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package xiaozhi

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// realtimeOutPacer buffers assistant PCM from pkg/realtime and emits
// fixed-size frames to the device at wall-clock rate — same contract as
// voice/tts.Pipeline with PaceRealtime + ttsSink (xiaozhi-esp32 expects
// ~60 ms Opus frames, not one giant binary blob per response).
type realtimeOutPacer struct {
	s *session

	mu         sync.Mutex
	pcmBuf     []byte
	frameBytes int
	frameDur   time.Duration

	playCtx    context.Context
	playCancel context.CancelFunc
	playWG     sync.WaitGroup

	turnActive atomic.Bool
}

func newRealtimeOutPacer(s *session) *realtimeOutPacer {
	ms := s.ttsWireFrameMs
	if ms <= 0 {
		ms = s.outFrameMs
	}
	if ms <= 0 {
		ms = 60
	}
	dur := time.Duration(ms) * time.Millisecond
	sr := s.outSR
	if sr <= 0 {
		sr = 16000
	}
	samples := int(float64(sr) * dur.Seconds())
	fb := samples * 2
	if fb < 2 {
		fb = 640
	}
	return &realtimeOutPacer{
		s:          s,
		frameBytes: fb,
		frameDur:   dur,
	}
}

func (p *realtimeOutPacer) close() {
	p.interrupt(false)
	p.playWG.Wait()
}

// push appends PCM16 @ s.outSR and ensures the pace loop is running.
func (p *realtimeOutPacer) push(pcm []byte) {
	if p == nil || len(pcm) == 0 || p.s.closed.Load() {
		return
	}
	p.mu.Lock()
	p.pcmBuf = append(p.pcmBuf, pcm...)
	if p.playCancel == nil {
		ctx, cancel := context.WithCancel(context.Background())
		p.playCancel = cancel
		p.playCtx = ctx
		p.playWG.Add(1)
		go p.playLoop(ctx)
	}
	p.mu.Unlock()
}

// endTurn drains any remaining PCM with pacing, then returns. Caller sends
// tts:stop after this returns when appropriate.
func (p *realtimeOutPacer) endTurn() {
	if p == nil {
		return
	}
	p.turnActive.Store(false)
	p.mu.Lock()
	cancel := p.playCancel
	p.playCancel = nil
	p.playCtx = nil
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	p.playWG.Wait()
}

// interrupt drops buffered audio and stops the pace loop (barge-in).
func (p *realtimeOutPacer) interrupt(sendStop bool) {
	if p == nil {
		return
	}
	p.turnActive.Store(false)
	p.mu.Lock()
	p.pcmBuf = nil
	cancel := p.playCancel
	p.playCancel = nil
	p.playCtx = nil
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	p.playWG.Wait()
	if sendStop && p.s.ttsActive.CompareAndSwap(true, false) {
		p.s.writeText(MakeTTSStateReply(p.s.sessionID, "stop", p.s.outFormat))
	}
}

func (p *realtimeOutPacer) playLoop(ctx context.Context) {
	defer p.playWG.Done()
	defer func() {
		p.mu.Lock()
		p.playCancel = nil
		p.playCtx = nil
		p.mu.Unlock()
	}()

	nextDeadline := time.Now()

	flush := func(forceTail bool) {
		for {
			if ctx.Err() != nil && !forceTail {
				return
			}
			p.mu.Lock()
			if len(p.pcmBuf) < p.frameBytes && !forceTail {
				p.mu.Unlock()
				return
			}
			if len(p.pcmBuf) == 0 {
				p.mu.Unlock()
				return
			}
			var frame []byte
			if len(p.pcmBuf) >= p.frameBytes {
				frame = make([]byte, p.frameBytes)
				copy(frame, p.pcmBuf[:p.frameBytes])
				p.pcmBuf = p.pcmBuf[p.frameBytes:]
			} else {
				frame = make([]byte, p.frameBytes)
				copy(frame, p.pcmBuf)
				p.pcmBuf = nil
			}
			p.mu.Unlock()

			wait := time.Until(nextDeadline)
			if wait > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(wait):
				}
			}
			nextDeadline = nextDeadline.Add(p.frameDur)

			_ = p.s.emitOutboundPCMFrame(frame)
			if forceTail {
				return
			}
		}
	}

	for {
		if ctx.Err() != nil {
			flush(true)
			return
		}
		p.mu.Lock()
		ready := len(p.pcmBuf) >= p.frameBytes
		p.mu.Unlock()
		if ready {
			flush(false)
			continue
		}
		select {
		case <-ctx.Done():
			flush(true)
			return
		case <-time.After(2 * time.Millisecond):
		}
	}
}
