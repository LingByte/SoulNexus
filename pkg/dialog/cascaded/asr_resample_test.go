package cascaded

import (
	"context"
	"testing"
)

type captureASR struct {
	last []byte
}

func (c *captureASR) ProcessPCM(_ context.Context, pcm []byte) error {
	c.last = append([]byte(nil), pcm...)
	return nil
}

func (c *captureASR) SetTextCallback(func(string, bool)) {}
func (c *captureASR) SetErrorCallback(func(error, bool)) {}

func TestWrapASRResampler8kTo16k(t *testing.T) {
	inner := &captureASR{}
	wrapped := WrapASRResampler(inner, 16000, 8000)
	// 320 bytes @ 16kHz mono 16-bit = 10ms; resampled to 8000 → 160 bytes.
	in := make([]byte, 320)
	for i := range in {
		in[i] = byte(i % 256)
	}
	if err := wrapped.ProcessPCM(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if len(inner.last) != 160 {
		t.Fatalf("resampled len=%d want 160", len(inner.last))
	}
}

func TestWrapASRResamplerNoOpWhenRatesMatch(t *testing.T) {
	inner := &captureASR{}
	wrapped := WrapASRResampler(inner, 16000, 16000)
	if wrapped != inner {
		t.Fatal("expected same instance when rates match")
	}
}
