package rtp

import (
	"fmt"
	"net"
	"sync"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"go.uber.org/zap"
)

// Session is a minimal RTP-over-UDP session.
//
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

	logFirstUDP sync.Once
	logFirstTX  sync.Once
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
	}, nil
}

// FirstPacket returns a channel that is closed once the first RTP packet is received.
func (s *Session) FirstPacket() <-chan struct{} {
	return s.firstPacketCh
}

// SetRemoteAddr sets the remote RTP address for outgoing packets.
func (s *Session) SetRemoteAddr(addr *net.UDPAddr) {
	s.RemoteAddr = addr
	if s.sdpRemote == nil && addr != nil {
		s.sdpRemote = cloneUDPAddr(addr)
	}
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
	if s.Conn != nil {
		return s.Conn.Close()
	}
	return nil
}

