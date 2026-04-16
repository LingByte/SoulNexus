// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// SIP call recording: tagged G.711/G.722/Opus payloads → mono or stereo WAV (sip call persistence).
package utils

import (
	"bytes"
	"encoding/binary"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/media/encoder"
	"github.com/hraban/opus"
)

// Recording dir tags must match pkg/sip/session.CallSession (recDirUser=0, recDirAI=1).
const (
	recTagUserLeg byte = 0
	recTagAILeg   byte = 1
)

// recordingMonoDuckUserFrames: mono-mix decoders used to drop uplink for N frames after each AI
// packet to reduce interleave stutter; for call archival that removes the caller whenever TTS plays.
// Keep at 0 so mono/stereo WAV reconstructions retain both legs.
const recordingMonoDuckUserFrames = 0

// G711TaggedRecordingToWav decodes tagged SIP recordings:
// - v2: "SN2" + [dir u8][seq u16LE][ts u32LE][len u16LE][payload]
// - v1: "SN1" + [dir u8][len u16LE][payload]
// otherwise raw G.711 pass-through.
func G711TaggedRecordingToWav(b []byte, codec string) []byte {
	if len(b) >= 3 && b[0] == 'S' && b[1] == 'N' && b[2] == '2' {
		return g711TaggedFramesV2ToPcmWav(b[3:], codec)
	}
	if len(b) >= 3 && b[0] == 'S' && b[1] == 'N' && b[2] == '1' {
		return g711TaggedFramesToPcmWav(b[3:], codec)
	}
	return G711PayloadsToWav(b, codec)
}

func g711TaggedFramesV2ToPcmWav(b []byte, codec string) []byte {
	c := strings.ToLower(strings.TrimSpace(codec))
	var pcm []int16
	aiPriorityLeft := 0
	for len(b) >= 9 {
		dir := b[0]
		n := int(binary.LittleEndian.Uint16(b[7:9]))
		b = b[9:]
		if n <= 0 || n > 2000 || len(b) < n {
			break
		}
		chunk := b[:n]
		b = b[n:]
		if dir == 1 {
			aiPriorityLeft = recordingMonoDuckUserFrames
		} else if aiPriorityLeft > 0 {
			aiPriorityLeft--
			continue
		}
		var decoded []int16
		if strings.Contains(c, "pcma") {
			decoded = decodeALaw(chunk)
		} else {
			decoded = decodeMuLaw(chunk)
		}
		pcm = append(pcm, decoded...)
	}
	return pcm16MonoToWav(pcm, 8000)
}

