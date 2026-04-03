package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip/conversation"
	"github.com/LingByte/SoulNexus/pkg/sip/protocol"
	"github.com/LingByte/SoulNexus/pkg/sip/rtp"
	sipSession "github.com/LingByte/SoulNexus/pkg/sip/session"
	"go.uber.org/zap"
)

// SIPRegisterStore persists REGISTER bindings for INVITE proxy and outbound dial lookup.
// Implementations must be safe for concurrent use (e.g. GORM).
type SIPRegisterStore interface {
	// SaveRegister stores the resolved Contact signaling target (UDP), same as INVITE proxy destination.
	SaveRegister(ctx context.Context, user, domain, contactURI string, sig *net.UDPAddr, expiresAt time.Time, userAgent string) error

	DeleteRegister(ctx context.Context, user, domain string) error
	// LookupRegister returns the UDP signaling target for a registered AOR (Contact / Via path).
	LookupRegister(ctx context.Context, user, domain string) (*net.UDPAddr, bool, error)
}

// SIPServer is a minimal SIP over UDP server skeleton.
//
// It supports:
// - INVITE: parses SDP, creates an RTP session, and replies 200 OK with SDP.
// - BYE: closes the associated RTP session (if any).
//
// On ACK, optional ASR→LLM→TTS is attached when ASR/LLM/TTS env vars are set (see pkg/sip/conversation).
type SIPServer struct {
	proto *protocol.Server

	localIP    string
	listenHost string
	listenPort int

	mu        sync.Mutex
	callStore map[string]*sipSession.CallSession // Call-ID -> call session

	regStoreMu sync.RWMutex
	regStore   SIPRegisterStore // optional: persisted REGISTER (sip_users), set via SetRegisterStore
}

type Config struct {
	Host string
	Port int

	// localIP is used in SDP response's c=IN IP4 <localIP>.
	// If empty, server will try to use 127.0.0.1.
	LocalIP string

	// OnSIPResponse is optional: SIP responses on the listen socket (UAC / outbound legs).
	// Typically set to outbound.Manager.HandleSIPResponse.
	OnSIPResponse func(resp *protocol.Message, addr *net.UDPAddr)
}

func New(cfg Config) *SIPServer {
	s := &SIPServer{
		localIP:    strings.TrimSpace(cfg.LocalIP),
		listenHost: strings.TrimSpace(cfg.Host),
		listenPort: cfg.Port,
		callStore:  make(map[string]*sipSession.CallSession),
	}
	if s.localIP == "" {
		s.localIP = "127.0.0.1"
	}

	s.proto = protocol.NewServer(cfg.Host, cfg.Port)
	if cfg.OnSIPResponse != nil {
		s.proto.OnSIPResponse = cfg.OnSIPResponse
	}
	// Default protocol-level logs for visibility during development.
	s.proto.OnEvent = func(e protocol.Event) {
		switch e.Type {
		case protocol.EventDatagramReceived:
			logger.Debug("sip datagram received",
				zap.String("remote", addrString(e.Addr)),
				zap.Int("bytes", len(e.Raw)),
			)
		case protocol.EventParseError:
			logger.Warn("sip parse error",
				zap.String("remote", addrString(e.Addr)),
				zap.Int("bytes", len(e.Raw)),
				zap.Error(e.Err),
			)
		case protocol.EventRequestReceived:
			req := e.Request
			logger.Info("sip request received",
				zap.String("remote", addrString(e.Addr)),
				zap.String("method", safe(req, func(m *protocol.Message) string { return m.Method })),
				zap.String("uri", safe(req, func(m *protocol.Message) string { return m.RequestURI })),
				zap.String("call_id", safe(req, func(m *protocol.Message) string { return m.GetHeader("Call-ID") })),
				zap.String("from", safe(req, func(m *protocol.Message) string { return m.GetHeader("From") })),
				zap.String("to", safe(req, func(m *protocol.Message) string { return m.GetHeader("To") })),
				zap.String("cseq", safe(req, func(m *protocol.Message) string { return m.GetHeader("CSeq") })),
			)
		case protocol.EventResponseSent:
			req := e.Request
			resp := e.Response
			logger.Info("sip response sent",
				zap.String("remote", addrString(e.Addr)),
				zap.String("method", safe(req, func(m *protocol.Message) string { return m.Method })),
				zap.String("call_id", safe(req, func(m *protocol.Message) string { return m.GetHeader("Call-ID") })),
				zap.Int("status", safeI(resp, func(m *protocol.Message) int { return m.StatusCode })),
				zap.String("reason", safe(resp, func(m *protocol.Message) string { return m.StatusText })),
			)
		case protocol.EventResponseReceived:
			if e.Response != nil {
				logger.Debug("sip response received",
					zap.String("remote", addrString(e.Addr)),
					zap.String("call_id", e.Response.GetHeader("Call-ID")),
					zap.Int("status", e.Response.StatusCode),
				)
			}
		}
	}
	s.proto.RegisterHandler(protocol.MethodInvite, s.handleInvite)
	s.proto.RegisterHandler(protocol.MethodAck, s.handleAck)
	s.proto.RegisterHandler(protocol.MethodBye, s.handleBye)
	s.proto.RegisterHandler(protocol.MethodOptions, s.handleOptions)
	s.proto.RegisterHandler(protocol.MethodRegister, s.handleRegister)
	s.proto.RegisterHandler(protocol.MethodInfo, s.handleInfo)
	s.proto.RegisterHandler(protocol.MethodCancel, s.handleCancel)
	s.proto.RegisterHandler(protocol.MethodPublish, s.handlePublish)
	s.proto.RegisterNoRoute(func(_ *protocol.Message, _ *net.UDPAddr) *protocol.Message {
		// No route: respond 404.
		return &protocol.Message{
			IsRequest:  false,
			Version:    "SIP/2.0",
			StatusCode: 404,
			StatusText: "Not Found",
		}
	})
	return s
}

