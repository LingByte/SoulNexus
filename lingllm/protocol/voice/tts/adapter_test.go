package tts

import (
	"context"
	"testing"

	"github.com/LingByte/lingllm/media"
	"github.com/LingByte/lingllm/synthesizer"
)

type stubSynth struct{}

func (stubSynth) Provider() synthesizer.TTSProvider { return synthesizer.ProviderLocal }
func (stubSynth) Format() media.StreamFormat        { return media.StreamFormat{} }
func (stubSynth) CacheKey(string) string            { return "" }
func (stubSynth) Close() error                      { return nil }
func (stubSynth) Synthesize(_ context.Context, h synthesizer.AudioSynthesisHandler, _ string) error {
	h.OnMessage([]byte{1, 2, 3})
	return nil
}

func TestFromSynthesisEngine(t *testing.T) {
	svc := FromSynthesisEngine(stubSynth{})
	var got int
	err := svc.Synthesize(context.Background(), "hi", func(b []byte) error {
		got = len(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != 3 {
		t.Fatalf("got %d bytes", got)
	}
}