func g711TaggedFramesToPcmWav(b []byte, codec string) []byte {
	c := strings.ToLower(strings.TrimSpace(codec))
	var pcm []int16
	aiPriorityLeft := 0
	for len(b) >= 3 {
		dir := b[0]
		n := int(binary.LittleEndian.Uint16(b[1:3]))
		b = b[3:]
		if n <= 0 || n > 2000 || len(b) < n {
			break
		}
		chunk := b[:n]
		b = b[n:]
		if dir == 1 {
			aiPriorityLeft = recordingMonoDuckUserFrames
		} else if aiPriorityLeft > 0 {
			aiPriorityLeft--
			continue
		}
		var decoded []int16
		if strings.Contains(c, "pcma") {
			decoded = decodeALaw(chunk)
		} else {
			decoded = decodeMuLaw(chunk)
		}
		pcm = append(pcm, decoded...)
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

// G722TaggedRecordingToWav decodes SN2/SN1-tagged G.722 RTP payloads to 16 kHz mono WAV (wideband).
func G722TaggedRecordingToWav(b []byte) []byte {
	if len(b) >= 3 && b[0] == 'S' && b[1] == 'N' && b[2] == '2' {
		return g722TaggedFramesV2ToPcmWav(b[3:])
	}
	if len(b) >= 3 && b[0] == 'S' && b[1] == 'N' && b[2] == '1' {
		return g722TaggedFramesV1ToPcmWav(b[3:])
	}
	return g722RawPayloadsToWav(b)
}

func g722PCMBytesToMono16(pcmBytes []byte) []int16 {
	out := make([]int16, 0, len(pcmBytes)/4)
	for i := 0; i+3 < len(pcmBytes); i += 4 {
		s1 := int16(binary.LittleEndian.Uint16(pcmBytes[i:]))
		s2 := int16(binary.LittleEndian.Uint16(pcmBytes[i+2:]))
		out = append(out, int16((int32(s1)+int32(s2))/2))
	}
	return out
}

func g722TaggedFramesV2ToPcmWav(b []byte) []byte {
	decUser := encoder.NewG722Decoder(encoder.G722_RATE_DEFAULT, encoder.G722_DEFAULT)
	decAI := encoder.NewG722Decoder(encoder.G722_RATE_DEFAULT, encoder.G722_DEFAULT)
	var pcm []int16
	aiPriorityLeft := 0
	for len(b) >= 9 {
		dir := b[0]
		n := int(binary.LittleEndian.Uint16(b[7:9]))
		b = b[9:]
		if n <= 0 || n > 2000 || len(b) < n {
			break
		}
		chunk := b[:n]
		b = b[n:]
		if dir == recTagAILeg {
			aiPriorityLeft = recordingMonoDuckUserFrames
		} else if aiPriorityLeft > 0 {
			aiPriorityLeft--
			continue
		}
		dec := decUser
		switch dir {
		case recTagUserLeg:
			dec = decUser
		case recTagAILeg:
			dec = decAI
		default:
			continue
		}
		pcmBytes := dec.Decode(chunk)
		pcm = append(pcm, g722PCMBytesToMono16(pcmBytes)...)
	}
	return pcm16MonoToWav(pcm, 16000)
}

func g722TaggedFramesV1ToPcmWav(b []byte) []byte {
	decUser := encoder.NewG722Decoder(encoder.G722_RATE_DEFAULT, encoder.G722_DEFAULT)
	decAI := encoder.NewG722Decoder(encoder.G722_RATE_DEFAULT, encoder.G722_DEFAULT)
	var pcm []int16
	aiPriorityLeft := 0
	for len(b) >= 3 {
		dir := b[0]
		n := int(binary.LittleEndian.Uint16(b[1:3]))
		b = b[3:]
		if n <= 0 || n > 2000 || len(b) < n {
			break
		}
		chunk := b[:n]
		b = b[n:]
		if dir == recTagAILeg {
			aiPriorityLeft = recordingMonoDuckUserFrames
		} else if aiPriorityLeft > 0 {
			aiPriorityLeft--
			continue
		}
		dec := decUser
		switch dir {
		case recTagUserLeg:
			dec = decUser
		case recTagAILeg:
			dec = decAI
		default:
			continue
		}
		pcmBytes := dec.Decode(chunk)
		pcm = append(pcm, g722PCMBytesToMono16(pcmBytes)...)
	}
	return pcm16MonoToWav(pcm, 16000)
}

func g722RawPayloadsToWav(payloads []byte) []byte {
	if len(payloads) == 0 {
		return nil
	}
	dec := encoder.NewG722Decoder(encoder.G722_RATE_DEFAULT, encoder.G722_DEFAULT)
	return pcm16MonoToWav(g722PCMBytesToMono16(dec.Decode(payloads)), 16000)
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

// pcm16StereoInterleavedToWav builds 16-bit stereo WAV from L,R,L,R,... interleaved samples.
func pcm16StereoInterleavedToWav(lr []int16, sampleRate int) []byte {
	if len(lr) == 0 || len(lr)%2 != 0 {
		return nil
	}
	dataSize := len(lr) * 2
	buf := &bytes.Buffer{}
	_, _ = buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+dataSize))
	_, _ = buf.WriteString("WAVE")
	_, _ = buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1)) // PCM
	_ = binary.Write(buf, binary.LittleEndian, uint16(2)) // stereo
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(uint32(sampleRate)*4)) // byte rate
	_ = binary.Write(buf, binary.LittleEndian, uint16(4))                    // block align
	_ = binary.Write(buf, binary.LittleEndian, uint16(16))
	_, _ = buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataSize))
	for _, s := range lr {
		_ = binary.Write(buf, binary.LittleEndian, s)
	}
	return buf.Bytes()
}

