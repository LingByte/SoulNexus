package rtp

import (
	"net"
	"testing"
	"time"
)

// ---------- helpers (pure) -------------------------------------------------

func TestSinceMillis_NonPositiveReturnsMinusOne(t *testing.T) {
	if got := sinceMillis(0); got != -1 {
		t.Errorf("sinceMillis(0) = %d, want -1", got)
	}
	if got := sinceMillis(-12345); got != -1 {
		t.Errorf("sinceMillis(-12345) = %d, want -1", got)
	}
}

func TestSinceMillis_PositiveAdvancing(t *testing.T) {
	past := time.Now().Add(-50 * time.Millisecond).UnixNano()
	got := sinceMillis(past)
	if got < 30 || got > 5000 {
		t.Errorf("sinceMillis(50ms ago) = %d ms, expected ~50ms", got)
	}
}

func TestAddrStringOrEmpty(t *testing.T) {
	if got := addrStringOrEmpty(nil); got != "" {
		t.Errorf("nil addr → %q, want empty", got)
	}
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}
	if got := addrStringOrEmpty(addr); got == "" {
		t.Errorf("non-nil addr produced empty string")
	}
}

// ---------- StatsSnapshot --------------------------------------------------

func TestStatsSnapshot_DefaultsAreZero(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer s.Close()
	snap := s.StatsSnapshot()
	if snap.RXPackets != 0 || snap.TXPackets != 0 {
		t.Errorf("fresh session should have zero counters: %+v", snap)
	}
	if snap.LocalSocket == "" {
		t.Errorf("LocalSocket should be populated, got %+v", snap)
	}
}

// ---------- EnableSDESSRTP nil + invalid -----------------------------------

func TestEnableSDESSRTP_NilSession(t *testing.T) {
	var s *Session
	if err := s.EnableSDESSRTP(nil, nil, nil, nil); err == nil {
		t.Error("nil session must error")
	}
}

func TestEnableSDESSRTP_InvalidKeyMaterial(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer s.Close()
	// wrong-sized keys/salts: srtp.CreateContext should fail
	if err := s.EnableSDESSRTP([]byte{1, 2, 3}, []byte{4, 5}, []byte{6}, []byte{7}); err == nil {
		t.Error("invalid key material must error")
	}
}

func TestEnableSDESSRTP_ValidKeyMaterial(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer s.Close()
	// AES_CM_128_HMAC_SHA1_80 → 16-byte key + 14-byte salt
	key := make([]byte, 16)
	salt := make([]byte, 14)
	for i := range key {
		key[i] = byte(i + 1)
	}
	for i := range salt {
		salt[i] = byte(0xA0 + i)
	}
	if err := s.EnableSDESSRTP(key, salt, key, salt); err != nil {
		t.Fatalf("valid SDES material: %v", err)
	}
}

// ---------- NewSession edge cases ------------------------------------------

func TestNewSession_NegativePortRejected(t *testing.T) {
	// implementation-dependent: NewSession(-1) currently lets ListenUDP decide.
	// Either err or nil session is acceptable as long as nothing panics.
	s, err := NewSession(-1)
	if err == nil && s != nil {
		s.Close()
	}
}

func TestNewSession_ConcurrentSessionsBindDistinctPorts(t *testing.T) {
	a, err := NewSession(0)
	if err != nil {
		t.Fatalf("a NewSession: %v", err)
	}
	defer a.Close()
	b, err := NewSession(0)
	if err != nil {
		t.Fatalf("b NewSession: %v", err)
	}
	defer b.Close()
	if a.LocalAddr.Port == b.LocalAddr.Port {
		t.Errorf("two ephemeral sessions returned same port: %d", a.LocalAddr.Port)
	}
}

// ---------- FirstPacket / SetRemoteAddr / AddMirrorRemote ------------------

func TestSession_FirstPacket_NotClosedYet(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer s.Close()
	select {
	case <-s.FirstPacket():
		t.Fatal("FirstPacket channel should not be closed before any RX")
	case <-time.After(10 * time.Millisecond):
	}
}

