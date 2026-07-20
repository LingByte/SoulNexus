// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package audutil

import (
	"encoding/binary"
)

// oggCRCTable is the non-reflected CRC-32 used by Ogg (poly 0x04c11db7).
var oggCRCTable = makeOggCRCTable()

func makeOggCRCTable() [256]uint32 {
	var t [256]uint32
	const poly = uint32(0x04c11db7)
	for i := 0; i < 256; i++ {
		r := uint32(i) << 24
		for j := 0; j < 8; j++ {
			if r&0x80000000 != 0 {
				r = (r << 1) ^ poly
			} else {
				r <<= 1
			}
		}
		t[i] = r
	}
	return t
}

// PackOggOpus wraps a SOA1 archive into a complete Ogg-Opus bitstream (no disk I/O).
// Wall-clock gaps between frames are preserved by densifying through PCM placement
// then re-encoding, so turn silence is not collapsed.
// Granule positions follow RFC 7845 (48 kHz sample units).
func PackOggOpus(soa1 []byte) ([]byte, error) {
	_, frames, err := ParseSOA1(soa1)
	if err != nil {
		return nil, err
	}
	if len(frames) == 0 {
		return nil, errEmptySOA1
	}
	// Densify sparse SOA1 (absolute wall_ns) so Ogg plays with real turn gaps.
	if soaHasWallGaps(frames) {
		wav, err := SOA1ToWAV(soa1)
		if err != nil {
			return nil, err
		}
		sr, _, pcm, err := parsePCM16WAV(wav)
		if err != nil {
			return nil, err
		}
		dense, err := EncodePCM16MonoToSOA1(pcm, sr)
		if err != nil {
			return nil, err
		}
		return packOggOpusDense(dense)
	}
	return packOggOpusDense(soa1)
}

func soaHasWallGaps(frames []SOA1Frame) bool {
	if len(frames) < 2 {
		return false
	}
	const gapThreshNs = 40_000_000 // >40ms between successive packets
	for i := 1; i < len(frames); i++ {
		if frames[i].WallNs > frames[i-1].WallNs+gapThreshNs {
			return true
		}
	}
	return false
}

func packOggOpusDense(soa1 []byte) ([]byte, error) {
	h, frames, err := ParseSOA1(soa1)
	if err != nil {
		return nil, err
	}
	if len(frames) == 0 {
		return nil, errEmptySOA1
	}
	sr := int(h.SampleRate)
	if sr <= 0 {
		sr = 48000
	}
	ch := int(h.Channels)
	if ch < 1 {
		ch = 1
	}
	preSkip := h.PreSkip
	if preSkip == 0 {
		preSkip = SOA1DefaultPreSkip
	}

	var out []byte
	serial := uint32(1)
	pageSeq := uint32(0)

	idHdr := buildOpusHead(ch, preSkip, uint32(sr))
	out = append(out, encodeOggPage(serial, pageSeq, 0, 0x02, [][]byte{idHdr})...) // BOS
	pageSeq++

	tags := buildOpusTags()
	out = append(out, encodeOggPage(serial, pageSeq, 0, 0x00, [][]byte{tags})...)
	pageSeq++

	samplesPerFrame48k := opusFrameSamples48k(frames[0].Payload, sr)
	if samplesPerFrame48k <= 0 {
		samplesPerFrame48k = 960 // 20ms @ 48k
	}
	var granule uint64
	const maxPktsPerPage = 48
	for i := 0; i < len(frames); {
		n := len(frames) - i
		if n > maxPktsPerPage {
			n = maxPktsPerPage
		}
		pkts := make([][]byte, 0, n)
		for j := 0; j < n; j++ {
			pkts = append(pkts, frames[i+j].Payload)
			granule += uint64(samplesPerFrame48k)
		}
		i += n
		headerType := byte(0)
		if i >= len(frames) {
			headerType = 0x04 // EOS
		}
		out = append(out, encodeOggPage(serial, pageSeq, granule, headerType, pkts)...)
		pageSeq++
	}
	return out, nil
}

