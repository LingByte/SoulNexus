// Copyright (c) 2026 LinByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package outbound provides a UAC-side Manager that places, tracks, and
// terminates SIP calls. It is the multi-call counterpart of the demo
// runOutbound flow in cmd/voiceserver: instead of a single one-shot
// dial-and-block, the Manager runs N concurrent calls, exposes Hangup
// by Call-ID, and surfaces lifecycle events through registered hooks.
//
// What this package does NOT do (yet):
//   - Re-INVITE / SDP renegotiation (call hold, codec change mid-call)
//   - Outbound REFER for attended transfer of an existing call
//   - SRTP / TLS signalling
//   - Authentication challenges (401/407 + nonce) — assumes a trusted
//     trunk or a session-border-controller absorbs auth upstream.
//
// These are tracked as TODOs in the package doc; the basic Manager is
// enough for AI outbound campaigns (dial customer → speak → bye) and
// for "transfer the inbound caller by hanging up after the carrier
// honours our REFER" flows. Full attended transfer (place a second
// leg, bridge, hand the customer off, hang up the original leg) lives
// in pkg/sip/bridge layered on top of this Manager.
package outbound

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/sdp"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/session"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/transaction"
	siprtp "github.com/LingByte/SoulNexus/pkg/voiceserver/sip/rtp"
)

// Errors surfaced by the Manager API.
var (
	ErrManagerClosed = errors.New("sip/outbound: manager closed")
	ErrCallNotFound  = errors.New("sip/outbound: call not found")
	ErrCallRejected  = errors.New("sip/outbound: call rejected")
)

// SendFunc is the same shape transaction.Manager wants — sending one
// message to one UDP address. Allocated lazily from the Endpoint.
type SendFunc = transaction.SendFunc

// DialRequest is what callers pass to Manager.Dial.
type DialRequest struct {
	// TargetURI is the full sip: URI ("sip:bob@pbx.example.com:5060").
	// The Manager parses host:port out of it and uses that as the
	// transport-layer destination.
	TargetURI string

	// FromURI is the SIP "From" identity. Empty → "sip:voiceserver@<LocalIP>".
	FromURI string

	// CallID, if empty, is auto-generated. Provide it explicitly when
	// the dialog plane wants to correlate this leg with an inbound
	// call (e.g. attended transfer second leg).
	CallID string

	// OfferCodecs, if empty, defaults to PCMA/PCMU. Provide a wider
	// set when the upstream supports Opus / G.722.
	OfferCodecs []sdp.Codec

	// CorrelationID is opaque metadata the Manager echoes on event
	// hooks. Useful for "this dial belongs to inbound Call-ID X".
	CorrelationID string
}

// DialResult is returned on a successful 2xx final.
type DialResult struct {
	// CallID is the SIP Call-ID for this dialog (auto-generated if
	// the caller didn't supply one).
	CallID string

	// Leg is the live media leg. Already started; caller can wire
	// recorder filters, ASR taps, etc. via Leg's APIs (or via the
	// MediaLeg returned at Dial time before sending audio).
	Leg *session.MediaLeg

	// RemoteRTP is the peer's RTP endpoint as advertised in 200 OK
	// SDP. Stored so observability rows can surface it.
	RemoteRTP string
}

// EventType enumerates lifecycle events on a tracked outbound call.
type EventType string

const (
	EventDialing      EventType = "dialing"   // INVITE sent
	EventProvisional  EventType = "provisional" // 1xx received
	EventAnswered     EventType = "answered"  // 2xx received, ACK sent
	EventEnded        EventType = "ended"     // BYE sent or peer BYE absorbed
	EventFailed       EventType = "failed"    // non-2xx final, transport failure, ctx cancel
)

// Event surfaces a state change on a tracked call.
type Event struct {
	Type          EventType
	CallID        string
	CorrelationID string
	StatusCode    int    // populated on EventProvisional / EventFailed
	Reason        string // optional human-readable
	At            time.Time
}

