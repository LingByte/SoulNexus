package server

import (
	"net"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

func (s *SIPServer) handleNotify(msg *stack.Message, addr *net.UDPAddr) *stack.Message {
	if s == nil || msg == nil {
		return nil
	}
	if s.absorbNonInviteRetransmit(msg, addr) {
		return nil
	}
	ev := strings.ToLower(strings.TrimSpace(msg.GetHeader("Event")))
	if strings.HasPrefix(ev, "refer") {
		resp := s.makeResponse(msg, 200, "OK", "", "")
		resp.SetHeader("Content-Length", "0")
		return resp
	}
	return s.handleNotifyPresence(msg, addr)
}

func (s *SIPServer) handleUpdate(msg *stack.Message, addr *net.UDPAddr) *stack.Message {
	if s == nil || msg == nil {
		return nil
	}
	if s.absorbNonInviteRetransmit(msg, addr) {
		return nil
	}
	resp := s.makeResponse(msg, 200, "OK", "", "")
	resp.SetHeader("Content-Length", "0")
	return resp
}

func (s *SIPServer) handleMessage(msg *stack.Message, addr *net.UDPAddr) *stack.Message {
	if s == nil || msg == nil {
		return nil
	}
	if s.absorbNonInviteRetransmit(msg, addr) {
		return nil
	}
	if strings.TrimSpace(msg.Body) == "" {
		resp := s.makeResponse(msg, 200, "OK", "", "")
		resp.SetHeader("Content-Length", "0")
		return resp
	}
	resp := s.makeResponse(msg, 415, "Unsupported Media Type", "", "")
	resp.SetHeader("Accept", "text/plain")
	resp.SetHeader("Content-Length", "0")
	return resp
}
