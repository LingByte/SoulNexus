package rtp

import (
	"bytes"
	"testing"
)

func TestRTPPacket_MarshalUnmarshal_RoundTrip(t *testing.T) {
	orig := &RTPPacket{
		Header: RTPHeader{
			Version:        2,
			Padding:        false,
			Extension:      true,
			CSRCCount:      2,
			Marker:         true,
			PayloadType:    96,
			SequenceNumber: 12345,
			Timestamp:      0xAABBCCDD,
			SSRC:           0x01020304,
		},
		CSRC:             []uint32{0x11111111, 0x22222222},
		ExtensionProfile: 0xBEDE,
		ExtensionPayload: []byte{0x00, 0x01, 0x02, 0x03},
		Payload: []byte{0x01, 0x02, 0x03, 0xFF, 0x10},
	}

	b, err := orig.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	got := &RTPPacket{}
	if err := got.Unmarshal(b); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if got.Header.Version != orig.Header.Version ||
		got.Header.Padding != orig.Header.Padding ||
		got.Header.Extension != orig.Header.Extension ||
		got.Header.CSRCCount != orig.Header.CSRCCount ||
		got.Header.Marker != orig.Header.Marker ||
		got.Header.PayloadType != orig.Header.PayloadType ||
		got.Header.SequenceNumber != orig.Header.SequenceNumber ||
		got.Header.Timestamp != orig.Header.Timestamp ||
		got.Header.SSRC != orig.Header.SSRC {
		t.Fatalf("header mismatch: got=%+v orig=%+v", got.Header, orig.Header)
	}

	if !bytes.Equal(got.Payload, orig.Payload) {
		t.Fatalf("payload mismatch: got=%v orig=%v", got.Payload, orig.Payload)
	}

	if got.ExtensionProfile != orig.ExtensionProfile {
		t.Fatalf("extension profile mismatch: got=%x want=%x", got.ExtensionProfile, orig.ExtensionProfile)
	}
	if !bytes.Equal(got.ExtensionPayload, orig.ExtensionPayload) {
		t.Fatalf("extension payload mismatch: got=%v want=%v", got.ExtensionPayload, orig.ExtensionPayload)
	}
	if len(got.CSRC) != len(orig.CSRC) {
		t.Fatalf("csrc length mismatch: got=%d want=%d", len(got.CSRC), len(orig.CSRC))
	}
	for i := range got.CSRC {
		if got.CSRC[i] != orig.CSRC[i] {
			t.Fatalf("csrc[%d] mismatch: got=%x want=%x", i, got.CSRC[i], orig.CSRC[i])
		}
	}
}

func TestRTPPacket_Unmarshal_TooShort(t *testing.T) {
	p := &RTPPacket{}
	err := p.Unmarshal(make([]byte, 11))
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestRTPPacket_Unmarshal_InvalidPaddingLength(t *testing.T) {
	p := &RTPPacket{}
	// Version 2, padding bit, minimal header + payload "x" + invalid pad len 0
	b := []byte{0xA0, 0x00, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 'x', 0}
	if err := p.Unmarshal(b); err == nil {
		t.Fatal("expected padding error")
	}
}

func TestRTPPacket_Unmarshal_Padding(t *testing.T) {
	// Build a packet with 3 bytes padding. Padding bytes can be anything; last byte is padding length.
	p := &RTPPacket{
		Header: RTPHeader{
			Version:     2,
			Padding:     true,
			Extension:   false,
			CSRCCount:   0,
			PayloadType: 0,
		},
		Payload: []byte{0xAA, 0xBB, 0xCC},
	}
	b, err := p.Marshal()
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	// Append 3 padding bytes, last byte = 3.
	b = append(b, 0x00, 0x00, 0x03)

	got := &RTPPacket{}
	if err := got.Unmarshal(b); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if !bytes.Equal(got.Payload, p.Payload) {
		t.Fatalf("payload mismatch: got=%v want=%v", got.Payload, p.Payload)
	}
}

func TestSession_SendRTP_RemoteAddrNotSet(t *testing.T) {
	s := &Session{}
	err := s.SendRTP([]byte{0x01}, 0, 160)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestSession_BuildPacketAndUpdateAfterSend(t *testing.T) {
	s := &Session{
		SSRC:       0x11223344,
		SeqNum:     10,
		Timestamp: 1000,
	}

	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	pkt := s.buildPacket(payload, 0x60)
	if pkt.Header.SequenceNumber != 10 {
		t.Fatalf("expected seq=10, got=%d", pkt.Header.SequenceNumber)
	}
	if pkt.Header.Timestamp != 1000 {
		t.Fatalf("expected ts=1000, got=%d", pkt.Header.Timestamp)
	}
	if pkt.Header.SSRC != 0x11223344 {
		t.Fatalf("expected ssrc mismatch, got=%d", pkt.Header.SSRC)
	}
	if !bytes.Equal(pkt.Payload, payload) {
		t.Fatalf("payload mismatch")
	}

	s.updateAfterSend(320) // timestamp +320 samples
	if s.SeqNum != 11 {
		t.Fatalf("expected seq=11, got=%d", s.SeqNum)
	}
	if s.Timestamp != 1320 {
		t.Fatalf("expected ts=1320, got=%d", s.Timestamp)
	}
}

