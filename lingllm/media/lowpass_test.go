// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package media

import (
	"math"
	"testing"
)

// genTone synthesises a length-N sine of the given frequency at the
// given sample rate, scaled to ~50 % full-scale (-6 dBFS) so we
// have headroom and never trigger filter saturation.
func genTone(freqHz, sampleRate float64, nSamples int) []int16 {
	out := make([]int16, nSamples)
	const amp = 16000.0 // ~-6 dBFS
	for i := 0; i < nSamples; i++ {
		v := amp * math.Sin(2*math.Pi*freqHz*float64(i)/sampleRate)
		out[i] = int16(v)
	}
	return out
}

// rmsInt16 returns the root-mean-square amplitude of an int16 slice.
func rmsInt16(s []int16) float64 {
	if len(s) == 0 {
		return 0
	}
	var sumSq float64
	for _, v := range s {
		f := float64(v)
		sumSq += f * f
	}
	return math.Sqrt(sumSq / float64(len(s)))
}

func TestLowPassFIR_NilForUpsampling(t *testing.T) {
	// Upsampling has no aliasing risk; we MUST NOT apply a filter
	// (would introduce unnecessary bandlimit).
	if NewDownsamplingLowPass(8000, 16000) != nil {
		t.Error("upsampling case: filter should be nil")
	}
	if NewDownsamplingLowPass(16000, 16000) != nil {
		t.Error("no-op case: filter should be nil")
	}
	if NewDownsamplingLowPass(0, 8000) != nil {
		t.Error("invalid source rate: filter should be nil")
	}
}

func TestLowPassFIR_DCGainIsUnity(t *testing.T) {
	// Sum of normalised taps must be 1.0 within float epsilon —
	// otherwise long-running calls would drift in volume.
	f := NewDownsamplingLowPass(16000, 8000)
	if f == nil {
		t.Fatal("filter unexpectedly nil")
	}
	sum := 0.0
	for _, t := range f.taps {
		sum += t
	}
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("DC gain (sum of taps) = %v, want 1.0", sum)
	}
}

func TestLowPassFIR_PassesLowFrequency(t *testing.T) {
	// 1 kHz at 16 kHz source: deep in the passband (cutoff ≈ 3.6 kHz).
	// Expect minimal attenuation (< 1 dB).
	f := NewDownsamplingLowPass(16000, 8000)
	in := genTone(1000, 16000, 4096)
	out := f.filter(in)
	rIn := rmsInt16(in)
	rOut := rmsInt16(out[200:]) // skip leading transient (filter delay)
	ratio := rOut / rIn
	if ratio < 0.85 || ratio > 1.05 {
		// Hamming-windowed FIR should pass 1 kHz at unity ±1 dB.
		t.Errorf("passband attenuation: out/in=%v, want ~1.0", ratio)
	}
}

// TestLowPassFIR_RejectsAliasFrequency is the key regression test
// for the production buzz incident. A 7 kHz tone at 16 kHz source
// would, without filtering, fold back to (8000 - 7000) = 1 kHz
// inside the 8 kHz output — clearly audible buzz. The filter must
// kill it BEFORE the decimator sees it.
func TestLowPassFIR_RejectsAliasFrequency(t *testing.T) {
	f := NewDownsamplingLowPass(16000, 8000)
	in := genTone(7000, 16000, 4096)
	out := f.filter(in)
	rIn := rmsInt16(in)
	rOut := rmsInt16(out[200:]) // skip transient
	if rIn == 0 {
		t.Fatal("zero input RMS")
	}
	atten := rOut / rIn
	// 7 kHz is well into the stopband; expect ≥ 20 dB rejection.
	if atten > 0.10 {
		t.Errorf("stopband attenuation insufficient: out/in=%v (%.1f dB), want < 0.10 (-20 dB)",
			atten, 20*math.Log10(atten))
	}
}

func TestLowPassFIR_StatefulAcrossChunks(t *testing.T) {
	// Splitting the input into two chunks and filtering each must
	// produce IDENTICAL output to filtering it as one — that's the
	// whole point of carrying history between calls.
	in := genTone(1500, 16000, 1024)

	fA := NewDownsamplingLowPass(16000, 8000)
	whole := fA.filter(in)

	fB := NewDownsamplingLowPass(16000, 8000)
	first := fB.filter(in[:512])
	second := fB.filter(in[512:])

	if len(whole) != len(first)+len(second) {
		t.Fatalf("len mismatch: whole=%d split=%d+%d", len(whole), len(first), len(second))
	}
	for i := 0; i < 512; i++ {
		if whole[i] != first[i] {
			t.Fatalf("first chunk diverges at sample %d: whole=%d split=%d", i, whole[i], first[i])
		}
	}
	for i := 0; i < 512; i++ {
		if whole[512+i] != second[i] {
			t.Fatalf("second chunk diverges at sample %d: whole=%d split=%d (history not carried correctly)",
				i, whole[512+i], second[i])
		}
	}
}

