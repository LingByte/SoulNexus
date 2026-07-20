package preview

import (
	"bytes"
	"encoding/binary"
)

// PCM16LEToWav wraps mono PCM16LE bytes as a WAV file.
func PCM16LEToWav(pcm []byte, sampleRate int) []byte {
	if len(pcm) < 2 || sampleRate <= 0 {
		return nil
	}
	if len(pcm)%2 != 0 {
		pcm = pcm[:len(pcm)-1]
	}
	n := len(pcm) / 2
	dataSize := n * 2
	buf := &bytes.Buffer{}
	_, _ = buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+dataSize))
	_, _ = buf.WriteString("WAVE")
	_, _ = buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate*2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))
	_, _ = buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataSize))
	_, _ = buf.Write(pcm)
	return buf.Bytes()
}
