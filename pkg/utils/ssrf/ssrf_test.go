// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0
package ssrf

import "testing"

func TestValidateURL_RejectsPrivateAndLoopback(t *testing.T) {
	p := Default()
	cases := []string{
		"http://127.0.0.1/x",
		"http://10.0.0.1/x",
		"http://192.168.1.1/x",
		"http://169.254.169.254/latest/meta-data/", // AWS metadata
		"http://[::1]/x",
		"http://localhost/x",
		"file:///etc/passwd",
		"http://0.0.0.0/x",
		"http://100.64.0.1/x", // CGNAT
	}
	for _, u := range cases {
		if err := p.ValidateURL(u); err == nil {
			t.Errorf("expected reject for %s", u)
		}
	}
}

func TestValidateURL_RejectsBadPort(t *testing.T) {
	if err := Default().ValidateURL("http://example.com:22/x"); err == nil {
		t.Error("expected reject port 22")
	}
}

func TestValidateURL_AllowsCustomPorts(t *testing.T) {
	p := &Protection{AllowedPorts: []int{8443}}
	if err := p.ValidateURL("https://example.com:8443/x"); err != nil {
		// DNS may fail in offline test env; only fail when error is about port.
		if err.Error() == "ssrf: port 8443 not allowed" {
			t.Errorf("port should be allowed: %v", err)
		}
	}
}

func TestValidateURL_AllowPrivateBypass(t *testing.T) {
	p := &Protection{AllowPrivateIP: true}
	if err := p.ValidateURL("http://127.0.0.1/x"); err != nil {
		t.Errorf("expected allow with bypass: %v", err)
	}
}

func TestValidateURL_RejectsNonHTTP(t *testing.T) {
	bad := []string{"gopher://x/", "ftp://x/", "javascript:alert(1)", ""}
	for _, u := range bad {
		if err := Default().ValidateURL(u); err == nil {
			t.Errorf("expected reject for %s", u)
		}
	}
}