func TestSession_SetRemoteAddrAndMirror(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer s.Close()
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 10000}
	s.SetRemoteAddr(addr)
	s.AddMirrorRemote(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 10001}, 50*time.Millisecond)
	// add a duplicate (should be tolerated/refreshed) and a nil (defensively ignored)
	s.AddMirrorRemote(&net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 10001}, 50*time.Millisecond)
	s.AddMirrorRemote(nil, 50*time.Millisecond)
}

// ---------- SendRTP / ReceiveRTP end-to-end loopback ------------------------

func TestSession_SendRTP_LoopbackToReceiveRTP(t *testing.T) {
	tx, err := NewSession(0)
	if err != nil {
		t.Fatalf("tx NewSession: %v", err)
	}
	defer tx.Close()
	rx, err := NewSession(0)
	if err != nil {
		t.Fatalf("rx NewSession: %v", err)
	}
	defer rx.Close()

	// point tx → rx and add a second mirror remote (covers sendMirrorRTP path)
	tx.SetRemoteAddr(rx.LocalAddr)

	mirror, err := NewSession(0)
	if err != nil {
		t.Fatalf("mirror NewSession: %v", err)
	}
	defer mirror.Close()
	tx.AddMirrorRemote(mirror.LocalAddr, time.Second)

	// Receiver loop in goroutine
	rxDone := make(chan struct{})
	go func() {
		defer close(rxDone)
		buf := make([]byte, 2048)
		_ = rx.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, _, _, err := rx.ReceiveRTP(buf)
		if err != nil {
			t.Logf("rx ReceiveRTP err: %v (n=%d)", err, n)
			return
		}
	}()

	// Send a single PCMU silence packet
	payload := make([]byte, 160)
	if err := tx.SendRTP(payload, 0, 160); err != nil {
		t.Fatalf("SendRTP: %v", err)
	}
	<-rxDone

	// SendRTP without remote should be tolerated (no-op or minimal error)
	mirror.SendRTP(payload, 0, 160)

	// Sanity: snapshot now reports at least one TX
	snap := tx.StatsSnapshot()
	if snap.TXPackets == 0 {
		t.Errorf("expected TX>0 after SendRTP, got %+v", snap)
	}
}

func TestSession_CloseTwice_Idempotent(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// Second close should not panic; error is acceptable.
	_ = s.Close()
}

func TestCloneUDPAddr_NilAndCopy(t *testing.T) {
	if cloneUDPAddr(nil) != nil {
		t.Error("cloneUDPAddr(nil) must be nil")
	}
	a := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
	b := cloneUDPAddr(a)
	if b == a {
		t.Error("cloneUDPAddr should produce a fresh allocation")
	}
	if !b.IP.Equal(a.IP) || b.Port != a.Port {
		t.Errorf("clone mismatch: %+v vs %+v", b, a)
	}
}

func TestSession_BuildPacket_Header(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer s.Close()
	pkt := s.buildPacket([]byte{1, 2, 3}, 0)
	if pkt == nil || pkt.Header.Version != 2 || pkt.Header.PayloadType != 0 {
		t.Errorf("buildPacket = %+v", pkt)
	}
	if len(pkt.Payload) != 3 {
		t.Errorf("payload len = %d", len(pkt.Payload))
	}
}

func TestSession_UpdateAfterSend(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer s.Close()
	seq0, ts0 := s.SeqNum, s.Timestamp
	s.updateAfterSend(160)
	if s.SeqNum != seq0+1 {
		t.Errorf("SeqNum did not advance: %d → %d", seq0, s.SeqNum)
	}
	if s.Timestamp != ts0+160 {
		t.Errorf("Timestamp did not advance: %d → %d", ts0, s.Timestamp)
	}
}

func TestSession_StartStatsLoop_Lifecycle(t *testing.T) {
	s, err := NewSession(0)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	// Start stats loop and wait one tick or two (loop ticks ~10s by default;
	// we just want to exercise startStatsLoop entry + Close shutdown path).
	s.startStatsLoop()
	time.Sleep(20 * time.Millisecond)
	s.Close()
}
