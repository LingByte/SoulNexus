package listeners

// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0

import (
	"github.com/LingByte/SoulNexus/internal/models"

	"crypto/tls"
	"sync"

	"github.com/LingByte/SoulNexus/internal/config"
	"github.com/LingByte/SoulNexus/pkg/logger"
	"github.com/LingByte/SoulNexus/pkg/utils"
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
	utils.Sig().Connect(models.SigInitSystemConfig, func(sender any, params ...any) {
		// Load SSL certificates
		loadSSLCertificates()
	})
	InitAgentListener()
	InitUserListeners()
	// InitLLMListener is initialized in main.go (requires database connection)
	logger.Info("system module listener is already")
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

// IsSSLEnabled checks if SSL is enabled and certificates are loaded
func IsSSLEnabled() bool {
	return config.GlobalConfig.Server.SSLEnabled && sslCertErr == nil
}

// StopAll stops all registered signal listeners.
func StopAll() {
	// Clear common signal channels that listeners subscribe to.
	// The signal system's Clear() removes all handlers for the given event names.
	utils.Sig().Clear(
		"LLMUsage",
		"llm.request.start",
		"llm.request.end",
		"llm.request.error",
		"session.created",
		"session.updated",
		"session.deleted",
		"message.created",
		"message.updated",
		"message.deleted",
		"billing.usage",
		"billing.charge",
		"agent.created",
		"agent.updated",
		"agent.deleted",
		"user.created",
		"user.updated",
		"user.deleted",
	)
	logger.Info("all system listeners stopped")
}

// GetTLSConfig gets TLS configuration
func GetTLSConfig() (*tls.Config, error) {
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
		// Recommended TLS configuration
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
