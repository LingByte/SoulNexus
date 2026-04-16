package outbound

import (
	"strings"

	"github.com/LingByte/SoulNexus/pkg/config"
)

func normalizeSIPRequestURI(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return u
	}
	if !strings.HasPrefix(strings.ToLower(u), "sip:") {
		u = "sip:" + u
	}
	return u
}

// DialTargetFromEnv reads SIP_* outbound env vars (see pkg/config) and maps them to DialTarget.
func DialTargetFromEnv() (DialTarget, bool) {
	e, ok := config.DialTargetFromEnv()
	if !ok {
		return DialTarget{}, false
	}
	return DialTarget{RequestURI: e.RequestURI, SignalingAddr: e.SignalingAddr, WebSeat: e.WebSeat}, true
}

// TransferDialTargetFromEnv reads SIP_TRANSFER_* env vars via pkg/config.
func TransferDialTargetFromEnv() (DialTarget, bool) {
	e, ok := config.TransferDialTargetFromEnv()
	if !ok {
		return DialTarget{}, false
	}
	return DialTarget{RequestURI: e.RequestURI, SignalingAddr: e.SignalingAddr, WebSeat: e.WebSeat}, true
}
