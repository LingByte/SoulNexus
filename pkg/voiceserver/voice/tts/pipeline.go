package tts

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
	"go.uber.org/zap"
)

// FrameSink receives a single PCM16 frame ready to be placed on the wire
// (e.g. pushed into a MediaSession output queue or an RTP transport).
//
// frame is PCM16LE mono at Config.OutputSampleRate, with byte length
// corresponding to Config.FrameDuration.
type FrameSink func(frame []byte) error

// Config configures a streaming TTS Pipeline.
type Config struct {
	// Service is the streaming TTS engine. Required.
	Service Service

	// InputSampleRate is the sample rate emitted by Service.SynthesizeStream.
	// Required. Example: 16000.
	InputSampleRate int

	// OutputSampleRate is the sample rate the Sink expects. 0 = same as input.
	OutputSampleRate int

	// Channels = 1 (stereo TTS is out of scope here).
	Channels int

	// FrameDuration is the wire frame size. Default 20 ms.
	FrameDuration time.Duration

	// PaceRealtime, if true, sleeps between frames so the Sink receives audio
	// at wall-clock rate (required for RTP). If false, frames are flushed as
	// fast as the Sink drains.
	PaceRealtime bool

	// Sink receives framed PCM16 at OutputSampleRate. Required.
	Sink FrameSink

	// Logger optional.
	Logger *zap.Logger
}

// Pipeline turns text → streaming PCM16 frames.
//
//   p, _ := tts.New(tts.Config{
//       Service: tts.FromSynthesisService(qcloudTTS),
//       InputSampleRate: 16000, OutputSampleRate: 8000,
//       FrameDuration: 20 * time.Millisecond, PaceRealtime: true,
//       Sink: func(frame []byte) error { ms.SendToOutput(...); return nil },
//   })
//   p.Start(ctx)
//   p.Speak("你好，请问有什么可以帮您？")
//   p.Interrupt()   // stop current playback (barge-in)
//   p.Stop()        // tear down
type Pipeline struct {
	cfg Config

	log *zap.Logger

	runCtx    context.Context
	runCancel context.CancelFunc

	frameBytes int

	// speakCtx is per-Speak; Interrupt cancels it.
	speakMu     sync.Mutex
	speakCtx    context.Context
	speakCancel context.CancelFunc

	started atomic.Bool
	playing atomic.Bool

	// firstFrameHook is fired exactly once, the first time a Sink call
	// emits PCM after the hook was armed. Used by the dialog gateway to
	// measure end-to-end "ASR final → first audible byte" latency: it
	// arms the hook before each Speak and reads the timestamp as soon as
	// real audio leaves the pipeline. Atomic-swap-and-clear semantics
	// guarantee the hook runs at most once per arming, even if Speak
	// emits hundreds of frames or two threads race.
	firstFrameHook atomic.Pointer[func()]
}

// New validates cfg and returns a ready-to-Start Pipeline.
func New(cfg Config) (*Pipeline, error) {
	if cfg.Service == nil {
		return nil, fmt.Errorf("voice/tts: nil Service")
	}
	if cfg.Sink == nil {
		return nil, fmt.Errorf("voice/tts: nil Sink")
	}
	if cfg.InputSampleRate <= 0 {
		return nil, fmt.Errorf("voice/tts: InputSampleRate must be >0")
	}
	if cfg.OutputSampleRate <= 0 {
		cfg.OutputSampleRate = cfg.InputSampleRate
	}
	if cfg.Channels == 0 {
		cfg.Channels = 1
	}
	if cfg.FrameDuration <= 0 {
		cfg.FrameDuration = 20 * time.Millisecond
	}
	samples := int(float64(cfg.OutputSampleRate) * cfg.FrameDuration.Seconds())
	frameBytes := samples * 2 // PCM16 mono
	if frameBytes < 2 {
		return nil, fmt.Errorf("voice/tts: frame too small (%d bytes)", frameBytes)
	}
	p := &Pipeline{
		cfg:        cfg,
		log:        cfg.Logger,
		frameBytes: frameBytes,
	}
	if p.log == nil {
		p.log = zap.NewNop()
	}
	return p, nil
}

// Start arms the pipeline with a parent context. Safe to call multiple times
// (subsequent calls replace the run context if Stop was called in between).
func (p *Pipeline) Start(parent context.Context) {
	if p == nil {
		return
	}
	if parent == nil {
		parent = context.Background()
	}
	p.speakMu.Lock()
	if p.runCancel != nil {
		p.runCancel()
	}
	p.runCtx, p.runCancel = context.WithCancel(parent)
	p.speakMu.Unlock()
	p.started.Store(true)
}

// Stop cancels any in-flight Speak and shuts the pipeline down. Idempotent.
func (p *Pipeline) Stop() {
	if p == nil {
		return
	}
	p.speakMu.Lock()
	if p.speakCancel != nil {
		p.speakCancel()
	}
	p.speakCancel = nil
	p.speakCtx = nil
	if p.runCancel != nil {
		p.runCancel()
	}
	p.runCancel = nil
	p.speakMu.Unlock()
	p.started.Store(false)
	p.playing.Store(false)
}

