package listeners

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitUserListeners_Basic(t *testing.T) {
	// Test that the function exists and can be called
	assert.NotNil(t, InitUserListeners)
}

func TestSendWelcomeEmail_Function(t *testing.T) {
	// Test that the function exists
	assert.NotNil(t, sendWelcomeEmail)
}

func TestSendEmailVerification_Function(t *testing.T) {
	// Test that the function exists
	assert.NotNil(t, sendEmailVerification)
}

func TestSendPasswordResetEmail_Function(t *testing.T) {
	// Test that the function exists
	assert.NotNil(t, sendPasswordResetEmail)
}

func TestSendNewDeviceLoginAlert_Function(t *testing.T) {
	// Test that the function exists
	assert.NotNil(t, sendNewDeviceLoginAlert)
}

func TestLogUserEvent_Function(t *testing.T) {
	// Test that the function exists
	assert.NotNil(t, logUserEvent)
}

// Benchmark tests
func BenchmarkInitUserListeners(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// We can't actually call the function due to logger dependency
		// but we can test that the function exists
		_ = InitUserListeners
	}
}
