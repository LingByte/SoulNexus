package protocol

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// HandlerFunc handles an incoming SIP request and returns a response message (or nil).
//
// The returned response should have IsRequest=false and valid StatusCode/StatusText.
type HandlerFunc func(msg *Message, addr *net.UDPAddr) *Message

// Server is a minimal SIP UDP server.
//
// It does not implement SIP transactions (retransmissions, timers, digest auth, etc.).
// It focuses on parsing and dispatching.
type Server struct {
	Host string
	Port int

	Conn *net.UDPConn

	Handlers       map[string]HandlerFunc
	NoRouteHandler HandlerFunc

	// Optional event hooks
	OnDatagram func(raw []byte, addr *net.UDPAddr)
	OnParseErr func(raw []byte, addr *net.UDPAddr, err error)
	OnRequest  func(req *Message, addr *net.UDPAddr)
	OnResponse func(req *Message, resp *Message, addr *net.UDPAddr)
	// OnSIPResponse is invoked for every SIP response received on the socket (UAC / outbound legs).
	// If nil, responses are ignored (UAS-only mode).
	OnSIPResponse func(resp *Message, addr *net.UDPAddr)
	OnEvent       func(e Event)

	readBufSize int
}

func NewServer(host string, port int) *Server {
	return &Server{
		Host:           host,
		Port:           port,
		Handlers:       make(map[string]HandlerFunc),
		readBufSize:   65535,
		NoRouteHandler: nil,
	}
}

func (s *Server) RegisterHandler(method string, h HandlerFunc) {
	if s == nil {
		return
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	s.Handlers[method] = h
}

func (s *Server) RegisterNoRoute(h HandlerFunc) {
	if s == nil {
		return
	}
	s.NoRouteHandler = h
}

func (s *Server) Start() error {
	addr := &net.UDPAddr{
		IP:   net.ParseIP(s.Host),
		Port: s.Port,
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("sip: listen udp: %w", err)
	}
	s.Conn = conn

	go s.readLoop()
	return nil
}

func (s *Server) Stop() error {
	if s == nil {
		return nil
	}
	if s.Conn == nil {
		return nil
	}
	// Ensure readLoop sees stop condition even if Close returns fast.
	c := s.Conn
	s.Conn = nil
	return c.Close()
}

func (s *Server) readLoop() {
	buf := make([]byte, s.readBufSize)
	for {
		// Basic stop condition: if Conn is nil, exit.
		if s.Conn == nil {
			return
		}
		_ = s.Conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, addr, err := s.Conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			continue
		}

		rawBytes := append([]byte(nil), buf[:n]...)
		if s.OnDatagram != nil {
			s.OnDatagram(rawBytes, addr)
		}
		if s.OnEvent != nil {
			s.OnEvent(Event{Type: EventDatagramReceived, Raw: rawBytes, Addr: addr})
		}

		raw := string(rawBytes)
		msg, err := Parse(raw)
		if err != nil {
			// ignore parse errors for robustness
			if s.OnParseErr != nil {
				s.OnParseErr(rawBytes, addr, err)
			}
			if s.OnEvent != nil {
				s.OnEvent(Event{Type: EventParseError, Raw: rawBytes, Addr: addr, Err: err})
			}
			continue
		}

		if msg == nil {
			continue
		}
		if !msg.IsRequest {
			if s.OnSIPResponse != nil {
				s.OnSIPResponse(msg, addr)
			}
			if s.OnEvent != nil {
				s.OnEvent(Event{Type: EventResponseReceived, Addr: addr, Raw: rawBytes, Response: msg})
			}
			continue
		}

		if s.OnRequest != nil {
			s.OnRequest(msg, addr)
		}
		if s.OnEvent != nil {
			s.OnEvent(Event{Type: EventRequestReceived, Addr: addr, Raw: rawBytes, Request: msg})
		}

		method := strings.ToUpper(msg.Method)
		h := s.Handlers[method]
		if h == nil {
			h = s.NoRouteHandler
		}
		if h == nil {
			continue
		}

		resp := h(msg, addr)
		if resp == nil {
			continue
		}

		if s.OnResponse != nil {
			s.OnResponse(msg, resp, addr)
		}
		if s.OnEvent != nil {
			s.OnEvent(Event{Type: EventResponseSent, Addr: addr, Request: msg, Response: resp})
		}
		_ = s.Send(resp, addr)
	}
}

func (s *Server) Send(msg *Message, addr *net.UDPAddr) error {
	if s == nil || s.Conn == nil {
		return fmt.Errorf("sip: server not started")
	}
	if msg == nil {
		return fmt.Errorf("sip: nil message")
	}

	raw := msg.String()
	_, err := s.Conn.WriteToUDP([]byte(raw), addr)
	return err
}

