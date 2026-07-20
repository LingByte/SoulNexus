// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

// Package acme provides optional Let's Encrypt certificate management for HTTP TLS.
package acme

import (
	"crypto/tls"
	"os"
	"strings"
	"sync"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const (
	EnvDomains  = "SSL_ACME_DOMAINS"   // comma-separated hostnames
	EnvCacheDir = "SSL_ACME_CACHE_DIR" // default ./data/acme-cache
	EnvEmail    = "SSL_ACME_EMAIL"
	EnvDisable  = "SSL_ACME_DISABLE" // "1" to force off
)

// Enabled reports whether ACME is configured (domains non-empty and not disabled).
func Enabled() bool {
	if strings.TrimSpace(os.Getenv(EnvDisable)) == "1" {
		return false
	}
	return len(Domains()) > 0
}

// Domains parses SSL_ACME_DOMAINS.
func Domains() []string {
	raw := strings.TrimSpace(os.Getenv(EnvDomains))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

var (
	mgrOnce sync.Once
	mgrInst *autocert.Manager
)

// Manager returns the process-wide autocert manager (nil when disabled).
// HTTP-01, HTTPS TLSConfig, and TLS listeners must share this instance so challenge
// tokens stay coherent.
func Manager() *autocert.Manager {
	if !Enabled() {
		return nil
	}
	mgrOnce.Do(func() {
		cache := strings.TrimSpace(os.Getenv(EnvCacheDir))
		if cache == "" {
			cache = "data/acme-cache"
		}
		_ = os.MkdirAll(cache, 0o700)
		email := strings.TrimSpace(os.Getenv(EnvEmail))
		mgrInst = &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(Domains()...),
			Cache:      autocert.DirCache(cache),
			Email:      email,
			Client:     &acme.Client{DirectoryURL: acme.LetsEncryptURL},
		}
	})
	return mgrInst
}

// TLSConfig returns a tls.Config using Let's Encrypt when Enabled, else nil.
func TLSConfig() *tls.Config {
	m := Manager()
	if m == nil {
		return nil
	}
	cfg := m.TLSConfig()
	cfg.MinVersion = tls.VersionTLS12
	return cfg
}

// HTTPHandler returns the ACME manager for HTTP-01 (nil when disabled).
func HTTPHandler() *autocert.Manager {
	return Manager()
}
