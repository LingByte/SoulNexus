package outbound

import (
	"fmt"
	"strings"
)

// DialTargetFromACDTrunk builds INVITE target from ACD pool trunk row (user part + gateway).
// userPart is TargetValue (digits or dial string); host is SipTrunkHost; port defaults to 5060 when invalid.
// If signalingOverride is empty, SignalingAddr is host:port.
func DialTargetFromACDTrunk(userPart, host, signalingOverride string, port int) (DialTarget, bool) {
	userPart = strings.TrimSpace(userPart)
	host = strings.TrimSpace(host)
	if userPart == "" || host == "" {
		return DialTarget{}, false
	}
	if port <= 0 || port >= 65536 {
		port = 5060
	}
	var t DialTarget
	t.RequestURI = normalizeSIPRequestURI(fmt.Sprintf("sip:%s@%s:%d", userPart, host, port))
	sig := strings.TrimSpace(signalingOverride)
	if sig != "" {
		t.SignalingAddr = sig
	} else {
		t.SignalingAddr = fmt.Sprintf("%s:%d", host, port)
	}
	return t, true
}
