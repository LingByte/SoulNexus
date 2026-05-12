// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
//
// 防 SSRF 工具：在拉取用户给定 URL（音频/图片/Webhook/file_url 等）前，
// 校验目标主机不会指向内网 / loopback / link-local / 元数据地址。
//
// 设计要点：
//   - URL 必须为 http/https；
//   - 解析 host：直接 IP 直接拒绝私网；hostname 走 net.LookupIP 并对全部 A/AAAA 拒绝私网；
//   - 端口默认仅允许 80/443，可在 Protection 里覆盖；
//   - 提供 SafeHTTPClient：在 DialContext 时再做一次 IP 防护（防 DNS rebinding）。

package ssrf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Protection 单次校验配置；零值即默认严格策略（仅外网 + 80/443）。
type Protection struct {
	// AllowPrivateIP 为 true 时跳过私网/loopback 拒绝，仅做 scheme 校验。仅本地开发用。
	AllowPrivateIP bool
	// AllowedPorts nil = 仅 80/443；提供时仅这些端口被放行。
	AllowedPorts []int
	// AllowedHosts 域名白名单（精确匹配；为空表示不启用白名单）。
	AllowedHosts []string
	// LookupTimeout DNS 解析超时；默认 3s。
	LookupTimeout time.Duration
}

// Default 严格策略：禁止私网，端口仅 80/443，3s DNS 超时。
func Default() *Protection { return &Protection{LookupTimeout: 3 * time.Second} }

// ValidateURL 主入口；err == nil 表示该 URL 当前可被安全拉取。
func (p *Protection) ValidateURL(rawURL string) error {
	if p == nil {
		p = Default()
	}
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return errors.New("ssrf: empty url")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("ssrf: parse url: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("ssrf: scheme %q not allowed", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("ssrf: empty host")
	}
	if err := p.validatePort(u, scheme); err != nil {
		return err
	}
	if len(p.AllowedHosts) > 0 {
		ok := false
		for _, h := range p.AllowedHosts {
			if strings.EqualFold(strings.TrimSpace(h), host) {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("ssrf: host %q not in allowlist", host)
		}
	}
	// 直接 IP
	if ip := net.ParseIP(host); ip != nil {
		return p.validateIP(ip)
	}
	// 域名 → 解析 → 全部校验
	timeout := p.LookupTimeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("ssrf: dns lookup failed: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("ssrf: dns lookup returned no ip for %q", host)
	}
	for _, ip := range ips {
		if err := p.validateIP(ip); err != nil {
			return err
		}
	}
	return nil
}

func (p *Protection) validatePort(u *url.URL, scheme string) error {
	allowed := p.AllowedPorts
	if len(allowed) == 0 {
		allowed = []int{80, 443}
	}
	port := u.Port()
	var pn int
	if port == "" {
		if scheme == "https" {
			pn = 443
		} else {
			pn = 80
		}
	} else {
		var err error
		pn, err = parsePort(port)
		if err != nil {
			return fmt.Errorf("ssrf: invalid port %q", port)
		}
	}
	for _, a := range allowed {
		if a == pn {
			return nil
		}
	}
	return fmt.Errorf("ssrf: port %d not allowed", pn)
}

// validateIP 对 IPv4 / IPv6 都做防护：拒绝 loopback / private / link-local / multicast / unspecified / 元数据地址。
func (p *Protection) validateIP(ip net.IP) error {
	if ip == nil {
		return errors.New("ssrf: nil ip")
	}
	if p.AllowPrivateIP {
		return nil
	}
	if ip.IsLoopback() {
		return fmt.Errorf("ssrf: loopback ip %s", ip)
	}
	if ip.IsUnspecified() {
		return fmt.Errorf("ssrf: unspecified ip %s", ip)
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("ssrf: link-local ip %s", ip)
	}
	if ip.IsMulticast() {
		return fmt.Errorf("ssrf: multicast ip %s", ip)
	}
	if ip.IsPrivate() {
		return fmt.Errorf("ssrf: private ip %s", ip)
	}
	// 阻止 169.254.169.254（云元数据） / fd00::/8 / fc00::/7 / 100.64/10（CGNAT）
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 169 && v4[1] == 254 {
			return fmt.Errorf("ssrf: link-local v4 %s", ip)
		}
		if v4[0] == 100 && v4[1]&0xC0 == 64 { // 100.64.0.0/10
			return fmt.Errorf("ssrf: cgnat ip %s", ip)
		}
		// 0.0.0.0/8
		if v4[0] == 0 {
			return fmt.Errorf("ssrf: zero-net ip %s", ip)
		}
	} else {
		// IPv6 ULA fc00::/7（IsPrivate 已覆盖）；额外阻止 ::ffff:0:0/96 (IPv4-mapped) 中的私网。
		if ip4 := ip.To4(); ip4 != nil {
			return p.validateIP(ip4)
		}
	}
	return nil
}

func parsePort(s string) (int, error) {
	n := 0
	if len(s) == 0 || len(s) > 5 {
		return 0, errors.New("bad port")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("bad port")
		}
		n = n*10 + int(r-'0')
	}
	if n < 1 || n > 65535 {
		return 0, errors.New("bad port")
	}
	return n, nil
}

// SafeHTTPClient 返回一个客户端，其 DialContext 会再次校验目标 IP，
// 防止 DNS rebinding（解析时是公网 IP，连接时换成内网 IP）。
func (p *Protection) SafeHTTPClient(timeout time.Duration) *http.Client {
	if p == nil {
		p = Default()
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 10 * time.Second}
	tr := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ip := net.ParseIP(host)
			if ip == nil {
				ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
				if err != nil {
					return nil, err
				}
				if len(ips) == 0 {
					return nil, fmt.Errorf("ssrf: no ip for %s", host)
				}
				ip = ips[0]
			}
			if err := p.validateIP(ip); err != nil {
				return nil, err
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		},
		ForceAttemptHTTP2: false,
	}
	return &http.Client{Transport: tr, Timeout: timeout}
}
