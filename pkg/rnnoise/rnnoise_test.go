package rnnoise

import "testing"

func TestFrameSizesConsistent(t *testing.T) {
	fs := FrameSamples()
	fb := FrameBytes()
	if fb != fs*2 {
		t.Fatalf("FrameBytes %d != 2*FrameSamples %d", fb, fs*2)
	}
	if fs <= 0 {
		t.Fatalf("FrameSamples: %d", fs)
	}
}