// SetRegisterStore wires DB-backed REGISTER persistence and INVITE proxy lookup.
// Safe to call before Start; typically once at process init.
func (s *SIPServer) SetRegisterStore(st SIPRegisterStore) {
	if s == nil {
		return
	}
	s.regStoreMu.Lock()
	defer s.regStoreMu.Unlock()
	s.regStore = st
}

func (s *SIPServer) registerStore() SIPRegisterStore {
	if s == nil {
		return nil
	}
	s.regStoreMu.RLock()
	defer s.regStoreMu.RUnlock()
	return s.regStore
}

func addrString(a *net.UDPAddr) string {
	if a == nil {
		return ""
	}
	return a.String()
}

func safe(m *protocol.Message, f func(*protocol.Message) string) string {
	if m == nil || f == nil {
		return ""
	}
	return f(m)
}

func safeI(m *protocol.Message, f func(*protocol.Message) int) int {
	if m == nil || f == nil {
		return 0
	}
	return f(m)
}

func preview(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || s == "" {
		return ""
	}
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

func ensureToTag(to string) string {
	to = strings.TrimSpace(to)
	if to == "" {
		return to
	}
	if strings.Contains(strings.ToLower(to), "tag=") {
		return to
	}
	return to + ";tag=" + newTag()
}

func newTag() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "srvtag"
	}
	return hex.EncodeToString(b[:])
}

func (s *SIPServer) Start() error {
	return s.proto.Start()
}

func (s *SIPServer) Stop() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	for callID, cs := range s.callStore {
		if cs != nil {
			cs.Stop()
		}
		delete(s.callStore, callID)
	}
	s.mu.Unlock()

	if s.proto != nil {
		return s.proto.Stop()
	}
	return nil
}

// StartInviteHandler is exported for unit tests.
func (s *SIPServer) StartInviteHandler(msg *protocol.Message, addr *net.UDPAddr) *protocol.Message {
	return s.handleInvite(msg, addr)
}

// StartByeHandler is exported for unit tests.
func (s *SIPServer) StartByeHandler(msg *protocol.Message, addr *net.UDPAddr) *protocol.Message {
	return s.handleBye(msg, addr)
}

func (s *SIPServer) StartAckHandler(msg *protocol.Message, addr *net.UDPAddr) *protocol.Message {
	return s.handleAck(msg, addr)
}

