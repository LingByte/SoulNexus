package sippersist

import (
	"bytes"
	"encoding/binary"
	"strings"
)

// G711TaggedRecordingToWav decodes "SN1" + [dir u8][len u16LE][pcmu/pcma payload]...; otherwise raw G.711 pass-through.
func G711TaggedRecordingToWav(b []byte, codec string) []byte {
	if len(b) >= 3 && b[0] == 'S' && b[1] == 'N' && b[2] == '1' {
		return g711TaggedFramesToPcmWav(b[3:], codec)
	}
	return G711PayloadsToWav(b, codec)
}

func g711TaggedFramesToPcmWav(b []byte, codec string) []byte {
	c := strings.ToLower(strings.TrimSpace(codec))
	var pcm []int16
	for len(b) >= 3 {
		_ = b[0]
		n := int(binary.LittleEndian.Uint16(b[1:3]))
		b = b[3:]
		if n <= 0 || n > 2000 || len(b) < n {
			break
		}
		chunk := b[:n]
		b = b[n:]
		if strings.Contains(c, "pcma") {
			pcm = append(pcm, decodeALaw(chunk)...)
		} else {
			pcm = append(pcm, decodeMuLaw(chunk)...)
		}
	}
	return pcm16MonoToWav(pcm, 8000)
}

// G711PayloadsToWav builds 8kHz mono WAV from concatenated PCMU or PCMA RTP payloads.
func G711PayloadsToWav(payloads []byte, codec string) []byte {
	if len(payloads) == 0 {
		return nil
	}
	c := strings.ToLower(strings.TrimSpace(codec))
	var pcm []int16
	if strings.Contains(c, "pcma") {
		pcm = decodeALaw(payloads)
	} else {
		pcm = decodeMuLaw(payloads)
	}
	return pcm16MonoToWav(pcm, 8000)
}

func decodeMuLaw(in []byte) []int16 {
	out := make([]int16, len(in))
	for i, b := range in {
		out[i] = ulawToLinear(b)
	}
	return out
}

func decodeALaw(in []byte) []int16 {
	out := make([]int16, len(in))
	for i, b := range in {
		out[i] = alawToLinear(b)
	}
	return out
}

func ulawToLinear(u uint8) int16 {
	u = ^u
	sign := (u & 0x80)
	exponent := (u >> 4) & 0x07
	mantissa := u & 0x0F
	sample := int32(mantissa<<4) + 0x08
	sample <<= uint(exponent + 3)
	sample -= 0x84
	if sign != 0 {
		sample = -sample
	}
	if sample > 32767 {
		sample = 32767
	}
	if sample < -32768 {
		sample = -32768
	}
	return int16(sample)
}

func alawToLinear(a uint8) int16 {
	a ^= 0x55
	t := int32((a & 0x0F) << 4)
	seg := (a >> 4) & 0x07
	switch seg {
	case 0:
		t += 8
	case 1:
		t += 0x108
	default:
		t += 0x108
		t <<= uint(seg - 1)
	}
	if a&0x80 == 0 {
		t = -t
	}
	if t > 32767 {
		t = 32767
	}
	if t < -32768 {
		t = -32768
	}
	return int16(t)
}

func pcm16MonoToWav(samples []int16, sampleRate int) []byte {
	if len(samples) == 0 {
		return nil
	}
	dataSize := len(samples) * 2
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
	for _, s := range samples {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}
	return buf.Bytes()
}
