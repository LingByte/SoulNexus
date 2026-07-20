// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

//go:build lingcodec

package encoder

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/LingByte/lingllm/media"
)

func TestLECodecBackend(t *testing.T) {
	if !LECodecEnabled() {
		t.Fatal("expected lecodec backend")
	}
	if LECodecVersion() == "" {
		t.Fatal("empty version")
	}
	for _, name := range []string{CodecPCMU, CodecPCMA, CodecG722, CodecOPUS, CodecG729, CodecPCM} {
		if !HasCodec(name) {
			t.Errorf("missing %s", name)
		}
	}
}

func sineLE(n, rate int, hz float64, amp int16) []byte {
	out := make([]byte, n*2)
	for i := 0; i < n; i++ {
		s := int16(float64(amp) * math.Sin(2*math.Pi*hz*float64(i)/float64(rate)))
		binary.LittleEndian.PutUint16(out[i*2:], uint16(s))
	}
	return out
}

func TestLECodecPCMURoundTrip(t *testing.T) {
	src := media.CodecConfig{Codec: CodecPCMU, SampleRate: 8000, Channels: 1}
	mid := media.CodecConfig{Codec: "pcm", SampleRate: 8000, Channels: 1, BitDepth: 16}
	enc, err := CreateEncode(src, mid)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := CreateDecode(src, mid)
	if err != nil {
		t.Fatal(err)
	}
	pcm := sineLE(160, 8000, 440, 8000)
	out, err := enc(&media.AudioPacket{Payload: append([]byte(nil), pcm...)})
	if err != nil || len(out) == 0 {
		t.Fatalf("encode: %v", err)
	}
	payload := append([]byte(nil), out[0].(*media.AudioPacket).Payload...)
	got, err := dec(&media.AudioPacket{Payload: payload})
	if err != nil || len(got) == 0 {
		t.Fatalf("decode: %v", err)
	}
}

func TestLECodecG729AndOpus(t *testing.T) {
	{
		src := media.CodecConfig{Codec: CodecG729, SampleRate: 8000, Channels: 1}
		mid := media.CodecConfig{Codec: "pcm", SampleRate: 8000, Channels: 1, BitDepth: 16}
		enc, _ := CreateEncode(src, mid)
		dec, _ := CreateDecode(src, mid)
		out, err := enc(&media.AudioPacket{Payload: sineLE(80, 8000, 440, 5000)})
		if err != nil || len(out) == 0 {
			t.Fatalf("g729: %v", err)
		}
		_, err = dec(&media.AudioPacket{Payload: append([]byte(nil), out[0].(*media.AudioPacket).Payload...)})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		src := media.CodecConfig{Codec: CodecOPUS, SampleRate: 48000, Channels: 1, FrameDuration: "20ms"}
		mid := media.CodecConfig{Codec: "pcm", SampleRate: 48000, Channels: 1, BitDepth: 16}
		enc, _ := CreateEncode(src, mid)
		dec, _ := CreateDecode(src, mid)
		out, err := enc(&media.AudioPacket{Payload: sineLE(960, 48000, 440, 12000)})
		if err != nil || len(out) == 0 {
			t.Fatalf("opus: %v", err)
		}
		_, err = dec(&media.AudioPacket{Payload: append([]byte(nil), out[0].(*media.AudioPacket).Payload...)})
		if err != nil {
			t.Fatal(err)
		}
	}
}
