package outbound

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/pkg/sip/protocol"
)

// inviteParams carries dialog fields needed for INVITE and later ACK.
type inviteParams struct {
	LocalIP       string
	SIPHost       string
	SIPPort       int
	RequestURI    string
	CallID        string
	FromTag       string
	Branch        string
	CSeq          int
	LocalRTPPort  int
	SDPBody       string
	FromUser      string // sip:FromUser@host:port
}

func buildINVITE(p inviteParams) *protocol.Message {
	via := fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=z9hG4bK%s;rport",
		nonEmpty(p.SIPHost, "127.0.0.1"), nonZero(p.SIPPort, 5060), p.Branch)

	from := fmt.Sprintf("<sip:%s@%s:%d>;tag=%s", p.FromUser, p.SIPHost, p.SIPPort, p.FromTag)
	to := formatToHeader(p.RequestURI)

	msg := &protocol.Message{
		IsRequest:  true,
		Method:     protocol.MethodInvite,
		RequestURI: p.RequestURI,
		Version:    "SIP/2.0",
		Body:       p.SDPBody,
	}
	msg.SetHeader("Via", via)
	msg.SetHeader("Max-Forwards", "70")
	msg.SetHeader("From", from)
	msg.SetHeader("To", to)
	msg.SetHeader("Call-ID", p.CallID)
	msg.SetHeader("CSeq", fmt.Sprintf("%d INVITE", p.CSeq))
	msg.SetHeader("Contact", fmt.Sprintf("<sip:%s@%s:%d>", p.FromUser, p.SIPHost, p.SIPPort))
	msg.SetHeader("User-Agent", "SoulNexus-SIP/1.0")
	msg.SetHeader("Content-Type", "application/sdp")
	msg.SetHeader("Allow", "INVITE, ACK, BYE, CANCEL, OPTIONS")
	msg.SetHeader("Content-Length", strconv.Itoa(protocol.BodyBytesLen(p.SDPBody)))
	return msg
}

func formatToHeader(requestURI string) string {
	u := strings.TrimSpace(requestURI)
	if u == "" {
		return "<sip:invalid@invalid>"
	}
	if !strings.HasPrefix(strings.ToLower(u), "sip:") {
		u = "sip:" + u
	}
	return "<" + u + ">"
}

func nonEmpty(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}

func nonZero(n, def int) int {
	if n <= 0 {
		return def
	}
	return n
}

// defaultOfferCodecs mirrors typical softphone offers (PCMU + Opus).
func defaultOfferCodecs() []protocol.SDPCodec {
	return []protocol.SDPCodec{
		{PayloadType: 0, Name: "pcmu", ClockRate: 8000, Channels: 1},
		{PayloadType: 111, Name: "opus", ClockRate: 48000, Channels: 1},
	}
}

func newCallID(localIP string) string {
	return fmt.Sprintf("%d@%s", time.Now().UnixNano(), nonEmpty(localIP, "127.0.0.1"))
}
