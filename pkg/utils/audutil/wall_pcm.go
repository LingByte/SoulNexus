// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package audutil

import (
	"bytes"
	"encoding/binary"
	"math"
	"sort"
)

// recMixPeakThreshold: gentle peak scaling before hard clip when interleaving stereo.
const recMixPeakThreshold = 0.92

func floatToPCM16Clamped(v float64) int16 {
	if v > 1 {
		v = 1
	} else if v < -1 {
		v = -1
	}
	x := int(math.Round(v * 32767))
	if x > 32767 {
		x = 32767
	}
	if x < -32768 {
		x = -32768
	}
	return int16(x)
}

func peakNormalizeInterleavedStereo(lr []int16) {
	if len(lr) < 2 {
		return
	}
	peak := 1e-12
	for i := 0; i < len(lr); i++ {
		if x := math.Abs(float64(lr[i]) / 32768.0); x > peak {
			peak = x
		}
	}
	scale := 1.0
	if peak > recMixPeakThreshold {
		scale = recMixPeakThreshold / peak
	}
	for i := 0; i < len(lr); i++ {
		lr[i] = floatToPCM16Clamped(float64(lr[i]) / 32768.0 * scale)
	}
}

type wallPCMSeg struct {
	wallNs uint64
	seq    uint16
	pcm    []int16
}

// placeWallPCMTrack maps capture-time offsets to samples.
//
// JITTER SNAP（与 pkg/voice/recorder/placePCMTrackBytes 同源）：当相邻帧
// wall 间距 < RecordingJitterSnapNs 时 snap 到连续时间线，避免调度抖动被当成静音。
func placeWallPCMTrack(segs []wallPCMSeg, pcmHz int) []int16 {
	if len(segs) == 0 {
		return nil
	}
	return placeWallPCMTrackAt(segs, pcmHz, segs[0].wallNs)
}

// placeWallPCMTrackAt places segments on a wall-clock timeline relative to base.
// Stereo legs MUST share the same base (min wall across both legs).
func placeWallPCMTrackAt(segs []wallPCMSeg, pcmHz int, base uint64) []int16 {
	if len(segs) == 0 || pcmHz <= 0 {
		return nil
	}
	sort.Slice(segs, func(i, j int) bool {
		if segs[i].wallNs != segs[j].wallNs {
			return segs[i].wallNs < segs[j].wallNs
		}
		return segs[i].seq < segs[j].seq
	})
	jitterSnapNs := uint64(RecordingJitterSnapNs())
	var out []int16
	pen := 0
	prevTailNs := base
	for _, s := range segs {
		posFromWall := int(int64(s.wallNs-base) * int64(pcmHz) / 1e9)
		if posFromWall < 0 {
			posFromWall = 0
		}
		if posFromWall > pen && s.wallNs >= prevTailNs && (s.wallNs-prevTailNs) < jitterSnapNs {
			posFromWall = pen
		}
		if posFromWall < pen {
			posFromWall = pen
		}
		end := posFromWall + len(s.pcm)
		if len(out) < end {
			out = append(out, make([]int16, end-len(out))...)
		}
		copy(out[posFromWall:], s.pcm)
		if end > pen {
			pen = end
		}
		prevTailNs = base + uint64(int64(pen)*int64(1_000_000_000)/int64(pcmHz))
	}
	return out
}

func minWallNs(segs ...[]wallPCMSeg) (uint64, bool) {
	var min uint64
	ok := false
	for _, list := range segs {
		for _, s := range list {
			if !ok || s.wallNs < min {
				min = s.wallNs
				ok = true
			}
		}
	}
	return min, ok
}

func sn2StereoInterleaveWAV(left, right []int16, pcmHz int) []byte {
	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	if n == 0 {
		return nil
	}
	lr := make([]int16, n*2)
	for i := 0; i < n; i++ {
		if i < len(left) {
			lr[i*2] = left[i]
		}
		if i < len(right) {
			lr[i*2+1] = right[i]
		}
	}
	peakNormalizeInterleavedStereo(lr)
	return pcm16StereoInterleavedToWav(lr, pcmHz)
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
