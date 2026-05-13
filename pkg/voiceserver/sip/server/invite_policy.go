package server

import (
	"net"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

func parseIPCIDRList(csv string) []*net.IPNet {
	var out []*net.IPNet
	for _, raw := range strings.Split(csv, ",") {
		tok := strings.TrimSpace(raw)
		if tok == "" {
			continue
		}
		n, err := parseOneIPCIDR(tok)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out
}

func parseOneIPCIDR(tok string) (*net.IPNet, error) {
	if strings.Contains(tok, "/") {
		_, n, err := net.ParseCIDR(tok)
		return n, err
	}
	ip := net.ParseIP(tok)
	if ip == nil {
		return nil, net.InvalidAddrError(tok)
	}
	if ip4 := ip.To4(); ip4 != nil {
		return &net.IPNet{IP: ip4, Mask: net.CIDRMask(32, 32)}, nil
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}, nil
}

func ipAllowed(nets []*net.IPNet, ip net.IP) bool {
	if len(nets) == 0 {
		return true
	}
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n != nil && n.Contains(ip) {
			return true
		}
	}
	return false
}

type inviteRateState struct {
	mu    sync.Mutex
	limit map[string]*rate.Limiter
}

func (s *inviteRateState) allow(ip net.IP, perSec float64, burst int) bool {
	if perSec <= 0 {
		return true
	}
	if ip == nil {
		return false
	}
	key := ip.String()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.limit == nil {
		s.limit = make(map[string]*rate.Limiter)
	}
	lim, ok := s.limit[key]
	if !ok {
		b := burst
		if b <= 0 {
			b = 5
		}
		lim = rate.NewLimiter(rate.Limit(perSec), b)
		s.limit[key] = lim
	}
	return lim.Allow()
}
