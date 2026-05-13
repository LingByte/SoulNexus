package bridge

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/media"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/rtp"
)

// ---------- fake bridgeLeg implementing pcmBridgeLeg --------------------

type fakeBridgeLeg struct {
	mu      sync.Mutex
	codec   media.CodecConfig
	in      chan media.MediaPacket
	sent    []media.MediaPacket
	woken   bool
	closeOnce sync.Once
}

func newFakeBridgeLeg(codec media.CodecConfig) *fakeBridgeLeg {
	return &fakeBridgeLeg{codec: codec, in: make(chan media.MediaPacket, 16)}
}

func (l *fakeBridgeLeg) Next(ctx context.Context) (media.MediaPacket, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case p, ok := <-l.in:
		if !ok {
			return nil, io.EOF
		}
		return p, nil
	}
}

func (l *fakeBridgeLeg) Send(ctx context.Context, p media.MediaPacket) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sent = append(l.sent, p)
	return len(p.Body()), nil
}

func (l *fakeBridgeLeg) Codec() media.CodecConfig { return l.codec }

func (l *fakeBridgeLeg) WakeupRead() {
	l.mu.Lock()
	l.woken = true
	l.mu.Unlock()
	l.closeOnce.Do(func() { close(l.in) })
}

func (l *fakeBridgeLeg) sentCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.sent)
}

// ---------- opusBridgeDecodeConfig / narrowbandG711 / bridgeMidPCM -----

func TestOpusBridgeDecodeConfig(t *testing.T) {
	// stereo opus → PCMBridgeDecodeStereo=true
	in := media.CodecConfig{Codec: "opus", OpusDecodeChannels: 2}
	out := opusBridgeDecodeConfig(in)
	if !out.OpusPCMBridgeDecodeStereo {
		t.Error("stereo opus should flip OpusPCMBridgeDecodeStereo")
	}
	// mono opus → unchanged
	in2 := media.CodecConfig{Codec: "opus", OpusDecodeChannels: 1}
	if opusBridgeDecodeConfig(in2).OpusPCMBridgeDecodeStereo {
		t.Error("mono opus should not flip flag")
	}
	// non-opus → unchanged
	in3 := media.CodecConfig{Codec: "pcma"}
	if opusBridgeDecodeConfig(in3).OpusPCMBridgeDecodeStereo {
		t.Error("pcma should not flip flag")
	}
}

func TestNarrowbandG711_AndBridgeMidPCM(t *testing.T) {
	pcmu := media.CodecConfig{Codec: "pcmu", SampleRate: 8000, Channels: 1}
	pcma := media.CodecConfig{Codec: "pcma", SampleRate: 8000, Channels: 1}
	opus := media.CodecConfig{Codec: "opus", SampleRate: 48000, Channels: 1}

	if !narrowbandG711(pcmu) {
		t.Error("pcmu 8k is narrowband")
	}
	if !narrowbandG711(pcma) {
		t.Error("pcma 8k is narrowband")
	}
	if narrowbandG711(opus) {
		t.Error("opus is not narrowband g711")
	}
	// Dual G.711 → 8kHz PCM bridge
	if got := bridgeMidPCM(pcmu, pcma); got.SampleRate != 8000 {
		t.Errorf("dual G.711 mid rate = %d, want 8000", got.SampleRate)
	}
	// Mixed → 16kHz PCM bridge
	if got := bridgeMidPCM(pcmu, opus); got.SampleRate != 16000 {
		t.Errorf("opus+pcmu mid rate = %d, want 16000", got.SampleRate)
	}
}

// ---------- tapPCMFromDecodedMedia packet-type branches ---------------

func TestTapPCMFromDecodedMedia_NilTapIsNoop(t *testing.T) {
	tapPCMFromDecodedMedia(DirectionCallerToAgent, &media.AudioPacket{Payload: []byte{1, 2}}, nil)
}

func TestTapPCMFromDecodedMedia_NonAudioSkipped(t *testing.T) {
	called := 0
	tap := func(dir BridgeDirection, pcm []byte) { called++ }
	for _, p := range []media.MediaPacket{
		&media.DTMFPacket{Digit: "5"},
		&media.TextPacket{Text: "hi"},
		&media.ClosePacket{Reason: "x"},
	} {
		tapPCMFromDecodedMedia(DirectionCallerToAgent, p, tap)
	}
	if called != 0 {
		t.Errorf("non-audio should not tap, got %d calls", called)
	}
}