func g711DecodeChunk(codec string, chunk []byte) []int16 {
	c := strings.ToLower(strings.TrimSpace(codec))
	if strings.Contains(c, "pcma") {
		return decodeALaw(chunk)
	}
	return decodeMuLaw(chunk)
}

// G711TaggedRecordingToStereoWav builds stereo WAV: L=user leg, R=AI leg (SN2 only).
// Each channel concatenates that leg’s RTP payloads in capture order (no mono ducking).
func G711TaggedRecordingToStereoWav(b []byte, codec string) []byte {
	if len(b) < 3 || b[0] != 'S' || b[1] != 'N' || b[2] != '2' {
		return nil
	}
	var userPCM, aiPCM []int16
	body := b[3:]
	for len(body) >= 9 {
		dir := body[0]
		n := int(binary.LittleEndian.Uint16(body[7:9]))
		body = body[9:]
		if n <= 0 || n > 2000 || len(body) < n {
			break
		}
		chunk := body[:n]
		body = body[n:]
		dec := g711DecodeChunk(codec, chunk)
		switch dir {
		case recTagUserLeg:
			userPCM = append(userPCM, dec...)
		case recTagAILeg:
			aiPCM = append(aiPCM, dec...)
		}
	}
	if len(userPCM) == 0 && len(aiPCM) == 0 {
		return nil
	}
	n := len(userPCM)
	if len(aiPCM) > n {
		n = len(aiPCM)
	}
	lr := make([]int16, 2*n)
	for i := 0; i < len(userPCM); i++ {
		lr[2*i] = userPCM[i]
	}
	for i := 0; i < len(aiPCM); i++ {
		lr[2*i+1] = aiPCM[i]
	}
	return pcm16StereoInterleavedToWav(lr, 8000)
}

// G722TaggedRecordingToStereoWav is like G711 stereo: L=user, R=AI, SN2 only.
func G722TaggedRecordingToStereoWav(b []byte) []byte {
	if len(b) < 3 || b[0] != 'S' || b[1] != 'N' || b[2] != '2' {
		return nil
	}
	decUser := encoder.NewG722Decoder(encoder.G722_RATE_DEFAULT, encoder.G722_DEFAULT)
	decAI := encoder.NewG722Decoder(encoder.G722_RATE_DEFAULT, encoder.G722_DEFAULT)
	var userPCM, aiPCM []int16
	body := b[3:]
	for len(body) >= 9 {
		dir := body[0]
		n := int(binary.LittleEndian.Uint16(body[7:9]))
		body = body[9:]
		if n <= 0 || n > 2000 || len(body) < n {
			break
		}
		chunk := body[:n]
		body = body[n:]
		var dec *encoder.G722Decoder
		switch dir {
		case recTagUserLeg:
			dec = decUser
		case recTagAILeg:
			dec = decAI
		default:
			continue
		}
		pcmBytes := dec.Decode(chunk)
		m := g722PCMBytesToMono16(pcmBytes)
		switch dir {
		case recTagUserLeg:
			userPCM = append(userPCM, m...)
		case recTagAILeg:
			aiPCM = append(aiPCM, m...)
		}
	}
	if len(userPCM) == 0 && len(aiPCM) == 0 {
		return nil
	}
	n := len(userPCM)
	if len(aiPCM) > n {
		n = len(aiPCM)
	}
	lr := make([]int16, 2*n)
	for i := 0; i < len(userPCM); i++ {
		lr[2*i] = userPCM[i]
	}
	for i := 0; i < len(aiPCM); i++ {
		lr[2*i+1] = aiPCM[i]
	}
	return pcm16StereoInterleavedToWav(lr, 16000)
}