// Config wires the Manager to its UDP endpoint and transaction layer.
type Config struct {
	// Endpoint is the shared signalling endpoint. The Manager reuses
	// the SAME endpoint as the SIP server when one exists, so that
	// inbound and outbound both share Via/Contact host:port.
	Endpoint *stack.Endpoint

	// TxManager dispatches retransmits / response correlation.
	// Pass the same instance the endpoint's OnSIPResponse forwards to.
	TxManager *transaction.Manager

	// LocalIP is the public-reachable address the Manager advertises
	// in Via/Contact and SDP "c=" lines. Required.
	LocalIP string

	// LocalSigPort is the UDP port the Endpoint is bound to. The
	// Manager uses this in Via/Contact and the SDP origin line.
	// Required.
	LocalSigPort int

	// UserAgent is the SIP User-Agent header value. Empty → "VoiceServer-Outbound/1.0".
	UserAgent string

	// OnEvent fires for every lifecycle transition. Optional. Runs
	// on the goroutine that observed the transition; keep it cheap.
	OnEvent func(Event)
}

// Manager places and tracks outbound calls.
type Manager struct {
	cfg Config

	mu     sync.Mutex
	calls  map[string]*activeCall // keyed by Call-ID
	closed atomic.Bool
}

// activeCall holds the per-dialog state we need to send a follow-up
// BYE — request URI, From-tag, To header (with peer tag), CSeq.
type activeCall struct {
	callID        string
	correlationID string
	fromURI       string
	fromTag       string
	toRaw         string // To header from 200 OK (carries peer tag)
	requestURI    string // ackURI: Contact in 200 OK, or original target
	remoteAddr    *net.UDPAddr
	leg           *session.MediaLeg
	cseq          int32 // atomic; bumped on each in-dialog request
	rtpSess       *siprtp.Session
}

// New validates cfg and returns a Manager.
func New(cfg Config) (*Manager, error) {
	if cfg.Endpoint == nil {
		return nil, errors.New("sip/outbound: nil Endpoint")
	}
	if cfg.TxManager == nil {
		return nil, errors.New("sip/outbound: nil TxManager")
	}
	if strings.TrimSpace(cfg.LocalIP) == "" {
		return nil, errors.New("sip/outbound: empty LocalIP")
	}
	if cfg.LocalSigPort <= 0 {
		return nil, errors.New("sip/outbound: bad LocalSigPort")
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = "VoiceServer-Outbound/1.0"
	}
	return &Manager{
		cfg:   cfg,
		calls: make(map[string]*activeCall),
	}, nil
}

