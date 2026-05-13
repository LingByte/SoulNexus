package bridge

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
)

func TestCanRawDatagramRelay(t *testing.T) {
	cases := []struct {
		name string
		a, b media.CodecConfig
		want bool
	}{
		{"pcmu_ok", media.CodecConfig{Codec: "pcmu", SampleRate: 8000, Channels: 1}, media.CodecConfig{Codec: "pcmu", SampleRate: 8000, Channels: 1}, true},
		{"pcmu_rate_mismatch", media.CodecConfig{Codec: "pcmu", SampleRate: 8000, Channels: 1}, media.CodecConfig{Codec: "pcmu", SampleRate: 16000, Channels: 1}, false},
		{"pcma_ok", media.CodecConfig{Codec: "pcma", SampleRate: 8000, Channels: 1, PayloadType: 8}, media.CodecConfig{Codec: "pcma", SampleRate: 8000, Channels: 1, PayloadType: 8}, true},
		{"g722_ok", media.CodecConfig{Codec: "g722", SampleRate: 16000, Channels: 1}, media.CodecConfig{Codec: "g722", SampleRate: 16000, Channels: 1}, true},
		{"g722_wrong_rate", media.CodecConfig{Codec: "g722", SampleRate: 8000, Channels: 1}, media.CodecConfig{Codec: "g722", SampleRate: 8000, Channels: 1}, false},
		{"opus_48k", media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 1}, media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 1}, true},
		{"opus_stereo", media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 2}, media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 2}, true},
		{"opus_bad_rate", media.CodecConfig{Codec: "opus", SampleRate: 44100, Channels: 1}, media.CodecConfig{Codec: "opus", SampleRate: 44100, Channels: 1}, false},
		{"l16_8k", media.CodecConfig{Codec: "l16", SampleRate: 8000, Channels: 1}, media.CodecConfig{Codec: "l16", SampleRate: 8000, Channels: 1}, true},
		{"codec_name_mismatch", media.CodecConfig{Codec: "pcmu", SampleRate: 8000, Channels: 1}, media.CodecConfig{Codec: "pcma", SampleRate: 8000, Channels: 1}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanRawDatagramRelay(tc.a, tc.b); got != tc.want {
				t.Fatalf("got %v want %v (a=%v b=%v)", got, tc.want, tc.a, tc.b)
			}
		})
	}
}