func (s *SIPServer) handleInvite(msg *protocol.Message, addr *net.UDPAddr) *protocol.Message {
	if msg == nil || !msg.IsRequest || strings.ToUpper(msg.Method) != "INVITE" {
		return nil
	}

	callID := msg.GetHeader("Call-ID")
	if callID == "" {
		// SIP requires Call-ID; if absent respond 400.
		return s.makeResponse(msg, 400, "Bad Request", "", "")
	}

	// Provisional response: 100 Trying (helps many clients' state machines).
	if s.proto != nil && addr != nil {
		trying := s.makeResponse(msg, 100, "Trying", "", "")
		_ = s.proto.Send(trying, addr)
	}

	// Parse remote RTP endpoint from SDP.
	sdp, err := protocol.ParseSDP(msg.Body)
	if err != nil {
		logger.Warn("sip invite rejected (sdp not acceptable)",
			zap.String("call_id", callID),
			zap.String("content_type", msg.GetHeader("Content-Type")),
			zap.Int("content_length", len(msg.Body)),
			zap.Error(err),
			zap.String("sdp_preview", preview(msg.Body, 800)),
		)
		return s.makeResponse(msg, 488, "Not Acceptable Here", "", "")
	}

	// REGISTERed AOR: proxy INVITE to that UA (same host: different user in Request-URI).
	if u, h, ok := parseURIUserHost(msg.RequestURI); ok {
		if st := s.registerStore(); st != nil {
			dst, found, lerr := st.LookupRegister(context.Background(), u, h)
			if lerr != nil {
				logger.Warn("sip invite lookup register failed",
					zap.String("call_id", callID),
					zap.String("aor", registrationKey(u, h)),
					zap.Error(lerr),
				)
			} else if found && dst != nil {
				if err := s.proxyInviteToRegistrar(msg, dst); err != nil {
					logger.Warn("sip invite proxy to registered UA failed",
						zap.String("call_id", callID),
						zap.String("aor", registrationKey(u, h)),
						zap.Error(err),
					)
				} else {
					logger.Info("sip invite proxied to registered UA",
						zap.String("call_id", callID),
						zap.String("aor", registrationKey(u, h)),
						zap.String("dst", dst.String()),
					)
					return nil
				}
			}
		}
	}

	remoteIP := net.ParseIP(sdp.IP)
	if remoteIP == nil || sdp.Port <= 0 {
		return s.makeResponse(msg, 400, "Bad Request", "", "")
	}

	remoteAddr := &net.UDPAddr{IP: remoteIP, Port: sdp.Port}

	// Allocate RTP session on an ephemeral port to avoid fixed-port conflicts.
	rtpSess, err := rtp.NewSession(0)
	if err != nil {
		return s.makeResponse(msg, 500, "Internal Server Error", "", "")
	}
	rtpSess.SetRemoteAddr(remoteAddr)

	// Store session for BYE.
	s.mu.Lock()
	// If a session already exists for this call, stop it first (idempotency-ish).
	if old := s.callStore[callID]; old != nil {
		old.Stop()
	}
	cs, err := sipSession.NewCallSession(callID, rtpSess, sdp.Codecs)
	if err != nil {
		s.mu.Unlock()
		_ = rtpSess.Close()
		logger.Warn("sip invite rejected (no supported codec)",
			zap.String("call_id", callID),
			zap.Any("offered_codecs", sdp.Codecs),
			zap.Error(err),
		)
		return s.makeResponse(msg, 488, "Not Acceptable Here", "", "")
	}
	s.callStore[callID] = cs
	s.mu.Unlock()

	// IMPORTANT: Do not start media until ACK (call established).

	localPort := rtpSess.LocalAddr.Port

	// Reply with negotiated audio codec; add telephone-event from offer so UAs can send RFC 2833 DTMF.
	neg := cs.NegotiatedCodec()
	codecs := []protocol.SDPCodec{neg}
	if te, ok := protocol.PickTelephoneEventFromOffer(sdp.Codecs, neg.ClockRate); ok {
		codecs = append(codecs, te)
	}
	respSDP := protocol.GenerateSDPWithProto(s.localIP, localPort, sdp.Proto, codecs)

	// Use a single To-tag consistently across provisional/final responses.
	toWithTag := ensureToTag(msg.GetHeader("To"))

	// Provisional response: 180 Ringing (often expected by softphones).
	if s.proto != nil && addr != nil {
		ringing := s.makeResponse(msg, 180, "Ringing", "", toWithTag)
		ringing.SetHeader("To", toWithTag)
		ringing.SetHeader("Contact", fmt.Sprintf("<sip:server@%s:%d>", s.localIP, s.listenPort))
		ringing.SetHeader("Content-Length", "0")
		_ = s.proto.Send(ringing, addr)
	}

	respMsg := s.makeResponse(msg, 200, "OK", respSDP, toWithTag)
	respMsg.SetHeader("Content-Type", "application/sdp")
	respMsg.SetHeader("To", toWithTag)
	// For dialog establishment many clients expect a Contact header from UAS.
	// Use SDP local-ip as a reachable contact host.
	respMsg.SetHeader("Contact", fmt.Sprintf("<sip:server@%s:%d>", s.localIP, s.listenPort))
	respMsg.SetHeader("Allow", strings.Join([]string{
		protocol.MethodInvite,
		protocol.MethodAck,
		protocol.MethodBye,
		protocol.MethodRegister,
		protocol.MethodOptions,
		protocol.MethodCancel,
		protocol.MethodInfo,
	}, ", "))
	respMsg.SetHeader("Content-Length", strconv.Itoa(protocol.BodyBytesLen(respSDP)))

	logger.Info("sip invite negotiated",
		zap.String("call_id", callID),
		zap.String("remote_rtp", remoteAddr.String()),
		zap.String("answer_proto", sdp.Proto),
		zap.Any("offered_codecs", sdp.Codecs),
		zap.Any("negotiated_codec", neg),
		zap.String("answer_sdp_preview", preview(respSDP, 800)),
		zap.Int("answer_content_length", len(respSDP)),
		zap.Int("answer_raw_bytes", len(respMsg.String())),
		zap.String("answer_raw_preview", preview(respMsg.String(), 1200)),
	)
	return respMsg
}

