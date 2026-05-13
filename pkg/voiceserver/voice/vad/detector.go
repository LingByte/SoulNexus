// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package vad

import (
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Detector performs energy-based (RMS) gating suitable for barge-in
// while downlink TTS synthesis plays. The design is intentionally
// simple: no ML, no per-frame allocation, one mutex. For
// production-grade endpointing we rely on the cloud ASR's own VAD; this
// detector exists solely to decide "is the human trying to interrupt
// the AI right now?" which needs a decision every 20 ms with zero
// network dependency.
type Detector struct {
	mu                      sync.RWMutex
	enabled                 bool
	threshold               float64
	adaptiveThreshold       float64
	consecutiveFramesNeeded int
	frameCounter            int
	logger                  *zap.Logger
	lastLogTime             time.Time
	noiseLevel              float64
	noiseSamples            []float64
	maxNoiseSamples         int
}

// NewDetector builds a detector with defaults calibrated for 20 ms @
// 16 kHz PCM frames from a typical VoIP microphone. Defaults:
//
//   - enabled = true
//   - threshold = 1500 RMS (int16 units)
//   - consecutive frames needed = 1 (fire on first over-threshold)
//   - adaptive noise floor tracked over the last 20 quiet frames
//
// Call the Set* methods to tune for a specific transport profile.
func NewDetector() *Detector {
	return &Detector{
		enabled:                 true,
		threshold:               1500.0,
		adaptiveThreshold:       0,
		consecutiveFramesNeeded: 1,
		frameCounter:            0,
		lastLogTime:             time.Now(),
		noiseLevel:              0,
		noiseSamples:            make([]float64, 0),
		maxNoiseSamples:         20,
	}
}

// SetLogger attaches an optional zap logger (debug for sampled frame
// decisions, info for actual barge-in fires). Pass nil to disable.
func (v *Detector) SetLogger(logger *zap.Logger) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.logger = logger
}

// CheckBargeIn returns true when uplink PCM suggests the user is
// speaking during TTS playback. pcmData must be 16-bit little-endian
// mono PCM (typically 20 ms @ 16 kHz = 320 bytes from the shared ASR
// pipeline). synthPlaying is the caller-tracked flag indicating a TTS
// Speak is currently streaming frames; when false the detector resets
// its counter and returns false without doing any work (cheap check).
//
// Returning true is edge-triggered: the caller should call
// TTS.Interrupt() exactly once per returned true, and the detector's
// internal counter is reset so the next barge-in requires another run
// of consecutive over-threshold frames.
func (v *Detector) CheckBargeIn(pcmData []byte, synthPlaying bool) bool {
	if len(pcmData) < 2 {
		return false
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.enabled || !synthPlaying {
		v.frameCounter = 0
		return false
	}

	rms := calculateRMS(pcmData)

	// Track noise floor from frames quieter than a comfortable human
	// voice baseline (350 RMS ≈ room tone). The adaptive threshold is
	// 4× the running noise so it tracks environment changes (fan, AC)
	// without tripping on them.
	if rms < 350 {
		v.noiseSamples = append(v.noiseSamples, rms)
		if len(v.noiseSamples) > v.maxNoiseSamples {
			v.noiseSamples = v.noiseSamples[1:]
		}
		var sum float64
		for _, sample := range v.noiseSamples {
			sum += sample
		}
		if len(v.noiseSamples) > 0 {
			v.noiseLevel = sum / float64(len(v.noiseSamples))
			v.adaptiveThreshold = v.noiseLevel * 4.0
			if v.adaptiveThreshold < 180 {
				v.adaptiveThreshold = 180
			}
			if v.adaptiveThreshold > v.threshold {
				v.adaptiveThreshold = v.threshold
			}
		}
	}

	effectiveThreshold := v.threshold
	if v.adaptiveThreshold > 0 {
		// Floor the adaptive threshold so a very quiet room doesn't
		// end up tripping on every keystroke. 65% of the configured
		// ceiling or 300 RMS, whichever is larger.
		minAdaptiveFloor := v.threshold * 0.65
		if minAdaptiveFloor < 300 {
			minAdaptiveFloor = 300
		}
		effectiveThreshold = v.adaptiveThreshold
		if effectiveThreshold < minAdaptiveFloor {
			effectiveThreshold = minAdaptiveFloor
		}
	}

	now := time.Now()
	shouldLog := v.logger != nil && now.Sub(v.lastLogTime) >= time.Second

	if rms > effectiveThreshold {
		v.frameCounter++
		if shouldLog {
			v.lastLogTime = now
			v.logger.Debug("vad: energy above threshold",
				zap.Float64("rms", rms),
				zap.Float64("effective_threshold", effectiveThreshold),
				zap.Float64("noise_level", v.noiseLevel),
				zap.Int("frame_counter", v.frameCounter),
				zap.Int("frames_needed", v.consecutiveFramesNeeded),
			)
		}
		if v.frameCounter >= v.consecutiveFramesNeeded {
			if v.logger != nil {
				v.logger.Info("vad: barge-in",
					zap.Float64("rms", rms),
					zap.Float64("effective_threshold", effectiveThreshold),
					zap.Float64("noise_level", v.noiseLevel),
				)
			}
			v.frameCounter = 0
			return true
		}
	} else {
		if v.frameCounter > 0 && shouldLog {
			v.lastLogTime = now
			v.logger.Debug("vad: energy below threshold, reset",
				zap.Float64("rms", rms),
				zap.Int("previous_frames", v.frameCounter),
			)
		}
		v.frameCounter = 0
	}

	return false
}

// SetEnabled turns detection on/off atomically.
func (v *Detector) SetEnabled(enabled bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.enabled = enabled
	if !enabled {
		v.frameCounter = 0
	}
}

// Enabled returns the current enabled state.
func (v *Detector) Enabled() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.enabled
}

// SetThreshold sets the RMS ceiling used in conjunction with the
// adaptive noise floor. Typical values: 1200-1800 for close-mic
// VoIP, 800-1200 for browser getUserMedia (AGC pumps the mic gain).
func (v *Detector) SetThreshold(threshold float64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.threshold = threshold
}

// SetConsecutiveFrames sets how many consecutive over-threshold frames
// trigger barge-in. 1 = maximum responsiveness (may false-trigger on
// coughs); 2-3 = more robust, ~40-60 ms extra latency before interrupt.
func (v *Detector) SetConsecutiveFrames(frames int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if frames < 1 {
		frames = 1
	}
	v.consecutiveFramesNeeded = frames
}

// calculateRMS computes the root-mean-square energy of a PCM16 LE mono
// buffer. Returns 0 for empty / malformed input. Pure function, safe
// to call without the mutex.
func calculateRMS(pcmData []byte) float64 {
	if len(pcmData) < 2 {
		return 0
	}
	var sumSquares float64
	sampleCount := len(pcmData) / 2
	if sampleCount == 0 {
		return 0
	}
	for i := 0; i < len(pcmData)-1; i += 2 {
		sample := int16(pcmData[i]) | int16(pcmData[i+1])<<8
		absSample := math.Abs(float64(sample))
		sumSquares += absSample * absSample
	}
	return math.Sqrt(sumSquares / float64(sampleCount))
}
