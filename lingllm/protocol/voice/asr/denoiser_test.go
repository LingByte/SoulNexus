package asr

import (
	"context"
	"testing"
)

type stubDenoiser struct {
	called bool
}

func (s *stubDenoiser) Process(pcm []byte) []byte {
	s.called = true
	out := make([]byte, len(pcm))
	copy(out, pcm)
	return out
}

func TestDenoiserComponentProcessesPCM(t *testing.T) {
	stub := &stubDenoiser{}
	comp := NewDenoiserComponent(stub)
	out, cont, err := comp.Process(context.Background(), []byte{1, 2, 3, 4})
	if err != nil || !cont {
		t.Fatalf("Process: err=%v cont=%v", err, cont)
	}
	if !stub.called {
		t.Fatal("expected denoiser to be invoked")
	}
	if b, ok := out.([]byte); !ok || len(b) != 4 {
		t.Fatalf("unexpected output: %T %v", out, out)
	}
}

func TestDenoiserComponentEmptyPCM(t *testing.T) {
	comp := NewDenoiserComponent(&stubDenoiser{})
	out, cont, err := comp.Process(context.Background(), []byte{})
	if err != nil || !cont {
		t.Fatalf("Process empty: err=%v cont=%v", err, cont)
	}
	if b, ok := out.([]byte); !ok || len(b) != 0 {
		t.Fatalf("expected empty slice, got %v", out)
	}
}
