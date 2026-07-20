package middleware

import (
	"net"
	"strings"
)

// IPList holds parsed IP/CIDR entries for allow/block checks.
type IPList struct {
	Any  bool
	IPs  []net.IP
	Nets []*net.IPNet
}

// ParseIPList parses a comma-separated list of IPs and CIDRs.
// "*" means allow any (used by allowlists only).
func ParseIPList(raw string) IPList {
	out := IPList{}
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		if p == "*" {
			out.Any = true
			continue
		}
		if strings.Contains(p, "/") {
			if _, n, err := net.ParseCIDR(p); err == nil {
				out.Nets = append(out.Nets, n)
			}
			continue
		}
		if ip := net.ParseIP(p); ip != nil {
			out.IPs = append(out.IPs, ip)
		}
	}
	return out
}

func (l IPList) Empty() bool {
	return !l.Any && len(l.IPs) == 0 && len(l.Nets) == 0
}

// Contains reports whether ipStr matches the list.
func (l IPList) Contains(ipStr string) bool {
	if l.Any {
		return true
	}
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}
	for _, allow := range l.IPs {
		if allow.Equal(ip) {
			return true
		}
	}
	for _, n := range l.Nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// IPACLRule is a per-route IP allow/block overlay.
type IPACLRule struct {
	Allowed string
	Blocked string
}

// IPACLDecision evaluates block-first, then allow-if-configured semantics.
func IPACLDecision(globalBlocked, globalAllowed, routeBlocked, routeAllowed IPList, clientIP string) (allowed bool, reason string) {
	if routeBlocked.Contains(clientIP) || globalBlocked.Contains(clientIP) {
		return false, "blocked"
	}
	if !routeAllowed.Empty() {
		if routeAllowed.Contains(clientIP) {
			return true, ""
		}
		return false, "route_allowlist"
	}
	if !globalAllowed.Empty() {
		if globalAllowed.Contains(clientIP) {
			return true, ""
		}
		return false, "global_allowlist"
	}
	return true, ""
}