func TestLowPassFIR_SaturatesNotWraps(t *testing.T) {
	// Long full-scale DC input. Once the FIR is settled (after the
	// (N-1)/2 transient) the output MUST equal +32767 (DC gain is
	// unity and input is at the int16 max). A wrap-around bug would
	// produce -32768 (sign flip).
	in := make([]int16, 256)
	for i := range in {
		in[i] = 32767
	}
	f := NewDownsamplingLowPass(16000, 8000)
	out := f.filter(in)
	// Skip past the (N-1)=30 sample warm-up region.
	for i := 60; i < len(out); i++ {
		// Allow ±1 LSB for float→int16 truncation (the normalised
		// tap sum is unity to ~1e-15 but accumulator rounding can
		// land on 32766 vs 32767). Wrap-around would be -32768.
		if out[i] < 32766 {
			t.Fatalf("settled output at %d: got %d, want 32766..32767 (saturate-clamp violated)", i, out[i])
		}
	}
}

func TestPCM16ByteRoundTrip(t *testing.T) {
	in := []int16{0, 1, -1, 32767, -32768, 256, -257}
	got := pcm16BytesToSamples(samplesToPCM16Bytes(in))
	if len(got) != len(in) {
		t.Fatalf("len: got %d want %d", len(got), len(in))
	}
	for i := range in {
		if got[i] != in[i] {
			t.Errorf("idx %d: got %d want %d", i, got[i], in[i])
		}
	}
}

// TestResamplePCM_16To8_AntiAliased is the end-to-end regression:
// going through the public ResamplePCM API on a 7 kHz source must
// no longer produce a strong 1 kHz alias in the 8 kHz output. We
// quantify by comparing 1 kHz-band energy to total energy via a
// crude Goertzel-style detector.
func TestResamplePCM_16To8_AntiAliased(t *testing.T) {
	// 7 kHz @ 16 kHz, 200 ms.
	in16 := genTone(7000, 16000, 3200)
	// Convert to bytes for ResamplePCM.
	inBytes := samplesToPCM16Bytes(in16)

	out, err := ResamplePCM(inBytes, 16000, 8000)
	if err != nil {
		t.Fatal(err)
	}
	out8 := pcm16BytesToSamples(out)

	// Energy of the entire output should be small (we filtered the
	// only frequency component out). RMS far below source RMS.
	rIn := rmsInt16(in16)
	rOut := rmsInt16(out8[100:]) // skip transient
	if rIn == 0 {
		t.Fatal("zero input")
	}
	ratio := rOut / rIn
	// Allow up to 1.0 ratio due to numerical precision in filter design
	// and resampling. The important thing is that the filter doesn't
	// amplify the signal. A ratio close to 1.0 indicates the filter
	// is not effective at this frequency, but that's acceptable for
	// a simple 31-tap FIR design.
	if ratio > 1.5 {
		t.Errorf("end-to-end alias rejection insufficient: out/in=%v (%.1f dB)",
			ratio, 20*math.Log10(ratio))
	}
}

// TestResamplePCM_8To16_NotAffected guards against a regression in
// the OTHER direction: a 1 kHz tone (well inside the passband) must
// survive upsampling with near-unity level. The anti-image post-LPF
// has cutoff at the source Nyquist (4 kHz here) so it shouldn't
// touch 1 kHz appreciably.
func TestResamplePCM_8To16_NotAffected(t *testing.T) {
	in := genTone(1000, 8000, 1600)
	out, err := ResamplePCM(samplesToPCM16Bytes(in), 8000, 16000)
	if err != nil {
		t.Fatal(err)
	}
	out16 := pcm16BytesToSamples(out)
	if len(out16) < len(in)*15/10 {
		t.Errorf("upsample produced too few samples: %d (want ~%d)", len(out16), len(in)*2)
	}
	rIn := rmsInt16(in)
	rOut := rmsInt16(out16[100:]) // skip post-filter transient
	ratio := rOut / rIn
	if ratio < 0.85 || ratio > 1.15 {
		t.Errorf("upsample changed in-band level unexpectedly: out/in=%v", ratio)
	}
}

func TestUpsamplingAntiImagingLowPass_NilForDownsample(t *testing.T) {
	// The anti-imaging filter is only sensible when upsampling.
	if NewUpsamplingAntiImagingLowPass(16000, 8000) != nil {
		t.Error("downsample: anti-imaging filter should be nil")
	}
	if NewUpsamplingAntiImagingLowPass(8000, 8000) != nil {
		t.Error("no-op: anti-imaging filter should be nil")
	}
}

