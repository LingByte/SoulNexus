package audio

import (
	"fmt"
	"sync"

	"github.com/LingByte/lingllm/media/aec"
	"github.com/LingByte/lingllm/media/denoise"
	llmasr "github.com/LingByte/lingllm/protocol/voice/asr"
)

// UplinkConfig controls uplink denoise / AEC from assistant audioProcessConfig.
type UplinkConfig struct {
	NoiseSuppressionEnabled bool
	NoiseSuppressionType    string
	SampleRate              int
}

// UplinkProcessor applies optional denoise / far-end AEC on caller PCM before ASR/VAD.
// SNR monitoring always runs (before denoise) so AI reply style can adapt to environment noise.
type UplinkProcessor struct {
	denoiser   llmasr.Denoiser
	aec        *aec.Canceller
	snr        *SNRMonitor
	callID     string
	sampleRate int
	closeFn    func() error
	mu         sync.Mutex
}

type ledenoiseAdapter struct {
	d *denoise.Ledenoise
}

func (a *ledenoiseAdapter) Process(pcm []byte) []byte {
	if a == nil || a.d == nil {
		return pcm
	}
	return a.d.Process(pcm)
}

// NewUplinkProcessor builds uplink denoise/AEC from assistant audioProcessConfig.
//
// Types:
//   - none / "" — passthrough
//   - aec / nlms — NLMS far-end echo cancellation (requires ProcessFar from playback)
//   - ledenoise / native / nnnoiseless — Rust RNNoise (requires -tags ledenoise)
//   - rnnoise — Go librnnoise (requires -tags rnnoise)
//   - simple — lightweight AGC/noise gate (NOT reference-based AEC)
func NewUplinkProcessor(cfg UplinkConfig, sampleRate int) (*UplinkProcessor, error) {
	if sampleRate <= 0 {
		sampleRate = cfg.SampleRate
	}
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	p := &UplinkProcessor{
		snr:        NewSNRMonitor(sampleRate),
		sampleRate: sampleRate,
	}
	if !cfg.NoiseSuppressionEnabled {
		return p, nil
	}
	switch cfg.NoiseSuppressionType {
	case "none", "":
		return p, nil
	case "aec", "nlms", "speex": // speex alias maps to NLMS until SpeexDSP CGO lands
		c := aec.New(aec.Config{SampleRate: sampleRate, FilterMs: 64, Mu: 0.4})
		p.aec = c
		p.closeFn = c.Close
		return p, nil
	case "ledenoise", "native", "nnnoiseless":
		d, err := denoise.NewLedenoise(sampleRate)
		if err != nil {
			dn, err2 := llmasr.NewSimpleDenoiser(&llmasr.SimpleDenoiserConfig{
				SampleRate:    sampleRate,
				Channels:      1,
				BitsPerSample: 16,
				AECEnable:     false,
				AGCEnable:     true,
			})
			if err2 != nil {
				return nil, fmt.Errorf("uplink ledenoise unavailable (%v); simple fallback: %w", err, err2)
			}
			p.denoiser = dn
			p.closeFn = dn.Close
			return p, nil
		}
		p.denoiser = &ledenoiseAdapter{d: d}
		p.closeFn = d.Close
		return p, nil
	case "rnnoise":
		factory := llmasr.NewDenoiserFactory()
		compAny, err := factory.CreateDenoiser(llmasr.DenoiserTypeRNNoise, nil)
		if err != nil {
			return nil, err
		}
		if dn, ok := compAny.(llmasr.Denoiser); ok {
			p.denoiser = dn
			return p, nil
		}
		if denoise.LedenoiseEnabled() {
			d, err := denoise.NewLedenoise(sampleRate)
			if err == nil {
				p.denoiser = &ledenoiseAdapter{d: d}
				p.closeFn = d.Close
				return p, nil
			}
		}
		return p, nil
	default: // simple
		dn, err := llmasr.NewSimpleDenoiser(&llmasr.SimpleDenoiserConfig{
			SampleRate:    sampleRate,
			Channels:      1,
			BitsPerSample: 16,
			AECEnable:     false, // placeholder amplitude cut disabled; use type=aec for real AEC
			AGCEnable:     true,
		})
		if err != nil {
			return nil, err
		}
		p.denoiser = dn
		p.closeFn = dn.Close
		return p, nil
	}
}

// BindCallID associates uplink SNR updates with a session id.
func (p *UplinkProcessor) BindCallID(callID string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.callID = callID
	snr := p.snr
	p.mu.Unlock()
	if snr == nil {
		return
	}
	snr.SetListener(func(level NoiseLevel, snrDB float64) {
		_ = snrDB
		GlobalCallNoise.Set(callID, level)
		notifyCallNoiseListeners(callID, level, snrDB)
	})
}

// SNRMonitor returns the uplink SNR tracker (may be nil).
func (p *UplinkProcessor) SNRMonitor() *SNRMonitor {
	if p == nil {
		return nil
	}
	return p.snr
}

// ProcessFar feeds far-end (playback) PCM into the AEC reference path.
func (p *UplinkProcessor) ProcessFar(pcm []byte) {
	if p == nil || len(pcm) == 0 || p.aec == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.aec.ProcessFar(pcm)
}

// HasAEC reports whether far-end reference AEC is active.
func (p *UplinkProcessor) HasAEC() bool {
	return p != nil && p.aec != nil
}

// ProcessPCM returns cleaned near-end PCM (or input when disabled).
func (p *UplinkProcessor) ProcessPCM(pcm []byte) []byte {
	if p == nil || len(pcm) == 0 {
		return pcm
	}
	// Estimate SNR on raw uplink before denoise (noise floor must see the environment).
	if p.snr != nil {
		p.snr.ObservePCM(pcm)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.aec != nil {
		pcm = p.aec.ProcessNear(pcm)
	}
	if p.denoiser != nil {
		return p.denoiser.Process(pcm)
	}
	return pcm
}

// Close releases resources.
func (p *UplinkProcessor) Close() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	callID := p.callID
	snr := p.snr
	p.snr = nil
	p.mu.Unlock()
	if callID != "" {
		GlobalCallNoise.Clear(callID)
		clearCallNoiseListeners(callID)
	}
	if snr != nil {
		snr.Close()
	}
	if p.closeFn == nil {
		return nil
	}
	return p.closeFn()
}
