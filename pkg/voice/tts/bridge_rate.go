package tts

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"context"
	"fmt"

	"github.com/LingByte/lingllm/media"
)

// BridgeRateService wraps a streaming TTS service and resamples its PCM output to
// bridgeRate before the Pipeline frames it. Per-frame ResamplePCM on 20ms chunks
// resets converter phase at every boundary and produces a broadband click train
// ("电流音"); this wrapper uses chunk-safe streaming policies instead:
//
//   - cloudRate == bridgeRate: pass-through
//   - cloudRate == 2 * bridgeRate: streaming pair-average decimation (16k→8k)
//   - cloudRate == 3 * bridgeRate: streaming triple-average (24k→8k)
//   - cloudRate*2 == bridgeRate*3: streaming 3:2 (24k→16k)
//   - otherwise: stateful streaming resampler (carry leftover samples)
type BridgeRateService struct {
	Inner      Service
	CloudRate  int
	BridgeRate int
	halfRem    []byte // 2:1 / N:1 / 3:2 decimation carry
}

// BridgeRate wraps inner when cloud and bridge rates differ.
func BridgeRate(inner Service, cloudRate, bridgeRate int) Service {
	if inner == nil {
		return nil
	}
	if cloudRate <= 0 {
		cloudRate = bridgeRate
	}
	if bridgeRate <= 0 || cloudRate == bridgeRate {
		return inner
	}
	return &BridgeRateService{Inner: inner, CloudRate: cloudRate, BridgeRate: bridgeRate}
}

func (b *BridgeRateService) SynthesizeStream(
	ctx context.Context,
	text string,
	onPCMChunk func([]byte) error,
) error {
	if b == nil || b.Inner == nil {
		return fmt.Errorf("voice/tts: nil BridgeRateService")
	}
	if onPCMChunk == nil {
		return fmt.Errorf("voice/tts: nil onPCMChunk")
	}
	cloud := b.CloudRate
	bridge := b.BridgeRate
	if cloud <= 0 {
		cloud = bridge
	}
	if bridge <= 0 || cloud == bridge {
		return b.Inner.SynthesizeStream(ctx, text, onPCMChunk)
	}
	if cloud == 2*bridge {
		return b.stream2to1(ctx, text, onPCMChunk)
	}
	if cloud == 3*bridge {
		return b.streamNto1(ctx, text, onPCMChunk, 3)
	}
	if cloud*2 == bridge*3 {
		return b.stream3to2(ctx, text, onPCMChunk)
	}
	return b.streamResample(ctx, text, onPCMChunk)
}

func (b *BridgeRateService) stream2to1(
	ctx context.Context,
	text string,
	onPCMChunk func([]byte) error,
) error {
	return b.Inner.SynthesizeStream(ctx, text, func(pcm []byte) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		out := b.streamingHalveDecimate16to8(pcm)
		if len(out) == 0 {
			return nil
		}
		return onPCMChunk(out)
	})
}

func (b *BridgeRateService) streamNto1(
	ctx context.Context,
	text string,
	onPCMChunk func([]byte) error,
	n int,
) error {
	if n < 2 {
		return b.Inner.SynthesizeStream(ctx, text, onPCMChunk)
	}
	groupBytes := n * 2
	return b.Inner.SynthesizeStream(ctx, text, func(pcm []byte) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		src := pcm
		if len(b.halfRem) > 0 {
			merged := make([]byte, 0, len(b.halfRem)+len(pcm))
			merged = append(merged, b.halfRem...)
			merged = append(merged, pcm...)
			src = merged
			b.halfRem = nil
		}
		if len(src)&1 != 0 {
			b.halfRem = append(b.halfRem[:0], src[len(src)-1])
			src = src[:len(src)-1]
		}
		groups := len(src) / groupBytes
		tail := len(src) - groups*groupBytes
		if tail > 0 {
			b.halfRem = append(b.halfRem[:0], src[groups*groupBytes:]...)
		}
		if groups == 0 {
			return nil
		}
		out := make([]byte, groups*2)
		for i := 0; i < groups; i++ {
			var sum int32
			base := i * groupBytes
			for j := 0; j < n; j++ {
				off := base + j*2
				sum += int32(int16(src[off]) | int16(src[off+1])<<8)
			}
			avg := int16(sum / int32(n))
			out[i*2] = byte(avg)
			out[i*2+1] = byte(avg >> 8)
		}
		return onPCMChunk(out)
	})
}

