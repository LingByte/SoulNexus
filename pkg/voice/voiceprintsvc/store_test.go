package voiceprintsvc

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

func TestStoreSaveLoadDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	wav := makeTestWAV(t, 1600)
	if err := store.Save("tenant-1", "speaker-a", wav); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load("tenant-1", "speaker-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(wav) {
		t.Fatalf("wav length mismatch: %d vs %d", len(got), len(wav))
	}
	total, err := store.CountAll()
	if err != nil || total != 1 {
		t.Fatalf("count=%d err=%v", total, err)
	}
	if err := store.Delete("tenant-1", "speaker-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load("tenant-1", "speaker-a"); !os.IsNotExist(err) {
		t.Fatalf("expected not exist after delete, got %v", err)
	}
}

func TestCompareWAVScoreIdentical(t *testing.T) {
	wav := makeTestWAV(t, 2400)
	score, err := compareWAVScore(wav, wav)
	if err != nil {
		t.Fatal(err)
	}
	if score < 0.99 {
		t.Fatalf("expected near-perfect score, got %f", score)
	}
}

func makeTestWAV(t *testing.T, sampleCount int) []byte {
	t.Helper()
	dataSize := sampleCount * 2
	pcm := make([]byte, dataSize)
	for i := 0; i < sampleCount; i++ {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16((i*7)%1200))
	}

	var buf bytes.Buffer
	writeString(&buf, "RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize))
	writeString(&buf, "WAVE")
	writeString(&buf, "fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16000))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(32000))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(16))
	writeString(&buf, "data")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(dataSize))
	_, _ = buf.Write(pcm)
	return buf.Bytes()
}

func writeString(buf *bytes.Buffer, s string) {
	_, _ = buf.WriteString(s)
}
