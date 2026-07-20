package cascaded

import "context"

type asrPreprocess struct {
	inner ASRRecognizer
	fn    func([]byte) []byte
}

// WrapASRPreprocessor applies uplink PCM preprocessing before ASR.
func WrapASRPreprocessor(inner ASRRecognizer, fn func([]byte) []byte) ASRRecognizer {
	if inner == nil || fn == nil {
		return inner
	}
	return &asrPreprocess{inner: inner, fn: fn}
}

func (w *asrPreprocess) ProcessPCM(ctx context.Context, pcm []byte) error {
	if w == nil || w.inner == nil {
		return nil
	}
	if w.fn != nil && len(pcm) > 0 {
		pcm = w.fn(pcm)
	}
	return w.inner.ProcessPCM(ctx, pcm)
}

func (w *asrPreprocess) SetTextCallback(cb func(text string, isFinal bool)) {
	if w != nil && w.inner != nil {
		w.inner.SetTextCallback(cb)
	}
}

func (w *asrPreprocess) SetErrorCallback(cb func(err error, fatal bool)) {
	if w != nil && w.inner != nil {
		w.inner.SetErrorCallback(cb)
	}
}
