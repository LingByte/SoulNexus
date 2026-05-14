package vad

import (
	"bytes"
	"testing"

	"go.uber.org/zap"
)

func loudPCMFrame() []byte {
	buf := make([]byte, 320)
	for i := 0; i < len(buf); i += 2 {
		buf[i] = 0xff
		buf[i+1] = 0x7f // int16 LE ≈ 32767
	}
	return buf
}

func quietPCMFrame() []byte {
	return bytes.Repeat([]byte{0x00, 0x00}, 80)
}

func TestCalculateRMS_Edges(t *testing.T) {
	if calculateRMS(nil) != 0 || calculateRMS([]byte{1}) != 0 {
		t.Fatal()
	}
	if v := calculateRMS([]byte{0x00, 0x10}); v <= 0 {
		t.Fatalf("rms %v", v)
	}
}

func TestDetector_CheckBargeIn_DisabledOrNotPlaying(t *testing.T) {
	d := NewDetector()
	d.SetEnabled(false)
	if d.CheckBargeIn(loudPCMFrame(), true) {
		t.Fatal("disabled")
	}
	d.SetEnabled(true)
	if d.CheckBargeIn(loudPCMFrame(), false) {
		t.Fatal("not playing")
	}
}

func TestDetector_CheckBargeIn_Triggers(t *testing.T) {
	d := NewDetector()
	d.SetThreshold(500)
	d.SetConsecutiveFrames(1)
	d.SetLogger(zap.NewNop())
	if !d.CheckBargeIn(loudPCMFrame(), true) {
		t.Fatal("expected barge-in")
	}
}

func TestDetector_CheckBargeIn_ConsecutiveFrames(t *testing.T) {
	d := NewDetector()
	d.SetThreshold(500)
	d.SetConsecutiveFrames(2)
	frame := loudPCMFrame()
	if d.CheckBargeIn(frame, true) {
		t.Fatal("first frame should not trigger")
	}
	if !d.CheckBargeIn(frame, true) {
		t.Fatal("second frame should trigger")
	}
}

func TestDetector_AdaptiveNoiseFloor(t *testing.T) {
	d := NewDetector()
	d.SetThreshold(2000)
	d.SetConsecutiveFrames(1)
	quiet := quietPCMFrame()
	for i := 0; i < 25; i++ {
		_ = d.CheckBargeIn(quiet, true)
	}
	if d.CheckBargeIn(quiet, true) {
		t.Fatal("quiet should stay below adaptive threshold")
	}
}

func TestDetector_SetThreshold(t *testing.T) {
	d := NewDetector()
	d.SetThreshold(40000) // above max int16 RMS (~32767)
	d.SetConsecutiveFrames(1)
	if d.CheckBargeIn(loudPCMFrame(), true) {
		t.Fatal("high threshold should block")
	}
}