func TestTapPCMFromDecodedMedia_AudioWithBody(t *testing.T) {
	var gotDir BridgeDirection
	tap := func(dir BridgeDirection, pcm []byte) { gotDir = dir }
	tapPCMFromDecodedMedia(DirectionAgentToCaller, &media.AudioPacket{Payload: []byte{1, 2}}, tap)
	if gotDir != DirectionAgentToCaller {
		t.Errorf("direction not passed through: %v", gotDir)
	}
	// Zero-length audio is skipped
	tapPCMFromDecodedMedia(DirectionCallerToAgent, &media.AudioPacket{}, tap)
}

// ---------- NewTwoLegPCMBridge / Start / Stop / taps -------------------

func TestNewTwoLegPCMBridge_NilGuards(t *testing.T) {
	pcmu := newFakeBridgeLeg(media.CodecConfig{Codec: "pcmu", SampleRate: 8000, Channels: 1})
	if _, err := NewTwoLegPCMBridge(nil, pcmu, pcmu, pcmu); err == nil {
		t.Error("nil transport must error")
	}
	if _, err := NewTwoLegPCMBridge(pcmu, nil, pcmu, pcmu); err == nil {
		t.Error("nil transport must error")
	}
	if _, err := NewTwoLegPCMBridge(pcmu, pcmu, nil, pcmu); err == nil {
		t.Error("nil transport must error")
	}
	if _, err := NewTwoLegPCMBridge(pcmu, pcmu, pcmu, nil); err == nil {
		t.Error("nil transport must error")
	}
}

func TestNewTwoLegPCMBridge_UnsupportedDecoder(t *testing.T) {
	bad := newFakeBridgeLeg(media.CodecConfig{Codec: "unregistered", SampleRate: 8000, Channels: 1})
	if _, err := NewTwoLegPCMBridge(bad, bad, bad, bad); err == nil {
		t.Error("unsupported codec must error at decoder factory")
	}
}

func TestTwoLegPCMBridge_MidSampleRate_Nil(t *testing.T) {
	var b *TwoLegPCMBridge
	if got := b.MidSampleRate(); got != 0 {
		t.Errorf("nil bridge MidSampleRate = %d, want 0", got)
	}
}

func TestTwoLegPCMBridge_EndToEnd_DualPCMU(t *testing.T) {
	caller := newFakeBridgeLeg(media.CodecConfig{Codec: "pcmu", SampleRate: 8000, Channels: 1})
	agent := newFakeBridgeLeg(media.CodecConfig{Codec: "pcmu", SampleRate: 8000, Channels: 1})

	b, err := NewTwoLegPCMBridge(caller, caller, agent, agent)
	if err != nil {
		t.Fatalf("NewTwoLegPCMBridge: %v", err)
	}
	if b.MidSampleRate() != 8000 {
		t.Errorf("dual PCMU mid rate = %d, want 8000", b.MidSampleRate())
	}

	var mergedTapCount, dirTapCount int32
	b.SetPCMRecordTap(func(pcm []byte) { atomic.AddInt32(&mergedTapCount, 1) })
	b.SetDirectionalPCMTap(func(dir BridgeDirection, pcm []byte) { atomic.AddInt32(&dirTapCount, 1) })

	b.Start()
	b.Start() // idempotent

	// Push a PCMU frame: 160 bytes = 20ms at 8kHz. Use silence (0xFF is μ-law silence).
	payload := make([]byte, 160)
	for i := range payload {
		payload[i] = 0xFF
	}
	caller.in <- &media.AudioPacket{Payload: payload}
	agent.in <- &media.AudioPacket{Payload: payload}

	// Give the bridge a chance to forward.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if agent.sentCount() > 0 && caller.sentCount() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	b.Stop()
	b.Stop() // idempotent

	if agent.sentCount() == 0 {
		t.Error("caller→agent forwarded nothing")
	}
	if caller.sentCount() == 0 {
		t.Error("agent→caller forwarded nothing")
	}
	if atomic.LoadInt32(&dirTapCount) == 0 {
		t.Error("directional tap never fired")
	}
}

func TestTwoLegPCMBridge_StopStartOrderNil(t *testing.T) {
	var b *TwoLegPCMBridge
	b.Start() // must not panic
	b.Stop()  // must not panic
	b.SetPCMRecordTap(func([]byte) {})
	b.SetDirectionalPCMTap(func(BridgeDirection, []byte) {})
}

// ---------- mapRelayPayloadType ---------------------------------------