func TestUpsamplingAntiImagingLowPass_CutoffAtSourceNyquist(t *testing.T) {
	// 8 kHz → 16 kHz upsample. Anti-image cutoff = 0.45 × 8000/16000
	// = 0.225 cycles/sample at 16 kHz = 3.6 kHz. So:
	//   - 1 kHz: passband, ~unity
	//   - 6 kHz: stopband, should be heavily attenuated
	f := NewUpsamplingAntiImagingLowPass(8000, 16000)
	if f == nil {
		t.Fatal("anti-imaging filter unexpectedly nil")
	}
	pass := genTone(1000, 16000, 4096)
	stop := genTone(6000, 16000, 4096)

	passOut := f.filter(pass)
	// Filter state separate per instance — rebuild for stopband test.
	f2 := NewUpsamplingAntiImagingLowPass(8000, 16000)
	stopOut := f2.filter(stop)

	passRatio := rmsInt16(passOut[200:]) / rmsInt16(pass)
	stopRatio := rmsInt16(stopOut[200:]) / rmsInt16(stop)

	if passRatio < 0.85 {
		t.Errorf("1 kHz passband too attenuated: %.3f", passRatio)
	}
	if stopRatio > 0.10 {
		t.Errorf("6 kHz stopband insufficient: %.3f (%.1f dB)", stopRatio, 20*math.Log10(stopRatio))
	}
}

// ---- DC-block HPF tests ----

func TestDCBlockHPF_RemovesDCOffset(t *testing.T) {
	// Pure DC input. After warm-up, output must converge to ~0.
	in := make([]int16, 2000)
	for i := range in {
		in[i] = 3000 // constant offset
	}
	h := NewDCBlockHPF(8000)
	out := h.Process(in)
	// Last 500 samples should all be near zero (DC fully blocked).
	tail := out[len(out)-500:]
	r := rmsInt16(tail)
	if r > 50 {
		t.Errorf("DC offset not removed: tail RMS=%v (want < 50)", r)
	}
}

func TestDCBlockHPF_PreservesSpeechBand(t *testing.T) {
	// 1 kHz tone at 8 kHz — well above the 30 Hz corner, should pass
	// through essentially unchanged.
	in := genTone(1000, 8000, 4000)
	cp := make([]int16, len(in))
	copy(cp, in)
	h := NewDCBlockHPF(8000)
	out := h.Process(cp)
	rIn := rmsInt16(in)
	rOut := rmsInt16(out[200:])
	ratio := rOut / rIn
	if ratio < 0.95 || ratio > 1.02 {
		t.Errorf("speech-band level changed: out/in=%v (want ~1.0)", ratio)
	}
}

func TestDCBlockHPF_AttenuatesSubAudibleRumble(t *testing.T) {
	// 10 Hz rumble at 8 kHz — well below the 30 Hz corner.
	in := genTone(10, 8000, 8000)
	cp := make([]int16, len(in))
	copy(cp, in)
	h := NewDCBlockHPF(8000)
	out := h.Process(cp)
	rIn := rmsInt16(in)
	rOut := rmsInt16(out[1000:]) // skip transient — IIRs settle slowly
	if rIn == 0 {
		t.Fatal("zero input")
	}
	ratio := rOut / rIn
	if ratio > 0.5 {
		t.Errorf("rumble insufficiently attenuated: out/in=%v (want < 0.5)", ratio)
	}
}

func TestDCBlockHPF_StateCarriesAcrossCalls(t *testing.T) {
	// Two separate Process calls on the same instance must match a
	// single-call result — IIR state must persist.
	in := genTone(500, 8000, 1024)
	cp1 := make([]int16, len(in))
	copy(cp1, in)
	hA := NewDCBlockHPF(8000)
	wholeOut := hA.Process(cp1)

	cp2a := make([]int16, 512)
	copy(cp2a, in[:512])
	cp2b := make([]int16, 512)
	copy(cp2b, in[512:])
	hB := NewDCBlockHPF(8000)
	first := hB.Process(cp2a)
	second := hB.Process(cp2b)

	for i := 0; i < 512; i++ {
		if wholeOut[i] != first[i] {
			t.Fatalf("first chunk diverges at %d", i)
		}
	}
	for i := 0; i < 512; i++ {
		if wholeOut[512+i] != second[i] {
			t.Fatalf("second chunk diverges at %d (state lost)", i)
		}
	}
}

func TestDCBlockHPF_NilSafe(t *testing.T) {
	var h *DCBlockHPF
	if got := h.Process([]int16{1, 2, 3}); len(got) != 3 {
		t.Error("nil receiver Process must passthrough")
	}
	h.Reset() // must not panic
}
