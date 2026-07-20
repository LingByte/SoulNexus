package common

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	apperr "github.com/LingByte/SoulNexus/pkg/errors"
	"golang.org/x/net/http/httpproxy"
)

var restrictedHostnames = []string{
	"localhost",
	"127.0.0.1",
	"::1",
	"0.0.0.0",
	"metadata.google.internal",
	"metadata.tencentyun.com",
	"metadata.aws.internal",
	"host.docker.internal",
	"gateway.docker.internal",
	"kubernetes.docker.internal",
	"kubernetes",
	"kubernetes.default",
	"kubernetes.default.svc",
	"kubernetes.default.svc.cluster.local",
}

var restrictedHostSuffixes = []string{
	".local",
	".localhost",
	".internal",
	".corp",
	".lan",
	".home",
	".localdomain",
	".svc.cluster.local",
	".pod.cluster.local",
}

var restrictedIPv4Ranges = []*net.IPNet{
	mustParseCIDR("100.64.0.0/10"),
	mustParseCIDR("198.18.0.0/15"),
	mustParseCIDR("198.51.100.0/24"),
	mustParseCIDR("203.0.113.0/24"),
	mustParseCIDR("192.0.0.0/24"),
	mustParseCIDR("192.0.2.0/24"),
	mustParseCIDR("0.0.0.0/8"),
	mustParseCIDR("240.0.0.0/4"),
	mustParseCIDR("255.255.255.255/32"),
	mustParseCIDR("172.17.0.0/16"),
	mustParseCIDR("172.18.0.0/16"),
	mustParseCIDR("172.19.0.0/16"),
	mustParseCIDR("172.20.0.0/16"),
}

func mustParseCIDR(s string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(s)
	if err != nil {
		panic(fmt.Sprintf("invalid CIDR: %s", s))
	}
	return ipNet
}

func isRestrictedIP(ip net.IP) (bool, string) {
	if ip.IsPrivate() {
		return true, "private IP address"
	}
	if ip.IsLoopback() {
		return true, "loopback address"
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true, "link-local address"
	}
	if ip.IsMulticast() {
		return true, "multicast address"
	}
	if ip.IsUnspecified() {
		return true, "unspecified address"
	}
	if ip4 := ip.To4(); ip4 != nil {
		for _, cidr := range restrictedIPv4Ranges {
			if cidr.Contains(ip4) {
				return true, fmt.Sprintf("restricted range %s", cidr.String())
			}
		}
	}
	if ip.To4() == nil && len(ip) == 16 {
		if ip[0] == 0xfe && (ip[1]&0xc0) == 0xc0 {
			return true, "site-local IPv6 address"
		}
		if (ip[0] & 0xfe) == 0xfc {
			return true, "unique local IPv6 address"
		}
		if isZeros(ip[0:10]) && ip[10] == 0xff && ip[11] == 0xff {
			mappedIP := ip[12:16]
			if restricted, reason := isRestrictedIP(net.IP(mappedIP)); restricted {
				return true, fmt.Sprintf("IPv4-mapped %s", reason)
			}
		}
		if ip[0] == 0x20 && ip[1] == 0x01 && ip[2] == 0x00 && ip[3] == 0x00 {
			return true, "Teredo tunneling address"
		}
		if ip[0] == 0x20 && ip[1] == 0x02 {
			embeddedIP := net.IP(ip[2:6])
			if restricted, reason := isRestrictedIP(embeddedIP); restricted {
				return true, fmt.Sprintf("6to4 embedded %s", reason)
			}
		}
	}
	return false, ""
}

// IsPublicIP reports whether ip is safe for outbound fetch (not private/loopback/etc.).
func IsPublicIP(ip net.IP) bool {
	restricted, _ := isRestrictedIP(ip)
	return !restricted
}

