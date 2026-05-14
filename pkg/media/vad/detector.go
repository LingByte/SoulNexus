package vad

import (
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Detector performs energy-based (RMS) gating suitable for barge-in while downlink synthesis plays.
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

// NewDetector builds a detector with sipold-aligned defaults.
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

// SetLogger attaches an optional zap logger (debug/info).
func (v *Detector) SetLogger(logger *zap.Logger) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.logger = logger
}

// CheckBargeIn returns true when uplink PCM suggests the user is speaking during synthesis playback.
// pcmData must be 16-bit little-endian mono PCM (typically 20 ms @ 16 kHz from the sip1 decode path).
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
			v.logger.Debug("sip vad: energy above threshold",
				zap.Float64("rms", rms),
				zap.Float64("effectiveThreshold", effectiveThreshold),
				zap.Float64("noiseLevel", v.noiseLevel),
				zap.Int("frameCounter", v.frameCounter),
				zap.Int("framesNeeded", v.consecutiveFramesNeeded),
			)
		}
		if v.frameCounter >= v.consecutiveFramesNeeded {
			if v.logger != nil {
				v.logger.Info("sip vad: barge-in",
					zap.Float64("rms", rms),
					zap.Float64("effectiveThreshold", effectiveThreshold),
					zap.Float64("noiseLevel", v.noiseLevel),
				)
			}
			v.frameCounter = 0
			return true
		}
	} else {
		if v.frameCounter > 0 && shouldLog {
			v.lastLogTime = now
			v.logger.Debug("sip vad: energy below threshold, reset",
				zap.Float64("rms", rms),
				zap.Int("previousFrames", v.frameCounter),
			)
		}
		v.frameCounter = 0
	}

	return false
}

// SetEnabled turns detection on/off.
func (v *Detector) SetEnabled(enabled bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.enabled = enabled
	if !enabled {
		v.frameCounter = 0
	}
}

// SetThreshold sets the RMS ceiling used with adaptive noise tracking.
func (v *Detector) SetThreshold(threshold float64) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.threshold = threshold
}

// SetConsecutiveFrames sets how many consecutive over-threshold frames trigger barge-in.
func (v *Detector) SetConsecutiveFrames(frames int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.consecutiveFramesNeeded = frames
}

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
