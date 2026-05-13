package asr

// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/LingByte/SoulNexus/pkg/recognizer"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/voice/vad"
	"go.uber.org/zap"
)

// TextCallback is fired for each partial / final hypothesis.
//
// isFinal=true means the recognizer marked the utterance finished (sentence
// closed). Non-final hypotheses represent the current cumulative best guess.
type TextCallback func(text string, isFinal bool)

// ErrorCallback is fired whenever the underlying recognizer reports an error.
// fatal=true means the session is dead and the caller should stop feeding audio.
type ErrorCallback func(err error, fatal bool)

// Options configures a Pipeline.
type Options struct {
	// ASR is the underlying recognizer. Required.
	ASR recognizer.TranscribeService

	// InputSampleRate is the sample rate of PCM16 bytes handed to ProcessPCM.
	// 0 means "same as SampleRate" (no resample).
	InputSampleRate int

	// SampleRate is the sample rate expected by the recognizer. Required.
	// Typical values: 8000 (phone 8k models) or 16000.
	SampleRate int

	// Channels is always 1 in this package; reserved for future stereo ASR.
	Channels int

	// DialogID is passed to TranscribeService.ConnAndReceive. Empty = auto.
	DialogID string

	// MinFeedBytes caps how small a single SendAudioBytes chunk may be.
	// 0 → 20 ms @ SampleRate (e.g. 640 bytes at 16 kHz PCM16).
	MinFeedBytes int

	// Logger is optional; nil → zap.NewNop.
	Logger *zap.Logger
}

// Pipeline streams PCM16 into a recognizer and surfaces transcripts via callbacks.
//
//   pipe, _ := asr.New(asr.Options{ASR: svc, SampleRate: 16000})
//   pipe.SetTextCallback(func(text string, isFinal bool) { ... })
//   go pipe.ProcessPCM(ctx, pcm)
//   defer pipe.Close()
type Pipeline struct {
	opt Options

	textCb    atomic.Value // TextCallback
	errCb     atomic.Value // ErrorCallback
	closed    atomic.Bool
	connected atomic.Bool

	minFeed int

	buf     []byte
	bufMu   sync.Mutex
	feedMu  sync.Mutex
	connect sync.Once

	log *zap.Logger

	// Barge-in is wired AFTER Pipeline construction via
	// SetBargeInDetector so the ASR Options API stays compact. When
	// all three fields are set, every incoming PCM frame is checked
	// for user-started-talking-during-TTS and onBargeIn fires on
	// match. Zero-value (all nil) disables the feature entirely and
	// costs exactly one pointer load per frame.
	bargeDet        *vad.Detector
	bargeSynthFn    func() bool
	bargeOnFire     func()
	bargeMu         sync.RWMutex

	// Denoise hook. nil = passthrough (zero overhead). When set, the
	// hook runs ONCE per ProcessPCM call on the raw input PCM, before
	// any resample → ASR feed. Rate compatibility is the caller's
	// responsibility: RNNoise expects 48 kHz, so for the WebRTC opus
	// path (InputSampleRate=48000) it is a perfect fit. For SIP 8k
	// audio you would typically NOT wire a denoiser — the codec has
	// already lost the high-frequency band the model was trained on.
	denoise   Denoiser
	denoiseMu sync.RWMutex
}

// Denoiser is the minimal interface ASR consumes for noise suppression.
// pkg/audio/rnnoise.Denoiser satisfies it directly. Callers can also
// plug in their own implementations (WebRTC AEC3, custom DSP, ...).
//
// Process must NOT mutate the input slice and must return a slice the
// pipeline is free to keep. Implementations that have nothing to do
// (rate mismatch, library unavailable) should return the input
// unchanged so the call is effectively a passthrough.
type Denoiser interface {
	Process(pcm []byte) []byte
}

// New constructs a Pipeline. It does not connect to the recognizer until the
// first ProcessPCM call (lazy), which keeps Options cheap and avoids burning
// credits on idle sessions.
func New(opt Options) (*Pipeline, error) {
	if opt.ASR == nil {
		return nil, fmt.Errorf("voice/asr: nil recognizer")
	}
	if opt.SampleRate <= 0 {
		return nil, fmt.Errorf("voice/asr: SampleRate must be >0")
	}
	if opt.Channels == 0 {
		opt.Channels = 1
	}
	if opt.InputSampleRate <= 0 {
		opt.InputSampleRate = opt.SampleRate
	}
	minFeed := opt.MinFeedBytes
	if minFeed <= 0 {
		// 20 ms @ SampleRate, PCM16 mono.
		minFeed = (opt.SampleRate / 50) * 2
		if minFeed <= 0 {
			minFeed = 320
		}
	}
	p := &Pipeline{
		opt:     opt,
		minFeed: minFeed,
		log:     opt.Logger,
	}
	if p.log == nil {
		p.log = zap.NewNop()
	}
	return p, nil
}

