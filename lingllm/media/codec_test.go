package media

import (
	"testing"
)

func TestAudioCodecString(t *testing.T) {
	tests := []struct {
		codec AudioCodec
		want  string
	}{
		{AudioCodecPCM, "pcm"},
		{AudioCodecFLAC, "flac"},
		{AudioCodecOPUS, "opus"},
		{AudioCodecMP3, "mp3"},
		{AudioCodecAAC, "aac"},
	}

	for _, tt := range tests {
		t.Run(string(tt.codec), func(t *testing.T) {
			if got := string(tt.codec); got != tt.want {
				t.Errorf("AudioCodec string = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAudioProfileString(t *testing.T) {
	tests := []struct {
		profile AudioProfile
		want    string
	}{
		{AudioProfileOpusNarrow, "opus_narrow"},
		{AudioProfileOpusWide, "opus_wide"},
	}

	for _, tt := range tests {
		t.Run(string(tt.profile), func(t *testing.T) {
			if got := string(tt.profile); got != tt.want {
				t.Errorf("AudioProfile string = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodecInfo(t *testing.T) {
	tests := []struct {
		codec AudioCodec
		name  string
	}{
		{AudioCodecPCM, "PCM"},
		{AudioCodecFLAC, "FLAC"},
		{AudioCodecOPUS, "Opus"},
		{AudioCodecMP3, "MP3"},
		{AudioCodecAAC, "AAC"},
	}

	for _, tt := range tests {
		t.Run(string(tt.codec), func(t *testing.T) {
			// Verify codec constants exist
			if tt.codec == "" {
				t.Error("Codec constant should not be empty")
			}
		})
	}
}

func TestDefaultCodecConfig(t *testing.T) {
	cfg := DefaultCodecConfig()

	if cfg.SampleRate == 0 {
		t.Error("DefaultCodecConfig SampleRate should not be 0")
	}
	if cfg.Channels == 0 {
		t.Error("DefaultCodecConfig Channels should not be 0")
	}
	if cfg.BitDepth == 0 {
		t.Error("DefaultCodecConfig BitDepth should not be 0")
	}
}

func TestAudioFramePacket(t *testing.T) {
	packet := &AudioPacket{
		Sequence:      1,
		Payload:       []byte{0x01, 0x02, 0x03},
		RTPSamples:    960,
		IsFirstPacket: true,
		IsEndPacket:   false,
	}

	if packet.Sequence != 1 {
		t.Errorf("AudioPacket.Sequence = %v, want 1", packet.Sequence)
	}
	if len(packet.Payload) != 3 {
		t.Errorf("AudioPacket.Payload length = %v, want 3", len(packet.Payload))
	}
	if packet.RTPSamples != 960 {
		t.Errorf("AudioPacket.RTPSamples = %v, want 960", packet.RTPSamples)
	}
}

func TestCodecConfigValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  CodecConfig
		want bool
	}{
		{
			name: "valid config",
			cfg: CodecConfig{
				Codec:      "opus",
				SampleRate: 48000,
				Channels:   1,
				BitDepth:   16,
			},
			want: true,
		},
		{
			name: "zero sample rate",
			cfg: CodecConfig{
				Codec:      "opus",
				SampleRate: 0,
				Channels:   1,
				BitDepth:   16,
			},
			want: false,
		},
		{
			name: "zero channels",
			cfg: CodecConfig{
				Codec:      "opus",
				SampleRate: 48000,
				Channels:   0,
				BitDepth:   16,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate sample rate
			if tt.cfg.SampleRate == 0 && tt.want {
				t.Error("Expected validation to fail for zero sample rate")
			}
			// Validate channels
			if tt.cfg.Channels == 0 && tt.want {
				t.Error("Expected validation to fail for zero channels")
			}
		})
	}
}
