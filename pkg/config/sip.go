package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/constants"
	"github.com/LingByte/SoulNexus/pkg/utils"
)

// SIPDialEnv holds outbound dial fields parsed from environment. Callers map this to
// pkg/sip/outbound.DialTarget at the SIP boundary so this package does not import outbound
// (outbound HTTP helpers live in the same module and would create an import cycle).
type SIPDialEnv struct {
	RequestURI    string
	SignalingAddr string
	WebSeat       bool
}

// DialTargetFromEnv builds SIP dial fields from .env using utils.GetEnv.
//
// Modes:
//  1. SIP_OUTBOUND_REQUEST_URI + SIP_SIGNALING_ADDR — full control.
//  2. SIP_TARGET_NUMBER + SIP_OUTBOUND_HOST — builds sip:TARGET@HOST:PORT and signaling HOST:PORT unless SIP_SIGNALING_ADDR is set.
//
// Returns ok=false if required variables are missing.
func DialTargetFromEnv() (t SIPDialEnv, ok bool) {
	reqURI := strings.TrimSpace(utils.GetEnv(constants.EnvSIPOutboundReqURI))
	sig := strings.TrimSpace(utils.GetEnv(constants.EnvSIPSignalingAddr))

	if reqURI != "" {
		if sig == "" {
			return SIPDialEnv{}, false
		}
		t.RequestURI = normalizeSIPRequestURI(reqURI)
		t.SignalingAddr = sig
		return t, true
	}

	target := strings.TrimSpace(utils.GetEnv(constants.EnvSIPTargetNumber))
	host := strings.TrimSpace(utils.GetEnv(constants.EnvSIPOutboundHost))
	if target == "" || host == "" {
		return SIPDialEnv{}, false
	}

	port := 5060
	if ps := strings.TrimSpace(utils.GetEnv(constants.EnvSIPOutboundPort)); ps != "" {
		if p, err := strconv.Atoi(ps); err == nil && p > 0 && p < 65536 {
			port = p
		}
	}

	t.RequestURI = fmt.Sprintf("sip:%s@%s:%d", target, host, port)
	if sig == "" {
		t.SignalingAddr = fmt.Sprintf("%s:%d", host, port)
	} else {
		t.SignalingAddr = sig
	}
	return t, true
}

// AutoDialFromEnv is true when SIP_OUTBOUND_AUTO_DIAL is "1" or "true" (case-insensitive).
func AutoDialFromEnv() bool {
	v := strings.ToLower(strings.TrimSpace(utils.GetEnv(constants.EnvSIPOutboundAutoDial)))
	return v == "1" || v == "true" || v == "yes"
}

// CallerIdentityFromEnv reads SIP_CALLER_ID / SIP_CALLER_DISPLAY_NAME for outbound INVITE From/Contact.
// User is the SIP URI user part; displayName is optional (empty → From has no quoted display-name).
func CallerIdentityFromEnv() (user, displayName string) {
	user = strings.TrimSpace(utils.GetEnv(constants.EnvSIPCallerID))
	displayName = strings.TrimSpace(utils.GetEnv(constants.EnvSIPCallerDisplayName))
	return user, displayName
}

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

// RegisterPasswordFromEnv returns SIP_PASSWORD when set (trimmed). Empty means REGISTER is open
// (no shared secret). Non-empty means clients must send matching X-SIP-Register-Password on REGISTER.
func RegisterPasswordFromEnv() string {
	return strings.TrimSpace(utils.GetEnv(constants.EnvSIPRegisterPassword))
}

// TransferDialTargetFromEnv reads SIP_TRANSFER_* (agent extension / URI for blind transfer dial).
// Same shape as DialTargetFromEnv but separate keys so campaign and transfer can coexist.
func TransferDialTargetFromEnv() (t SIPDialEnv, ok bool) {
	reqURI := strings.TrimSpace(utils.GetEnv(constants.EnvSIPTransferReqURI))
	sig := strings.TrimSpace(utils.GetEnv(constants.EnvSIPTransferSigAddr))

	if reqURI != "" {
		if sig == "" {
			return SIPDialEnv{}, false
		}
		t.RequestURI = normalizeSIPRequestURI(reqURI)
		t.SignalingAddr = sig
		return t, true
	}

	num := strings.TrimSpace(utils.GetEnv(constants.EnvSIPTransferNumber))
	if strings.EqualFold(num, "web") {
		return SIPDialEnv{WebSeat: true}, true
	}
	host := strings.TrimSpace(utils.GetEnv(constants.EnvSIPTransferHost))
	if num == "" || host == "" {
		return SIPDialEnv{}, false
	}

	port := 5060
	if ps := strings.TrimSpace(utils.GetEnv(constants.EnvSIPTransferPort)); ps != "" {
		if p, err := strconv.Atoi(ps); err == nil && p > 0 && p < 65536 {
			port = p
		}
	}

	t.RequestURI = fmt.Sprintf("sip:%s@%s:%d", num, host, port)
	if sig == "" {
		t.SignalingAddr = fmt.Sprintf("%s:%d", host, port)
	} else {
		t.SignalingAddr = sig
	}
	return t, true
}
