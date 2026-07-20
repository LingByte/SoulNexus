package cascaded

import (
	"context"

	"github.com/LingByte/lingllm/media"
)

type asrResample struct {
	inner   ASRRecognizer
	inRate  int
	outRate int
}

// WrapASRResampler resamples uplink PCM from inRate to outRate before ASR.
// Gateway paths do this inline; voice-session WebSocket debug must match.
func WrapASRResampler(inner ASRRecognizer, inRate, outRate int) ASRRecognizer {
	if inner == nil || inRate <= 0 || outRate <= 0 || inRate == outRate {
		return inner
	}
	return &asrResample{inner: inner, inRate: inRate, outRate: outRate}
}

func (w *asrResample) ProcessPCM(ctx context.Context, pcm []byte) error {
	if w == nil || w.inner == nil {
		return nil
	}
	if len(pcm) == 0 {
		return nil
	}
	pcmASR := pcm
	if w.inRate != w.outRate {
		out, err := media.ResamplePCM(pcm, w.inRate, w.outRate)
		if err != nil {
			return err
		}
		pcmASR = out
	}
	return w.inner.ProcessPCM(ctx, pcmASR)
}

func (w *asrResample) SetTextCallback(cb func(text string, isFinal bool)) {
	if w != nil && w.inner != nil {
		w.inner.SetTextCallback(cb)
	}
}

func (w *asrResample) SetErrorCallback(cb func(err error, fatal bool)) {
	if w != nil && w.inner != nil {
		w.inner.SetErrorCallback(cb)
	}
}
