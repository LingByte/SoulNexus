package media

import (
	"testing"
)

func TestAudioPacket(t *testing.T) {
	packet := &AudioPacket{
		Sequence:      1,
		Payload:       []byte{0x01, 0x02},
		RTPSamples:    960,
		IsFirstPacket: true,
		IsEndPacket:   false,
		IsSynthesized: false,
	}

	if packet.Sequence != 1 {
		t.Errorf("Sequence = %v, want 1", packet.Sequence)
	}
	if len(packet.Payload) != 2 {
		t.Errorf("Payload length = %v, want 2", len(packet.Payload))
	}
	if packet.RTPSamples != 960 {
		t.Errorf("RTPSamples = %v, want 960", packet.RTPSamples)
	}
}

func TestClosePacket(t *testing.T) {
	packet := &ClosePacket{
		Reason: "test close",
	}

	if packet.Reason != "test close" {
		t.Errorf("Reason = %v, want 'test close'", packet.Reason)
	}
}

func TestStateChange(t *testing.T) {
	state := StateChange{
		State:  "begin",
		Params: []interface{}{"param1", 123},
	}

	if state.State != "begin" {
		t.Errorf("State = %v, want 'begin'", state.State)
	}
	if len(state.Params) != 2 {
		t.Errorf("Params length = %v, want 2", len(state.Params))
	}
}

func TestCodecConfig(t *testing.T) {
	cfg := CodecConfig{
		Codec:      "opus",
		SampleRate: 48000,
		Channels:   1,
		BitDepth:   16,
	}

	if cfg.Codec != "opus" {
		t.Errorf("Codec = %v, want 'opus'", cfg.Codec)
	}
	if cfg.SampleRate != 48000 {
		t.Errorf("SampleRate = %v, want 48000", cfg.SampleRate)
	}
	if cfg.Channels != 1 {
		t.Errorf("Channels = %v, want 1", cfg.Channels)
	}
}
