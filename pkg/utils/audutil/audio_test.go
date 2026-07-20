package audutil

import (
	"encoding/binary"
	"math"
	"testing"
	"time"
)

func TestRMSPCM16LE_Empty(t *testing.T) {
	if got := RMSPCM16LE(nil); got != 0 {
		t.Fatalf("nil: got %v want 0", got)
	}
	if got := RMSPCM16LE([]byte{0}); got != 0 {
		t.Fatalf("short: got %v want 0", got)
	}
}

func TestRMSPCM16LE_Silence(t *testing.T) {
	pcm := make([]byte, 320)
	if got := RMSPCM16LE(pcm); got != 0 {
		t.Fatalf("silence: got %v want 0", got)
	}
}

func TestRMSPCM16LE_FullScaleSquare(t *testing.T) {
	pcm := make([]byte, 4)
	binary.LittleEndian.PutUint16(pcm[0:2], uint16(int16(10000)))
	neg := int16(-10000)
	binary.LittleEndian.PutUint16(pcm[2:4], uint16(neg))
	want := math.Sqrt(10000 * 10000)
	got := RMSPCM16LE(pcm)
	if math.Abs(got-want) > 0.01 {
		t.Fatalf("got %v want ~%v", got, want)
	}
}

func TestRecordingObjectKey(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if got := RecordingObjectKey(7, "call-1", at, "sn2"); got != "recordings/7/2026-01-02/call-1.sn2" {
		t.Fatalf("got %q", got)
	}
	if got := RecordingPartObjectKey(7, "call-1", at, 2, 99); got != "recordings/7/2026-01-02/call-1-part-2-99.wav" {
		t.Fatalf("part got %q", got)
	}
}

func TestRMSPCM16LE_OddLengthIgnoredTail(t *testing.T) {
	pcm := []byte{0, 0, 0xFF} // third byte ignored
	if got := RMSPCM16LE(pcm); got != 0 {
		t.Fatalf("got %v want 0", got)
	}
}