// MixedOpusRecordingToStereoWav decodes SN2 Opus per leg (separate decoders) into L=user R=AI stereo WAV.
func MixedOpusRecordingToStereoWav(b []byte, sampleRate, decodeChannels int) []byte {
	if len(b) < 3 || b[0] != 'S' || b[1] != 'N' || b[2] != '2' {
		return nil
	}
	var frames []taggedOpusFrame
	body := b[3:]
	for len(body) >= 9 {
		dir := body[0]
		seq := binary.LittleEndian.Uint16(body[1:3])
		n := int(binary.LittleEndian.Uint16(body[7:9]))
		body = body[9:]
		if n <= 0 || n > 4000 || len(body) < n {
			break
		}
		pkt := make([]byte, n)
		copy(pkt, body[:n])
		body = body[n:]
		if dir == recTagUserLeg || dir == recTagAILeg {
			frames = append(frames, taggedOpusFrame{dir: dir, seq: seq, payload: pkt})
		}
	}
	return opusFramesPerLegStereoWav(frames, sampleRate, decodeChannels)
}

func opusFramesPerLegStereoWav(frames []taggedOpusFrame, sampleRate, decodeChannels int) []byte {
	if len(frames) == 0 {
		return nil
	}
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	if decodeChannels < 1 {
		decodeChannels = 1
	}
	if decodeChannels > 2 {
		decodeChannels = 2
	}
	decUser, errU := opus.NewDecoder(sampleRate, decodeChannels)
	decAI, errA := opus.NewDecoder(sampleRate, decodeChannels)
	if errU != nil || errA != nil {
		return nil
	}
	maxSamples := sampleRate * 120 / 1000
	if maxSamples < 960 {
		maxSamples = 960
	}
	pcmScratch := make([]int16, maxSamples*decodeChannels)

	decodeChain := func(dec *opus.Decoder, chain []taggedOpusFrame) []int16 {
		var out []int16
		var lastSeq uint16
		var hasSeq bool
		for _, f := range chain {
			if hasSeq {
				gap := int(uint16(f.seq - lastSeq))
				if gap > 1 && gap < 20 {
					for i := 0; i < gap-1; i++ {
						if plc, err := dec.Decode(nil, pcmScratch); err == nil && plc > 0 {
							out = appendOpusPCMSlice(out, pcmScratch, plc, decodeChannels)
						}
					}
				}
			}
			lastSeq = f.seq
			hasSeq = true
			samples, derr := dec.Decode(f.payload, pcmScratch)
			if derr != nil || samples <= 0 {
				plc, err2 := dec.Decode(nil, pcmScratch)
				if err2 == nil && plc > 0 {
					samples = plc
				} else {
					continue
				}
			}
			out = appendOpusPCMSlice(out, pcmScratch, samples, decodeChannels)
		}
		return out
	}

	var userFrames, aiFrames []taggedOpusFrame
	for _, f := range frames {
		switch f.dir {
		case recTagUserLeg:
			userFrames = append(userFrames, f)
		case recTagAILeg:
			aiFrames = append(aiFrames, f)
		}
	}
	userPCM := decodeChain(decUser, userFrames)
	aiPCM := decodeChain(decAI, aiFrames)
	if len(userPCM) == 0 && len(aiPCM) == 0 {
		return nil
	}
	n := len(userPCM)
	if len(aiPCM) > n {
		n = len(aiPCM)
	}
	lr := make([]int16, 2*n)
	for i := 0; i < len(userPCM); i++ {
		lr[2*i] = userPCM[i]
	}
	for i := 0; i < len(aiPCM); i++ {
		lr[2*i+1] = aiPCM[i]
	}
	return pcm16StereoInterleavedToWav(lr, sampleRate)
}

func appendOpusPCMSlice(dst []int16, pcmScratch []int16, samples int, decodeChannels int) []int16 {
	if decodeChannels == 2 {
		for i := 0; i < samples; i++ {
			L := int32(pcmScratch[i*2])
			R := int32(pcmScratch[i*2+1])
			dst = append(dst, int16((L+R)/2))
		}
		return dst
	}
	return append(dst, pcmScratch[:samples]...)
}

func MixedOpusRecordingToWav(b []byte, sampleRate, decodeChannels int) []byte {
	if len(b) >= 3 && b[0] == 'S' && b[1] == 'N' && b[2] == '2' {
		return opusTaggedFramesV2ToWav(b[3:], sampleRate, decodeChannels)
	}
	if len(b) >= 3 && b[0] == 'S' && b[1] == 'N' && b[2] == '1' {
		return opusTaggedFramesToWav(b[3:], sampleRate, decodeChannels)
	}
	return OpusLengthPrefixedPayloadsToWav(b, sampleRate, decodeChannels)
}

