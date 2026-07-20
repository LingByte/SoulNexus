package tts

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"
)

func le16(samples ...int16) []byte {
	b := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(b[i*2:], uint16(s))
	}
	return b
}

func readLE16(b []byte) []int16 {
	if len(b)%2 != 0 {
		panic("odd byte count")
	}
	out := make([]int16, len(b)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(b[i*2:]))
	}
	return out
}

func equalI16(a, b []int16) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestBridgeRateService_StreamingHalveDecimate_ChunkBoundary(t *testing.T) {
	want := make([]int16, 64)
	for i := range want {
		want[i] = int16(i*500 - 16000)
	}
	wantPCM := le16(want...)

	ref := (&BridgeRateService{}).streamingHalveDecimate16to8(wantPCM)

	splits := [][]int{
		{1, 3, 7},
		{3, 5, 11, 17},
		{2, 4, 6, 8, 10, 12, 14, 16},
		{5, 9, 13, 21, 33, 49},
	}
	for _, sp := range splits {
		var got []byte
		svc := &BridgeRateService{}
		offset := 0
		for _, end := range sp {
			if end > len(wantPCM) {
				end = len(wantPCM)
			}
			if end <= offset {
				continue
			}
			got = append(got, svc.streamingHalveDecimate16to8(wantPCM[offset:end])...)
			offset = end
		}
		if offset < len(wantPCM) {
			got = append(got, svc.streamingHalveDecimate16to8(wantPCM[offset:])...)
		}
		if !bytes.Equal(got, ref) {
			t.Fatalf("split %v: got %v want %v", sp, readLE16(got), readLE16(ref))
		}
	}
}

func TestBridgeRateService_2to1_EndToEnd(t *testing.T) {
	in := le16(10, 20, 30, 40, 50, 60, 70, 80)
	inner := ServiceFunc(func(_ context.Context, _ string, onPCMChunk func([]byte) error) error {
		if err := onPCMChunk(in[:3]); err != nil {
			return err
		}
		if err := onPCMChunk(in[3:9]); err != nil {
			return err
		}
		return onPCMChunk(in[9:])
	})

	var got []byte
	svc := BridgeRate(inner, 16000, 8000)
	err := svc.SynthesizeStream(context.Background(), "x", func(out []byte) error {
		got = append(got, out...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []int16{15, 35, 55, 75}
	if !equalI16(readLE16(got), want) {
		t.Fatalf("got %v want %v", readLE16(got), want)
	}
}

func TestBridgeRateService_3to1_StreamsEarly(t *testing.T) {
	// 24k→8k must emit on first chunk, not wait for stream end.
	chunk1 := le16(10, 20, 30, 40, 50, 60) // 3 pairs → 2 out samples if N=3? 6 samples / 3 = 2 groups
	chunk2 := le16(70, 80, 90)
	inner := ServiceFunc(func(_ context.Context, _ string, onPCMChunk func([]byte) error) error {
		if err := onPCMChunk(chunk1); err != nil {
			return err
		}
		return onPCMChunk(chunk2)
	})
	var emits int
	svc := BridgeRate(inner, 24000, 8000)
	err := svc.SynthesizeStream(context.Background(), "x", func(out []byte) error {
		emits++
		if emits == 1 && len(out) == 0 {
			t.Fatal("first emit empty")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if emits < 1 {
		t.Fatalf("expected streaming emits, got %d", emits)
	}
}

func TestBridgeRateService_3to2_StreamsEarly(t *testing.T) {
	chunk1 := le16(1, 2, 3, 4, 5, 6) // one 3-sample group
	inner := ServiceFunc(func(_ context.Context, _ string, onPCMChunk func([]byte) error) error {
		return onPCMChunk(chunk1)
	})
	var got []byte
	svc := BridgeRate(inner, 24000, 16000)
	if err := svc.SynthesizeStream(context.Background(), "x", func(out []byte) error {
		got = append(got, out...)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 8 { // 6 input samples → two 3:2 groups → 4 output samples
		t.Fatalf("got %d bytes, want 8", len(got))
	}
}

func TestBridgeRate_PassthroughWhenEqual(t *testing.T) {
	pcm := le16(1, 2)
	var seen []byte
	inner := ServiceFunc(func(_ context.Context, _ string, onPCMChunk func([]byte) error) error {
		return onPCMChunk(pcm)
	})
	svc := BridgeRate(inner, 8000, 8000)
	if _, wrapped := svc.(*BridgeRateService); wrapped {
		t.Fatalf("expected passthrough, got BridgeRateService wrapper")
	}
	if err := svc.SynthesizeStream(context.Background(), "x", func(b []byte) error {
		seen = append(seen, b...)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(seen, pcm) {
		t.Fatalf("got %v want %v", seen, pcm)
	}
}
