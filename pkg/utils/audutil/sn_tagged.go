// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package audutil

import "encoding/binary"

// Recording dir tags: user/caller leg = 0, AI leg = 1.
const (
	recTagUserLeg byte = 0
	recTagAILeg   byte = 1
)

// recTaggedFrame is one SN2/SN3 payload frame after the magic.
type recTaggedFrame struct {
	Dir     byte
	Seq     uint16
	RTPTs   uint32
	WallNs  uint64 // SN2: 0 (RTP timeline only); SN3: capture-time nanoseconds
	Payload []byte
}

func parseRecordingTaggedFrames(blob []byte) []recTaggedFrame {
	if len(blob) < 3 {
		return nil
	}
	switch string(blob[:3]) {
	case "SN3":
		return parseSN3TaggedFrames(blob[3:])
	case "SN2":
		return parseSN2TaggedFrames(blob[3:])
	default:
		return nil
	}
}

func parseSN2TaggedFrames(body []byte) []recTaggedFrame {
	var out []recTaggedFrame
	for len(body) >= 9 {
		dir := body[0]
		seq := binary.LittleEndian.Uint16(body[1:3])
		ts := binary.LittleEndian.Uint32(body[3:7])
		n := int(binary.LittleEndian.Uint16(body[7:9]))
		body = body[9:]
		if n <= 0 || n > 4000 || len(body) < n {
			break
		}
		p := append([]byte(nil), body[:n]...)
		body = body[n:]
		out = append(out, recTaggedFrame{Dir: dir, Seq: seq, RTPTs: ts, WallNs: 0, Payload: p})
	}
	return out
}

func parseSN3TaggedFrames(body []byte) []recTaggedFrame {
	var out []recTaggedFrame
	for len(body) >= 17 {
		dir := body[0]
		seq := binary.LittleEndian.Uint16(body[1:3])
		ts := binary.LittleEndian.Uint32(body[3:7])
		wall := binary.LittleEndian.Uint64(body[7:15])
		n := int(binary.LittleEndian.Uint16(body[15:17]))
		body = body[17:]
		if n <= 0 || n > 4000 || len(body) < n {
			break
		}
		p := append([]byte(nil), body[:n]...)
		body = body[n:]
		out = append(out, recTaggedFrame{Dir: dir, Seq: seq, RTPTs: ts, WallNs: wall, Payload: p})
	}
	return out
}