func isZeros(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

var ipLikePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`),
	regexp.MustCompile(`^\d{8,10}$`),
	regexp.MustCompile(`^0[0-7]+\.`),
	regexp.MustCompile(`(?i)^0x[0-9a-f]+\.`),
	regexp.MustCompile(`(?i)^0x[0-9a-f]{6,8}$`),
	regexp.MustCompile(`(?i)^[0-9a-f:]+::[0-9a-f:]*$`),
	regexp.MustCompile(`(?i)^[0-9a-f]{1,4}(:[0-9a-f]{1,4}){7}$`),
	regexp.MustCompile(`(?i)^::ffff:\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`),
	regexp.MustCompile(`(?i)^\[[0-9a-f:]+\]$`),
}

func isIPLikeHostname(hostname string) bool {
	for _, pattern := range ipLikePatterns {
		if pattern.MatchString(hostname) {
			return true
		}
	}
	return false
}

func isSSRFSafeURL(rawURL string) (bool, string) {
	if rawURL == "" {
		return false, "URL is empty"
	}
	if len(rawURL) > 2048 {
		return false, "URL exceeds maximum length"
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false, fmt.Sprintf("invalid URL format: %v", err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return false, fmt.Sprintf("invalid scheme: %s (only http/https allowed)", scheme)
	}
	hostname := parsed.Hostname()
	if hostname == "" {
		return false, "URL has no hostname"
	}
	hostnameLower := strings.ToLower(hostname)
	for _, restricted := range restrictedHostnames {
		if hostnameLower == restricted {
			return false, fmt.Sprintf("hostname %s is restricted", hostname)
		}
	}
	for _, suffix := range restrictedHostSuffixes {
		if strings.HasSuffix(hostnameLower, suffix) {
			return false, fmt.Sprintf("hostname suffix %s is restricted", suffix)
		}
	}
	if net.ParseIP(hostname) != nil {
		return false, "direct IP address access is not allowed, use domain name or add to SSRF_WHITELIST"
	}
	if isIPLikeHostname(hostname) {
		return false, "IP-like hostname format is not allowed"
	}
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return false, fmt.Sprintf("DNS resolution failed for hostname %s: cannot verify if it resolves to safe IP", hostname)
	}
	for _, resolvedIP := range ips {
		if restricted, reason := isRestrictedIP(resolvedIP); restricted {
			return false, fmt.Sprintf("hostname %s resolves to restricted IP %s: %s", hostname, resolvedIP.String(), reason)
		}
	}
	port := parsed.Port()
	if port != "" {
		blockedPorts := map[string]bool{
			"22": true, "23": true, "25": true, "445": true, "3389": true,
			"5432": true, "3306": true, "6379": true, "27017": true, "9200": true,
			"2379": true, "2380": true, "8500": true, "4001": true,
		}
		if blockedPorts[port] {
			return false, fmt.Sprintf("port %s is blocked for security reasons", port)
		}
	}
	return true, ""
}

var (
	ssrfWhitelistOnce   sync.Once
	ssrfWhitelist       *ssrfWhitelistConfig
	ssrfWhitelistAtomic atomic.Pointer[ssrfWhitelistConfig]
)

type ssrfWhitelistConfig struct {
	exactHosts  map[string]bool
	suffixHosts []string
	cidrNets    []*net.IPNet
}

func loadSSRFWhitelist() *ssrfWhitelistConfig {
	if cur := ssrfWhitelistAtomic.Load(); cur != nil {
		return cur
	}
	ssrfWhitelistOnce.Do(func() {
		raw := os.Getenv("SSRF_WHITELIST")
		extra := os.Getenv("SSRF_WHITELIST_EXTRA")
		ssrfWhitelist = parseSSRFWhitelistRaw(mergeSSRFWhitelistRaws(raw, extra))
	})
	return ssrfWhitelist
}

// SetSSRFWhitelistFromRaw replaces the active SSRF whitelist at runtime.
func SetSSRFWhitelistFromRaw(raw string) {
	ssrfWhitelistAtomic.Store(parseSSRFWhitelistRaw(raw))
}

func parseSSRFWhitelistRaw(raw string) *ssrfWhitelistConfig {
	cfg := &ssrfWhitelistConfig{exactHosts: make(map[string]bool)}
	if raw == "" {
		return cfg
	}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, ipNet, err := net.ParseCIDR(entry)
			if err != nil {
				log.Printf("[ssrf-whitelist] dropping invalid CIDR entry %q: %v", entry, err)
				continue
			}
			cfg.cidrNets = append(cfg.cidrNets, ipNet)
			continue
		}
		if strings.HasPrefix(entry, "*.") {
			suffix := strings.ToLower(entry[1:])
			if len(suffix) <= 1 {
				log.Printf("[ssrf-whitelist] dropping bare wildcard entry %q (need *.<domain>)", entry)
				continue
			}
			cfg.suffixHosts = append(cfg.suffixHosts, suffix)
			continue
		}
		if strings.Contains(entry, "*") {
			log.Printf("[ssrf-whitelist] dropping unsupported wildcard pattern %q (only \"*.\" prefix is supported)", entry)
			continue
		}
		cfg.exactHosts[strings.ToLower(entry)] = true
	}
	return cfg
}

func mergeSSRFWhitelistRaws(primary, extra string) string {
	primary = strings.TrimSpace(primary)
	extra = strings.TrimSpace(extra)
	switch {
	case primary == "" && extra == "":
		return ""
	case primary == "":
		return extra
	case extra == "":
		return primary
	default:
		return primary + "," + extra
	}
}

// IsSSRFWhitelisted reports whether hostname is covered by SSRF_WHITELIST / SSRF_WHITELIST_EXTRA.
func IsSSRFWhitelisted(hostname string) bool {
	wl := loadSSRFWhitelist()
	if wl == nil {
		return false
	}
	lower := strings.ToLower(hostname)
	if wl.exactHosts[lower] {
		return true
	}
	for _, suffix := range wl.suffixHosts {
		if strings.HasSuffix(lower, suffix) || lower == suffix[1:] {
			return true
		}
	}
	if ip := net.ParseIP(hostname); ip != nil {
		for _, cidr := range wl.cidrNets {
			if cidr.Contains(ip) {
				return true
			}
		}
	}
	if net.ParseIP(hostname) == nil && len(wl.cidrNets) > 0 {
		if ips, err := net.LookupIP(hostname); err == nil {
			for _, ip := range ips {
				for _, cidr := range wl.cidrNets {
					if cidr.Contains(ip) {
						return true
					}
				}
			}
		}
	}
	return false
}

// ResetSSRFWhitelistForTest resets the SSRF whitelist singleton for tests.
func ResetSSRFWhitelistForTest() {
	ssrfWhitelistOnce = sync.Once{}
	ssrfWhitelist = nil
	ssrfWhitelistAtomic.Store(nil)
}

// tenantBlockedHostnames are cloud metadata / control-plane hosts that must
// never be targeted even from tenant-configured assistant-tool URLs.
var tenantBlockedHostnames = []string{
	"metadata.google.internal",
	"metadata.tencentyun.com",
	"metadata.aws.internal",
}

// ValidateTenantConfiguredURL validates URLs saved in the tenant assistant-tool
// catalog (HTTP tools, MCP SSE). Unlike ValidateURLForSSRF, loopback and private
// addresses are allowed because the tenant explicitly configured the endpoint.
func ValidateTenantConfiguredURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("%w: empty URL", apperr.ErrInvalidServerURL)
	}
	if len(rawURL) > 2048 {
		return fmt.Errorf("%w: URL exceeds maximum length", apperr.ErrInvalidServerURL)
	}
	normalized := rawURL
	if !strings.Contains(normalized, "://") {
		normalized = "https://" + normalized
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return fmt.Errorf("%w: %v", apperr.ErrInvalidServerURL, err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("%w: invalid scheme %s (only http/https allowed)", apperr.ErrInvalidServerURL, scheme)
	}
	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return fmt.Errorf("%w: URL has no hostname", apperr.ErrInvalidServerURL)
	}
	for _, blocked := range tenantBlockedHostnames {
		if hostname == blocked {
			return fmt.Errorf("%w: hostname %s is not allowed", apperr.ErrInvalidServerURL, hostname)
		}
	}
	if port := parsed.Port(); port != "" {
		blockedPorts := map[string]bool{
			"22": true, "23": true, "25": true, "445": true, "3389": true,
			"5432": true, "3306": true, "6379": true, "27017": true, "9200": true,
			"2379": true, "2380": true, "8500": true, "4001": true,
		}
		if blockedPorts[port] {
			return fmt.Errorf("%w: port %s is blocked for security reasons", apperr.ErrInvalidServerURL, port)
		}
	}
	return nil
}

// NewTenantToolHTTPClient creates an HTTP client for tenant-configured tool
// endpoints (catalog HTTP / MCP SSE). No SSRF dial restrictions — the tenant
// saved the target URL explicitly. When timeout is zero, no global Client.Timeout
// is set (required for long-lived MCP SSE streams).
func NewTenantToolHTTPClient(timeout time.Duration) *http.Client {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after %d redirects", len(via))
			}
			redirectScheme := strings.ToLower(req.URL.Scheme)
			if redirectScheme != "http" && redirectScheme != "https" {
				return fmt.Errorf("redirect blocked: invalid scheme %s", redirectScheme)
			}
			if err := ValidateTenantConfiguredURL(req.URL.String()); err != nil {
				return fmt.Errorf("redirect blocked: %w", err)
			}
			return nil
		},
	}
	if timeout > 0 {
		client.Timeout = timeout
	}
	return client
}

// ValidateURLForSSRF validates a user-supplied URL against SSRF protections.
func ValidateURLForSSRF(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("%w: empty URL", apperr.ErrInvalidServerURL)
	}
	normalized := rawURL
	if !strings.Contains(normalized, "://") {
		normalized = "https://" + normalized
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return fmt.Errorf("%w: %v", apperr.ErrInvalidServerURL, err)
	}
	hostname := parsed.Hostname()
	if hostname == "" {
		return fmt.Errorf("%w: URL has no hostname", apperr.ErrInvalidServerURL)
	}
	if IsSSRFWhitelisted(hostname) {
		return nil
	}
	if safe, reason := isSSRFSafeURL(normalized); !safe {
		return fmt.Errorf("%w: %s", ErrSSRFRedirectBlocked, reason)
	}
	return nil
}

// IsSystemProxy reports whether host matches a configured HTTP(S) proxy.
func IsSystemProxy(host string) bool {
	proxyCfg := httpproxy.FromEnvironment()
	for _, proxyURL := range []string{proxyCfg.HTTPProxy, proxyCfg.HTTPSProxy} {
		if proxyURL == "" {
			continue
		}
		if parse, err := url.Parse(proxyURL); err == nil {
			if parse.Host == host {
				return true
			}
		}
	}
	return false
}

// SSRFSafeHTTPClientConfig configures the SSRF-safe HTTP client.
type SSRFSafeHTTPClientConfig struct {
	Timeout            time.Duration
	MaxRedirects       int
	DisableKeepAlives  bool
	DisableCompression bool
}

// DefaultSSRFSafeHTTPClientConfig returns default SSRF-safe client settings.
func DefaultSSRFSafeHTTPClientConfig() SSRFSafeHTTPClientConfig {
	return SSRFSafeHTTPClientConfig{
		Timeout:            30 * time.Second,
		MaxRedirects:       10,
		DisableKeepAlives:  false,
		DisableCompression: false,
	}
}

// ErrSSRFRedirectBlocked is returned when a redirect target fails SSRF validation.
var ErrSSRFRedirectBlocked = fmt.Errorf("redirect blocked: target URL failed SSRF validation")

// NewSSRFSafeHTTPClient creates an HTTP client with SSRF-safe dial and redirect checks.
func NewSSRFSafeHTTPClient(config SSRFSafeHTTPClientConfig) *http.Client {
	transport := &http.Transport{
		DisableKeepAlives:  config.DisableKeepAlives,
		DisableCompression: config.DisableCompression,
		DialContext:        SSRFSafeDialContext,
	}
	return &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= config.MaxRedirects {
				return fmt.Errorf("stopped after %d redirects", config.MaxRedirects)
			}
			redirectScheme := strings.ToLower(req.URL.Scheme)
			if redirectScheme != "http" && redirectScheme != "https" {
				return fmt.Errorf("%w: invalid scheme %s", ErrSSRFRedirectBlocked, redirectScheme)
			}
			redirectHost := req.URL.Hostname()
			if redirectHost != "" && IsSSRFWhitelisted(redirectHost) {
				return nil
			}
			redirectURL := req.URL.String()
			if safe, reason := isSSRFSafeURL(redirectURL); !safe {
				return fmt.Errorf("%w: %s", ErrSSRFRedirectBlocked, reason)
			}
			return nil
		},
	}
}

// SSRFSafeDialContext dials only after validating resolved IPs against SSRF rules.
func SSRFSafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address %s: %w", addr, err)
	}
	if IsSystemProxy(addr) || IsSSRFWhitelisted(host) {
		dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
		return dialer.DialContext(ctx, network, addr)
	}
	hostLower := strings.ToLower(host)
	for _, restricted := range restrictedHostnames {
		if hostLower == restricted {
			return nil, fmt.Errorf("connection blocked: hostname %s is restricted", host)
		}
	}
	for _, suffix := range restrictedHostSuffixes {
		if strings.HasSuffix(hostLower, suffix) {
			return nil, fmt.Errorf("connection blocked: hostname suffix %s is restricted", suffix)
		}
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("DNS resolution failed for %s: %w", host, err)
	}
	for _, ipAddr := range ips {
		if restricted, reason := isRestrictedIP(ipAddr.IP); restricted {
			return nil, fmt.Errorf("connection blocked: %s resolves to restricted IP %s (%s)", host, ipAddr.IP.String(), reason)
		}
	}
	dialer := &net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}
	return dialer.DialContext(ctx, network, addr)
}
