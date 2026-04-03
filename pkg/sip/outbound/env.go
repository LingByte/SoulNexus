package outbound

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/LingByte/SoulNexus/pkg/utils"
)

// Environment variable keys for outbound dialing (SoulNexus .env, read via utils.GetEnv / LookupEnv).
const (
	EnvSIPTargetNumber    = "SIP_TARGET_NUMBER"         // user part, e.g. extension or E.164 local part
	EnvSIPOutboundHost    = "SIP_OUTBOUND_HOST"         // domain or IP for Request-URI host
	EnvSIPOutboundPort    = "SIP_OUTBOUND_PORT"         // port in Request-URI (default 5060)
	EnvSIPSignalingAddr   = "SIP_SIGNALING_ADDR"        // UDP host:port where INVITE is sent (default host:port from above)
	EnvSIPOutboundReqURI  = "SIP_OUTBOUND_REQUEST_URI"  // optional full override, e.g. sip:user@domain:5060;user=phone
	EnvSIPOutboundAutoDial = "SIP_OUTBOUND_AUTO_DIAL" // "true"/"1" to Dial once at sip process startup

	// DB-backed dial (sip_users): optional domain filter and Request-URI port when building sip:user@domain:port.
	EnvSIPDefaultDomain   = "SIP_DEFAULT_DOMAIN"
	EnvSIPDefaultURIPort  = "SIP_DEFAULT_URI_PORT"

	// Transfer-to-agent (press 0): separate from campaign outbound.
	EnvSIPTransferReqURI  = "SIP_TRANSFER_REQUEST_URI"
	EnvSIPTransferSigAddr = "SIP_TRANSFER_SIGNALING_ADDR"
	EnvSIPTransferNumber  = "SIP_TRANSFER_NUMBER"
	EnvSIPTransferHost    = "SIP_TRANSFER_HOST"
	EnvSIPTransferPort    = "SIP_TRANSFER_PORT"
)

// DialTargetFromEnv builds DialTarget from .env using utils.GetEnv.
//
// Modes:
//   1) SIP_OUTBOUND_REQUEST_URI + SIP_SIGNALING_ADDR — full control.
//   2) SIP_TARGET_NUMBER + SIP_OUTBOUND_HOST — builds sip:TARGET@HOST:PORT and signaling HOST:PORT unless SIP_SIGNALING_ADDR is set.
//
// Returns ok=false if required variables are missing.
func DialTargetFromEnv() (t DialTarget, ok bool) {
	reqURI := strings.TrimSpace(utils.GetEnv(EnvSIPOutboundReqURI))
	sig := strings.TrimSpace(utils.GetEnv(EnvSIPSignalingAddr))

	if reqURI != "" {
		if sig == "" {
			return DialTarget{}, false
		}
		t.RequestURI = normalizeSIPRequestURI(reqURI)
		t.SignalingAddr = sig
		return t, true
	}

	target := strings.TrimSpace(utils.GetEnv(EnvSIPTargetNumber))
	host := strings.TrimSpace(utils.GetEnv(EnvSIPOutboundHost))
	if target == "" || host == "" {
		return DialTarget{}, false
	}

	port := 5060
	if ps := strings.TrimSpace(utils.GetEnv(EnvSIPOutboundPort)); ps != "" {
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
	v := strings.ToLower(strings.TrimSpace(utils.GetEnv(EnvSIPOutboundAutoDial)))
	return v == "1" || v == "true" || v == "yes"
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

// TransferDialTargetFromEnv reads SIP_TRANSFER_* (agent extension / URI for blind transfer dial).
// Same shape as DialTargetFromEnv but separate keys so campaign and transfer can coexist.
func TransferDialTargetFromEnv() (t DialTarget, ok bool) {
	reqURI := strings.TrimSpace(utils.GetEnv(EnvSIPTransferReqURI))
	sig := strings.TrimSpace(utils.GetEnv(EnvSIPTransferSigAddr))

	if reqURI != "" {
		if sig == "" {
			return DialTarget{}, false
		}
		t.RequestURI = normalizeSIPRequestURI(reqURI)
		t.SignalingAddr = sig
		return t, true
	}

	num := strings.TrimSpace(utils.GetEnv(EnvSIPTransferNumber))
	host := strings.TrimSpace(utils.GetEnv(EnvSIPTransferHost))
	if num == "" || host == "" {
		return DialTarget{}, false
	}

	port := 5060
	if ps := strings.TrimSpace(utils.GetEnv(EnvSIPTransferPort)); ps != "" {
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