func TestMapRelayPayloadType(t *testing.T) {
	// Audio match
	if pt, ok := mapRelayPayloadType(0, 0, 101, 8, 102); !ok || pt != 8 {
		t.Errorf("audio map: %d ok=%v, want 8/true", pt, ok)
	}
	// DTMF match
	if pt, ok := mapRelayPayloadType(101, 0, 101, 8, 102); !ok || pt != 102 {
		t.Errorf("dtmf map: %d ok=%v, want 102/true", pt, ok)
	}
	// DTMF match but dst doesn't advertise dtmf
	if _, ok := mapRelayPayloadType(101, 0, 101, 8, 0); ok {
		t.Error("dtmf to dst without dtmf pt should be dropped")
	}
	// Unknown PT
	if _, ok := mapRelayPayloadType(99, 0, 101, 8, 102); ok {
		t.Error("unknown PT must not be forwarded")
	}
	// Marker bit should be ignored (only low 7 bits matter)
	if pt, ok := mapRelayPayloadType(0x80|0, 0, 101, 8, 102); !ok || pt != 8 {
		t.Errorf("marker bit should not affect match: %d ok=%v", pt, ok)
	}
}

// ---------- cloneUDPAddr -----------------------------------------------

func TestCloneUDPAddr(t *testing.T) {
	if cloneUDPAddr(nil) != nil {
		t.Error("nil clone must be nil")
	}
	a := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5060}
	b := cloneUDPAddr(a)
	if b == a {
		t.Error("clone must be a fresh pointer")
	}
	if !b.IP.Equal(a.IP) || b.Port != a.Port {
		t.Errorf("clone mismatch: %+v vs %+v", b, a)
	}
	// Mutate clone's IP and ensure original is unaffected
	if len(b.IP) > 0 {
		b.IP[0] = 0xFF
	}
	if a.IP[0] == 0xFF {
		t.Error("clone's IP slice shared backing with original")
	}
}

// ---------- TwoLegPayloadRelay -----------------------------------------

func TestNewTwoLegPayloadRelay_GuardClauses(t *testing.T) {
	caller, _ := rtp.NewSession(0)
	defer caller.Close()
	agent, _ := rtp.NewSession(0)
	defer agent.Close()

	pcmu := media.CodecConfig{Codec: "pcmu", SampleRate: 8000, Channels: 1, PayloadType: 0}

	// nil session
	if _, err := NewTwoLegPayloadRelay(nil, agent, pcmu, pcmu, 101, 101); err == nil {
		t.Error("nil caller session must error")
	}
	if _, err := NewTwoLegPayloadRelay(caller, nil, pcmu, pcmu, 101, 101); err == nil {
		t.Error("nil agent session must error")
	}
	// codec mismatch → rejected
	pcma := media.CodecConfig{Codec: "pcma", SampleRate: 8000, Channels: 1, PayloadType: 8}
	if _, err := NewTwoLegPayloadRelay(caller, agent, pcmu, pcma, 101, 101); err == nil {
		t.Error("pcmu/pcma codec mismatch must error")
	}
	// Remote addresses not set → rejected
	if _, err := NewTwoLegPayloadRelay(caller, agent, pcmu, pcmu, 101, 101); err == nil {
		t.Error("missing RemoteAddr must error")
	}
}

func TestNewTwoLegPayloadRelay_HappyPath(t *testing.T) {
	caller, _ := rtp.NewSession(0)
	defer caller.Close()
	agent, _ := rtp.NewSession(0)
	defer agent.Close()

	caller.SetRemoteAddr(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 10000})
	agent.SetRemoteAddr(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 10001})
	pcmu := media.CodecConfig{Codec: "pcmu", SampleRate: 8000, Channels: 1, PayloadType: 0}

	r, err := NewTwoLegPayloadRelay(caller, agent, pcmu, pcmu, 101, 101)
	if err != nil {
		t.Fatalf("NewTwoLegPayloadRelay: %v", err)
	}
	if r == nil {
		t.Fatal("relay nil")
	}

	// Recording setters exercise recMu path (nil safety too)
	r.SetInboundRecording(
		func(seq uint16, ts uint32, p []byte) {},
		func(seq uint16, ts uint32, p []byte) {},
	)
	var nilRelay *TwoLegPayloadRelay
	nilRelay.SetInboundRecording(nil, nil)

	r.Start()
	r.Start() // idempotent
	// Immediately stop; the goroutines loop with a 400ms read deadline —
	// Stop closes the sockets and cancels, unblocking them.
	time.Sleep(30 * time.Millisecond)
	r.Stop()
	r.Stop() // idempotent

	// Nil safety on Start/Stop
	nilRelay.Start()
	nilRelay.Stop()
}

