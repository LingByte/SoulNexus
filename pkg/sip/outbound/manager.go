package outbound

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip/protocol"
	"github.com/LingByte/SoulNexus/pkg/sip/rtp"
	sipSession "github.com/LingByte/SoulNexus/pkg/sip/session"
	"go.uber.org/zap"
)

// SignalingSender sends SIP on the shared UDP socket (typically *server.SIPServer).
type SignalingSender interface {
	SendSIP(msg *protocol.Message, addr *net.UDPAddr) error
}

// ManagerConfig configures outbound legs.
type ManagerConfig struct {
	// LocalIP is used in SDP c= line and Call-ID host part.
	LocalIP string
	// SIPHost / SIPPort identify this UA in Via/Contact (usually listen addr).
	SIPHost string
	SIPPort int
	// FromUser is the local SIP user part for From/Contact (CLI / 外显号码；默认 soulnexus).
	FromUser string
	// FromDisplayName is optional quoted display-name in From (empty → no display-name).
	FromDisplayName string

	// MediaAttach is invoked after ACK for MediaProfileAI (e.g. conversation.AttachVoicePipeline).
	MediaAttach MediaAttachFunc

	// OnRegisterSession optionally registers the CallSession with the SIP server (BYE handling).
	OnRegisterSession func(callID string, cs *sipSession.CallSession)

	// OnEstablished is optional analytics hook after media hooks succeed.
	OnEstablished func(EstablishedLeg)

	// OnTransferBridge runs after 200 OK + ACK for MediaProfileTransferBridge.
	// CorrelationID on the request is the inbound Call-ID; cs is the outbound UAC leg.
	OnTransferBridge func(correlationID string, cs *sipSession.CallSession, outboundCallID string)

	// OnScript runs when MediaProfileScript is established.
	OnScript func(ctx context.Context, leg EstablishedLeg, scriptID string)

	// OnEvent reports dial lifecycle transitions for queue workers and metrics.
	OnEvent func(DialEvent)
}

// Manager owns outbound SIP legs keyed by Call-ID.
type Manager struct {
	cfg  ManagerConfig
	send func(*protocol.Message, *net.UDPAddr) error

	mu   sync.Mutex
	legs map[string]*outLeg
}

// NewManager constructs a manager; call BindSender before Dial.
func NewManager(cfg ManagerConfig) *Manager {
	if cfg.FromUser == "" {
		cfg.FromUser = "soulnexus"
	}
	return &Manager{
		cfg:  cfg,
		legs: make(map[string]*outLeg),
	}
}

// BindSender wires the UDP signaling path (required for Dial).
func (m *Manager) BindSender(s SignalingSender) {
	if m == nil || s == nil {
		return
	}
	m.send = func(msg *protocol.Message, addr *net.UDPAddr) error {
		return s.SendSIP(msg, addr)
	}
}

// HandleSIPResponse must be set on protocol.Server.OnSIPResponse / server.Config.OnSIPResponse.
func (m *Manager) HandleSIPResponse(resp *protocol.Message, addr *net.UDPAddr) {
	if m == nil || resp == nil {
		return
	}
	callID := resp.GetHeader("Call-ID")
	if callID == "" {
		return
	}
	m.mu.Lock()
	leg := m.legs[callID]
	m.mu.Unlock()
	if leg == nil {
		return
	}
	leg.handleResponse(context.Background(), resp, addr)
}

