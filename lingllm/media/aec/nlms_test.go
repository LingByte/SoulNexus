package aec

import (
	"encoding/binary"
	"math"
	"testing"
)

func pcmSine(n int, amp float64, phase float64) []byte {
	out := make([]byte, n*2)
	for i := 0; i < n; i++ {
		v := int16(amp * math.Sin(phase+float64(i)*0.2))
		binary.LittleEndian.PutUint16(out[i*2:], uint16(v))
	}
	return out
}

func TestNLMS_reducesEchoEnergy(t *testing.T) {
	c := New(Config{SampleRate: 8000, FilterMs: 32, Mu: 0.5})
	far := pcmSine(320, 8000, 0)
	// Near = delayed echo of far (scaled) + quiet speech
	near := make([]byte, len(far))
	copy(near, far)
	for i := 0; i+1 < len(near); i += 2 {
		v := int16(binary.LittleEndian.Uint16(near[i:]))
		v = int16(float64(v) * 0.6)
		binary.LittleEndian.PutUint16(near[i:], uint16(v))
	}
	energy := func(b []byte) float64 {
		var s float64
		for i := 0; i+1 < len(b); i += 2 {
			v := float64(int16(binary.LittleEndian.Uint16(b[i:])))
			s += v * v
		}
		return s
	}
	before := energy(near)
	for i := 0; i < 40; i++ {
		c.ProcessFar(far)
		_ = c.ProcessNear(near)
	}
	c.ProcessFar(far)
	afterPCM := c.ProcessNear(near)
	after := energy(afterPCM)
	if after >= before*0.85 {
		t.Fatalf("expected echo energy drop; before=%v after=%v", before, after)
	}
}
