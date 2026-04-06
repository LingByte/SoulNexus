package outbound

import (
	"fmt"
	"net"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/sip/protocol"
)

func buildBYE(inv inviteParams, toHeader200, requestURI string, cseq int, branch string) *protocol.Message {
	reqURI := strings.TrimSpace(requestURI)
	if reqURI == "" {
		reqURI = inv.RequestURI
	}
	msg := &protocol.Message{
		IsRequest:  true,
		Method:     protocol.MethodBye,
		RequestURI: reqURI,
		Version:    "SIP/2.0",
	}
	via := fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=z9hG4bK%s;rport",
		nonEmpty(inv.SIPHost, "127.0.0.1"), nonZero(inv.SIPPort, 5060), branch)
	msg.SetHeader("Via", via)
	msg.SetHeader("Max-Forwards", "70")
	msg.SetHeader("From", formatOutboundFromHeader(inv.FromDisplayName, inv.FromUser, inv.SIPHost, inv.SIPPort, inv.FromTag))
	if strings.TrimSpace(toHeader200) != "" {
		msg.SetHeader("To", toHeader200)
	} else {
		msg.SetHeader("To", formatToHeader(inv.RequestURI))
	}
	msg.SetHeader("Call-ID", inv.CallID)
	msg.SetHeader("CSeq", fmt.Sprintf("%d BYE", cseq))
	msg.SetHeader("Content-Length", "0")
	return msg
}

// SendBYE sends an in-dialog BYE for an established outbound leg (after 200 OK to INVITE).
func (m *Manager) SendBYE(callID string) error {
	if m == nil || m.send == nil {
		return fmt.Errorf("sip/outbound: manager not ready")
	}
	callID = strings.TrimSpace(callID)
	if callID == "" {
		return fmt.Errorf("sip/outbound: empty call-id")
	}
	m.mu.Lock()
	leg := m.legs[callID]
	m.mu.Unlock()
	if leg == nil {
		return fmt.Errorf("sip/outbound: unknown call-id %s", callID)
	}
	leg.sigMu.Lock()
	defer leg.sigMu.Unlock()
	if strings.TrimSpace(leg.byeToHeader) == "" {
		return fmt.Errorf("sip/outbound: dialog not ready for BYE")
	}
	dst := leg.byeRemote
	if dst == nil {
		dst = leg.dst
	}
	if dst == nil {
		return fmt.Errorf("sip/outbound: no signaling address for BYE")
	}
	cseq := leg.byeCSeqNext
	if cseq <= 0 {
		cseq = leg.params.CSeq + 1
	}
	leg.byeCSeqNext = cseq + 1
	branch := randomHex(10)
	msg := buildBYE(leg.params, leg.byeToHeader, leg.byeRequestURI, cseq, branch)
	return m.send(msg, dst)
}

func cloneUDPAddr(a *net.UDPAddr) *net.UDPAddr {
	if a == nil {
		return nil
	}
	b := *a
	return &b
}
