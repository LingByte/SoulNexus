// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

package rtcsfu_replica

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/LingByte/SoulNexus/internal/config"
)

// JoinPrimaryAPI joins primary base URL (with or without trailing slash) with an API path that must start with "/".
func JoinPrimaryAPI(primaryBase, path string) string {
	base := strings.TrimRight(strings.TrimSpace(primaryBase), "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func newReplicaHTTPClient(cfg config.RTCSFUConfig) (*http.Client, error) {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("rtcsfu_replica: default transport is not *http.Transport")
	}
	tr := t.Clone()

	tlsConf, err := replicaTLSConfig(cfg)
	if err != nil {
		return nil, err
	}
	if tlsConf != nil {
		tr.TLSClientConfig = tlsConf
	}

	return &http.Client{
		Transport: tr,
		Timeout:   25 * time.Second,
	}, nil
}

func replicaTLSConfig(cfg config.RTCSFUConfig) (*tls.Config, error) {
	caFile := strings.TrimSpace(cfg.ReplicaMTLSClientCAFile)
	certFile := strings.TrimSpace(cfg.ReplicaMTLSClientCertFile)
	keyFile := strings.TrimSpace(cfg.ReplicaMTLSClientKeyFile)
	insec := cfg.ReplicaMTLSInsecureSkipVerify

	if caFile == "" && certFile == "" && keyFile == "" && !insec {
		return nil, nil
	}

	tlsConf := &tls.Config{MinVersion: tls.VersionTLS12}
	if insec {
		// Explicit dev-only escape hatch; pair with RTCSFU_PRIMARY_BASE_URL on trusted networks only.
		tlsConf.InsecureSkipVerify = true // #nosec G402
	}
	if caFile != "" {
		b, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read RTCSFU_REPLICA_MTLS_CA_FILE: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(b) {
			return nil, fmt.Errorf("RTCSFU_REPLICA_MTLS_CA_FILE: no valid PEM certificates")
		}
		tlsConf.RootCAs = pool
	}
	if certFile != "" || keyFile != "" {
		if certFile == "" || keyFile == "" {
			return nil, fmt.Errorf("RTCSFU_REPLICA_MTLS_CERT_FILE and RTCSFU_REPLICA_MTLS_KEY_FILE must both be set")
		}
		c, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("load mTLS client cert: %w", err)
		}
		tlsConf.Certificates = []tls.Certificate{c}
	}
	return tlsConf, nil
}
