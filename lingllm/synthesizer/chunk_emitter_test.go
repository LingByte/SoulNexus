package synthesizer

import (
	"context"
	"testing"
)

type collectHandler struct {
	chunks [][]byte
}

func (h *collectHandler) OnMessage(data []byte) {
	if len(data) == 0 {
		return
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	h.chunks = append(h.chunks, cp)
}

func (h *collectHandler) OnTimestamp(SentenceTimestamp) {}

func TestFrameBytes(t *testing.T) {
	cfg := PCMEmitConfig{SampleRate: 16000, BitDepth: 16, Channels: 1, FrameMS: 20}
	if got := FrameBytes(cfg); got != 640 {
		t.Fatalf("FrameBytes() = %d, want 640", got)
	}
}

func TestEmitPCMChunks(t *testing.T) {
	cfg := PCMEmitConfig{SampleRate: 16000, BitDepth: 16, Channels: 1, FrameMS: 20}
	pcm := make([]byte, 1500)
	for i := range pcm {
		pcm[i] = byte(i)
	}
	h := &collectHandler{}
	if err := EmitPCMChunks(context.Background(), h, pcm, cfg); err != nil {
		t.Fatal(err)
	}
	if len(h.chunks) != 3 {
		t.Fatalf("chunks = %d, want 3", len(h.chunks))
	}
	if len(h.chunks[0]) != 640 || len(h.chunks[1]) != 640 || len(h.chunks[2]) != 220 {
		t.Fatalf("unexpected chunk sizes: %d %d %d", len(h.chunks[0]), len(h.chunks[1]), len(h.chunks[2]))
	}
}