// SetTextCallback registers the transcript sink. Safe to swap at runtime.
func (p *Pipeline) SetTextCallback(cb TextCallback) {
	if cb == nil {
		cb = func(string, bool) {}
	}
	p.textCb.Store(cb)
}

// SetErrorCallback registers the error sink. Safe to swap at runtime.
func (p *Pipeline) SetErrorCallback(cb ErrorCallback) {
	if cb == nil {
		cb = func(error, bool) {}
	}
	p.errCb.Store(cb)
}

// Vendor returns the underlying recognizer vendor label.
func (p *Pipeline) Vendor() string {
	if p == nil || p.opt.ASR == nil {
		return ""
	}
	return p.opt.ASR.Vendor()
}

// ProcessPCM feeds one chunk of PCM16 samples. It is safe to call from multiple
// goroutines (serialized internally). Returns immediately on ctx cancel.
//
// Rules:
//   - data must be PCM16 little-endian mono at opt.InputSampleRate.
//   - If InputSampleRate != SampleRate we resample via pkg/media.ResamplePCM.
//   - Sub-minFeed chunks are buffered to avoid frame-level overhead on some
//     providers (QCloud, Volcengine); the tail is flushed on Close().
//
// The recognizer is lazily connected on the first call.
func (p *Pipeline) ProcessPCM(ctx context.Context, data []byte) error {
	if p == nil || p.closed.Load() {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if len(data) == 0 {
		return nil
	}

	// Lazy connect.
	if !p.connected.Load() {
		if err := p.connectOnce(); err != nil {
			return err
		}
	}

	// Denoise on the raw input PCM — before resample. Doing it here
	// (rather than after resample to ASR rate) means the denoiser
	// sees the highest-fidelity audio available. RNNoise specifically
	// is a 48 kHz model so the WebRTC opus path benefits most.
	in := data
	if dn := p.denoiseHook(); dn != nil {
		in = dn.Process(in)
	}

	pcm := in
	if p.opt.InputSampleRate != p.opt.SampleRate {
		out, err := media.ResamplePCM(in, p.opt.InputSampleRate, p.opt.SampleRate)
		if err != nil {
			p.log.Debug("voice/asr resample failed", zap.Error(err))
			return nil
		}
		pcm = out
	}

	// Barge-in check — runs on the resampled PCM (same rate the
	// ASR sees) so energy numbers are comparable across transports.
	// Cheap fast-path: one atomic-style pointer read when disabled.
	if det, synthFn, onFire := p.bargeInHooks(); det != nil && synthFn != nil && onFire != nil {
		if det.CheckBargeIn(pcm, synthFn()) {
			// Fire in a goroutine because onFire typically calls
			// TTS.Interrupt() which takes a mutex that may serialize
			// with the TTS pipeline's own Speak loop — don't block
			// the ASR feed path.
			go onFire()
		}
	}

	// Coalesce into >= minFeed chunks.
	p.bufMu.Lock()
	p.buf = append(p.buf, pcm...)
	var toSend []byte
	if len(p.buf) >= p.minFeed {
		toSend = append([]byte(nil), p.buf...)
		p.buf = p.buf[:0]
	}
	p.bufMu.Unlock()
	if toSend == nil {
		return nil
	}

	// SendAudioBytes may block on the websocket writer; serialize to preserve order.
	p.feedMu.Lock()
	err := p.opt.ASR.SendAudioBytes(toSend)
	p.feedMu.Unlock()
	if err != nil {
		p.emitError(err, false)
		return err
	}
	return nil
}

// Flush pushes whatever bytes are still buffered without closing the session.
// Useful between utterances or before Close().
func (p *Pipeline) Flush() error {
	if p == nil {
		return nil
	}
	p.bufMu.Lock()
	pending := p.buf
	p.buf = nil
	p.bufMu.Unlock()
	if len(pending) == 0 {
		return nil
	}
	p.feedMu.Lock()
	err := p.opt.ASR.SendAudioBytes(pending)
	p.feedMu.Unlock()
	return err
}

// Close stops the underlying recognizer session. Idempotent.
func (p *Pipeline) Close() error {
	if p == nil {
		return nil
	}
	if p.closed.Swap(true) {
		return nil
	}
	_ = p.Flush()
	if p.connected.Load() {
		_ = p.opt.ASR.SendEnd()
		_ = p.opt.ASR.StopConn()
	}
	return nil
}

// Restart tears down the current recognizer session and reconnects. The
// existing text/error callbacks are preserved.
func (p *Pipeline) Restart(ctx context.Context) error {
	if p == nil || p.closed.Load() {
		return errors.New("voice/asr: pipeline closed")
	}
	if p.connected.Load() {
		_ = p.opt.ASR.SendEnd()
		_ = p.opt.ASR.StopConn()
	}
	p.connected.Store(false)
	p.connect = sync.Once{}
	return p.connectOnce()
}

// ----- internal -----

func (p *Pipeline) connectOnce() (err error) {
	p.connect.Do(func() {
		// Hook the vendor's result stream into our callbacks before connecting
		// to avoid missing early partials.
		p.opt.ASR.Init(
			func(text string, isFinal bool, _ time.Duration, _ string) {
				p.emitText(text, isFinal)
			},
			func(e error, fatal bool) {
				p.emitError(e, fatal)
			},
		)
		dialog := strings.TrimSpace(p.opt.DialogID)
		err = p.opt.ASR.ConnAndReceive(dialog)
		if err == nil {
			p.connected.Store(true)
		}
	})
	return err
}

// SetBargeInDetector wires an energy-based VAD + "is TTS playing"
// predicate + "what to do on barge-in" callback into the ASR pipeline.
// When all three are non-nil, every successfully resampled PCM frame
// in ProcessPCM is checked for barge-in; on fire, onFire runs in a
// fresh goroutine. Pass (nil, nil, nil) to disable at runtime.
//
// Typical wiring at the gateway client:
//
//	det := vad.NewDetector()
//	asr.SetBargeInDetector(det, tts.IsPlaying, func() {
//	    tts.Interrupt()
//	    _ = gw.sendEvent(Event{Type: EvTTSInterrupt, CallID: callID})
//	})
func (p *Pipeline) SetBargeInDetector(det *vad.Detector, synthPlayingFn func() bool, onFire func()) {
	if p == nil {
		return
	}
	p.bargeMu.Lock()
	p.bargeDet = det
	p.bargeSynthFn = synthPlayingFn
	p.bargeOnFire = onFire
	p.bargeMu.Unlock()
}

// bargeInHooks returns a consistent snapshot of the three barge-in
// parameters. We take an RLock rather than three atomic loads so the
// three-tuple is observed coherently (no tearing on race with Set*).
func (p *Pipeline) bargeInHooks() (*vad.Detector, func() bool, func()) {
	p.bargeMu.RLock()
	defer p.bargeMu.RUnlock()
	return p.bargeDet, p.bargeSynthFn, p.bargeOnFire
}

// SetDenoiser wires (or clears, with nil) the noise-suppression hook
// the pipeline runs before resampling each ProcessPCM batch. Calling
// it after the pipeline has started is safe — subsequent batches
// pick up the new value via denoiseHook's RWMutex snapshot.
func (p *Pipeline) SetDenoiser(d Denoiser) {
	if p == nil {
		return
	}
	p.denoiseMu.Lock()
	p.denoise = d
	p.denoiseMu.Unlock()
}

func (p *Pipeline) denoiseHook() Denoiser {
	p.denoiseMu.RLock()
	defer p.denoiseMu.RUnlock()
	return p.denoise
}

func (p *Pipeline) emitText(text string, isFinal bool) {
	cb, _ := p.textCb.Load().(TextCallback)
	if cb == nil {
		return
	}
	cb(text, isFinal)
}

func (p *Pipeline) emitError(err error, fatal bool) {
	if err == nil {
		return
	}
	cb, _ := p.errCb.Load().(ErrorCallback)
	if cb != nil {
		cb(err, fatal)
	}
	p.log.Warn("voice/asr error",
		zap.String("vendor", p.Vendor()),
		zap.Bool("fatal", fatal),
		zap.Error(err),
	)
}
