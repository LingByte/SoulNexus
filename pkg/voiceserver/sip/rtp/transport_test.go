package rtp

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/media"
)

// This test verifies an end-to-end path:
// client RTP session -> server MediaSession(Input SIPRTPTransport) -> output-router -> server SIPRTPTransport -> client RTP session.
//
// We intentionally use two UDP sockets (client/server) to avoid creating an infinite echo loop on a single socket.
func TestMediaSession_WithSIPRTPTransport_Echo(t *testing.T) {
	// media package relies on pkg/logger global zap logger.
	// Initialize it to avoid nil pointer panics in tests.
	tmp := t.TempDir()
	if err := logger.Init(&logger.LogConfig{
		Level:      "debug",
		Filename:   tmp + "/test.log",
		MaxSize:    1,
		MaxAge:     1,
		MaxBackups: 1,
		Daily:      false,
	}, "dev"); err != nil {
		t.Fatalf("logger.Init failed: %v", err)
	}

	// Server RTP socket
	serverSess, err := NewSession(0)
	if err != nil {
		t.Fatalf("server NewSession failed: %v", err)
	}
	defer serverSess.Close()

	// Client RTP socket
	clientSess, err := NewSession(0)
	if err != nil {
		t.Fatalf("client NewSession failed: %v", err)
	}
	defer clientSess.Close()

	// Wire remote addresses.
	serverSess.SetRemoteAddr(clientSess.LocalAddr)
	clientSess.SetRemoteAddr(serverSess.LocalAddr)

	codec := media.CodecConfig{
		Codec:       "pcmu",
		SampleRate:  8000,
		Channels:    1,
		BitDepth:    8,
		PayloadType: 0,
	}

	rx := NewSIPRTPTransport(serverSess, codec, media.DirectionInput, 0)
	tx := NewSIPRTPTransport(serverSess, codec, media.DirectionOutput, 0)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ms := media.NewDefaultSession().
		Context(ctx).
		SetSessionID("test-sip-rtp-echo").
		Input(rx).
		Output(tx)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- ms.Serve()
	}()

	// Give the server a moment to start its goroutines.
	time.Sleep(50 * time.Millisecond)

	original := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	if err := clientSess.SendRTP(original, 0, 160); err != nil {
		t.Fatalf("client SendRTP failed: %v", err)
	}

	// Receive echoed RTP on the client socket.
	buf := make([]byte, 1500)
	recvCh := make(chan *RTPPacket, 1)
	go func() {
		_, _, pkt, err := clientSess.ReceiveRTP(buf)
		if err != nil {
			t.Errorf("client ReceiveRTP failed: %v", err)
			recvCh <- nil
			return
		}
		recvCh <- pkt
	}()

	select {
	case pkt := <-recvCh:
		if pkt == nil {
			t.Fatalf("expected packet, got nil")
		}
		if !bytes.Equal(pkt.Payload, original) {
			t.Fatalf("echo payload mismatch: got=%v want=%v", pkt.Payload, original)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for echoed RTP")
	}

	// Stop the media session and ensure Serve() returns.
	_ = ms.Close()
	cancel()

	select {
	case err := <-serveErr:
		// Serve may return nil or context cancellation related errors depending on internals.
		_ = err
	case <-time.After(8 * time.Second):
		t.Fatalf("timeout waiting for MediaSession.Serve() to return")
	}
}

