package rtp

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// Session is a minimal RTP-over-UDP session.
// It is intentionally protocol-agnostic:
// - Timestamp increments are provided by the caller via `samples` argument.
// - Payload framing / codec packetization happens above this layer.
type Session struct {
	LocalAddr  *net.UDPAddr
	RemoteAddr *net.UDPAddr
	Conn       *net.UDPConn

	// sdpRemote is a copy of the first SetRemoteAddr (from SDP c=/m=). Used only for logs vs symmetric RTP.
	sdpRemote *net.UDPAddr

	// SSRC/sequence/timestamp are advanced by this session.
	SSRC      uint32
	SeqNum    uint16
	Timestamp uint32

	// UDP read signal for "first packet received".
	firstPacketOnce sync.Once
	firstPacketCh   chan struct{}

	logFirstUDP  sync.Once
	logFirstTX   sync.Once
	logStatsOnce sync.Once
	closeOnce    sync.Once
	statsStopCh  chan struct{}

	txPackets uint64
	txBytes   uint64
	rxPackets uint64
	rxBytes   uint64

	firstTxUnixNano int64
	firstRxUnixNano int64
	natWarned       uint32

	mirrorMu      sync.RWMutex
	mirrorRemotes []mirrorRemote
}

type mirrorRemote struct {
	addr      *net.UDPAddr
	expiresAt time.Time
}

type SessionStats struct {
	LocalSocket string
	RemoteSDP   string
	RemoteNow   string
	TXPackets   uint64
	TXBytes     uint64
	RXPackets   uint64
	RXBytes     uint64
	FirstTXAgo  int64
	FirstRXAgo  int64
}

// NewSession creates a RTP UDP session.
//
// If localPort is 0 or negative, the OS will choose an available ephemeral port.
func NewSession(localPort int) (*Session, error) {
	addr := &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: localPort,
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("rtp: listen udp: %w", err)
	}

	return &Session{
		LocalAddr:     conn.LocalAddr().(*net.UDPAddr),
		Conn:          conn,
		SSRC:          0x12345678,
		SeqNum:        0,
		Timestamp:     0,
		firstPacketCh: make(chan struct{}),
		statsStopCh:   make(chan struct{}),
	}, nil
}

// FirstPacket returns a channel that is closed once the first RTP packet is received.
func (s *Session) FirstPacket() <-chan struct{} {
	return s.firstPacketCh
}

func (s *Session) StatsSnapshot() SessionStats {
	if s == nil {
		return SessionStats{}
	}
	return SessionStats{
		LocalSocket: addrStringOrEmpty(s.LocalAddr),
		RemoteSDP:   addrStringOrEmpty(s.sdpRemote),
		RemoteNow:   addrStringOrEmpty(s.RemoteAddr),
		TXPackets:   atomic.LoadUint64(&s.txPackets),
		TXBytes:     atomic.LoadUint64(&s.txBytes),
		RXPackets:   atomic.LoadUint64(&s.rxPackets),
		RXBytes:     atomic.LoadUint64(&s.rxBytes),
		FirstTXAgo:  sinceMillis(atomic.LoadInt64(&s.firstTxUnixNano)),
		FirstRXAgo:  sinceMillis(atomic.LoadInt64(&s.firstRxUnixNano)),
	}
}

// SetRemoteAddr sets the remote RTP address for outgoing packets.
func (s *Session) SetRemoteAddr(addr *net.UDPAddr) {
	s.RemoteAddr = addr
	if s.sdpRemote == nil && addr != nil {
		s.sdpRemote = cloneUDPAddr(addr)
	}
}

// AddMirrorRemote adds a temporary extra RTP destination for outbound packets.
// Useful for NAT/ALG scenarios where real media port differs from SDP offer.
func (s *Session) AddMirrorRemote(addr *net.UDPAddr, ttl time.Duration) {
	if s == nil || addr == nil || addr.IP == nil || addr.Port <= 0 || ttl <= 0 {
		return
	}
	exp := time.Now().Add(ttl)
	cp := cloneUDPAddr(addr)
	s.mirrorMu.Lock()
	defer s.mirrorMu.Unlock()
	// refresh existing mirror target if present
	for i := range s.mirrorRemotes {
		m := s.mirrorRemotes[i]
		if m.addr != nil && m.addr.IP.Equal(cp.IP) && m.addr.Port == cp.Port {
			s.mirrorRemotes[i].expiresAt = exp
			return
		}
	}
	s.mirrorRemotes = append(s.mirrorRemotes, mirrorRemote{
		addr:      cp,
		expiresAt: exp,
	})
}