func (s *SIPServer) handleAck(msg *protocol.Message, _ *net.UDPAddr) *protocol.Message {
	if msg == nil || !msg.IsRequest || strings.ToUpper(msg.Method) != protocol.MethodAck {
		return nil
	}
	callID := msg.GetHeader("Call-ID")
	if callID == "" {
		return nil
	}

	s.mu.Lock()
	cs := s.callStore[callID]
	s.mu.Unlock()
	if cs != nil {
		// After transfer, media is handed to TwoLegPCMBridge; do not attach ASR/TTS again
		// (e.g. late or duplicate ACK / re-INVITE ACK would hit a cancelled MediaSession).
		if conversation.ActiveTransferBridgeForCallID(callID) {
			return nil
		}
		var voiceLog *zap.Logger
		if logger.Lg != nil {
			voiceLog = logger.Lg.Named("sip-voice")
		}
		if err := conversation.AttachVoicePipeline(context.Background(), cs, voiceLog); err != nil {
			logger.Warn("sip voice pipeline attach failed",
				zap.String("call_id", callID),
				zap.Error(err),
			)
		}
		cs.StartOnACK()
	}
	// ACK has no SIP response.
	return nil
}

func (s *SIPServer) handleBye(msg *protocol.Message, _ *net.UDPAddr) *protocol.Message {
	if msg == nil || !msg.IsRequest || strings.ToUpper(msg.Method) != "BYE" {
		return nil
	}
	callID := msg.GetHeader("Call-ID")
	if callID == "" {
		return s.makeResponse(msg, 400, "Bad Request", "", "")
	}

	if conversation.HangupTransferBridgeIfAny(callID) {
		return s.makeResponse(msg, 200, "OK", "", "")
	}

	s.mu.Lock()
	cs := s.callStore[callID]
	delete(s.callStore, callID)
	s.mu.Unlock()

	if cs != nil {
		cs.Stop()
	}
	return s.makeResponse(msg, 200, "OK", "", "")
}

func (s *SIPServer) handleOptions(msg *protocol.Message, _ *net.UDPAddr) *protocol.Message {
	if msg == nil || !msg.IsRequest || strings.ToUpper(msg.Method) != protocol.MethodOptions {
		return nil
	}
	resp := s.makeResponse(msg, 200, "OK", "", "")
	// Minimal Allow capability.
	resp.SetHeader("Allow", strings.Join([]string{
		protocol.MethodInvite,
		protocol.MethodAck,
		protocol.MethodBye,
		protocol.MethodRegister,
		protocol.MethodOptions,
		protocol.MethodCancel,
		protocol.MethodInfo,
	}, ", "))
	resp.SetHeader("Content-Length", "0")
	return resp
}

func (s *SIPServer) handleRegister(msg *protocol.Message, addr *net.UDPAddr) *protocol.Message {
	if msg == nil || !msg.IsRequest || strings.ToUpper(msg.Method) != protocol.MethodRegister {
		return nil
	}
	s.upsertRegistration(msg, addr)
	// Minimal REGISTER: accept registration. Echo Contact if present and Expires if provided.
	resp := s.makeResponse(msg, 200, "OK", "", "")
	if c := msg.GetHeader("Contact"); c != "" {
		resp.SetHeader("Contact", c)
	}
	if exp := msg.GetHeader("Expires"); exp != "" {
		resp.SetHeader("Expires", exp)
	}
	resp.SetHeader("Content-Length", "0")
	return resp
}

