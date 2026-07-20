// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package audutil

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

// SOA1 is a single-leg Opus frame archive stored in object storage.
//
//	magic "SOA1"
//	u8  version=1
//	u8  channels
//	u32 sample_rate LE
//	u16 pre_skip LE
//	u16 reserved
//	frames:
//	  u64 wall_ns LE
//	  u16 seq LE
//	  u32 rtp_ts LE
//	  u16 len LE
//	  [payload]
const (
	SOA1Magic          = "SOA1"
	SOA1Version        = 1
	SOA1HeaderSize     = 4 + 1 + 1 + 4 + 2 + 2 // 14
	SOA1FrameHdrLen    = 8 + 2 + 4 + 2         // 16
	SOA1DefaultPreSkip = 312
)

// Recording format values persisted on recording_format.
const (
	RecordingFormatSOA1 = "soa1"
	RecordingFormatWAV  = "wav"
)

// SOA1Header describes one leg's Opus archive.
type SOA1Header struct {
	Version    uint8
	Channels   uint8
	SampleRate uint32
	PreSkip    uint16
}

// SOA1Frame is one Opus packet with capture timing.
type SOA1Frame struct {
	WallNs  uint64
	Seq     uint16
	RTPTs   uint32
	Payload []byte
}

// PackSOA1 serialises header + frames into a SOA1 blob.
func PackSOA1(h SOA1Header, frames []SOA1Frame) ([]byte, error) {
	if h.Version == 0 {
		h.Version = SOA1Version
	}
	if h.Channels < 1 {
		h.Channels = 1
	}
	if h.Channels > 2 {
		h.Channels = 2
	}
	if h.SampleRate == 0 {
		h.SampleRate = 48000
	}
	if h.PreSkip == 0 {
		h.PreSkip = SOA1DefaultPreSkip
	}
	n := SOA1HeaderSize
	for _, f := range frames {
		if len(f.Payload) == 0 || len(f.Payload) > 4000 {
			continue
		}
		n += SOA1FrameHdrLen + len(f.Payload)
	}
	out := make([]byte, 0, n)
	out = append(out, SOA1Magic...)
	out = append(out, h.Version, h.Channels)
	var u32 [4]byte
	binary.LittleEndian.PutUint32(u32[:], h.SampleRate)
	out = append(out, u32[:]...)
	var u16 [2]byte
	binary.LittleEndian.PutUint16(u16[:], h.PreSkip)
	out = append(out, u16[:]...)
	out = append(out, 0, 0) // reserved
	for _, f := range frames {
		if len(f.Payload) == 0 || len(f.Payload) > 4000 {
			continue
		}
		var hdr [SOA1FrameHdrLen]byte
		binary.LittleEndian.PutUint64(hdr[0:8], f.WallNs)
		binary.LittleEndian.PutUint16(hdr[8:10], f.Seq)
		binary.LittleEndian.PutUint32(hdr[10:14], f.RTPTs)
		binary.LittleEndian.PutUint16(hdr[14:16], uint16(len(f.Payload)))
		out = append(out, hdr[:]...)
		out = append(out, f.Payload...)
	}
	return out, nil
}

