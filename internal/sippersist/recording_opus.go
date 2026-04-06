package sippersist

import (
	"encoding/binary"

	"github.com/hraban/opus"
)

// MixedOpusRecordingToWav decodes CallSession recording: "SN1" + repeated [dir u8][len u16LE][opus frame]
// (user + AI RTP payloads in capture order). Legacy inbound-only [len u16LE][frame]... is still accepted.
func MixedOpusRecordingToWav(b []byte, sampleRate, decodeChannels int) []byte {
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

func opusTaggedFramesToWav(b []byte, sampleRate, decodeChannels int) []byte {
	if len(b) < 3 {
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
	for len(b) >= 3 {
		dir := b[0]
		n := int(binary.LittleEndian.Uint16(b[1:3]))
		b = b[3:]
		if n <= 0 || n > 4000 || len(b) < n {
			break
		}
		pkt := b[:n]
		b = b[n:]
		dec := decUser
		switch dir {
		case recTagUserLeg:
			dec = decUser
		case recTagAILeg:
			dec = decAI
		default:
			continue
		}
		samples, derr := dec.Decode(pkt, pcmScratch)
		if derr != nil || samples <= 0 {
			continue
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
			continue
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