func (s *SIPServer) handleInfo(msg *protocol.Message, _ *net.UDPAddr) *protocol.Message {
	if msg == nil || !msg.IsRequest || strings.ToUpper(msg.Method) != protocol.MethodInfo {
		return nil
	}
	callID := msg.GetHeader("Call-ID")
	var voiceLog *zap.Logger
	if logger.Lg != nil {
		voiceLog = logger.Lg.Named("sip-voice")
	}
	conversation.HandleSIPINFODTMF(context.Background(), callID, msg.GetHeader("Content-Type"), msg.Body, voiceLog)

	resp := s.makeResponse(msg, 200, "OK", "", "")
	resp.SetHeader("Content-Length", "0")
	return resp
}

func (s *SIPServer) handleCancel(msg *protocol.Message, _ *net.UDPAddr) *protocol.Message {
	if msg == nil || !msg.IsRequest || strings.ToUpper(msg.Method) != protocol.MethodCancel {
		return nil
	}
	// Minimal behavior: 200 OK for CANCEL. (Full SIP transaction mapping omitted.)
	resp := s.makeResponse(msg, 200, "OK", "", "")
	resp.SetHeader("Content-Length", "0")
	return resp
}

func (s *SIPServer) handlePublish(msg *protocol.Message, _ *net.UDPAddr) *protocol.Message {
	if msg == nil || !msg.IsRequest || strings.ToUpper(msg.Method) != "PUBLISH" {
		return nil
	}
	// Many softphones send PUBLISH for presence; accept to reduce noise.
	resp := s.makeResponse(msg, 200, "OK", "", "")
	resp.SetHeader("Content-Length", "0")
	return resp
}

// makeResponse builds a response by copying dialog/transaction headers and allowing
// method-specific behavior. If toOverride is provided, it replaces the To header.
func (s *SIPServer) makeResponse(req *protocol.Message, code int, text string, body string, toOverride string) *protocol.Message {
	resp := &protocol.Message{
		IsRequest:    false,
		Version:      "SIP/2.0",
		StatusCode:   code,
		StatusText:   text,
		Headers:      make(map[string]string),
		HeadersMulti: make(map[string][]string),
		Body:         body,
		Method:       "",
		RequestURI:   "",
	}

	if req != nil {
		// Via (multi-value) must be echoed back as-is.
		if vias := req.GetHeaders("Via"); len(vias) > 0 {
			resp.SetHeader("Via", vias[0])
			for i := 1; i < len(vias); i++ {
				resp.AddHeader("Via", vias[i])
			}
		}
		if v := req.GetHeader("From"); v != "" {
			resp.SetHeader("From", v)
		}
		if v := req.GetHeader("To"); v != "" {
			resp.SetHeader("To", v)
		}
		if v := req.GetHeader("Call-ID"); v != "" {
			resp.SetHeader("Call-ID", v)
		}
		if v := req.GetHeader("CSeq"); v != "" {
			resp.SetHeader("CSeq", v)
		}
	}
	if strings.TrimSpace(toOverride) != "" {
		resp.SetHeader("To", toOverride)
	}

	// Always emit explicit Content-Length (many clients expect it even for empty body).
	resp.SetHeader("Content-Length", strconv.Itoa(protocol.BodyBytesLen(body)))
	return resp
}

func (s *SIPServer) String() string {
	return fmt.Sprintf("SIPServer{localIP=%s}", s.localIP)
}

// SendSIP sends a raw SIP request or response on the server's UDP socket.
// Used by the outbound module to send INVITE/ACK/BYE for UAC legs.
func (s *SIPServer) SendSIP(msg *protocol.Message, addr *net.UDPAddr) error {
	if s == nil || s.proto == nil {
		return fmt.Errorf("sip: server not ready")
	}
	return s.proto.Send(msg, addr)
}

// ListenAddr returns the UDP listen address (host:port) for Contact/Via headers.
func (s *SIPServer) ListenAddr() (host string, port int) {
	if s == nil {
		return "", 0
	}
	return s.listenHost, s.listenPort
}

// GetCallSession returns the active CallSession for a Call-ID, or nil.
func (s *SIPServer) GetCallSession(callID string) *sipSession.CallSession {
	if s == nil || callID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.callStore[callID]
}

// RemoveCallSession deletes a Call-ID from the store without stopping media (used when RTP was torn down elsewhere).
func (s *SIPServer) RemoveCallSession(callID string) {
	if s == nil || callID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.callStore, callID)
}

// RegisterCallSession adds an established session (e.g. outbound UAC leg after ACK) so BYE and
// other in-dialog requests are handled the same as inbound calls.
func (s *SIPServer) RegisterCallSession(callID string, cs *sipSession.CallSession) {
	if s == nil || callID == "" || cs == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old := s.callStore[callID]; old != nil {
		old.Stop()
	}
	s.callStore[callID] = cs
}