// Dial places an outbound call. Blocks until a final response is
// received (or ctx is done). On 2xx success the returned DialResult
// references a live MediaLeg ready to carry audio; the call stays
// tracked until Hangup is called or the peer BYEs.
func (m *Manager) Dial(ctx context.Context, req DialRequest) (DialResult, error) {
	if m == nil || m.closed.Load() {
		return DialResult{}, ErrManagerClosed
	}
	target, err := parseSIPTarget(req.TargetURI)
	if err != nil {
		return DialResult{}, fmt.Errorf("parse target: %w", err)
	}

	// Allocate RTP session + offer SDP up front. We do this BEFORE
	// the INVITE so we can advertise our own RTP address; if the
	// allocation fails there's nothing to roll back.
	rtpSess, err := siprtp.NewSession(0)
	if err != nil {
		return DialResult{}, fmt.Errorf("rtp alloc: %w", err)
	}
	codecs := req.OfferCodecs
	if len(codecs) == 0 {
		codecs = []sdp.Codec{
			{PayloadType: 8, Name: "PCMA", ClockRate: 8000, Channels: 1},
			{PayloadType: 0, Name: "PCMU", ClockRate: 8000, Channels: 1},
		}
	}
	offerBody := sdp.Generate(m.cfg.LocalIP, rtpSess.LocalAddr.Port, codecs)

	callID := strings.TrimSpace(req.CallID)
	if callID == "" {
		callID = newCallID()
	}
	fromURI := strings.TrimSpace(req.FromURI)
	if fromURI == "" {
		fromURI = fmt.Sprintf("sip:voiceserver@%s", m.cfg.LocalIP)
	}
	fromTag := newTag()
	branch := "z9hG4bK-" + newTag()

	invite := buildInvite(buildInviteArgs{
		RequestURI:   target.uri,
		Branch:       branch,
		LocalIP:      m.cfg.LocalIP,
		LocalSigPort: m.cfg.LocalSigPort,
		UserAgent:    m.cfg.UserAgent,
		FromURI:      fromURI,
		FromTag:      fromTag,
		ToURI:        target.uri,
		CallID:       callID,
		CSeq:         1,
		Body:         offerBody,
	})

	m.emit(Event{Type: EventDialing, CallID: callID, CorrelationID: req.CorrelationID, At: time.Now()})

	send := func(msg *stack.Message, addr *net.UDPAddr) error {
		return m.cfg.Endpoint.Send(msg, addr)
	}
	result, err := m.cfg.TxManager.RunInviteClient(ctx, invite, target.addr, send, func(prov *stack.Message) {
		m.emit(Event{
			Type:          EventProvisional,
			CallID:        callID,
			CorrelationID: req.CorrelationID,
			StatusCode:    prov.StatusCode,
			Reason:        prov.StatusText,
			At:            time.Now(),
		})
	})
	if err != nil {
		rtpSess.Close()
		m.emit(Event{Type: EventFailed, CallID: callID, CorrelationID: req.CorrelationID, Reason: err.Error(), At: time.Now()})
		return DialResult{}, fmt.Errorf("invite: %w", err)
	}
	final := result.Final
	if final.StatusCode < 200 || final.StatusCode >= 300 {
		// Non-2xx absorbs ACK in the server transaction; we don't
		// send our own. Tear down RTP and surface the failure.
		rtpSess.Close()
		m.emit(Event{
			Type: EventFailed, CallID: callID, CorrelationID: req.CorrelationID,
			StatusCode: final.StatusCode, Reason: final.StatusText, At: time.Now(),
		})
		return DialResult{}, fmt.Errorf("%w: %d %s", ErrCallRejected, final.StatusCode, final.StatusText)
	}

	// 2xx: send ACK, parse remote SDP, build media leg.
	ackURI := transaction.AckRequestURIFor2xx(final, invite.RequestURI)
	ack, err := transaction.BuildAckForInvite(invite, final, ackURI)
	if err != nil {
		rtpSess.Close()
		return DialResult{}, fmt.Errorf("build ack: %w", err)
	}
	if err := m.cfg.Endpoint.Send(ack, result.Remote); err != nil {
		rtpSess.Close()
		return DialResult{}, fmt.Errorf("send ack: %w", err)
	}

	remoteSDP, err := sdp.Parse(final.Body)
	if err != nil {
		rtpSess.Close()
		return DialResult{}, fmt.Errorf("parse 200 sdp: %w", err)
	}
	if err := session.ApplyRemoteSDP(rtpSess, remoteSDP); err != nil {
		rtpSess.Close()
		return DialResult{}, fmt.Errorf("apply remote sdp: %w", err)
	}

	leg, err := session.NewMediaLeg(ctx, callID, rtpSess, remoteSDP.Codecs, session.MediaLegConfig{})
	if err != nil {
		rtpSess.Close()
		return DialResult{}, fmt.Errorf("media leg: %w", err)
	}
	leg.Start()

	ac := &activeCall{
		callID:        callID,
		correlationID: req.CorrelationID,
		fromURI:       fromURI,
		fromTag:       fromTag,
		toRaw:         final.GetHeader("To"),
		requestURI:    ackURI,
		remoteAddr:    result.Remote,
		leg:           leg,
		rtpSess:       rtpSess,
	}
	atomic.StoreInt32(&ac.cseq, 1) // matches the INVITE's CSeq

	m.mu.Lock()
	m.calls[callID] = ac
	m.mu.Unlock()

	remoteRTP := ""
	if remoteSDP != nil && remoteSDP.IP != "" && remoteSDP.Port > 0 {
		remoteRTP = net.JoinHostPort(remoteSDP.IP, strconv.Itoa(remoteSDP.Port))
	}
	m.emit(Event{Type: EventAnswered, CallID: callID, CorrelationID: req.CorrelationID, At: time.Now()})

	return DialResult{CallID: callID, Leg: leg, RemoteRTP: remoteRTP}, nil
}

