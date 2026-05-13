package stack

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// HandlerFunc handles an incoming SIP request and returns a response (or nil to send nothing).
type HandlerFunc func(msg *Message, addr *net.UDPAddr) *Message

// EventType identifies Endpoint telemetry events.
type EventType int

const (
	EventDatagramReceived EventType = iota
	EventParseError
	EventRequestReceived
	EventResponseReceived
	EventResponseSent
)

// Event is a lightweight observation from the read loop.
type Event struct {
	Type     EventType
	Addr     *net.UDPAddr
	Raw      []byte
	Request  *Message
	Response *Message
	Err      error
}

// EndpointConfig configures a UDP signaling Endpoint.
type EndpointConfig struct {
	// Host and Port are the listen address (UDP IPv4).
	Host string
	Port int

	ReadBufSize  int           // default 65535
	ReadDeadline time.Duration // per read; default 1s (poll + responsive shutdown)
	// OnReadError is called once for non-timeout read errors before Serve returns.
	OnReadError func(err error)
	OnDatagram  func(raw []byte, addr *net.UDPAddr)
	OnParseErr  func(raw []byte, addr *net.UDPAddr, err error)
	OnRequest   func(req *Message, addr *net.UDPAddr)
	OnResponse  func(req *Message, resp *Message, addr *net.UDPAddr)
	// OnResponseSent is invoked after a response has been successfully sent to addr (UAS final + server tx).
	OnResponseSent func(req *Message, resp *Message, addr *net.UDPAddr)
	OnSIPResponse  func(resp *Message, addr *net.UDPAddr)
	OnEvent        func(e Event)
	NoRouteHandler HandlerFunc
}

// Endpoint listens for SIP over UDP, parses datagrams, and dispatches requests.
// It provides context-aware Serve and
// non-timeout read errors exiting the loop (after optional OnReadError).
type Endpoint struct {
	cfg EndpointConfig

	mu       sync.Mutex
	handlers map[string]HandlerFunc
	tr       *UDPTransport
}

// NewEndpoint constructs an endpoint. Call Open then Serve.
func NewEndpoint(cfg EndpointConfig) *Endpoint {
	if cfg.ReadBufSize <= 0 {
		cfg.ReadBufSize = 65535
	}
	if cfg.ReadDeadline <= 0 {
		cfg.ReadDeadline = time.Second
	}
	return &Endpoint{
		cfg:      cfg,
		handlers: make(map[string]HandlerFunc),
	}
}

// RegisterHandler registers a SIP method handler (method is case-insensitive).
func (e *Endpoint) RegisterHandler(method string, h HandlerFunc) {
	if e == nil {
		return
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.handlers == nil {
		e.handlers = make(map[string]HandlerFunc)
	}
	e.handlers[method] = h
}

// Open binds the UDP listen socket.
func (e *Endpoint) Open() error {
	if e == nil {
		return fmt.Errorf("sip1/stack: nil endpoint")
	}
	addr := &net.UDPAddr{
		IP:   net.ParseIP(strings.TrimSpace(e.cfg.Host)),
		Port: e.cfg.Port,
	}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("sip1/stack: listen udp: %w", err)
	}
	e.mu.Lock()
	e.tr = NewUDPTransport(conn)
	e.mu.Unlock()
	return nil
}

// Transport returns the datagram transport after Open, or nil.
func (e *Endpoint) Transport() DatagramTransport {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.tr
}

// ListenAddr returns the bound UDP address after Open, or nil.
func (e *Endpoint) ListenAddr() net.Addr {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.tr == nil || e.tr.conn == nil {
		return nil
	}
	return e.tr.conn.LocalAddr()
}

// Send writes a SIP message to addr using the bound UDP socket.
func (e *Endpoint) Send(msg *Message, addr *net.UDPAddr) error {
	if e == nil {
		return fmt.Errorf("sip1/stack: nil endpoint")
	}
	e.mu.Lock()
	tr := e.tr
	e.mu.Unlock()
	if tr == nil || tr.conn == nil {
		return fmt.Errorf("sip1/stack: endpoint not open")
	}
	if msg == nil {
		return fmt.Errorf("sip1/stack: nil message")
	}
	raw := msg.String()
	_, err := tr.conn.WriteToUDP([]byte(raw), addr)
	return err
}

// Close closes the listen socket and unblocks Serve.
func (e *Endpoint) Close() error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	tr := e.tr
	e.tr = nil
	e.mu.Unlock()
	if tr == nil {
		return nil
	}
	return tr.Close()
}

// AppendOnResponseSent chains fn after the existing OnResponseSent callback.
// Call before Serve starts, or only from the setup goroutine (not concurrently with the read loop).
func (e *Endpoint) AppendOnResponseSent(fn func(*Message, *Message, *net.UDPAddr)) {
	if e == nil || fn == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	prev := e.cfg.OnResponseSent
	e.cfg.OnResponseSent = func(req, resp *Message, addr *net.UDPAddr) {
		if prev != nil {
			prev(req, resp, addr)
		}
		fn(req, resp, addr)
	}
}

