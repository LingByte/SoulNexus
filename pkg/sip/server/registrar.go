package server

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/sip/protocol"
	"go.uber.org/zap"
)

// sipRegisterPasswordHeader is sent by SIP clients when SIP_PASSWORD is set server-side.
const sipRegisterPasswordHeader = "X-SIP-Register-Password"

// registerPasswordOK is true when REGISTER is allowed: SIP_PASSWORD env empty, or header matches.
func registerPasswordOK(msg *protocol.Message) bool {
	required := config.RegisterPasswordFromEnv()
	if required == "" {
		return true
	}
	if msg == nil {
		return false
	}
	got := strings.TrimSpace(msg.GetHeader(sipRegisterPasswordHeader))
	if len(got) != len(required) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(required)) == 1
}

func randomBranch() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "srvbranch"
	}
	return hex.EncodeToString(b)
}

// stripAngle extracts addr-spec from "<sip:...>" or returns trimmed s.
func stripAngle(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "<") && strings.HasSuffix(s, ">") {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

// extractSIPAddrSpec pulls sip:... out of a name-addr field value, e.g.
// "Bob" <sip:bob@example.com>;tag=1  ->  sip:bob@example.com
func extractSIPAddrSpec(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.LastIndex(s, "<"); idx >= 0 {
		rest := s[idx+1:]
		if end := strings.Index(rest, ">"); end >= 0 {
			return strings.TrimSpace(rest[:end])
		}
	}
	return s
}

// parseURIUserHost parses sip:user@host or sip:user@host:port into user and host (no port in host).
// Accepts raw header values including name-addr (display name + angle brackets).
func parseURIUserHost(uri string) (user, host string, ok bool) {
	u := extractSIPAddrSpec(uri)
	u = stripAngle(u)
	if !strings.HasPrefix(strings.ToLower(u), "sip:") {
		return "", "", false
	}
	u = u[4:]
	at := strings.Index(u, "@")
	if at < 0 {
		return "", "", false
	}
	user = strings.TrimSpace(u[:at])
	rest := strings.TrimSpace(u[at+1:])
	if semi := strings.Index(rest, ";"); semi >= 0 {
		rest = rest[:semi]
	}
	host = rest
	if colon := strings.LastIndex(host, ":"); colon > 0 {
		// IPv6 in brackets [::1] — if host starts with [, find closing ]
		if strings.HasPrefix(host, "[") {
			if end := strings.Index(host, "]"); end > 0 {
				host = host[:end+1]
			}
		} else {
			host = host[:colon]
		}
	}
	if user == "" || host == "" {
		return "", "", false
	}
	return user, host, true
}

// registrationKey is "user@host" lowercased for map lookup.
func registrationKey(user, host string) string {
	return strings.ToLower(strings.TrimSpace(user) + "@" + strings.TrimSpace(host))
}

// parseContactUDPAddr prefers host:port from Contact; falls back to src.
func parseContactUDPAddr(contact string, src *net.UDPAddr) *net.UDPAddr {
	c := extractSIPAddrSpec(contact)
	c = stripAngle(c)
	if !strings.HasPrefix(strings.ToLower(c), "sip:") {
		if src != nil {
			return src
		}
		return nil
	}
	rest := c[4:]
	at := strings.Index(rest, "@")
	if at < 0 {
		if src != nil {
			return src
		}
		return nil
	}
	hostport := rest[at+1:]
	if semi := strings.Index(hostport, ";"); semi >= 0 {
		hostport = hostport[:semi]
	}
	hostport = strings.TrimSpace(hostport)
	if hostport == "" {
		if src != nil {
			return src
		}
		return nil
	}
	host, pstr, err := net.SplitHostPort(hostport)
	if err != nil {
		// no port in URI — use default 5060 for SIP
		ip := net.ParseIP(hostport)
		if ip != nil {
			return &net.UDPAddr{IP: ip, Port: 5060}
		}
		if src != nil {
			return src
		}
		return nil
	}
	port, err := strconv.Atoi(pstr)
	if err != nil || port <= 0 {
		port = 5060
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			if src != nil {
				return src
			}
			return nil
		}
		ip = ips[0]
	}
	return &net.UDPAddr{IP: ip, Port: port}
}

