// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package audio

import (
	"encoding/binary"
	"math"
	"sync"

	"github.com/LingByte/lingllm/media/denoise"
)

// NoiseLevel is a coarse acoustic class used to adapt AI reply style.
type NoiseLevel int

const (
	// NoiseLevelUnknown means not enough audio yet.
	NoiseLevelUnknown NoiseLevel = iota
	// NoiseLevelClear SNR roughly > 25 dB.
	NoiseLevelClear
	// NoiseLevelMild SNR roughly 15–25 dB.
	NoiseLevelMild
	// NoiseLevelNoisy SNR roughly < 15 dB.
	NoiseLevelNoisy
)

func (l NoiseLevel) String() string {
	switch l {
	case NoiseLevelClear:
		return "clear"
	case NoiseLevelMild:
		return "mild"
	case NoiseLevelNoisy:
		return "noisy"
	default:
		return "unknown"
	}
}

// Product SNR banding (dB).
const (
	SNRNoisyMaxDB = 15.0
	SNRMildMaxDB  = 25.0
)

// SNRMonitor tracks a time-domain noise floor + SNR with ~1s EMA and
// hysteresis. Prefers Rust `ld_snr_*` (ledenoise CGO) when built with
// `-tags ledenoise`; otherwise uses an equivalent Go path. No Unix-socket
// media process — estimation stays in-process on the uplink.
type SNRMonitor struct {
	sampleRate   int
	frameSamples int
	warmupFrames int
	native       *denoise.LedenoiseSNR // optional Rust backend

	mu sync.Mutex

	noisePower float64
	snrEMA     float64
	haveNoise  bool
	haveSNR    bool
	framesSeen int

	level     NoiseLevel
	published NoiseLevel // last level delivered to listener

	listener func(NoiseLevel, float64)
}

// NewSNRMonitor builds a monitor for mono PCM16 at sampleRate (Hz).
func NewSNRMonitor(sampleRate int) *SNRMonitor {
	if sampleRate <= 0 {
		sampleRate = 8000
	}
	frame := sampleRate / 50 // ~20 ms
	if frame < 80 {
		frame = 80
	}
	m := &SNRMonitor{
		sampleRate:   sampleRate,
		frameSamples: frame,
		noisePower:   1e-6,
		level:        NoiseLevelUnknown,
		published:    NoiseLevelUnknown,
		warmupFrames: sampleRate / frame, // ~1 s
	}
	if denoise.SNREnabled() {
		if n, err := denoise.NewLedenoiseSNR(sampleRate); err == nil {
			m.native = n
		}
	}
	return m
}

// Backend reports "rust" when native ledenoise SNR is active, else "go".
func (m *SNRMonitor) Backend() string {
	if m != nil && m.native != nil {
		return "rust"
	}
	return "go"
}

// Close releases the optional native estimator.
func (m *SNRMonitor) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	n := m.native
	m.native = nil
	m.mu.Unlock()
	if n != nil {
		_ = n.Close()
	}
}

// SetListener is invoked when the hysteretic level changes after warmup.
// The callback must not call back into the monitor (deadlock risk).
func (m *SNRMonitor) SetListener(fn func(NoiseLevel, float64)) {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.listener = fn
	m.mu.Unlock()
}

