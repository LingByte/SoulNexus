package vad

import (
	"context"
	"math"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/LingByte/lingllm/media"
	llmasr "github.com/LingByte/lingllm/protocol/voice/asr"
)

// EnergyDetector performs RMS-based barge-in while downlink synthesis plays.
type EnergyDetector struct {
	playing   atomic.Bool
	bargeIn   atomic.Bool
	enabled   atomic.Bool
	threshold atomic.Uint64 // float64 bits
	gate      *llmasr.PlaybackGate
	vad       *llmasr.VADComponent
	logFn     func(string)
}

// NewEnergyDetector builds a barge-in detector with voice-aligned defaults.
// Uses DefaultBargeInVADConfig (higher RMS + multi-frame) so TTS echo / line
// noise is less likely to false-trigger interrupt during playback.
func NewEnergyDetector() *EnergyDetector {
	d := &EnergyDetector{}
	d.enabled.Store(true)
	d.gate = llmasr.NewPlaybackGate(func() bool { return d.playing.Load() }, nil, 0)
	cfg := llmasr.DefaultBargeInVADConfig()
	d.vad = llmasr.NewVADComponent(cfg, d.gate)
	d.vad.SetBargeInCallback(func() { d.bargeIn.Store(true) })
	d.threshold.Store(math.Float64bits(cfg.Threshold))
	return d
}

// SetLogFunc attaches an optional log sink (e.g. zap.Info).
func (d *EnergyDetector) SetLogFunc(fn func(string)) {
	if d == nil {
		return
	}
	d.logFn = fn
	if d.vad == nil {
		return
	}
	if fn == nil {
		d.vad.SetLogger(nil)
		return
	}
	d.vad.SetLogger(fn)
}

// CheckBargeIn returns true when uplink PCM suggests the user is speaking during synthesis playback.
func (d *EnergyDetector) CheckBargeIn(pcmData []byte, synthPlaying bool) bool {
	if d == nil || len(pcmData) < 2 || !d.enabled.Load() {
		return false
	}
	d.playing.Store(synthPlaying)
	if !synthPlaying {
		d.bargeIn.Store(false)
		if d.vad != nil {
			d.vad.Reset()
		}
		return false
	}
	if d.vad == nil {
		return false
	}
	_, _, _ = d.vad.Process(context.Background(), pcmData)
	deadline := time.Now().Add(3 * time.Millisecond)
	for time.Now().Before(deadline) {
		if d.bargeIn.Load() {
			d.bargeIn.Store(false)
			return true
		}
		runtime.Gosched()
	}
	return false
}

// SetEnabled turns detection on/off.
func (d *EnergyDetector) SetEnabled(enabled bool) {
	if d == nil {
		return
	}
	d.enabled.Store(enabled)
	if d.vad != nil {
		d.vad.SetEnabled(enabled)
	}
	if !enabled {
		d.bargeIn.Store(false)
	}
}

// Enabled returns whether detection is enabled.
func (d *EnergyDetector) Enabled() bool {
	return d != nil && d.enabled.Load()
}

// SetThreshold sets the RMS ceiling.
func (d *EnergyDetector) SetThreshold(threshold float64) {
	if d != nil && d.vad != nil {
		d.vad.SetThreshold(threshold)
		d.threshold.Store(math.Float64bits(threshold))
	}
}

// SetConsecutiveFrames sets how many consecutive over-threshold frames trigger barge-in.
func (d *EnergyDetector) SetConsecutiveFrames(frames int) {
	if d != nil && d.vad != nil {
		d.vad.SetConsecutiveFrames(frames)
	}
}

// UserSpeechLikely reports uplink speech activity during listen windows.
func (d *EnergyDetector) UserSpeechLikely(pcmData []byte) bool {
	if d == nil || len(pcmData) < 2 || !d.enabled.Load() {
		return false
	}
	thr := math.Float64frombits(d.threshold.Load())
	if thr <= 0 {
		thr = llmasr.DefaultBargeInVADConfig().Threshold
	}
	return CalculateRMS(pcmData) > thr
}

// CalculateRMS computes RMS for PCM16LE frames.
func CalculateRMS(pcmData []byte) float64 {
	if len(pcmData) < 2 {
		return 0
	}
	return media.RMSPCM16LE(pcmData)
}
