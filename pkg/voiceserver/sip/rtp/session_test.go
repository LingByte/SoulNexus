package rtp

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/media"
)

// TestSession_UDPLoopback verifies that a single Session can send and receive
// an RTP packet to itself over UDP (loopback).
func TestSession_UDPLoopback(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	defer s.Close()

	// Loopback: send to self.
	s.SetRemoteAddr(s.LocalAddr)

	payload := []byte{0x01, 0x02, 0x03, 0x04}

	done := make(chan *RTPPacket, 1)
	go func() {
		buf := make([]byte, 1500)
		_, addr, pkt, err := s.ReceiveRTP(buf)
		if err != nil {
			t.Errorf("ReceiveRTP error: %v", err)
			done <- nil
			return
		}
		if addr == nil || addr.Port == 0 {
			t.Errorf("unexpected addr: %v", addr)
		}
		done <- pkt
	}()

	if err := s.SendRTP(payload, 0, 160); err != nil {
		t.Fatalf("SendRTP failed: %v", err)
	}

	select {
	case pkt := <-done:
		if pkt == nil {
			t.Fatalf("nil packet from receiver")
		}
		if !bytes.Equal(pkt.Payload, payload) {
			t.Fatalf("payload mismatch: got=%v want=%v", pkt.Payload, payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for received packet")
	}
}

func TestSession_ReceiveRTP_DiscardNonV2(t *testing.T) {
	rx, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer rx.Close()
	tx, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Close()

	hdr := RTPHeader{
		Version:        1,
		PayloadType:    0,
		SequenceNumber: 1,
		Timestamp:      1,
		SSRC:           0x11111111,
	}
	raw, err := (&RTPPacket{Header: hdr, Payload: []byte{0x7F}}).Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Conn.WriteToUDP(raw, rx.LocalAddr); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)

	buf := make([]byte, 1500)
	_, _, pkt, err := rx.ReceiveRTP(buf)
	if !errors.Is(err, ErrRTPDiscard) {
		t.Fatalf("want ErrRTPDiscard, got err=%v pkt=%v", err, pkt)
	}
}

// TestSIPRTPTransport_SendAndNext verifies that SIPRTPTransport can
// send and then receive an AudioPacket over a loopback Session.
func TestSIPRTPTransport_SendAndNext(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	defer s.Close()

	s.SetRemoteAddr(s.LocalAddr)

	codec := media.CodecConfig{
		Codec:       "pcmu",
		SampleRate:  8000,
		Channels:    1,
		BitDepth:    8,
		PayloadType: 0,
	}

	tx := NewSIPRTPTransport(s, codec, media.DirectionOutput, 0)
	rx := NewSIPRTPTransport(s, codec, media.DirectionInput, 0)

	payload := []byte{0x7F, 0x00, 0x7F, 0x00}

	done := make(chan media.MediaPacket, 1)
	go func() {
		pkt, err := rx.Next(context.Background())
		if err != nil {
			t.Errorf("rx.Next error: %v", err)
			done <- nil
			return
		}
		done <- pkt
	}()

	n, err := tx.Send(context.Background(), &media.AudioPacket{Payload: payload})
	if err != nil {
		t.Fatalf("tx.Send failed: %v", err)
	}
	if n != len(payload) {
		t.Fatalf("unexpected bytes written: got=%d want=%d", n, len(payload))
	}

	select {
	case mpkt := <-done:
		audio, ok := mpkt.(*media.AudioPacket)
		if !ok {
			t.Fatalf("expected *AudioPacket, got %T", mpkt)
		}
		if !bytes.Equal(audio.Payload, payload) {
			t.Fatalf("payload mismatch: got=%v want=%v", audio.Payload, payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for rx.Next")
	}
}

func TestAddMirrorRemoteRefresh(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	addr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 19000}
	s.AddMirrorRemote(addr, time.Minute)
	s.AddMirrorRemote(addr, 3*time.Minute)
}

func TestSendG722TimestampUses8kClock(t *testing.T) {
	a, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	a.SetRemoteAddr(b.LocalAddr)
	tx := NewSIPRTPTransport(a, media.CodecConfig{
		Codec: "g722", SampleRate: 16000, PayloadType: 9, BitDepth: 16, Channels: 1,
	}, media.DirectionOutput, 0)
	before := a.Timestamp
	if _, err := tx.Send(context.Background(), &media.AudioPacket{Payload: make([]byte, 320)}); err != nil {
		t.Fatal(err)
	}
	delta := int(a.Timestamp) - int(before)
	if delta != 160 {
		t.Fatalf("timestamp delta %d (want 160 for 20ms @ 8k RTP clock)", delta)
	}
}

func TestRTPMarshalExtensionBadLength(t *testing.T) {
	p := &RTPPacket{
		Header: RTPHeader{
			Version: 2, Extension: true,
		},
		ExtensionPayload: []byte{1, 2, 3},
		Payload:          []byte{0},
	}
	if _, err := p.Marshal(); err == nil {
		t.Fatal("expected error for extension len % 4")
	}
}

func TestMarshalNilRTPPacket(t *testing.T) {
	if _, err := (*RTPPacket)(nil).Marshal(); err == nil {
		t.Fatal("expected error")
	}
}

func TestRTPMarshalTooManyCSRC(t *testing.T) {
	cs := make([]uint32, 16)
	p := &RTPPacket{Header: RTPHeader{Version: 2}, CSRC: cs}
	if _, err := p.Marshal(); err == nil {
		t.Fatal("expected error")
	}
}

func TestFirstPacketChan(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ch := s.FirstPacket()
	select {
	case <-ch:
		t.Fatal("closed early")
	default:
	}
	s.SetRemoteAddr(s.LocalAddr)
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 500)
		_, _, _, _ = s.ReceiveRTP(buf)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	if err := s.SendRTP([]byte{1}, 0, 160); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("recv timeout")
	}
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("first packet chan not closed")
	}
}