func parseExpiresRegister(msg *protocol.Message) (seconds int, ok bool) {
	if msg == nil {
		return 0, false
	}
	// Header Expires
	if e := strings.TrimSpace(msg.GetHeader("Expires")); e != "" {
		if n, err := strconv.Atoi(e); err == nil {
			return n, true
		}
	}
	// Contact ...;expires=n
	contacts := msg.GetHeaders("Contact")
	for _, c := range contacts {
		low := strings.ToLower(c)
		if idx := strings.Index(low, "expires="); idx >= 0 {
			sub := c[idx+len("expires="):]
			if semi := strings.Index(sub, ";"); semi >= 0 {
				sub = sub[:semi]
			}
			sub = strings.TrimSpace(sub)
			if n, err := strconv.Atoi(sub); err == nil {
				return n, true
			}
		}
	}
	return 3600, true
}

func (s *SIPServer) upsertRegistration(msg *protocol.Message, src *net.UDPAddr) {
	if s == nil || msg == nil || src == nil {
		return
	}
	st := s.registerStore()
	if st == nil {
		return
	}
	to := msg.GetHeader("To")
	user, host, ok := parseURIUserHost(to)
	if !ok {
		logger.Warn("sip register: could not parse To AOR",
			zap.String("to", to))
		return
	}
	key := registrationKey(user, host)
	sec, _ := parseExpiresRegister(msg)
	contact := strings.TrimSpace(msg.GetHeader("Contact"))
	dst := parseContactUDPAddr(contact, src)
	if dst == nil {
		logger.Warn("sip register: no Contact / UDP target",
			zap.String("aor", key),
			zap.String("contact", contact))
		return
	}

	ctx := context.Background()
	if sec <= 0 {
		if err := st.DeleteRegister(ctx, user, host); err != nil {
			logger.Warn("sip register db delete failed",
				zap.String("aor", key),
				zap.Error(err))
		} else {
			logger.Info("sip register removed",
				zap.String("aor", key),
				zap.String("remote", src.String()))
		}
		return
	}
	exp := time.Now().Add(time.Duration(sec) * time.Second)
	ua := msg.GetHeader("User-Agent")
	if err := st.SaveRegister(ctx, user, host, contact, dst, exp, ua); err != nil {
		logger.Warn("sip register db save failed",
			zap.String("aor", key),
			zap.Error(err))
		return
	}
	logger.Info("sip register bound",
		zap.String("aor", key),
		zap.String("dst", dst.String()),
		zap.Int("expires_sec", sec))
}

// prependProxyVia adds a Via on top so responses route back through this server.
func prependProxyVia(msg *protocol.Message, sipHost string, sipPort int) {
	if msg == nil {
		return
	}
	if sipPort <= 0 {
		sipPort = 5060
	}
	h := strings.TrimSpace(sipHost)
	if h == "" {
		h = "127.0.0.1"
	}
	branch := "z9hG4bK" + randomBranch()
	via := fmt.Sprintf("SIP/2.0/UDP %s:%d;branch=%s;rport", h, sipPort, branch)
	old := msg.GetHeaders("via")
	if len(old) == 0 {
		if v := msg.GetHeader("Via"); v != "" {
			old = []string{v}
		}
	}
	msg.HeadersMulti["via"] = append([]string{via}, old...)
	if len(msg.HeadersMulti["via"]) > 0 {
		msg.Headers["via"] = msg.HeadersMulti["via"][0]
	}
	if mf := strings.TrimSpace(msg.GetHeader("Max-Forwards")); mf != "" {
		if n, err := strconv.Atoi(mf); err == nil && n > 0 {
			msg.SetHeader("Max-Forwards", strconv.Itoa(n-1))
		}
	}
}

func (s *SIPServer) proxyInviteToRegistrar(msg *protocol.Message, dst *net.UDPAddr) error {
	if s == nil || s.proto == nil || msg == nil || dst == nil {
		return fmt.Errorf("sip: proxy invite: nil")
	}
	raw := msg.String()
	fwd, err := protocol.Parse(raw)
	if err != nil {
		return err
	}
	prependProxyVia(fwd, s.localIP, s.listenPort)
	// Ensure Content-Length matches normalized body after parse/re-serialize.
	fwd.SetHeader("Content-Length", strconv.Itoa(protocol.BodyBytesLen(fwd.Body)))
	return s.proto.Send(fwd, dst)
}
