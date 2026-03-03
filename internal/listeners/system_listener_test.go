package listeners

import (
	"crypto/tls"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitSystemListeners_Basic(t *testing.T) {
	// Test that the function exists
	assert.NotNil(t, InitSystemListeners)
}

func TestLoadSSLCertificates_Function(t *testing.T) {
	// Test that the function exists
	assert.NotNil(t, loadSSLCertificates)
}

func TestGetSSLCertificate_Function(t *testing.T) {
	// Test that the function exists
	assert.NotNil(t, GetSSLCertificate)
}

func TestIsSSLEnabled_Function(t *testing.T) {
	// Test that the function exists
	assert.NotNil(t, IsSSLEnabled)
}

func TestGetTLSConfig_Function(t *testing.T) {
	// Test that the function exists
	assert.NotNil(t, GetTLSConfig)
}

func TestResetSSLGlobals(t *testing.T) {
	// Test that we can reset the global variables
	resetSSLGlobals()

	// Verify the globals are reset
	assert.Equal(t, tls.Certificate{}, sslCert)
	assert.NotNil(t, sslCertOnce)
	assert.Nil(t, sslCertErr)
}

// Helper function to reset global SSL variables for testing
func resetSSLGlobals() {
	sslCert = tls.Certificate{}
	sslCertOnce = sync.Once{}
	sslCertErr = nil
}

// Benchmark tests
func BenchmarkResetSSLGlobals(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resetSSLGlobals()
	}
}