type taggedOpusFrame struct {
	dir     byte
	seq     uint16
	payload []byte
}

func opusTaggedFramesV2ToWav(b []byte, sampleRate, decodeChannels int) []byte {
	if len(b) < 9 {
		return nil
	}
	var frames []taggedOpusFrame
	for len(b) >= 9 {
		dir := b[0]
		seq := binary.LittleEndian.Uint16(b[1:3])
		n := int(binary.LittleEndian.Uint16(b[7:9]))
		b = b[9:]
		if n <= 0 || n > 4000 || len(b) < n {
			break
		}
		pkt := make([]byte, n)
		copy(pkt, b[:n])
		b = b[n:]
		if dir != recTagUserLeg && dir != recTagAILeg {
			continue
		}
		frames = append(frames, taggedOpusFrame{dir: dir, seq: seq, payload: pkt})
	}
	return decodeTaggedOpusFrames(frames, sampleRate, decodeChannels, true)
}

func opusTaggedFramesToWav(b []byte, sampleRate, decodeChannels int) []byte {
	if len(b) < 3 {
		return nil
	}
	var frames []taggedOpusFrame
	for len(b) >= 3 {
		dir := b[0]
		n := int(binary.LittleEndian.Uint16(b[1:3]))
		b = b[3:]
		if n <= 0 || n > 4000 || len(b) < n {
			break
		}
		pkt := make([]byte, n)
		copy(pkt, b[:n])
		b = b[n:]
		if dir != recTagUserLeg && dir != recTagAILeg {
			continue
		}
		frames = append(frames, taggedOpusFrame{dir: dir, payload: pkt})
	}
	return decodeTaggedOpusFrames(frames, sampleRate, decodeChannels, false)
}

func decodeTaggedOpusFrames(frames []taggedOpusFrame, sampleRate, decodeChannels int, withSeq bool) []byte {
	if len(frames) == 0 {
		return nil
	}
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	if decodeChannels < 1 {
		decodeChannels = 1
	}
	if decodeChannels > 2 {
		decodeChannels = 2
	}
	// User uplink and AI/TTS downlink are independent RTP streams; a single OpusDecoder
	// keeps cross-packet state, so alternating legs corrupts decode (often the AI leg).
	decUser, errU := opus.NewDecoder(sampleRate, decodeChannels)
	decAI, errA := opus.NewDecoder(sampleRate, decodeChannels)
	if errU != nil || errA != nil {
		return nil
	}
	maxSamples := sampleRate * 120 / 1000
	if maxSamples < 960 {
		maxSamples = 960
	}
	pcmScratch := make([]int16, maxSamples*decodeChannels)
	var mono []int16
	defaultFrameSamples := sampleRate / 50 // 20ms
	if defaultFrameSamples <= 0 {
		defaultFrameSamples = 960
	}
	var lastSeqUser uint16
	var lastSeqAI uint16
	var hasSeqUser bool
	var hasSeqAI bool
	// archival: keep all uplink (see recordingMonoDuckUserFrames).
	aiPriorityLeft := 0
	for _, f := range frames {
		if f.dir == recTagAILeg {
			aiPriorityLeft = recordingMonoDuckUserFrames
		} else if aiPriorityLeft > 0 {
			aiPriorityLeft--
			// Skip uplink frames during short AI-active window to avoid fragmenting AI audio.
			continue
		}

		dec := decUser
		switch f.dir {
		case recTagUserLeg:
			dec = decUser
		case recTagAILeg:
			dec = decAI
		default:
			continue
		}
		if withSeq {
			var gap int
			if f.dir == recTagUserLeg {
				if hasSeqUser {
					gap = int(uint16(f.seq - lastSeqUser))
					if gap > 1 && gap < 20 {
						for i := 0; i < gap-1; i++ {
							if plc, err := dec.Decode(nil, pcmScratch); err == nil && plc > 0 {
								if decodeChannels == 2 {
									for j := 0; j < plc; j++ {
										L := int32(pcmScratch[j*2])
										R := int32(pcmScratch[j*2+1])
										mono = append(mono, int16((L+R)/2))
									}
								} else {
									mono = append(mono, pcmScratch[:plc]...)
								}
							}
						}
					}
				}
				lastSeqUser = f.seq
				hasSeqUser = true
			} else {
				if hasSeqAI {
					gap = int(uint16(f.seq - lastSeqAI))
					if gap > 1 && gap < 20 {
						for i := 0; i < gap-1; i++ {
							if plc, err := dec.Decode(nil, pcmScratch); err == nil && plc > 0 {
								if decodeChannels == 2 {
									for j := 0; j < plc; j++ {
										L := int32(pcmScratch[j*2])
										R := int32(pcmScratch[j*2+1])
										mono = append(mono, int16((L+R)/2))
									}
								} else {
									mono = append(mono, pcmScratch[:plc]...)
								}
							}
						}
					}
				}
				lastSeqAI = f.seq
				hasSeqAI = true
			}
		}
		samples, derr := dec.Decode(f.payload, pcmScratch)
		if derr != nil || samples <= 0 {
			// Packet loss concealment: use decoder PLC to keep timeline continuous.
			plcSamples, plcErr := dec.Decode(nil, pcmScratch)
			if plcErr == nil && plcSamples > 0 {
				samples = plcSamples
			} else {
				samples = defaultFrameSamples
				if samples > maxSamples {
					samples = maxSamples
				}
				// PLC unavailable; fill with silence to avoid timeline collapse.
				for i := 0; i < samples*decodeChannels; i++ {
					pcmScratch[i] = 0
				}
			}
		}
		if decodeChannels == 2 {
			for i := 0; i < samples; i++ {
				L := int32(pcmScratch[i*2])
				R := int32(pcmScratch[i*2+1])
				mono = append(mono, int16((L+R)/2))
			}
		} else {
			mono = append(mono, pcmScratch[:samples]...)
		}
	}
	return pcm16MonoToWav(mono, sampleRate)
}