// stream3to2 converts 24k→16k by emitting 2 samples for every 3 input samples.
func (b *BridgeRateService) stream3to2(
	ctx context.Context,
	text string,
	onPCMChunk func([]byte) error,
) error {
	const inGroup = 6  // 3 samples
	const outGroup = 4 // 2 samples
	return b.Inner.SynthesizeStream(ctx, text, func(pcm []byte) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		src := pcm
		if len(b.halfRem) > 0 {
			merged := make([]byte, 0, len(b.halfRem)+len(pcm))
			merged = append(merged, b.halfRem...)
			merged = append(merged, pcm...)
			src = merged
			b.halfRem = nil
		}
		if len(src)&1 != 0 {
			b.halfRem = append(b.halfRem[:0], src[len(src)-1])
			src = src[:len(src)-1]
		}
		groups := len(src) / inGroup
		tail := len(src) - groups*inGroup
		if tail > 0 {
			b.halfRem = append(b.halfRem[:0], src[groups*inGroup:]...)
		}
		if groups == 0 {
			return nil
		}
		out := make([]byte, groups*outGroup)
		for i := 0; i < groups; i++ {
			base := i * inGroup
			s0 := int32(int16(src[base]) | int16(src[base+1])<<8)
			s1 := int32(int16(src[base+2]) | int16(src[base+3])<<8)
			s2 := int32(int16(src[base+4]) | int16(src[base+5])<<8)
			// Linear mix: out0 ≈ s0, out1 ≈ (s1+s2)/2 — continuous enough for speech.
			o0 := int16(s0)
			o1 := int16((s1 + s2) >> 1)
			outOff := i * outGroup
			out[outOff] = byte(o0)
			out[outOff+1] = byte(o0 >> 8)
			out[outOff+2] = byte(o1)
			out[outOff+3] = byte(o1 >> 8)
		}
		return onPCMChunk(out)
	})
}

// streamResample uses a stateful converter with leftover carry so arbitrary
// rate ratios still emit PCM incrementally (no wait-for-full-utterance).
func (b *BridgeRateService) streamResample(
	ctx context.Context,
	text string,
	onPCMChunk func([]byte) error,
) error {
	conv := media.DefaultResampler(b.CloudRate, b.BridgeRate)
	var pend []byte
	err := b.Inner.SynthesizeStream(ctx, text, func(pcm []byte) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if len(pcm) == 0 {
			return nil
		}
		src := pcm
		if len(pend) > 0 {
			merged := make([]byte, 0, len(pend)+len(pcm))
			merged = append(merged, pend...)
			merged = append(merged, pcm...)
			src = merged
			pend = nil
		}
		if len(src)&1 != 0 {
			pend = append(pend[:0], src[len(src)-1])
			src = src[:len(src)-1]
		}
		if len(src) == 0 {
			return nil
		}
		// Keep a few trailing samples so the next Write sees continuity at
		// the boundary (cubic needs ±2 neighbors). Re-feed them next chunk.
		const keep = 8 // 4 samples
		if len(src) <= keep {
			pend = append(pend[:0], src...)
			return nil
		}
		process := src[:len(src)-keep]
		pend = append(pend[:0], src[len(src)-keep:]...)
		if _, err := conv.Write(process); err != nil {
			return err
		}
		out := conv.Samples()
		if len(out) == 0 {
			return nil
		}
		return onPCMChunk(out)
	})
	if err != nil {
		return err
	}
	if len(pend) > 0 {
		if len(pend)&1 != 0 {
			pend = pend[:len(pend)-1]
		}
		if len(pend) > 0 {
			_, _ = conv.Write(pend)
		}
	}
	_ = conv.Close()
	if out := conv.Samples(); len(out) > 0 {
		return onPCMChunk(out)
	}
	return nil
}

// streamingHalveDecimate16to8 averages consecutive int16 LE sample pairs with
// carry across chunks so boundaries never split a pair.
func (b *BridgeRateService) streamingHalveDecimate16to8(pcm []byte) []byte {
	src := pcm
	if len(b.halfRem) > 0 {
		merged := make([]byte, 0, len(b.halfRem)+len(pcm))
		merged = append(merged, b.halfRem...)
		merged = append(merged, pcm...)
		src = merged
		b.halfRem = nil
	}
	// Each output sample = avg of 2 input samples = 4 input bytes → 2 output bytes.
	pairs := len(src) / 4
	tail := len(src) - pairs*4
	if tail > 0 {
		// Save trailing 1-3 bytes for next call (incomplete sample-pair).
		b.halfRem = append(b.halfRem[:0], src[pairs*4:]...)
	}
	if pairs == 0 {
		return nil
	}
	out := make([]byte, pairs*2)
	for i := 0; i < pairs; i++ {
		s1 := int32(int16(src[i*4]) | int16(src[i*4+1])<<8)
		s2 := int32(int16(src[i*4+2]) | int16(src[i*4+3])<<8)
		avg := int16((s1 + s2) >> 1)
		out[i*2] = byte(avg)
		out[i*2+1] = byte(avg >> 8)
	}
	return out
}
