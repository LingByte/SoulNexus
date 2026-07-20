package audutil

import (
	"encoding/binary"
	"testing"
	"time"
)

func buildTestStereoWAV(leftSamples, rightSamples []int16, sampleRate int) []byte {
	if len(leftSamples) != len(rightSamples) {
		panic("stereo test: channel length mismatch")
	}
	interleaved := make([]int16, len(leftSamples)*2)
	for i := range leftSamples {
		interleaved[2*i] = leftSamples[i]
		interleaved[2*i+1] = rightSamples[i]
	}
	pcm := make([]byte, len(interleaved)*2)
	for i, s := range interleaved {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(s))
	}
	return pcm16StereoInterleavedToWav(interleaved, sampleRate)
}

func TestSplitStereoWAVChannels(t *testing.T) {
	wav := buildTestStereoWAV([]int16{100, 200, 300}, []int16{10, 20, 30}, 8000)
	left, right, sr, err := SplitStereoWAVChannels(wav)
	if err != nil {
		t.Fatal(err)
	}
	if sr != 8000 {
		t.Fatalf("sampleRate=%d", sr)
	}
	if len(left) != 6 || len(right) != 6 {
		t.Fatalf("channel bytes left=%d right=%d", len(left), len(right))
	}
	if int16(binary.LittleEndian.Uint16(left[0:2])) != 100 {
		t.Fatalf("left[0] wrong")
	}
	if int16(binary.LittleEndian.Uint16(right[2:4])) != 20 {
		t.Fatalf("right[1] wrong")
	}
}

func TestTrimPCMS16FromTime(t *testing.T) {
	pcm := make([]byte, 8000*2*2) // 2s @ 8kHz mono
	for i := 0; i < len(pcm)/2; i++ {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(i))
	}
	trimmed := TrimPCMS16FromTime(pcm, 8000, 500*time.Millisecond)
	wantBytes := (16000 - 4000) * 2 // 1.5s left @ 8kHz mono
	if len(trimmed) != wantBytes {
		t.Fatalf("trimmed len=%d want=%d", len(trimmed), wantBytes)
	}
}

func TestMonoPCM16ToWAV_roundTrip(t *testing.T) {
	pcm := []byte{0x01, 0x00, 0x02, 0x00}
	wav, err := MonoPCM16ToWAV(pcm, 8000)
	if err != nil {
		t.Fatal(err)
	}
	left, right, sr, err := SplitStereoWAVChannels(wav)
	if err != nil {
		t.Fatal(err)
	}
	if sr != 8000 || len(right) != 0 || len(left) != 4 {
		t.Fatalf("roundTrip left=%d right=%d sr=%d", len(left), len(right), sr)
	}
}
