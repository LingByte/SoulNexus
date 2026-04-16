package protocol

import (
	"context"
	"fmt"
	"net"
)

// DatagramTransport is the reserved abstraction for SIP message IO.
//
// Today we only implement UDP. For SIP over WebSocket, implement this interface
// with a WS-backed transport that frames SIP messages as datagrams.
type DatagramTransport interface {
	ReadFrom(ctx context.Context, buf []byte) (n int, addr *net.UDPAddr, err error)
	WriteTo(ctx context.Context, p []byte, addr *net.UDPAddr) (n int, err error)
	Close() error
	String() string
}

// UDPTransport adapts net.UDPConn to DatagramTransport.
type UDPTransport struct {
	conn *net.UDPConn
}

func NewUDPTransport(conn *net.UDPConn) *UDPTransport {
	return &UDPTransport{conn: conn}
}

func (t *UDPTransport) String() string { return "UDPTransport" }

func (t *UDPTransport) ReadFrom(ctx context.Context, buf []byte) (int, *net.UDPAddr, error) {
	if t == nil || t.conn == nil {
		return 0, nil, fmt.Errorf("sip: udp transport not started")
	}
	return t.conn.ReadFromUDP(buf)
}

func (t *UDPTransport) WriteTo(ctx context.Context, p []byte, addr *net.UDPAddr) (int, error) {
	if t == nil || t.conn == nil {
		return 0, fmt.Errorf("sip: udp transport not started")
	}
	return t.conn.WriteToUDP(p, addr)
}

func (t *UDPTransport) Close() error {
	if t == nil || t.conn == nil {
		return nil
	}
	return t.conn.Close()
}

// WebSocketTransport is a placeholder reserved for future SIP over WebSocket.
// It is intentionally not implemented yet.
type WebSocketTransport struct{}

func (t *WebSocketTransport) String() string { return "WebSocketTransport(not-implemented)" }

func (t *WebSocketTransport) ReadFrom(ctx context.Context, buf []byte) (int, *net.UDPAddr, error) {
	return 0, nil, fmt.Errorf("sip: websocket transport not implemented")
}

func (t *WebSocketTransport) WriteTo(ctx context.Context, p []byte, addr *net.UDPAddr) (int, error) {
	return 0, fmt.Errorf("sip: websocket transport not implemented")
}

func (t *WebSocketTransport) Close() error { return nil }

