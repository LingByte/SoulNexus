package sippersist

import (
	"encoding/binary"

	"github.com/hraban/opus"
)

// MixedOpusRecordingToWav decodes CallSession recording:
// - v2: "SN2" + repeated [dir u8][seq u16LE][ts u32LE][len u16LE][opus frame]
// - v1: "SN1" + repeated [dir u8][len u16LE][opus frame]
// Legacy inbound-only [len u16LE][frame]... is still accepted.
func MixedOpusRecordingToWav(b []byte, sampleRate, decodeChannels int) []byte {
	if len(b) >= 3 && b[0] == 'S' && b[1] == 'N' && b[2] == '2' {
		return opusTaggedFramesV2ToWav(b[3:], sampleRate, decodeChannels)
	}
	if len(b) >= 3 && b[0] == 'S' && b[1] == 'N' && b[2] == '1' {
		return opusTaggedFramesToWav(b[3:], sampleRate, decodeChannels)
	}
	return OpusLengthPrefixedPayloadsToWav(b, sampleRate, decodeChannels)
}

// Recording dir tags must match pkg/sip/session.CallSession (recDirUser=0, recDirAI=1).
const (
	recTagUserLeg byte = 0
	recTagAILeg   byte = 1
)

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
	// When AI is actively speaking, uplink comfort-noise / residual user packets may interleave
	// and make AI sound "stuttered" in offline reconstruction. Keep a short AI priority window.
	const aiPriorityHoldFrames = 8 // about 160ms with 20ms packets
	aiPriorityLeft := 0
	for _, f := range frames {
		if f.dir == recTagAILeg {
			aiPriorityLeft = aiPriorityHoldFrames
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