// Dial starts an outbound INVITE. Returns Call-ID on success.
func (m *Manager) Dial(ctx context.Context, req DialRequest) (callID string, err error) {
	if m == nil {
		return "", fmt.Errorf("sip/outbound: nil manager")
	}
	if m.send == nil {
		return "", ErrNoSignalingSender
	}
	if strings.TrimSpace(req.Target.RequestURI) == "" {
		return "", fmt.Errorf("sip/outbound: empty target request URI")
	}
	if strings.TrimSpace(req.Target.SignalingAddr) == "" {
		return "", fmt.Errorf("sip/outbound: empty signaling address")
	}

	addr, err := net.ResolveUDPAddr("udp", req.Target.SignalingAddr)
	if err != nil {
		return "", fmt.Errorf("sip/outbound: resolve signaling: %w", err)
	}

	localPort := m.cfg.SIPPort
	if localPort <= 0 {
		localPort = 5060
	}
	lh := strings.TrimSpace(m.cfg.SIPHost)
	if lh != "" && addr.IP != nil && addr.Port == localPort {
		if lip := net.ParseIP(lh); lip != nil && lip.Equal(addr.IP) {
			logger.Debug("sip outbound: signaling to same IP:port as listener (hairpin); REGISTERed users are proxied, others answered locally.",
				zap.String("dst", addr.String()),
				zap.String("local_listener", fmt.Sprintf("%s:%d", lh, localPort)),
				zap.String("request_uri", strings.TrimSpace(req.Target.RequestURI)))
		}
	}

	localSDP := m.cfg.LocalIP
	if localSDP == "" {
		localSDP = "127.0.0.1"
	}

	rtpSess, err := rtp.NewSession(0)
	if err != nil {
		return "", fmt.Errorf("sip/outbound: rtp session: %w", err)
	}
	localPort = rtpSess.LocalAddr.Port
	codecs := defaultOfferCodecs()
	if req.MediaProfile == MediaProfileScript {
		// Script callbacks prioritize SIP endpoint compatibility over wideband quality.
		// Some UAs negotiate Opus but still produce garbled playout in this path.
		codecs = transferAgentBridgeOfferCodecs()
	} else if req.Scenario == ScenarioTransferAgent && req.MediaProfile == MediaProfileTransferBridge {
		codecs = transferAgentBridgeOfferCodecs()
	}
	sdpBody := protocol.GenerateSDPWithProto(localSDP, localPort, "RTP/AVP", codecs)

	callID = newCallID(localSDP)
	ip := m.cfg.SIPHost
	if ip == "" {
		ip = "127.0.0.1"
	}
	port := m.cfg.SIPPort
	if port <= 0 {
		port = 5060
	}

	fromUser := m.cfg.FromUser
	fromDisp := m.cfg.FromDisplayName
	if u := strings.TrimSpace(req.CallerUser); u != "" {
		fromUser = u
		fromDisp = strings.TrimSpace(req.CallerDisplayName)
	} else if u := strings.TrimSpace(req.Target.CallerUser); u != "" {
		fromUser = u
		fromDisp = strings.TrimSpace(req.Target.CallerDisplayName)
	}

	params := inviteParams{
		LocalIP:         localSDP,
		SIPHost:         ip,
		SIPPort:         port,
		RequestURI:      strings.TrimSpace(req.Target.RequestURI),
		CallID:          callID,
		FromTag:         randomHex(8),
		Branch:          randomHex(10),
		CSeq:            1,
		LocalRTPPort:    localPort,
		SDPBody:         sdpBody,
		FromUser:        fromUser,
		FromDisplayName: fromDisp,
	}

	invite := buildINVITE(params)
	leg := &outLeg{
		m:       m,
		params:  params,
		req:     req,
		rtpSess: rtpSess,
		dst:     addr,
	}

	m.mu.Lock()
	m.legs[callID] = leg
	m.mu.Unlock()

	if err := m.send(invite, addr); err != nil {
		m.mu.Lock()
		delete(m.legs, callID)
		m.mu.Unlock()
		_ = rtpSess.Close()
		return "", fmt.Errorf("sip/outbound: send INVITE: %w", err)
	}

	logger.Info("sip outbound INVITE sent",
		zap.String("call_id", callID),
		zap.String("request_uri", strings.TrimSpace(req.Target.RequestURI)),
		zap.String("scenario", string(req.Scenario)),
		zap.String("media_profile", string(req.MediaProfile)),
		zap.String("correlation_id", strings.TrimSpace(req.CorrelationID)),
		zap.String("script_id", strings.TrimSpace(req.ScriptID)),
		zap.String("dst", addr.String()),
	)
	if m.cfg.OnEvent != nil {
		m.cfg.OnEvent(DialEvent{
			CallID:        callID,
			CorrelationID: strings.TrimSpace(req.CorrelationID),
			Scenario:      req.Scenario,
			MediaProfile:  req.MediaProfile,
			State:         DialEventInvited,
			At:            time.Now(),
		})
	}
	return callID, nil
}

type outLeg struct {
	m       *Manager
	params  inviteParams
	req     DialRequest
	rtpSess *rtp.Session
	dst     *net.UDPAddr

	mu          sync.Mutex
	established bool
	callSession *sipSession.CallSession

	sigMu         sync.Mutex
	byeToHeader   string // To from 200 OK (remote tag)
	byeRequestURI string // in-dialog Request-URI (Contact)
	byeRemote     *net.UDPAddr
	byeCSeqNext   int
}