// Regression: large payloads on the telephone-event PT must not skip recording — audio frames were
// mistaken for partial DTMF (EventFromRFC2833 ok on arbitrary bytes), yielding SN2 with AI-only.
func TestSIPRTPTransport_Input_OnInputRTPBeforeDTMFHeuristic(t *testing.T) {
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

	codec := media.CodecConfig{
		Codec:       "pcmu",
		SampleRate:  8000,
		Channels:    1,
		BitDepth:    8,
		PayloadType: 0,
	}
	tePT := uint8(101)
	rx := NewSIPRTPTransport(a, codec, media.DirectionInput, tePT)

	var gotPayload []byte
	rx.OnInputRTP = func(_ uint16, _ uint32, p []byte) {
		gotPayload = append([]byte(nil), p...)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_, _ = rx.Next(ctx)
	}()

	// Looks like digit "3" without end bit + padding like a 20 ms PCMU frame — old logic dropped recording + decode.
	payload := make([]byte, 160)
	payload[0] = 3 // maps to DTMF digit "3"
	payload[1] = 0 // end bit clear

	hdr := RTPHeader{
		Version: 2, PayloadType: tePT, SequenceNumber: 7, Timestamp: 100, SSRC: 1,
	}
	pkt := RTPPacket{Header: hdr, Payload: payload}
	raw, err := pkt.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Conn.WriteToUDP(raw, a.LocalAddr); err != nil {
		t.Fatal(err)
	}

	time.Sleep(300 * time.Millisecond)
	cancel()

	if len(gotPayload) != len(payload) {
		t.Fatalf("OnInputRTP payload len: got %d want %d", len(gotPayload), len(payload))
	}
	for i := range payload {
		if gotPayload[i] != payload[i] {
			t.Fatalf("byte %d: got %02x want %02x", i, gotPayload[i], payload[i])
		}
	}
}

