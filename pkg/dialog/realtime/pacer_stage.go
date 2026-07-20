// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package realtime

import (
	"context"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/pipeline"
	"github.com/LingByte/lingllm/media"
)

// pacerStage solves a realtime-specific pathology: multimodal
// providers (Qwen-Omni, GPT-4o realtime, …) emit synthesised audio
// in irregular bursts — sometimes a 50 ms chunk, sometimes a 500 ms
// chunk, sometimes silence for 200 ms then five chunks back-to-back.
// The media RTP transport, by contrast, expects exactly one 20 ms
// frame per RTP packet; if it gets handed a 320 ms blob in a single
// SendOutputPCM call, it ships it as one oversize datagram with the
// codec's 20 ms-advanced timestamp, and the media peer plays the audio
// back as a burst with scrambled timing. Symptom: "一下子蹦出来很多
// 字" — the AI's reply pops out compressed in time.
//
// pacerStage smooths this out by:
//
//  1. Buffering inbound KindPCM frames into a contiguous byte slice
//     keyed off the (configurable) sample rate / bytes-per-frame.
//  2. A ticker fires every 20 ms; on each tick we slice exactly one
//     20 ms frame off the buffer and emit it. Underruns (buffer too
//     short) are skipped — we deliberately do NOT zero-fill, since
//     synthesised silence over PCM16LE produces "电流音" on cheap
//     voice clients (see voicedialog/tts_segmenter.go for the legacy
//     path's identical reasoning).
//  3. A configurable jitter prebuffer holds back the first frame
//     until N frames-worth of audio has accumulated, so an early
//     slow chunk doesn't cause an immediate underrun. Once started,
//     underrun is a tolerated transient (skip the tick, keep
//     playing the next available frame).
//
// Non-PCM frames pass through unchanged with no pacing — text /
// barge-in / turn-end signals must reach downstream observers (eg.
// persistStage) at their original arrival times so latency
// histograms stay accurate.
//
// The stage is direction-agnostic: callers compose it on whichever
// channel (input vs output) they need pacing on. In practice
// realtime.Engine inserts pacerStage between the agentStage's
// output and the rest of the pipeline so the AI-side PCM is paced
// before reaching the MediaPort.
type pacerStage struct {
	cfg PacerConfig

	// Test hooks — production wiring uses the defaults.
	nowFn   func() time.Time
	tickerF func(d time.Duration) (<-chan time.Time, func())
}

// PacerConfig defines a pacerStage's framing parameters. All fields
// are validated at construction; zero values get sensible defaults
// so callers can pass an empty struct for "16 kHz, 20 ms frames, 60
// ms prebuffer" — the modal media bridge configuration.
type PacerConfig struct {
	// SampleRate is the PCM rate in Hz at which the buffered audio
	// is interpreted. Defaults to 16000.
	SampleRate int

	// FrameMillis is the duration (ms) per emitted frame. Defaults
	// to 20.
	FrameMillis int

	// PrebufferFrames is the number of frames-worth of audio that
	// must accumulate before the first frame is emitted. Defaults
	// to 3 (60 ms at 20 ms framing). Lower values risk early
	// underrun; higher values increase latency.
	PrebufferFrames int
}

func (c PacerConfig) bytesPerFrame() int {
	rate := c.SampleRate
	if rate <= 0 {
		rate = 16000
	}
	frameMs := c.FrameMillis
	if frameMs <= 0 {
		frameMs = 20
	}
	// PCM16LE: 2 bytes per sample.
	return rate * 2 * frameMs / 1000
}

func (c PacerConfig) frameDuration() time.Duration {
	frameMs := c.FrameMillis
	if frameMs <= 0 {
		frameMs = 20
	}
	return time.Duration(frameMs) * time.Millisecond
}

func (c PacerConfig) prebufferBytes() int {
	pre := c.PrebufferFrames
	if pre <= 0 {
		pre = 3
	}
	return c.bytesPerFrame() * pre
}

// newPacerStage builds a pacerStage with the supplied config. Pass
// an empty PacerConfig{} for the bridge defaults.
func newPacerStage(cfg PacerConfig) *pacerStage {
	return &pacerStage{cfg: cfg}
}

// Name implements pipeline.Stage.
func (*pacerStage) Name() string { return "pcm_pacer" }

