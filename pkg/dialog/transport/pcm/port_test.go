package pcm

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
)

func TestPortLifecycle(t *testing.T) {
	p := NewPort(Config{SessionID: "s1", TenantID: "9", SampleRate: 16000})
	var out int
	p.OutputFn = func(f engine.PCMFrame) error {
		out += len(f.Data)
		return nil
	}
	if err := p.PushInput([]byte{0, 0, 1, 0}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-p.InputPCM():
	default:
		t.Fatal("expected input frame")
	}
	if err := p.SendOutputPCM(engine.PCMFrame{Data: []byte{1, 2, 3, 4}, SampleRate: 16000}); err != nil {
		t.Fatal(err)
	}
	if out != 4 {
		t.Fatalf("out=%d", out)
	}
	var bargeHits int
	p.OnBargeIn(func() { bargeHits++ })
	p.TriggerBargeIn()
	if bargeHits != 1 {
		t.Fatalf("TriggerBargeIn hits=%d want 1", bargeHits)
	}
	_ = p.Close()
	if err := p.PushInput([]byte{1}); err != ErrClosed {
		t.Fatalf("push after close: %v", err)
	}
}