func TestNextSkipsWrongPayloadType(t *testing.T) {
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

	rx := NewSIPRTPTransport(a, media.CodecConfig{Codec: "pcma", SampleRate: 8000, PayloadType: 8}, media.DirectionInput, 0)

	const ssrc = uint32(0x11111111)
	bad := &RTPPacket{
		Header:  RTPHeader{Version: 2, PayloadType: 99, SequenceNumber: 1, Timestamp: 1, SSRC: ssrc},
		Payload: []byte{1, 2, 3, 4, 5},
	}
	rawBad, _ := bad.Marshal()
	good := &RTPPacket{
		Header:  RTPHeader{Version: 2, PayloadType: 8, SequenceNumber: 2, Timestamp: 2, SSRC: ssrc},
		Payload: []byte{0x7F},
	}
	rawGood, _ := good.Marshal()

	go func() {
		time.Sleep(20 * time.Millisecond)
		_, _ = b.Conn.WriteToUDP(rawBad, a.LocalAddr)
		_, _ = b.Conn.WriteToUDP(rawGood, a.LocalAddr)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	mp, err := rx.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ap, ok := mp.(*media.AudioPacket)
	if !ok || len(ap.Payload) != 1 || ap.Payload[0] != 0x7F {
		t.Fatalf("got %T %#v", mp, mp)
	}
}

func TestNextRFC2833ShortPayloadIgnoredWhenNotEnd(t *testing.T) {
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

	tePT := uint8(101)
	rx := NewSIPRTPTransport(a, media.CodecConfig{Codec: "pcmu", SampleRate: 8000, PayloadType: 0}, media.DirectionInput, tePT)

	const ssrc2 = uint32(0x22222222)
	ev := []byte{3, 0}
	pkt := &RTPPacket{
		Header:  RTPHeader{Version: 2, PayloadType: tePT, SequenceNumber: 9, Timestamp: 100, SSRC: ssrc2},
		Payload: ev,
	}
	raw, _ := pkt.Marshal()

	go func() {
		time.Sleep(15 * time.Millisecond)
		_, _ = b.Conn.WriteToUDP(raw, a.LocalAddr)
		good := &RTPPacket{
			Header:  RTPHeader{Version: 2, PayloadType: 0, SequenceNumber: 10, Timestamp: 200, SSRC: ssrc2},
			Payload: []byte{0x55},
		}
		r2, _ := good.Marshal()
		_, _ = b.Conn.WriteToUDP(r2, a.LocalAddr)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	mp, err := rx.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ap, ok := mp.(*media.AudioPacket)
	if !ok || len(ap.Payload) != 1 || ap.Payload[0] != 0x55 {
		t.Fatalf("got %T", mp)
	}
}

func TestNextJitterBufferDeliversAudio(t *testing.T) {
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

	rx := NewSIPRTPTransport(a, media.CodecConfig{Codec: "pcmu", SampleRate: 8000, PayloadType: 0}, media.DirectionInput, 0)
	rx.JitterPlaybackDelay = 45 * time.Millisecond

	const ssrc = uint32(0x33333333)
	pkt := &RTPPacket{
		Header:  RTPHeader{Version: 2, PayloadType: 0, SequenceNumber: 44, Timestamp: 400, SSRC: ssrc},
		Payload: []byte{0x66},
	}
	raw, _ := pkt.Marshal()

	go func() {
		time.Sleep(15 * time.Millisecond)
		_, _ = b.Conn.WriteToUDP(raw, a.LocalAddr)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	mp, err := rx.Next(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ap, ok := mp.(*media.AudioPacket)
	if !ok || len(ap.Payload) != 1 || ap.Payload[0] != 0x66 {
		t.Fatalf("got %T %#v err=%v", mp, mp, err)
	}
}

func TestJitterJBPLCNonPCM(t *testing.T) {
	tr := &SIPRTPTransport{codec: media.CodecConfig{Codec: "opus"}, JitterPlaybackDelay: time.Millisecond}
	tr.jbRememberPLC([]byte{1, 2, 3})
	if tr.jbPLCReplacement() != nil {
		t.Fatal("opus should not PLC repeat")
	}
	tr.codec.Codec = "pcmu"
	if len(tr.jbPLCReplacement()) != 3 {
		t.Fatal("pcmu PLC")
	}
}

func TestJitterJBResetClears(t *testing.T) {
	tr := &SIPRTPTransport{JitterPlaybackDelay: time.Millisecond}
	tr.jbStarted = true
	tr.jbAdaptiveDelay = time.Second
	tr.jbSlots[3].valid = true
	tr.jbReset()
	if tr.jbStarted || tr.jbAdaptiveDelay != 0 || tr.jbSlots[3].valid {
		t.Fatal("reset incomplete")
	}
}

func TestJitterJBExpireStale(t *testing.T) {
	tr := &SIPRTPTransport{JitterPlaybackDelay: time.Millisecond}
	tr.jbSlots[1].valid = true
	tr.jbSlots[1].held.at = time.Now().Add(-time.Hour)
	tr.jbExpireStale(time.Now())
	if tr.jbSlots[1].valid {
		t.Fatal("expected expiry")
	}
}

func TestJitterJBSlotCollisionDrop(t *testing.T) {
	tr := &SIPRTPTransport{JitterPlaybackDelay: 20 * time.Millisecond, codec: media.CodecConfig{PayloadType: 0}}
	p1 := &RTPPacket{Header: RTPHeader{Version: 2, PayloadType: 0, SequenceNumber: 100}, Payload: []byte{1}}
	p2 := &RTPPacket{Header: RTPHeader{Version: 2, PayloadType: 0, SequenceNumber: uint16(100 + jbSlotCount)}, Payload: []byte{2}}
	now := time.Now()
	tr.jbPush(p1, now)
	tr.jbPush(p2, now)
	idx := int(uint16(100)) % jbSlotCount
	if !tr.jbSlots[idx].valid || tr.jbSlots[idx].seq != 100 {
		t.Fatal("first packet should remain on collision drop-incoming policy")
	}
}

func TestJitterHoleRaisesDelay(t *testing.T) {
	tr := &SIPRTPTransport{JitterPlaybackDelay: 50 * time.Millisecond}
	tr.jbAdaptiveDelay = 50 * time.Millisecond
	tr.jbNoteHole()
	if tr.jbAdaptiveDelay <= 50*time.Millisecond {
		t.Fatalf("delay %v", tr.jbAdaptiveDelay)
	}
}

func TestJitterTryPopWithoutStart(t *testing.T) {
	tr := &SIPRTPTransport{JitterPlaybackDelay: time.Millisecond}
	if tr.jbTryPop(time.Now()) != nil {
		t.Fatal("expected nil")
	}
}

func TestInputCallbacksNil(t *testing.T) {
	var tr *SIPRTPTransport
	tr.inputCallbacks(nil)
	tr = &SIPRTPTransport{}
	tr.inputCallbacks(&RTPPacket{Payload: []byte{}})
	tr.inputCallbacks(&RTPPacket{Payload: []byte{1}})
}

func TestSessLocalRemoteNil(t *testing.T) {
	var tr *SIPRTPTransport
	if tr.sessLocalAddr() != nil || tr.sessRemoteAddr() != nil {
		t.Fatal()
	}
	tr = &SIPRTPTransport{}
	if tr.sessLocalAddr() != nil {
		t.Fatal()
	}
}

func TestClearReadDeadlineNil(t *testing.T) {
	var tr *SIPRTPTransport
	tr.clearReadDeadline()
}

func TestWakeupReadNil(t *testing.T) {
	var tr *SIPRTPTransport
	tr.WakeupRead()
}

func TestAttachTransport(t *testing.T) {
	tr := &SIPRTPTransport{}
	tr.Attach(nil)
}

func TestTransportOutputNext(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	tr := NewSIPRTPTransport(s, media.CodecConfig{}, media.DirectionOutput, 0)
	p, err := tr.Next(nil)
	if p != nil || err != nil {
		t.Fatalf("%v %v", p, err)
	}
}

func TestTransportClosePreserve(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	tr := NewSIPRTPTransport(s, media.CodecConfig{}, media.DirectionInput, 0)
	tr.PreserveSessionOnClose = true
	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
}

func TestTransportSendWrongDirAndEmpty(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	tr := NewSIPRTPTransport(s, media.CodecConfig{}, media.DirectionInput, 0)
	if n, err := tr.Send(nil, nil); n != 0 || err != nil {
		t.Fatal()
	}
	tr = NewSIPRTPTransport(s, media.CodecConfig{}, media.DirectionOutput, 0)
	if n, err := tr.Send(nil, &media.TextPacket{Text: "x"}); n != 0 || err != nil {
		t.Fatal()
	}
	if n, err := tr.Send(nil, &media.AudioPacket{}); n != 0 || err != nil {
		t.Fatal()
	}
}

func TestSessionStatsNil(t *testing.T) {
	var s *Session
	if s.StatsSnapshot().TXPackets != 0 {
		t.Fatal()
	}
}

func TestJitterPopOrderedAfterDelay(t *testing.T) {
	tr := &SIPRTPTransport{JitterPlaybackDelay: 80 * time.Millisecond, codec: media.CodecConfig{PayloadType: 0}}
	pkt := &RTPPacket{Header: RTPHeader{Version: 2, PayloadType: 0, SequenceNumber: 77, Timestamp: 1000}, Payload: []byte{9}}
	tr.jbPush(pkt, time.Now().Add(-120*time.Millisecond))
	out := tr.jbTryPop(time.Now())
	if out == nil || len(out.Payload) != 1 || out.Payload[0] != 9 {
		t.Fatalf("got %#v", out)
	}
}

func TestJitterLossHolePLC(t *testing.T) {
	tr := &SIPRTPTransport{JitterPlaybackDelay: time.Millisecond, codec: media.CodecConfig{Codec: "pcmu", PayloadType: 0}}
	tr.jbStarted = true
	tr.jbNext = 10
	tr.jbRememberPLC([]byte{0x7F, 0x7F})
	tr.jbLossWait = time.Now().Add(-200 * time.Millisecond)
	tr.jbHoleSkips = 0
	out := tr.jbTryPop(time.Now())
	if out == nil || len(out.Payload) != 2 {
		t.Fatalf("plc %#v", out)
	}
}

func TestAddMirrorRemoteInvalid(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.AddMirrorRemote(nil, time.Second)
	s.AddMirrorRemote(&net.UDPAddr{Port: -1}, time.Second)
}

func TestRTPMarshalDefaultsVersion(t *testing.T) {
	p := &RTPPacket{Header: RTPHeader{PayloadType: 0}, Payload: []byte{1}}
	b, err := p.Marshal()
	if err != nil || len(b) < 13 || (b[0]>>6)&3 != 2 {
		t.Fatalf("err=%v", err)
	}
}

func TestTransportSendNilSession(t *testing.T) {
	tr := NewSIPRTPTransport(nil, media.CodecConfig{}, media.DirectionOutput, 0)
	if _, err := tr.Send(nil, &media.AudioPacket{Payload: []byte{1}}); err == nil {
		t.Fatal("expected error")
	}
}

func TestTransportNextNilSession(t *testing.T) {
	tr := NewSIPRTPTransport(nil, media.CodecConfig{}, media.DirectionInput, 0)
	if _, err := tr.Next(nil); err == nil {
		t.Fatal("expected error")
	}
}