func TestTransportSend_OutputHooks(t *testing.T) {
	a, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	a.SetRemoteAddr(b.LocalAddr)

	var payloadSeen, rtpSeen bool
	tx := NewSIPRTPTransport(a, media.CodecConfig{Codec: "pcmu", SampleRate: 8000, PayloadType: 0}, media.DirectionOutput, 0)
	tx.OnOutputPayload = func(p []byte) { payloadSeen = len(p) > 0 }
	tx.OnOutputRTP = func(_ uint16, _ uint32, p []byte) { rtpSeen = len(p) > 0 }

	if _, err := tx.Send(context.Background(), &media.AudioPacket{Payload: []byte{0x7F}}); err != nil {
		t.Fatal(err)
	}
	if !payloadSeen || !rtpSeen {
		t.Fatalf("hooks payload=%v rtp=%v", payloadSeen, rtpSeen)
	}
}

func TestTransportInputHooks(t *testing.T) {
	a, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	a.SetRemoteAddr(b.LocalAddr)
	b.SetRemoteAddr(a.LocalAddr)

	var payOK, rtpOK bool
	rx := NewSIPRTPTransport(a, media.CodecConfig{Codec: "pcmu", SampleRate: 8000, PayloadType: 0}, media.DirectionInput, 0)
	rx.OnInputPayload = func([]byte) { payOK = true }
	rx.OnInputRTP = func(uint16, uint32, []byte) { rtpOK = true }

	pkt := buildPCMUPacket(1, 1000, []byte{0x7F})
	raw, err := pkt.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Conn.WriteToUDP(raw, a.LocalAddr); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	mp, err := rx.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if mp == nil {
		t.Fatal("expected packet")
	}
	if !payOK || !rtpOK {
		t.Fatalf("input hooks pay=%v rtp=%v", payOK, rtpOK)
	}
}

func buildPCMUPacket(seq uint16, ts uint32, payload []byte) *RTPPacket {
	return &RTPPacket{
		Header: RTPHeader{
			Version:        2,
			PayloadType:    0,
			SequenceNumber: seq,
			Timestamp:      ts,
			SSRC:           0x11223344,
		},
		Payload: payload,
	}
}