// Hangup terminates a tracked outbound call by sending BYE. Returns
// ErrCallNotFound if the call is unknown (already torn down or never
// existed). Idempotent: a second Hangup on the same Call-ID returns
// ErrCallNotFound.
func (m *Manager) Hangup(ctx context.Context, callID, reason string) error {
	if m == nil || m.closed.Load() {
		return ErrManagerClosed
	}
	m.mu.Lock()
	ac, ok := m.calls[callID]
	if ok {
		delete(m.calls, callID)
	}
	m.mu.Unlock()
	if !ok {
		return ErrCallNotFound
	}

	// Tear down the media path first so we stop emitting RTP before
	// the peer ACKs the BYE.
	if ac.leg != nil {
		ac.leg.Stop()
	}

	cseq := atomic.AddInt32(&ac.cseq, 1)
	bye := buildBye(buildByeArgs{
		RequestURI:   ac.requestURI,
		Branch:       "z9hG4bK-" + newTag(),
		LocalIP:      m.cfg.LocalIP,
		LocalSigPort: m.cfg.LocalSigPort,
		UserAgent:    m.cfg.UserAgent,
		FromURI:      ac.fromURI,
		FromTag:      ac.fromTag,
		ToRaw:        ac.toRaw,
		CallID:       ac.callID,
		CSeq:         int(cseq),
	})

	send := func(msg *stack.Message, addr *net.UDPAddr) error {
		return m.cfg.Endpoint.Send(msg, addr)
	}
	byeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	res, err := m.cfg.TxManager.RunNonInviteClient(byeCtx, bye, ac.remoteAddr, send)
	if ac.rtpSess != nil {
		ac.rtpSess.Close()
	}
	at := time.Now()
	if err != nil {
		m.emit(Event{Type: EventEnded, CallID: ac.callID, CorrelationID: ac.correlationID, Reason: "bye-error: " + err.Error(), At: at})
		return fmt.Errorf("bye: %w", err)
	}
	m.emit(Event{
		Type:          EventEnded,
		CallID:        ac.callID,
		CorrelationID: ac.correlationID,
		StatusCode:    res.Final.StatusCode,
		Reason:        reason,
		At:            at,
	})
	return nil
}

// List returns the Call-IDs of all currently tracked outbound calls.
// Snapshot — the returned slice may go stale by the time the caller
// inspects it.
func (m *Manager) List() []string {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.calls))
	for k := range m.calls {
		out = append(out, k)
	}
	return out
}

