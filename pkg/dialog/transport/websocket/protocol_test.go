package websocket

import (
	"testing"

	"github.com/LingByte/SoulNexus/pkg/dialog/engine"
	"github.com/LingByte/SoulNexus/pkg/dialog/transport/pcm"
)

func TestNewPort_BinaryPCMOutput(t *testing.T) {
	var got []byte
	port := NewPort(pcm.Config{SessionID: "s1", SampleRate: 16000}, WireWriter{
		WriteBinary: func(b []byte) error {
			got = append([]byte(nil), b...)
			return nil
		},
	})
	raw := []byte{0x01, 0x02, 0x03, 0x04}
	if err := port.SendOutputPCM(engine.PCMFrame{Data: raw, SampleRate: 16000}); err != nil {
		t.Fatal(err)
	}
	if string(got) != string(raw) {
		t.Fatalf("binary out=%v want %v", got, raw)
	}
}

func TestPushInputFromBinary(t *testing.T) {
	port := pcm.NewPort(pcm.Config{SessionID: "s1", SampleRate: 16000})
	raw := []byte{0x00, 0x01, 0x02, 0x03}
	if err := PushInputFromBinary(port, raw); err != nil {
		t.Fatal(err)
	}
	select {
	case fr := <-port.InputPCM():
		if string(fr.Data) != string(raw) {
			t.Fatalf("data=%v", fr.Data)
		}
	default:
		t.Fatal("expected frame on input")
	}
}