// Serve runs the read/dispatch loop until ctx is cancelled, Close is called, or a non-timeout
// read error occurs (then returns a wrapped error after optional OnReadError).
// It returns ctx.Err() when the context was cancelled before a read completes.
func (e *Endpoint) Serve(ctx context.Context) error {
	if e == nil {
		return fmt.Errorf("sip1/stack: nil endpoint")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	e.mu.Lock()
	tr := e.tr
	e.mu.Unlock()
	if tr == nil || tr.conn == nil {
		return fmt.Errorf("sip1/stack: endpoint not open")
	}

	buf := make([]byte, e.cfg.ReadBufSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_ = tr.conn.SetReadDeadline(time.Now().Add(e.cfg.ReadDeadline))
		n, addr, err := tr.conn.ReadFromUDP(buf)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if e.cfg.OnReadError != nil {
				e.cfg.OnReadError(err)
			}
			return fmt.Errorf("sip1/stack: read udp: %w", err)
		}

		rawBytes := append([]byte(nil), buf[:n]...)
		if e.cfg.OnDatagram != nil {
			e.cfg.OnDatagram(rawBytes, addr)
		}
		if IsSignalingNoiseDatagram(rawBytes) {
			continue
		}
		if e.cfg.OnEvent != nil {
			e.cfg.OnEvent(Event{Type: EventDatagramReceived, Raw: rawBytes, Addr: addr})
		}

		raw := string(rawBytes)
		msg, err := Parse(raw)
		if err != nil {
			if e.cfg.OnParseErr != nil {
				e.cfg.OnParseErr(rawBytes, addr, err)
			}
			if e.cfg.OnEvent != nil {
				e.cfg.OnEvent(Event{Type: EventParseError, Raw: rawBytes, Addr: addr, Err: err})
			}
			continue
		}
		if msg == nil {
			continue
		}

		if !msg.IsRequest {
			if e.cfg.OnSIPResponse != nil {
				e.cfg.OnSIPResponse(msg, addr)
			}
			if e.cfg.OnEvent != nil {
				e.cfg.OnEvent(Event{Type: EventResponseReceived, Addr: addr, Raw: rawBytes, Response: msg})
			}
			continue
		}

		if e.cfg.OnRequest != nil {
			e.cfg.OnRequest(msg, addr)
		}
		if e.cfg.OnEvent != nil {
			e.cfg.OnEvent(Event{Type: EventRequestReceived, Addr: addr, Raw: rawBytes, Request: msg})
		}

		method := strings.ToUpper(msg.Method)
		e.mu.Lock()
		h := e.handlers[method]
		if h == nil {
			h = e.cfg.NoRouteHandler
		}
		e.mu.Unlock()

		if h == nil {
			continue
		}

		resp := h(msg, addr)
		if resp == nil {
			continue
		}

		if e.cfg.OnResponse != nil {
			e.cfg.OnResponse(msg, resp, addr)
		}
		if e.cfg.OnEvent != nil {
			e.cfg.OnEvent(Event{Type: EventResponseSent, Addr: addr, Request: msg, Response: resp})
		}
		if err := e.Send(resp, addr); err != nil {
			return fmt.Errorf("sip1/stack: send response: %w", err)
		}
		var onSent func(*Message, *Message, *net.UDPAddr)
		e.mu.Lock()
		onSent = e.cfg.OnResponseSent
		e.mu.Unlock()
		if onSent != nil {
			onSent(msg, resp, addr)
		}
	}
}

// SetNoRouteHandler sets the fallback handler for unknown methods (same as EndpointConfig.NoRouteHandler).
func (e *Endpoint) SetNoRouteHandler(h HandlerFunc) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg.NoRouteHandler = h
}

// DispatchRequest runs the registered method handler (or NoRouteHandler) without sending on UDP.
// Used by alternate transports (e.g. TCP/TLS) that write responses themselves.
func (e *Endpoint) DispatchRequest(req *Message, addr *net.UDPAddr) *Message {
	if e == nil || req == nil {
		return nil
	}
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	e.mu.Lock()
	h := e.handlers[method]
	if h == nil {
		h = e.cfg.NoRouteHandler
	}
	e.mu.Unlock()
	if h == nil {
		return nil
	}
	return h(req, addr)
}

// InvokeOnSIPResponse calls the configured OnSIPResponse hook (e.g. for responses received on TCP).
func (e *Endpoint) InvokeOnSIPResponse(resp *Message, addr *net.UDPAddr) {
	if e == nil {
		return
	}
	e.mu.Lock()
	fn := e.cfg.OnSIPResponse
	e.mu.Unlock()
	if fn != nil {
		fn(resp, addr)
	}
}