func cloneUDPAddr(a *net.UDPAddr) *net.UDPAddr {
	if a == nil {
		return nil
	}
	b := *a
	return &b
}

func (s *Session) buildPacket(payload []byte, payloadType uint8) *RTPPacket {
	return &RTPPacket{
		Header: RTPHeader{
			Version:        2,
			Padding:        false,
			Extension:      false,
			CSRCCount:      0,
			Marker:         false,
			PayloadType:    payloadType,
			SequenceNumber: s.SeqNum,
			Timestamp:      s.Timestamp,
			SSRC:           s.SSRC,
		},
		Payload: payload,
	}
}

func (s *Session) updateAfterSend(samples uint32) {
	s.SeqNum++
	// RTP timestamp is measured in units of the codec's sampling clock.
	s.Timestamp += samples
}

// SendRTP sends one RTP packet.
//
// `samples` is the number of audio samples represented by `payload` at the RTP clock rate.
// For PCM-based codecs, this should match the negotiated codec frame duration.
func (s *Session) SendRTP(payload []byte, payloadType uint8, samples uint32) error {
	if s == nil {
		return fmt.Errorf("rtp: nil session")
	}
	if s.Conn == nil {
		return fmt.Errorf("rtp: nil udp conn")
	}
	if s.RemoteAddr == nil {
		return fmt.Errorf("rtp: remote address not set")
	}

	pkt := s.buildPacket(payload, payloadType)
	data, err := pkt.Marshal()
	if err != nil {
		return fmt.Errorf("rtp: marshal: %w", err)
	}

	if _, err := s.Conn.WriteToUDP(data, s.RemoteAddr); err != nil {
		return fmt.Errorf("rtp: send: %w", err)
	}
	s.sendMirrorRTP(data, s.RemoteAddr)
	atomic.AddUint64(&s.txPackets, 1)
	atomic.AddUint64(&s.txBytes, uint64(len(payload)))
	nowUnix := time.Now().UnixNano()
	_ = atomic.CompareAndSwapInt64(&s.firstTxUnixNano, 0, nowUnix)
	s.startStatsLoop()

	s.logFirstTX.Do(func() {
		if logger.Lg != nil {
			logger.Lg.Info("rtp first outbound packet (diagnostics)",
				zap.String("to", s.RemoteAddr.String()),
				zap.String("local_socket", s.LocalAddr.String()),
				zap.Int("payload_bytes", len(payload)),
				zap.Uint8("payload_type", payloadType),
			)
		}
	})

	s.updateAfterSend(samples)
	return nil
}

// ReceiveRTP reads a UDP datagram and parses it into an RTPPacket.
//
// It also opportunistically "learns" remote address (symmetric RTP behavior).
func (s *Session) ReceiveRTP(buffer []byte) (n int, from *net.UDPAddr, packet *RTPPacket, err error) {
	if s == nil {
		return 0, nil, nil, fmt.Errorf("rtp: nil session")
	}
	if s.Conn == nil {
		return 0, nil, nil, fmt.Errorf("rtp: nil udp conn")
	}

	n, addr, err := s.Conn.ReadFromUDP(buffer)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("rtp: read udp: %w", err)
	}
	atomic.AddUint64(&s.rxPackets, 1)
	atomic.AddUint64(&s.rxBytes, uint64(n))
	_ = atomic.CompareAndSwapInt64(&s.firstRxUnixNano, 0, time.Now().UnixNano())
	s.startStatsLoop()

	s.logFirstUDP.Do(func() {
		if logger.Lg != nil {
			logger.Lg.Info("rtp first udp datagram on media socket (diagnostics)",
				zap.String("from", addr.String()),
				zap.String("local_socket", s.LocalAddr.String()),
				zap.Int("bytes", n),
			)
		}
	})

	s.firstPacketOnce.Do(func() {
		close(s.firstPacketCh)
	})

	before := s.RemoteAddr
	if s.RemoteAddr == nil || !s.RemoteAddr.IP.Equal(addr.IP) || s.RemoteAddr.Port != addr.Port {
		s.RemoteAddr = addr
		if logger.Lg != nil && before != nil &&
			(before.IP.String() != addr.IP.String() || before.Port != addr.Port) {
			logger.Lg.Info("rtp symmetric path: send target updated to source of first received packet (NAT)",
				zap.String("sdp_remote_was", func() string {
					if s.sdpRemote != nil {
						return s.sdpRemote.String()
					}
					return ""
				}()),
				zap.String("previous_send_target", before.String()),
				zap.String("learned_remote", addr.String()),
			)
		}
	}

	pkt := &RTPPacket{}
	if err := pkt.Unmarshal(buffer[:n]); err != nil {
		return n, addr, nil, fmt.Errorf("rtp: unmarshal: %w", err)
	}

	return n, addr, pkt, nil
}