// Get returns the live MediaLeg + remote address for a tracked call.
// (nil, false) if unknown. Useful for the dialog plane to attach an
// ASR/TTS pipeline post-Dial.
func (m *Manager) Get(callID string) (*session.MediaLeg, *net.UDPAddr, bool) {
	if m == nil {
		return nil, nil, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	ac, ok := m.calls[callID]
	if !ok {
		return nil, nil, false
	}
	return ac.leg, ac.remoteAddr, true
}

// Close gracefully tears down every tracked call. Best-effort: BYE
// timeouts are absorbed and logged via OnEvent.
func (m *Manager) Close(ctx context.Context) {
	if m == nil || m.closed.Swap(true) {
		return
	}
	m.mu.Lock()
	ids := make([]string, 0, len(m.calls))
	for k := range m.calls {
		ids = append(ids, k)
	}
	m.mu.Unlock()
	// Hangup outside the mutex (each Hangup re-locks briefly).
	for _, id := range ids {
		_ = m.Hangup(ctx, id, "manager-close")
	}
}

func (m *Manager) emit(ev Event) {
	if m == nil || m.cfg.OnEvent == nil {
		return
	}
	m.cfg.OnEvent(ev)
}

// ---------- helpers (kept private; cmd/voiceserver has its own copies for the demo flow) ----

type buildInviteArgs struct {
	RequestURI   string
	Branch       string
	LocalIP      string
	LocalSigPort int
	UserAgent    string
	FromURI      string
	FromTag      string
	ToURI        string
	CallID       string
	CSeq         int
	Body         string
}

func buildInvite(a buildInviteArgs) *stack.Message {
	m := &stack.Message{
		IsRequest:    true,
		Method:       stack.MethodInvite,
		RequestURI:   a.RequestURI,
		Version:      "SIP/2.0",
		Headers:      map[string]string{},
		HeadersMulti: map[string][]string{},
	}
	m.SetHeader("Via", fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=%s;rport", a.LocalIP, a.LocalSigPort, a.Branch))
	m.SetHeader("Max-Forwards", "70")
	m.SetHeader("From", fmt.Sprintf("<%s>;tag=%s", a.FromURI, a.FromTag))
	m.SetHeader("To", fmt.Sprintf("<%s>", a.ToURI))
	m.SetHeader("Call-ID", a.CallID)
	m.SetHeader("CSeq", fmt.Sprintf("%d INVITE", a.CSeq))
	m.SetHeader("Contact", fmt.Sprintf("<sip:voiceserver@%s:%d>", a.LocalIP, a.LocalSigPort))
	m.SetHeader("Allow", "INVITE, ACK, BYE, CANCEL, OPTIONS, INFO, PRACK, UPDATE")
	m.SetHeader("User-Agent", a.UserAgent)
	if a.Body != "" {
		m.SetHeader("Content-Type", "application/sdp")
		m.Body = a.Body
	}
	m.SetHeader("Content-Length", strconv.Itoa(stack.BodyBytesLen(a.Body)))
	return m
}

type buildByeArgs struct {
	RequestURI   string
	Branch       string
	LocalIP      string
	LocalSigPort int
	UserAgent    string
	FromURI      string
	FromTag      string
	ToRaw        string
	CallID       string
	CSeq         int
}

func buildBye(a buildByeArgs) *stack.Message {
	m := &stack.Message{
		IsRequest:    true,
		Method:       stack.MethodBye,
		RequestURI:   a.RequestURI,
		Version:      "SIP/2.0",
		Headers:      map[string]string{},
		HeadersMulti: map[string][]string{},
	}
	m.SetHeader("Via", fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=%s;rport", a.LocalIP, a.LocalSigPort, a.Branch))
	m.SetHeader("Max-Forwards", "70")
	m.SetHeader("From", fmt.Sprintf("<%s>;tag=%s", a.FromURI, a.FromTag))
	m.SetHeader("To", a.ToRaw)
	m.SetHeader("Call-ID", a.CallID)
	m.SetHeader("CSeq", fmt.Sprintf("%d BYE", a.CSeq))
	m.SetHeader("User-Agent", a.UserAgent)
	m.SetHeader("Content-Length", "0")
	return m
}

type sipTarget struct {
	uri  string
	addr *net.UDPAddr
}

// parseSIPTarget accepts "sip:user@host[:port]" with optional <> braces
// and ;params after the host. Default port is 5060.
func parseSIPTarget(raw string) (sipTarget, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return sipTarget{}, errors.New("empty target")
	}
	s = strings.TrimPrefix(s, "<")
	s = strings.TrimSuffix(s, ">")
	if !strings.HasPrefix(strings.ToLower(s), "sip:") {
		return sipTarget{}, errors.New("target must start with sip:")
	}
	rest := s[4:]
	at := strings.Index(rest, "@")
	if at < 0 {
		return sipTarget{}, errors.New("target missing user@host")
	}
	hostport := rest[at+1:]
	if semi := strings.Index(hostport, ";"); semi >= 0 {
		hostport = hostport[:semi]
	}
	host, portStr, err := net.SplitHostPort(hostport)
	port := 5060
	if err != nil {
		host = hostport
	} else if portStr != "" {
		p, perr := strconv.Atoi(portStr)
		if perr != nil || p <= 0 || p > 65535 {
			return sipTarget{}, fmt.Errorf("bad port: %q", portStr)
		}
		port = p
	}
	ip := net.ParseIP(host)
	if ip == nil {
		// Caller-supplied hostname — resolve.
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return sipTarget{}, fmt.Errorf("resolve %q: %w", host, err)
		}
		// Prefer the first IPv4; pion / SIP UDP doesn't speak v6 yet
		// in our stack, so v6-only targets would need a future PR.
		for _, candidate := range ips {
			if v4 := candidate.To4(); v4 != nil {
				ip = v4
				break
			}
		}
		if ip == nil {
			ip = ips[0]
		}
	}
	canonical := fmt.Sprintf("sip:%s@%s:%d", rest[:at], host, port)
	return sipTarget{uri: canonical, addr: &net.UDPAddr{IP: ip, Port: port}}, nil
}

func newTag() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func newCallID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:]) + "@voiceserver"
}
