// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package audutil

import (
	"fmt"

	"github.com/hraban/opus"
)

// SOA1ToWAV decodes a SOA1 Opus archive to mono PCM16 WAV, placing frames on the wall-clock timeline.
func SOA1ToWAV(soa1 []byte) ([]byte, error) {
	h, frames, err := ParseSOA1(soa1)
	if err != nil {
		return nil, err
	}
	if len(frames) == 0 {
		return nil, errEmptySOA1
	}
	sr := int(h.SampleRate)
	ch := int(h.Channels)
	if ch < 1 {
		ch = 1
	}
	if ch > 2 {
		ch = 2
	}
	segs, err := decodeSOA1FramesWall(frames, sr, ch)
	if err != nil {
		return nil, err
	}
	pcm := placeWallPCMTrack(segs, sr)
	if len(pcm) == 0 {
		return nil, fmt.Errorf("audutil: decoded empty PCM")
	}
	return pcm16MonoToWav(pcm, sr), nil
}

// MergeSOA1StereoWAV decodes caller + AI SOA1 archives into L/R stereo WAV.
func MergeSOA1StereoWAV(callerSOA1, aiSOA1 []byte) ([]byte, error) {
	var leftSegs, rightSegs []wallPCMSeg
	var sr int
	if len(callerSOA1) > 0 {
		h, frames, err := ParseSOA1(callerSOA1)
		if err != nil {
			return nil, err
		}
		sr = int(h.SampleRate)
		ch := int(h.Channels)
		if ch < 1 {
			ch = 1
		}
		segs, err := decodeSOA1FramesWall(frames, sr, ch)
		if err != nil {
			return nil, err
		}
		leftSegs = segs
	}
	if len(aiSOA1) > 0 {
		h, frames, err := ParseSOA1(aiSOA1)
		if err != nil {
			return nil, err
		}
		if sr <= 0 {
			sr = int(h.SampleRate)
		}
		ch := int(h.Channels)
		if ch < 1 {
			ch = 1
		}
		segs, err := decodeSOA1FramesWall(frames, sr, ch)
		if err != nil {
			return nil, err
		}
		rightSegs = segs
	}
	if sr <= 0 {
		sr = 48000
	}
	base, ok := minWallNs(leftSegs, rightSegs)
	if !ok {
		return nil, errEmptySOA1
	}
	left := placeWallPCMTrackAt(leftSegs, sr, base)
	right := placeWallPCMTrackAt(rightSegs, sr, base)
	if len(left) == 0 && len(right) == 0 {
		return nil, errEmptySOA1
	}
	return sn2StereoInterleaveWAV(left, right, sr), nil
}

func decodeSOA1FramesWall(frames []SOA1Frame, sampleRate, decodeChannels int) ([]wallPCMSeg, error) {
	dec, err := opus.NewDecoder(sampleRate, decodeChannels)
	if err != nil {
		return nil, fmt.Errorf("opus decoder: %w", err)
	}
	maxSamples := sampleRate * 120 / 1000 // 120ms
	pcmScratch := make([]int16, maxSamples*decodeChannels)
	out := make([]wallPCMSeg, 0, len(frames))
	for _, f := range frames {
		n, err := dec.Decode(f.Payload, pcmScratch)
		if err != nil || n <= 0 {
			continue
		}
		mono := appendOpusPCMSlice(nil, pcmScratch, n, decodeChannels)
		out = append(out, wallPCMSeg{wallNs: f.WallNs, seq: f.Seq, pcm: mono})
	}
	return out, nil
}

// EncodePCM16MonoToSOA1 encodes mono PCM16 LE into a SOA1 Opus archive (20ms frames).
func EncodePCM16MonoToSOA1(pcm []byte, sampleRate int) ([]byte, error) {
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	if len(pcm) < 2 {
		return nil, fmt.Errorf("audutil: empty PCM")
	}
	enc, err := opus.NewEncoder(sampleRate, 1, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("opus encoder: %w", err)
	}
	_ = enc.SetBitrate(24000)
	frameSamples := sampleRate * 20 / 1000
	frameBytes := frameSamples * 2
	h := SOA1Header{
		Version:    SOA1Version,
		Channels:   1,
		SampleRate: uint32(sampleRate),
		PreSkip:    SOA1DefaultPreSkip,
	}
	var frames []SOA1Frame
	var wallNs uint64
	var seq uint16
	var rtpTs uint32
	buf := make([]byte, 4000)
	pcmSamples := make([]int16, frameSamples)
	for off := 0; off < len(pcm); {
		chunk := pcm[off:]
		if len(chunk) > frameBytes {
			chunk = chunk[:frameBytes]
		}
		// clear then fill
		for i := range pcmSamples {
			pcmSamples[i] = 0
		}
		for i := 0; i+1 < len(chunk); i += 2 {
			pcmSamples[i/2] = int16(uint16(chunk[i]) | uint16(chunk[i+1])<<8)
		}
		n, err := enc.Encode(pcmSamples, buf)
		if err != nil || n <= 0 {
			off += len(chunk)
			if len(chunk) < frameBytes {
				break
			}
			continue
		}
		payload := append([]byte(nil), buf[:n]...)
		frames = append(frames, SOA1Frame{
			WallNs:  wallNs,
			Seq:     seq,
			RTPTs:   rtpTs,
			Payload: payload,
		})
		seq++
		rtpTs += uint32(frameSamples)
		wallNs += 20_000_000 // 20ms
		off += frameBytes
		if len(chunk) < frameBytes {
			break
		}
	}
	if len(frames) == 0 {
		return nil, fmt.Errorf("audutil: no opus frames encoded")
	}
	return PackSOA1(h, frames)
}

// EncodeStereoWAVLegsToSOA1 splits a stereo PCM16 WAV into two mono SOA1 archives (L=caller, R=ai).
func EncodeStereoWAVLegsToSOA1(wav []byte) (caller, ai []byte, sampleRate int, err error) {
	sr, ch, pcm, err := parsePCM16WAV(wav)
	if err != nil {
		return nil, nil, 0, err
	}
	sampleRate = sr
	if ch == 1 {
		caller, err = EncodePCM16MonoToSOA1(pcm, sr)
		return caller, nil, sr, err
	}
	// deinterleave
	n := len(pcm) / 4
	left := make([]byte, n*2)
	right := make([]byte, n*2)
	for i := 0; i < n; i++ {
		left[i*2] = pcm[i*4]
		left[i*2+1] = pcm[i*4+1]
		right[i*2] = pcm[i*4+2]
		right[i*2+1] = pcm[i*4+3]
	}
	caller, err = EncodePCM16MonoToSOA1(left, sr)
	if err != nil {
		return nil, nil, sr, err
	}
	ai, err = EncodePCM16MonoToSOA1(right, sr)
	return caller, ai, sr, err
}
