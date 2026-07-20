// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package audutil

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

func TestPackParseSOA1(t *testing.T) {
	h := SOA1Header{Version: SOA1Version, Channels: 1, SampleRate: 48000, PreSkip: SOA1DefaultPreSkip}
	frames := []SOA1Frame{
		{WallNs: 0, Seq: 1, RTPTs: 0, Payload: []byte{0x78, 0x01, 0x02}},
		{WallNs: 20_000_000, Seq: 2, RTPTs: 960, Payload: []byte{0x78, 0x03, 0x04}},
	}
	blob, err := PackSOA1(h, frames)
	if err != nil {
		t.Fatal(err)
	}
	if string(blob[:4]) != SOA1Magic {
		t.Fatalf("magic %q", blob[:4])
	}
	gotH, gotF, err := ParseSOA1(blob)
	if err != nil {
		t.Fatal(err)
	}
	if gotH.SampleRate != 48000 || gotH.Channels != 1 || len(gotF) != 2 {
		t.Fatalf("header/frames %#v n=%d", gotH, len(gotF))
	}
	if !bytes.Equal(gotF[1].Payload, frames[1].Payload) || gotF[1].WallNs != frames[1].WallNs {
		t.Fatalf("frame1 %#v", gotF[1])
	}
}

func TestSplitSN3ToSOA1(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("SN3")
	appendSN3 := func(dir byte, seq uint16, ts uint32, wall uint64, payload []byte) {
		buf.WriteByte(dir)
		var hdr [16]byte
		binary.LittleEndian.PutUint16(hdr[0:2], seq)
		binary.LittleEndian.PutUint32(hdr[2:6], ts)
		binary.LittleEndian.PutUint64(hdr[6:14], wall)
		binary.LittleEndian.PutUint16(hdr[14:16], uint16(len(payload)))
		buf.Write(hdr[:])
		buf.Write(payload)
	}
	appendSN3(recTagUserLeg, 1, 0, 0, []byte{0xaa, 0x01})
	appendSN3(recTagAILeg, 1, 0, 0, []byte{0xbb, 0x02})
	appendSN3(recTagUserLeg, 2, 960, 20_000_000, []byte{0xaa, 0x03})

	caller, ai, err := SplitSN3ToSOA1(buf.Bytes(), 48000, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, cf, err := ParseSOA1(caller)
	if err != nil || len(cf) != 2 {
		t.Fatalf("caller frames=%d err=%v", len(cf), err)
	}
	_, af, err := ParseSOA1(ai)
	if err != nil || len(af) != 1 {
		t.Fatalf("ai frames=%d err=%v", len(af), err)
	}
}

func TestRecordingLegObjectKey(t *testing.T) {
	at := time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC)
	k := RecordingLegObjectKey(7, "cid@host", at, "caller")
	if k != "recordings/7/2026-07-14/cid_host-caller.soa1" {
		t.Fatalf("got %q", k)
	}
	k2 := RecordingLegObjectKey(7, "cid@host", at, "ai")
	if k2 != "recordings/7/2026-07-14/cid_host-ai.soa1" {
		t.Fatalf("got %q", k2)
	}
}

func TestEncodeDecodeSOA1RoundTrip(t *testing.T) {
	sr := 16000
	// 60ms of quiet + tone-ish samples
	n := sr * 60 / 1000
	pcm := make([]byte, n*2)
	for i := 0; i < n; i++ {
		v := int16((i % 40) * 100)
		pcm[i*2] = byte(v)
		pcm[i*2+1] = byte(v >> 8)
	}
	soa, err := EncodePCM16MonoToSOA1(pcm, sr)
	if err != nil {
		t.Fatal(err)
	}
	ogg, err := PackOggOpus(soa)
	if err != nil {
		t.Fatal(err)
	}
	if len(ogg) < 8 || string(ogg[:4]) != "OggS" {
		t.Fatalf("ogg magic %q", ogg[:min(4, len(ogg))])
	}
	if !bytes.Contains(ogg, []byte("OpusHead")) {
		t.Fatal("missing OpusHead")
	}
	wav, err := SOA1ToWAV(soa)
	if err != nil {
		t.Fatal(err)
	}
	if len(wav) < 44 || string(wav[:4]) != "RIFF" {
		t.Fatalf("wav %#v", wav[:min(8, len(wav))])
	}
}

func TestMergeSOA1StereoWAV(t *testing.T) {
	sr := 16000
	n := sr * 40 / 1000
	mk := func(amp int16) []byte {
		pcm := make([]byte, n*2)
		for i := 0; i < n; i++ {
			pcm[i*2] = byte(amp)
			pcm[i*2+1] = byte(amp >> 8)
		}
		return pcm
	}
	c, err := EncodePCM16MonoToSOA1(mk(1000), sr)
	if err != nil {
		t.Fatal(err)
	}
	a, err := EncodePCM16MonoToSOA1(mk(2000), sr)
	if err != nil {
		t.Fatal(err)
	}
	wav, err := MergeSOA1StereoWAV(c, a)
	if err != nil {
		t.Fatal(err)
	}
	if string(wav[:4]) != "RIFF" {
		t.Fatal("not wav")
	}
}