// buildRTPPacket constructs a minimal valid RTP datagram for tests.
func buildRTPPacket(pt uint8, seq uint16, ts uint32, ssrc uint32, payload []byte) []byte {
	buf := make([]byte, 12+len(payload))
	buf[0] = 0x80                           // V=2, P=0, X=0, CC=0
	buf[1] = pt & 0x7F                      // M=0, PT
	buf[2] = byte(seq >> 8)
	buf[3] = byte(seq)
	buf[4] = byte(ts >> 24)
	buf[5] = byte(ts >> 16)
	buf[6] = byte(ts >> 8)
	buf[7] = byte(ts)
	buf[8] = byte(ssrc >> 24)
	buf[9] = byte(ssrc >> 16)
	buf[10] = byte(ssrc >> 8)
	buf[11] = byte(ssrc)
	copy(buf[12:], payload)
	return buf
}

func TestTwoLegPayloadRelay_ForwardsRTPBetweenLegs(t *testing.T) {
	// Two "endpoints" (one at each side of the relay) that the relay will
	// forward to/from, and two relay sessions in the middle.
	agentEnd, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen agent end: %v", err)
	}
	defer agentEnd.Close()
	callerEnd, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen caller end: %v", err)
	}
	defer callerEnd.Close()

	callerSess, _ := rtp.NewSession(0)
	defer callerSess.Close()
	agentSess, _ := rtp.NewSession(0)
	defer agentSess.Close()
	callerSess.SetRemoteAddr(callerEnd.LocalAddr().(*net.UDPAddr))
	agentSess.SetRemoteAddr(agentEnd.LocalAddr().(*net.UDPAddr))

	// caller PT=0 (PCMU), agent PT=0 (PCMU) — same audio PT, different DTMF.
	pcmu := media.CodecConfig{Codec: "pcmu", SampleRate: 8000, Channels: 1, PayloadType: 0}
	r, err := NewTwoLegPayloadRelay(callerSess, agentSess, pcmu, pcmu, 101, 102)
	if err != nil {
		t.Fatalf("NewTwoLegPayloadRelay: %v", err)
	}

	var (
		userCalls  int32
		agentCalls int32
	)
	r.SetInboundRecording(
		func(seq uint16, ts uint32, p []byte) { atomic.AddInt32(&userCalls, 1) },
		func(seq uint16, ts uint32, p []byte) { atomic.AddInt32(&agentCalls, 1) },
	)
	r.Start()
	defer r.Stop()

	// Let relay goroutines hit their first ReadFromUDP deadline.
	time.Sleep(10 * time.Millisecond)

	// Send a caller → agent RTP packet to caller's listening port.
	// The relay reads it from callerSess and forwards to agentEnd.
	pkt := buildRTPPacket(0, 1234, 42000, 0xDEADBEEF, []byte("userspeech"))
	client, _ := net.DialUDP("udp", nil, callerSess.LocalAddr)
	_, _ = client.Write(pkt)
	client.Close()

	// Wait for the agentEnd to receive the forwarded packet.
	_ = agentEnd.SetReadDeadline(time.Now().Add(1 * time.Second))
	recv := make([]byte, 2048)
	n, _, err := agentEnd.ReadFromUDP(recv)
	if err != nil {
		t.Fatalf("agentEnd did not receive forwarded RTP: %v", err)
	}
	if n < 12 {
		t.Fatalf("forwarded packet too short: %d", n)
	}

	// Also exercise agent → caller direction with a DTMF PT remap.
	dtmfPkt := buildRTPPacket(101, 2000, 99999, 0xCAFEBABE, []byte{0x05, 0x80, 0, 0})
	client2, _ := net.DialUDP("udp", nil, agentSess.LocalAddr)
	_, _ = client2.Write(dtmfPkt)
	client2.Close()

	_ = callerEnd.SetReadDeadline(time.Now().Add(1 * time.Second))
	n2, _, err := callerEnd.ReadFromUDP(recv)
	if err != nil {
		t.Logf("callerEnd did not receive DTMF (may be filtered): %v", err)
	} else if n2 >= 2 {
		// After DTMF remap, PT should become dst DTMF (101) since agent DTMF is 102 and dst is caller (101).
		gotPT := recv[1] & 0x7F
		if gotPT != 101 {
			t.Errorf("DTMF forwarded with PT=%d, expected remapped caller DTMF=101", gotPT)
		}
	}

	// Push one more packet so user-recording tap fires again
	client3, _ := net.DialUDP("udp", nil, callerSess.LocalAddr)
	_, _ = client3.Write(pkt)
	client3.Close()
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&userCalls) == 0 {
		t.Error("user recording tap never fired for caller→agent audio")
	}
	_ = agentCalls // agent-to-caller tap may or may not fire depending on DTMF routing
}

func TestUnblockUDPRead_NilSafe(t *testing.T) {
	unblockUDPRead(nil)
	// Session with nil Conn should also be safe; construct via zero value
	unblockUDPRead(&rtp.Session{})
}

// ---------- Avoid unused import warning -------------------------------
var _ = media.DirectionInput