func (leg *outLeg) handleResponse(ctx context.Context, resp *protocol.Message, from *net.UDPAddr) {
	if leg == nil || resp == nil {
		return
	}
	st := resp.StatusCode
	cseqAll := strings.ToUpper(resp.GetHeader("CSeq"))
	if strings.Contains(cseqAll, "BYE") {
		if st >= 200 && st < 300 {
			leg.cleanupLeg()
		}
		return
	}
	if st >= 100 && st < 200 {
		if st != 100 && from != nil {
			logger.Info("sip outbound provisional response",
				zap.String("call_id", leg.params.CallID),
				zap.Int("status", st),
				zap.String("remote", from.String()),
				zap.String("scenario", string(leg.req.Scenario)),
				zap.String("correlation_id", strings.TrimSpace(leg.req.CorrelationID)))
		}
		if leg.m.cfg.OnEvent != nil {
			leg.m.cfg.OnEvent(DialEvent{
				CallID:        leg.params.CallID,
				CorrelationID: strings.TrimSpace(leg.req.CorrelationID),
				Scenario:      leg.req.Scenario,
				MediaProfile:  leg.req.MediaProfile,
				State:         DialEventProvisional,
				StatusCode:    st,
				At:            time.Now(),
			})
		}
		return
	}
	if st != 200 {
		reason := strings.TrimSpace(resp.StatusText)
		if reason == "" {
			reason = "non_200"
		}
		logger.Warn("sip outbound non-200 response",
			zap.String("call_id", leg.params.CallID),
			zap.Int("status", st),
			zap.String("reason", reason),
			zap.String("remote", udpAddrString(from)),
			zap.String("content_type", strings.TrimSpace(resp.GetHeader("Content-Type"))),
			zap.Int("content_length", len(resp.Body)),
			zap.String("body_preview", previewBody(resp.Body, 500)),
		)
		if leg.m.cfg.OnEvent != nil {
			leg.m.cfg.OnEvent(DialEvent{
				CallID:        leg.params.CallID,
				CorrelationID: strings.TrimSpace(leg.req.CorrelationID),
				Scenario:      leg.req.Scenario,
				MediaProfile:  leg.req.MediaProfile,
				State:         DialEventFailed,
				StatusCode:    st,
				Reason:        reason,
				At:            time.Now(),
			})
		}
		leg.cleanupLeg()
		return
	}
	cseq := resp.GetHeader("CSeq")
	if !strings.Contains(strings.ToUpper(cseq), "INVITE") {
		return
	}

	leg.mu.Lock()
	if leg.established {
		leg.mu.Unlock()
		return
	}
	leg.mu.Unlock()

	if strings.TrimSpace(resp.Body) == "" {
		logger.Warn("sip outbound 200 OK without SDP", zap.String("call_id", leg.params.CallID))
		leg.cleanupLeg()
		return
	}

	sdp, err := protocol.ParseSDP(resp.Body)
	if err != nil {
		logger.Warn("sip outbound bad answer SDP", zap.String("call_id", leg.params.CallID), zap.Error(err))
		leg.cleanupLeg()
		return
	}
	remoteIP := net.ParseIP(sdp.IP)
	if remoteIP == nil || sdp.Port <= 0 {
		logger.Warn("sip outbound invalid RTP in answer", zap.String("call_id", leg.params.CallID))
		leg.cleanupLeg()
		return
	}
	leg.rtpSess.SetRemoteAddr(&net.UDPAddr{IP: remoteIP, Port: sdp.Port})

	cs, err := sipSession.NewCallSession(leg.params.CallID, leg.rtpSess, sdp.Codecs)
	if err != nil {
		logger.Warn("sip outbound CallSession", zap.String("call_id", leg.params.CallID), zap.Error(err))
		leg.cleanupLeg()
		return
	}

	ackURI := ackRequestURI(resp, leg.params.RequestURI)
	ack := buildACK(leg.params, resp, ackURI)
	if ack == nil {
		leg.cleanupLeg()
		return
	}
	dst := from
	if dst == nil {
		dst = leg.dst
	}
	if err := leg.m.send(ack, dst); err != nil {
		logger.Warn("sip outbound ACK failed", zap.String("call_id", leg.params.CallID), zap.Error(err))
		cs.Stop()
		leg.cleanupLeg()
		return
	}

	leg.sigMu.Lock()
	leg.byeToHeader = resp.GetHeader("To")
	leg.byeRequestURI = ackRequestURI(resp, leg.params.RequestURI)
	if from != nil {
		leg.byeRemote = cloneUDPAddr(from)
	} else {
		leg.byeRemote = cloneUDPAddr(leg.dst)
	}
	leg.byeCSeqNext = leg.params.CSeq + 1
	leg.sigMu.Unlock()

	leg.mu.Lock()
	leg.established = true
	leg.callSession = cs
	leg.mu.Unlock()

	if leg.m.cfg.OnRegisterSession != nil {
		leg.m.cfg.OnRegisterSession(leg.params.CallID, cs)
	}

	// Bridge profile owns RTP via conversation.StartTransferBridge (raw relay or PCM transcode fallback).
	// Starting the default MediaSession here would race ReadFromUDP on the same socket and cause noise.
	startDefaultMedia := true
	switch leg.req.MediaProfile {
	case MediaProfileAI:
		if leg.m.cfg.MediaAttach != nil {
			if err := leg.m.cfg.MediaAttach(ctx, cs); err != nil {
				logger.Warn("sip outbound media attach", zap.String("call_id", leg.params.CallID), zap.Error(err))
			}
		}
	case MediaProfileScript:
		if leg.m.cfg.MediaAttach != nil {
			if err := leg.m.cfg.MediaAttach(ctx, cs); err != nil {
				logger.Warn("sip outbound script media attach", zap.String("call_id", leg.params.CallID), zap.Error(err))
			}
		}
		if leg.m.cfg.OnScript != nil {
			fromH := formatOutboundFromHeader(leg.params.FromDisplayName, leg.params.FromUser,
				leg.params.SIPHost, leg.params.SIPPort, leg.params.FromTag)
			leg.m.cfg.OnScript(ctx, EstablishedLeg{
				CallID:              leg.params.CallID,
				Scenario:            leg.req.Scenario,
				CorrelationID:       leg.req.CorrelationID,
				Session:             cs,
				CreatedAt:           time.Now(),
				FromHeader:          fromH,
				ToHeader:            leg.params.RequestURI,
				RemoteSignalingAddr: leg.dst.String(),
				CSeqInvite:          fmt.Sprintf("%d INVITE", leg.params.CSeq),
			}, strings.TrimSpace(leg.req.ScriptID))
		}
	case MediaProfileTransferBridge:
		startDefaultMedia = false
		cid := strings.TrimSpace(leg.req.CorrelationID)
		if cid == "" {
			logger.Warn("sip outbound bridge: empty correlation id (inbound Call-ID)",
				zap.String("call_id", leg.params.CallID))
			leg.cleanupLeg()
			return
		}
		if leg.m.cfg.OnTransferBridge != nil {
			leg.m.cfg.OnTransferBridge(cid, cs, leg.params.CallID)
		} else {
			logger.Warn("sip outbound bridge: OnTransferBridge not configured",
				zap.String("call_id", leg.params.CallID))
		}
	default:
		// MediaProfileNone
	}

	if startDefaultMedia {
		cs.StartOnACK()
	}

	if leg.m.cfg.OnEstablished != nil {
		fromH := formatOutboundFromHeader(leg.params.FromDisplayName, leg.params.FromUser,
			leg.params.SIPHost, leg.params.SIPPort, leg.params.FromTag)
		leg.m.cfg.OnEstablished(EstablishedLeg{
			CallID:              leg.params.CallID,
			Scenario:            leg.req.Scenario,
			CorrelationID:       leg.req.CorrelationID,
			Session:             cs,
			CreatedAt:           time.Now(),
			FromHeader:          fromH,
			ToHeader:            leg.params.RequestURI,
			RemoteSignalingAddr: leg.dst.String(),
			CSeqInvite:          fmt.Sprintf("%d INVITE", leg.params.CSeq),
		})
	}
	if leg.m.cfg.OnEvent != nil {
		leg.m.cfg.OnEvent(DialEvent{
			CallID:        leg.params.CallID,
			CorrelationID: strings.TrimSpace(leg.req.CorrelationID),
			Scenario:      leg.req.Scenario,
			MediaProfile:  leg.req.MediaProfile,
			State:         DialEventEstablished,
			StatusCode:    200,
			At:            time.Now(),
		})
	}

	logger.Info("sip outbound established",
		zap.String("call_id", leg.params.CallID),
		zap.String("correlation_id", strings.TrimSpace(leg.req.CorrelationID)),
		zap.String("scenario", string(leg.req.Scenario)),
		zap.String("media_profile", string(leg.req.MediaProfile)),
		zap.String("negotiated_codec", cs.NegotiatedCodec().Name),
		zap.Int("negotiated_clock_rate", cs.NegotiatedCodec().ClockRate),
		zap.Int("negotiated_channels", cs.NegotiatedCodec().Channels),
	)
}

func previewBody(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" || max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

func udpAddrString(a *net.UDPAddr) string {
	if a == nil {
		return ""
	}
	return a.String()
}

func (leg *outLeg) cleanupLeg() {
	if leg == nil || leg.m == nil {
		return
	}
	callID := leg.params.CallID
	m := leg.m
	m.mu.Lock()
	delete(m.legs, callID)
	m.mu.Unlock()
	if leg.rtpSess != nil {
		_ = leg.rtpSess.Close()
	}
}

func randomHex(nBytes int) string {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", nBytes)
	}
	return hex.EncodeToString(b)
}