// ParseSOA1 parses a SOA1 blob.
func ParseSOA1(b []byte) (SOA1Header, []SOA1Frame, error) {
	if len(b) < SOA1HeaderSize || string(b[:4]) != SOA1Magic {
		return SOA1Header{}, nil, fmt.Errorf("audutil: not SOA1")
	}
	h := SOA1Header{
		Version:    b[4],
		Channels:   b[5],
		SampleRate: binary.LittleEndian.Uint32(b[6:10]),
		PreSkip:    binary.LittleEndian.Uint16(b[10:12]),
	}
	if h.Version != SOA1Version {
		return SOA1Header{}, nil, fmt.Errorf("audutil: unsupported SOA1 version %d", h.Version)
	}
	if h.Channels < 1 {
		h.Channels = 1
	}
	if h.SampleRate == 0 {
		h.SampleRate = 48000
	}
	body := b[SOA1HeaderSize:]
	var frames []SOA1Frame
	for len(body) >= SOA1FrameHdrLen {
		wall := binary.LittleEndian.Uint64(body[0:8])
		seq := binary.LittleEndian.Uint16(body[8:10])
		ts := binary.LittleEndian.Uint32(body[10:14])
		n := int(binary.LittleEndian.Uint16(body[14:16]))
		body = body[SOA1FrameHdrLen:]
		if n <= 0 || n > 4000 || len(body) < n {
			break
		}
		p := append([]byte(nil), body[:n]...)
		body = body[n:]
		frames = append(frames, SOA1Frame{WallNs: wall, Seq: seq, RTPTs: ts, Payload: p})
	}
	return h, frames, nil
}

// SplitSN3ToSOA1 splits an SN3 (or SN2) blob into caller and AI SOA1 archives.
// Returns nil blobs when a leg has no frames.
func SplitSN3ToSOA1(sn []byte, sampleRate, channels int) (caller, ai []byte, err error) {
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	if channels < 1 {
		channels = 1
	}
	if channels > 2 {
		channels = 2
	}
	frames := parseRecordingTaggedFrames(sn)
	if len(frames) == 0 {
		return nil, nil, fmt.Errorf("audutil: no tagged frames")
	}
	var uf, af []SOA1Frame
	for _, f := range frames {
		sf := SOA1Frame{WallNs: f.WallNs, Seq: f.Seq, RTPTs: f.RTPTs, Payload: f.Payload}
		switch f.Dir {
		case recTagUserLeg:
			uf = append(uf, sf)
		case recTagAILeg:
			af = append(af, sf)
		}
	}
	h := SOA1Header{
		Version:    SOA1Version,
		Channels:   uint8(channels),
		SampleRate: uint32(sampleRate),
		PreSkip:    SOA1DefaultPreSkip,
	}
	if len(uf) > 0 {
		caller, err = PackSOA1(h, uf)
		if err != nil {
			return nil, nil, err
		}
	}
	if len(af) > 0 {
		ai, err = PackSOA1(h, af)
		if err != nil {
			return nil, nil, err
		}
	}
	if len(caller) == 0 && len(ai) == 0 {
		return nil, nil, fmt.Errorf("audutil: empty legs after split")
	}
	return caller, ai, nil
}

// TrimSOA1AfterWallNs keeps frames with WallNs >= afterWallNs (inclusive).
// Used to skip the AI phase when transcribing post-transfer agent legs.
func TrimSOA1AfterWallNs(soa1 []byte, afterWallNs int64) ([]byte, error) {
	if afterWallNs <= 0 {
		return soa1, nil
	}
	h, frames, err := ParseSOA1(soa1)
	if err != nil {
		return nil, err
	}
	out := frames[:0]
	min := uint64(afterWallNs)
	for _, f := range frames {
		if f.WallNs >= min {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("audutil: no SOA1 frames after wall_ns=%d", afterWallNs)
	}
	return PackSOA1(h, out)
}

// RecordingLegObjectKey returns recordings/{tenant}/{date}/{callID}-{leg}.soa1
// leg is "caller" or "ai".
func RecordingLegObjectKey(tenantID uint, callID string, at time.Time, leg string) string {
	leg = strings.ToLower(strings.TrimSpace(leg))
	switch leg {
	case "caller", "user":
		leg = "caller"
	case "ai", "agent":
		leg = "ai"
	default:
		return ""
	}
	base := RecordingObjectKey(tenantID, callID, at, "soa1")
	if base == "" {
		return ""
	}
	// insert -{leg} before extension
	if i := strings.LastIndex(base, ".soa1"); i > 0 {
		return base[:i] + "-" + leg + ".soa1"
	}
	return base + "-" + leg
}
