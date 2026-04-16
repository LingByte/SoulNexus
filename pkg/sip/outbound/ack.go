package outbound

import (
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/sip/protocol"
	"github.com/LingByte/SoulNexus/pkg/sip/siputil"
)

// buildACK builds a SIP ACK for a completed INVITE transaction (200 OK with SDP answer).
func buildACK(inv inviteParams, resp200 *protocol.Message, requestURI string) *protocol.Message {
	if resp200 == nil {
		return nil
	}
	reqURI := strings.TrimSpace(requestURI)
	if reqURI == "" {
		reqURI = inv.RequestURI
	}

	msg := &protocol.Message{
		IsRequest:  true,
		Method:     protocol.MethodAck,
		RequestURI: reqURI,
		Version:    "SIP/2.0",
	}
	// Via must match INVITE (single Via for our client)
	via := fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=z9hG4bK%s;rport",
		nonEmpty(inv.SIPHost, "127.0.0.1"), nonZero(inv.SIPPort, 5060), inv.Branch)
	msg.SetHeader("Via", via)
	msg.SetHeader("Max-Forwards", "70")

	msg.SetHeader("From", formatOutboundFromHeader(inv.FromDisplayName, inv.FromUser, inv.SIPHost, inv.SIPPort, inv.FromTag))
	if to := resp200.GetHeader("To"); to != "" {
		msg.SetHeader("To", to)
	} else {
		msg.SetHeader("To", inv.RequestURI)
	}
	msg.SetHeader("Call-ID", inv.CallID)
	msg.SetHeader("CSeq", siputil.WithCSeqACK(inv.CSeq))
	msg.SetHeader("Content-Length", "0")
	return msg
}

// ackRequestURI prefers Contact from 200 OK (RFC 3261).
func ackRequestURI(resp200 *protocol.Message, fallback string) string {
	if resp200 == nil {
		return fallback
	}
	c := strings.TrimSpace(resp200.GetHeader("Contact"))
	if c == "" {
		return fallback
	}
	c = strings.TrimPrefix(c, "<")
	c = strings.TrimSuffix(c, ">")
	if idx := strings.Index(c, ";"); idx > 0 {
		c = c[:idx]
	}
	c = strings.TrimSpace(c)
	if strings.HasPrefix(strings.ToLower(c), "sip:") {
		return c
	}
	return fallback
}