var errEmptySOA1 = errString("audutil: empty SOA1")

type errString string

func (e errString) Error() string { return string(e) }

func buildOpusHead(channels int, preSkip uint16, inputSampleRate uint32) []byte {
	b := make([]byte, 19)
	copy(b[0:8], "OpusHead")
	b[8] = 1
	b[9] = byte(channels)
	binary.LittleEndian.PutUint16(b[10:12], preSkip)
	binary.LittleEndian.PutUint32(b[12:16], inputSampleRate)
	binary.LittleEndian.PutUint16(b[16:18], 0) // output gain
	b[18] = 0                                  // channel mapping family 0
	return b
}

func buildOpusTags() []byte {
	vendor := []byte("SoulNexus")
	b := make([]byte, 8+4+len(vendor)+4)
	copy(b[0:8], "OpusTags")
	binary.LittleEndian.PutUint32(b[8:12], uint32(len(vendor)))
	copy(b[12:], vendor)
	binary.LittleEndian.PutUint32(b[12+len(vendor):], 0) // user comment list length
	return b
}

// opusFrameSamples48k estimates PCM samples at 48 kHz for one Opus packet (RFC 6716 TOC).
func opusFrameSamples48k(pkt []byte, encodeRate int) int {
	_ = encodeRate
	if len(pkt) == 0 {
		return 0
	}
	toc := pkt[0]
	config := toc >> 3
	// frame size codes map to 2.5/5/10/20/40/60 ms (see RFC 6716 §3.1)
	var ms int
	switch {
	case config < 12:
		ms = []int{10, 20, 40, 60}[config&3]
	case config < 16:
		ms = []int{10, 20, 40, 60}[config&3]
	case config < 20:
		ms = []int{10, 20}[config&1]
	case config < 24:
		ms = []int{10, 20}[config&1]
	case config < 28:
		ms = []int{2, 5, 10, 20}[config&3]
		if ms < 10 {
			ms = 10
		}
	default:
		ms = []int{2, 5, 10, 20}[config&3]
		if ms < 10 {
			ms = 20
		}
	}
	code := toc & 3
	frames := 1
	switch code {
	case 1, 2:
		frames = 2
	case 3:
		if len(pkt) > 1 {
			frames = int(pkt[1] & 0x3f)
			if frames < 1 {
				frames = 1
			}
		}
	}
	return frames * ms * 48 // 48 samples/ms @ 48 kHz
}

func encodeOggPage(serial, pageSeq uint32, granule uint64, headerType byte, packets [][]byte) []byte {
	// Build segment table
	var segs []byte
	var body []byte
	for _, p := range packets {
		remain := len(p)
		off := 0
		for {
			n := remain
			if n > 255 {
				n = 255
			}
			segs = append(segs, byte(n))
			body = append(body, p[off:off+n]...)
			off += n
			remain -= n
			if remain == 0 {
				// exact multiple of 255 needs a 0-length terminating segment
				if n == 255 {
					segs = append(segs, 0)
				}
				break
			}
		}
	}
	hdr := make([]byte, 27+len(segs))
	copy(hdr[0:4], "OggS")
	hdr[4] = 0
	hdr[5] = headerType
	binary.LittleEndian.PutUint64(hdr[6:14], granule)
	binary.LittleEndian.PutUint32(hdr[14:18], serial)
	binary.LittleEndian.PutUint32(hdr[18:22], pageSeq)
	// checksum zeroed for calc
	hdr[26] = byte(len(segs))
	copy(hdr[27:], segs)

	page := append(hdr, body...)
	crc := oggCRC(page)
	binary.LittleEndian.PutUint32(page[22:26], crc)
	return page
}

func oggCRC(page []byte) uint32 {
	var crc uint32
	for _, b := range page {
		crc = (crc << 8) ^ oggCRCTable[byte(crc>>24)^b]
	}
	return crc
}
