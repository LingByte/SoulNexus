package synthesizer

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// StreamingSynthesizer — provider-agnostic streaming TTS for the SIP plane.
//
// Why this exists:
// ----------------
// The SIP voice pipeline (pkg/sip/voicedialog/gateway_media.go and
// pkg/sip/conversation/voice.go) used to construct `synthesizer.QCloudService`
// directly and adapt its WebSocket variant into a `SynthesizeStream(ctx,
// text, cb)` shape consumed by `pkg/voice/tts`. Adding any new TTS provider
// (Aliyun Qwen-TTS realtime, ElevenLabs, Volcengine, …) required forking
// those wiring sites.
//
// `NewStreamingSynthesizerFromCredential` solves that by:
//  1. delegating provider routing to the existing factory
//     `NewSynthesisServiceFromCredential` (single source of truth — same one
//     the offline `WithSynthesis` player uses), then
//  2. picking the fastest streaming path for the underlying service:
//       * QCloud → SpeechWsSynthesizer (binary PCM frames)
//       * Aliyun Qwen-TTS realtime → DashScope WS, base64 PCM deltas
//       * any other provider → wrap `SynthesisService.Synthesize` so the
//         single response payload is delivered as one (or a few) chunks via
//         the same callback contract. Strictly correct, but loses the
//         streaming first-byte latency win until a native fast path is
//         added for that provider.
//
// The returned svc also reports the actual output sample-rate the cloud
// will emit — callers configure `siptts.New(SampleRate: …)` from this so
// the downstream resampler chain produces matching PCM for the call leg.

import (
	"context"
	"fmt"
)

// StreamingSynthesizer is the minimal contract `pkg/voice/tts.Service` and
// SIP segmenters consume. PCM16LE little-endian mono samples are forwarded
// to `callback` as they arrive at `Service.Format().SampleRate`.
type StreamingSynthesizer interface {
	SynthesizeStream(ctx context.Context, text string, callback func(pcm []byte) error) error
}

// StreamingHandle bundles the resolved streaming TTS together with the
// underlying SynthesisService (kept so callers can introspect Provider() /
// Format() / Close() without duplicating the type-assert).
type StreamingHandle struct {
	Service    SynthesisService
	Stream     StreamingSynthesizer
	SampleRate int
}

// NewStreamingFromCredential builds a StreamingHandle from a credential
// blob (the same `provider + per-vendor fields` map persisted by the
// tenant AI config UI). The returned handle is safe to use concurrently as
// long as `Service.Synthesize` / `Stream.SynthesizeStream` are themselves
// safe — most vendor SDKs are.
func NewStreamingFromCredential(cfg TTSCredentialConfig) (*StreamingHandle, error) {
	svc, err := NewSynthesisServiceFromCredential(cfg)
	if err != nil {
		return nil, err
	}
	stream := AsStreaming(svc)
	if stream == nil {
		return nil, fmt.Errorf("synthesizer: no streaming adapter for provider %q", svc.Provider())
	}
	return &StreamingHandle{
		Service:    svc,
		Stream:     stream,
		SampleRate: svc.Format().SampleRate,
	}, nil
}

// AsStreaming returns a StreamingSynthesizer view of svc. If svc natively
// satisfies the interface (e.g. QCloudService, AliyunService) it is
// returned as-is; otherwise a fallback adapter is used that re-uses the
// generic `Synthesize` entrypoint and forwards `OnMessage` payloads.
func AsStreaming(svc SynthesisService) StreamingSynthesizer {
	if svc == nil {
		return nil
	}
	if s, ok := svc.(StreamingSynthesizer); ok {
		return s
	}
	return &synthesisFallbackStream{svc: svc}
}

// synthesisFallbackStream adapts a non-streaming SynthesisService into the
// streaming callback contract. `Synthesize` typically delivers the whole
// utterance in a single OnMessage call; we forward each chunk to the
// callback verbatim and rely on the caller's framer to chunk on its own.
type synthesisFallbackStream struct {
	svc SynthesisService
}

func (f *synthesisFallbackStream) SynthesizeStream(
	ctx context.Context,
	text string,
	callback func(pcm []byte) error,
) error {
	if f == nil || f.svc == nil {
		return fmt.Errorf("synthesizer: nil service")
	}
	if callback == nil {
		return fmt.Errorf("synthesizer: nil callback")
	}
	h := &callbackHandler{cb: callback}
	if err := f.svc.Synthesize(ctx, h, text); err != nil {
		return err
	}
	return h.err
}

type callbackHandler struct {
	cb  func([]byte) error
	err error
}

func (h *callbackHandler) OnMessage(data []byte) {
	if h == nil || h.cb == nil || len(data) == 0 || h.err != nil {
		return
	}
	if err := h.cb(data); err != nil {
		h.err = err
	}
}

func (h *callbackHandler) OnTimestamp(_ SentenceTimestamp) {}
