package session

import (
	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
)

type uplinkPreprocessPort struct {
	inner engine.MediaPort
	fn    func([]byte) []byte
}

func wrapUplinkPort(inner engine.MediaPort, fn func([]byte) []byte) engine.MediaPort {
	if inner == nil || fn == nil {
		return inner
	}
	return &uplinkPreprocessPort{inner: inner, fn: fn}
}

func (p *uplinkPreprocessPort) InputPCM() <-chan engine.PCMFrame {
	src := p.inner.InputPCM()
	out := make(chan engine.PCMFrame, 32)
	go func() {
		defer close(out)
		for frame := range src {
			if p.fn != nil && len(frame.Data) > 0 {
				frame.Data = p.fn(frame.Data)
			}
			out <- frame
		}
	}()
	return out
}

func (p *uplinkPreprocessPort) SendOutputPCM(frame engine.PCMFrame) error {
	return p.inner.SendOutputPCM(frame)
}

func (p *uplinkPreprocessPort) OnBargeIn(cb func()) {
	p.inner.OnBargeIn(cb)
}

func (p *uplinkPreprocessPort) Codec() engine.CodecSpec {
	return p.inner.Codec()
}

func (p *uplinkPreprocessPort) SampleRate() int  { return p.inner.SampleRate() }
func (p *uplinkPreprocessPort) CallID() string   { return p.inner.CallID() }
func (p *uplinkPreprocessPort) TenantID() string { return p.inner.TenantID() }
