package session

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/media"
)

func TestInternalPCMSampleRate_PCMU(t *testing.T) {
	src := media.CodecConfig{Codec: "pcmu", SampleRate: 8000}
	if got := InternalPCMSampleRate(src); got != 8000 {
		t.Fatalf("InternalPCMSampleRate(pcmu)=%d want 8000", got)
	}
}

func TestInternalPCMSampleRate_OPUS48k(t *testing.T) {
	src := media.CodecConfig{Codec: "opus", SampleRate: 48000}
	if got := InternalPCMSampleRate(src); got != 48000 {
		t.Fatalf("InternalPCMSampleRate(opus 48k)=%d want 48000", got)
	}
}

func TestInternalPCMSampleRate_G722(t *testing.T) {
	src := media.CodecConfig{Codec: "g722", SampleRate: 16000}
	if got := InternalPCMSampleRate(src); got != 16000 {
		t.Fatalf("InternalPCMSampleRate(g722)=%d want 16000", got)
	}
}
