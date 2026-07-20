package audio

import (
	"math"
	"testing"
)

func TestClassifySNR_hysteresis(t *testing.T) {
	if got := classifySNR(NoiseLevelUnknown, 10); got != NoiseLevelNoisy {
		t.Fatalf("got %v", got)
	}
	if got := classifySNR(NoiseLevelNoisy, 16); got != NoiseLevelNoisy {
		t.Fatalf("stay noisy at 16, got %v", got)
	}
	if got := classifySNR(NoiseLevelNoisy, 18); got != NoiseLevelMild {
		t.Fatalf("leave noisy at 18, got %v", got)
	}
	if got := classifySNR(NoiseLevelMild, 26); got != NoiseLevelClear {
		t.Fatalf("enter clear, got %v", got)
	}
	if got := classifySNR(NoiseLevelClear, 22); got != NoiseLevelMild {
		t.Fatalf("leave clear at 22, got %v", got)
	}
}

func TestSNRMonitor_noisyThenClear(t *testing.T) {
	m := NewSNRMonitor(8000)
	m.warmupFrames = 5

	var gotLevel NoiseLevel
	m.SetListener(func(l NoiseLevel, _ float64) { gotLevel = l })

	// Seed noise floor with quiet frames.
	quiet := make([]int16, 160)
	for i := range quiet {
		quiet[i] = int16(40 * math.Sin(float64(i)))
	}
	for i := 0; i < 8; i++ {
		m.ObserveSamples(quiet)
	}

	// Loud noise-like power with no speech structure (high floor).
	loudNoise := make([]int16, 160)
	for i := range loudNoise {
		loudNoise[i] = int16(8000 * ((i%3)-1))
	}
	for i := 0; i < 40; i++ {
		m.ObserveSamples(loudNoise)
	}
	if m.Level() == NoiseLevelUnknown {
		t.Fatal("expected classified level")
	}

	// Strong speech over low noise.
	m2 := NewSNRMonitor(8000)
	m2.warmupFrames = 3
	for i := 0; i < 5; i++ {
		m2.ObserveSamples(quiet)
	}
	speech := make([]int16, 160)
	for i := range speech {
		speech[i] = int16(12000 * math.Sin(2*math.Pi*float64(i)/20))
	}
	for i := 0; i < 30; i++ {
		m2.ObserveSamples(speech)
	}
	if m2.SNRdB() < 15 {
		t.Fatalf("expected higher SNR for speech, got %v level=%v", m2.SNRdB(), m2.Level())
	}
	_ = gotLevel
}

func TestApplyNoiseHint(t *testing.T) {
	base := "你是客服"
	full := ApplyNoiseHint(base, NoiseSessionHint(NoiseLevelNoisy))
	if full == base {
		t.Fatal("expected hint appended")
	}
	stripped := StripNoiseHint(full)
	if stripped != base {
		t.Fatalf("strip got %q", stripped)
	}
	cleared := ApplyNoiseHint(full, "")
	if cleared != base {
		t.Fatalf("clear hint got %q", cleared)
	}
}