func (s *Session) Close() error {
	if s == nil {
		return nil
	}
	var err error
	s.closeOnce.Do(func() {
		if s.statsStopCh != nil {
			close(s.statsStopCh)
		}
		if s.Conn != nil {
			err = s.Conn.Close()
		}
	})
	return err
}

func (s *Session) sendMirrorRTP(data []byte, primary *net.UDPAddr) {
	if s == nil || s.Conn == nil || len(data) == 0 {
		return
	}
	now := time.Now()
	s.mirrorMu.Lock()
	if len(s.mirrorRemotes) == 0 {
		s.mirrorMu.Unlock()
		return
	}
	live := s.mirrorRemotes[:0]
	for _, m := range s.mirrorRemotes {
		if m.addr == nil || m.addr.IP == nil || m.addr.Port <= 0 || !m.expiresAt.After(now) {
			continue
		}
		live = append(live, m)
	}
	s.mirrorRemotes = live
	remotes := make([]*net.UDPAddr, 0, len(live))
	for _, m := range live {
		remotes = append(remotes, cloneUDPAddr(m.addr))
	}
	s.mirrorMu.Unlock()

	for _, r := range remotes {
		if r == nil {
			continue
		}
		if primary != nil && primary.IP != nil && primary.IP.Equal(r.IP) && primary.Port == r.Port {
			continue
		}
		_, _ = s.Conn.WriteToUDP(data, r)
	}
}

func (s *Session) startStatsLoop() {
	if s == nil {
		return
	}
	s.logStatsOnce.Do(func() {
		go s.statsLoop()
	})
}

func (s *Session) statsLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.statsStopCh:
			return
		case <-ticker.C:
			if logger.Lg == nil {
				continue
			}
			txP := atomic.LoadUint64(&s.txPackets)
			rxP := atomic.LoadUint64(&s.rxPackets)
			if txP == 0 && rxP == 0 {
				continue
			}
			txB := atomic.LoadUint64(&s.txBytes)
			rxB := atomic.LoadUint64(&s.rxBytes)
			firstTx := atomic.LoadInt64(&s.firstTxUnixNano)
			if txP > 0 && rxP == 0 && firstTx > 0 && time.Since(time.Unix(0, firstTx)) >= 10*time.Second {
				if atomic.CompareAndSwapUint32(&s.natWarned, 0, 1) {
					logger.Lg.Warn("rtp nat suspected: outbound active but inbound silent",
						zap.String("local_socket", addrStringOrEmpty(s.LocalAddr)),
						zap.String("remote_target", addrStringOrEmpty(s.RemoteAddr)),
						zap.Uint64("tx_packets", txP),
						zap.Uint64("tx_bytes", txB),
						zap.Uint64("rx_packets", rxP),
						zap.Uint64("rx_bytes", rxB),
						zap.Int64("first_tx_ms_ago", sinceMillis(firstTx)),
					)
				}
			}
		}
	}
}

func sinceMillis(unixNano int64) int64 {
	if unixNano <= 0 {
		return -1
	}
	return time.Since(time.Unix(0, unixNano)).Milliseconds()
}

func addrStringOrEmpty(a *net.UDPAddr) string {
	if a == nil {
		return ""
	}
	return a.String()
}