// Level returns the current classified level.
func (m *SNRMonitor) Level() NoiseLevel {
	if m == nil {
		return NoiseLevelUnknown
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.level
}

// SNRdB returns the smoothed SNR in dB (0 when not ready).
func (m *SNRMonitor) SNRdB() float64 {
	if m == nil {
		return 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.haveSNR {
		return 0
	}
	return m.snrEMA
}

// ObservePCM feeds mono PCM16LE bytes.
func (m *SNRMonitor) ObservePCM(pcm []byte) {
	if m == nil || len(pcm) < 2 {
		return
	}
	n := len(pcm) / 2
	samples := make([]int16, n)
	for i := 0; i < n; i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	m.ObserveSamples(samples)
}

// ObserveSamples feeds mono int16 PCM (typically one 20 ms RTP frame).
func (m *SNRMonitor) ObserveSamples(samples []int16) {
	if m == nil || len(samples) == 0 {
		return
	}

	var (
		notify   bool
		level    NoiseLevel
		snr      float64
		listener func(NoiseLevel, float64)
	)

	m.mu.Lock()
	native := m.native
	m.mu.Unlock()

	if native != nil {
		snrDB, ready := native.Process(samples)
		m.mu.Lock()
		m.framesSeen++
		if ready {
			m.snrEMA = snrDB
			m.haveSNR = true
			m.level = classifySNR(m.level, m.snrEMA)
		}
		if m.framesSeen >= m.warmupFrames && m.haveSNR && m.level != m.published {
			m.published = m.level
			notify = true
			level = m.level
			snr = m.snrEMA
			listener = m.listener
		}
		m.mu.Unlock()
	} else {
		ps := meanSquare(samples)
		m.mu.Lock()
		m.updateLocked(ps)
		if m.framesSeen >= m.warmupFrames && m.haveSNR && m.level != m.published {
			m.published = m.level
			notify = true
			level = m.level
			snr = m.snrEMA
			listener = m.listener
		}
		m.mu.Unlock()
	}

	if notify && listener != nil {
		listener(level, snr)
	}
}

func (m *SNRMonitor) updateLocked(framePower float64) {
	const (
		eps        = 1e-12
		noiseAlpha = 0.05 // adapt Pn slowly
		noiseBeta  = 0.25 // catch-up when quieter
		snrAlpha   = 0.15 // ~1s EMA at 20ms (1-(1-a)^50 ≈ 0.999)
	)
	if framePower < eps {
		framePower = eps
	}
	m.framesSeen++

	// Minimum-statistics style: always pull noise floor down quickly;
	// raise it slowly when the frame looks noise-like (near the floor).
	if !m.haveNoise {
		m.noisePower = framePower
		m.haveNoise = true
	} else if framePower < m.noisePower {
		m.noisePower = (1-noiseBeta)*m.noisePower + noiseBeta*framePower
	} else if framePower < m.noisePower*4 {
		// Near-noise / low speech — keep tracking.
		m.noisePower = (1-noiseAlpha)*m.noisePower + noiseAlpha*framePower
	}
	// Speech-dominant frames do not raise the noise floor (matches WebRTC-style
	// minimum statistics: noise tracked in pauses / low-energy regions).
	if m.noisePower < eps {
		m.noisePower = eps
	}

	snr := 10 * math.Log10(framePower/m.noisePower)
	if snr < -10 {
		snr = -10
	}
	if snr > 60 {
		snr = 60
	}
	if !m.haveSNR {
		m.snrEMA = snr
		m.haveSNR = true
	} else {
		m.snrEMA = (1-snrAlpha)*m.snrEMA + snrAlpha*snr
	}
	m.level = classifySNR(m.level, m.snrEMA)
}

func classifySNR(prev NoiseLevel, snr float64) NoiseLevel {
	// Hysteresis bands around 15 / 25 dB.
	const (
		enterNoisy = SNRNoisyMaxDB        // 15
		leaveNoisy = SNRNoisyMaxDB + 2    // 17
		enterMild  = SNRMildMaxDB         // 25
		leaveMild  = SNRMildMaxDB - 2     // 23
	)
	switch prev {
	case NoiseLevelNoisy:
		if snr >= leaveNoisy && snr < enterMild {
			return NoiseLevelMild
		}
		if snr >= enterMild {
			return NoiseLevelClear
		}
		return NoiseLevelNoisy
	case NoiseLevelMild:
		if snr < enterNoisy {
			return NoiseLevelNoisy
		}
		if snr >= enterMild {
			return NoiseLevelClear
		}
		return NoiseLevelMild
	case NoiseLevelClear:
		if snr < enterNoisy {
			return NoiseLevelNoisy
		}
		if snr < leaveMild {
			return NoiseLevelMild
		}
		return NoiseLevelClear
	default:
		if snr < SNRNoisyMaxDB {
			return NoiseLevelNoisy
		}
		if snr < SNRMildMaxDB {
			return NoiseLevelMild
		}
		return NoiseLevelClear
	}
}

func meanSquare(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		v := float64(s) / 32768.0
		sum += v * v
	}
	return sum / float64(len(samples))
}