// OpusLengthPrefixedPayloadsToWav decodes legacy inbound-only [uint16LE len][opus frame]... to mono WAV.
func OpusLengthPrefixedPayloadsToWav(b []byte, sampleRate, decodeChannels int) []byte {
	if len(b) < 2 {
		return nil
	}
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	if decodeChannels < 1 {
		decodeChannels = 1
	}
	if decodeChannels > 2 {
		decodeChannels = 2
	}
	dec, err := opus.NewDecoder(sampleRate, decodeChannels)
	if err != nil {
		return nil
	}
	maxSamples := sampleRate * 120 / 1000
	if maxSamples < 960 {
		maxSamples = 960
	}
	pcmScratch := make([]int16, maxSamples*decodeChannels)
	var mono []int16
	defaultFrameSamples := sampleRate / 50 // 20ms
	if defaultFrameSamples <= 0 {
		defaultFrameSamples = 960
	}
	for len(b) >= 2 {
		n := int(binary.LittleEndian.Uint16(b[:2]))
		b = b[2:]
		if n <= 0 || n > 4000 || len(b) < n {
			break
		}
		pkt := b[:n]
		b = b[n:]
		samples, derr := dec.Decode(pkt, pcmScratch)
		if derr != nil || samples <= 0 {
			// Packet loss concealment for legacy stream.
			plcSamples, plcErr := dec.Decode(nil, pcmScratch)
			if plcErr == nil && plcSamples > 0 {
				samples = plcSamples
			} else {
				samples = defaultFrameSamples
				if samples > maxSamples {
					samples = maxSamples
				}
				for i := 0; i < samples*decodeChannels; i++ {
					pcmScratch[i] = 0
				}
			}
		}
		if decodeChannels == 2 {
			for i := 0; i < samples; i++ {
				L := int32(pcmScratch[i*2])
				R := int32(pcmScratch[i*2+1])
				mono = append(mono, int16((L+R)/2))
			}
		} else {
			mono = append(mono, pcmScratch[:samples]...)
		}
	}
	return pcm16MonoToWav(mono, sampleRate)
}
