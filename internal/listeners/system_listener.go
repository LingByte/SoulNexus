package listeners

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"crypto/tls"
	"sync"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/internal/constants"
	"github.com/LingByte/SoulNexus/pkg/logger"
	acmeutil "github.com/LingByte/SoulNexus/pkg/tlsutil/acme"
	"github.com/LingByte/SoulNexus/pkg/utils"
	"github.com/LingByte/SoulNexus/pkg/utils/system"
	"go.uber.org/zap"
)

var (
	// Global variables for SSL certificate
	sslCert     tls.Certificate
	sslCertOnce sync.Once
	sslCertErr  error
)

// InitSystemListeners initializes system listeners
func InitSystemListeners() {
	// Connect system initialization signal
	utils.Sig().Connect(constants.SigInitSystemConfig, func(sender any, params ...any) {
		// Load SSL certificates
		loadSSLCertificates()
	})
	// InitLLMListener is initialized in main.go (requires database connection)
	logger.Info("system module listener is already")
	utils.Sig().Emit(constants.SigInitSystemConfig, nil)
}

// loadSSLCertificates loads SSL certificates
func loadSSLCertificates() {
	if !config.GlobalConfig.Server.SSLEnabled {
		logger.Info("SSL is disabled, skipping SSL certificate loading")
		return
	}

	certFile := config.GlobalConfig.Server.SSLCertFile
	keyFile := config.GlobalConfig.Server.SSLKeyFile

	if certFile == "" || keyFile == "" {
		logger.Warn("SSL enabled but certificate files not configured",
			zap.String("certFile", certFile),
			zap.String("keyFile", keyFile))
		return
	}

	// Use sync.Once to ensure loading only once
	sslCertOnce.Do(func() {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			sslCertErr = err
			system.IncSSLCertLoadFail()
			logger.Error("Failed to load SSL certificates",
				zap.String("certFile", certFile),
				zap.String("keyFile", keyFile),
				zap.Error(err))
			return
		}

		sslCert = cert
		logger.Info("SSL certificates loaded successfully",
			zap.String("certFile", certFile),
			zap.String("keyFile", keyFile))
	})
}

// GetSSLCertificate gets the loaded SSL certificate
func GetSSLCertificate() (tls.Certificate, error) {
	if sslCertErr != nil {
		return tls.Certificate{}, sslCertErr
	}
	return sslCert, nil
}

// IsSSLEnabled checks if SSL is enabled via ACME or loaded file certificates.
func IsSSLEnabled() bool {
	if acmeutil.Enabled() {
		return true
	}
	return config.GlobalConfig.Server.SSLEnabled && sslCertErr == nil
}

// GetTLSConfig gets TLS configuration (ACME Let's Encrypt when SSL_ACME_DOMAINS set, else file certs).
func GetTLSConfig() (*tls.Config, error) {
	if cfg := acmeutil.TLSConfig(); cfg != nil {
		logger.Info("using ACME TLS (Let's Encrypt)", zap.Strings("domains", acmeutil.Domains()))
		return cfg, nil
	}
	if !IsSSLEnabled() {
		return nil, nil
	}

	cert, err := GetSSLCertificate()
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
	}, nil
}