// Interrupt cancels the currently-speaking utterance (barge-in). Future Speak
// calls still work — the pipeline remains Started.
func (p *Pipeline) Interrupt() {
	if p == nil {
		return
	}
	p.speakMu.Lock()
	if p.speakCancel != nil {
		p.speakCancel()
	}
	p.speakCancel = nil
	p.speakCtx = nil
	p.speakMu.Unlock()
	p.playing.Store(false)
}

// IsPlaying reports whether a Speak call is currently streaming frames.
func (p *Pipeline) IsPlaying() bool { return p != nil && p.playing.Load() }

// ArmFirstFrameHook installs a one-shot callback that fires when the
// next successful Sink call ships PCM. The hook is consumed atomically
// (swap-and-clear) so it triggers at most once per arming. Calling
// ArmFirstFrameHook with a non-nil fn replaces any prior pending hook;
// passing nil disarms.
//
// Intended for end-to-end latency measurement around a Speak call:
//
//	p.ArmFirstFrameHook(func() { firstByte = time.Now() })
//	_ = p.Speak(text)
//	// firstByte now holds the wall-clock at which the first frame
//	// actually left the pipeline (or zero if Speak failed before any
//	// frame was produced).
//
// Safe to call before, during, or between Speak invocations.
func (p *Pipeline) ArmFirstFrameHook(fn func()) {
	if p == nil {
		return
	}
	if fn == nil {
		p.firstFrameHook.Store(nil)
		return
	}
	p.firstFrameHook.Store(&fn)
}

// Speak synthesizes text and streams frames synchronously. Returns when either
// Service returns, Interrupt/Stop is called, or the run context is cancelled.
func (p *Pipeline) Speak(text string) error {
	if p == nil {
		return fmt.Errorf("voice/tts: nil pipeline")
	}
	if !p.started.Load() {
		return fmt.Errorf("voice/tts: Start not called")
	}
	if text == "" {
		return nil
	}

	p.speakMu.Lock()
	if p.runCtx == nil || p.runCtx.Err() != nil {
		p.speakMu.Unlock()
		return fmt.Errorf("voice/tts: pipeline stopped")
	}
	ctx, cancel := context.WithCancel(p.runCtx)
	p.speakCtx = ctx
	p.speakCancel = cancel
	p.speakMu.Unlock()

	defer func() {
		p.speakMu.Lock()
		if p.speakCancel != nil {
			p.speakCancel()
		}
		p.speakCancel = nil
		p.speakCtx = nil
		p.speakMu.Unlock()
		p.playing.Store(false)
	}()

	p.playing.Store(true)

	// Pace wall clock across Service callbacks.
	var nextDeadline time.Time
	if p.cfg.PaceRealtime {
		nextDeadline = time.Now()
	}

	// One growing buffer per Speak call; we slice it into fixed frames.
	buf := make([]byte, 0, p.frameBytes*32)
	flush := func(forceTail bool) error {
		for {
			if len(buf) < p.frameBytes && !forceTail {
				return nil
			}
			if len(buf) == 0 {
				return nil
			}
			var frame []byte
			if len(buf) >= p.frameBytes {
				frame = make([]byte, p.frameBytes)
				copy(frame, buf[:p.frameBytes])
				buf = buf[p.frameBytes:]
			} else {
				// pad tail with silence to a full frame
				frame = make([]byte, p.frameBytes)
				copy(frame, buf)
				buf = buf[:0]
			}

			out := frame
			if p.cfg.InputSampleRate != p.cfg.OutputSampleRate {
				r, err := media.ResamplePCM(frame, p.cfg.InputSampleRate, p.cfg.OutputSampleRate)
				if err == nil && len(r) > 0 {
					out = r
				}
			}

			if p.cfg.PaceRealtime {
				wait := time.Until(nextDeadline)
				if wait > 0 {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(wait):
					}
				}
				nextDeadline = nextDeadline.Add(p.cfg.FrameDuration)
			} else if err := ctx.Err(); err != nil {
				return err
			}

			if err := p.cfg.Sink(out); err != nil {
				return err
			}
			// Fire the armed first-frame hook AFTER Sink succeeded so we
			// only stamp "audio actually shipped" — a Sink that errored
			// out shouldn't be counted as audible. Swap-and-clear so a
			// per-Speak arming can never trigger twice.
			if hookPtr := p.firstFrameHook.Swap(nil); hookPtr != nil {
				(*hookPtr)()
			}

			if forceTail && len(buf) == 0 {
				return nil
			}
		}
	}

	err := p.cfg.Service.SynthesizeStream(ctx, text, func(chunk []byte) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if len(chunk) == 0 {
			return nil
		}
		buf = append(buf, chunk...)
		return flush(false)
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return fmt.Errorf("voice/tts: synthesize: %w", err)
	}

	// Drain tail.
	if err := flush(true); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