func TestSession_SendMirrorRTP(t *testing.T) {
	primaryLeg, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer primaryLeg.Close()

	mainRx, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer mainRx.Close()

	mirrorRx, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer mirrorRx.Close()

	primaryLeg.SetRemoteAddr(mainRx.LocalAddr)
	primaryLeg.AddMirrorRemote(mirrorRx.LocalAddr, time.Minute)

	if err := primaryLeg.SendRTP([]byte{0xAA}, 0, 160); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 500)
	_ = mainRx.Conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, pkt, err := mainRx.ReceiveRTP(buf); err != nil || pkt == nil || len(pkt.Payload) == 0 || pkt.Payload[0] != 0xAA {
		t.Fatalf("main recv err=%v", err)
	}

	_ = mirrorRx.Conn.SetReadDeadline(time.Now().Add(time.Second))
	_, _, pkt2, err := mirrorRx.ReceiveRTP(buf)
	if err != nil || pkt2 == nil || len(pkt2.Payload) == 0 || pkt2.Payload[0] != 0xAA {
		t.Fatalf("mirror recv err=%v", err)
	}
}

func TestSession_SendAfterClose(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	s.SetRemoteAddr(s.LocalAddr)
	_ = s.Close()
	if err := s.SendRTP([]byte{0x7F}, 0, 160); err == nil {
		t.Fatal("expected error after close")
	}
}

func TestSession_ReceiveAfterClose(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	buf := make([]byte, 500)
	_, _, _, err = s.ReceiveRTP(buf)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSIPRTPTransport_Send_RTPSamples(t *testing.T) {
	a, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	a.SetRemoteAddr(b.LocalAddr)

	codec := media.CodecConfig{Codec: "opus", SampleRate: 48000, PayloadType: 111, Channels: 1}
	tx := NewSIPRTPTransport(a, codec, media.DirectionOutput, 0)
	ctx := context.Background()
	n, err := tx.Send(ctx, &media.AudioPacket{Payload: []byte{1, 2, 3}, RTPSamples: 960})
	if err != nil || n != 3 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestSIPRTPTransport_JitterAdaptiveBaseline(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	codec := media.CodecConfig{Codec: "pcmu", SampleRate: 8000, PayloadType: 0}
	rx := NewSIPRTPTransport(s, codec, media.DirectionInput, 0)
	rx.JitterPlaybackDelay = 60 * time.Millisecond
	rx.jbEnsureAdaptiveBaseline()
	if rx.jbAdaptiveDelay != 60*time.Millisecond {
		t.Fatalf("baseline %v", rx.jbAdaptiveDelay)
	}
	d := rx.jbEffectiveDelay()
	if d < jbDelayFloor {
		t.Fatalf("delay %v", d)
	}
}

func TestSIPRTPTransport_String(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	tr := NewSIPRTPTransport(s, media.CodecConfig{Codec: "pcmu"}, media.DirectionInput, 0)
	if !strings.Contains(tr.String(), "dir=rx") {
		t.Fatal(tr.String())
	}
}

func TestNextContextCancelled(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rx := NewSIPRTPTransport(s, media.CodecConfig{Codec: "pcmu", SampleRate: 8000, PayloadType: 0}, media.DirectionInput, 0)
	rx.JitterPlaybackDelay = 10 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	mp, err := rx.Next(ctx)
	if mp != nil || err != nil {
		t.Fatalf("got %v err=%v", mp, err)
	}
}

func TestCloneRTPPacketForJitterNil(t *testing.T) {
	if cloneRTPPacketForJitter(nil) != nil {
		t.Fatal()
	}
}

func TestAddrStringNilUDPAddr(t *testing.T) {
	if addrString(nil) != "" {
		t.Fatal()
	}
}

func TestSIPRTPTransport_CodecAccessor(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	tr := NewSIPRTPTransport(s, media.CodecConfig{Codec: "pcma", PayloadType: 8}, media.DirectionInput, 0)
	if tr.Codec().Codec != "pcma" || tr.Codec().PayloadType != 8 {
		t.Fatal()
	}
}
