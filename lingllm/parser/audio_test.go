package parser

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestWAV(t *testing.T, samples []int16, sampleRate int) []byte {
	t.Helper()
	numChannels := 1
	bitsPerSample := 16
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	dataSize := len(samples) * 2
	buf := make([]byte, 44+dataSize)

	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(36+dataSize))
	copy(buf[8:12], "WAVE")
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16)
	binary.LittleEndian.PutUint16(buf[20:22], 1)
	binary.LittleEndian.PutUint16(buf[22:24], uint16(numChannels))
	binary.LittleEndian.PutUint32(buf[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(buf[28:32], uint32(byteRate))
	binary.LittleEndian.PutUint16(buf[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(buf[34:36], uint16(bitsPerSample))
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(buf[44+i*2:46+i*2], uint16(s))
	}
	return buf
}

func TestWAVToPCM_Mono16k(t *testing.T) {
	wav := buildTestWAV(t, []int16{0, 1000, -1000, 2000}, 16000)
	pcm, err := wavToPCM(wav, 16000)
	require.NoError(t, err)
	assert.Equal(t, 16000, pcm.SampleRate)
	assert.Len(t, pcm.Samples, 8)
}

func TestResampleInt16(t *testing.T) {
	in := []int16{0, 100, 200, 300}
	out := resampleInt16(in, 16000, 8000)
	assert.Len(t, out, 2)
	assert.Equal(t, int16(0), out[0])
}

func TestExtractVoskText(t *testing.T) {
	assert.Equal(t, "hello world", extractVoskText(`{"text" : "hello world"}`))
	assert.Equal(t, "", extractVoskText(`{"text" : ""}`))
}

func TestDecodeAudioToPCM_WAV(t *testing.T) {
	wav := buildTestWAV(t, []int16{0, 500, -500}, 16000)
	pcm, err := decodeAudioToPCM(wav, "clip.wav")
	require.NoError(t, err)
	assert.Equal(t, 16000, pcm.SampleRate)
	assert.NotEmpty(t, pcm.Samples)
}
