package transaction

import (
	"fmt"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/voiceserver/sip/stack"
)

// AckRequestURIFor2xx prefers Contact from a 2xx response (RFC 3261 dialog establishment).
func AckRequestURIFor2xx(resp *stack.Message, inviteRequestURI string) string {
	if resp == nil {
		return strings.TrimSpace(inviteRequestURI)
	}
	c := strings.TrimSpace(resp.GetHeader("Contact"))
	if c == "" {
		return strings.TrimSpace(inviteRequestURI)
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
	return strings.TrimSpace(inviteRequestURI)
}

// BuildAckForInvite builds an ACK for the completed INVITE transaction.
// For status 200–299, requestURI should usually be AckRequestURIFor2xx(resp, invite.RequestURI).
// For 300–699, requestURI should be the same as the INVITE Request-URI (RFC 3261 §17.1.1.2).
func BuildAckForInvite(invite *stack.Message, final *stack.Message, requestURI string) (*stack.Message, error) {
	if invite == nil || final == nil {
		return nil, fmt.Errorf("sip1/transaction: nil message")
	}
	if !IsInviteCSeq(invite) {
		return nil, fmt.Errorf("sip1/transaction: invite CSeq is not INVITE")
	}
	st := final.StatusCode
	if st < 200 || st > 699 {
		return nil, fmt.Errorf("sip1/transaction: final status %d is not a final response", st)
	}

	reqURI := strings.TrimSpace(requestURI)
	if reqURI == "" {
		reqURI = strings.TrimSpace(invite.RequestURI)
	}

	n := stack.ParseCSeqNum(invite.GetHeader("CSeq"))
	if n <= 0 {
		return nil, fmt.Errorf("sip1/transaction: invalid CSeq on INVITE")
	}

	ack := &stack.Message{
		IsRequest:    true,
		Method:       stack.MethodAck,
		RequestURI:   reqURI,
		Version:      "SIP/2.0",
		Headers:      map[string]string{},
		HeadersMulti: map[string][]string{},
	}

	// Single Via: reuse the top Via from the INVITE (UAC behavior).
	if v := TopVia(invite); v != "" {
		ack.SetHeader("Via", v)
	} else {
		return nil, fmt.Errorf("sip1/transaction: invite missing Via")
	}
	ack.SetHeader("Max-Forwards", "70")

	if f := strings.TrimSpace(invite.GetHeader("From")); f != "" {
		ack.SetHeader("From", f)
	}
	if to := strings.TrimSpace(final.GetHeader("To")); to != "" {
		ack.SetHeader("To", to)
	} else if to := strings.TrimSpace(invite.GetHeader("To")); to != "" {
		ack.SetHeader("To", to)
	}
	ack.SetHeader("Call-ID", strings.TrimSpace(invite.GetHeader("Call-ID")))
	ack.SetHeader("CSeq", stack.WithCSeqACK(n))
	ack.SetHeader("Content-Length", "0")
	return ack, nil
}