// Run implements pipeline.Stage. The loop has three concurrent
// concerns:
//
//   - read inbound frames; KindPCM accumulates into the buffer,
//     other frames pass through immediately;
//   - tick a 20 ms timer to slice frames off the buffer;
//   - honour ctx cancellation / in-channel close.
//
// out is closed before return per the pipeline.Stage contract.
func (s *pacerStage) Run(
	ctx context.Context,
	in <-chan pipeline.Frame,
	out chan<- pipeline.Frame,
	lg engine.Logger,
) error {
	defer close(out)
	if lg == nil {
		lg = engine.NopLogger{}
	}

	bytesPerFrame := s.cfg.bytesPerFrame()
	prebuf := s.cfg.prebufferBytes()
	frameDur := s.cfg.frameDuration()
	if bytesPerFrame <= 0 || frameDur <= 0 {
		// Defensive; bytesPerFrame uses 16k/20ms defaults so 0
		// is only reachable with a deliberately-pathological
		// caller config.
		return nil
	}

	var (
		bufMu   sync.Mutex
		buf     []byte
		started bool
	)

	tickC, stopTicker := s.startTicker(frameDur)
	defer stopTicker()

	emit := func(f pipeline.Frame) bool {
		select {
		case out <- f:
			return true
		case <-ctx.Done():
			return false
		}
	}

	// Pre-emptive in-channel drain: when the select would otherwise
	// pick tickC over an immediately-available in frame, the buffer
	// stays empty for that tick and the producer's PCM frame waits
	// in the channel until the next iteration. Tests that send a
	// frame followed by an immediate tick depend on the buffer
	// being populated first; production has the same property
	// (PCM bursts arrive faster than 20ms ticks) so honouring
	// inbound frames first is also more correct semantically.
	drainInbound := func() bool {
		for {
			select {
			case f, ok := <-in:
				if !ok {
					s.drain(ctx, &bufMu, &buf, bytesPerFrame, emit)
					return false
				}
				if f.Kind == pipeline.KindPCM {
					if data := normalizePCMRate(f.PCM.Data, f.PCM.SampleRate, s.cfg.SampleRate); len(data) > 0 {
						bufMu.Lock()
						buf = append(buf, data...)
						bufMu.Unlock()
					}
					continue
				}
				if f.Kind == pipeline.KindBargeIn {
					bufMu.Lock()
					buf = buf[:0]
					started = false
					bufMu.Unlock()
				}
				if !emit(f) {
					return false
				}
			default:
				return true
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case f, ok := <-in:
			if !ok {
				// Drain any remaining whole frames so a final
				// audio chunk doesn't get truncated when
				// upstream closes mid-buffer. Bounded by the
				// buffer size so no risk of livelock.
				s.drain(ctx, &bufMu, &buf, bytesPerFrame, emit)
				return nil
			}
			if f.Kind == pipeline.KindPCM {
				if data := normalizePCMRate(f.PCM.Data, f.PCM.SampleRate, s.cfg.SampleRate); len(data) > 0 {
					bufMu.Lock()
					buf = append(buf, data...)
					bufMu.Unlock()
				}
				continue
			}
			if f.Kind == pipeline.KindBargeIn {
				// Drop queued AI audio so the user does not hear leftover TTS
				// after interrupting (parity with sipattach DrainOutputs).
				bufMu.Lock()
				buf = buf[:0]
				started = false
				bufMu.Unlock()
			}
			if !emit(f) {
				return ctx.Err()
			}

		case <-tickC:
			if !drainInbound() {
				return nil
			}
			bufMu.Lock()
			if !started {
				if len(buf) < prebuf {
					bufMu.Unlock()
					continue
				}
				started = true
			}
			if len(buf) < bytesPerFrame {
				bufMu.Unlock()
				continue
			}
			frame := make([]byte, bytesPerFrame)
			copy(frame, buf[:bytesPerFrame])
			// Pop. The slice grows bounded by the agent's max
			// in-flight audio (~600 ms) and shrinks as we tick;
			// the append-shift below avoids unbounded growth of
			// the underlying array.
			buf = append(buf[:0], buf[bytesPerFrame:]...)
			bufMu.Unlock()
			if !emit(pipeline.Frame{
				Kind: pipeline.KindPCM,
				PCM: engine.PCMFrame{
					Data:        frame,
					SampleRate:  s.cfg.SampleRate,
					Synthesized: true,
				},
				EmittedAt: s.now(),
			}) {
				return ctx.Err()
			}
		}
	}
}

// drain emits whole frames remaining in buf. Partial trailing bytes
// are dropped (no zero-padding for the same reason underruns aren't
// padded). Stops on ctx-Done.
func (s *pacerStage) drain(
	ctx context.Context,
	bufMu *sync.Mutex,
	buf *[]byte,
	bytesPerFrame int,
	emit func(pipeline.Frame) bool,
) {
	for {
		bufMu.Lock()
		if len(*buf) < bytesPerFrame {
			bufMu.Unlock()
			return
		}
		frame := make([]byte, bytesPerFrame)
		copy(frame, (*buf)[:bytesPerFrame])
		*buf = append((*buf)[:0], (*buf)[bytesPerFrame:]...)
		bufMu.Unlock()
		if ctx.Err() != nil {
			return
		}
		emit(pipeline.Frame{
			Kind: pipeline.KindPCM,
			PCM: engine.PCMFrame{
				Data:        frame,
				SampleRate:  s.cfg.SampleRate,
				Synthesized: true,
			},
			EmittedAt: s.now(),
		})
	}
}

func (s *pacerStage) now() time.Time {
	if s.nowFn != nil {
		return s.nowFn()
	}
	return time.Now()
}

func (s *pacerStage) startTicker(d time.Duration) (<-chan time.Time, func()) {
	if s.tickerF != nil {
		return s.tickerF(d)
	}
	t := time.NewTicker(d)
	return t.C, t.Stop
}

func normalizePCMRate(data []byte, inRate, outRate int) []byte {
	if len(data) == 0 {
		return nil
	}
	if outRate <= 0 {
		outRate = 16000
	}
	if inRate <= 0 {
		inRate = outRate
	}
	if inRate == outRate {
		return data
	}
	out, err := media.ResamplePCM(data, inRate, outRate)
	if err != nil || len(out) == 0 {
		return data
	}
	return out
}

// NewPacerStage is the exported entry point so voice wiring (or
// tests outside the package) can instantiate the stage without
// referencing the internal type.
func NewPacerStage(cfg PacerConfig) pipeline.Stage {
	return newPacerStage(cfg)
}
