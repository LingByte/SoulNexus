package asr

import (
	"context"
	"testing"
)

func TestPCMCoalesceBuffersUntilMinBytes(t *testing.T) {
	c := NewPCMCoalesceComponent(16000, 20)
	small := make([]byte, 100)
	_, ok, err := c.Process(context.Background(), small)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected hold (shouldContinue=false)")
	}

	out, ok, err := c.Process(context.Background(), make([]byte, 600))
	if err != nil || !ok {
		t.Fatal(err)
	}
	outBytes, ok := out.([]byte)
	if !ok || outBytes == nil {
		t.Fatal("expected flush")
	}
	if len(outBytes) < 320 {
		t.Fatalf("got %d bytes", len(outBytes))
	}
}
